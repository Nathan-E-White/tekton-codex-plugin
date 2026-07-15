#!/bin/sh
set -eu

cluster=${TEKTON_KIND_CLUSTER:-tekton-codex-plugin}
context="kind-$cluster"
cleanup() { kind delete cluster --name "$cluster" >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM

kind create cluster --name "$cluster" --wait 120s
pipeline_manifest=https://github.com/tektoncd/pipeline/releases/download/v1.14.0/release.yaml
kubectl --context "$context" apply -f "$pipeline_manifest"
kubectl --context "$context" wait --namespace tekton-pipelines --for=condition=Available deployment/tekton-pipelines-controller --timeout=180s
kubectl --context "$context" apply -f test/fixtures/smoke.yaml
kubectl --context "$context" wait --namespace default --for=condition=Succeeded taskrun/tekton-codex-smoke --timeout=180s
kubectl --context "$context" delete -f test/fixtures/smoke.yaml
kubectl --context "$context" delete -f "$pipeline_manifest"
