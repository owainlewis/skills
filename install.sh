#!/bin/sh
# Install the `skills` binary from GitHub Releases.
#
#   curl -fsSL https://raw.githubusercontent.com/owainlewis/skills/main/install.sh | sh
#
# Environment:
#   VERSION   release tag to install (default: latest), e.g. VERSION=v0.1.0
#   BINDIR    install directory (default: /usr/local/bin, fallback ~/.local/bin)
set -eu

REPO="owainlewis/skills"
BIN="skills"

os=$(uname -s)
case "$os" in
  Darwin) os=darwin ;;
  Linux)  os=linux ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

asset="${BIN}_${os}_${arch}.tar.gz"
if [ -n "${VERSION:-}" ]; then
  base="https://github.com/${REPO}/releases/download/${VERSION}"
else
  base="https://github.com/${REPO}/releases/latest/download"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading ${asset} ..." >&2
curl -fsSL "${base}/${asset}" -o "$tmp/$asset"

# Verify checksum when the checksums file is available.
if curl -fsSL "${base}/checksums.txt" -o "$tmp/checksums.txt" 2>/dev/null; then
  ( cd "$tmp" && grep " ${asset}\$" checksums.txt | (sha256sum -c - 2>/dev/null || shasum -a 256 -c -) ) \
    && echo "Checksum OK" >&2 || { echo "checksum verification failed" >&2; exit 1; }
fi

tar -xzf "$tmp/$asset" -C "$tmp"

# Choose an install directory we can write to.
bindir="${BINDIR:-/usr/local/bin}"
if [ ! -d "$bindir" ] || [ ! -w "$bindir" ]; then
  bindir="$HOME/.local/bin"
  mkdir -p "$bindir"
fi

install -m 0755 "$tmp/$BIN" "$bindir/$BIN" 2>/dev/null || {
  cp "$tmp/$BIN" "$bindir/$BIN" && chmod 0755 "$bindir/$BIN"
}

echo "Installed $BIN to $bindir/$BIN" >&2
case ":$PATH:" in
  *":$bindir:"*) ;;
  *) echo "Note: $bindir is not on your PATH." >&2 ;;
esac
"$bindir/$BIN" --version
