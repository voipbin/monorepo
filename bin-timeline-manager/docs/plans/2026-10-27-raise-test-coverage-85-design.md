# VOIP-1659 bin-timeline-manager unit test coverage to 85%

## 1. Ticket review

Verdict: valid. Measured on origin/main a47f5a2a1 with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, excluding gens/, mock_*.go and `Code generated ... DO NOT EDIT` files, duplicates merged by max count: hand-written 2572/3143 = 81.83%. 85% needs 100 more statements (2672 covered), 85.5% needs 116 (2688). A reviewer re-measurement differed by one statement (2571), which does not change the conclusion; the doc commit differs from a47f5a2a1 only by this file. No open or merged PR covers this (`gh pr list --state all --search VOIP-1659`). Only test files are added.

## 2. Reachable ceiling

pkg/dbhandler (117/293, 39.9%) holds most of the gap. `dbHandler.conn` is the interface `clickhouse.Conn`, so a test double embedding that interface and overriding only Query and PrepareBatch (and likewise for Rows and Batch) reaches every ClickHouse query path with no signature change and no network. Reachable: event.go, correlation.go, event_insert.go, peer_event_insert.go, peer_event_read.go, 166 statements as the upper bound (conservative estimate about 150). cmd/timeline-control adds about 5 to 8 error paths. The five files have no unreachable branch. With the three WaitForConnection statements and timeline-control the ceiling is about 2745/3143 = 87.3%, above 85%, so no block.

Not targeted: runEventList after the input validation (NewHandler dials and WaitForConnection waits up to 10 seconds), cmd/timeline-manager main and runServices (external connections, signals), dbHandler connectionLoop (real ClickHouse dial), siphandler GCS reader (real GCS client), NewAnalysisHandler/NewAnalysisDBHandler constructors.

## 3. Plan (new test files only, no product code change)

Conventions: `Test_<Func>_<scenario>`, table style where it fits, no testify, synthetic values only. Test doubles live in pkg/dbhandler/fake_conn_test.go: `fakeConn` (embeds clickhouse.Conn, records query text and args, returns configured rows, batch or error), `fakeRows` (embeds driver.Rows; Next/Scan/Err/Close over a slice of rows; Scan assigns by destination type (string, time.Time, []string) through the destination pointers and fails on a destination count mismatch), `fakeBatch` (embeds driver.Batch; records Append arguments, configurable Append and Send errors, counts Send).

Every query and insert function also gets a `conn == nil` case where the existing test file lacks one. Tests that change `config.Get()` restore it with t.Cleanup and do not call t.Parallel.

1. correlation_test.go: ResourceActiveflowIDGet: query error wrapped; no row with nil Err returns "" and nil; no row with Err set returns wrapped error; scan error wrapped; success returns the id and the query text and args are asserted. ResourceExists: query error; no row; no row with Err; row returns true. CorrelatedResourceList: query error; scan error; rows.Err error after iteration; empty result is a non-nil empty slice; two rows are mapped field by field; args carry the activeflow id and the limit. withQueryTimeout: the derived context follows the parent's cancellation.
2. event_test.go additions: EventList and AggregatedEventList: query error wrapped; scan error; rows.Err; success maps timestamp, event type, publisher, data type and data; the query args match buildEventQuery output for a page token and a filter case; zero rows give a non-nil empty slice.
3. event_insert_test.go: EventBatchInsert: empty rows with a connection is a no-op (PrepareBatch not called); PrepareBatch error; Append error stops before Send; Send error; success appends one call per row with the exact five values in column order and sends once.
4. peer_event_insert_test.go additions: the same cases for PeerEventBatchInsert with the exact thirteen values in order.
5. peer_event_read_test.go additions: PeerEventList: query error; scan error; rows.Err; success decodes uuid strings, Peer and Local JSON and Data; an invalid peer or local JSON gives a zero Address and the row is kept (one input is a type mismatch, a numeric `target`, so decoding fills `type` before failing and the reset to a zero Address is observable; a syntactically broken input is an equivalent mutant for the reset and is not counted); empty peer or local text stays zero.
6. cmd/timeline-control/main_test.go additions: runEventList input validation errors (missing publisher, missing id, invalid id, missing events) set through viper where the existing tests do not already cover them; getMigrate with an empty address returns the required-address error; migrate up, down and status with an empty address return the wrapped init error (the test sets `config.Get().ClickHouseAddress` directly and restores it with t.Cleanup; the environment is not used because LoadGlobalConfig runs once). A non-empty address with no `migrations` directory in the package working directory fails in migrate.New at the source step before any connection, so no network is used; this covers the address-formatting statements.
7. WaitForConnection with a non-nil connection returns nil at the first tick.

## 4. Verify

- `bash scripts/check-test-conventions.sh` from the repository root; `go test ./...` and `golangci-lint run`.
- Hand-written total >= 85.5% with the measurement command above; per-package before/after table in the PR.
- Expected gain: dbhandler about +150, timeline-control about +5 to +8, which exceeds the 116 needed.
- Mutation sample on a scratch copy only (the worktree stays clean); each mutation must fail the named test: `errors.Wrap` removed in a query path (wrapped-error case), `rows.Err` check removed (Err case), `result := []*..{}` replaced by nil (empty non-nil case), Append column order swapped (exact-argument case), `len(rows) == 0` early return removed (no-op case), JSON fallback `Address{}` removed (type mismatch case only).
