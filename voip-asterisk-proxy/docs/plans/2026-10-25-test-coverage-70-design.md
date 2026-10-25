# voip-asterisk-proxy: raise hand-written unit test coverage to 70%

## Goal and baseline

Raise hand-written statement coverage (excluding `mock_*.go` and `Code generated ... DO NOT EDIT` files, duplicate blocks merged by max count) from 65.1% (675/1037) to above 70%. The working target is 71% or more as a margin for small CI differences. Only test files are added (plus `go.mod`/`go.sum` for a test-only module); no product code changes. The raw `go test` figure for `pkg/servicehandler` (82.9%) differs because it includes the generated mock.

| package | before |
|---|---|
| cmd/asterisk-proxy | 61.8% (471/762) |
| pkg/eventhandler | 65.6% (42/64) |
| pkg/listenhandler | 68.3% (99/145) |
| pkg/servicehandler | 95.5% (63/66) |

## Structural limits

- `init.go` binding-error branches (88 statements) and `main()` (needs a live broker) stay uncovered. The `processRequest` branch that maps a handler error to 400 is also unreachable because every handler returns a nil error. `connectAMI`, `setProxyInfoRedis` and `getAsteriskIDAddress` have no tests today and are reachable.
- `setProxyInfoAnnotation` after the in-cluster config step needs a cluster; only the pre-cluster error return, `getPodName` and `getPodNamespace` are tested. The namespace test skips when the in-cluster namespace file exists on the host.

## Planned tests (about +100 statements)

1. `listenhandler` AMI: invalid JSON, and an `amigo.New(&amigo.Settings{})` that was never connected (error propagated, never a nil pointer; `Connect()` and `Close()` are not called on it because `Connect` would dial the default AMI port and `Close` dereferences a nil adapter). Success path against a loopback fake AMI server (below). ARI: invalid method, a `httptest` server that is closed after its address is captured. `processRequest`: default 404; proxy route with the wrong method. `processProxyRecordingFileMovePost`: bad body 400, service error 500, success 200 with the file list forwarded to the mock.
2. `eventhandler`: `eventARIReceive` publishes the raw payload with type `ari_event` and content type `application/json` to the notify mock, and wraps a read error. `eventARIRun` (endless reconnect loop) is driven through `Run()` in the background against a `httptest` websocket server with its own `ServeMux`: only the first connection sends one message, later connections are refused (atomic counter), and publication is awaited on a channel with a deadline. The test never reads `ariSock` directly. No `DefaultServeMux` and no fixed port in new tests; the existing 8080 based test is moved to `httptest` (test-only edit).
3. `cmd/asterisk-proxy`: `setProxyInfoAnnotation` pre-cluster error return (the test clears `KUBERNETES_SERVICE_HOST` with `t.Setenv` so a pod host never reaches a real API server), `getPodName` (env, hostname fallback), `getPodNamespace` (env fallback), `getAsteriskIDAddress` (unknown interface; loopback-only interface found dynamically with `FlagLoopback`, skip if none; non-loopback case is host dependent, skipped when absent, never logged, and not counted toward the target), `setProxyInfoRedis` with miniredis (poll for the key `asterisk.<id>.address-internal`, check value and 24h TTL; the goroutine cannot be stopped), `connectAMI` against the fake AMI server.
4. Fake AMI server (loopback listener, test file only, one copy per package): sends the banner line first; reads the Login action and checks the username and secret it received; answers with `Response: Success`/`Message: Authentication accepted` ending in a blank line; for each later action echoes the received `ActionID` in a blank-line terminated response. Assertions compare individual keys (`Response`, `ActionID`, message), never the whole map, because the client adds timing keys. Tests wait for `Connected()` with a deadline before calling `Action` or `Close()` (Close is racy before connection) and never send Logoff. The login response is followed by nothing else (the client's login reader discards extra bytes). The server goroutine does not call `t` methods; received username, secret and `ActionID` go through a channel and are asserted on the test goroutine. Requests use a fixed `ActionID`. Cleanup order: `Close()` on the client first (it stops the reconnect loop), then the listener and connections. amigo's internal goroutines outlive `Close()`, so no goroutine leak checker is used.
5. miniredis is used only because the same version (v2.36.1) already exists in other services and `go mod tidy` must leave no other diff. Fake credentials and `127.0.0.1` / `203.0.113.x` values only.

## Verify

- ARI tests also assert the method, URI, body, `Authorization` and `Content-Type` seen by the server. The closed-server case reuses the address of a just-closed listener, which is accepted as a negligible port reuse risk.
- Assertions check observable values (status code, response keys, forwarded request, published payload and type, stored key and TTL), not constants compared to themselves.
- Mutation sample, one per area: AMI error mapping, 404 default, 400/500 mapping, ARI status code mapping, `ari_event` type string, Login username/secret forwarding, Redis key format and TTL, receive error wrap, namespace fallback. Each must make a test fail. `NewListenHandler` and `simpleResponse` are covered too as easy margin.
- Final number: `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, then drop `mock_*.go` and `DO NOT EDIT` files and merge duplicate blocks by max count (same method as the baseline). Pass criterion is 71% or more (737 of 1037). Expected gain per package is recorded in the PR table.
- Mutations are made in a scratch copy only and never committed; the namespace fallback mutation runs on a host without the in-cluster namespace file.
- Gates: `go mod tidy && go mod vendor && go generate ./... && go test -race -count=3 ./... && golangci-lint run`, `bash scripts/check-test-conventions.sh` from the repo root (test names start with `Test_`, no testify, gomock controller named `mc`), `git diff --stat origin/main...HEAD` limited to `voip-asterisk-proxy/`. Test run time must not grow beyond the existing `cmd` package time.
- No external network; loopback listeners only, closed at test end.
