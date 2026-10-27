# bin-hook-manager unit test coverage to 90% (VOIP-1663)

## Goal

Raise the hand-written statement coverage of bin-hook-manager (generated code, mocks and
`DO NOT EDIT` files excluded) from 80.2% to at least 90.5%. Only test files are added or adjusted;
no product code, signature or go.mod change.

## Approach

The start paths of both binaries open a RabbitMQ connection and a database connection, which
kept them untested. The merged bin-common-handler seams make them reachable:
`sockhandler.SetRabbitFactory` (with `MockSockHandler`) and `databasehandler.SetConnectFunc`.
Every seam call is restored with `defer`, and no such test uses `t.Parallel`.

1. `cmd/hook-control`: `initHandler` and `runSendEmail`, `runSendMessage`, `runSendConversation`
   run with a mock sock handler that expects `Connect` and `RequestPublish`. Success prints the
   result JSON (stdout captured through a pipe, as the existing tests do) and the factory receives
   the configured address; a publish error and an error status return the wrapped error without
   printing. The `--file` payload path uses a temp file.
2. `cmd/hook-manager`: `runService` runs with a connect function that returns a lazily opened
   `*sql.DB` (never dialed) or an error, plus a sock handler. The existing helpers of
   `run_service_test.go` (`prepareRunService`, `uniqueEndpoint`, `runServiceReturns`) are reused: a
   unique metrics endpoint, a loopback listen address, synthetic SSL content, and the config
   pointer of `config.Get()` edited and restored. The TLS server call always returns: without
   privilege the :443 bind fails, and with privilege the synthetic certificate fails to load. The
   plain HTTP server goroutine may bind :80 in a privileged environment and then lives until the
   process ends, so the sock handler used here is a small fake whose `Connect` does nothing and
   whose `RequestPublish` returns a fixed success response, not a gomock controller that ends
   with the test. No assertion depends on the :80 outcome or its log lines.
   `prepareRunService` is called once per test function (it skips when the fixed files exist), so
   repeated runs are split into subtests that each prepare their own state.
3. `requesthandler.NewRequestHandler` and its circuit breaker register Prometheus collectors on
   every call, so a fresh registry replaces `prometheus.DefaultRegisterer` before every call of
   `initHandler`, `NewRequestHandler` or `runService` (also per case of a table test) and is
   restored right after.
4. The `config` singleton is loaded once per process, by `LoadGlobalConfig` (called from the root
   `PersistentPreRunE`, which runs only for an executed subcommand) and the first load wins. A
   hook-control helper therefore calls `config.LoadGlobalConfig()` itself (so `initHandler` never hits
   the not-initialized panic, also for a single `-run`), then overwrites `*config.Get()` with a fixed
   address literal and restores the copy, as `prepareRunService` does. Tests assert that the sock
   factory receives that literal. `Test_initCommand` checks `viper.GetString("rabbitmq_address")`
   (read before the `resetViper` cleanup) instead of the singleton. Subcommands are run through
   `initCommand()` with `SetArgs`; cobra usage text goes to the command's error stream, so those
   tests call `SetOut` and `SetErr` with buffers, and only `printJSON` output (`fmt.Println`) is
   captured through the `os.Stdout` pipe. Tests that swap `os.Stdout` never use `t.Parallel`.
   The database seam returns `sql.Open("mysql", <synthetic DSN without credentials>)`, never a
   zero-value `*sql.DB`. gomock is `go.uber.org/mock/gomock`.
5. Mock expectations name the exact queue, URI, method and data of `RequestPublish`; no
   `gomock.Any()` for those (the context argument may use it). Log assertions on `runService` never use an exact match, a count or
   "no error" (the plain HTTP goroutine can log late into a later hook); a negative "contains" check
   on a specific other message is allowed.

## Budget

577 hand-written statements, 463 covered. 90.5% needs 523 covered, so at least +60 (57 only gives
90.0%). Estimate: hook-control `initHandler` and the three runners about 42, `runService` success
and error paths about 53 that do not depend on the environment; the sum leaves margin over 60.
Process exit paths are excluded.

## Verify

- Expected publish data, URI, method and queue are literals (a JSON string literal for the data, including the
  base64 `received_data` field and, for conversation, the method and an empty signature; the queue
  constant of each target service; the publisher and data type of the request), not values computed with the code under test. The
  "error status" case is a non-2xx `sock.Response` (parseResponse turns it into an error).
- `runService` tests assert the connect function receives `cfg.DatabaseDSN` and the sock factory
  receives `cfg.RabbitMQAddress`, as literals, and the opened `*sql.DB` is closed by the deferred
  call (a `Ping` on it afterwards reports a closed database; the synthetic DSN is parseable). The mutation sample includes emptying one of these arguments.
- A single `-run` of the multi-case tests passes. In a privileged run, later runs may log
  "address already in use" for :80; nothing asserts on it.
- Hand-written total >= 90.5% with `GOFLAGS=-mod=mod go test -count=1 -race -coverprofile`.
- `-count=3 -shuffle=on` and a single `-run` of each new test pass; `/tmp/ssl_*.pem` do not exist
  before the run.
- Mutation sample: change one value in product code (publish target, status text, error wrap) in a
  scratch copy and confirm a new test fails.
- No test depends on the :443 outcome. The package tests are not run in parallel on one host
  (fixed `/tmp/ssl_*.pem`). `go test -v` of the new tests shows no `SKIP`, and the package is run once as an unprivileged
  user and once in a root container (or with `ip_unprivileged_port_start=0`) to see both :80/:443
  behaviors, and `/tmp/ssl_*.pem` are gone afterwards; `-race -count=3 -shuffle=on` covers the swapped Prometheus globals.
- Error paths (publish error, error status) leave stdout empty; stdout capture uses a reader
  goroutine like the existing tests. No docs sync is needed (tests only).
- `git diff --exit-code go.mod go.sum` is clean after the coverage run (`-mod=mod` can rewrite them).
- Coverage is measured per package from `go test -coverprofile` and filtered to hand-written
  files (mock and generated files excluded); the PR body states the method and the totals.
- Expected `Hook.ReceivedURI` follows the real handlers (`Host + URL.Path` for email and message,
  `Host + URL.RequestURI()` for conversation); tests pass a host without a path.
- The fake sock handler embeds `sockhandler.SockHandler` and overrides only `Connect` and
  `RequestPublish` (a comment states that other methods are intentionally unset).
- The :80 goroutine statements are environment dependent; the reported total is the lower of the
  unprivileged and root runs.
- The full workflow runs: `go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`,
  `golangci-lint run -v --timeout 5m`; go.mod and go.sum stay unchanged.
