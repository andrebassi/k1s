#!/usr/bin/env bash
#
# k1s - Kubernetes TUI Debugger
# Installation script
#
# Usage:
#   curl -sSL https://cdn.jsdelivr.net/gh/andrebassi/k1s@main/scripts/install.sh | bash
#   curl -sSL https://cdn.jsdelivr.net/gh/andrebassi/k1s@main/scripts/install.sh | bash -s -- --version v0.1.2
#
# Options:
#   --version VERSION    Install specific version (default: latest)
#   --dir DIR            Install to specific directory (default: /usr/local/bin)
#   --help               Show this help message
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Defaults
VERSION="latest"
INSTALL_DIR="/usr/local/bin"
GITHUB_REPO="andrebassi/k1s"
IS_TERMUX=false

# Print functions
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[OK]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Show banner
show_banner() {
    echo ""
    echo -e "${GREEN}k1s${NC} - Kubernetes TUI Debugger"
    echo -e "One screen to see why your pod is broken."
    echo -e "${BLUE}────────────────────────────────────────────${NC}"
    echo ""
}

# Show help
show_help() {
    echo "Usage: install.sh [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --version VERSION    Install specific version (default: latest)"
    echo "  --dir DIR            Install to specific directory (default: /usr/local/bin)"
    echo "  --help               Show this help message"
    echo ""
    echo "Examples:"
    echo "  # Install latest version"
    echo "  curl -sSL https://cdn.jsdelivr.net/gh/andrebassi/k1s@main/scripts/install.sh | bash"
    echo ""
    echo "  # Install specific version"
    echo "  curl -sSL https://cdn.jsdelivr.net/gh/andrebassi/k1s@main/scripts/install.sh | bash -s -- --version v0.1.2"
    echo ""
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --version)
                VERSION="$2"
                shift 2
                ;;
            --dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done
}

# Detect Termux environment
detect_termux() {
    if [[ -n "$PREFIX" ]] && [[ "$PREFIX" == *"com.termux"* ]]; then
        IS_TERMUX=true
        INSTALL_DIR="$PREFIX/bin"
        info "Termux environment detected"
        info "Install directory: $INSTALL_DIR"
    fi
}

# Detect OS
detect_os() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$OS" in
        linux*)  OS="linux" ;;
        darwin*) OS="darwin" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *)       error "Unsupported OS: $OS" ;;
    esac

    # Detect Termux on Linux
    if [[ "$OS" == "linux" ]]; then
        detect_termux
    fi

    if [[ "$IS_TERMUX" == "true" ]]; then
        info "Detected OS: Android (Termux)"
    else
        info "Detected OS: $OS"
    fi
}

# Detect architecture
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)  ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        armv7l)        ARCH="armv7" ;;
        *)             error "Unsupported architecture: $ARCH" ;;
    esac
    info "Detected architecture: $ARCH"
}

# Get latest version
get_latest_version() {
    if [[ "$VERSION" == "latest" ]]; then
        info "Fetching latest version..."
        VERSION=$(curl -sI "https://github.com/$GITHUB_REPO/releases/latest" | grep -i "^location:" | sed 's/.*tag\///' | tr -d '\r\n')
        if [[ -z "$VERSION" ]]; then
            error "Failed to get latest version"
        fi
    fi
    info "Version to install: $VERSION"
}

# Build download URL
build_download_url() {
    local binary_name="k1s-${OS}-${ARCH}"

    # Use android-arm64 binary for Termux (requires PIE)
    if [[ "$IS_TERMUX" == "true" ]] && [[ "$ARCH" == "arm64" ]]; then
        binary_name="k1s-android-arm64"
    fi

    if [[ "$OS" == "windows" ]]; then
        binary_name="${binary_name}.exe"
    fi
    DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/$VERSION/$binary_name"
    info "Download URL: $DOWNLOAD_URL"
}

# Check if running with sudo when needed
check_permissions() {
    # Termux doesn't need sudo
    if [[ "$IS_TERMUX" == "true" ]]; then
        SUDO=""
        return
    fi

    if [[ ! -w "$INSTALL_DIR" ]]; then
        if [[ $EUID -ne 0 ]]; then
            warn "Installation directory $INSTALL_DIR requires root permissions"
            SUDO="sudo"
        fi
    else
        SUDO=""
    fi
}

# Download and install
install_binary() {
    # Use TMPDIR for Termux, /tmp otherwise
    local tmp_dir="${TMPDIR:-/tmp}"
    local tmp_file="$tmp_dir/k1s-$$"

    info "Downloading k1s $VERSION..."
    if ! curl -sL "$DOWNLOAD_URL" -o "$tmp_file"; then
        error "Failed to download k1s"
    fi

    chmod +x "$tmp_file"

    info "Installing to $INSTALL_DIR/k1s..."
    $SUDO mkdir -p "$INSTALL_DIR"
    $SUDO mv "$tmp_file" "$INSTALL_DIR/k1s"

    success "k1s installed successfully!"
}

# Verify installation
verify_installation() {
    info "Verifying installation..."

    if ! command -v k1s &> /dev/null; then
        # Check if install dir is in PATH
        if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
            warn "$INSTALL_DIR is not in your PATH"
            warn "Add it with: export PATH=\"\$PATH:$INSTALL_DIR\""
        fi
        # Try direct execution
        if [[ -x "$INSTALL_DIR/k1s" ]]; then
            echo ""
            "$INSTALL_DIR/k1s" --version
            success "Installation verified!"
        else
            error "Installation failed - binary not found"
        fi
    else
        echo ""
        k1s --version
        success "Installation verified!"
    fi
}

# Check for package managers (informational only in non-interactive mode)
try_package_manager() {
    # Check if running interactively
    local interactive=false
    if [[ -t 0 ]]; then
        interactive=true
    fi

    # macOS - Homebrew
    if [[ "$OS" == "darwin" ]] && command -v brew &> /dev/null; then
        info "Homebrew detected. You can also install via:"
        echo "  brew tap andrebassi/k1s && brew install k1s"
        echo ""
        if [[ "$interactive" == "true" ]]; then
            read -p "Install via Homebrew instead? [y/N] " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                brew tap andrebassi/k1s
                brew install k1s
                success "Installed via Homebrew!"
                k1s --version
                exit 0
            fi
        fi
    fi

    # Linux - Check for apt/dnf/yum (skip for Termux)
    if [[ "$OS" == "linux" ]] && [[ "$IS_TERMUX" != "true" ]]; then
        if command -v apt &> /dev/null; then
            info "Debian/Ubuntu detected. .deb package available:"
            echo "  curl -LO https://github.com/$GITHUB_REPO/releases/download/$VERSION/k1s_${VERSION#v}_${ARCH}.deb"
            echo "  sudo apt install ./k1s_${VERSION#v}_${ARCH}.deb"
            echo ""
        elif command -v dnf &> /dev/null || command -v yum &> /dev/null; then
            info "RHEL/Fedora detected. .rpm package available:"
            echo "  curl -LO https://github.com/$GITHUB_REPO/releases/download/$VERSION/k1s-${VERSION#v}-1.${ARCH}.rpm"
            echo "  sudo dnf install ./k1s-${VERSION#v}-1.${ARCH}.rpm"
            echo ""
        fi
    fi

    # Termux info
    if [[ "$IS_TERMUX" == "true" ]]; then
        info "Installing k1s for Termux (Android)"
        info "Make sure you have kubectl configured for your cluster"
        echo ""
    fi
}

# Main
main() {
    show_banner
    parse_args "$@"
    detect_os
    detect_arch
    get_latest_version
    try_package_manager
    build_download_url
    check_permissions
    install_binary
    verify_installation

    echo ""
    success "k1s is ready to use!"
    echo ""
    echo "Quick start:"
    echo "  k1s              # Start k1s"
    echo "  k1s -n kube-system  # Start in specific namespace"
    echo "  k1s --help       # Show help"
    echo ""
}

main "$@"
