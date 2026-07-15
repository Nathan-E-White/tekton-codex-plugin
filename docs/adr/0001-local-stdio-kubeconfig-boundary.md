# Keep the MCP server local and kubeconfig-bound

The Tekton MCP server runs as a bundled Go STDIO process and uses an explicitly named context from the caller's local kubeconfig. This avoids storing multi-cluster credentials in a remote service and makes existing Kubernetes RBAC the authority, at the cost of limiting v0.1.0 to local Codex clients.
