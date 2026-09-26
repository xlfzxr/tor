#!/usr/bin/env bash
# torstack auto-installer for Linux / Termux. Copy-paste ready:
#   curl -fsSL https://raw.githubusercontent.com/xlfzxr/tor/main/install.sh | bash
set -eu

REPO_URL="${REPO_URL:-https://github.com/xlfzxr/tor.git}"
REPO_DIR="${TMPDIR:-/tmp}/torstack-install"
INSTALL_DIR="${HOME}/.local/bin"
TARGET_BIN="torstack"

if [ "$(id -u)" -eq 0 ] && [ -d /usr/local/bin ]; then
  INSTALL_DIR="/usr/local/bin"
fi
mkdir -p "$INSTALL_DIR"

need() { command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; return 1; }; }

echo "[1/4] Checking prerequisites (git, go)..."
need git
if ! command -v go >/dev/null 2>&1; then
  echo "Go toolchain not found." >&2
  if [ -n "${PREFIX:-}" ] && command -v pkg >/dev/null 2>&1; then
    echo "Termux detected. Run: pkg install golang git" >&2
  else
    echo "Install Go 1.22+ from https://go.dev/dl/" >&2
  fi
  exit 1
fi
go version

echo "[2/4] Cloning ${REPO_URL} ..."
rm -rf "$REPO_DIR"
git clone --depth 1 "$REPO_URL" "$REPO_DIR"
cd "$REPO_DIR"

# Support both legacy repo root and new torstack-go subdir
if [ -f "torstack-go/go.mod" ]; then
  cd torstack-go
fi

echo "[3/4] Building ${TARGET_BIN} ..."
go vet ./...
go build -o "${INSTALL_DIR}/${TARGET_BIN}" ./cmd/torstack

echo "[4/4] Verifying..."
"${INSTALL_DIR}/${TARGET_BIN}" --help | head -5 || "${INSTALL_DIR}/${TARGET_BIN}" help | head -5

echo ""
echo "Installation complete."
echo "Binary: ${INSTALL_DIR}/${TARGET_BIN}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "NOTE: ${INSTALL_DIR} is not in PATH. Add: export PATH=\"\$HOME/.local/bin:\$PATH\"" ;;
esac
echo "Run: ${TARGET_BIN} doctor -v"
