# verify-service

Run mandatory pre-commit verification workflow for a service.

## Usage

```bash
/verify-service <service-name> [--with-deps]
```

## Arguments

- `service-name`: Name of the service to verify
  - Accepts: "call", "call-manager", or "bin-call-manager" (all work)
  - Will normalize to proper directory name
- `--with-deps`: Include dependency updates (adds `go get -u ./...` step)

## What This Does

Runs the mandatory 5-step verification workflow that MUST complete before any commit:

**Regular verification (default):**
1. `go mod tidy` - Clean up go.mod and go.sum
2. `go mod vendor` - Update vendored dependencies
3. `go generate ./...` - Regenerate mocks and generated code
4. `go test ./...` - Run all tests
5. `golangci-lint run -v --timeout 5m` - Check code quality

**With dependency updates (--with-deps flag):**
1. `go get -u ./...` - Update all dependencies to latest versions
2. `go mod tidy` - Clean up go.mod and go.sum
3. `go mod vendor` - Update vendored dependencies
4. `go generate ./...` - Regenerate mocks and generated code
5. `go test ./...` - Run all tests
6. `golangci-lint run -v --timeout 5m` - Check code quality

## Examples

```bash
# Regular verification after code changes
/verify-service call-manager

# Works with different name formats
/verify-service bin-call-manager
/verify-service call
/verify-service flow

# With dependency updates
/verify-service bin-api-manager --with-deps
```

## Exit Codes

- `0`: All steps passed - safe to commit
- `1`: At least one step failed - DO NOT commit

## Implementation

This command:
1. Normalizes service name (handles "call", "call-manager", "bin-call-manager")
2. Verifies service directory exists in monorepo
3. Navigates to service directory
4. Runs verification steps sequentially
5. Shows progress for each step with timing
6. Reports which step failed if any fail
7. Exits immediately on first failure

## Why This Command Exists

- Used before EVERY commit (dozens of times per day)
- Currently requires copy-pasting 5-command sequence from CLAUDE.md
- Easy to forget a step or run in wrong order
- Saves 30-60 seconds per commit
- Prevents broken commits from stale mocks, failing tests, or lint issues
