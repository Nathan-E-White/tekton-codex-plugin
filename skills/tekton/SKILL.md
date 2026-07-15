---
name: tekton
description: Route broad or multi-step upstream Tekton CI/CD requests to the correct platform, Pipelines, Triggers, Pipelines-as-Code, Chains, Results, or operations workflow. Use when the Tekton concern spans components or is not yet clear.
---

# Tekton

Use this as the entrypoint for the upstream-only Tekton plugin.

## Route the request

- Dashboard or visual tool selection: run `tekton_preflight` with explicit
  context and namespace; its MCP Apps output opens the shared dashboard for all
  twelve tools.

- Component install, repair, reconcile, or teardown: `$tekton-platform`
- Tasks, Pipelines, TaskRuns, or PipelineRuns: `$tekton-pipelines`
- EventListeners, bindings, templates, or interceptors: `$tekton-triggers`
- GitHub Apps, `.tekton/`, checks, or ChatOps: `$tekton-pipelines-as-code`
- Provenance, signatures, attestations, or verification: `$tekton-chains`
- Run history, retention, records, or stored logs: `$tekton-results`
- Health, bounded logs, incident diagnosis, or cleanup: `$tekton-operations`

## Shared contract

1. Inspect before mutation with `tekton_preflight` and the relevant read tools.
2. Require an explicit kubeconfig context, namespace where applicable, and `dev|stg|prod` profile for every mutation plan.
3. Use plan tools before `tekton_execute_plan`. Never construct or reuse confirmation tokens yourself.
4. Treat a plan as single-use, cluster-bound, drift-sensitive, and valid for at most 15 minutes.
5. Never expose kubeconfig data, tokens, Secret values, private keys, or unbounded logs.
6. Stop on unsupported or mixed component versions. v0.2.0 does not perform cross-version upgrades.
7. Treat dashboard actions exactly like direct MCP calls. The UI does not relax
   profile, plan, confirmation, drift, backup, path, or redaction rules.

Read `references/supported-bundle.md` at the plugin root when exact component versions or install order matter.
