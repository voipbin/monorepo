# voip-rtpengine-proxy unit test coverage to 60% (VOIP-1647)

Scope: test files only under voip-rtpengine-proxy. No behavior change in product code.

## Baseline (origin/main d68e695b5, hand-written code)

Hand-written total 58.8% (716 statements). Measured per package (go test -coverprofile): cmd/rtpengine-proxy 0.0%, gcsuploader 49.0%, listenhandler 82.9%, ngclient 72.5%, pcapwatcher 77.7%, processmanager 79.5%. Generated code and `mock_*.go` are excluded.

## Uncovered, testable without external services

Coverage numbers: "hand-written" excludes gens/, `mock_*.go` and DO NOT EDIT files (duplicate blocks merged by max count); "raw" includes them (53.4% at baseline).

- listenhandler `NewListenHandler`, `Run`, `listenRun` (about +14 stmt): queue names and types ("normal", "volatile"), `QueueCreate` failure on either queue aborts and no consumer starts for later queues, `ConsumeRPC` is started per queue with consumer name and the request callback. `processRequest` POST /v1/commands dispatch (main.go:94, +1). The trailing log statements of the goroutines (main.go:54, 55, 77) are nondeterministic and not counted. Uses the generated MockSockHandler; the async consumer goroutines are synchronized with channels, not sleeps. `listenRun` is called directly; `Run` is covered only thinly.
- ngclient (about +8 stmt): `newNGClient` resolve error (invalid port such as 127.0.0.1:99999, no DNS), `Send` bencode encode failure (unencodable value, ngclient.go:48) and `Send` after `Close` (write failure, :64);  the dial error (ngclient.go:30) is unreachable on loopback and not counted, `Close` on an already closed connection (warning path), `readLoop` datagram without separator and with undecodable bencode must not release a pending request and the loop keeps serving a valid reply afterwards.
- Remaining single-block branches in gcsuploader `Upload`, pcapwatcher parser/`processProcessingFile`/`scanAndProcess`, processmanager error branches and listenhandler `processNG` (command.go:148) only where reachable with existing helpers and without root-only failures. command.go:148 (json.Marshal failure) is reached by a mock `Send` returning a map holding a func value. Next candidates if the margin is thin: gcsuploader.go:43 and :52-55.

Deliberately not tested:
- `cmd/rtpengine-proxy` (`main`, `init*`, `getInterfaceIP`, `registerProxyRedis`). A `_test.go` in that package makes `init()` run in the test binary (flag parsing, a Prometheus listener goroutine), which would add about 90 incidental statements and inflate the total without testing behavior. The target is reached without it. Whether to test `getInterfaceIP` is left to a later change that isolates `init`.
- `gcsuploader.New` (needs ADC and network).
- pcapwatcher `Run` failure branches: the metadata dir failure and context cancel are already covered (`run_test.go`); chmod, fsnotify and the 5 minute rescan need product changes or root-only failures.
- processmanager `CleanOrphans` glob error (fixed pattern) and `os.Remove` failure (root only).
- `registerProxyRedis` success path (needs a RESP stub and a 5 minute sleep goroutine).

Arithmetic (non-cmd only): listenhandler main.go +14, command.go:148 +2, ngclient +8 = +24 -> 445/716 = 62.2% (needs 437 for 61%).

## Test design details

- listenhandler sync: distinct queue names (`perm-q`, `vol-q`); `QueueCreate` calls ordered with `gomock.InOrder`; each `ConsumeRPC` expectation signals a channel from `DoAndReturn` and the test receives one signal per started consumer (bounded `time.After` only as failure guard) before the test returns. Exact args matched (`"rtpengine-proxy"`, false, false, false, 10, non-nil ctx). The callback is captured and fed a `POST /v1/commands` with an empty body to prove it is `processRequest` (400 response), not asserted by `gomock.Any()` alone. First-`QueueCreate`-failure case sets no `ConsumeRPC` expectation and `listenRun` returns synchronously; second-failure case waits for the first queue consumer signal. `Run` is covered with a `QueueCreate` signal and is not counted in the arithmetic.
- ngclient: new multi-datagram UDP stub helper (existing helper untouched) replying to the request cookie with a space-less but valid bencode datagram, then an undecodable one, then one valid reply; one valid reply per cookie (the pending channel has capacity 1). Single loopback sender assumed. Every client created via `newNGClient` is closed with `defer c.Close()`. The unencodable value for `Send` is chosen by first checking that `zeebo/bencode` errors for it (func or chan). Note: the space-less datagram must be valid bencode, so removing the separator check panics (`data[:idx]`) and the test binary fails.
- Every channel receive in tests is wrapped in `select` with a `time.After` guard (gomock failures inside goroutines would otherwise hang). The undecodable datagram carries the real cookie prefix so that removing the decode check is caught. The separator mutation fails the package by panic. In the `Send`-after-`Close` test the client is closed once explicitly; no deferred second close.
- processNG marshal failure case asserts status 500, nil Data, nil error.
- Style follows docs/conventions (table driven, `mc` controller with `defer mc.Finish()`, `Wrong match` messages).
- Stop rule: if the hand-written total is below 61% after the main candidates, add gcsuploader.go:43 and :52-55; below 60% after that, stop and report to default.

## Verify

0. Judgement value is the hand-written total computed by the stated method from a non-race `-coverprofile` run (`go tool cover -func` total is raw and only reported). `-race` runs are separate. New tests are named `Test_*`; existing names are not touched.

1. Hand-written total >= 61% by the stated method, reached with no test file in `cmd/`. Report per-package before/after, hand-written and raw totals.
2. Existing tests pass; `golangci-lint run` and `go vet` pass; `scripts/check-test-conventions.sh` passes for added lines (Test_ naming, gomock controller named mc).
3. Mutation sample: change the volatile queue type, replace the `QueueCreate` error return with a log only (must still compile), drop the NG separator check, swap the two queue names; the matching new test must fail.
4. No sleep based synchronization; `go test -race -count=3` passes for touched packages.
5. `go.mod`/`go.sum` unchanged; `git diff --stat origin/main...HEAD` lists only voip-rtpengine-proxy/ files, and tests use only synthetic values, loopback sockets and temp dirs.
