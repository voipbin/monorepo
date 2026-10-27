# bin-transcribe-manager test coverage 85% design

## 1. Ticket review (VOIP-1659)

Baseline measured on origin/main a47f5a2a1 (the 80% work, VOIP-1652 and VOIP-1657, is already merged): hand-written coverage 83.47% (2100 of 2516 statements). Generated code (`gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` files) is excluded and duplicate blocks keep the maximum count. The goal is 85.0% or more, 85.5% or more recommended, which needs 39 (2139/2516 = 85.02%) and 52 (2152/2516 = 85.53%) more covered statements. Profile blocks repeat the statement count of a split function, so gains below are the sum of profile blocks.

| Package | Before | Statements |
|---|---|---|
| cmd/transcribe-control | 61.1% | 321 |
| cmd/transcribe-manager | 12.2% | 180 |
| internal/config | 96.1% | 77 |
| models/streaming, transcribe, transcript | 88.9%, 91.7%, 90.0% | 9, 24, 10 |
| pkg/cachehandler | 98.6% | 71 |
| pkg/dbhandler | 88.9% | 359 |
| pkg/listenhandler | 94.0% | 281 |
| pkg/streaminghandler | 88.6% | 463 |
| pkg/subscribehandler | 99.0% | 97 |
| pkg/transcribehandler | 97.2% | 461 |
| pkg/transcripthandler | 99.4% | 163 |

The 85% target is reachable without product code changes (see section 2).

## 2. Reachable candidates (test only, no signature change)

| Area | Targets (file:line) | Statements |
|---|---|---|
| cmd/transcribe-manager | `runListen` (183-203), `runSubscribe` (207-223), `runStreaming` (227-237) with mock sockhandler and mock handlers (success and each error return); `runDaemon` (109-119) up to the database connect failure on a closed loopback address | about 58 |
| pkg/transcribehandler | start.go:90 (lookup error other than not found), 97 and 99 (recording start error), 124 (live start error without duplicate), health.go:68 (confbridge lookup error), event.go:41 (one delete fails), main.go:139 (BCP47 `und` fallback), transcribe.go:165 (stop succeeds before delete) | 12 |
| pkg/listenhandler | main.go:110 (`ToResponse` marshal failure through an unmarshalable error detail), main.go:156 and 177 (consume error logging) | 4 |
| pkg/dbhandler | empty list token (transcribe.go:204, transcript.go:156), empty update fields (transcribe.go:299, transcript.go:237), nil StreamingIDs (transcribe.go:30), update-to-cache read error (transcribe.go:84, transcript.go:70), unsupported filter type (transcribe.go:218 and 268, transcript.go:170), update value conversion error (transcribe.go:306, transcript.go:242), transcript cache hit (transcript.go:96 and 103), row and scan errors with malformed stored rows on a private in-memory database | about 25 |
| pkg/listenhandler, models webhook | response marshal failure through a time field with a year above 9999 (v1_transcribes.go:47, 99, 134, 168, 203, v1_transcripts.go:56) and `CreateWebhookEvent` marshal error (streaming, transcribe, transcript webhook.go) | about 14 |
| Single statements | subscribehandler/main.go:139, streaminghandler/streaming.go:65, gcp.go:18, models/transcribe/transcribe.go:120, cachehandler/handler.go:33, transcribe-control main.go:110 | 6 |

The sum is 58 + 12 + 4 + 25 + 14 + 6 = 119 against the 52 needed. A conservative sum without `runDaemon` (19) and without the single statements is 39 + 12 + 4 + 25 + 14 = 94.

Not tested because unreachable without product code changes: squirrel `ToSql` errors, `PrepareFields(t)` and delete-path `PrepareFields` errors, config binding errors, `processRequest` typed-error and not-found branches (the handlers convert them to responses first), the `in|out` direction `default` branch (regex only captures those two), start.go:109 (an earlier reference check filters it), `run()` and `main` of transcribe-manager (the broker connect retries forever), transcribe-control `initTranscribeHandler`, the `run*` bodies after `initHandler` and the success path of `initHandler` (they need a live database, cache and broker), and the AWS stream code (`awsRun` after init, `awsProcessMedia`), which needs a live event-stream endpoint and is not attempted.

## 3. Verify

- Every new test is deterministic: no real network except closed `127.0.0.1` ports and in-process fakes; DSNs use a fake account name and no password. Test values are synthetic.
- Goroutine based statements (consume error logs in `listenhandler/main.go:156`, `:177` and `subscribehandler/main.go:139`) run after the mock returns, so tests wait on a logrus hook that matches the log message, not on a channel inside `DoAndReturn`. The hook helper is defined in the test file of each package (the cmd helper `hookFunc` is package main only). Mock calls made by those goroutines (`ConsumeRPC` twice for `runListen`, including the case where the volatile queue creation fails after the first goroutine started) are all expected and awaited before the test ends.
- Global state restored in cleanup: `*config.Get()` (`saveConfig`), viper, logrus hooks and output (`ReplaceHooks`, `SetOutput`), `signal.Stop(chSigs)` with a drain of `chDone`, and `PrometheusEndpoint` stays empty so no duplicate `http.Handle` happens. The `runDaemon` test follows the cleanup of the existing `Test_initSignal` and compares `runtime.NumGoroutine` before and after after ending the `initSignal` goroutine with a value sent to `chSigs`. dbhandler tests that insert malformed rows use a private in-memory database. Existing helpers (`saveConfig`, `closedAddr`, `hookFunc`, `promEndpointSeq`) are reused, never redeclared.
- Assertions check specific returned errors, error text and mock call arguments; no empty calls and no self-comparison.
- Mutation sample, at least one per area, 5 or more in total: cmd (`runListen` error wrapping), dbhandler (empty token default), listenhandler (marshal failure to 500), transcribehandler (non-not-found lookup error), plus one more. Each is done in a scratch copy and the test must fail.
- Commands: `GOFLAGS=-mod=mod go test -count=1 -race ./...`; `go test -count=3 -shuffle=on ./...` for the new packages; hand-written total measured with the same profile merge as section 1 (blocks of generated files excluded, duplicates by max count) must be 85.5% or more; `golangci-lint run -v --timeout 5m`; `bash scripts/check-test-conventions.sh` (added lines only: `Test_` prefix, no testify, gomock controller named `mc`; needs `git fetch origin main`) and the structure conventions gate that CI runs; diff only under bin-transcribe-manager.
- VOIP-1416 (packages below 80%) overlaps only on the two cmd packages (61.1% and 12.2%); this is stated in the PR body and not absorbed.

## 4. Test plan (design)

Only new `_test.go` files are added (existing test files are not edited), named after the code under test.

- `cmd/transcribe-manager/main_run_test.go`: `runListen`, `runSubscribe`, `runStreaming` with `sockhandler.MockSockHandler`, `transcribehandler.MockTranscribeHandler`, `transcripthandler.MockTranscriptHandler`, `requesthandler.MockRequestHandler`, `streaminghandler.MockStreamingHandler`. `runListen`: queue create error; volatile queue create error (first `ConsumeRPC` still expected and awaited); success (two `ConsumeRPC`). `runSubscribe`: queue create, topic create, queue bind errors; success with `ConsumeMessage` awaited. `runStreaming`: `Run` error wrapped; success. `runDaemon`: closed loopback DSN returns the wrapped connect error.
- `pkg/transcribehandler/start_lookup_errors_test.go`: Start with a lookup error that is not not-found (mock db returns a plain error, error is returned as is), recording start error and live start error (reference lookup fails, error wrapped with the reference id) and the non-duplicate error return.
- `pkg/transcribehandler/health_confbridge_test.go`: `CallV1ConfbridgeGet` returns an error with a transcribe that has retry count below the limit; asserts `Stop` is attempted (db get, then the stop chain of the mock reqHandler as in `Test_HealthCheck_referenceGetError_stopFails_reschedules`) and, when stop fails with the retryable error, one health-check reschedule RPC.
- `pkg/transcribehandler/event_delete_error_test.go`: `EventCUCustomerDeleted` where `Delete` of the first transcribe fails (get error) and the second is deleted; asserts both ids reached the db mock. `pkg/transcribehandler/transcribe_delete_stop_ok_test.go`: Delete of a progressing transcribe where `Stop` succeeds (mock chain of `Test_Delete`), then the delete proceeds. `main_bcp47_test.go`: `getBCP47LanguageCode("zz-invalid")` is `en-US`.
- `pkg/listenhandler/response_marshal_error_test.go`: `errorResponse` with a `VoipbinError` whose `Details` holds a `chan` returns 500; Start, List, Get, Delete, Stop and transcripts List with a mocked result whose `TMCreate` is in year 10000 return 500. `pkg/listenhandler/consume_error_test.go`: both queue goroutines log the consume error (log hook awaited), no panic.
- `pkg/dbhandler/*_test.go`: `transcribe_branches_test.go` and `transcript_branches_test.go`: list with empty token uses the mocked current time, update with empty fields is a no-op, update-to-cache read error, unsupported filter type (a struct value) for list and get-by-reference, update value conversion error (a `chan` value), transcript cache hit, and malformed stored rows (a `BLOB` of 3 bytes in a uuid column) on a private in-memory database for the scan error paths; `row.Err()` branches are included only if they can be produced.
- Single statements: `models/transcribe` unknown status (transition check returns false), `models/*/webhook` `CreateWebhookEvent` with a year 10000 `TMCreate`, `cachehandler` `setSerialize` with a `chan`, `gcpRun` with a nil client returns the not initialized error, `streamingHandler.Delete` of an unknown id keeps the map unchanged, `subscribehandler` consume error log (log hook), transcribe-control `migrate` subcommand with a malformed DSN returns an error without network (viper and config restored).
