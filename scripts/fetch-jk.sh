#!/usr/bin/env bash
# Fetches the pinned jk release binary for every platform lazyjenkins
# bundles it for, into internal/embedjk/binaries/. Run before a release
# build with -tags embedjk (goreleaser's before.hooks does this
# automatically; run it by hand only if building an embedjk binary
# locally — plain `go build .` doesn't need this at all).
#
# Keep JK_VERSION in sync with internal/embedjk/embed.go's Version const.
set -euo pipefail

JK_VERSION="0.0.36"
DEST="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/internal/embedjk/binaries"
mkdir -p "$DEST"

# ours=theirs: our GOOS_GOARCH naming vs. jk's release asset naming.
targets=(
  "darwin_arm64=darwin_arm64"
  "darwin_amd64=darwin_x86_64"
  "linux_arm64=linux_arm64"
  "linux_amd64=linux_x86_64"
)

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

for entry in "${targets[@]}"; do
  ours="${entry%%=*}"
  theirs="${entry#*=}"
  url="https://github.com/avivsinai/jenkins-cli/releases/download/v${JK_VERSION}/jk_${JK_VERSION}_${theirs}.tar.gz"
  echo "fetching jk ${JK_VERSION} for ${ours} (${theirs})..."
  curl -sSfL "$url" -o "$tmp/jk_${ours}.tar.gz"
  tar -xzf "$tmp/jk_${ours}.tar.gz" -C "$tmp" jk
  mv "$tmp/jk" "$DEST/jk_${ours}"
  chmod +x "$DEST/jk_${ours}"
done

echo "done: $DEST"
