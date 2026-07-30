# Privacy

The Tekton Codex Plugin runs as a local STDIO process. It reads the kubeconfig
and Kubernetes context selected in each tool call. It does not operate a remote
service, download executables at startup, or send cluster data to the plugin
publisher.

Evidence written locally is redacted JSONL. Kubeconfig contents, bearer tokens,
Kubernetes Secret data, private keys, and raw logs are forbidden. Log responses
are bounded and are not persisted by default.

Codex or ChatGPT may process prompts and MCP results under the policies and
settings of the user's OpenAI account. Kubernetes API servers, registries,
GitHub, KMS systems, and Results databases remain governed by their respective
operators.

Report a privacy concern through the repository's
[issue tracker](https://github.com/Nathan-E-White/tekton-codex-plugin/issues).
