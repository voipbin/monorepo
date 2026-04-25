# PR 2B+C — New sentinels (Identity / State / Unavailable) + NOT_FOUND migration

**Date:** 2026-04-26
**Branch:** `NOJIRA-typed-errors-pr2bc`
**Scope:** Combined PR 2B (add 3 new sentinels for IDENTITY / STATE_INVALID / UNAVAILABLE patterns) and PR 2C (migrate ~14 local NOT_FOUND sites to existing `serviceerrors.ErrNotFound`).

## Why

After PR 2A migrated the 488 highest-volume sites (PERMISSION/AUTH/INVALID_ARG), the translator section 4 (substring fallback) still catches:
- `"identity verification"` (2 sites)
- `"already" / "deleted call" / "not active"` (2 sites)
- `"unavailable"` (2 source sites; 3 test mock returns ignored)
- `"not found"` (14 source sites; 7 test mock returns ignored)

These add up to ~20 source sites. PR 2A already covered the bulk; this PR closes the remaining typed-sentinel gap so PR 2D (cross-service typed RPC) and PR 2E (remove translator section 4) can land cleanly.

## Changes

### 1. New sentinels in `bin-api-manager/pkg/serviceerrors/sentinels.go`

```go
ErrIdentityVerificationRequired = stderrors.New("identity verification required")
ErrStateInvalid                 = stderrors.New("state invalid")
ErrServiceUnavailable           = stderrors.New("service unavailable")
```

### 2. Translator section 2 update (`server/error_translate.go`)

Add three new cases mapping each sentinel to its canonical envelope:

```go
case stderrors.Is(err, serviceerrors.ErrIdentityVerificationRequired):
    return cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "IDENTITY_VERIFICATION_REQUIRED",
        "Customer identity verification is required for this operation.").Wrap(err)
case stderrors.Is(err, serviceerrors.ErrStateInvalid):
    return cerrors.FailedPrecondition(commonoutline.ServiceNameAPIManager, "STATE_INVALID",
        "The operation is invalid for the current resource state.").Wrap(err)
case stderrors.Is(err, serviceerrors.ErrServiceUnavailable):
    return cerrors.Unavailable(commonoutline.ServiceNameAPIManager, "SERVICE_UNAVAILABLE",
        "An upstream service is temporarily unavailable.").Wrap(err)
```

These are added BEFORE the existing section 4 substring fallbacks so the typed path wins.

### 3. Site migrations

**Identity (2 sites, wrap form to preserve context):**
```go
// numbers.go:131
return nil, fmt.Errorf("%w: number purchase", serviceerrors.ErrIdentityVerificationRequired)

// call.go:84
return nil, nil, fmt.Errorf("%w: PSTN calls", serviceerrors.ErrIdentityVerificationRequired)
```

**State (2 sites, wrap form):**
```go
// boot.go:64
return nil, fmt.Errorf("%w: customer not active", serviceerrors.ErrStateInvalid)

// call.go:37
return nil, fmt.Errorf("%w: deleted call", serviceerrors.ErrStateInvalid)
```

**Unavailable (2 source sites, wrap form):**
```go
// timeline_sip.go:84, :157
return nil, fmt.Errorf("%w: upstream service unavailable", serviceerrors.ErrServiceUnavailable)
```

Test mock returns at `providercall_test.go:302`, `aggregated_events_test.go:469`, `timeline_test.go:478` are NOT migrated — these simulate upstream RPC errors that flow through the translator's section 4 (still active for upstream-string compatibility).

**NOT_FOUND (14 source sites, bare form):**
```go
// flow.go:28, activeflows.go:35, recording.go:35, billingaccount.go:36, storage_file.go:26, etc.
return nil, serviceerrors.ErrNotFound

// aggregated_events.go:101,:115,:126 (multi-return)
return nil, "", serviceerrors.ErrNotFound
```

Test mock returns at `providercall_test.go:179`, `billingaccount_test.go:773`, etc. are NOT migrated.

### 4. Test updates

Search for any tests that assert against these strings:
```bash
grep -rn 'err.Error()' bin-api-manager/pkg/servicehandler/*_test.go | grep -iE "identity verification|deleted call|customer not active|service unavailable|^.*\"not found\""
```

For each match, switch from string equality to `errors.Is(err, serviceerrors.Err...)`.

## Out of scope

- Removing translator section 4 (PR 2E — after 2D lands)
- Cross-service typed RPC contract for upstream managers (PR 2D)
- Migrating PR 2A patterns again (already done)

## Verification

```bash
cd bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Risk

Low. Behavior change: identity/state/unavailable/not-found errors continue to produce identical envelopes (translator section 4 already catches them; section 2 with new sentinels is just stricter). End-to-end HTTP responses are byte-identical.
