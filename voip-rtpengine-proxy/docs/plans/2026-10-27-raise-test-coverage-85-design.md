# voip-rtpengine-proxy unit test coverage to 85% (VOIP-1659)

## Goal

Raise hand-written statement coverage (excluding `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files) from 82.26% (589/716) to at least 85% (609), recommended 85.5% (613), expected 614. Only test files are added; product code, `go.mod` and `go.sum` are unchanged (check `git diff` after the `-mod=mod` measurement).
Measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`; duplicate blocks keep the maximum count.

## Common rules

- Table-driven where cases share a shape, `Test_` prefix, no `t.Parallel()`.
- Log lines written by goroutines or warning branches are asserted through a logrus hook: each test swaps hooks with `logrus.StandardLogger().ReplaceHooks` (it returns the previous set) and restores them in `t.Cleanup`. The hook has its own mutex and copies the message and fields in `Fire`, because goroutines log concurrently (`-race`). Entries are matched by exact message (several messages share prefixes, e.g. `failed to process metadata file` and `... on scan`) plus a test-specific field (`t.TempDir()` path, id or listener address), and waiting polls with a deadline (no fixed sleeps). A warning branch is asserted by its log line, not only by the return value, so removing the handling fails the test.
- Every process, goroutine and file a test creates is stopped or removed in `t.Cleanup`.

## Tests

1. `cmd/rtpengine-proxy/main_run_test.go`
   - `Test_main_unknownInterface`: set `interfaceName` to a missing interface (restored in `t.Cleanup`), run `main()` in a goroutine, require it to return within a deadline, assert the "Could not get proxy ID" error log and no "Proxy ID:" info log.
   - `Test_initProm_listenerBusy`: keep a loopback listener in a package variable and never close it (closing lets the retry goroutine bind the port); call `initProm` with an endpoint made unique by a counter (a duplicate pattern panics) and wait for "Could not start prometheus listener" with that listener address in the entry (earlier `-count` iterations leave retry goroutines logging the same text). The retry goroutine leak is intentional; the test is safe with `-count=3`.
2. `pkg/ngclient`: `Test_newNGClient_dialError` dials `[fe80::1]:1` (a link-local address without zone fails with `invalid argument`), asserts the "dial RTPEngine NG" error prefix, and fails (no skip) if the dial succeeds.
3. `pkg/listenhandler`: `Test_listenRun_consumeError`: the mocked `ConsumeRPC` returns an error for both queues; wait for both calls and both error logs before the controller finishes (reuse `expectConsume`-style channels).
4. `pkg/pcapwatcher`
   - `Test_scanAndProcess_processingRetryFails`: an invalid `.processing` file is retried by the scan; assert the retry-failure log.
   - `Test_processProcessingFile_cleanupFailures`: inside the `Upload` mock replace the pcap path (case 1) or the `.processing` path (case 2) with a non-empty directory so `os.Remove` fails with `ENOTEMPTY` regardless of uid; assert nil return, one upload and the matching warning log.
   - `Test_Run_metadataDirRemoved`: the startup scan's `Upload` mock removes the metadata dir so `fsWatcher.Add` fails; assert the "could not watch metadata dir" error.
   - `Test_Run_processFailureWarning`: `Run` in a goroutine; because there is no readiness signal, create a directory in the metadata dir repeatedly (read fails) until the exact "failed to process metadata file" warning (not the `... on scan` variant) appears or a 10s deadline passes; on exit cancel the context and wait for `Run` to return before the temp dir is removed.
5. `pkg/processmanager` (fake tcpdump and the `TestMain` `/tmp` lock are reused)
   - safety timeout kill failure: `Exec` with `time.Hour`, take `m.mu`, delete the entry, then `tracked.timer.Reset(1ms)`; assert the "Safety timeout kill failed for <id>" log; the detached process is killed and waited, and `/tmp/<id>.pcap` removed, in `t.Cleanup`.
   - SIGTERM failure: inject a `trackedProcess` with a command that has already finished (`cmd.Run()`) and a non-nil `time.AfterFunc(time.Hour, ...)` timer (`killAndUpload` always stops it); assert the "Could not send SIGTERM" log and the returned pcap path.
   - local pcap delete failure: inject a finished command with `uploader=nil` and a non-empty directory as the pcap path, call `Kill`; assert the "Could not delete local pcap" log.
   - `Shutdown` with a failing kill: the first `Upload` call (guarded by `atomic.Bool`, never `sync.Once`, which deadlocks on re-entry) deletes every other id under `m.mu`; `Upload` expects `Times(1)`, so the outcome does not depend on map order; assert exactly N-1 warn-level "Shutdown: kill <id>: no process with id" entries (N ids; the info line "Shutdown: killing process" is not counted); the removed entries are captured before deletion so `t.Cleanup` can stop their timer, kill and wait their process and delete their pcap.
   - `CleanOrphans` delete failure: a non-empty `/tmp/<uuid.NewString()>.pcap` directory, skipped when a foreign uuid pcap already exists (as `Test_CleanOrphans` does), removed with `RemoveAll` in `t.Cleanup`; a skip loses that statement, so verification checks `-v` shows no SKIP; assert the "Could not delete orphaned pcap" log.

## Verify

- Same measurement command: 85.0% or more required, 85.5% or more recommended. The raw `go tool cover` total includes mock files and is lower.
- All existing tests pass; `-race` and `-count=3` pass for every touched package.
- `golangci-lint run` and `scripts/check-test-conventions.sh` pass; `git diff --stat origin/main...HEAD` lists only files under `voip-rtpengine-proxy/` and no non-test `.go`, `go.mod` or `go.sum`.
- Mutation spot checks must fail the new tests: drop the early return in `main`, drop each `os.Remove` error check (`watcher.go` pcap and processing, `manager.go` local pcap and orphan), drop the SIGTERM error log, change the `Add` error text in `Run`, skip the `.processing` retry in `scanAndProcess`, drop the `ConsumeRPC` error log, and drop the `initProm` retry log, drop the dial error wrapping in `newNGClient`, drop the shutdown-path warning.
- Test data is synthetic only (`127.0.0.1`, `203.0.113.x`, generated uuids); no credentials or DSNs.
