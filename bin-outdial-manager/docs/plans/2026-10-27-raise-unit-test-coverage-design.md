# bin-outdial-manager unit test coverage to 85% (VOIP-1659)

## Ticket review
Baseline on origin/main a47f5a2a1: hand-written statements 2021, covered 1715 (84.86%). Whole-module value including generated and mock files is 73.6%. The generated/mock exclusion rule is `gens/`, `mock_*.go`, and files with a `DO NOT EDIT` header. No product code change is needed; only `_test.go` files are added.

## Goal
Hand-written coverage of 85.5% or more (at least 1728 of 2021 statements, about 13 more than the baseline; the denominator must stay 2021 because no product code changes). Existing tests stay unchanged.

## Test targets (all reachable without DB, Redis or RabbitMQ)
1. `internal/config`: `LoadGlobalConfig` copies viper values into the global config. The `sync.Once`, the global config and the viper state are reset inside the test and reset or restored by cleanup, so test order does not matter.
2. `pkg/listenhandler`: the 13 handlers that parse the request URI (`v1Outdials*`, `v1Outdialtargets*`) return the parse error and no response when the request URI is not a valid URL (invalid percent escape). The mocks must see no call. Cases are table entries run as subtests, and the error is asserted to be a URL parse error.
3. `cmd/outdial-control`: `initHandler` and each `run*` command return a wrapped initialization error when the database cannot be opened. The DSN points at a nonexistent unix socket (an empty DSN would default to a local TCP port), so the result does not depend on a local database. The global config value is copied before and restored by cleanup. Only `runCreate` validates its flags before `initHandler`, so it needs valid flag values set through the existing `resetViper` helper; the other commands call `initHandler` first. Tests do not use `t.Parallel`. The DSN uses only fake values. Assertions check the wrapped messages (`failed to initialize handlers`, `could not connect to the database`), not only `err != nil`.
4. `cmd/outdial-manager`: `createDBHandler` returns a nil handler and an error for the same unreachable DSN; the connection fails first and the later steps are not reached, so the test fixes the contract (nil handler plus error), not which step fails.

## Not targeted
Branches that need a failure of a call that cannot fail with valid input (JSON marshal, SQL builders, field preparation, config binding), real server loops, and constructors that need live connections. No signature or visibility change is made for them.

## Verify
- Hand-written total at least 85.5% with the measurement command `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`.
- `go test -race -count=5 -shuffle=on` on the touched packages passes (no time or external state dependence).
- Mutation sample: disabling the parse-error return in a handler, returning a nil error from `createDBHandler`, and dropping a field in `LoadGlobalConfig` make the new tests fail.
- Full workflow `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run` and `scripts/check-test-conventions.sh` pass; `git diff --stat origin/main...HEAD` lists only `_test.go` files and this plan under `bin-outdial-manager/`, with no go.mod or go.sum change; `git fetch origin main` runs before the convention script.
