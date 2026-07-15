#!/bin/sh
set -eu

version=${1:-0.1.0}
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
dist="$root/dist"
mkdir -p "$dist"

for platform in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do
  os=${platform%/*}
  arch=${platform#*/}
  name="tekton-codex-plugin_${version}_${os}_${arch}"
  stage=$(mktemp -d)
  mkdir -p "$stage/$name/bin"
  GOOS=$os GOARCH=$arch CGO_ENABLED=0 GOFLAGS=-p=1 go build -trimpath -ldflags "-s -w" -o "$stage/$name/bin/tekton-mcp-$os-$arch" "$root/cmd/tekton-mcp"
  tar -C "$root" --exclude .git --exclude bin --exclude dist -cf - . | tar -C "$stage/$name" -xf -
  tar -C "$stage" -czf "$dist/$name.tar.gz" "$name"
  rm -rf "$stage"
done

cd "$dist"
shasum -a 256 ./*.tar.gz > SHA256SUMS
