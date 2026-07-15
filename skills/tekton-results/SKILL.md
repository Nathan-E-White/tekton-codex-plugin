---
name: tekton-results
description: Install, configure, query, inspect, and diagnose upstream Tekton Results records, retention, stored logs, API connectivity, watcher health, and database-backed run history.
---

# Tekton Results

1. Inspect API, watcher, database connectivity, TLS, retention configuration, and CLI availability.
2. Use `tekton_query_results` for bounded queries; never return unbounded record or log sets.
3. Correlate Results records with live TaskRuns and PipelineRuns by UID.
4. Treat database backup as external infrastructure evidence, not as a Tekton MCP operation.
5. Before staging or production teardown, require a non-secret backup reference and a successful resource export.

Do not store raw Results credentials, database URLs containing passwords, or complete logs in evidence JSONL.
