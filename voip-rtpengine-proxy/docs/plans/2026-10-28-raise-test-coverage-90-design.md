# voip-rtpengine-proxy unit test coverage to 90% (VOIP-1663)

## Goal

Raise the hand-written statement coverage (excluding `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files) from 85.75% (614/716) to at least 90%, recommended 90.5% or more. Only test files are added; product code and `go.mod` are not changed.
Measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in `voip-rtpengine-proxy`, then filtering the profile (duplicate blocks keep the maximum count).

## Findings

- The uncovered statements are mostly the body of `main()` in `cmd/rtpengine-proxy`. It was not testable before because `sockhandler.NewSockHandler(...).Connect()` opened a real broker connection.
- `bin-common-handler` now has `sockhandler.SetRabbitFactory`, which returns a restore function. With a gomock `MockSockHandler` behind it, `main()` can run to the end.
- The other dependencies of `main()` are local: a loopback fake Redis (already in `main_test.go`), a loopback UDP socket for the NG client, `STORAGE_EMULATOR_HOST` for the GCS client (no network call at creation), and `chSigs` (the termination signal channel), which the test fills directly instead of sending a real signal.

## Design

All new tests use the `Test_` prefix and no `t.Parallel()` (they replace process-wide state). They go into a new file `cmd/rtpengine-proxy/main_start_test.go` and reuse `captureLogs`/`waitFor` (`loghook_test.go`), `newFakeRedis` and `ifaceIPv4s` (`main_test.go`). The existing unknown-interface test already covers that early return.

`runMain(t, setup)` is the shared helper: it takes a fresh buffered `chSigs` per scenario (assigned before the goroutine starts), saves and restores the globals the scenario changes (`interfaceName`, `redisAddress`, `rtpengineNGAddress`, `rtpengineNGTimeout`, `recordingDir`, `gcpBucketNameMedia`, `chSigs`), installs the mock sock factory (restored with the returned function), starts `main()` in a goroutine and waits on a done channel with `select` and `time.After`. The signal is injected with a non-blocking send (`select` with `default`), because the buffer holds one value. If the bounded wait for `main()` times out the test fails with `t.Fatal` and does not restore globals the goroutine may still read. Cleanup order (LIFO): gomock controller is created first, then the cleanup that injects the signal, waits for `main()` (bounded) and only afterwards restores globals and releases the lock. Sequence for scenarios that run to the signal wait: wait for the two `ConsumeRPC` calls, then send the signal. Log checks that happen after `main()` returned (the watcher stop log) use `waitFor`, not an immediate count. The tests do not change `rabbitMQAddress`, `rabbitMQQueueListen`, `redisPassword` or `redisDatabase` (the bindings test reads them without restoring); the permanent queue name is asserted against the literal from the bindings test environment, not against the variable. The Redis key IP and the volatile queue IP must be the same value, and it must be in the `ifaceIPv4s` set. `Connect` is expected once in every scenario that reaches the NG client or later (all except the Redis failure); `QueueCreate` and `ConsumeRPC` only in scenarios that reach the listen handler.

| Scenario | Assertions | Mutation it catches |
|---|---|---|
| Start with watcher: recording dir `t.TempDir()`, bucket set, GCS emulator | Redis `SET rtpengine.<ip>.address-internal` where `<ip>` is in the set from `ifaceIPv4s`; `QueueCreate` with the configured permanent name and `normal`, with `rtpengine.<ip>.request` and `volatile`; both `ConsumeRPC`; logs "pcap watcher enabled" and "pcap watcher started"; `metadata` dir exists; after the signal a "pcap watcher stopping" log; an uuid-named orphan pcap planted under the lock is gone | key or queue names, queue types, enable condition, `CleanOrphans` call removed |
| Recording dir set, bucket empty (no uploader) | "pcap watcher disabled" log, no watcher log | `&&` changed to `||` |
| Recording dir empty, bucket set (emulator) | uploader created, "pcap watcher disabled" log, no watcher log | `&&` changed to `||` |
| Invalid NG timeout | warning log with the value, startup continues | fallback warning removed |
| Redis unreachable (closed loopback port) | "Could not register proxy in Redis" log, no sock factory call | early return removed |
| Invalid NG address `bad` (no DNS) | "Could not create NG client" log | early return removed |
| GCS client creation fails (`STORAGE_EMULATOR_HOST` empty, `GOOGLE_APPLICATION_CREDENTIALS` to a missing file) | "Could not create GCS uploader" log, no `QueueCreate` | early return removed |
| Recording dir below a regular file | "pcap watcher error" log, `main()` still ends on the signal | error log removed |

Scenarios 1 to 4 and 8 (those reaching `CleanOrphans`) take the same `/tmp` lock file as the `pkg/processmanager` tests (`flock` polling, up to 120 seconds, longer than the 60 seconds of the processmanager `TestMain` because this side waits for a whole package run under `-race`; the processmanager `TestMain` holds it for its whole run, about 6 seconds, and the packages run in parallel in CI). The foreign uuid pcap check applies to all of them (1 to 4 and 8) and runs after the lock is taken and before the orphan is planted; the orphan is removed in cleanup. The same-path constant is duplicated on purpose, and a lock file owned by another user cannot be opened (`fs.protected_regular`), the same known limit as the processmanager tests.

Skips: every precondition (non-loopback IPv4 interface, lock taken and opened, no foreign uuid pcap file) skips locally with a logged reason and fails when `CI` or `CIRCLECI` is set (stricter than the existing success-case test of `getInterfaceIP`, which skips unconditionally), so CI cannot pass silently with a lower number. The 5 minute Redis refresh goroutine and its client cannot be stopped and are harmless for the lifetime of the test binary; the closed-port race for the Redis failure scenario is accepted as in the existing unreachable-Redis test.

## Verify

- Same measurement command; target 90% or more of hand-written statements, 90.5% recommended. The final PR body table uses the same flags (and `-race -shuffle=on` as an additional run) on the final HEAD.
- Existing tests pass; `-race -shuffle=on -count=3` passes for `cmd/rtpengine-proxy`; `go test -v ./cmd/... ./pkg/processmanager/...` shows 0 `--- SKIP` on a host with a non-loopback IPv4 interface; parallel run of the two packages (`-p 8`) passes without skips.
- `golangci-lint run` and `scripts/check-test-conventions.sh` pass.
- Mutation spot checks must fail the new tests: change the Redis key or the queue name passed to `QueueCreate`, remove the `CleanOrphans` call, change the watcher enable `&&` to `||`, remove the fallback warning. The 5 second NG fallback value and the effect of `ng.Close` are not observable from the test and are not claimed.
- Test data uses only `127.0.0.1`, `10.0.0.x` and `203.0.113.x` style values.
- `git diff --stat origin/main...HEAD` lists only files under `voip-rtpengine-proxy/`; `go.mod`, `go.sum` and product code are unchanged.
