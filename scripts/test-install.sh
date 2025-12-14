#!/usr/bin/env bash
#
# k1s - Installation Tests
# Tests installation across different platforms using Docker
#
# Usage:
#   ./scripts/test-install.sh           # Run all tests
#   ./scripts/test-install.sh debian    # Test specific platform
#   ./scripts/test-install.sh --local   # Test on local machine only
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
VERSION="${K1S_VERSION:-v0.1.2}"
GITHUB_REPO="andrebassi/k1s"
PASSED=0
FAILED=0

# Print functions
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[PASS]${NC} $1"; ((PASSED++)); }
fail() { echo -e "${RED}[FAIL]${NC} $1"; ((FAILED++)); }
header() { echo -e "\n${CYAN}════════════════════════════════════════════════════════════${NC}"; echo -e "${CYAN}  $1${NC}"; echo -e "${CYAN}════════════════════════════════════════════════════════════${NC}\n"; }

# Detect current architecture for Docker
detect_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)             echo "amd64" ;;
    esac
}

ARCH=$(detect_arch)

# Test: Debian with .deb package
test_debian_deb() {
    header "Testing Debian (.deb package) - $ARCH"

    if docker run --rm debian:bookworm bash -c "
        apt-get update -qq >/dev/null 2>&1
        apt-get install -y -qq curl >/dev/null 2>&1
        echo 'Downloading .deb package...'
        curl -sLO https://github.com/$GITHUB_REPO/releases/download/$VERSION/k1s_${VERSION#v}_${ARCH}.deb
        echo 'Installing package...'
        dpkg -i ./k1s_${VERSION#v}_${ARCH}.deb
        echo ''
        echo 'Verifying installation...'
        k1s --version
        echo ''
        echo 'Testing help...'
        k1s --help | head -3
    " 2>&1; then
        success "Debian .deb installation"
    else
        fail "Debian .deb installation"
    fi
}

# Test: Ubuntu with .deb package
test_ubuntu_deb() {
    header "Testing Ubuntu 22.04 (.deb package) - $ARCH"

    if docker run --rm ubuntu:22.04 bash -c "
        apt-get update -qq >/dev/null 2>&1
        apt-get install -y -qq curl >/dev/null 2>&1
        echo 'Downloading .deb package...'
        curl -sLO https://github.com/$GITHUB_REPO/releases/download/$VERSION/k1s_${VERSION#v}_${ARCH}.deb
        echo 'Installing package...'
        dpkg -i ./k1s_${VERSION#v}_${ARCH}.deb
        echo ''
        echo 'Verifying installation...'
        k1s --version
        echo ''
        echo 'Testing help...'
        k1s --help | head -3
    " 2>&1; then
        success "Ubuntu .deb installation"
    else
        fail "Ubuntu .deb installation"
    fi
}

# Test: Fedora with .rpm package
test_fedora_rpm() {
    header "Testing Fedora (.rpm package) - $ARCH"

    if docker run --rm fedora:latest bash -c "
        echo 'Downloading .rpm package...'
        curl -sLO https://github.com/$GITHUB_REPO/releases/download/$VERSION/k1s-${VERSION#v}-1.${ARCH}.rpm
        echo 'Installing package...'
        dnf install -y ./k1s-${VERSION#v}-1.${ARCH}.rpm 2>&1 | tail -5
        echo ''
        echo 'Verifying installation...'
        k1s --version
        echo ''
        echo 'Testing help...'
        k1s --help | head -3
    " 2>&1; then
        success "Fedora .rpm installation"
    else
        fail "Fedora .rpm installation"
    fi
}

# Test: Rocky Linux (RHEL clone) with .rpm package
test_rocky_rpm() {
    header "Testing Rocky Linux 9 (.rpm package) - $ARCH"

    if docker run --rm rockylinux:9 bash -c "
        echo 'Downloading .rpm package...'
        curl -sLO https://github.com/$GITHUB_REPO/releases/download/$VERSION/k1s-${VERSION#v}-1.${ARCH}.rpm
        echo 'Installing package...'
        dnf install -y ./k1s-${VERSION#v}-1.${ARCH}.rpm 2>&1 | tail -5
        echo ''
        echo 'Verifying installation...'
        k1s --version
        echo ''
        echo 'Testing help...'
        k1s --help | head -3
    " 2>&1; then
        success "Rocky Linux .rpm installation"
    else
        fail "Rocky Linux .rpm installation"
    fi
}

# Test: Alpine with binary
test_alpine_binary() {
    header "Testing Alpine (binary) - $ARCH"

    local binary_arch="$ARCH"
    [[ "$ARCH" == "amd64" ]] && binary_arch="amd64"
    [[ "$ARCH" == "arm64" ]] && binary_arch="arm64"

    if docker run --rm alpine:latest sh -c "
        apk add --no-cache curl >/dev/null 2>&1
        echo 'Downloading binary...'
        curl -sL -o /usr/local/bin/k1s https://github.com/$GITHUB_REPO/releases/download/$VERSION/k1s-linux-${binary_arch}
        chmod +x /usr/local/bin/k1s
        echo ''
        echo 'Verifying installation...'
        k1s --version
        echo ''
        echo 'Testing help...'
        k1s --help | head -3
    " 2>&1; then
        success "Alpine binary installation"
    else
        fail "Alpine binary installation"
    fi
}

# Test: Debian with install.sh script
test_debian_script() {
    header "Testing Debian (install.sh script) - $ARCH"

    if docker run --rm debian:bookworm bash -c "
        apt-get update -qq >/dev/null 2>&1
        apt-get install -y -qq curl >/dev/null 2>&1
        echo 'Running install script...'
        curl -sSL https://raw.githubusercontent.com/$GITHUB_REPO/main/scripts/install.sh | bash -s -- --version $VERSION
    " 2>&1; then
        success "Debian install.sh script"
    else
        fail "Debian install.sh script"
    fi
}

# Test: Local macOS installation
test_local_macos() {
    header "Testing Local macOS Installation"

    if [[ "$(uname -s)" != "Darwin" ]]; then
        info "Skipping macOS test (not on macOS)"
        return
    fi

    # Test Homebrew installation
    if command -v brew &> /dev/null; then
        info "Testing Homebrew installation..."

        # Uninstall if exists
        brew uninstall k1s 2>/dev/null || true
        brew untap andrebassi/k1s 2>/dev/null || true

        # Install
        if brew tap andrebassi/k1s && brew install k1s; then
            echo ""
            k1s --version
            success "macOS Homebrew installation"

            # Cleanup
            brew uninstall k1s
            brew untap andrebassi/k1s
        else
            fail "macOS Homebrew installation"
        fi
    else
        info "Homebrew not installed, testing binary installation..."

        local tmp_file="/tmp/k1s-test-$$"
        local binary_arch=$(detect_arch)

        if curl -sL -o "$tmp_file" "https://github.com/$GITHUB_REPO/releases/download/$VERSION/k1s-darwin-${binary_arch}"; then
            chmod +x "$tmp_file"
            echo ""
            "$tmp_file" --version
            rm -f "$tmp_file"
            success "macOS binary installation"
        else
            fail "macOS binary installation"
        fi
    fi
}

# Print summary
print_summary() {
    header "Test Summary"
    echo -e "  ${GREEN}Passed:${NC} $PASSED"
    echo -e "  ${RED}Failed:${NC} $FAILED"
    echo ""

    if [[ $FAILED -gt 0 ]]; then
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    else
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    fi
}

# Run all tests
run_all_tests() {
    info "Running all installation tests for k1s $VERSION"
    info "Architecture: $ARCH"
    echo ""

    # Docker tests
    test_debian_deb
    test_ubuntu_deb
    test_fedora_rpm
    test_alpine_binary

    # Local test
    test_local_macos

    print_summary
}

# Main
main() {
    case "${1:-all}" in
        debian)
            test_debian_deb
            print_summary
            ;;
        ubuntu)
            test_ubuntu_deb
            print_summary
            ;;
        fedora)
            test_fedora_rpm
            print_summary
            ;;
        rocky)
            test_rocky_rpm
            print_summary
            ;;
        alpine)
            test_alpine_binary
            print_summary
            ;;
        script)
            test_debian_script
            print_summary
            ;;
        local|--local)
            test_local_macos
            print_summary
            ;;
        all|--all)
            run_all_tests
            ;;
        --help|-h)
            echo "Usage: $0 [PLATFORM]"
            echo ""
            echo "Platforms:"
            echo "  debian    Test Debian .deb installation"
            echo "  ubuntu    Test Ubuntu .deb installation"
            echo "  fedora    Test Fedora .rpm installation"
            echo "  rocky     Test Rocky Linux .rpm installation"
            echo "  alpine    Test Alpine binary installation"
            echo "  script    Test install.sh script"
            echo "  local     Test local machine installation"
            echo "  all       Run all tests (default)"
            echo ""
            echo "Environment variables:"
            echo "  K1S_VERSION   Version to test (default: v0.1.2)"
            echo ""
            ;;
        *)
            echo "Unknown platform: $1"
            echo "Run '$0 --help' for usage"
            exit 1
            ;;
    esac
}

main "$@"
