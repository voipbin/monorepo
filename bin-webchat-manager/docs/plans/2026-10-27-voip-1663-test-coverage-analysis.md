# VOIP-1663 bin-webchat-manager test coverage: analysis

## Ticket review

Verdict: partially valid. The request is to raise the hand-written coverage (excluding `gens/`, `mock_*.go`, `DO NOT EDIT` files; duplicate blocks merged by max count) of this service to at least 90%, 90.5% recommended, with unit tests only. Re-measured on origin/main 0bfbe2a1d with `GOFLAGS=-mod=mod go test -count=1 -coverprofile=cp.out ./...`: 89.18% (1319 of 1479 statements), all tests pass, so the baseline claim holds. 90.0% needs 1332 covered statements (13 more) and 90.5% needs 1339 (20 more). The part that did not hold: the ticket class "cannot be reached" is too broad, because several branches are reachable from tests without product change (see below). Prior coverage work on this service is in the history of the service directory (VOIP-1647, VOIP-1657, VOIP-1659 commits on main); `gh pr list --state all --search "webchat-manager coverage"` shows no open PR for the same target.

## Baseline per package (hand-written)

| Package | Coverage |
|---|---|
| cmd/webchat-control | 53.8% (7/13) |
| cmd/webchat-manager | 12.2% (12/98) |
| internal/config | 97.4% (74/76) |
| models/message, session, widget | 90.0%, 88.9%, 88.9% |
| pkg/cachehandler | 100% |
| pkg/dbhandler | 93.1% (416/447) |
| pkg/listenhandler | 93.4% (451/483) |
| pkg/messagehandler, sessionhandler, widgethandler | 100% |

The reachable total without the deferred item is 50 statements, so the ceiling is about 92.6% and no block is needed. The deferred item is not attempted, because it registers on the default HTTP mux, adds a signal consumer goroutine and starts a listener goroutine, which cannot be isolated from other tests of the package. Product code, signatures and visibility stay unchanged.

## How each reachable item is reached (verified in a scratch copy)

- Encoding failures: the domain and webhook timestamps are `*time.Time`; a value in year 10000 makes `json.Marshal` fail. A mocked handler returns such a model, and the listen handler answers 500. The session `TMDelete` is excluded from encoding, so the test sets `TMCreate`, not `TMDelete`.
- `main` of both commands: `os.Args` set to `--help` makes cobra print the help and return without `os.Exit`.
- Row iteration errors: a minimal in-test `database/sql/driver` whose rows return a fixed error from `Next`, opened with `sql.OpenDB`.
- Unsupported filter: a value such as a slice makes the filter application fail before any SQL.
- Update with an unencodable value: a slice with at least one channel element makes the field preparation fail; an empty slice encodes fine and a bare channel value passes through to the statement (the update functions pass the caller map through).
- migrate subcommand: a connection string pointing at a nonexistent unix socket path (no credentials), set in the environment before `initCommand`, because `Bootstrap` overwrites the config value from viper on every call.

## Risks of shared global state

viper, the config value, logrus settings, `os.Args` and `os.Stdout` are saved and restored in cleanup. `Bootstrap` rewrites the config value on every call while `LoadGlobalConfig` loads once per binary, so tests of the commands set the environment, not the config value. The deferred `runDaemon` item is excluded for the reasons above.
