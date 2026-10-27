# voip-rtpengine-proxy unit test coverage to 90% (VOIP-1663)

## Goal

Raise the hand-written statement coverage (excluding `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files) from 85.75% (614/716) to at least 90%, recommended 90.5% or more. Only test files are added; product code and `go.mod` are not changed.
Measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in `voip-rtpengine-proxy`, then filtering the profile (duplicate blocks keep the maximum count).

## Findings

- The uncovered statements are mostly the body of `main()` in `cmd/rtpengine-proxy`. It was not testable before because `sockhandler.NewSockHandler(...).Connect()` opened a real broker connection.
- `bin-common-handler` now has `sockhandler.SetRabbitFactory`, which returns a restore function. With a gomock `MockSockHandler` behind it, `main()` can run to the end.
- The other dependencies of `main()` are local: a loopback fake Redis (already in `main_test.go`), a loopback UDP socket for the NG client, `STORAGE_EMULATOR_HOST` for the GCS client (no network call at creation), and `chSigs` (the termination signal channel), which the test fills directly instead of sending a real signal.

## Design

All new tests use the `Test_` prefix, table-driven where it fits, and no `t.Parallel()` (they replace process-wide state).

1. `cmd/rtpengine-proxy/main_run_test.go` runs `main()` in a goroutine per scenario with a bounded wait (`select` with `time.After`). Scenarios:
   - normal start with the pcap watcher enabled (recording dir is `t.TempDir()`, GCS emulator), then termination; asserts the mock calls (`Connect`, two `QueueCreate`, two `ConsumeRPC`) and the "pcap watcher enabled" log;
   - recording dir unset (watcher disabled, no uploader) with an invalid NG timeout (warning and fallback);
   - early returns: no usable interface, Redis registration failure (closed loopback port), invalid NG address (`bad`, no DNS), GCS client creation failure (`STORAGE_EMULATOR_HOST` empty and `GOOGLE_APPLICATION_CREDENTIALS` pointing to a missing file, so host credentials cannot leak in);
   - watcher start failure (recording dir under a regular file) asserted through the error log.
   Each scenario asserts why `main()` returned (log hook text or mock calls), so a wrong early return cannot pass.
2. Process-wide state: the sock factory, `interfaceName`, `redisAddress`, `rtpengineNGAddress`, `rtpengineNGTimeout`, `recordingDir`, `gcpBucketNameMedia` and `chSigs` are saved and restored in `t.Cleanup`. Cleanup first makes sure `main()` has returned (fills `chSigs`, waits for the done channel) and only then restores globals and releases the lock; on the watcher scenarios it waits for the watcher stop log.
3. Host isolation: `procMgr.CleanOrphans()` deletes uuid-named `/tmp/*.pcap`. Scenarios that reach it take the same `/tmp` lock file as `pkg/processmanager` tests (`flock`, polling up to 120 seconds) and hold it until `main()` returned; the check for foreign uuid pcap files runs after the lock is taken. Scenarios that return earlier take no lock.
4. Skips: a missing non-loopback IPv4 interface skips with a logged reason. A lock that cannot be taken or opened, or a foreign pcap file, skips locally and fails when `CI` or `CIRCLECI` is set, so CI cannot pass silently with a lower number.
5. Mock wiring: `ConsumeRPC` `DoAndReturn` uses the full non-variadic signature; the test waits for both calls before it injects the signal, so no mock call happens after the test finished.

## Verify

- Same measurement command; target 90% or more of hand-written statements, 90.5% recommended. The final PR body table uses the same flags (and `-race -shuffle=on` as an additional run) on the final HEAD.
- Existing tests pass; `-race -shuffle=on -count=3` passes for `cmd/rtpengine-proxy`; `go test -v ./cmd/... ./pkg/processmanager/...` shows 0 `--- SKIP` on a host with a non-loopback IPv4 interface; parallel run of the two packages (`-p 8`) passes without skips.
- `golangci-lint run` and `scripts/check-test-conventions.sh` pass.
- Mutation spot checks must fail the new tests: change the Redis key or the queue name passed to `QueueCreate`, drop the `ngTimeout` fallback, skip `CleanOrphans`, invert the watcher enable condition. The effect of `ng.Close` in the defer is not observable from the test and is not claimed.
- Test data uses only `127.0.0.1`, `10.0.0.x` and `203.0.113.x` style values.
- `git diff --stat origin/main...HEAD` lists only files under `voip-rtpengine-proxy/`; `go.mod`, `go.sum` and product code are unchanged.
