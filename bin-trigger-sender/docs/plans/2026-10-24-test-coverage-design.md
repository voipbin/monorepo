# bin-trigger-sender unit test coverage (VOIP-1647)

## 1. Ticket review

Valid. Baseline (origin/main d68e695b5): 10.6% of 66 statements (hand-written and raw are identical; the service has no generated code or mocks). `main` 0.0% (22 stmts), `buildRequest` 100% (4), `run` 7.5% (3 of 40). 60% needs 40 statements covered (33 more); target 41 (61%). The service has a single package, so the whole gain has to come from `main.go`.

Seven statements in `main` are `os.Exit` bodies (lines 45-49, 51-54, 61-64) and cannot be covered in-process. Everything else is reachable.

## 2. Scope

Test-only change in `cmd/bin-trigger-sender`: new `fakebroker_test.go` and `run_test.go`, plus a one-line comment correction in `main_test.go`. No product code change, no exported-surface change.

## 3. Approach

`run` takes the broker address as a string, so the test starts an in-process fake AMQP 0-9-1 broker on a loopback listener (`127.0.0.1:0`, dummy credentials only, no external network, no cost). It implements only what amqp091-go v1.14.0 needs:

- Handshake: protocol header, Connection.Start (mechanisms contain PLAIN), Tune (frame-max 131072, channel-max 2047, heartbeat 0; heartbeat frames from the client are read and ignored), Open, Channel.Open.
- Queue.Declare with an empty name: reply DeclareOk with a generated queue name; the test asserts the declared arguments (exclusive, auto-delete, non-durable).
- Basic.Consume: read the client-generated consumer tag and use the same tag in ConsumeOk and in Deliver.
- Basic.Publish: method (exchange, routing key), content header (properties flags: content-type bit 15, correlation-id bit 10, reply-to bit 9, in that order) and body frames. The broker records exchange, routing key, content-type, correlation id, reply-to and the decoded body.
- Basic.Deliver with a header frame carrying content-type and correlation-id, and a body frame.
- Channel.Close and Connection.Close are answered with CloseOk unless the mode says otherwise.

Every connection has read and write deadlines, and the listener, accept goroutine and connection goroutines are closed and waited for in `t.Cleanup`, so a protocol mismatch fails fast instead of hanging; server goroutines always leave through a read deadline when the client closes first.

Broker modes and the branches they cover (statement budget in the last column; `run` has 40 statements, `main` 15 reachable, `buildRequest` 4):

| mode | server behaviour | branch |
|---|---|---|
| reply | deliver response JSON, matching correlation id; status table 199, 200, 299, 300, 404, 500 | success, non-2xx (`< 200` and `>= 300` boundaries) |
| bad correlation | deliver a different correlation id | correlation error |
| bad json | deliver a non-JSON body | unmarshal error |
| no reply | never answer, timeout 200 ms or more | `ctx.Done` timeout branch (assert "timeout waiting" and that the broker did receive the publish) |
| reject declare | Channel.Close 406 on Queue.Declare | declare error only (the library marks the channel closed, so the deferred `ch.Close` returns nil and logs nothing; no warning is asserted) |
| reject consume | Channel.Close 406 on Basic.Consume | consume error only (same as above) |
| drop on close | deliver the reply normally; when the client's deferred Channel.Close arrives, close the TCP connection without CloseOk | `run` still returns nil (reply is processed before the deferred closes); `close channel` warning is a required assertion, `close connection` warning is also asserted (a 50-run `-race` scratch check showed both are always logged); prefixes are matched by substring since the error may be EOF or ErrClosed |
| reject open | Connection.Close in response to Channel.Open | `open channel` error (optional) |
| none (working broker through Declare and Consume) | `timeoutMs <= 0` | publish error (the library returns the context error when the context is already done) |
| none (working broker through Declare and Consume) | `data` that is not valid JSON passed directly to `run` | marshal error |
| none | no listener | dial error (existing test) |

Reachable budget: `run` 39 (all but the optional open-channel error), `main` 15, `buildRequest` 4, i.e. 58 of 66 (about 88%) as an upper bound (the open-channel error is asserted too, so `run` reaches 100% and the total 59 of 66, 89.4%); the requirement is at least 41 of 66, so partial coverage of the rare modes is acceptable; 7 `os.Exit` bodies in `main` are not coverable in-process, and re-running the binary as a subprocess does not feed `-coverprofile`, so exit paths and missing required flags are not tested.

`main` is tested by replacing `flag.CommandLine` with a fresh `FlagSet` and `os.Args`, restored in `t.Cleanup`. These tests (and any test using that global state) must not use `t.Parallel()`. Cases: delay 0 and delay greater than 0 (50 ms, lower bound only), with `-data`, `-method`, `-data_type`, `-queue` asserted on what the broker received.

Assertions are on broker-received values (exchange is empty, routing key is the queue, decoded request uri, method, publisher, data_type, request_id, data, content-type, reply-to non-empty, correlation id) and on error message prefixes (`dial RabbitMQ`, `publish`, `unmarshal response`, `non-2xx response: N`, `unexpected CorrelationId`, `timeout waiting`, `marshal request`, `declare reply queue`, `consume reply queue`), not on values the test constructed itself. Warning branches are asserted with a logrus test hook. The comment on the existing `TestRunDialFailure` that says the timeout branch is untestable is corrected.

Conventions: test names use the `Test_` form with table-driven cases and plain `t.Errorf("Wrong match. expect: %v, got: %v")` messages, no testify (per `scripts/check-test-conventions.sh` and docs/conventions/testing.md); the existing test lines are left alone except the corrected comment. The logrus test hook is attached to the global logger and removed with `ReplaceHooks` in cleanup, so hook tests do not run in parallel. The broker replies with the literal correlation id `bin-trigger-sender-cronjob` (not an echo of the received one), and the received value is asserted separately, so a constant mutation is caught on both sides. For delay greater than 0 only a lower bound on elapsed time is asserted. Success-mode tests use a 5000 ms timeout so `ctx.Done` and the reply never race in `select`. The plan lives under the service `docs/plans/` (per-service plan precedent).

## 4. Verify items

1. `GOFLAGS=-mod=mod go test -count=1 -race -coverprofile` passes; hand-written total >= 61%.
2. `-count=20 -race` passes (and a 50-run scratch check of the drop-on-close warnings), no hang, every goroutine started by the test broker has exited (WaitGroup) at cleanup.
3. Mutation sample on a scratch copy; each mutation must fail at least one new test: 2xx lower and upper bounds, `Publisher`, `RequestID`, `DataType`, correlation id constant, `ReplyTo`, exchange argument, queue argument to publish, QueueDeclare flags, removing the correlation check.
4. CLAUDE.md standard five steps (`go mod tidy`, `go mod vendor`, `go generate ./...`, `go test ./...`, `golangci-lint run -v --timeout 5m`) and `scripts/check-test-conventions.sh` (run after committing, it diffs against the merge base) pass; the diff is only under `bin-trigger-sender/`.
5. No real host, credential, customer data or internal URL in tests, comments or docs.

## 5. Risks

The fake broker can diverge from the amqp091-go handshake; mitigated by deadlines and by validating the frames against the library source and a scratch run before writing the full test set.
