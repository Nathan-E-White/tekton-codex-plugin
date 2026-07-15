#!/usr/bin/env bash
set -euo pipefail

required=(GITHUB_APP_ID GITHUB_APP_INSTALLATION_ID GITHUB_APP_PRIVATE_KEY GITHUB_WEBHOOK_SECRET GITHUB_SMEE_URL GITHUB_TEST_REPOSITORY GITHUB_TEST_PR)
for name in "${required[@]}"; do
  if [[ -z ${!name:-} ]]; then echo "missing required environment variable: $name" >&2; exit 2; fi
done
for command in curl gosmee jq kubectl openssl; do
  command -v "$command" >/dev/null || { echo "missing command: $command" >&2; exit 2; }
done

context=${TEKTON_CONTEXT:?set TEKTON_CONTEXT to the disposable cluster context}
namespace=${TEKTON_NAMESPACE:-default}
artifacts=${TEKTON_SMOKE_ARTIFACTS:-"$(mktemp -d)"}
repo_url="https://github.com/$GITHUB_TEST_REPOSITORY"
api="https://api.github.com/repos/$GITHUB_TEST_REPOSITORY"
mkdir -p "$artifacts"
private_key="$artifacts/github-app.pem"
printf '%s\n' "$GITHUB_APP_PRIVATE_KEY" >"$private_key"
chmod 600 "$private_key"
cleanup() {
  jobs -p | xargs -r kill >/dev/null 2>&1 || true
  rm -f "$private_key" "$artifacts/installation-token"
}
trap cleanup EXIT INT TERM
k() { kubectl --context "$context" "$@"; }

base64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }
now=$(date +%s)
header=$(printf '%s' '{"alg":"RS256","typ":"JWT"}' | base64url)
payload=$(printf '{"iat":%d,"exp":%d,"iss":"%s"}' "$((now - 60))" "$((now + 540))" "$GITHUB_APP_ID" | base64url)
signature=$(printf '%s' "$header.$payload" | openssl dgst -sha256 -sign "$private_key" | base64url)
jwt="$header.$payload.$signature"
curl --fail --silent --show-error -X POST \
  -H 'Accept: application/vnd.github+json' -H "Authorization: Bearer $jwt" \
  "https://api.github.com/app/installations/$GITHUB_APP_INSTALLATION_ID/access_tokens" \
  | jq -er .token >"$artifacts/installation-token"
token=$(<"$artifacts/installation-token")

k -n pipelines-as-code delete secret pipelines-as-code-secret --ignore-not-found >/dev/null
k -n pipelines-as-code create secret generic pipelines-as-code-secret \
  --from-file=github-private-key="$private_key" \
  --from-literal="github-application-id=$GITHUB_APP_ID" \
  --from-literal="webhook.secret=$GITHUB_WEBHOOK_SECRET" >/dev/null
k -n pipelines-as-code rollout restart deployment/pipelines-as-code-controller deployment/pipelines-as-code-watcher deployment/pipelines-as-code-webhook >/dev/null
for deployment in pipelines-as-code-controller pipelines-as-code-watcher pipelines-as-code-webhook; do
  k -n pipelines-as-code rollout status "deployment/$deployment" --timeout=180s >/dev/null
done

cat <<EOF | k -n "$namespace" apply -f - >/dev/null
apiVersion: pipelinesascode.tekton.dev/v1alpha1
kind: Repository
metadata:
  name: tekton-codex-plugin-smoke
spec:
  url: $repo_url
EOF

k -n pipelines-as-code port-forward service/pipelines-as-code-controller 18082:8080 >"$artifacts/pac-port-forward.log" 2>&1 &
port_forward_pid=$!
sleep 3
gosmee client "$GITHUB_SMEE_URL" http://127.0.0.1:18082 >"$artifacts/gosmee.log" 2>&1 &
gosmee_pid=$!
sleep 3

github() {
  curl --fail --silent --show-error -H 'Accept: application/vnd.github+json' -H "Authorization: Bearer $token" "$@"
}
comment() {
  github -X POST "$api/issues/$GITHUB_TEST_PR/comments" -d "$(jq -nc --arg body "$1" '{body:$body}')" >/dev/null
}
latest_uid() {
  k -n "$namespace" get pipelineruns --sort-by=.metadata.creationTimestamp -o json | jq -r '.items[-1].metadata.uid // empty'
}
wait_for_new_uid() {
  local previous=$1 current
  for _ in {1..90}; do
    current=$(latest_uid)
    if [[ -n $current && $current != "$previous" ]]; then printf '%s' "$current"; return 0; fi
    sleep 5
  done
  echo "timed out waiting for PipelineRun creation" >&2
  return 1
}

head_sha=$(github "$api/pulls/$GITHUB_TEST_PR" | jq -er .head.sha)
before=$(latest_uid)
comment /test
test_uid=$(wait_for_new_uid "$before")
for _ in {1..60}; do
  check_count=$(github "$api/commits/$head_sha/check-runs" | jq '[.check_runs[] | select(.app.id == ('"$GITHUB_APP_ID"'))] | length')
  [[ $check_count -gt 0 ]] && break
  sleep 5
done
[[ ${check_count:-0} -gt 0 ]] || { echo "GitHub App check reporting was not observed" >&2; exit 1; }

comment /retest
retest_uid=$(wait_for_new_uid "$test_uid")
comment /cancel
for _ in {1..60}; do
  reason=$(k -n "$namespace" get pipelinerun -o json | jq -r --arg uid "$retest_uid" '.items[] | select(.metadata.uid == $uid) | .status.conditions[0].reason // empty')
  [[ $reason == Cancelled ]] && break
  sleep 5
done
[[ ${reason:-} == Cancelled ]] || { echo "ChatOps /cancel did not cancel the retested PipelineRun" >&2; exit 1; }

kill "$gosmee_pid" "$port_forward_pid" >/dev/null 2>&1 || true
echo "credentialed GitHub App smoke passed: delivery, PipelineRun, check reporting, /test, /retest, /cancel"
