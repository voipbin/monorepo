# VOIP-1663 bin-tag-manager test coverage to 90%

## 1. Ticket review

Baseline on origin/main 0bfbe2a1d: hand-written coverage 86.85% (938/1080 statements), raw `go test` per-package figures unchanged from the card. Tests all pass. No open PR touches this service's tests. Earlier coverage PRs (VOIP-1647, 1657, 1659) already covered most handler, db and cmd paths.

## 2. Gap analysis

Remaining uncovered statements by area: `cmd/tag-control` command bodies that run after the database connection, `cmd/tag-manager` `run` after the database connection and `runService`, plus a few error branches in dbhandler, listenhandler, config, cachehandler and models that depend on library failures.

## 3. Approach

Only `runService` in `cmd/tag-manager` is reachable without a database or a real broker: it needs a broker connection and a cache handler that is never called. A loopback AMQP stub is copied into `cmd/tag-manager` test files from the `cmd/tag-control` precedent (different `package main`) and extended with replies for queue declare, queue bind, qos and consume. It records queue declarations, bindings (queue, key, exchange) and consumers, and its listener is never closed (reconnect goroutines would only add noise).

`runService` constructs handlers that register Prometheus collectors, so it can run only once per process: a `sync.Once` runs it against the stub and shares the result, and each test run only inspects it, so `-count` and `-shuffle` stay stable. The test saves and restores `*config.Get()`, passes a mock cache handler and a zero `sql.DB`, and waits for the recorded consume with a deadline poll instead of a fixed sleep.

Assertions: no error; both queues (request, subscribe) declared; the subscribe queue bound to the global topic exchange with the tag-manager patterns; a consumer registered on each queue. The topic exchange declaration alone is not asserted because the notify handler declares it too.

No product code is changed. No new module is added.

Expected effect (scratch spike): 980/1080 = 90.74%, 8 statements above 90% and 2 above 90.5%. The final figure is measured again on the final HEAD.

## 4. Verify

- `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`: all pass, hand-written total (generated, mock and `DO NOT EDIT` files excluded) at least 90.5%.
- `go test -race -shuffle=on -count=3 ./cmd/...` passes (other packages are run with `-race -shuffle=on -count=1`).
- Mutation samples that must fail the new test: remove the `runSubscribe` call, remove the `runListen` call, change one bound topic pattern.
- `golangci-lint run` and `scripts/check-test-conventions.sh` pass.
