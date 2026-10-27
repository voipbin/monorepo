# bin-call-manager unit test coverage to 85%

## Goal and scope

Raise the hand-written statement coverage of `bin-call-manager` from 83.2% to at least 85.5%. Hand-written means: not under `gens/`, not `mock_*.go`, no `Code generated ... DO NOT EDIT` header; the denominator includes `pkg/testhelper`. Only unit tests (and helpers inside test files) are added. No production code, signature or visibility change.

Baseline (origin/main 320acbfc2, `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, a flag needed only when the local vendor directory is stale; CI runs without it): 11738 statements, 1967 uncovered, 83.24%. 85.0% needs 207 more, 85.5% needs 265 more.

## Candidates (uncovered statements measured per function)

| Area | Targets | Candidate gain |
|------|---------|----------------|
| cmd | command constructors and `initCommand` (flags, subcommand wiring), `resolveUUID`, `printJSON`, `runSubscribe`, `runRequestListen` (mock sockhandler error paths) | about 110 |
| listenhandler | `v1_calls`, `v1_confbridges`, `v1_groupcalls`, `v1_outbound_configs`, `v1_recordings`, `v1_external_medias`: bad JSON, bad id, handler error mapping | about 120 |
| channelhandler | `mute`, `hold`, `moh`, `silence`, `playback`, `health`, `start` | about 60 |
| callhandler | `chained_call`, `start_service`, sip direct queue start, `status`, `media`, `recovery`, `outgoing_call` | about 90 |
| dbhandler, confbridgehandler, config | Get/List not-found paths on the in-process sqlite test schema; `join`, `external_media`, `recording`, `terminate`; `LoadGlobalConfig` | about 80 |

Candidate total is about 460 against 265 needed (about 1.7x). Branches reachable only through a real external connection or a library failure are not tested. After each area, coverage is re-measured with `go tool cover`; if the total is below 85.5% after all areas, the next-largest uncovered functions per `go tool cover -func` are added. The PR body carries the per-file table.

## Isolation rules

- cmd tests run serially, reset viper in cleanup, use `t.Setenv`, restore redirected stdout in cleanup. Initializers that open external connections or register process-global state (metrics endpoint, signal notification) are not called. Goroutine-starting paths use a mock that returns immediately; blocking channel tests use channel synchronization with a bounded timeout, no sleeps.
- No database, Redis or RabbitMQ connection; fixed UUIDs and timestamps; mocks with exact arguments.
- `Test_` naming, gomock controller named `mc`, synthetic values only.

## Verify

1. Root CLAUDE.md workflow (`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run`) and `scripts/check-test-conventions.sh` pass; `go test -race -count=3` on touched packages.
2. Hand-written total at least 85.5% by the stated exclusion rule (duplicate coverprofile blocks merged by maximum count; the measuring script and flags are written in the PR body); per-package before/after in the PR body.
3. `git diff --name-only origin/main...HEAD | grep -v '_test.go$'` lists only this document and go.mod/go.sum if touched; all paths under `bin-call-manager/`.
4. Mutation check in a scratch copy, on code exercised by the new tests: at least 6 mutations across at least 4 packages, half condition inversions and half removed error returns; every one must fail a new test. Count and results go in the PR body.
5. Review checklist: every test asserts an outcome; no assertion-free test, no comparison of a value against itself, no `gomock.Any()` as the only check.
