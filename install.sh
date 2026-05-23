#!/usr/bin/env sh
# Andromeda installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.sh | sh -s -- <app> [version]
#
# Examples:
#   curl -fsSL .../install.sh | sh -s -- sipp           # latest
#   curl -fsSL .../install.sh | sh -s -- feeds v0.4.0   # specific
#
# Env:
#   INSTALL_DIR   target dir (default: /usr/local/bin)

set -eu

REPO="stevedylandev/andromeda"
APP="${1:-}"
VERSION="${2:-}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

if [ -z "$APP" ]; then
  echo "error: app name required" >&2
  echo "usage: install.sh <app> [version]" >&2
  exit 1
fi

# Detect OS
uname_os=$(uname -s)
case "$uname_os" in
  Linux)  os="linux" ;;
  Darwin) os="macos" ;;
  *) echo "error: unsupported OS: $uname_os" >&2; exit 1 ;;
esac

# Detect arch
uname_arch=$(uname -m)
case "$uname_arch" in
  x86_64|amd64) arch="x86_64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "error: unsupported arch: $uname_arch" >&2; exit 1 ;;
esac

# Resolve version
if [ -z "$VERSION" ]; then
  echo "Looking up latest ${APP} release..."
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases?per_page=50" \
    | grep -o "\"tag_name\": *\"${APP}/v[^\"]*\"" \
    | head -1 \
    | sed -E 's/.*"'"${APP}"'\/(v[^"]+)".*/\1/')
  if [ -z "$VERSION" ]; then
    echo "error: no releases found for ${APP}" >&2
    exit 1
  fi
fi

# Strip leading v for archive filename (goreleaser strips it from {{.Version}})
ver_num="${VERSION#v}"
tag="${APP}/${VERSION}"
archive="${APP}_${ver_num}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${tag}/${archive}"

echo "Downloading ${url}"
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

if ! curl -fsSL "$url" -o "${tmpdir}/${archive}"; then
  echo "error: download failed — does ${tag} have a ${os}/${arch} build?" >&2
  exit 1
fi

tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"

if [ ! -f "${tmpdir}/${APP}" ]; then
  echo "error: binary ${APP} not found in archive" >&2
  exit 1
fi

chmod +x "${tmpdir}/${APP}"

# Install. Use sudo if INSTALL_DIR not writable.
if [ -w "$INSTALL_DIR" ] || [ ! -e "$INSTALL_DIR" ]; then
  mkdir -p "$INSTALL_DIR"
  mv "${tmpdir}/${APP}" "${INSTALL_DIR}/${APP}"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)"
  sudo mkdir -p "$INSTALL_DIR"
  sudo mv "${tmpdir}/${APP}" "${INSTALL_DIR}/${APP}"
fi

echo "Installed ${APP} ${VERSION} to ${INSTALL_DIR}/${APP}"
