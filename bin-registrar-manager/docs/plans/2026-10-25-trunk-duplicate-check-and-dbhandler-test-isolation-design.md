# Trunk duplicate check and dbhandler test isolation

## Ticket review
- `trunkhandler.Create` treats a failed domain lookup like "not found" and continues. Only one such place exists under `pkg/`.
- `GetByDomainName` returns a typed not-found error that wraps `dbhandler.ErrNotFound` (`errors.Is` works through `Unwrap`); other errors are wrapped with `errors.Wrap`. `TrunkGetByDomainName` returns `ErrNotFound` only for an empty list; cache failures fall through to the DB and list errors are returned as is.
- Error mapping (Create path): both "already exists" and a lookup error are plain errors, mapped to 500 by `errorResponse`. The duplicate response does not change; status codes are out of scope.
- Behavior change: when the lookup fails, creation is now refused instead of continuing.
- dbhandler tests: `-count=2` fails (23 tests) because all tests share one in-memory SQLite database (`cache=shared`) opened once in `TestMain`, and tests insert fixed ids and values without removing them (`UNIQUE constraint failed`; 11 test files use `dbTest`, none run in parallel). The cause is test code only.

## Change
1. `Create`: continue only when the lookup error satisfies `errors.Is(err, dbhandler.ErrNotFound)`; return other lookup errors (logged and wrapped); a successful lookup keeps returning "already exists".
2. Tests: lookup error stops creation (neither `UUIDCreate` nor `TrunkCreate` is called, enforced by gomock), not found (also once wrapped with `%w`) continues, found is rejected. Mocks returning a bare error for "not found" use `dbhandler.ErrNotFound` instead (`trunk_error_test.go:184`, `trunk_test.go:133`).
3. dbhandler tests: add `resetTestTables(t)` in `main_test.go`. It lists the tables from `sqlite_master` (`type='table'`, not `sqlite_%`), runs `DELETE FROM` on each with a timeout context and calls `t.Fatalf` on failure. It is the first statement of every top-level `Test*` that uses `dbTest` directly or through a helper; a subtest that reuses fixed ids calls it as well. It is safe with `-shuffle`, `-race` and `-count` because no test is parallel, the single connection serializes access and every test builds its own gomock controller.

## Verify
- The new lookup-error test fails on the unfixed code (new test applied alone) and passes after the fix.
- `go test -count=3 -shuffle=on -race` passes for `./pkg/dbhandler/...` and `./pkg/trunkhandler/...`; the failing output before the change is attached to the PR.
- A scripted check lists top-level tests that reach `dbTest` and confirms each calls `resetTestTables`.
- Service wide `go test -count=2 -shuffle=on ./...`, `golangci-lint`, CI green.
- No docs/ text describes the duplicate check; no sync needed.
