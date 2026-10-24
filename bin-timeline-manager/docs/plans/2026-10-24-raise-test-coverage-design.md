# VOIP-1649 bin-timeline-manager unit test coverage to 65%

## 1. Ticket review

Verdict: valid. Measured on the branch (base 2fce50a00, which differs from origin/main bffdd69e3 only by a comment line and README, so statement counts are equal) with `GOFLAGS=-mod=mod go test -count=1 -coverprofile`, excluding gens/, mock_*.go and
`Code generated ... DO NOT EDIT` files, duplicates merged by max count: hand-written 1979/3142 = 62.99% (raw profile includes generated code).
65% needs 64 more statements, 66% needs 95. No existing PR or commit covers this (`gh pr list --search VOIP-1649` is empty). Only test files are added.

## 2. Baseline (hand-written, before)

| package | covered/total | % |
|---|---|---|
| cmd/timeline-control | 73/142 | 51.4 |
| cmd/timeline-manager | 2/199 | 1.0 |
| internal/config | 82/85 | 96.5 |
| models/analysis | 0/9 | 0.0 |
| models/sipmessage | 67/67 | 100 |
| models/verdict | 0/20 | 0.0 |
| pkg/analysisdbhandler | 137/160 | 85.6 |
| pkg/analysishandler | 495/845 | 58.6 |
| pkg/dbhandler | 117/293 | 39.9 |
| pkg/eventhandler | 109/109 | 100 |
| pkg/homerhandler | 0/186 | 0.0 |
| pkg/listenhandler | 316/365 | 86.6 |
| pkg/peereventhandler | 23/23 | 100 |
| pkg/siphandler | 399/473 | 84.4 |
| pkg/subscribehandler | 159/166 | 95.8 |

## 3. Plan (new test files only, no product code change)

cmd/timeline-manager main (197 stmts) and pkg/dbhandler ClickHouse paths need live connections and are skipped.

1. pkg/homerhandler/main_test.go: httptest server. Per method (GetSIPMessages, GetPcap, GetRTCPPcap): missing address or token, empty call id,
   non-2xx status (500, asserted separately for each of the three methods), transport error (closed server), success. Assert request path, method, Auth-Token header, and decoded payload
   (time buffer, limit, search key per hepid). GetSIPMessages additionally: invalid JSON, ascending sort (inputs with the same fractional digits), cap at 50 messages (51 inputs, first 50 kept).
2. pkg/analysishandler/reduce_test.go: reduceEvents (under target unchanged, over target keeps error events and drops non-error, order kept),
   summarizeData (empty, non-object, key extraction, truncation at 240 runes), resourceIDOf, truncateRunes boundary,
   collectTranscripts (nil activeflow, list error, per-transcribe error skipped, empty message skipped, cap per resource), joinTranscript.
3. pkg/analysishandler/collect_test.go: collectEvents with the eventhandler mock (error, nil/empty page, pagination, de-duplication,
   analysisMaxEvents cap), collectInput (error wrap, freeze indexes, error signals, inputReduced), dedupKey, buildInventory unknown
   publisher, isErrorEvent table.
4. pkg/analysishandler/get_test.go: Get, GetByActiveflowID, Delete map ErrNotFound to the masked ErrNotFound and wrap other errors; List passes
   arguments through and wraps errors.
5. models/verdict/verdict_test.go: ValidateRaw table (bad JSON, invalid status, invalid severity, non-ok issue without evidence, index below 0
   and at eventCount, ok status with no evidence allowed, success). models/analysis/webhook_test.go: Model dropped from the projection,
   CreateWebhookEvent output keys.

## 4. Verify

- Hand-written total >= 66% with the same measurement; per-package before/after table in the PR.
- `go test ./...`, `golangci-lint run`, `bash scripts/check-test-conventions.sh` pass.
- Mutation sample (scratch copy only): flip `>= 300` in homer status check, drop the 50 message cap, change evidence upper bound `>=` to `>`,
  drop the ErrNotFound mapping in Get; each must fail at least one new test.
- Error cases assert the message and that no request reached the server for validation failures. Get/Delete assert errors.Is(masked ErrNotFound) true and errors.Is(dbhandler ErrNotFound) false.
- No assertion compares a value produced by the same code path; expectations are literals.
