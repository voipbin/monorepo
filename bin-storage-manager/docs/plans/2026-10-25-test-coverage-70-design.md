# bin-storage-manager unit test coverage

## Goal

Raise the unit test coverage of the hand-written code of bin-storage-manager to 70% or more. Hand-written code excludes `mock_*.go` and files with a `Code generated ... DO NOT EDIT` header (the six `mock_main.go` files). Duplicate blocks of one profile are merged with the maximum count. The measurement command is `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in the service directory.

## Baseline

2542 hand-written statements, 1738 covered (68.37%). The two commands under `cmd/` had 0 covered statements (670 in total) and every other package was at or above 87%. The coverage of `pkg/subscribehandler` varies by one statement between runs because a goroutine start is sometimes not instrumented.

## Approach

Only test files are added. No product code, `go.mod` or `go.sum` changes.

- `cmd/storage-control`: flag definitions of all ten sub commands, the command tree, `resolveUUID`, `printJSON`, the input validation of `runFileCreate` and `runAccountCreate` (which returns before any connection), and the handler initialization failure of the other `run*` functions. The database DSN is set to an unparsable value without a slash, so the driver rejects it while parsing and no connection is attempted. `migrate status` is executed with the same DSN and must fail with the DSN parse error.
- `cmd/storage-manager`: flag definitions, `createDBHandler` with the unparsable DSN, and `runListen` / `runSubscribe` with a mock socket handler (queue creation failure, and consumer start with the goroutine synchronized through a channel).
- `internal/config`: every field mapping of `LoadGlobalConfig`.
- `pkg/listenhandler`: `errorResponse` with a typed nil error.
- `pkg/filehandler`: boundaries of the WAV header and peak decoding (multi channel, bucket count above frame count, truncated data, odd chunk sizes, malformed headers).

Global state (`viper`, the global config, `os.Stdout`) is reset and restored with `t.Cleanup`. Tests do not use `t.Parallel`, and they pass with `-race -shuffle=on -count=3`.

Not covered on purpose: `main`, `run`, `runService`, `initProm` and the code behind real Redis, RabbitMQ and database connections.

## Result

2151 of 2542 statements (84.62%). Per package: cmd/storage-control 0.0 to 67.1, cmd/storage-manager 0.0 to 36.1, internal/config 90.4 to 95.9, pkg/filehandler 89.1 to 91.6, pkg/listenhandler 91.8 to 92.3, the others unchanged.

## Verification

- All tests pass, `golangci-lint` reports no issue, and `go mod tidy` leaves `go.mod` and `go.sum` unchanged.
- Mutation samples on a copy (resolveUUID empty check, a required-flag check, JSON indentation, listen and subscribe error return, flag default, migrate DSN argument, init error text, typed nil response status, WAV pad byte, WAV channel loop, WAV data size, config field mapping) are each caught by a new test.
- Rollback: revert the PR; it only adds tests.
