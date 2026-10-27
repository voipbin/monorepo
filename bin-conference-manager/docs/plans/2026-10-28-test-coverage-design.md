# bin-conference-manager unit test coverage design

## Goal

Raise the hand-written statement coverage of bin-conference-manager (generated code, mocks and `DO NOT EDIT` files excluded) from 85.2% to 90% or more. Only test files are added. No production code changes.

## Approach

The two command packages hold 338 of the 403 uncovered statements because their start paths opened real connections. The test injection points in bin-common-handler (`sockhandler.SetRabbitFactory`, `databasehandler.SetConnectFunc`, `migrationhandler.SetEngineFunc`) remove that limit. About 132 statements are needed for 90%; the command packages alone can supply that, and the package level work below is the margin.

- Database: the connect function returns an in-memory sqlite database built from `scripts/database_scripts_test` with `SetMaxOpenConns(1)` and a shared in-memory name, so every connection sees the same data. Only the queries that the existing dbhandler tests already run on sqlite are used.
- Config: the config singleton loads once. The `TestMain` of each command package starts one miniredis, sets the redis address and `prometheus_listen_address=127.0.0.1:0` with `viper.Set` (it wins over the host environment variables and flags), and only then loads the config. The config pointer is never mutated by tests. A cache failure is produced by closing the miniredis and restarted on the same address afterwards.
- Message broker: a gomock `MockSockHandler`. `TopicCreateWithKind` always returns nil because a failure ends the process, the calls made from the listen and subscribe goroutines (`QueueCreate`, `QueueBind`, `ConsumeRPC`, `ConsumeMessage`) use `AnyTimes`, and a router answers `RequestPublish` with synthetic responses per URI. Error returns of the listen and subscribe start are made with a mock that returns an error.
- Migration: a fake engine that records the calls.
- Global state per test: `prometheus.DefaultRegisterer` and `DefaultGatherer` are replaced by a fresh registry (requesthandler and the circuit breaker register on every construction), `http.DefaultServeMux` by a fresh mux (initProm registers a handler), and each is restored in `t.Cleanup`. viper keys that a command reads are reset before each command test.
- Signals: `runMain` waits on `chDone`. A test helper drains `chSigs` and `chDone` before and after the test, sends exactly one value to `chSigs` (never a real signal), waits for `runMain` with a timeout select, and calls `signal.Stop(chSigs)` on cleanup. `initSignal` leaves one goroutine per call, so the number of `initSignal` calls is kept to the number of `runMain` tests that reach it and each drains its own value.
- `runMain` ignores the error of `run`, so the listen and subscribe failures are tested by calling `run`, `runListen` and `runSubscribe` directly. `runMain` itself is tested for the connect, migration and cache failures and the normal stop.

Isolation rules:

- The mock also allows `Connect` (called by `run` and by the command handler setup), `EventPublish` and `EventPublishWithDelay` with `AnyTimes` because notify and handler goroutines publish after the call returns. The subscribe start failure is tested three times: with a `QueueCreate` error, with a `QueueBind` error, and with a `TopicCreateWithKind` error on a dedicated mock. Tests that trigger them wait for the goroutines they start or tolerate late calls through the `AnyTimes` expectations, so no call reaches a finished test.
- The notify handler publishes webhooks from a goroutine that reads a process wide metric variable of the request handler after the answer, and every request handler construction writes it. The command test answers a webhook publication only when the test binary ends, so such a goroutine never overlaps the next test and `-race` stays clean.
- sqlite lacks `json_array_append`, so the command tests register a sqlite driver with a small implementation of it for the root path.
- The sqlite database name is unique per test (counter based), the connect function hands out that database and `t.Cleanup` closes it.
- The miniredis state is flushed at the start of each test. A restart after a cache failure must succeed on the same address, otherwise the test fails with `t.Fatal`.
- Every `runMain` test, including the failure ones, ends with the same helper: one value to `chSigs`, drain `chDone`, timeout select. The initialization of the handlers is called once per test (and once per subtest with a fresh registry) because each construction registers metrics.
- The server goroutine started by `initProm` stays until the process ends; tests never send requests to it and call `initProm` as few times as possible.
- The command test `TestMain` keeps the invalid DSN for the existing failure tests; new tests install the connect function first and restore it afterwards. Flags are rebound by `initCommand` for every test and the viper keys of a command are reset to their default before it runs.

Tests that use these process-wide injection points do not call `t.Parallel`.

## Tests and order

1. conference-control: every `run*` command with a ready handler: input validation after initialization, success output (stdout captured and decoded) and handler errors. Initialization failures of the database and the cache. Measured after this step.
2. conference-manager: `runMain`, `run`, `runListen`, `runSubscribe`, `initLog`, `initSignal`, `signalHandler`, `initProm`. Measured after this step.
3. Margin, in this order if the total is still below 90.5% (not needed once the two command packages are covered): dbhandler (scan, cache and empty input branches), conferencehandler, conferencecallhandler, cachehandler, subscribehandler, models.

## Verification

- `go test -race -count=1 ./...` and `golangci-lint run` pass in the service directory, also with `-shuffle=on` and repeated runs of the command packages.
- Coverage is measured at the final HEAD with the same flags as the baseline and reported per package with the hand-written total.
- A sample mutation check in a scratch copy confirms that the new tests fail when a production branch is changed.
- Test data is synthetic, there is no external network use except the loopback miniredis, no fixed port and no host file write.
