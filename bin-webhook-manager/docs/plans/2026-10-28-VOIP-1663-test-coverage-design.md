# bin-webhook-manager unit test coverage to 90% (VOIP-1663)

## Goal

Raise the hand-written statement coverage of bin-webhook-manager (generated mocks and
`DO NOT EDIT` files excluded) from 87.4% to at least 90.5%, with unit tests only. No
production code changes. The service start paths (`run`, `runDaemon`, `initHandler`,
`initWebhookHandler`) are the largest untested part; they become reachable through the
test seams of bin-common-handler (`sockhandler.SetRabbitFactory`, `databasehandler.SetConnectFunc`).

## Test isolation rules

- The seams are process-wide: tests that use them do not call `t.Parallel` and restore the
  previous factory with `t.Cleanup`.
- `requesthandler.NewRequestHandler` registers Prometheus collectors on every call and panics
  on the second call in one process. A test helper swaps `prometheus.DefaultRegisterer` for a
  fresh registry per test and restores it.
- Database: `sql.Open("mysql", ...)` is lazy, so a handle that never runs a query needs no server.
- Redis: miniredis, as the existing tests do. Broker: `sockhandler.MockSockHandler` (gomock).
- No network, no fixed ports, no host files. Webhook targets use literal loopback or
  reserved addresses that URL validation rejects before any dial.
- Goroutine-side calls (`ConsumeRPC`, `ConsumeMessage`) are observed through channels with a timeout.

## Planned tests

Existing helpers (`setConfig`, `closedAddr`, `resetViper`, stdout capture, the signal and
`chDone` cleanup of the daemon test) are reused; new helpers use different names.

Sock mock expectations for a full `run`: `Connect`; `TopicCreateWithKind(webhook event topic, "topic")`;
`QueueCreate(request queue, "normal")`; `ConsumeRPC` (goroutine); `QueueCreate(subscribe queue, "normal")`;
`TopicCreateWithKind(global event exchange, "topic")`; `QueueBind` (any number of times, arguments matched; the pattern list is private to its package);
`ConsumeMessage` (goroutine). Goroutine-side calls use the exact method signature in `DoAndReturn`
and signal a channel that the test waits on with a timeout.

cmd/webhook-manager
- `run` success: every call above, with queue and exchange names matched exactly.
- `run` failures, one per failing call: webhook event topic declare, request queue create,
  subscribe queue create, global event exchange declare, queue bind. Each returns the wrapped error
  and no later call is expected (gomock fails on an unexpected one). The error message is asserted. Only the synchronous calls are ordered; goroutine-side calls are not.
- `runDaemon`: success with `chDone` pre-filled; cache failure; run failure. The daemon test cleanup
  (`signal.Stop`, draining `chDone`) is reused. The goroutine left by `initSignal` is a known
  harmless leak of the existing code.
- `main` is not exercised with real arguments: the config loader runs once per process, so such a
  test would depend on test order and on the host environment.

cmd/webhook-control
- `initHandler`: success; cache failure after a successful database connection.
- `initWebhookHandler`: success; topic exchange declare failure.
- `runSendToCustomer` and `runSendToURI` success: output JSON checked field by field. The customer
  command reads an account without a webhook address from the cache; the URI command uses the
  system customer (`cscustomer.IDSystem`, imported) and an address that URL validation rejects
  before any connection. The payload is a routing-keyed event, so the delivery is observed as one
  publish with the expected routing key and the payload unchanged.
- `initCache`: the configured Redis database is the one that receives the write.
- The broker address given to the sock factory is checked against the configuration.

## Prometheus note

`NewRequestHandler` and the circuit breaker register collectors on every call; `notifyhandler`
registers once per namespace. Swapping `prometheus.DefaultRegisterer` per test covers the first two.

## Verify

- Coverage per package and hand-written total with the card's command, before and after.
- `go test -race -shuffle=on -count=3` of the service passes (no flakiness, no leaked goroutine
  failures).
- Mutation sample: change the topic kind, drop an error return in `run`, swap a queue constant,
  change the success JSON field; the new tests must fail each time.
- `go mod tidy` leaves go.mod and go.sum unchanged; the change touches `*_test.go` files and this
  document only.
