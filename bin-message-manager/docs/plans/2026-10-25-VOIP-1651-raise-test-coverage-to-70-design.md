# VOIP-1651 bin-message-manager unit test coverage to 70%

## 1. Ticket review
Valid. Baseline on origin/main 99b59c386, measured with `GOFLAGS=-mod=mod go test -count=1 -coverprofile` and the hand-written filter (exclude `gens/`, `mock_*.go`, `Code generated ... DO NOT EDIT` files; duplicate blocks merged by max count): hand-written 69.1% (951/1376), raw 63.6%. Target 71% or more (977 covered, at least 26 more statements).

## 2. Analysis
Missed hand-written statements: cmd/message-control 213, cmd/message-manager 151, pkg/dbhandler 19, pkg/requestexternal 18, pkg/listenhandler 13, internal/config 5, other 6 (models/message 1, cachehandler 1, messagehandler 2, messagehandlermessagebird 2).

| Area (missed) | Testable without product change |
|---|---|
| cmd/message-manager (151) | Partly (about 45 statements, of which runService up to the DSN error is about 37): initProm disabled branch, initSignal/signalHandler through the signal channel, runService early return on an invalid database DSN (`sql.Open` fails at DSN parse, no network). `main` (os.Exit) and the RabbitMQ and Redis paths are excluded |
| cmd/message-control (213) | Mostly (about 120): resolveUUID, printJSON, the four command constructors, initCommand, runSend validation, and the first error branch of initHandler/runGet/runList/runDelete with an invalid DSN (parse failure, offline). Success paths need DB, Redis, RabbitMQ and are excluded |
| pkg/dbhandler/message.go (19) | About 10 of 19: scan error (27, 134, 234), cache set error (84, 98), messageUpdateToCache error (94), not found (129), ApplyFields error (214), cache hit return (144). The fixed-input `ToSql` and `PrepareFields` errors (53, 58, 114, 165, 173, 219, 257, 265) are mostly unreachable |
| pkg/listenhandler (13) | About 5: main.go 103 (ConsumeRPC goroutine error), 137, 214, 216. The 8 `json.Marshal` failure branches in messages.go (59, 94, 129, 164) are unreachable for `*message.Message` |
| pkg/requestexternal (18) | messagebird.go: only the NewRequest error (52) is conceivable, and needs a nil context which trips lint SA1012, so excluded; the rest follows client.Do on a fixed URL and private transport. telnyx.go 32 and 37 are unreachable without a nil context |
| config 5, send.go 2, messagebird send 2, webhook.go 1, cachehandler 1 | About 3: messagehandlermessagebird/send.go 33 and messagehandler/send.go 96 with mocks, cachehandler/handler.go 31 with an unmarshalable value. config (flag bind errors, Fatalf) and webhook.go 57 are unreachable |

Reachable gain upper bound per area cannot exceed the missed count. Realistic estimate: about 120 + 45 + 10 + 5 + 3 = 183 at most, about 150 after discounting; either way above 77%. The target needs only 26 or more, so unreachable blocks are accepted as uncovered and the plan is ordered by value.

## 3. Plan
Test files only, added to the existing `*_test.go` file of each package; message-control tests go in `cmd/message-control/main_test.go`. Test names carry a scenario suffix. No `t.Parallel` where global state (viper, http.DefaultTransport) is touched; state is restored by `t.Cleanup`.
1. cmd/message-control (and cmd/message-manager helpers in their own main_test.go): resolveUUID (empty, malformed, valid), printJSON (success, unmarshalable value), cmdSend/cmdGet/cmdList/cmdDelete flag definitions and defaults (limit 100), runSend validation order (customer id missing/invalid, source empty, destinations empty, text empty; each asserts the specific message and that nothing is dispatched), initCommand subcommand tree (message with send/get/list/delete, migrate), initHandler and runGet/List/Delete returning the wrapped initialization error with an invalid DSN (cases that fail before any connection attempt; each test asserts the error text). cmd/message-manager: initProm with endpoint or listen empty, signalHandler and initSignal via the channel, runService with an invalid DSN. viper and config globals are reset per test.
   Each test maps to the uncovered line ranges in the cover profile, checked before and after.
2. dbhandler: use the existing sqlite harness and fake driver. Cases: Get cache hit (db not queried), Get not found, Get after cache miss with cache set error ignored, Create/Update/Delete with a closed DB (exec error), query error, MessageList filter error (unsupported filter type), scan error row, cache update failure ignored after create/update.
3. listenhandler: list with bad URI, bad filters, handler error; POST bad JSON, handler error; ID GET and DELETE with short URI and handler error; errorResponse and Run error branches where the existing table allows.
4. Others: send.go (messagehandler line 96, messagebird line 33) with mocks, cachehandler setSerialize with an unmarshalable value. telnyx already has its error cases; nothing is added there.

Global state: signalHandler and initSignal start a goroutine that never ends and reads a package channel, so the cmd/message-manager tests call them at most once, assert through the channel, and are checked with `-race -shuffle=on -count=3`. `config.Get().DatabaseDSN` is set directly and restored by `t.Cleanup`.

## 4. Verify
- Standard workflow: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`; `go.mod` and `go.sum` unchanged.
- Measurement: same command and same hand-written filter as the baseline, before and after, per package table in the PR. Result must be 71% or more.
- `go test -race -shuffle=on -count=3 ./...` passes.
- Mutation sample (scratch copy only, never the worktree), at least 6, each mapped to a failing test: resolveUUID nil check removed; runSend destinations check removed; Get cache-hit early return removed; Get not-found mapping changed; list filter error swallowed; listenhandler bad-URI status changed.
- Hygiene: fictitious values only, no network, no `time.Sleep`, no self-comparing assertions; `scripts/check-test-conventions.sh` clean.
- Scope: `git diff --name-only origin/main...HEAD` lists only `bin-message-manager/**/*_test.go` and this document; no non-test `.go` file changes.
