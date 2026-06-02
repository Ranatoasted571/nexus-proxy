#!/bin/sh
# NEXUS installer — downloads the right prebuilt binary for your OS/arch.
#
#   curl -fsSL https://raw.githubusercontent.com/lynuxis2026-pixel/nexus-proxy/main/install.sh | sh
#
# Override the source repo or version:
#   NEXUS_REPO=you/nexus NEXUS_VERSION=v0.2.0 sh install.sh
set -e

REPO="${NEXUS_REPO:-lynuxis2026-pixel/nexus-proxy}"
VERSION="${NEXUS_VERSION:-latest}"
BINARY="nexus"

err() { echo "error: $*" >&2; exit 1; }

# ── Detect platform ─────────────────────────────────────────────────────────
os="$(uname -s)"
case "$os" in
  Linux)  os="linux" ;;
  Darwin) os="darwin" ;;
  *) err "unsupported OS '$os' — use install.ps1 on Windows, or download manually" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64)  arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) err "unsupported architecture '$arch'" ;;
esac

asset="nexus_${os}_${arch}.tar.gz"
if [ "$VERSION" = "latest" ]; then
  url="https://github.com/${REPO}/releases/latest/download/${asset}"
else
  url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"
fi

# ── Download ────────────────────────────────────────────────────────────────
echo "→ Downloading ${asset} (${REPO} @ ${VERSION})"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$url" -o "$tmp/$asset" || err "download failed: $url"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tmp/$asset" "$url" || err "download failed: $url"
else
  err "neither curl nor wget found"
fi

tar -xzf "$tmp/$asset" -C "$tmp" || err "failed to extract $asset"
[ -f "$tmp/$BINARY" ] || err "binary '$BINARY' not found in archive"
chmod +x "$tmp/$BINARY"

# ── Install ─────────────────────────────────────────────────────────────────
dir="/usr/local/bin"
if [ ! -w "$dir" ]; then
  if command -v sudo >/dev/null 2>&1; then
    echo "→ Installing to $dir (sudo)"
    sudo mv "$tmp/$BINARY" "$dir/$BINARY" || err "install failed"
  else
    dir="$HOME/.local/bin"
    mkdir -p "$dir"
    mv "$tmp/$BINARY" "$dir/$BINARY" || err "install failed"
    case ":$PATH:" in
      *":$dir:"*) ;;
      *) echo "⚠ $dir is not on your PATH — add it to your shell profile" ;;
    esac
  fi
else
  mv "$tmp/$BINARY" "$dir/$BINARY" || err "install failed"
fi

echo "✓ Installed $BINARY to $dir/$BINARY"
echo
echo "Get started:"
echo "  nexus start"
echo "  export ANTHROPIC_BASE_URL=http://localhost:3000"
echo "  export ANTHROPIC_API_KEY=nexus-local"
echo "  claude"
