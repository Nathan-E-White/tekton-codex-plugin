---
name: tekton-operations
description: Diagnose upstream Tekton component and workload failures, inspect bounded logs and events, export redacted evidence, cancel runs, clean history, and coordinate guarded incident or teardown workflows.
---

# Tekton Operations

1. Start with `tekton_preflight`, `tekton_list_runs`, and `tekton_get_run`.
2. Fetch logs only on demand with explicit byte and duration bounds.
3. Correlate controller conditions, Pod states, Kubernetes events, PipelineRun/TaskRun status, Chains annotations, and Results records.
4. Use plan tools for cancellation, deletion, cleanup, repair, or teardown.
5. Hand off the run ID, plan ID, cluster identity hash, context, namespace, profile, affected resource UIDs, and redacted evidence paths.

Do not infer environment profile from names. Never mutate global kubeconfig context or persist raw logs by default.
