# Separate Chains configuration from key ownership

The Tekton plugin consumes and verifies keyless, KMS, or Kubernetes Secret references but never manages private key material. OpenTofu or another infrastructure owner provisions KMS, IAM, and secret delivery so Tekton evidence and OpenTofu state do not casually become competing secret stores.
