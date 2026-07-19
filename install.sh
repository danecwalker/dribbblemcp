#!/usr/bin/env bash
# Install dribbblemcp from GitHub Releases.
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/danecwalker/dribbblemcp/main/install.sh | bash
#   curl -fsSL ... | bash -s -- --version v1.0.0 --dir /usr/local/bin

set -euo pipefail

REPO="danecwalker/dribbblemcp"
BINARY="dribbblemcp"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-latest}"
VERIFY_CHECKSUM="${VERIFY_CHECKSUM:-1}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()  { printf "${BLUE}==>${NC} %s\n" "$*"; }
ok()   { printf "${GREEN}✓${NC} %s\n" "$*"; }
warn() { printf "${YELLOW}!${NC} %s\n" "$*"; }
die()  { printf "${RED}error:${NC} %s\n" "$*" >&2; exit 1; }

usage() {
  cat <<EOF
Install ${BINARY} (Dribbble MCP server)

Usage:
  install.sh [options]

Options:
  -h, --help              Show this help
  -d, --dir DIR           Install directory (default: \$HOME/.local/bin)
  -v, --version VER       Version tag (default: latest), e.g. v1.0.0
  --no-checksum           Skip checksum verification
  --print-path            Print install path only after install

Examples:
  curl -fsSL https://raw.githubusercontent.com/danecwalker/dribbblemcp/main/install.sh | bash
  curl -fsSL ... | bash -s -- --version v0.1.0
  curl -fsSL ... | bash -s -- --dir /usr/local/bin

Requirements:
  - curl or wget
  - tar (or unzip on Windows/Git Bash for .zip)
  - Chrome or Chromium at runtime
EOF
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

download() {
  local url="$1" out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$out"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$out" "$url"
  else
    die "need curl or wget"
  fi
}

http_get() {
  local url="$1"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url"
  else
    wget -qO- "$url"
  fi
}

detect_os() {
  case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
    linux*)  echo "Linux" ;;
    darwin*) echo "Darwin" ;;
    msys*|cygwin*|mingw*) echo "Windows" ;;
    *) die "unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "x86_64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) die "unsupported arch: $(uname -m)" ;;
  esac
}

resolve_version() {
  if [[ "$VERSION" != "latest" ]]; then
    # Normalize: allow 1.0.0 or v1.0.0
    if [[ "$VERSION" != v* ]]; then
      VERSION="v${VERSION}"
    fi
    echo "$VERSION"
    return
  fi
  local tag
  tag="$(http_get "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n1)"
  [[ -n "$tag" ]] || die "could not resolve latest release from GitHub API"
  echo "$tag"
}

# Parse args
PRINT_PATH=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    -d|--dir) INSTALL_DIR="$2"; shift 2 ;;
    -v|--version) VERSION="$2"; shift 2 ;;
    --no-checksum) VERIFY_CHECKSUM=0; shift ;;
    --print-path) PRINT_PATH=1; shift ;;
    *) die "unknown option: $1 (try --help)" ;;
  esac
done

need_cmd tar
OS="$(detect_os)"
ARCH="$(detect_arch)"

log "Resolving version…"
TAG="$(resolve_version)"
ok "Version ${TAG}"

EXT="tar.gz"
if [[ "$OS" == "Windows" ]]; then
  EXT="zip"
  need_cmd unzip
fi

ASSET="${BINARY}_${OS}_${ARCH}.${EXT}"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
ASSET_URL="${BASE_URL}/${ASSET}"
SUMS_URL="${BASE_URL}/checksums.txt"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

log "Downloading ${ASSET}…"
download "$ASSET_URL" "${TMPDIR}/${ASSET}"
ok "Downloaded"

if [[ "$VERIFY_CHECKSUM" == "1" ]]; then
  log "Verifying checksum…"
  download "$SUMS_URL" "${TMPDIR}/checksums.txt"
  (
    cd "$TMPDIR"
    if command -v sha256sum >/dev/null 2>&1; then
      grep " ${ASSET}\$" checksums.txt | sha256sum -c -
    elif command -v shasum >/dev/null 2>&1; then
      # macOS
      expected="$(grep " ${ASSET}\$" checksums.txt | awk '{print $1}')"
      actual="$(shasum -a 256 "$ASSET" | awk '{print $1}')"
      [[ "$expected" == "$actual" ]] || die "checksum mismatch for ${ASSET}"
    else
      warn "no sha256sum/shasum; skipping checksum verification"
    fi
  )
  ok "Checksum OK"
fi

log "Extracting…"
if [[ "$EXT" == "zip" ]]; then
  unzip -q "${TMPDIR}/${ASSET}" -d "$TMPDIR/out"
else
  mkdir -p "$TMPDIR/out"
  tar -xzf "${TMPDIR}/${ASSET}" -C "$TMPDIR/out"
fi

BIN_PATH="$(find "$TMPDIR/out" -type f -name "$BINARY" -o -name "${BINARY}.exe" | head -n1)"
[[ -n "$BIN_PATH" ]] || die "binary not found inside archive"

mkdir -p "$INSTALL_DIR"
install -m 755 "$BIN_PATH" "${INSTALL_DIR}/${BINARY}"
ok "Installed to ${INSTALL_DIR}/${BINARY}"

# PATH hint
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    warn "${INSTALL_DIR} is not on your PATH"
    echo "    Add this to your shell config:"
    echo "      export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

# Chrome hint
if ! command -v google-chrome >/dev/null 2>&1 \
  && ! command -v chromium >/dev/null 2>&1 \
  && [[ ! -x "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" ]]; then
  warn "Chrome/Chromium not detected. dribbblemcp needs a local browser at runtime."
  echo "    Install Google Chrome, or set CHROME_PATH to the binary."
fi

echo
ok "dribbblemcp ${TAG} ready"
echo
echo "Configure Grok:"
echo "  grok mcp add dribbble -- ${INSTALL_DIR}/${BINARY}"
echo
echo "Or add to ~/.grok/config.toml:"
echo "  [mcp_servers.dribbble]"
echo "  command = \"${INSTALL_DIR}/${BINARY}\""
echo "  enabled = true"
echo "  startup_timeout_sec = 60"
echo "  tool_timeout_sec = 120"
echo
echo "Skill (optional — copy into a project or ~/.grok/skills):"
echo "  https://github.com/${REPO}/tree/main/.grok/skills/dribbble-inspiration"
echo

if [[ "$PRINT_PATH" == "1" ]]; then
  echo "${INSTALL_DIR}/${BINARY}"
fi

"${INSTALL_DIR}/${BINARY}" --version 2>/dev/null || true
