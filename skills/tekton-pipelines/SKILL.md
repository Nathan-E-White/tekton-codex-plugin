---
name: tekton-pipelines
description: Author, validate, apply, run, retry, cancel, inspect, and diagnose upstream Tekton Tasks, Pipelines, TaskRuns, and PipelineRuns. Use for Tekton YAML, parameters, workspaces, results, finally tasks, execution graphs, or run failures.
---

# Tekton Pipelines

Prefer `tekton.dev/v1`. Treat deprecated PipelineResources and ClusterTasks as blockers.

1. Inspect existing resources and runs.
2. Validate inline YAML or a file beneath an allowed manifest root with `tekton_validate`.
3. Check parameter types, workspace bindings, result dependencies, service accounts, timeouts, and resource requests.
4. Use `tekton_plan_resources` for apply/delete or `tekton_plan_run` for start/retry/cancel/cleanup.
5. Review and execute the immutable plan.
6. Use `tekton_get_run` and bounded `tekton_get_logs` for diagnosis.

Never place credentials in parameters, ConfigMaps, results, logs, or evidence. Use Kubernetes Secret references and workload identity.
