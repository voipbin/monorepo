# Move projectconfig from bin-common-handler to bin-call-manager

**Date:** 2026-02-11

## Problem

The `projectconfig` package in `bin-common-handler` provides SIP domain names and a storage bucket name. However, only `bin-call-manager` imports it. Placing a single-consumer package in the shared library violates the principle that `bin-common-handler` should only contain code used across multiple services.

## Approach

1. Move `bin-common-handler/pkg/projectconfig/` to `bin-call-manager/pkg/projectconfig/`
2. Update imports in `bin-call-manager` (2 files)
3. Delete the package from `bin-common-handler`
4. Add a `bin-common-handler` admission rule to root `CLAUDE.md`: packages must be used by 3+ services

## Files Changed

- **Create** `bin-call-manager/pkg/projectconfig/main.go` (moved from common-handler)
- **Create** `bin-call-manager/pkg/projectconfig/main_test.go` (moved from common-handler)
- **Update** `bin-call-manager/models/common/domain.go` — change import path
- **Update** `bin-call-manager/pkg/recordinghandler/main.go` — change import path
- **Delete** `bin-common-handler/pkg/projectconfig/main.go`
- **Delete** `bin-common-handler/pkg/projectconfig/main_test.go`
- **Update** `CLAUDE.md` — add admission rule under Code Quality section

## Trade-offs

- If another service later needs `projectconfig`, it must either import from `bin-call-manager` (creating a cross-service dependency) or the package gets promoted back to `bin-common-handler`. The 3+ services rule makes this explicit.
