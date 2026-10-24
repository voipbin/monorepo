# voip-rtpengine-proxy unit test coverage to 60% (VOIP-1647)

Scope: mainly test files under voip-rtpengine-proxy. The only product code change is hardening of duplicate NG reply handling in ngclient; normal request and reply flow is unchanged.

## Baseline (hand-written code, before the test additions)

Hand-written total 58.8% (716 statements). Per package values below are the `go test` percentages, which include generated mock files: cmd/rtpengine-proxy 0.0%, gcsuploader 49.0%, listenhandler 82.9%, ngclient 72.5%, pcapwatcher 77.7%, processmanager 79.5%. The PR description reports the hand-written values of the same run (mocks excluded). Generated code and `mock_*.go` are excluded.

## Uncovered, testable without external services

Coverage numbers: "hand-written" excludes gens/, `mock_*.go` and DO NOT EDIT files (duplicate blocks merged by max count); "raw" includes them (53.4% at baseline).

- listenhandler `NewListenHandler`, `Run`, `listenRun` (about +14 stmt): queue names and types ("normal", "volatile"), `QueueCreate` failure on either queue aborts and no consumer starts for later queues, `ConsumeRPC` is started per queue with consumer name and the request callback. `processRequest` POST /v1/commands dispatch (+1). The trailing log statements of the consumer goroutines are nondeterministic and not counted. Uses the generated MockSockHandler; the async consumer goroutines are synchronized with channels, not sleeps. `listenRun` is called directly; `Run` is covered only thinly.
- ngclient (about +8 stmt): `newNGClient` resolve error (invalid port such as 127.0.0.1:99999, no DNS), `Send` with a value that cannot be bencoded (returns an error) and `Send` after `Close` (returns an error);  the dial error path is not counted, `Close` on an already closed connection (warning path), `readLoop` datagram without separator and with undecodable bencode must not release a pending request and the loop keeps serving a valid reply afterwards.
- Remaining single-block branches in gcsuploader `Upload`, pcapwatcher parser/`processProcessingFile`/`scanAndProcess`, processmanager error branches and listenhandler `processNG` (the response marshal error path) only where reachable with existing helpers and without root-only failures. That path is exercised by a mock `Send` returning a value that cannot be marshaled. Next candidates if the margin is thin: the remaining single-block branches in gcsuploader `Upload`.

Deliberately not tested:
- `cmd/rtpengine-proxy` (`main`, `init*`, `getInterfaceIP`, `registerProxyRedis`). A `_test.go` in that package makes `init()` run in the test binary (flag parsing, a Prometheus listener goroutine), which would add about 90 incidental statements and inflate the total without testing behavior. The target is reached without it. Whether to test `getInterfaceIP` is left to a later change that isolates `init`.
- `gcsuploader.New` (needs ADC and network).
- pcapwatcher `Run` failure branches: the metadata dir failure and context cancel are already covered (`run_test.go`); chmod, fsnotify and the 5 minute rescan need product changes or root-only failures.
- processmanager `CleanOrphans` glob error (fixed pattern) and `os.Remove` failure (root only).
- `registerProxyRedis` success path (needs a RESP stub and a 5 minute sleep goroutine).

Arithmetic (non-cmd only): listenhandler main.go +14, processNG marshal path +2, ngclient +8 = +24 -> 445/716 = 62.2% (measured 447/716 = 62.4%; needs 437 for 61%).

## Test design details

- listenhandler sync: distinct queue names (`perm-q`, `vol-q`); `QueueCreate` calls ordered with `gomock.InOrder`; each `ConsumeRPC` expectation signals a channel from `DoAndReturn` and the test receives one signal per started consumer (bounded `time.After` only as failure guard) before the test returns. The callback is checked with a valid NG command that must reach the mock NG client and return its result (a constant response would not pass); both consumers wait like the real `ConsumeRPC` so a sequential start is detected. Exact args matched (`"rtpengine-proxy"`, false, false, false, 10, non-nil ctx). The callback is captured and fed a `POST /v1/commands` NG command to prove it is `processRequest`, not asserted by `gomock.Any()` alone. First-`QueueCreate`-failure case sets no `ConsumeRPC` expectation and `listenRun` returns synchronously; second-failure case waits for the first queue consumer signal. `Run` is covered with a `QueueCreate` signal and is not counted in the arithmetic.
- ngclient: new multi-datagram UDP stub helper (existing helper untouched) replying to the request cookie with a space-less but valid bencode datagram, then an undecodable one, then one valid reply. A further test sends several replies for one cookie before the reply for the request and checks that the first reply is kept, the extra ones are discarded and the client keeps serving requests. Every client created via `newNGClient` is closed with `defer c.Close()`. The unencodable value for `Send` is chosen by first checking that `zeebo/bencode` errors for it (func or chan). Note: the space-less datagram must be valid bencode, so that removing the separator check makes the test fail.
- Every channel receive in tests is wrapped in `select` with a `time.After` guard (gomock failures inside goroutines are reported through the guard). The undecodable datagram carries the real cookie prefix so that removing the decode check is caught. The separator mutation is caught by the test. In the `Send`-after-`Close` test the client is closed explicitly, then closed a second time on purpose to cover the already-closed warning path.
- processNG marshal failure case asserts status 500, nil Data, nil error.
- Style follows docs/conventions (table driven, `mc` controller with `defer mc.Finish()`, `Wrong match` messages).
- Stop rule: if the hand-written total is below 61% after the main candidates, add the remaining gcsuploader `Upload` branch tests; below 60% after that, stop and report.

## Verify

0. Judgement value is the hand-written total computed by the stated method from a non-race `-coverprofile` run (`go tool cover -func` total is raw and only reported). `-race` runs are separate. New tests are named `Test_*`; existing names are not touched.

1. Hand-written total >= 61% by the stated method, reached with no test file in `cmd/`. Report per-package before/after, hand-written and raw totals.
2. Existing tests pass; `golangci-lint run` and `go vet` pass; `scripts/check-test-conventions.sh` passes for added lines (Test_ naming, gomock controller named mc).
3. Mutation sample: change the volatile queue type, replace the `QueueCreate` error return with a log only (must still compile), remove the NG datagram format check, swap the two queue names; the matching new test must fail.
4. No sleep based synchronization; `go test -race -count=3` passes for touched packages.
5. `go.mod`/`go.sum` unchanged; `git diff --stat origin/main...HEAD` lists only voip-rtpengine-proxy/ files, and tests use only synthetic values, loopback sockets and temp dirs.
