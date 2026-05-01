#!/usr/bin/env bash
# setup.sh — install system dependencies (Go + Node.js) and project deps.
# Run once on a fresh server: bash setup.sh
set -euo pipefail

GO_VERSION="1.22.5"
NODE_MAJOR="20"

# ── helpers ────────────────────────────────────────────────────────────────────
info()  { echo "[setup] $*"; }
error() { echo "[setup] ERROR: $*" >&2; exit 1; }

# ── Go ─────────────────────────────────────────────────────────────────────────
if command -v go &>/dev/null; then
    info "Go already installed: $(go version)"
else
    info "Installing Go ${GO_VERSION}..."
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  GOARCH="amd64" ;;
        aarch64) GOARCH="arm64" ;;
        *)        error "Unsupported arch: $ARCH" ;;
    esac
    TARBALL="go${GO_VERSION}.linux-${GOARCH}.tar.gz"
    curl -fsSL "https://go.dev/dl/${TARBALL}" -o "/tmp/${TARBALL}"
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "/tmp/${TARBALL}"
    rm "/tmp/${TARBALL}"

    # Persist PATH for future shells
    PROFILE=/etc/profile.d/go.sh
    echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' > "$PROFILE"
    info "Go installed. PATH updated in $PROFILE"
fi

# Make Go available in this script's session
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin

go version || error "Go install failed"

# ── Node.js ────────────────────────────────────────────────────────────────────
if command -v node &>/dev/null; then
    info "Node.js already installed: $(node --version)"
else
    info "Installing Node.js ${NODE_MAJOR}.x via NodeSource..."
    if command -v apt-get &>/dev/null; then
        apt-get update -qq
        apt-get install -y -qq curl ca-certificates gnupg
        curl -fsSL "https://deb.nodesource.com/setup_${NODE_MAJOR}.x" | bash -
        apt-get install -y -qq nodejs
    elif command -v dnf &>/dev/null; then
        dnf install -y "nodejs:${NODE_MAJOR}"
    else
        error "Unsupported package manager — install Node.js ${NODE_MAJOR}+ manually then re-run."
    fi
fi

node --version || error "Node.js install failed"
npm --version

# ── Project dependencies ───────────────────────────────────────────────────────
info "Running make install..."
make install

info "Done. Next steps:"
info "  1. cp .env.example .env  (fill in your secrets)"
info "  2. make docker-up && make migrate-up"
info "  3. make build"
info "  4. make run-backend   (terminal 1)"
info "  5. make run-frontend  (terminal 2)"
