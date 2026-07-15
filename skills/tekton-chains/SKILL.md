---
name: tekton-chains
description: Configure and verify upstream Tekton Chains provenance, signatures, attestations, storage, keyless identity, KMS references, and existing Kubernetes Secret references without managing private key material.
---

# Tekton Chains

Tekton owns signing configuration and verification. OpenTofu owns KMS, IAM, and secret-delivery infrastructure.

1. Confirm Chains health and the credential-reference type.
2. Accept only keyless identity, KMS URI, or existing Secret name/namespace in handoffs.
3. Validate Chains configuration without retrieving Secret values.
4. Use `tekton_verify_attestation` to inspect run annotations and, when available, invoke Cosign verification.
5. Report identity, subject digest, predicate type, storage backend, and verification result without private material.

Never generate, rotate, export, display, or delete signing keys in this plugin.
