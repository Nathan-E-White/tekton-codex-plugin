#!/usr/bin/env bash
set -euo pipefail

cluster=${TEKTON_KIND_CLUSTER:-tekton-codex-plugin}
context="kind-$cluster"
artifacts=${TEKTON_SMOKE_ARTIFACTS:-"$(mktemp -d)"}
pipeline_manifest=https://github.com/tektoncd/pipeline/releases/download/v1.14.0/release.yaml
triggers_manifest=https://github.com/tektoncd/triggers/releases/download/v0.36.0/release.yaml
interceptors_manifest=https://github.com/tektoncd/triggers/releases/download/v0.36.0/interceptors.yaml
chains_manifest=https://github.com/tektoncd/chains/releases/download/v0.28.0/release.yaml
results_manifest=https://github.com/tektoncd/results/releases/download/v0.19.0/release.yaml
pac_manifest=https://github.com/tektoncd/pipelines-as-code/releases/download/v0.49.0/release.k8s.yaml

cleanup() {
  jobs -p | xargs -r kill >/dev/null 2>&1 || true
  kind delete cluster --name "$cluster" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

k() { kubectl --context "$context" "$@"; }
wait_deployment() { k -n "$1" wait --for=condition=Available "deployment/$2" --timeout=300s; }
wait_taskrun() { k -n default wait --for=condition=Succeeded "taskrun/$1" --timeout=300s; }
wait_annotation() {
  local resource=$1 key=$2
  for _ in {1..60}; do
    if k -n default get "$resource" -o json | jq -e --arg key "$key" '.metadata.annotations | has($key)' >/dev/null; then return 0; fi
    sleep 5
  done
  echo "timed out waiting for annotation $key on $resource" >&2
  return 1
}

mkdir -p "$artifacts"
kind create cluster --name "$cluster" --wait 120s

# Pipelines establishes the shared namespace used by Results and PAC.
k apply -f "$pipeline_manifest"
wait_deployment tekton-pipelines tekton-pipelines-controller
wait_deployment tekton-pipelines tekton-pipelines-webhook

# Results requires a database credential and API TLS material before its pods start.
db_password=$(openssl rand -hex 24)
k -n tekton-pipelines create secret generic tekton-results-postgres \
  --from-literal=POSTGRES_USER=postgres --from-literal="POSTGRES_PASSWORD=$db_password"
openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
  -subj /CN=tekton-results-api-service.tekton-pipelines.svc.cluster.local \
  -addext subjectAltName=DNS:tekton-results-api-service.tekton-pipelines.svc.cluster.local \
  -keyout "$artifacts/results.key" -out "$artifacts/results.crt" >/dev/null 2>&1
k -n tekton-pipelines create secret tls tekton-results-tls \
  --cert="$artifacts/results.crt" --key="$artifacts/results.key"

# PAC is installed but intentionally receives non-production, cluster-local smoke credentials.
openssl genrsa -out "$artifacts/pac.key" 2048 >/dev/null 2>&1
webhook_secret=$(openssl rand -hex 24)
k -n tekton-pipelines create secret generic pipelines-as-code-secret \
  --from-file=github-private-key="$artifacts/pac.key" \
  --from-literal=github-application-id=1 \
  --from-literal="webhook.secret=$webhook_secret"

k apply -f "$triggers_manifest"
k apply -f "$interceptors_manifest"
k apply -f "$chains_manifest"
k apply -f "$results_manifest"
k apply -f "$pac_manifest"

wait_deployment tekton-pipelines tekton-triggers-controller
wait_deployment tekton-pipelines tekton-triggers-webhook
wait_deployment tekton-pipelines tekton-triggers-core-interceptors
wait_deployment tekton-chains tekton-chains-controller
wait_deployment tekton-pipelines tekton-results-api
wait_deployment tekton-pipelines tekton-results-watcher
wait_deployment tekton-pipelines tekton-results-retention-policy-agent
k -n tekton-pipelines rollout status statefulset/tekton-results-postgres --timeout=300s
wait_deployment pipelines-as-code pipelines-as-code-controller
wait_deployment pipelines-as-code pipelines-as-code-watcher
wait_deployment pipelines-as-code pipelines-as-code-webhook

# Reapplying every pinned manifest is the public idempotent-reconciliation seam.
for manifest in "$pipeline_manifest" "$triggers_manifest" "$interceptors_manifest" "$chains_manifest" "$results_manifest" "$pac_manifest"; do
  k apply -f "$manifest" >/dev/null
done

# A deleted controller is repaired from the same immutable manifest.
k -n tekton-pipelines delete deployment tekton-triggers-controller
k apply -f "$triggers_manifest" >/dev/null
wait_deployment tekton-pipelines tekton-triggers-controller

# Chains stores signatures on run objects; Results records the same completed runs.
k -n tekton-chains patch configmap chains-config --type merge -p \
  '{"data":{"artifacts.taskrun.format":"in-toto","artifacts.taskrun.signer":"x509","artifacts.taskrun.storage":"tekton","artifacts.pipelinerun.format":"in-toto","artifacts.pipelinerun.signer":"x509","artifacts.pipelinerun.storage":"tekton"}}'
k -n tekton-chains rollout restart deployment/tekton-chains-controller
k -n tekton-chains rollout status deployment/tekton-chains-controller --timeout=300s

k apply -f test/fixtures/full-bundle.yaml
k -n default wait --for=condition=Succeeded pipelinerun/tekton-codex-pipeline-smoke --timeout=300s
wait_taskrun tekton-codex-task-smoke

for run in tekton-codex-task-smoke; do
  # Chains sets this only after the configured signer and storage backends complete.
  k -n default wait --for=jsonpath='{.metadata.annotations.chains\.tekton\.dev/signed}'=true "taskrun/$run" --timeout=300s
  wait_annotation "taskrun/$run" results.tekton.dev/result
  wait_annotation "taskrun/$run" results.tekton.dev/record
done

# A signed GitHub-shaped event must cross the live EventListener seam.
k create secret generic github-webhook --from-literal=secretToken="$webhook_secret"
k apply -f test/fixtures/trigger-smoke.yaml
k -n default wait --for=condition=Available deployment/el-tekton-codex-smoke --timeout=300s
k -n default port-forward service/el-tekton-codex-smoke 18080:8080 >"$artifacts/eventlistener-port-forward.log" 2>&1 &
eventlistener_pid=$!
sleep 3
signature=$(openssl dgst -sha256 -hmac "$webhook_secret" test/fixtures/github/webhook-push.json | awk '{print $2}')
curl --fail --silent --show-error -X POST http://127.0.0.1:18080 \
  -H 'Content-Type: application/json' -H 'X-GitHub-Event: push' \
  -H "X-Hub-Signature-256: sha256=$signature" \
  --data-binary @test/fixtures/github/webhook-push.json >/dev/null
kill "$eventlistener_pid" >/dev/null 2>&1 || true
wait_taskrun tekton-codex-trigger-smoke

# Cancellation is exercised against a live PipelineRun, not a mocked client.
k apply -f test/fixtures/cancel-smoke.yaml
k -n default patch pipelinerun tekton-codex-cancel-smoke --type merge -p '{"spec":{"status":"Cancelled"}}'
k -n default wait --for=jsonpath='{.status.conditions[0].reason}'=Cancelled pipelinerun/tekton-codex-cancel-smoke --timeout=180s

# Query the Results API with a short-lived service-account token; neither token nor bodies enter logs.
k create serviceaccount tekton-results-smoke-reader
k create clusterrolebinding tekton-results-smoke-reader --clusterrole=tekton-results-readonly --serviceaccount=default:tekton-results-smoke-reader
results_token=$(k create token tekton-results-smoke-reader --duration=10m)
k -n tekton-pipelines port-forward service/tekton-results-api-service 18081:8080 >"$artifacts/results-port-forward.log" 2>&1 &
results_pid=$!
sleep 3
curl --fail --silent --show-error --noproxy '*' --cacert "$artifacts/results.crt" \
  --resolve tekton-results-api-service.tekton-pipelines.svc.cluster.local:18081:127.0.0.1 \
  -H "Authorization: Bearer $results_token" \
  'https://tekton-results-api-service.tekton-pipelines.svc.cluster.local:18081/v1alpha2/parents/default/results/-/records?page_size=100' \
  -o "$artifacts/results-records.json"
jq -e '.records | length > 0' "$artifacts/results-records.json" >/dev/null
kill "$results_pid" >/dev/null 2>&1 || true
rm -f "$artifacts/results-records.json" "$artifacts/results.key" "$artifacts/pac.key"
k -n tekton-pipelines get configmap tekton-results-config-results-retention-policy >/dev/null

# Teardown follows reverse dependency order and verifies cluster-scoped residue is gone.
k delete -f "$pac_manifest" --ignore-not-found
k delete -f "$results_manifest" --ignore-not-found
k delete -f "$chains_manifest" --ignore-not-found
k delete -f "$interceptors_manifest" --ignore-not-found
k delete -f "$triggers_manifest" --ignore-not-found
k delete -f "$pipeline_manifest" --ignore-not-found
for crd in repositories.pipelinesascode.tekton.dev results.results.tekton.dev taskruns.tekton.dev pipelineruns.tekton.dev; do
  if k get crd "$crd" >/dev/null 2>&1; then
    echo "teardown left CRD $crd" >&2
    exit 1
  fi
done

echo "full pinned-bundle kind smoke passed"
