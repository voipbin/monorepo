#!/usr/bin/env bash
# verify.sh - Run the mandatory 5-step verification workflow for a service
#
# Usage:
#   ./scripts/verify.sh <service-dir>           # Verify one service
#   ./scripts/verify.sh --all                    # Verify all Go services
#   ./scripts/verify.sh --all --parallel 4       # Verify all, 4 at a time
#
# Steps (in order):
#   1. go mod tidy
#   2. go mod vendor
#   3. go generate ./...
#   4. go clean -testcache && go test ./...
#   5. golangci-lint run -v --timeout 5m
#
# The test cache is ALWAYS cleared before testing. This prevents the #1 cause
# of false-passing tests after bin-common-handler changes.

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
MODE="single"
SERVICE=""
SKIP_LINT=false

usage() {
    echo "Usage: $(basename "$0") [options] <service-dir>"
    echo ""
    echo "Options:"
    echo "  --all              Verify all Go services"
    echo "  --parallel N       Run N services in parallel (default: 1)"
    echo "  --skip-lint        Skip golangci-lint step"
    echo "  -h, --help         Show this help"
    echo ""
    echo "Examples:"
    echo "  $(basename "$0") bin-agent-manager"
    echo "  $(basename "$0") --all --parallel 4"
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --all)
            MODE="all"
            shift
            ;;
        --parallel)
            PARALLEL="${2:?--parallel requires a number}"
            shift 2
            ;;
        --skip-lint)
            SKIP_LINT=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            echo "Unknown option: $1" >&2
            usage >&2
            exit 1
            ;;
        *)
            SERVICE="$1"
            shift
            ;;
    esac
done

# ── Helpers ─────────────────────────────────────────────────────────────────

step() {
    local step_num="$1"
    local label="$2"
    shift 2
    local start
    start=$(date +%s)

    echo -e "  ${DIM}[$step_num/5]${NC} $label..."
    if "$@" > /tmp/verify_step_output_$$ 2>&1; then
        local elapsed=$(( $(date +%s) - start ))
        echo -e "  ${GREEN}✓${NC} $label ${DIM}(${elapsed}s)${NC}"
        rm -f /tmp/verify_step_output_$$
        return 0
    else
        local elapsed=$(( $(date +%s) - start ))
        echo -e "  ${RED}✗${NC} $label ${DIM}(${elapsed}s)${NC}"
        echo -e "  ${RED}Output:${NC}"
        sed 's/^/    /' /tmp/verify_step_output_$$ | tail -30
        rm -f /tmp/verify_step_output_$$
        return 1
    fi
}

verify_service() {
    local svc_dir="$1"
    local svc_path="$REPO_ROOT/$svc_dir"

    if [[ ! -f "$svc_path/go.mod" ]]; then
        echo -e "${YELLOW}Skipping $svc_dir (no go.mod)${NC}"
        return 0
    fi

    echo -e "\n${BOLD}Verifying: $svc_dir${NC}"
    local start
    start=$(date +%s)
    local failed=false

    cd "$svc_path"

    step 1 "go mod tidy"     go mod tidy     || failed=true
    if [[ "$failed" == "false" ]]; then
        step 2 "go mod vendor"   go mod vendor   || failed=true
    fi
    if [[ "$failed" == "false" ]]; then
        step 3 "go generate"     go generate ./... || failed=true
    fi
    if [[ "$failed" == "false" ]]; then
        go clean -testcache 2>/dev/null
        step 4 "go test"         go test ./...   || failed=true
    fi
    if [[ "$failed" == "false" && "$SKIP_LINT" == "false" ]]; then
        step 5 "golangci-lint"   golangci-lint run -v --timeout 5m || failed=true
    elif [[ "$SKIP_LINT" == "true" ]]; then
        echo -e "  ${DIM}[5/5]${NC} golangci-lint ${YELLOW}(skipped)${NC}"
    fi

    local elapsed=$(( $(date +%s) - start ))

    if [[ "$failed" == "true" ]]; then
        echo -e "${RED}FAIL${NC} $svc_dir ${DIM}(${elapsed}s total)${NC}"
        return 1
    else
        echo -e "${GREEN}PASS${NC} $svc_dir ${DIM}(${elapsed}s total)${NC}"
        return 0
    fi
}

# ── Single service mode ─────────────────────────────────────────────────────

if [[ "$MODE" == "single" ]]; then
    if [[ -z "$SERVICE" ]]; then
        echo "Error: specify a service directory or use --all" >&2
        usage >&2
        exit 1
    fi
    verify_service "$SERVICE"
    exit $?
fi

# ── All services mode ───────────────────────────────────────────────────────

echo -e "${BOLD}Verifying all Go services (parallel=$PARALLEL)${NC}"

# Collect all services with go.mod
services=()
for dir in "$REPO_ROOT"/bin-*/; do
    svc=$(basename "$dir")
    if [[ -f "$dir/go.mod" ]]; then
        services+=("$svc")
    fi
done

echo -e "Found ${#services[@]} Go services\n"

if [[ "$PARALLEL" -gt 1 ]]; then
    # Parallel mode: run verify.sh for each service in subprocesses
    results_dir=$(mktemp -d)
    trap 'rm -rf "$results_dir"' EXIT

    printf '%s\n' "${services[@]}" | xargs -I{} -P "$PARALLEL" bash -c '
        svc="$1"
        results_dir="$2"
        script="$3"
        skip_lint="$4"
        lint_flag=""
        if [[ "$skip_lint" == "true" ]]; then lint_flag="--skip-lint"; fi
        if "$script" $lint_flag "$svc" >/dev/null 2>&1; then
            echo "PASS" > "$results_dir/$svc"
        else
            echo "FAIL" > "$results_dir/$svc"
        fi
    ' _ {} "$results_dir" "$SCRIPT_DIR/verify.sh" "$SKIP_LINT"

    # Collect results
    pass_count=0
    fail_count=0
    failed_services=()

    echo -e "\n${BOLD}Results${NC}"
    for svc in "${services[@]}"; do
        result_file="$results_dir/$svc"
        if [[ -f "$result_file" && "$(cat "$result_file")" == "PASS" ]]; then
            echo -e "  ${GREEN}✓${NC} $svc"
            pass_count=$((pass_count + 1))
        else
            echo -e "  ${RED}✗${NC} $svc"
            fail_count=$((fail_count + 1))
            failed_services+=("$svc")
        fi
    done

    echo -e "\n${BOLD}Summary${NC}: ${GREEN}$pass_count passed${NC}, ${RED}$fail_count failed${NC}"
    if [[ "$fail_count" -gt 0 ]]; then
        echo -e "${RED}Failed services:${NC}"
        printf '  %s\n' "${failed_services[@]}"
        exit 1
    fi
else
    # Sequential mode: run inline with full output
    pass_count=0
    fail_count=0
    failed_services=()

    for svc in "${services[@]}"; do
        if verify_service "$svc"; then
            pass_count=$((pass_count + 1))
        else
            fail_count=$((fail_count + 1))
            failed_services+=("$svc")
        fi
    done

    echo -e "\n${BOLD}Summary${NC}: ${GREEN}$pass_count passed${NC}, ${RED}$fail_count failed${NC}"
    if [[ "$fail_count" -gt 0 ]]; then
        echo -e "${RED}Failed services:${NC}"
        printf '  %s\n' "${failed_services[@]}"
        exit 1
    fi
fi
