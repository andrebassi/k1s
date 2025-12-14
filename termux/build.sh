TERMUX_PKG_HOMEPAGE=https://github.com/andrebassi/k1s
TERMUX_PKG_DESCRIPTION="Kubernetes TUI Debugger - One screen to see why your pod is broken"
TERMUX_PKG_LICENSE="MIT"
TERMUX_PKG_MAINTAINER="@andrebassi"
TERMUX_PKG_VERSION="0.1.3"
TERMUX_PKG_SRCURL=https://github.com/andrebassi/k1s/archive/refs/tags/v${TERMUX_PKG_VERSION}.tar.gz
TERMUX_PKG_SHA256=SKIP
TERMUX_PKG_AUTO_UPDATE=true
TERMUX_PKG_BUILD_IN_SRC=true

termux_step_make() {
	termux_setup_golang

	go build \
		-buildmode=pie \
		-trimpath \
		-ldflags "-s -w -X main.version=${TERMUX_PKG_VERSION}" \
		-o k1s \
		./cmd/k1s
}

termux_step_make_install() {
	install -Dm700 k1s "${TERMUX_PREFIX}/bin/k1s"
}
