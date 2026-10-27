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

- Every new test is deterministic: no real network except closed loopback ports and in-process fakes. Goroutine based expectations (consume errors) are synchronized through channels inside the mock `DoAndReturn`, never `sleep`; this applies to the success paths of `runListen` (two queues) and `runSubscribe` as well, so no mock call happens after the test ends.
- Global state is restored in cleanup: `*config.Get()`, viper, `signal.Stop(chSigs)` with a drain of `chDone` (and the goroutine started by `initSignal` is ended by sending a signal value to `chSigs`), and `PrometheusEndpoint` stays empty so no duplicate `http.Handle` happens. dbhandler tests that insert malformed rows use a private in-memory database.
- Assertions check specific returned errors, error text and mock call arguments; no empty calls and no self-comparison.
- Mutation sample: for at least 5 new tests, break the guarded product line in a scratch copy and confirm the test fails.
- `GOFLAGS=-mod=mod go test -count=1 -race ./...`, `golangci-lint run`, hand-written total 85.5% or more, diff only under bin-transcribe-manager.
- VOIP-1416 (packages below 80%) overlaps only on the two cmd packages (61.1% and 12.2%); this is stated in the PR body and not absorbed.

## 4. Test plan (design)

All new tests are additions to `_test.go` files, one new file per area, named after the code under test.

- `cmd/transcribe-manager/main_run_test.go`: table tests of `runListen`, `runSubscribe`, `runStreaming` with `sockhandler.MockSockHandler`, `transcribehandler.MockTranscribeHandler`, `transcripthandler.MockTranscriptHandler`, `requesthandler.MockRequestHandler`, `streaminghandler.MockStreamingHandler`. `runListen`: queue create error, volatile queue create error, success with both `ConsumeRPC` calls signalled through a channel. `runSubscribe`: queue create, topic create, queue bind errors and success with `ConsumeMessage` signalled. `runStreaming`: `Run` error is wrapped, success. `runDaemon`: closed loopback DSN returns the wrapped connect error (empty Prometheus endpoint, config restored, `initSignal` goroutine ended in cleanup).
- `pkg/transcribehandler/*_extra_test.go`: Start with a lookup error that is not not-found, recording and live start errors; HealthCheck on confbridge lookup error (reschedule or stop path asserted through the mock calls); event handler where one `Delete` fails and the next is still deleted; Delete where `Stop` succeeds; `getBCP47LanguageCode("zz-invalid")` equals `en-US`.
- `pkg/listenhandler/*_extra_test.go`: `errorResponse` with a `VoipbinError` whose detail cannot be marshaled yields 500; consume error logged without panic (channel sync); each list/get/start/stop/delete handler returns 500 when the mocked result carries a year 10000 time stamp; `Run` error path of the volatile queue is already covered and is not repeated.
- `pkg/dbhandler/*_extra_test.go`: list with empty token uses the current time (mock util handler), update with empty fields is a no-op, update-to-cache read error, unsupported filter type, update value conversion error, transcript cache hit, malformed stored rows on a private in-memory database for the scan error paths.
- Single statements: one focused test each in the owning package.

Test naming follows `Test_<function>` with a `name` field per table row; gomock controller variable is `mc` (CI gate `scripts/check-test-conventions.sh`). The gate script and `golangci-lint` run before push. No product file is edited.
