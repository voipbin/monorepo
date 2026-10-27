# bin-billing-manager unit test coverage to 90% (VOIP-1663)

## 1. Ticket review and analysis

Verdict: valid. Measured at origin/main 0bfbe2a1d with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in the service directory (no test failures). Hand-written code excludes `gens/`, `mock_*.go` and files whose header has `Code generated ... DO NOT EDIT`; duplicate blocks are merged with the max count.

Result: hand-written 88.09% (3653/4147 statements). 90% needs 3733 covered (at least 80 more); the working goal is 90.5% (3753, at least 100 more) as margin for CI differences. Block weights depend on the Go toolchain, so the Go version used is recorded in the PR body and the measurement is repeated with it.

The previous coverage PRs of this service already covered the handlers, the Paddle paths and the dbhandler transactions. What is left falls into two groups:

- Code that needs a live database, cache or message broker connection (the command wiring and the daemon start-up). It stays untested; changing the command structure only to reach it is out of scope.
- Error and constructor paths that need no connection: the constructors of the handlers, the failure paths of the retry queue, cache writes of the second key, database reads and writes that fail on an unsupported value, response encoding failures, the metrics listener failure, and the command tree executed through the root command.

## 2. Plan

Only `*_test.go` files are added or extended inside bin-billing-manager (plus this document). No product code, signature, visibility or dependency change (`go.mod` and `go.sum` stay unchanged).

Tests, by package:
- cmd/billing-manager and cmd/billing-control: run the root command with the help flag; run the daemon up to the unreachable database and check that the signal goroutine still reacts; execute the command tree with an unreachable database and check that the flag value reaches the configuration source; the cache initialization against a local in-memory server and against a closed address.
- internal/config: the metrics listener failure is logged and the endpoint is registered.
- pkg/cachehandler: a failure of the second write (hook on the redis client) is returned after the first key was written.
- pkg/dbhandler: failures of the retry-queue read (query, scan, iteration) are returned with their cause; update and list calls with a value the layer cannot convert are rejected before a statement is sent; unknown idempotency key gives `ErrNotFound`; constructor wiring.
- pkg/failedeventhandler: store failures during retry do not stop the run and the counters stay correct; an event that cannot be encoded is not stored; constructor wiring.
- pkg/accounthandler, pkg/billinghandler, pkg/listenhandler: constructor wiring; a failed write on account creation is decided by the read-back; a failed finalization of an immediate-end billing is returned; results that the JSON encoder rejects are answered with an empty 500.
- pkg/paddlehandler: a request that cannot be sent is reported.
- models: the webhook payload builders return the encoder error; the recording cost type is covered.

Rules for every test: `Test_<function>` naming, gomock controller named `mc`, synthetic values only, no network, no wall clock, no sleeps except bounded polling on a log entry, process globals restored in cleanup. Errors are asserted with `errors.Is` or message fragments, not only `err != nil`.

Measurement: the command above plus an aggregation that drops generated files and merges duplicate blocks; the command, the exclusion rule and the per-package before/after table go into the PR body.

Verify:
1. Hand-written total at least 90.5% by the measurement above, at the final HEAD with the same flags.
2. All existing tests pass, also with `-race -shuffle=on`; the new tests also pass with `-count=2` (the dbhandler fixtures share one in-memory database and are not rerunnable in a single process, which is unchanged); lint passes.
3. Mutation spot checks: removing a guarded error return or changing a counter in the covered code makes the new tests fail.
4. `git diff --stat origin/main...HEAD` lists only files under bin-billing-manager.
