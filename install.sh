#!/usr/bin/env bash
set -euo pipefail

# OpenClaw MCP Server installer
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/MagicBurg/openclaw-mcp-server/main/install.sh | sh
#   ./install.sh [--prefix ~/.local] [--go-version 1.24.0]

BINARY_NAME="openclaw-mcp-server"
REPO_URL="https://github.com/MagicBurg/openclaw-mcp-server.git"
DEFAULT_PREFIX="$HOME/.local"
MIN_GO_MAJOR=1
MIN_GO_MINOR=24
DEFAULT_GO_VERSION="1.24.0"

# Colors (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED='' GREEN='' YELLOW='' BLUE='' NC=''
fi

info()  { echo -e "${BLUE}==> ${NC}$*"; }
ok()    { echo -e "${GREEN}==> ${NC}$*"; }
warn()  { echo -e "${YELLOW}==> ${NC}$*"; }
err()   { echo -e "${RED}==> ERROR: ${NC}$*" >&2; }
die()   { err "$@"; exit 1; }

# --- Parse arguments ---
PREFIX="$DEFAULT_PREFIX"
GO_VERSION="$DEFAULT_GO_VERSION"
SKIP_TESTS=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --prefix)    PREFIX="$2"; shift 2 ;;
        --go-version) GO_VERSION="$2"; shift 2 ;;
        --skip-tests) SKIP_TESTS=true; shift ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --prefix DIR      Install directory (default: $DEFAULT_PREFIX)"
            echo "  --go-version VER  Go version to install if missing (default: $DEFAULT_GO_VERSION)"
            echo "  --skip-tests      Skip running tests before install"
            echo "  -h, --help        Show this help"
            exit 0
            ;;
        *) die "Unknown option: $1" ;;
    esac
done

INSTALL_DIR="${PREFIX}/bin"

# --- Detect OS and architecture ---
detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$ARCH" in
        x86_64)  ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        armv7l)  ARCH="armv6l" ;;
        *) die "Unsupported architecture: $ARCH" ;;
    esac

    case "$OS" in
        linux|darwin) ;;
        *) die "Unsupported OS: $OS" ;;
    esac

    echo "${OS}/${ARCH}"
}

# --- Check Go installation ---
check_go() {
    local go_cmd=""

    # Try common Go binary locations
    if command -v go &>/dev/null; then
        go_cmd="go"
    elif [ -x "/usr/local/go/bin/go" ]; then
        go_cmd="/usr/local/go/bin/go"
    elif [ -x "$HOME/go/bin/go" ]; then
        go_cmd="$HOME/go/bin/go"
    fi

    if [ -n "$go_cmd" ]; then
        local version
        version="$($go_cmd version 2>/dev/null | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)"
        local major minor
        major="${version%%.*}"
        minor="${version#*.}"

        if [ "$major" -gt "$MIN_GO_MAJOR" ] || { [ "$major" -eq "$MIN_GO_MAJOR" ] && [ "$minor" -ge "$MIN_GO_MINOR" ]; }; then
            GO_CMD="$go_cmd"
            ok "Found Go $($go_cmd version 2>/dev/null | grep -oP 'go[0-9]+\.[0-9]+\.[0-9]+')"
            return 0
        else
            warn "Found Go $version but need >= $MIN_GO_MAJOR.$MIN_GO_MINOR"
        fi
    fi

    return 1
}

# --- Install Go ---
install_go() {
    local platform="$1"
    local go_tarball="go${GO_VERSION}.${platform/\//-}.tar.gz"
    local go_url="https://go.dev/dl/${go_tarball}"
    local go_dir="/usr/local/go"

    info "Installing Go ${GO_VERSION}..."

    # Check if we can write to /usr/local
    if [ -w "/usr/local" ] || [ "$(id -u)" -eq 0 ]; then
        info "Downloading ${go_url}..."
        download "$go_url" "${WORK_DIR}/${go_tarball}"

        # Remove old Go installation if present
        [ -d "$go_dir" ] && rm -rf "$go_dir"

        info "Extracting to ${go_dir}..."
        tar -C /usr/local -xzf "${WORK_DIR}/${go_tarball}"

        GO_CMD="${go_dir}/bin/go"
        ok "Go ${GO_VERSION} installed to ${go_dir}"
    else
        # Use Go's dl tool for user-local install
        info "No write access to /usr/local. Installing Go ${GO_VERSION} via go dl tool..."

        if command -v go &>/dev/null; then
            local dl_version="go${GO_VERSION}"
            GOTOOLCHAIN=auto go install "golang.org/dl/${dl_version}@latest" 2>/dev/null || true
            local dl_bin="$HOME/go/bin/${dl_version}"
            if [ -x "$dl_bin" ]; then
                "$dl_bin" download
                GO_CMD="$dl_bin"
                ok "Go ${GO_VERSION} installed via dl tool"
                return 0
            fi
        fi

        die "Cannot install Go automatically. Please install Go >= ${MIN_GO_MAJOR}.${MIN_GO_MINOR} manually: https://go.dev/dl/"
    fi
}

# --- Download helper ---
download() {
    local url="$1" dest="$2"
    if command -v curl &>/dev/null; then
        curl -fsSL "$url" -o "$dest"
    elif command -v wget &>/dev/null; then
        wget -q "$url" -O "$dest"
    else
        die "Neither curl nor wget found. Install one of them first."
    fi
}

# --- Ensure source is available ---
ensure_source() {
    # Already in project directory (ran as ./install.sh)
    if [ -f "go.mod" ] && grep -q "openclaw-mcp-server" go.mod 2>/dev/null; then
        SOURCE_DIR="$(pwd)"
        CLONED=false
        return
    fi

    # Piped via curl | sh — need to clone
    info "Cloning ${REPO_URL}..."
    if ! command -v git &>/dev/null; then
        die "git is required. Install git first."
    fi

    SOURCE_DIR="${WORK_DIR}/openclaw-mcp-server"
    git clone --depth 1 "$REPO_URL" "$SOURCE_DIR" 2>/dev/null
    CLONED=true
    ok "Cloned to ${SOURCE_DIR}"
}

# --- Main ---
main() {
    echo ""
    echo "  OpenClaw MCP Server Installer"
    echo "  =============================="
    echo ""

    # Create temp working directory
    WORK_DIR="$(mktemp -d)"
    trap 'rm -rf "$WORK_DIR"' EXIT

    # Detect platform
    info "Detecting platform..."
    PLATFORM="$(detect_platform)"
    ok "Platform: ${PLATFORM}"

    # Check/install Go
    info "Checking Go installation..."
    GO_CMD=""
    if ! check_go; then
        install_go "$PLATFORM"
    fi

    [ -z "$GO_CMD" ] && die "Go not found after installation attempt"

    # Ensure GOTOOLCHAIN=auto for version management
    export GOTOOLCHAIN=auto

    # Get source code
    ensure_source
    cd "$SOURCE_DIR"

    # Download dependencies
    info "Downloading dependencies..."
    $GO_CMD mod download
    ok "Dependencies downloaded"

    # Run tests
    if [ "$SKIP_TESTS" = false ]; then
        info "Running tests..."
        if $GO_CMD test ./... ; then
            ok "All tests passed"
        else
            die "Tests failed. Fix the issues or use --skip-tests to skip."
        fi
    else
        warn "Skipping tests (--skip-tests)"
    fi

    # Build
    info "Building ${BINARY_NAME}..."
    $GO_CMD build -o "${WORK_DIR}/${BINARY_NAME}" ./cmd/server/
    ok "Built ${BINARY_NAME}"

    # Install
    info "Installing to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR"
    if [ -w "$INSTALL_DIR" ]; then
        cp "${WORK_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        warn "No write access to ${INSTALL_DIR}. Trying with sudo..."
        sudo cp "${WORK_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    fi
    ok "Installed to ${INSTALL_DIR}/${BINARY_NAME}"

    # Verify
    if command -v "$BINARY_NAME" &>/dev/null; then
        ok "Verified: $(command -v "$BINARY_NAME")"
    else
        if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
            warn "${INSTALL_DIR} is not in your PATH. Add it with:"
            echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
        fi
    fi

    # Done
    echo ""
    ok "Installation complete!"
    echo ""
    echo "  Next steps:"
    echo ""
    echo "  1. Configure your OpenClaw gateway connection:"
    echo "     export OPENCLAW_URL=http://127.0.0.1:18789"
    echo "     export OPENCLAW_TOKEN=your-gateway-token"
    echo ""
    echo "  2. Run the server:"
    echo "     ${BINARY_NAME}                          # stdio mode"
    echo "     ${BINARY_NAME} --transport http         # HTTP mode"
    echo ""
    echo "  See https://github.com/MagicBurg/openclaw-mcp-server for full docs."
    echo ""
}

main "$@"
