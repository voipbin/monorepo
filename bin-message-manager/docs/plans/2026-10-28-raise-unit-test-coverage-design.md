# bin-message-manager: raise unit test coverage

Scope: unit tests only. No production code, signature or visibility changes. The goal is a hand-written
statement coverage (without `gens/`, `mock_*.go` and `Code generated ... DO NOT EDIT` files) of at least 90.5%;
the baseline on main is 85.9%.

## Approach

The code after the connection steps of `cmd/message-manager` and `cmd/message-control` is reached through the
test setters of `bin-common-handler`, each restored in the test cleanup:

- `databasehandler.SetConnectFunc` returns an in-memory sqlite database created from `scripts/database_scripts_test`.
- `migrationhandler.SetEngineFunc` returns a fake `Engine` that records the calls and can return errors.
- `sockhandler.SetRabbitFactory` returns `MockSockHandler`; `Connect`, `QueueCreate` and `ConsumeRPC` are expected calls,
  and so is `TopicCreateWithKind` of the topic exchange, which `NewNotifyHandler` calls when it is created. `Connect`
  runs once per handler construction, so the expectation uses a matching `Times`.
- Redis is a `miniredis` server whose address is set in the test config.

Preconditions found in the analysis:

- `requesthandler.NewRequestHandler` and the circuit breaker call `prometheus.MustRegister` without a guard, so a
  second construction in one test binary panics. A test helper replaces `prometheus.DefaultRegisterer` with a new
  registry right before each construction (not once per test, since one test may build the handler several times) and
  restores it afterwards.
- `config.LoadGlobalConfig` uses `sync.Once`: a test that runs a cobra command (including `main()`) calls
  `LoadGlobalConfig()` first to use up the once, then sets `*config.Get()` and the flags. The final value is set
  explicitly in the test and restored afterwards.
- `runDelete` publishes through `PublishWebhookEvent`, which starts an event goroutine and a webhook goroutine. The
  webhook goroutine keeps reading package globals of `requesthandler` after its mock call returns, and the next handler
  construction rewrites them, which `-race` reports. The delete test therefore stores a message without a customer (the
  webhook goroutine returns at once) and waits (bounded) for the event call through a channel in `DoAndReturn`.
- `runSend` first reads the customer (`CustomerV1CustomerGet`, an error there is fail-open) and then checks the balance
  (`valid=false` and error cases are both asserted); the customer id is not an internal system id.
- Every failing `runService` case also sends on `chSigs` and waits for `chDone`, through one shared helper, because
  `initSignal` starts a handler goroutine before the first failure point.
- `TopicCreateWithKind` is expected with the exact exchange name and kind constants.
- Database rows are asserted only in the `message-control` tests. `runService` closes the injected database on return, so
  its tests assert the engine call count, error text and queue arguments. Every test uses a new database name.
- The metrics endpoint and listen address are set empty, so no port is bound and no pattern is added to the default mux.
- The signal goroutine of `initSignal` is ended by sending on `chSigs` and waiting for `chDone`; the `ConsumeRPC`
  goroutine is synchronized with a channel and each expectation is an exact `Times(1)`.
- The sqlite database uses a unique shared-cache name or `SetMaxOpenConns(1)`, so every connection sees the same schema.
- The `migrate` command closure of `message-control` is already covered; the fake engine is used for `runService` only.
- Waits on channels have an upper bound (`signalWait`), and `DoAndReturn` bodies use the exact method signature, so a
  failed expectation in a helper goroutine ends the test instead of blocking `runService` on `chDone`.
- A `main()` success test runs without flags and sets `*config.Get()` directly (empty metrics settings), and resets
  viper, since the config loader runs once per process.
- Assertions pin the queue name and type of `QueueCreate`, the `ConsumeRPC` arguments, the engine call count, the error
  text and the database rows, so that the statement count is not the only signal.

## Tests

- `cmd/message-manager`: `runService` success (stopped through `chSigs`), and each failing step in order: database
  connect, migration, cache connect, `run`/`runListen` queue creation. `main` without flags, stopped by an injected signal.
- `cmd/message-control`: `initHandler` (which builds the message handler) success and failure, `runGet`, `runList`, `runDelete`
  success and handler errors, `runSend` failing at the balance check.

## Isolation and verification

- Tests that touch process globals (setters, viper, config, signal channels) do not use `t.Parallel`.
- No fixed ports (metrics endpoint left empty), no network, no process signal, no provider call: the send success
  path is not exercised because it starts a provider call in a goroutine.
- Assertions check outputs, error text, recorded mock calls and database rows, not self-made values.
- Verify: `go test -race -shuffle=on -count=1 ./...` and `-count=5` for the two command packages, `golangci-lint`, hand-written total from `-coverprofile`.
- Mutation spot checks: break an error return or a handler argument in a scratch copy and confirm a test fails.
- Acceptance: hand-written total (generated files, mocks and `DO NOT EDIT` files excluded, equal blocks merged by the
  highest count) of at least 90.5%; `git diff --stat origin/main...HEAD` lists only `bin-message-manager/` files and no
  `go.mod` or `go.sum` change.
- `runGet`, `runList` and `runDelete` tests capture stdout, restore it with `t.Cleanup`, and decode the printed JSON to
  check its fields. Test addresses carry no user or password part and the Redis password stays empty.
- The tests go into new files next to the existing `main_coverage_test.go` of each command (own helper names, no edit of
  existing helpers). A test that calls `initSignal` stops the notification with `signal.Stop(chSigs)` in its cleanup.
- About 63 more statements are needed; the two command packages hold the reachable gap, so no other package is touched.
