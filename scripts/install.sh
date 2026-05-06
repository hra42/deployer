#!/usr/bin/env sh
# Install the deployer CLI from the latest GitHub release.
#
# Usage:
#   curl -fsSL https://deployer.hra42.lol/install | sh
#   curl -fsSL https://deployer.hra42.lol/install | VERSION=v0.1.0 sh
#   curl -fsSL https://deployer.hra42.lol/install | INSTALL_DIR=$HOME/.local/bin sh
#
# Env vars:
#   VERSION      Tag to install (default: latest release).
#   INSTALL_DIR  Target directory (default: /usr/local/bin, or $HOME/.local/bin if not writable).

set -eu

REPO="hra42/deployer"
BIN="deployer"

err() { printf 'error: %s\n' "$*" >&2; exit 1; }
info() { printf '%s\n' "$*"; }

need() { command -v "$1" >/dev/null 2>&1 || err "missing required command: $1"; }
need uname
need tar
need mkdir
need install

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1"; }
  fetch_to() { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO- "$1"; }
  fetch_to() { wget -qO "$2" "$1"; }
else
  err "need curl or wget"
fi

# Detect OS
os_raw="$(uname -s)"
case "$os_raw" in
  Darwin) os="darwin" ;;
  Linux)  os="linux"  ;;
  *) err "unsupported OS: $os_raw (deployer ships darwin/arm64, linux/arm64, linux/amd64)" ;;
esac

# Detect arch
arch_raw="$(uname -m)"
case "$arch_raw" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64)  arch="amd64" ;;
  *) err "unsupported architecture: $arch_raw" ;;
esac

# darwin/amd64 is intentionally not built
if [ "$os" = "darwin" ] && [ "$arch" = "amd64" ]; then
  err "macOS x86_64 is not built. Install from source: go install github.com/${REPO}@latest"
fi

# Resolve version
version="${VERSION:-}"
if [ -z "$version" ]; then
  info "resolving latest release..."
  # Follow redirect from /releases/latest to /releases/tag/<version>
  if command -v curl >/dev/null 2>&1; then
    version="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest" | sed 's#.*/tag/##')"
  else
    version="$(wget --max-redirect=0 -qS "https://github.com/${REPO}/releases/latest" 2>&1 | awk '/Location:/ {print $2}' | tail -1 | sed 's#.*/tag/##')"
  fi
  [ -n "$version" ] || err "could not resolve latest release tag"
fi

asset="${BIN}-${os}-${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${version}/${asset}"
sha_url="${url}.sha256"

info "installing ${BIN} ${version} for ${os}/${arch}"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT HUP TERM

fetch_to "$url"     "$tmp/$asset"     || err "download failed: $url"
fetch_to "$sha_url" "$tmp/$asset.sha256" || err "download failed: $sha_url"

# Verify checksum. The .sha256 file is "<hash>  <filename>"; verify that exact line.
( cd "$tmp" && {
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum -c "$asset.sha256" >/dev/null
    elif command -v shasum >/dev/null 2>&1; then
      shasum -a 256 -c "$asset.sha256" >/dev/null
    else
      err "need sha256sum or shasum to verify checksum"
    fi
  } ) || err "checksum verification failed"

tar -xzf "$tmp/$asset" -C "$tmp"
bin_path="$tmp/${BIN}-${os}-${arch}"
[ -f "$bin_path" ] || err "expected binary not found in archive: ${BIN}-${os}-${arch}"
chmod +x "$bin_path"

# Pick install dir
install_dir="${INSTALL_DIR:-}"
if [ -z "$install_dir" ]; then
  if [ -w /usr/local/bin ] 2>/dev/null; then
    install_dir="/usr/local/bin"
  elif [ "$(id -u)" = "0" ]; then
    install_dir="/usr/local/bin"
  else
    install_dir="$HOME/.local/bin"
  fi
fi

mkdir -p "$install_dir"
target="$install_dir/$BIN"

if [ -w "$install_dir" ]; then
  install -m 0755 "$bin_path" "$target"
else
  info "elevating with sudo to write $target"
  sudo install -m 0755 "$bin_path" "$target"
fi

info "installed: $target"

# PATH hint
case ":${PATH}:" in
  *":${install_dir}:"*) ;;
  *)
    info ""
    info "note: $install_dir is not on your PATH. Add this to your shell rc:"
    info "    export PATH=\"$install_dir:\$PATH\""
    ;;
esac

# Sanity-run if on PATH
if command -v "$BIN" >/dev/null 2>&1; then
  "$BIN" --help >/dev/null 2>&1 || true
fi

info "done. run: $BIN --help"
