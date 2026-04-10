#!/usr/bin/env bash
# doctor.sh - Check development environment prerequisites
#
# Usage: ./scripts/doctor.sh
#
# Verifies that all required tools are installed and at acceptable versions.

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
NC='\033[0m'

pass_count=0
warn_count=0
fail_count=0

pass() {
    echo -e "  ${GREEN}✓${NC} $1"
    pass_count=$((pass_count + 1))
}

warn() {
    echo -e "  ${YELLOW}!${NC} $1"
    warn_count=$((warn_count + 1))
}

fail() {
    echo -e "  ${RED}✗${NC} $1"
    fail_count=$((fail_count + 1))
}

header() {
    echo -e "\n${BOLD}$1${NC}"
}

# ── Go ──────────────────────────────────────────────────────────────────────
header "Go Toolchain"

if command -v go &>/dev/null; then
    go_version=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+(\.[0-9]+)?')
    go_major=$(echo "$go_version" | cut -d. -f1)
    go_minor=$(echo "$go_version" | cut -d. -f2)
    if [[ "$go_major" -ge 1 && "$go_minor" -ge 23 ]]; then
        pass "go $go_version"
    else
        warn "go $go_version (recommend 1.23+)"
    fi
else
    fail "go not found (install from https://go.dev/dl/)"
fi

if command -v mockgen &>/dev/null; then
    pass "mockgen $(mockgen --version 2>&1 | head -1 || echo 'installed')"
else
    warn "mockgen not found (install: go install go.uber.org/mock/mockgen@latest)"
fi

# ── Linting ─────────────────────────────────────────────────────────────────
header "Linting"

if command -v golangci-lint &>/dev/null; then
    lint_version=$(golangci-lint version --format short 2>/dev/null || echo "unknown")
    pass "golangci-lint $lint_version"
else
    fail "golangci-lint not found (install: https://golangci-lint.run/welcome/install/)"
fi

# ── Containers ──────────────────────────────────────────────────────────────
header "Containers"

if command -v docker &>/dev/null; then
    docker_version=$(docker version --format '{{.Client.Version}}' 2>/dev/null || echo "unknown")
    pass "docker $docker_version"
else
    warn "docker not found (needed for make dev-up)"
fi

if docker compose version &>/dev/null 2>&1; then
    compose_version=$(docker compose version --short 2>/dev/null || echo "unknown")
    pass "docker compose $compose_version"
elif command -v docker-compose &>/dev/null; then
    compose_version=$(docker-compose version --short 2>/dev/null || echo "unknown")
    pass "docker-compose (legacy) $compose_version"
else
    warn "docker compose not found (needed for make dev-up)"
fi

# ── Database Tools ──────────────────────────────────────────────────────────
header "Database Tools"

if command -v mysql &>/dev/null; then
    mysql_version=$(mysql --version 2>/dev/null | grep -oP '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "unknown")
    pass "mysql client $mysql_version"
else
    warn "mysql client not found (optional, for direct DB access)"
fi

if command -v alembic &>/dev/null; then
    pass "alembic $(alembic --version 2>/dev/null | grep -oP '[0-9]+\.[0-9]+\.[0-9]+' || echo 'installed')"
else
    warn "alembic not found (optional, for schema migrations)"
fi

# ── Git ─────────────────────────────────────────────────────────────────────
header "Git"

if command -v git &>/dev/null; then
    git_version=$(git version | grep -oP '[0-9]+\.[0-9]+\.[0-9]+')
    pass "git $git_version"
else
    fail "git not found"
fi

# ── Summary ─────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}Summary${NC}"
echo -e "  ${GREEN}${pass_count} passed${NC}  ${YELLOW}${warn_count} warnings${NC}  ${RED}${fail_count} failed${NC}"

if [[ "$fail_count" -gt 0 ]]; then
    echo -e "\n${RED}Fix the failures above before developing.${NC}"
    exit 1
elif [[ "$warn_count" -gt 0 ]]; then
    echo -e "\n${YELLOW}Warnings are optional but recommended.${NC}"
    exit 0
else
    echo -e "\n${GREEN}Environment is ready.${NC}"
    exit 0
fi
