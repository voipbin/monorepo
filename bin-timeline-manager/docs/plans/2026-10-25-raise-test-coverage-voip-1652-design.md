# VOIP-1652 bin-timeline-manager unit test coverage to 75%

## 1. Ticket review

Verdict: valid. Measured on the branch (origin/main accc03e00 content for this service, `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, all tests pass), excluding gens/, mock_*.go and `Code generated ... DO NOT EDIT` files, duplicate blocks merged by max count: hand-written 2349/3142 = 74.76% (the raw profile also counts generated code). 75.0% needs 8 more statements, 76% needs 39 more.
VOIP-1649 (merged, PR 1533) raised the service from 62.99% to 74.76% with test files only; no open or merged PR covers the remaining gap (`gh pr list --state all --search`). Only test files are added; no product code or signature changes. cmd/timeline-manager main (197 uncovered) binds RabbitMQ, MySQL, ClickHouse and signals and stays out of scope. pkg/dbhandler (176 uncovered) runs ClickHouse queries through the driver connection, rows and batch interfaces; the existing tests only cover the connection-not-established branch, and a fake of those interfaces costs more than it returns, so it is excluded.

## 2. Baseline (hand-written, before)

| package | covered/total | % |
|---|---|---|
| cmd/timeline-manager | 2/199 | 1.0 |
| pkg/dbhandler | 117/293 | 39.9 |
| cmd/timeline-control | 73/142 | 51.4 |
| pkg/analysishandler | 657/845 | 77.8 |
| pkg/siphandler | 399/473 | 84.4 |
| pkg/analysisdbhandler | 137/160 | 85.6 |
| pkg/listenhandler | 316/365 | 86.6 |
| models/analysis | 8/9 | 88.9 |
| pkg/subscribehandler | 159/166 | 95.8 |
| internal/config | 82/85 | 96.5 |
| pkg/homerhandler | 180/186 | 96.8 |
| models/sipmessage, models/verdict, pkg/peereventhandler, pkg/eventhandler | 100% | 100 |

## 3. Plan (new test files only)

Target pkg/analysishandler, the largest hand-written gap that runs on mocks only: run.go (kickoff, persistCompleted, persistFailed: 63 statements, 0%) and chain.go (runChain 0%, useSingleCall 0%, callGateway, runStaged, buildFinalVerdict, errorSignalLines, addContextBlock edge branches: about 62 statements). ClickHouse dbhandler and the cmd mains are skipped (driver fakes cost more than they return).

Conventions: `Test_<Func>_<scenario>`, table style where it fits, `mc := gomock.NewController(t)`, no testify, fixed UUID literals, synthetic data only, expected values written as literals (not derived from product constants), exact mock arguments where practical. Reuse `newStartTestHandler`; the test handler must also set `metricEnrichment` (enrich dereferences it).

New files: `pkg/analysishandler/run_test.go`, `pkg/analysishandler/chain_run_test.go`.

Verify items:
1. persistCompleted / persistFailed: db success (n=1), row-gone (n=0), db error each return without panic; the db mock receives the exact status (completed or failed), verdict bytes, model and sanitized message.
2. kickoff: chain success persists StatusCompleted and counts one completed metric; chain error persists StatusFailed with the sanitized message only (no raw provider error text); a panic inside the chain (db mock panics in the collect step) is recovered and persisted as failed; the semaphore slot is released after each run (channel length 0). Tests synchronize on a channel signalled from the db mock, never on sleeps.
3. runChain: single-call path (few events, short transcript) returns a verdict JSON whose resources_used comes from the Go inventory, not the LLM; staged path (>= 50 events); collect error wrapped; raw validation failure (invalid overall_status or out-of-range evidence) fails the chain; gateway error propagates.
4. useSingleCall boundaries: 49 vs 50 events, transcripts of 3999 vs 4000 runes (multi-byte runes counted as runes).
5. callGateway: nil response, Truncated, finish_reason length, empty result, rpc error.
6. runStaged: stage1/stage2/stage3 error each stop the chain with the stage in the error; unparsable stage2 content.
7. buildFinalVerdict with issues resolves evidence tuples from the frozen list; addContextBlock includes only the resolved enrichment blocks and omits `context` when empty.
8. Mutation sample after the tests pass: flip the persisted failed status, drop the truncated guard, change the 50-event threshold comparison, remove the recover; each must fail at least one new test.

Goal: hand-written total at least 76% (the planned targets hold about 125 uncovered statements against the 39 needed). Gates: tests, golangci-lint, `scripts/check-test-conventions.sh`, `git fetch origin main` then `git diff --stat origin/main...HEAD` limited to bin-timeline-manager/.
