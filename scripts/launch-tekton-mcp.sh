#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64) arch=amd64 ;;
  aarch64) arch=arm64 ;;
esac

binary="$root/bin/tekton-mcp-$os-$arch"
if [ ! -x "$binary" ]; then
  binary="$root/bin/tekton-mcp"
fi
if [ ! -x "$binary" ]; then
  printf '%s\n' "tekton MCP binary is not packaged for $os/$arch; install a complete release archive or run make build" >&2
  exit 1
fi
exec "$binary"
