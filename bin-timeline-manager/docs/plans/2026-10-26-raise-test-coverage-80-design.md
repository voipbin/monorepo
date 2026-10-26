# VOIP-1657 bin-timeline-manager unit test coverage to 80%

## 1. Ticket review

Verdict: valid. Measured on origin/main 161e9748 (the service directory is unchanged since the ticket baseline a03bccd8) with
`GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, excluding gens/, mock_*.go and `Code generated ... DO NOT EDIT`
files, duplicate blocks merged by max count: hand-written 2474/3143 = 78.71% (raw profile total 72.8%, it includes generated code).
80.0% needs 41 more statements, 80.5% needs 57. No open or merged PR covers this (`gh pr list --search VOIP-1657`). Only test files are added.

## 2. Baseline (hand-written, before)

| package | covered/total | % |
|---|---|---|
| cmd/timeline-control | 73/142 | 51.4 |
| cmd/timeline-manager | 2/199 | 1.0 |
| internal/config | 82/85 | 96.5 |
| models/analysis | 8/9 | 88.9 |
| models/sipmessage | 67/67 | 100 |
| models/verdict | 20/20 | 100 |
| pkg/analysisdbhandler | 137/160 | 85.6 |
| pkg/analysishandler | 781/845 | 92.4 |
| pkg/dbhandler | 117/293 | 39.9 |
| pkg/eventhandler | 109/109 | 100 |
| pkg/homerhandler | 181/187 | 96.8 |
| pkg/listenhandler | 316/365 | 86.6 |
| pkg/peereventhandler | 23/23 | 100 |
| pkg/siphandler | 399/473 | 84.4 |
| pkg/subscribehandler | 159/166 | 95.8 |

Measurement note: pkg/subscribehandler differs by one statement between runs (158 or 159 of 166, timing dependent), so the total is 2473 or 2474
of 3143 (78.68% or 78.71%). The final figure is taken from one run and the margin below covers this.

Ceiling: of 669 uncovered statements, cmd/timeline-manager main (197, external connections), the ClickHouse paths of pkg/dbhandler (about 176; a fake of the driver connection is possible but costs more than it returns, so this is a conservative exclusion, and with only the card's unreachable types excluded the ceiling is about 90%) and the GCS client calls in pkg/siphandler/gcsreader_impl.go (36) are excluded, and the unreachable error
returns (squirrel ToSql, ApplyFields, json.Marshal, always-nil returns) are excluded too. Also excluded: cmd/timeline-control (69 uncovered; most commands open a MySQL or ClickHouse connection, the argument-validation and empty-address branches are reachable but are conservatively left out of the ceiling and are a reserve). Remaining reachable statements: 669 - 197 - 176 - 36 - 69 - about 25 unreachable returns = about 166, so the ceiling is (2473 + 166)/3143 = about 84%, above 80%, and no block is needed.

## 3. Plan (new test files only, no product code change)

Conventions: assertions use `t.Errorf` with a "Wrong match" message and avoid `t.Fatal` except for setup, structs are built directly, `gomock.Any()` is not used for parsed payload arguments (`docs/conventions/testing.md` 13.3, 13.5, 13.8); names `Test_<Func>` or `Test_<Func>_<scenario>`, table style, `mc := gomock.NewController(t)` with `defer mc.Finish()`, no testify,
fixed UUID literals, synthetic fixtures only. Expected values are literals, never derived from product code. Mock calls use exact arguments
where the value is known. No time, network or goroutine dependence: Start paths that reach `kickoff` (async goroutine) are not exercised.

1. pkg/listenhandler/v1_analyses_edge_test.go (about 30 statements of the 32 uncovered; the `marshalAnalysis` marshal failure is not targeted; the unparsable query string (205-206) is reached with a URI such as `?customer_id=%zz`): 400 for malformed POST body and nil customer or activeflow id;
   429 for the concurrency-limit sentinel and 500 for an unclassified error through `analysisErrorResponse`; GET list with missing customer id (400),
   malformed filter body (400), a body `customer_id` that must not override the query authority, id-typed filter normalization to uuid, page_size
   parsing (valid, empty, non-digit), and handler error mapping; GET and DELETE by id with bad id (400), missing customer id (400), handler error mapping;
   `analysisIDFromURI` and `normalizeFilterValue` boundary cases.
2. pkg/analysisdbhandler/analysis_edge_test.go (about 19 statements of the 22 uncovered in analysis.go; the three squirrel ToSql returns are unreachable):
   create with a result payload (`analysis.go:37-38`) and a non-duplicate exec error (62), built as a `*mysql.MySQLError` with a number other than 1062 so that dropping only the number comparison is caught; get by id not found (90) and scan error (95-96);
   get-by-activeflow not found (139), scan error (142-145) and success (147); list success (194), scan error (186),
   and the `ApplyFields` error (168), reached with a filter value of an unsupported kind such as a slice, which fails before any query so no sqlmock
   expectation is set; exec and rows-affected errors on reset (213, 218), update-result (250, 255) and delete (280, 284) (sqlmock `WillReturnError`,
   `NewErrorResult`); scan error on count (306).
3. pkg/analysishandler/start_edge_test.go (about 12 statements of the 15 uncovered; the three `kickoff` call statements (55-56, 77) are skipped): activeflow lookup error
   masked as not-found (33), unexpected error from the existing-row lookup (82-83), progressing count error (63), `createProgressing` success returned
   directly (152) with a fixed uuid, duplicate then re-read error (157), unexpected create error (162), reset error (111), re-read errors after a won
   or lost reset (117, 124), unknown status treated as in-flight (130). The reanalyze policy tests set `TMUpdate` to a fixed time literal and fix `TimeNow` through the util mock; the reset and re-read error cases (111, 117, 124) leave `TMUpdate` nil so no clock is read.
4. About 37 statements (the listed targets) in pkg/analysishandler/collect_context_edge_test.go, all through mocks of the request handler (no network, no goroutines):
   `chaseRecord` transcribe lookup error (169), recording lookup error (181), recording customer mismatch (185-187), unsupported reference type (191);
   `chaseOrigin` depth limit (125, called directly with a depth above the limit); `enrichCall` fallback get failure (208-212) and fallback get success after an empty list (213); `enrichConversation` get
   failure (265), customer mismatch (269) and message list failure (289-294); `enrichAIcall` get failure (366) and customer mismatch (370);
   `scopedFlowName` flow lookup error or nil result (394-395); `markerFor` default (482); helpers with boundary inputs: `parseCanonicalTS` invalid (624), `clamp0MS` negative
   (635), `callDurationSec` negative (646), `threadSpanSec` nil and negative (653, 657), `tmBefore` nil first and nil second (665, 667), `rfc3339OrEmpty`
   nil (682), `detailOrNil` empty (689), `boolStr` true (696). The unresolvable-origin return in `chaseOrigin` and the nil check in `enrich` are not
   reachable (`enrichRef` never returns a nil context) and are not targeted. Line numbers come from the measured profile and are re-checked when written.
   If this falls short, pkg/siphandler/main.go error branches are the reserve (`GCSReader` is an interface and temp files go to a `t.TempDir()` through
   `TMPDIR`, so no network or fixed paths are touched). `Start` error return of `createProgressing` (72-73) is added to item 3.

Items 1 to 3 sum to about 61 statements (80.5% needs 57, or 58 when the flaky statement is uncovered) and item 4 adds about 37, total about 98
statements, final about 81.7%; the last-message failed status (337) is a further spare.

Excluded and stated in the PR text: cmd/timeline-manager main (connects RabbitMQ, MySQL, ClickHouse), the ClickHouse paths of pkg/dbhandler
(needs a fake of the driver connection), and error returns that no input can reach.

## 4. Verify

- Measure with the command above and the hand-written exclusion; require at least 80.5%.
- `go test -count=3 ./...` and `go test -race` on the touched packages pass; `golangci-lint run` passes; `scripts/check-test-conventions.sh` passes.
- Mutation sample, one per new file, each in a scratch copy and each on a branch that only the new file covers (existing tests must still pass on the
  mutant, and the failing test must be one of the new tests, checked by name): skip of a body `customer_id` in the list filter merge
  (v1_analyses.go:109), the masked not-found for a failed activeflow lookup (start.go:33), the comparison of the MySQL error number with 1062 removed (analysis.go:59, keeping `errors.As` so the code compiles), the customer mismatch check in `chaseRecord` (collect_context.go:184).
- No test pins the wrap-around of an oversized `page_size`; a neutral comment is used where a test fixes behavior that looks unintended.
- `git diff --stat origin/main...HEAD` lists only files under bin-timeline-manager/.
