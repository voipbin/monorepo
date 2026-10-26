# VOIP-1657 bin-webchat-manager test coverage: analysis

## 1. Ticket review

Valid. Measured on origin/main (`GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`, no failing test): hand-written code 1117 of 1479 statements, 75.52%. Hand-written excludes `gens/`, `mock_*.go` and files with a `Code generated ... DO NOT EDIT` header; duplicate blocks are merged by maximum count. 80.0% needs 1184 statements (+67), 80.5% needs 1191 (+74).

## 2. Baseline per package (hand-written)

| Package | Covered / total | % |
|---|---|---|
| cmd/webchat-control | 0 / 13 | 0.0 |
| cmd/webchat-manager | 1 / 98 | 1.0 |
| internal/config | 74 / 76 | 97.4 |
| models/message, session, widget | 9/10, 8/9, 8/9 | 88.9 to 90.0 |
| pkg/cachehandler | 0 / 67 | 0.0 |
| pkg/dbhandler | 363 / 447 | 81.2 |
| pkg/listenhandler | 451 / 483 | 93.4 |
| pkg/messagehandler | 40 / 52 | 76.9 |
| pkg/sessionhandler | 77 / 95 | 81.1 |
| pkg/widgethandler | 86 / 120 | 71.7 |

## 3. What is uncovered and how it can be reached

- Handler packages (widget, session, message): database error returns of `Get`, `List`, `Update`, `Delete`, `End`, `Create`, the orphan-direct cleanup of widget creation, the failed second read after a write, and the `NewXxxHandler` constructors. All reachable with the existing gomock mocks and no signature change.
- `pkg/dbhandler`: exec and query failures, row scan failures, cache-hit returns, empty `fields` returns, `NewHandler`. Reachable with a second, private in-memory SQLite database (no tables for exec and query failures, a table with a non-numeric value in a numeric column for scan failures) and the existing `MockCacheHandler`.
- `pkg/listenhandler`: the remaining uncovered statements are response marshal-failure returns (plain structs cannot fail to marshal) and the typed-error and not-found lines of the `processRequest` error switch, which no handler reaches because every handler turns its errors into a response itself. Nothing is planned here.
- `cmd/webchat-control`: `initCommand` builds the command tree without any connection and `PersistentPreRunE` only binds flags and loads the config, so both are testable; `main` and the `migrate` subcommand (database connection) are not.
- `cmd/webchat-manager`: `runListen` (with a mock sock handler) and `initSignal` (fed through the package channels) are testable. `main`, `runDaemon`, `run` and the metrics server goroutine need external connections or a listening server and are left out.
- `pkg/cachehandler`: needs a Redis stand-in. A test-only module (miniredis) would be needed; it is only used if the measured total is short of the target.
- Not reachable without a signature change: configuration binding errors, marshal errors of constant-shaped values, query builder errors, field preparation errors.

## 4. Reachable ceiling

Handlers (about 56 statements of 64 uncovered), `dbhandler` (about 40), `cmd` helpers (about 18) add up to about 114 statements, i.e. about 83%, without any new module. The ceiling is above 80%, so work continues without escalation. Target: 80.5% or more, with the priority on error paths, then boundaries.

## 5. Constraints

Tests only, inside bin-webchat-manager. No product code change, no signature or visibility change. Synthetic data, no network, no time dependence.

## 6. Isolation notes

- A second SQLite database must be a private one (`:memory:` without `cache=shared`) with `SetMaxOpenConns(1)`; the shared one used by the existing tests is never modified. A closed `*sql.DB` is the simplest way to make exec and query fail. A scan failure needs the same columns as `scripts/database_scripts_test` with a non-numeric text value in a numeric column.
- `initSignal` registers process-wide signal handling: call it once, drive it through `chSigs` (no real signal), drain `chDone`, and call `signal.Stop(chSigs)` in cleanup.
- `initCommand` uses the global viper: reset it after the test.
- `widgethandler` and `sessionhandler` use the real UUID generator unless a mock is injected into the `utilHandler` field; expectations use the injected mock or `gomock.Any()` for generated ids.
- Measured at origin/main 161e9748bb4d94228deb71fab41f734fb5e1307c; later commits on this branch change only docs/plans of this service.
