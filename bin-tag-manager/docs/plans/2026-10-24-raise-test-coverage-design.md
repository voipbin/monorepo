# bin-tag-manager unit test coverage to 60% (VOIP-1647)

## 1. Ticket review

Baseline (measured on a tree whose bin-tag-manager sources equal origin/main d68e695b5), hand written code only (gens/, mock_*.go and `Code generated ... DO NOT EDIT` files excluded, duplicate blocks merged by max count): 570 / 1080 statements = 52.8%. Raw: 570 / 1210 = 47.1%. Roughly 78 more statements are needed for 60%, about 89 for 61%.

| Package | Hand statements | Covered | % |
|---|---|---|---|
| cmd/tag-control | 230 | 0 | 0.0 |
| cmd/tag-manager | 138 | 0 | 0.0 |
| internal/config | 78 | 20 | 25.6 |
| models/tag | 9 | 8 | 88.9 |
| pkg/cachehandler | 39 | 0 | 0.0 |
| pkg/dbhandler | 142 | 128 | 90.1 |
| pkg/listenhandler | 232 | 207 | 89.2 |
| pkg/subscribehandler | 56 | 52 | 92.9 |
| pkg/taghandler | 156 | 155 | 99.4 |

The business packages are already above 89%. The gap is config, cachehandler and the two cmd packages.

## 2. Scope and hard to test code

Only `_test.go` files are added. No product code, signature or go.mod change.

Not unit testable without a live DB or RabbitMQ (excluded): `cmd/tag-manager` `run`, `runService` (opens a real RabbitMQ connection), `initProm` (registers on the default mux and starts a never ending listener), `main`; `cmd/tag-control` `main`, `initHandler`, `initTagHandler` (real DB and RabbitMQ), and the `run*` paths that call `initHandler` before any validation (`runGet`, `runList`, `runUpdate`, `runDelete`). Only `runCreate` validates before `initHandler`.

Testable although they look like wiring: `runListen` and `runSubscribe` take the `sockhandler.SockHandler` interface (mock in bin-common-handler), and `initCache` can point at the in-test RESP server.

## 3. Plan

Conventions (scripts/check-test-conventions.sh, added lines only): `Test_` prefixed names, `reflect.DeepEqual` with `t.Errorf` (no testify), gomock controller named `mc`.

1. `internal/config` (new file `config_bootstrap_test.go`): every test starts with `viper.Reset()`, unsets the 7 env keys (`os.Unsetenv`, restored in cleanup) and restores `globalConfig`.
   - `Bootstrap`: flags registered on PersistentFlags with defaults (`redis_database` is a string flag "1" here), env override through `t.Setenv` for one case. The bind error branches are unreachable (`Lookup` is never nil) and are excluded.
   - `InitConfig`: one table case per missing flag (7 BindPFlag branches), asserting the wrapped error text.
   - `LoadGlobalConfig`: the `sync.Once` is a package variable, so only this package's tests can reset it (`once = sync.Once{}` before the test and again in cleanup; the old value is never copied, which would trip vet copylocks). One test verifies values loaded from viper using a different value for each of the 7 keys, so a swapped key mapping is caught.
2. `pkg/cachehandler` (new file `handler_test.go`): an in-test RESP server on a 127.0.0.1 ephemeral port that records commands. It supports PING, GET, SET (with EX parsing), SELECT, AUTH, case-insensitive, pipelined. No HELLO is needed for go-redis v8.11.5. Cases: `Connect` success; `Connect` failure (listener closed first; accepted cost is the client retry backoff, under about 1 second); `TagSet` sends key `tag:<id>` with `EX 86400` and the JSON body; `TagSet` then `TagGet` round trip compared through fixed UTC timestamps; `TagGet` miss asserts `errors.Is(err, redis.Nil)`; `TagGet` corrupt JSON asserts `*json.SyntaxError`; `TagSet` server `-ERR` is returned. The marshal error branch of `setSerialize` is unreachable for `tag.Tag` and is excluded. Server and listeners are closed in `t.Cleanup`; the recorded command slice is mutex protected (`-race`). Password case uses the obviously fake value `secret`.
3. `cmd/tag-control` (new file `main_test.go`): every test starts with `viper.Reset()`, unsets the 7 config env keys plus `NAME`, `ID`, `TOKEN`, `LIMIT`, `DETAIL`, `CUSTOMER_ID` (`AutomaticEnv` makes viper read host env for any key, e.g. `NAME` is set on some hosts, which would skip the name check and open a DB connection) and restores the global config through `saved := *config.Get()` / `*config.Get() = saved` in cleanup (`globalConfig` itself is private to the config package). `printJSON` stdout capture uses a pipe with a reader goroutine and restores `os.Stdout` in cleanup. `resolveUUID` (empty, invalid, valid, with `viper.Set`), `printJSON` (captures stdout; a `chan int` value gives the marshal error), `initCommand` asserting sub-command names (`tag`, `migrate`, `create|get|list|update|delete`, migrate children `status|up|down|force` compared by `cmd.Name()`), flag names and defaults (`list --limit` = 100); sub-command name assertions run on a fresh `initCommand()` instance before any `Execute()` (cobra adds `help` and `completion` after execution). The migrate closure body is never run (it would open the default DSN) and is excluded. `runCreate` validation through `cmd.SetArgs([...])` + `Execute()` with `SilenceUsage`/`SetOut`/`SetErr`: missing customer-id, invalid customer-id, missing name, asserting `required` / `failed to resolve customer ID` / `name is required`. Because `PersistentPreRunE` calls `LoadGlobalConfig` (once, not resettable from this package), `initCache` tests inject config right before each case through `config.InitConfig(cmd)` on a command whose `Flags()` define the 7 flags (merged persistent flags are not visible to `InitConfig`), pointing `redis_address` at the in-test RESP server (copy of the minimal server in this test package) or a closed listener.
4. `cmd/tag-manager` (new file `main_test.go`): `init()` runs once a test file exists. Flag defaults read from `rootCmd.Flags()` (`redis_database` is an Int flag here). Signal test: send a value on `chSigs`, read `chDone` with `select` + `time.After` and assert the received value is `true`, never `signalHandler()` directly and never a real OS signal; because the `init` goroutine exits after one signal, the test restarts `go signalHandler()` in `t.Cleanup` so `-count=N` works (a comment notes that `signal.Notify` intercepts SIGINT/SIGTERM/SIGHUP for the test binary and that a failed run can leave a duplicate goroutine). `runSubscribe` with the sockhandler mock: QueueCreate error, TopicCreateWithKind error, first QueueBind error (message contains `could not bind the topic pattern`), success (every QueueBind call is asserted against the literal strings `bin-manager.tag-manager.subscribe` and `bin-manager.event`, call count == 1 (one topic pattern), so a changed shared constant is caught instead of mirrored; `ConsumeMessage` goroutine synchronized through a channel closed in `DoAndReturn`). `runListen` (literal `bin-manager.tag-manager.request`; the delay exchange argument only reaches a log field in `listenhandler.Run`, so it is not observable through the mock and is excluded): QueueCreate error is logged and nil is returned (current behavior), success path with `ConsumeRPC` synchronized by a channel; the `ConsumeRPC` argument mutation is only observed after that synchronization, the synchronous `QueueCreate` argument is the primary observation point. The `QueueBind` count of 1 is an intentional pin of the current single topic pattern (commented in the test).

Fallback if cmd/tag-manager proves untestable: the other three items alone are estimated above 61%.

Gain estimate from profile statement counts: tag-control about 127 to 135 (initCommand 33, cmd* 78, resolveUUID 7, printJSON 5, initCache 4), tag-manager about 48 (init 33, signalHandler 3, runListen 6, runSubscribe 6), cachehandler 39, config up to 56. Needed: +89 for 61%.

## 4. Verify items

- Measurement: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=<scratch>/cp.out ./...` in bin-tag-manager with the profile outside the tree; `git status` is checked afterwards so go.mod, go.sum and stray files stay unchanged; hand total excludes gens/, mock_*.go and `Code generated ... DO NOT EDIT` files, duplicate blocks merged by max count; per package before/after table. Target >= 660/1080 (61%).
- All existing tests pass; `golangci-lint run -v --timeout 5m` clean; `go mod tidy`, `go mod vendor`, `go generate ./...` produce no change; `scripts/check-test-conventions.sh` clean.
- `go test -race -count=3 -shuffle=on` on the touched packages passes (global state, once, `chSigs`, goroutines), runtime within a few seconds, no leaked listeners.
- Every new test asserts concrete expected values or error text; no empty call tests and no self-comparison.
- Mutation sample in a scratch copy, each must fail only the new tests: cache TTL, cache key format, `redis.Nil` / JSON error handling in `TagGet`, `resolveUUID` validation, `runCreate` name check, `list --limit` default, config default, config key mapping swap (`RedisAddress` and `RabbitMQAddress`), `runSubscribe` error propagation, queue name literal at the cmd call site (`cmd/tag-manager/main.go` `runListen` request queue (QueueCreate and ConsumeRPC arguments) and `runSubscribe` subscribe queue; mutating the QueueBind exchange inside pkg/subscribehandler would also fail its existing tests, so it is not part of the new-tests-only criterion), signal to `chDone` path.
- Mutation copies: copy only bin-tag-manager into a scratch directory and place symlinks to the original sibling modules (`replace ../bin-*`) next to it; run them where no MySQL listens on the default DSN (the `runCreate` name mutation would otherwise reach `initHandler` and open a local connection) and confirm the failure reason is the expected assertion; call-site mutations only, never edits to shared constants.
- `git diff --stat origin/main...HEAD` lists only files under bin-tag-manager/, only `_test.go` files plus this document; go.mod and go.sum unchanged.
- The `init` (33) and `signalHandler` (3) statements are counted as soon as any test file exists in cmd/tag-manager; they are not credited to the signal test, whose value is checked by the mutation sample only.
- Test data is synthetic: 127.0.0.1 ephemeral ports, fake values, no network or cost-bearing calls.
