# Supported upstream bundle

| Component | Version | Namespace | Release manifest |
| --- | --- | --- | --- |
| Pipelines | `v1.14.0` | `tekton-pipelines` | `https://github.com/tektoncd/pipeline/releases/download/v1.14.0/release.yaml` |
| Triggers | `v0.36.0` | `tekton-pipelines` | `https://github.com/tektoncd/triggers/releases/download/v0.36.0/release.yaml` |
| Chains | `v0.28.0` | `tekton-chains` | `https://github.com/tektoncd/chains/releases/download/v0.28.0/release.yaml` |
| Results | `v0.19.0` | `tekton-pipelines` | `https://github.com/tektoncd/results/releases/download/v0.19.0/release.yaml` |
| Pipelines-as-Code | `v0.49.0` | `pipelines-as-code` | `https://github.com/tektoncd/pipelines-as-code/releases/download/v0.49.0/release.k8s.yaml` |
| tkn | `v0.45.0` | local | GitHub release binary |

Install in table order. Tear down in reverse order. Reject missing version evidence, mixed versions, or any detected version outside this bundle for platform mutations.
