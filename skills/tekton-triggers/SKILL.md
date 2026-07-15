---
name: tekton-triggers
description: Configure and diagnose upstream Tekton Triggers resources including EventListeners, Triggers, TriggerBindings, TriggerTemplates, interceptors, webhook payload extraction, and event delivery.
---

# Tekton Triggers

1. Inspect the EventListener service account, Service, deployment, bindings, templates, and interceptor references.
2. Validate payload paths and parameter names end to end.
3. Validate generated TaskRun or PipelineRun resources before planning an apply.
4. Use `tekton_plan_resources`; execute only its reviewed plan.
5. Diagnose with Kubernetes events, EventListener conditions, controller logs, and bounded request evidence.

Require webhook signature validation before exposing an EventListener. Do not persist raw webhook payloads when they may contain tokens or personal data.
