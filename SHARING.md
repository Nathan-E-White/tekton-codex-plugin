# Share the Tekton Codex Plugin

There are three practical distribution lanes. They are not interchangeable.

## 1. Public GitHub release

This is the default public lane for the local STDIO plugin.

1. Merge a reviewed version branch to `main`.
2. Tag the release, for example `v0.2.0`, and push the tag.
3. Let the release workflow build macOS and Linux archives for arm64 and amd64.
4. Confirm each archive contains the complete plugin, including skills, MCP
   launcher, dashboard resource, brand assets, legal notices, and the matching
   binary.
5. Publish `SHA256SUMS`, SBOMs, and build provenance with the archives.
6. Point users to the release notes and the install commands below.

Recipients can verify an archive with:

```sh
shasum -a 256 -c SHA256SUMS
```

## 2. Public Codex marketplace repository

The repository acts as a one-plugin marketplace through
`.agents/plugins/marketplace.json`. Its entry uses a Git-backed `url` source
because the complete plugin lives at the repository root.

```sh
codex plugin marketplace add Nathan-E-White/tekton-codex-plugin --ref main
codex plugin add tekton@tekton-community
codex plugin list
```

Pin `--ref` to a release tag such as `v0.2.0` when a team values
reproducibility over automatic upgrades. Do not publish a local `source.path`
that only works on the maintainer's machine.

After installation, start a new Codex task so the refreshed skills, MCP tools,
and dashboard resource are loaded.

## 3. ChatGPT public Plugins Directory

The current server deliberately uses local STDIO and local kubeconfig. Public
ChatGPT submission instead requires a publicly reachable MCP server, domain
verification, a privacy policy, terms, test cases, accurate tool annotations,
and a verified publisher identity.

Do not expose users' kubeconfigs or turn this binary into a public multi-tenant
cluster proxy merely to satisfy that form. A public ChatGPT app would be a
separate architecture with OAuth, tenant isolation, auditable cluster
authorization, hosted evidence storage, and a new security ADR. Submit that MCP
server through the OpenAI plugin submission portal only after those controls
exist. See OpenAI's
[plugin submission guide](https://learn.chatgpt.com/docs/submit-plugins.md) and
[Apps SDK deployment guidance](https://developers.openai.com/apps-sdk/deploy).

## Workspace sharing

For a private team trial, open the installed plugin's details page in the
ChatGPT desktop app, select **Share**, and grant access to workspace members or
groups. Workspace sharing is not public publication.

## Release checklist

- `make validate` and race-enabled Go tests pass.
- The kind full-bundle job passes.
- Every tool advertises `ui://tekton/operations-dashboard-v1`.
- The gallery screenshot matches the current dashboard.
- The plugin manifest version matches the tag and archive name.
- License, privacy, terms, trademark, and non-endorsement notices ship.
- Release checksums, SBOMs, and provenance are present.
- A fresh-task `tekton_preflight` renders the dashboard from the packaged binary.
