#!/usr/bin/env bash
# update-common.sh - Update bin-common-handler across all dependent services
#
# Usage:
#   ./scripts/update-common.sh                  # Verify common, then update all
#   ./scripts/update-common.sh --parallel 4     # Update 4 services at a time
#   ./scripts/update-common.sh --skip-lint      # Skip linting (faster)
#   ./scripts/update-common.sh --update-only    # Skip common-handler verification
#
# This script automates the most error-prone workflow in the monorepo:
# after changing bin-common-handler, every dependent service must run
# go mod tidy && go mod vendor && go generate && go clean -testcache && go test
#
# IMPORTANT: Always clears the test cache to prevent false passes.

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

PARALLEL=1
SKIP_LINT=false
UPDATE_ONLY=false

usage() {
    echo "Usage: $(basename "$0") [options]"
    echo ""
    echo "Options:"
    echo "  --parallel N       Update N services in parallel (default: 1)"
    echo "  --skip-lint        Skip golangci-lint step"
    echo "  --update-only      Skip bin-common-handler verification"
    echo "  -h, --help         Show this help"
    echo ""
    echo "Examples:"
    echo "  $(basename "$0")                      # Full workflow, sequential"
    echo "  $(basename "$0") --parallel 4          # 4 services at a time"
    echo "  $(basename "$0") --skip-lint           # Faster, no linting"
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --parallel)
            PARALLEL="${2:?--parallel requires a number}"
            shift 2
            ;;
        --skip-lint)
            SKIP_LINT=true
            shift
            ;;
        --update-only)
            UPDATE_ONLY=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            usage >&2
            exit 1
            ;;
    esac
done

# ── Step 1: Verify bin-common-handler ───────────────────────────────────────

if [[ "$UPDATE_ONLY" == "false" ]]; then
    echo -e "${BOLD}Step 1: Verify bin-common-handler${NC}"
    lint_flag=""
    if [[ "$SKIP_LINT" == "true" ]]; then lint_flag="--skip-lint"; fi

    if ! "$SCRIPT_DIR/verify.sh" $lint_flag bin-common-handler; then
        echo -e "\n${RED}bin-common-handler verification failed. Fix errors before updating services.${NC}"
        exit 1
    fi
    echo -e "\n${GREEN}bin-common-handler verified successfully.${NC}\n"
else
    echo -e "${YELLOW}Skipping bin-common-handler verification (--update-only).${NC}\n"
fi

# ── Step 2: Collect dependent services ──────────────────────────────────────

echo -e "${BOLD}Step 2: Collecting dependent services${NC}"

services=()
for dir in "$REPO_ROOT"/bin-*/; do
    svc=$(basename "$dir")
    # Skip common-handler itself and services without go.mod
    if [[ "$svc" == "bin-common-handler" ]]; then continue; fi
    if [[ ! -f "$dir/go.mod" ]]; then continue; fi
    services+=("$svc")
done

echo -e "Found ${#services[@]} dependent services\n"

# ── Step 3: Update and verify all services ──────────────────────────────────

echo -e "${BOLD}Step 3: Updating all services (parallel=$PARALLEL)${NC}\n"

update_service() {
    local svc="$1"
    local svc_path="$REPO_ROOT/$svc"
    local start
    start=$(date +%s)

    cd "$svc_path"

    # Run update steps
    local failed=false

    echo -e "  ${DIM}[$svc]${NC} go mod tidy..."
    if ! go mod tidy > /tmp/update_common_${svc}_$$ 2>&1; then
        echo -e "  ${RED}✗ $svc: go mod tidy failed${NC}"
        tail -5 /tmp/update_common_${svc}_$$ | sed 's/^/    /'
        rm -f /tmp/update_common_${svc}_$$
        return 1
    fi

    echo -e "  ${DIM}[$svc]${NC} go mod vendor..."
    if ! go mod vendor >> /tmp/update_common_${svc}_$$ 2>&1; then
        echo -e "  ${RED}✗ $svc: go mod vendor failed${NC}"
        tail -5 /tmp/update_common_${svc}_$$ | sed 's/^/    /'
        rm -f /tmp/update_common_${svc}_$$
        return 1
    fi

    echo -e "  ${DIM}[$svc]${NC} go generate..."
    if ! go generate ./... >> /tmp/update_common_${svc}_$$ 2>&1; then
        echo -e "  ${RED}✗ $svc: go generate failed${NC}"
        tail -5 /tmp/update_common_${svc}_$$ | sed 's/^/    /'
        rm -f /tmp/update_common_${svc}_$$
        return 1
    fi

    echo -e "  ${DIM}[$svc]${NC} go clean -testcache && go test..."
    go clean -testcache 2>/dev/null
    if ! go test ./... >> /tmp/update_common_${svc}_$$ 2>&1; then
        echo -e "  ${RED}✗ $svc: go test failed${NC}"
        tail -10 /tmp/update_common_${svc}_$$ | sed 's/^/    /'
        rm -f /tmp/update_common_${svc}_$$
        return 1
    fi

    if [[ "$SKIP_LINT" == "false" ]]; then
        echo -e "  ${DIM}[$svc]${NC} golangci-lint..."
        if ! golangci-lint run -v --timeout 5m >> /tmp/update_common_${svc}_$$ 2>&1; then
            echo -e "  ${RED}✗ $svc: golangci-lint failed${NC}"
            tail -10 /tmp/update_common_${svc}_$$ | sed 's/^/    /'
            rm -f /tmp/update_common_${svc}_$$
            return 1
        fi
    fi

    local elapsed=$(( $(date +%s) - start ))
    echo -e "  ${GREEN}✓ $svc${NC} ${DIM}(${elapsed}s)${NC}"
    rm -f /tmp/update_common_${svc}_$$
    return 0
}

if [[ "$PARALLEL" -gt 1 ]]; then
    # Parallel mode
    results_dir=$(mktemp -d)
    trap 'rm -rf "$results_dir"' EXIT

    printf '%s\n' "${services[@]}" | xargs -I{} -P "$PARALLEL" bash -c '
        svc="$1"
        results_dir="$2"
        script_dir="$3"
        skip_lint="$4"
        repo_root="$5"
        svc_path="$repo_root/$svc"

        cd "$svc_path"
        output=""
        failed=false

        go mod tidy 2>&1 || failed=true
        if [[ "$failed" == "false" ]]; then go mod vendor 2>&1 || failed=true; fi
        if [[ "$failed" == "false" ]]; then go generate ./... 2>&1 || failed=true; fi
        if [[ "$failed" == "false" ]]; then
            go clean -testcache 2>/dev/null
            go test ./... 2>&1 || failed=true
        fi
        if [[ "$failed" == "false" && "$skip_lint" == "false" ]]; then
            golangci-lint run -v --timeout 5m 2>&1 || failed=true
        fi

        if [[ "$failed" == "true" ]]; then
            echo "FAIL" > "$results_dir/$svc"
        else
            echo "PASS" > "$results_dir/$svc"
        fi
    ' _ {} "$results_dir" "$SCRIPT_DIR" "$SKIP_LINT" "$REPO_ROOT"

    # Collect results
    pass_count=0
    fail_count=0
    failed_services=()

    echo -e "\n${BOLD}Results${NC}"
    for svc in "${services[@]}"; do
        if [[ -f "$results_dir/$svc" && "$(cat "$results_dir/$svc")" == "PASS" ]]; then
            echo -e "  ${GREEN}✓${NC} $svc"
            pass_count=$((pass_count + 1))
        else
            echo -e "  ${RED}✗${NC} $svc"
            fail_count=$((fail_count + 1))
            failed_services+=("$svc")
        fi
    done
else
    # Sequential mode
    pass_count=0
    fail_count=0
    failed_services=()

    for svc in "${services[@]}"; do
        if update_service "$svc"; then
            pass_count=$((pass_count + 1))
        else
            fail_count=$((fail_count + 1))
            failed_services+=("$svc")
        fi
    done
fi

# ── Summary ─────────────────────────────────────────────────────────────────

echo -e "\n${BOLD}══════════════════════════════════════${NC}"
echo -e "${BOLD}Common Handler Update Summary${NC}"
echo -e "${BOLD}══════════════════════════════════════${NC}"
echo -e "  ${GREEN}$pass_count passed${NC}"
echo -e "  ${RED}$fail_count failed${NC}"

if [[ "$fail_count" -gt 0 ]]; then
    echo -e "\n${RED}Failed services:${NC}"
    printf '  - %s\n' "${failed_services[@]}"
    echo -e "\n${YELLOW}Run manually to see full errors:${NC}"
    echo -e "  ./scripts/verify.sh <service-name>"
    exit 1
else
    echo -e "\n${GREEN}All services updated and verified successfully.${NC}"
    exit 0
fi
