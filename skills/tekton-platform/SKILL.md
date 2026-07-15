---
name: tekton-platform
description: Inspect, install, reconcile, repair, or fully tear down the pinned upstream Tekton component bundle on Kubernetes. Use for component health, version compatibility, platform lifecycle, CRDs, controllers, or webhooks.
---

# Tekton Platform

1. Call `tekton_preflight` with the explicit context and intended profile.
2. Compare detected components to the pinned bundle in `references/supported-bundle.md`.
3. Use `tekton_plan_platform` with `install`, `reconcile`, `repair`, or `teardown`.
4. Review commands, namespaces, data-loss scope, expiry, and confirmation text.
5. Execute only the returned plan ID and exact confirmation token.

Reject older, newer, or mixed versions instead of improvising an upgrade. Install in dependency order and tear down in reverse order. For staging or production teardown, first use `tekton_export_teardown_backup` and supply an external Results database backup reference.

Use `scripts/kind-smoke.sh` as the disposable-cluster release gate. It covers the pinned bundle, reconciliation, repair, Pipelines and Triggers execution, Chains and Results evidence, cancellation, and reverse-order teardown. It does not replace the credentialed GitHub App smoke.
