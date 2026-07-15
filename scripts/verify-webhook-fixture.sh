#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
actual=$(openssl dgst -sha256 -hmac 'tekton-ci-fixture' "$root/test/fixtures/github/webhook-push.json" | awk '{print $NF}')
expected=$(tr -d '[:space:]' < "$root/test/fixtures/github/webhook-push.sha256")
test "$actual" = "$expected"
printf '%s\n' "signed GitHub webhook fixture valid"
