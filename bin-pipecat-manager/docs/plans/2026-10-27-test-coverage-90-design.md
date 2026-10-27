# bin-pipecat-manager test coverage to 90% (VOIP-1663)

## Goal

Raise the hand-written statement coverage of bin-pipecat-manager from 87.77% to at least 90.5%
(generated code, mocks and `DO NOT EDIT` files excluded). Only test files are added. No
production code, signature or visibility changes.

## Measurement

`GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in the service directory. The hand
written total drops `gens/`, `mock_*.go` and `DO NOT EDIT` files and merges duplicate blocks with
the maximum count. Baseline: 2412 statements, 2117 covered. 90.5% needs about 66 more.

## Approach

Target the reachable branches with deterministic tests that use gomock, miniredis, sqlmock,
`httptest` loopback servers and the existing mock websocket handler. No test touches a real
network, database or clock beyond bounded waits.

| Area | Test idea |
|---|---|
| pipecatcallhandler/start | Asterisk websocket connected path of both the call and the ai call (call reference) start: a mocked `DialContext` returns a loopback connection, `ReadMessage` returns the ready text message and later a normal close. All lookups of the ai call return the same success result on every call and the external media stop succeeds, so the terminate goroutine reaches the python runner stop. The test waits for the python runner start call of the runner goroutine and for the python runner stop call of the terminate goroutine, the last mock calls of both, before the controller finishes. |
| pipecatcallhandler/session, models/pipecatcall | `SessionCreate` with a connection, `SessionStop` while the connection is not ready yet, close failure of the connection, `SetConnAst`. |
| pipecatcallhandler/run | media handle with a done channel, an audio send failure, and a context cancelled by the read mock before it returns a text message (so the loop top check is hit deterministically). |
| pipecatcallhandler/runner | team resolve failure, python start failure, a tool of the optional tool list that duplicates a built-in tool name, resample failure, context cancelled while waiting for the Asterisk connection, websocket write failure, and a context cancelled flush that keeps the buffered tokens in the final text. |
| pipecatcallhandler/audiosocket | a one byte input at a valid rate and at a negative rate, which make the resampler library report an error. |
| pipecatcallhandler/misc | constructors with field wiring assertions (including the frame handler), pythonrunner request build errors (a marshal failure through a non serializable llm message value, a request creation failure through a nil context held in a variable), an unlisted error category that is not notified unless it is fatal, not found mapping, and a message id that is a uuid (recorded) or not (previous value kept). |
| listenhandler | response marshal failure through a non serializable value in the llm messages of a pipecatcall response, typed error response with non serializable details, consume error. |
| dbhandler, cachehandler, logtest, config | marshal failure through a non serializable llm message value, the marker for log fields that cannot be encoded, skipped local model entries (reported once at the error level) (the test restores the global model registry in `t.Cleanup`). |
| cmd | `main` with the help argument for both commands. |

Writes to external resources are not needed. Tests that already exist are not changed.

## Verify

- every new test passes with `-race -shuffle=on -count=3` (`pkg/dbhandler` is run with `-count=1` because its existing tests share one in-memory database)
- hand written total is at least 90.5% and the per package table is measured with the same flags
- sampled mutations of the production code (guard removal, wrong error mapping, removing the connection
  hand-over or the terminate call in the start path) make the matching new test fail
- the five step verification of the repository (`go mod tidy`, `go mod vendor`, `go generate`, `go test`,
  `golangci-lint run`) passes
- the change set is inside bin-pipecat-manager and has no non test `.go` file

## Risks

- the start path is the largest single contribution; without it the target is not reached, so it is built first.
- `main` tests restore `os.Args`.
- every wait for a goroutine signal is a select with a bounded timeout, so a failure shows as a test failure and not as a hang. The consume error test waits for its goroutine the same way.
- the start path spawns goroutines that outlive the call. Each test waits on an explicit signal
  (the python runner start call, the python runner stop call and the last log line of the runner and terminate goroutines, so no goroutine writes to the shared logger after the test) before it returns, so no mock is called after the test
  finished.
- `go.mod` and `go.sum` stay unchanged.
