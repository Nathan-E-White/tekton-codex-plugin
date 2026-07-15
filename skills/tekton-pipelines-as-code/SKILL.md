---
name: tekton-pipelines-as-code
description: Configure and diagnose GitHub-first Tekton Pipelines-as-Code workflows using GitHub Apps, webhooks, repository CRs, `.tekton/` PipelineRuns, checks, and `/test`, `/retest`, or `/cancel` ChatOps.
---

# Tekton Pipelines as Code

GitHub is the only first-class SCM provider in v0.2.0.

1. Inspect PAC health, GitHub App references, webhook delivery, Repository CR status, and `.tekton/` files.
2. Keep GitHub App private keys and webhook secrets in referenced Kubernetes Secrets; never pass them through MCP arguments or evidence.
3. Validate `.tekton/` PipelineRuns and event filters before planning changes.
4. Confirm check-run permissions and branch protection expectations.
5. Use `scripts/verify-webhook-fixture.sh` for the secret-free signature contract in ordinary CI. This proves fixture integrity only; it does not prove GitHub delivery or check reporting.
6. Before release, run `.github/workflows/github-app-smoke.yml` against a pull request containing `.tekton/pac-smoke.yaml`. Supply only the documented GitHub environment secret and variable names; never print their values or persist the private key.
7. Treat the credentialed smoke as complete only when one evidence run observes webhook delivery, PipelineRun creation, a check run owned by the configured App, and successful `/test`, `/retest`, and `/cancel` transitions.

Do not claim GitHub integration is verified until event delivery, PipelineRun creation, check reporting, and ChatOps have all succeeded live.
