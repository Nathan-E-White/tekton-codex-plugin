---
name: tekton-pipelines-as-code
description: Configure and diagnose GitHub-first Tekton Pipelines-as-Code workflows using GitHub Apps, webhooks, repository CRs, `.tekton/` PipelineRuns, checks, and `/test`, `/retest`, or `/cancel` ChatOps.
---

# Tekton Pipelines as Code

GitHub is the only first-class SCM provider in v0.1.0.

1. Inspect PAC health, GitHub App references, webhook delivery, Repository CR status, and `.tekton/` files.
2. Keep GitHub App private keys and webhook secrets in referenced Kubernetes Secrets; never pass them through MCP arguments or evidence.
3. Validate `.tekton/` PipelineRuns and event filters before planning changes.
4. Confirm check-run permissions and branch protection expectations.
5. Use signed webhook fixtures in ordinary tests and a credentialed manual smoke test before release.

Do not claim GitHub integration is verified until event delivery, PipelineRun creation, check reporting, and ChatOps have all succeeded live.
