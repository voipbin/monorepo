# VOIP-1663 bin-campaign-manager test coverage

Goal: hand-written statement coverage (generated files, `mock_*.go`, `DO NOT EDIT` files excluded)
from 88.81% to at least 90.5%, about 60 more statements. Tests only; no product code change.
Measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`; the final PR table
uses the final HEAD and the same flags.

## New tests and expected gain (statements)

- cmd/campaign-manager (about 50): createDBHandler and runService with a malformed fake database
  address (only the error return of runService is covered; both stop at the failed connection),
  runListen and runSubscribe on the mocked sock handler, initProm success path on a loopback
  address with port 0, the signal handler. The run function and what follows a successful
  database connection need live services and are not targets.
- cmd/campaign-control (about 35): initHandler with a malformed fake address, initCache on
  in-memory redis (success and failure), each sub-command up to the failed initialization, the
  entry point with `--help`.
- internal/config (4): Get, Bootstrap, LoadGlobalConfig.
- pkg/subscribehandler (3): the event run wrapper (called directly with an event routed to a
  mocked campaigncall handler, checking it returns before the handler finishes) and the failing
  consume goroutine.
- pkg/dbhandler (about 18): cache write failures, lookup by reference and by activeflow (query,
  no-row, row iteration and scan failures), ongoing list (empty token, query and scan failures),
  and filter failures of the list functions, on sqlmock and the cache mock, in the style of the
  existing failure tables. The actual gain is checked after writing.
- Sum about 110, roughly twice the need.

## Isolation

- No external network: malformed addresses (parse fails before any dial), loopback port 0, miniredis, sqlmock.
- Shared config is a once-loaded pointer: tests overwrite its fields and restore them in
  cleanup. viper, os.Args, stdout and environment keys are restored the same way.
- initProm registers on the default mux, which panics on a duplicate path. Each call in the
  test binary uses a unique path (counter based), so `-count` and `-shuffle` are safe. The
  listener goroutine cannot be stopped and is left until the process ends.
- The init goroutine consumes only one signal per process, so each signal test starts its own
  handler goroutine, drains both channels first, sends on the signal channel, and receives from
  the done channel with a timeout; whichever goroutine consumes it, the done channel is filled
  once. Both channels are drained in cleanup. No OS signal is sent.
- initProm and runService tests also overwrite the listen address with loopback port 0.
- runListen always returns nil, so its tests assert the mocked calls; runSubscribe returns the
  error, which is asserted. Goroutine work is synchronized on channels with a timeout.

## Verify

1. Every package except dbhandler: `go test -race -shuffle=on -count=3` passes. The dbhandler tests
   share one in-memory database, so only `-race -count=1` runs there (shuffled order and repeated
   runs in one process are not supported by the existing tests); new rows there use fresh ids.
2. Hand-written total at least 90.5% on the final HEAD.
3. Mutation sampling: at least 8 guarded statements across the packages are inverted in a scratch
   copy and each is caught by a new test.
4. `git status` shows no vendor or unrelated file change; go.mod/go.sum are unchanged and
   `go mod tidy` leaves no diff.
5. `git diff --stat origin/main...HEAD` lists only bin-campaign-manager/; lint and the test
   convention gate pass.
