# VOIP-1652 bin-timeline-manager unit test coverage to 75%

## 1. Ticket review

Verdict: valid. Measured on the branch (branch rebased on origin/main accc03e00, which contains the VOIP-1650 homerhandler change, `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, all tests pass), excluding gens/, mock_*.go and `Code generated ... DO NOT EDIT` files, duplicate blocks merged by max count: hand-written 2350/3143 = 74.77% (the raw profile also counts generated code; the acceptance criterion is this hand-written total, measured with -mod=mod, and CI may differ by a fraction of a point). 75.0% needs 8 more statements, 76% needs 39 more.
VOIP-1649 (merged, PR 1533) raised the service from 62.99% to about 74.8% with test files only; no open or merged PR covers the remaining gap (`gh pr list --state all --search`). Only test files are added; no product code or signature changes. cmd/timeline-manager main (197 uncovered) binds RabbitMQ, MySQL, ClickHouse and signals and stays out of scope. pkg/dbhandler (176 uncovered) runs ClickHouse queries through the driver connection, rows and batch interfaces; the existing tests only cover the connection-not-established branch, and a fake of those interfaces costs more than it returns, so it is excluded. cmd/timeline-control (69 uncovered) is a CLI entry point over live ClickHouse and MySQL connections, also excluded. pkg/siphandler/gcsreader_impl.go (36) wraps the cloud storage client and is excluded. pkg/analysishandler/collect_context.go (41) and pkg/listenhandler/v1_analyses.go (32) are mock-testable but smaller; they are the reserve if the planned targets fall short.

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
| pkg/homerhandler | 181/187 | 96.8 |
| models/sipmessage, models/verdict, pkg/peereventhandler, pkg/eventhandler | 100% | 100 |

## 3. Plan (new test files only)

Target pkg/analysishandler, the largest hand-written gap that runs on mocks only: run.go (kickoff, persistCompleted, persistFailed: 63 statements, 0%) and chain.go (runChain 0%, useSingleCall 0%, callGateway, runStaged, buildFinalVerdict, errorSignalLines, addContextBlock edge branches: about 62 statements). ClickHouse dbhandler and the cmd mains are skipped (driver fakes cost more than they return).

Conventions: `Test_<Func>_<scenario>`, table style where it fits, `mc := gomock.NewController(t)`, no testify, fixed UUID literals, synthetic data only, expected values written as literals (not derived from product constants), exact mock arguments where practical. Reuse `newStartTestHandler`; the new test file wraps it in a helper that also sets `metricEnrichment` (enrich dereferences it).

New files: `pkg/analysishandler/run_test.go`, `pkg/analysishandler/chain_run_test.go`.

Handler helper (in the new test files): wraps `newStartTestHandler`, exposes the `MockEventHandler` it already builds (type assertion on `h.eventHandler`), and replaces `metricStarted`, `metricCompleted`, `metricDuration` and `metricEnrichment` with fresh collectors registered in a local registry; values are read through `registry.Gather()` getters (no new module import, go.mod unchanged).

Verify items:
1. persistCompleted / persistFailed: db success (n=1), row-gone (n=0), db error each return without panic; the db mock receives the exact status, verdict bytes, model and message (completed with model; failed with nil bytes, empty model and the given message).
2. kickoff (all through the real goroutine): started metric is 1 right after the call; chain success persists StatusCompleted with expected verdict JSON written as a literal (never rebuilt by calling runChain again; for ReferenceTypeNone the session_context carries reference_type `none` and an empty channel) and the stage3 response model, completed(completed) metric 1; chain error (eventHandler.AggregatedList returns an error) persists StatusFailed with the literal message `analysis failed to produce a result` (no raw error text), completed(failed) metric 1; panic (AggregatedList DoAndReturn panics) is recovered and persists StatusFailed with the literal `internal error during analysis`, with no completed metric. Synchronization is channel based: the mock callbacks never call t.Fatalf (they record errors in a variable written before the signal send and the test goroutine fails after receiving); the db signal channel is buffered (1) so the goroutine never blocks after a test failure; after the db signal the test waits for the semaphore release: BEFORE calling kickoff the helper sets `h.sem` to a capacity-1 channel (never replaced afterwards, which would race with the goroutine's deferred release), and after the signal the test sends into that same channel with a 2s timeout (the send succeeds only after the goroutine's deferred release, which also orders the metric writes before the metric reads); every wait has a 2s timeout, fails from the test goroutine, and the timeout message names the signal being awaited; no sleeps, no parallel subtests, mocks are fully consumed before mc.Finish. Success cases use a non-nil activeflow of ReferenceTypeNone with a nil FlowID (enrich then needs no RPC) and a non-nil activeflow id with an empty TranscribeV1TranscribeList reply.
3. runChain: single-call path returns verdict JSON whose resources_used comes from the Go inventory even when the LLM response carries a different one, plus the stage3 model; staged path (>= 50 events, each event given a distinct timestamp and data so the dedup key keeps all 50; if transcripts are needed the transcribe list and transcript list mocks are set with a non-nil activeflow id); collect error is wrapped as `could not collect input`; raw validation failures (invalid overall_status; evidence index out of range) are wrapped as `raw verdict validation failed`; runCombined error is wrapped as `runCombined: gateway call failed` and the request used the stage3 model.
4. useSingleCall boundaries (built from a hand-made collectedInput, no collection involved): 49 vs 50 events; transcripts of 3999 vs 4000 runes, including a split case (2000+2000) and a multi-byte case (3999 Korean characters, whose byte length exceeds 4000, still single call).
5. callGateway: nil response, Truncated, finish_reason length, empty result, rpc error; each returns a distinct message asserted by substring; Truncated=true with finish_reason stop and Truncated=false with finish_reason length are separate cases; the empty result case runs with both nil and an empty RawMessage.
6. runStaged: stage1, stage2 and stage3 errors stop the chain with the stage in the message and no later call (expectations limited by Times); stage2 content that is valid JSON but not an object (`[]`); stage responses that feed asserted request data are valid JSON (the data builders ignore marshal errors); each stage request carries its own model (m1, m2, m3) and schema name.
7. buildFinalVerdict with issues resolves evidence tuples from the frozen list; with an enrichment it attaches SessionContext, Outcome and Metrics, with nil enrichment they are omitted. addContextBlock branches (nil enrichment, an empty `&sessionEnrichment{}` which is what enrich returns when nothing resolved, ctx only, outcome only, metrics only, both nil forms give no `context` key) are asserted after json.Unmarshal of the payload; errorSignalLines is exercised through buildCombinedData and buildStage3Data with non-empty error signals.
8. Mutation sample after the tests pass: swap the persisted failed status, drop the truncated guard, change the 50-event threshold comparison, remove the recover (expected to show as a test binary crash, recorded as such), swap a stage model; each must fail at least one new test. Raw validation cases: overall_status ok with evidence_index [99] for out of range (a non-ok status without evidence fails earlier for a different reason).
9. Stability gate: new tests pass with `-race -count=20`.

Reserve if 76% is not reached: Start's kickoff call lines through the same helper, and collect_context.go pure helpers (chaseRecord, threadSpanSec, tmBefore, enrichAIcall branches).

Goal: hand-written total at least 76% (the planned targets hold about 125 uncovered statements, of which about 120 are reachable, against the 39 needed). Gates: tests, golangci-lint, `scripts/check-test-conventions.sh`, `git fetch origin main` then `git diff --stat origin/main...HEAD` limited to bin-timeline-manager/.
