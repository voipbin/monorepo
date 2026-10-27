# bin-direct-manager unit test coverage to 90% (VOIP-1663) - analysis

## 1. Ticket review

Baseline on origin/main 9649841c7 (includes the bin-common-handler test seams), `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, 0 failures. Hand-written statements only (without `mock_*.go`, `gens/`, `Code generated ... DO NOT EDIT` files; duplicate blocks merged by max count).

| package | covered/total | % |
|---|---|---|
| cmd/direct-control | 146/218 | 67.0 |
| cmd/direct-manager | 75/130 | 57.7 |
| internal/config | 76/78 | 97.4 |
| models/direct | 6/6 | 100.0 |
| pkg/cachehandler | 45/45 | 100.0 |
| pkg/dbhandler | 107/115 | 93.0 |
| pkg/directhandler | 223/228 | 97.8 |
| pkg/listenhandler | 250/264 | 94.7 |
| pkg/subscribehandler | 56/56 | 100.0 |
| total (hand) | 984/1140 | 86.32 |

90% needs 1026 covered (+42), 90.5% needs 1032 (+48). The ticket figures (86.32%, 1140 statements) are reproduced exactly.

## 2. Where the gap is

156 statements are uncovered. 127 of them are in the two `cmd/` packages (72 in direct-control, 55 in direct-manager), behind calls that open real connections: the database (`databasehandler.Connect`), the schema engine (`migrationhandler.Run`/`New`) and the broker (`sockhandler.NewSockHandler`). The test seams `SetConnectFunc`, `SetEngineFunc` and `SetRabbitFactory` in bin-common-handler replace exactly these three calls (the service code opens connections only through them and through the Redis client, which miniredis serves), so `initHandler`, `initDirectHandler`, the success and result paths of the `run*` commands, `run` after the database step and `runService` can be executed with an in-memory database, a fake engine, a mock sock handler and miniredis.

The other 29 uncovered statements are in the library packages. 13 of them are reachable: the 500 answers of the six handlers that encode a result (list, get, get-by-hash, create, regenerate, delete) when the domain result cannot be encoded (a creation time outside the year range 0..9999 makes the encoder fail) and the field encoding failure of the update query (a map, slice or struct value that contains a type the JSON encoder rejects). The rest are not reachable with valid input or from any caller: query building, the creation query's field encoding, random source, two configuration binding failures, and two error mapping cases of the `listenhandler` dispatcher (the only error a handler passes up is a URI parse error, which is neither of the mapped types). In the cmd packages, 7 more statements are not reachable: the failure exits of the two entry points, two flag binding failures, the metrics listener's success exit and the error return after the listen step of `runService` (the listen step always returns nil). Not reachable in total: 16 + 7 = 23 statements. The reachable total is 156 - 23 = 133, so the upper bound is (984 + 133) / 1140 = 98.0%, above the target. The normal-path statements of the two `main()` functions are reachable only through the process arguments and are not planned, which leaves about 130 planned statements (97.7%).

## 3. Risks and isolation

- The seams, `viper`, the `config` singleton, `chSigs` and `chDone` are process wide: tests that use them do not call `t.Parallel()`, restore the seams with the returned function in reverse order of the calls (`defer`/`t.Cleanup`) and reset `viper` and the config. The direct-manager `resetState` blanks the connection environment variables; the new direct-control tests add the same blanking, so the result does not depend on the CI environment. `config.LoadGlobalConfig` runs once per process through `sync.Once`, so `initHandler` and `runService` tests set `config.Get()` directly instead of relying on the pre-run hook, while `run` calls `config.InitConfig` and so is driven by flags marked as changed (the existing `newServiceCommand` helper).
- `initProm` registers on `http.DefaultServeMux`; a repeated pattern panics, so each call uses a unique endpoint (existing helper). Its goroutine cannot be stopped and ends with the process.
- `init()` of direct-manager starts the signal goroutine, which consumes one value of `chSigs` per process; the new `run` tests never touch `chSigs` and only pre-fill the buffered `chDone`, which they drain.
- No fixed port, no host file, no network: loopback port 0, in-memory sqlite built from `scripts/database_scripts_test/directs.sql`, miniredis, mock sock handler, fake migration engine.
- `requesthandler.NewRequestHandler` and the circuit breaker register Prometheus collectors on `prometheus.DefaultRegisterer` with `MustRegister` on every call, so a second construction in one process panics. Each test that builds the handlers swaps `prometheus.DefaultRegisterer` for a fresh registry and restores it. `notifyhandler` guards its own registration. `NewNotifyHandler` with the global topic option declares the topic exchange and exits the process when that fails, so the mock sock handler always accepts that call and failure paths are injected elsewhere.
- `listenhandler.Run` and `subscribehandler.Run` start goroutines that call the consumer methods of the mock sock handler. Tests that start them wait for those calls (or accept them with `AnyTimes` and a returning `DoAndReturn`) before the gomock controller finishes.
- The in-memory sqlite database uses `mattn/go-sqlite3` (cgo), as the dbhandler tests already do. Every test opens its own database with a unique name, because `run` closes its connection and a shared name would leak rows between tests.
- The existing dbhandler tests share one in-memory database by design; the new cmd tests use their own unique databases and do not touch that package. The direct-manager signal test consumes `chSigs` once per process and relies on `-count=1`; the new `run` tests only fill and drain `chDone` themselves and never use `chSigs`.
