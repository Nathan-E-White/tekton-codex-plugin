# Tekton Codex Plugin

`tekton` is a Codex plugin for upstream Tekton Pipelines, Triggers, Chains, Results, Pipelines-as-Code, and `tkn`. It combines eight focused skills with a local Go STDIO MCP server that reads the user's kubeconfig without changing the global context.

Version 0.1.0 supports one tested bundle only: Pipelines v1.14.0, Triggers v0.36.0, Chains v0.28.0, Results v0.19.0, Pipelines-as-Code v0.49.0, and `tkn` v0.45.0. It does not perform cross-version upgrades and does not support OpenShift.

## Safety model

Read tools require an explicit kubeconfig context and namespace. Every mutation requires an environment profile (`dev`, `stg`, or `prod`), a fresh immutable 15-minute plan, its plan ID, and the exact token `CONFIRM <action> <context> <plan-id>`. Cluster identity or state drift invalidates a plan. Staging and production teardown also require a non-secret resource export and an external Results database backup reference.

Evidence is redacted JSONL. The server never records kubeconfig contents, tokens, Secret data, private keys, or raw logs. Manifest files are confined to canonical configured roots after symlink resolution. Chains accepts keyless, KMS, or existing Secret references; OpenTofu owns KMS, IAM, and secret delivery.

## Local development

```sh
make test
make build
make validate
```

Configure optional manifest roots and evidence storage through `.mcp.json`:

```sh
export TEKTON_MCP_MANIFEST_ROOTS="$PWD/test/fixtures:$HOME/src/tekton-manifests"
export TEKTON_MCP_ARTIFACT_DIR="$HOME/.local/state/tekton-codex-plugin"
```

The launcher selects a packaged binary for `darwin` or `linux` on `amd64` or `arm64`. It never downloads an executable at MCP startup.

## Release

`scripts/package-release.sh 0.1.0` builds four complete plugin archives plus SHA-256 checksums. CI adds SBOMs and provenance attestations to GitHub Releases. See [the pinned bundle](references/supported-bundle.md), [the glossary](CONTEXT.md), and [the ADRs](docs/adr/).

Released under the Unlicense.
