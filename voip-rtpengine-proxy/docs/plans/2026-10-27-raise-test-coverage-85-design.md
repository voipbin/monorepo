# voip-rtpengine-proxy unit test coverage to 85% (VOIP-1659)

## Goal

Raise hand-written statement coverage (excluding `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files) from 82.26% (589/716) to at least 85% (609), recommended 85.5% (613) or more. Only test files are added; product code and `go.mod` are not changed.
Measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...` in `voip-rtpengine-proxy`; duplicate blocks keep the maximum count.

## Design

All new tests are table-driven where several cases share a shape, use the `Test_` prefix, and use no `t.Parallel()`. Goroutine-side log lines are awaited with a logrus test hook and polling with a deadline, never a fixed sleep.

1. `cmd/rtpengine-proxy/main_test.go`
   - `Test_main_unknownInterface`: set `interfaceName` to a name that does not exist, call `main()`, expect an early return; restore the variable with `t.Cleanup`. No network is touched.
   - `Test_initProm_listenerBusy`: hold a loopback listener open for the whole test binary lifetime, call `initProm` with a unique endpoint and the busy address, wait for the "Could not start prometheus listener" log. The endpoint must be unique per call because `http.Handle` panics on a duplicate pattern (so the test does not support `-count>1` for that case; the endpoint is made unique with a counter).
2. `pkg/ngclient`: `Test_newNGClient_dialError` uses a link-local IPv6 address without a zone (`[fe80::1]:1`), which fails in `DialUDP` with `invalid argument`; the test skips if the dial unexpectedly succeeds.
3. `pkg/listenhandler`: `Test_listenRun_consumeError`: the mocked `ConsumeRPC` returns an error; wait for the error log.
4. `pkg/pcapwatcher`
   - `Test_scanAndProcess_processingRetryFails`: an invalid `.processing` file is retried by the scan and the failure is logged.
   - `Test_processProcessingFile_cleanupFailures`: inside the `Upload` mock, replace the pcap path (and, in a second case, the `.processing` path) with a non-empty directory so the following `os.Remove` fails with `ENOTEMPTY`; independent of uid. Assert the function still returns nil and the upload happened once.
   - `Test_Run_metadataDirRemoved`: the startup scan's `Upload` mock removes the metadata dir so `fsWatcher.Add` fails; assert the "could not watch metadata dir" error.
   - `Test_Run_processFailureWarning`: while `Run` is active a directory is created in the metadata dir (read fails); creation is repeated until the warning is logged, because there is no readiness signal for the watcher.
5. `pkg/processmanager` (reuses the fake tcpdump and the package `/tmp` lock in `TestMain`)
   - safety timeout kill failure: remove the entry from `m.processes` right after `Exec` with a long enough timeout, wait for the "Safety timeout kill failed" log.
   - SIGTERM failure: inject a tracked entry whose process was already waited, so `Signal` fails and `killAndUpload` still returns the pcap path.
   - local pcap delete failure: the pcap path is a non-empty directory.
   - `Shutdown` with a failing `killAndUpload`: the `Upload` mock removes another id (guarded with `atomic.Bool`, not `sync.Once`, to avoid re-entry deadlock); spawned fake processes are killed in `t.Cleanup`.
   - `CleanOrphans` delete failure: a non-empty `/tmp/<uuid>.pcap` directory, with the existing foreign-uuid skip guard and `RemoveAll` cleanup.

## Verify

- Same measurement command; target 85.0% or more, recommended 85.5% or more (expected 614/716, 85.75%). The raw `go tool cover` total includes mock files and is lower.
- Existing tests all pass; `-count=3` and `-race` pass for every touched package (except the single-registration `initProm` case, documented above).
- `golangci-lint run` and `scripts/check-test-conventions.sh` pass; `git diff --stat origin/main...HEAD` lists only files under `voip-rtpengine-proxy/` and no non-test `.go`, `go.mod` or `go.sum`.
- Mutation spot checks must fail the new tests: drop the `os.Remove` warning branches' error check, change the early-return in `main`, change the `Add` error wrapping text in `Run`, remove the SIGTERM error handling, and skip the retry in `scanAndProcess`.
- Test data uses only synthetic values (`127.0.0.1`, `203.0.113.x`, generated uuids); no credentials, DSNs or real hosts.
