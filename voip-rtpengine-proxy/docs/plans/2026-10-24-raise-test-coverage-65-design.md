# voip-rtpengine-proxy unit test coverage to 65% (VOIP-1649)

## Goal

Raise the hand-written statement coverage (excluding `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files) from 62.4% (447/716) to at least 65%, recommended 66% or more. Only test files are added; product code and `go.mod` are not changed.
Measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in `voip-rtpengine-proxy`, then filtering the profile (duplicate blocks keep the maximum count).

## Findings

- `cmd/rtpengine-proxy` has no test file, so its `init()` never runs under test and all 234 statements are uncovered: `init.go` 115, `getInterfaceIP` 21, `registerProxyRedis` 13, `main` 85.
- `main()` needs RabbitMQ, Redis and tcpdump and is out of scope.
- `pflag` ignores `-test.*` flags, so a test file in the package does not break `init()`.
- `initVariable` and `initProm` cannot be called again from a test (flag redefinition, duplicate mux pattern), so their behavior is checked through the single package `init()` run.
- The 65% target does not depend on the host-dependent success case of `getInterfaceIP`: `init.go` alone adds about 105 statements (447 + 105 = 552, 77%).

## Design

All new tests use the `Test_` prefix and table-driven style, no `t.Parallel()` where `t.Setenv` is used.

1. `cmd/rtpengine-proxy/main_test.go`
   - A package-level `var _ = ...` in the test file sets all 12 environment variables read by `init()` (`INTERFACE_NAME`, `RTPENGINE_NG_ADDRESS`, `RTPENGINE_NG_TIMEOUT`, `RABBITMQ_ADDRESS`, `RABBITMQ_QUEUE_LISTEN`, `REDIS_ADDRESS`, `REDIS_PASSWORD`, `REDIS_DATABASE`, `PROMETHEUS_ENDPOINT`, `PROMETHEUS_LISTEN_ADDRESS`, `RTPENGINE_RECORDING_DIR`, `GCP_BUCKET_NAME_MEDIA`) to distinct synthetic sentinel values that differ from the defaults (`REDIS_DATABASE` is an integer, the Prometheus address is a loopback ephemeral address, the endpoint is not `/metrics`). Package-level vars are initialized before any `init()`, so this makes the result independent of the host environment and keeps the metrics listener off the default port.
   - `Test_init_bindings`: table of variable name to expected sentinel, covering every bound variable; a swapped variable mapping or a changed flag key fails it. The test cannot detect a wrong env name passed to `BindEnv`, because `viper.AutomaticEnv()` already resolves the upper-case key name to the same value; that is out of scope.
   - `Test_getInterfaceIP_errors` and `Test_getInterfaceIP_success`: an unknown interface name returns an error; a loopback interface found at runtime through `net.FlagLoopback` (skip when none) returns the "no IPv4 address" error; the success case finds an interface with a non-loopback IPv4 address at runtime (skip when none) and asserts that the returned id and address are equal, parse as IPv4, are not loopback and belong to that interface's address set (no ordering assumption when it has several addresses).
   - `Test_registerProxyRedis`: a loopback fake RESP server (go-redis v8 sends no `HELLO`; it sends `SELECT` when the database is not 0) answers `+OK` to every command and records them; the test picks the `SET` command out of the record (no assertion on count or order) and asserts the `SELECT` of the given database and the key `rtpengine.<id>.address-internal`, the value and the `ex 86400` TTL. The unreachable case uses the address of a listener that was just closed (immediate connection refused; a port reuse race is theoretically possible and accepted) and asserts the "initial Redis registration" error. The listener and accepted connections are closed with `t.Cleanup`. The refresh goroutine (5 minute period) cannot be stopped and is harmless for `-count=3` and `-race`.
2. `pkg/gcsuploader`
   - `Test_New` and `Test_Upload_copyFailure` are added to the existing `upload_test.go` (reusing its fake endpoint). `Test_New`: cases are split. Success sets `STORAGE_EMULATOR_HOST` with `t.Setenv`. Failure sets `STORAGE_EMULATOR_HOST` to empty and `GOOGLE_APPLICATION_CREDENTIALS` to a file with invalid content, so local ADC cannot leak into the case.
   - `Test_Upload_copyFailure` uses a directory as the source file, which passes `os.Open` and fails in `io.Copy` (Linux behavior, asserted by the "could not upload to GCS" error text, which differs from the open failure text). The fake endpoint answers 200 and 500; both assert the error text and an empty URI, and the 500 case exists to run the writer-close warning branch.

## Verify

- Same measurement command; the target is 65% or more of hand-written statements (`gens/`, `mock_*.go` and generated files excluded). The raw `go tool cover` total includes mock files and is lower.
- Existing tests all pass; `-count=3` and `-race` pass for `cmd/rtpengine-proxy` and `pkg/gcsuploader`.
- Manual check that nothing listens on the default metrics port after `go test` (not asserted in a test).
- `golangci-lint run` and `scripts/check-test-conventions.sh` pass.
- Mutation spot checks must fail the new tests: change the Redis key format, change the TTL, drop the loopback skip in `getInterfaceIP` (needs a loopback interface on the host), swap two variable mappings in `init.go`, change one flag key in `init.go`. Removing the `To4()` check and renaming an env name passed to `BindEnv` may survive (host dependent and `AutomaticEnv` respectively) and are recorded as such, not as failures.
- Test data uses only `127.0.0.1`, `10.0.0.x` and `203.0.113.x` style values.
- `git diff --stat origin/main...HEAD` lists only files under `voip-rtpengine-proxy/`; `go.mod`, `go.sum` and product code are unchanged.

## Result

Measured hand-written coverage after implementation: 589/716 = 82.3% (raw 589/789 = 74.7%). The mutation checks listed above that are expected to fail did fail.
