# Trunk duplicate check and dbhandler test isolation

## Ticket review
- Duplicate check: `trunkhandler.Create` treats any `GetByDomainName` error with a nil trunk as "not found" and goes on creating. Confirmed in `pkg/trunkhandler/trunk.go` (`if err == nil || tmp != nil`). No other occurrence of the pattern under `pkg/`.
- `GetByDomainName` returns a typed not-found error wrapping `dbhandler.ErrNotFound` (`errors.Is` works through `Unwrap`); other errors are wrapped with `errors.Wrap`. `TrunkGetByDomainName` returns `ErrNotFound` only for an empty list; cache failures fall through to the DB and list errors are returned as is.
- Behavior change: when the lookup fails, creation is now refused instead of continuing. Duplicate detection relies on the `deleted=false` filter, so a deleted trunk's domain stays reusable.
- Error mapping (Create path): the "already exists" error is a plain error, which `errorResponse` maps to 500. A lookup error is also a plain error, so it maps to 500 too. The duplicate response does not change (its status code is out of scope).
- `registrar_trunks.domain_name` has a non-unique index only (migration baseline), so the application check is the only duplicate guard.
- Out of scope: the gap between lookup and insert is not closed by this change (no DB unique constraint).
- dbhandler tests: `-count=2` fails (23 tests) because all tests share one in-memory SQLite database (`cache=shared`) created once in `TestMain`, and tests insert fixed IDs and values without removing them (`UNIQUE constraint failed` on ids, realms, customer ids; 11 test files use `dbTest`, none run in parallel). The cause is test code only.

## Change
1. `Create`: continue only when the lookup error is `dbhandler.ErrNotFound`; return other lookup errors (wrapped, logged); `err == nil` keeps returning "already exists".
2. Tests: lookup error stops creation (no `TrunkCreate`), not found continues, found is rejected. Existing mocks that return a bare error for "not found" are changed to `dbhandler.ErrNotFound` (`trunk_error_test.go:184`, `trunk_test.go:133`).
3. dbhandler tests: add `resetTestTables(t)` in `main_test.go` that deletes all rows of every table created by `scripts/database_scripts_test` (registrar_* and ps_*); call it first in every test that uses `dbTest`.

## Verify
- New lookup-error test fails before the fix and passes after.
- `go test -count=3 -shuffle=on -race ./pkg/dbhandler/...` passes; failing output before the fix is kept on the card.
- `grep dbTest` shows every test function using it calls `resetTestTables(t)`.
- Service wide `go test -count=2 -shuffle=on ./...`, `golangci-lint`, CI green.
- No docs/ text describes the duplicate check; no sync needed.
