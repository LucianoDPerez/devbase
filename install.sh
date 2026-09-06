#!/usr/bin/env bash
# DevBase installer — macOS / Linux
#   curl -fsSL https://raw.githubusercontent.com/LucianoDPerez/devbase/main/install.sh | bash
set -euo pipefail

REPO="LucianoDPerez/devbase"
BIN_DIR="${DEVBIN:-$HOME/.local/bin}"
VERSION="${DEVBASE_VERSION:-latest}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac
case "$os" in
  linux|darwin) ;;
  *) echo "unsupported OS: $os (Windows: use install.ps1)" >&2; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)"
fi
[ -n "$VERSION" ] || { echo "could not resolve version" >&2; exit 1; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
tgz="devbase_${VERSION#v}_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/download/$VERSION/$tgz"
echo "downloading $url"
curl -fsSL -o "$tmp/$tgz" "$url"
curl -fsSL -o "$tmp/checksums.txt" "https://github.com/$REPO/releases/download/$VERSION/checksums.txt"
(cd "$tmp" && grep "  $tgz\$" checksums.txt | sha256sum -c -) || {
  # macOS ships shasum, not sha256sum
  (cd "$tmp" && grep "  $tgz\$" checksums.txt | shasum -a 256 -c -)
}
tar xzf "$tmp/$tgz" -C "$tmp"
mkdir -p "$BIN_DIR"
mv "$tmp/devbase" "$BIN_DIR/devbase"
chmod +x "$BIN_DIR/devbase"
echo "installed $($BIN_DIR/devbase version) to $BIN_DIR/devbase"
case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) echo "NOTE: $BIN_DIR is not in PATH. Add: export PATH=\"$BIN_DIR:\$PATH\"" ;;
esac
