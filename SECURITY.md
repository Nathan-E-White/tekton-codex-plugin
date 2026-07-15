# Security policy

Report vulnerabilities privately through GitHub Security Advisories for `Nathan-E-White/tekton-codex-plugin`. Do not include kubeconfig files, bearer tokens, Secret data, private keys, raw production logs, or unredacted evidence.

Version 0.1.x receives security fixes while it is the current supported line. The plugin is a local operator tool, not a network service; its trust boundary is the local user, local kubeconfig, configured manifest roots, and explicitly selected Kubernetes context.
