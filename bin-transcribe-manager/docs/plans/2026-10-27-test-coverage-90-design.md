# bin-transcribe-manager test coverage to 90% (VOIP-1663)

## Goal and scope

Raise the hand-written statement coverage of `bin-transcribe-manager` from 88.00% (2214/2516) to at least 90.0%, (51 more statements). 90.5% needs 63 more, and the reachable set below holds 66, so the firm target is 90.0% and 90.5% is a best effort. Generated code (`mock_*.go`, files with a `DO NOT EDIT` header, `gens/`) is excluded. Only `_test.go` files are added; no product code changes and no new modules.

## Baseline (origin/main 0bfbe2a1d, `GOFLAGS=-mod=mod go test -count=1 -coverprofile`)

| package | before (hand-written) |
|---|---|
| cmd/transcribe-control | 61.4% |
| cmd/transcribe-manager | 44.4% |
| internal/config | 96.1% |
| pkg/dbhandler | 94.4% |
| pkg/streaminghandler | 89.0% |
| pkg/listenhandler, pkg/transcribehandler, pkg/transcripthandler | 99.3% to 99.8% |

## Approach

Everything below is reachable without a signature change and without leaving the machine.

1. `pkg/streaminghandler` (AWS): a local HTTP/2 TLS `httptest` server answers `StartStreamTranscription` with a hand-encoded event stream; a `transcribestreaming.Client` pointed at it is injected into `streamingHandler.awsClient`. The handler flushes the response headers first and then keeps reading the request body (the call is bidirectional); frames are encoded by hand, so `go.mod` stays unchanged. These tests live in the external test package with a small `export_test.go` entry point, so the in-package tests keep avoiding a direct `net/http` import (the `bodyclose` linter reacts to it). The media side is a loopback websocket server. The service side only asserts the received audio bytes. Scenarios: normal run with transcript events and audio chunks reaching the server; response stream closed first, then a late binary frame (not delivered) or a late text frame (reader stops). `awsRun` returns on cancel, so tests wait on its return and then poll for the media reader to end with a deadline. The AWS client construction failure and the `NewStreamingHandler` fallback are driven with an invalid CA bundle path (`t.Setenv`; AWS profile, shared config and credentials files, EC2 metadata and Google application credentials isolated, and the test only asserts that no AWS client was created).
2. `pkg/dbhandler`: a small `database/sql/driver` fake (opened with `sql.OpenDB`, nothing registered globally) whose rows fail during iteration covers the row and rows error branches; a `NotEq` filter holding a failing `driver.Valuer` covers the list query build errors.
3. `cmd/transcribe-manager`, `cmd/transcribe-control`: a loopback fake MySQL server (handshake, OK for ping, ERR for everything else, several connections, no password) drives `runDaemon` to its migration failure; `initHandler` reaches its cache failure with the same fake plus a closed loopback port for the cache address. The fake handles `COM_QUIT`, and listener and connection goroutines are closed and waited for in `t.Cleanup`. `main()` runs with `help` (control) and `__complete ""` (manager, which has no subcommands). Global state (`os.Args`, viper, config, signal notification via `signal.Stop`) is restored, and the Prometheus endpoint is left empty so the default mux is not registered twice.

## Verify

- Each new test asserts an observable outcome (error text, call arguments, bytes received by the fake server, returned values), not just that a function ran.
- `go test -race -count=3` on the new tests is stable (the existing dbhandler tests share one sqlite state and already fail above `-count=1` on main, which is out of scope); waits are polls with a deadline rather than fixed sleeps, ports are `127.0.0.1:0`.
- Final hand-written total is at least 90.0% with the same measurement flags used for the baseline; the PR body lists before and after per package.
- Mutation spot checks: break a branch or return value in the product code on a scratch copy and confirm a new test fails.
- `git diff --stat origin/main...HEAD` shows only files under `bin-transcribe-manager/`; `go mod tidy` leaves no diff.
- The numbers above are as of the measurement base; the final table is re-measured on the final HEAD with the same flags. The full verification workflow (`go mod tidy`, `go generate`, `go test`, `golangci-lint`) passes.
- Test values are synthetic; no credentials, no real hosts.
