# PR 2A — Typed Sentinels for PERMISSION / AUTH / INVALID_ARG

**Date:** 2026-04-26
**Branch:** `NOJIRA-typed-errors-pr2a`
**Scope:** Replace ~633 brittle `fmt.Errorf` strings in `bin-api-manager/pkg/servicehandler/` with the existing typed sentinels in `pkg/serviceerrors/`. End-to-end behavior is unchanged (translator section 2 maps these sentinels to the exact same canonical envelope that section 4 currently produces via substring fallback).

## Why

After the 15-PR canonical error envelope rollout, the translator at `bin-api-manager/server/error_translate.go` has two layers that resolve servicehandler errors:

- **Section 2 — sentinel match** (`errors.Is`): clean, structured.
- **Section 4 — substring match**: brittle, catches `fmt.Errorf` strings.

Servicehandler still emits raw `fmt.Errorf` strings (~670 sites total), so section 4 carries most of the load. Migrating to sentinels removes brittleness — a future typo like `fmt.Errorf("no perms")` would silently mismap.

## Scope

| Pattern | Count | Sentinel |
|---|---:|---|
| `fmt.Errorf("user has no permission*")` | 307 | `ErrPermissionDenied` |
| `fmt.Errorf("direct access is not supported*")` | 266 | `ErrDirectAccessNotSupported` |
| `fmt.Errorf("*does not belong*")` | 5 | `ErrPermissionDenied` |
| `fmt.Errorf("authentication required*" / "*unauthorized*")` | 53 | `ErrAuthenticationRequired` |
| `fmt.Errorf("*is required" / "only one of*")` | 2 | `ErrInvalidArgument` |
| **Total** | **~633** | — |

All sentinels already exist in `bin-api-manager/pkg/serviceerrors/sentinels.go`. No new sentinels are added in this PR.

## Migration patterns

**Pattern A — bare error (most common):**
```go
// before
return nil, fmt.Errorf("user has no permission")

// after
return nil, serviceerrors.ErrPermissionDenied
```

**Pattern B — error with context (preserve via `%w`):**
```go
// before
return nil, fmt.Errorf("user has no permission for customer %s", customerID)

// after
return nil, fmt.Errorf("%w: customer %s", serviceerrors.ErrPermissionDenied, customerID)
```

**Pattern C — multi-return:**
```go
// before
return nil, "", fmt.Errorf("either activeflow_id or call_id is required")

// after
return nil, "", fmt.Errorf("%w: either activeflow_id or call_id is required", serviceerrors.ErrInvalidArgument)
```

## Out of scope

- New sentinels (PR 2B): `IDENTITY_VERIFICATION_REQUIRED`, `STATE_INVALID`, `SERVICE_UNAVAILABLE`
- `not found` migration (PR 2C)
- Cross-service typed RPC contract (PR 2D)
- Removing translator section 4 (PR 2E, after all of the above)

## Files affected

All under `bin-api-manager/pkg/servicehandler/`. Highest-volume files:

- `customer.go`, `agent.go`, `flow.go`, `call.go`, `campaigns.go`, `conferences.go`, `queues.go`, `service_agents*.go`, `extensions.go`, `recordings.go`, `messages.go`, `numbers.go`, `outdial*.go`, `outplan.go`, `tag.go`, `transcribe.go`, `transfers.go`, `chat*.go`, `talk*.go`

Plus the corresponding `*_test.go` files where tests assert against error strings.

## Test impact

Many `*_test.go` files assert `assert.Equal(t, err.Error(), "user has no permission")` or similar. These must change to:
```go
require.True(t, errors.Is(err, serviceerrors.ErrPermissionDenied))
```

The `assertMissingAuthIdentity` / `assertErrorResponse` test helpers used in `server/*_test.go` already test the envelope shape — those don't change.

## Verification

In `bin-api-manager/`:
1. `go mod tidy && go mod vendor`
2. `go generate ./...`
3. `go test ./pkg/servicehandler/... ./server/...`
4. `golangci-lint run -v --timeout 5m`

## Risk

Low. Translator section 2 already maps each sentinel to the same canonical envelope as section 4 currently produces. End-to-end shape and HTTP status codes are unchanged. Only structural cleanup.
