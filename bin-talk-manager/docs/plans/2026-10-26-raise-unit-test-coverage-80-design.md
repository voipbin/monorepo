# bin-talk-manager: raise hand-written unit test coverage to 80%

## Goal and baseline

Hand-written statements (files under gens/, mock_*.go and `Code generated ... DO NOT EDIT` files are excluded; duplicate blocks keep the maximum count) are at 79.30% (1789 of 2256). The target is 80.5% or more: 1817 statements, 28 more (80.0% needs 16 more). Only test files are added; no production code, signature or exposure change.

## Scope

Error and validation paths reachable through the public methods without external connections:

- pkg/dbhandler: failures of result set handling for Get, List and FindDirect methods of chat, message and participant; the list pagination token; an empty id list; invalid filter and update map values.
- pkg/participanthandler, pkg/messagehandler: error and fallback branches of add, remove and get (a missing chat on add, a failing member count update on add and remove, a not found result on message get).
- pkg/listenhandler: the error response mapping for an unserializable typed error.
- Constructors `New` of dbhandler, messagehandler, participanthandler, reactionhandler (as the existing chathandler `Test_New`, each injected collaborator is used through the returned handler; a collaborator that the handler only stores, such as the redis client of dbhandler, is checked for being kept).
- Spare candidates if the target is not met: `initProm` and `main` with `--help` in cmd/*.

## Test design

- Reuse the scripted `database/sql` driver of pkg/dbhandler. A new test file adds a rows type that embeds `valueRows` and fails `Close`; `scripted_driver_test.go` is not modified. The `sql.Register` call inside the test of rows_err_test.go (the only one not guarded yet; test-only edit) and the new driver are registered once with `sync.Once`, so `-count=N` works.
- A failing `Close` is returned to the caller only when the rows were not read to the end, so it is exercised through single-row Get/FindDirect calls and through list calls that stop on a scan error (rows with fewer columns than the model). A global logrus hook (`logrustest.NewGlobal`) asserts exactly one error entry containing `Failed to close rows` and an unchanged return value; the existing hooks are saved with `ReplaceHooks` before `NewGlobal` and restored in `t.Cleanup`. These log tests cover Get, FindDirect and List calls only. No `t.Parallel`.
- Required assertions: scan failures return a nil result and an error containing `scan failed`; the token tests assert the SQL text (`tm_create <`); the empty id list and the invalid filter tests (`struct{}{}` value) fail if the query script is called; invalid update map values (a slice holding a channel, which cannot be encoded) return an error and `recordedExecs()` stays empty; the error response test uses typed error details that cannot be encoded expects a 500 response and, through the same logrus hook, exactly one `ToResponse failed` error entry (a plain error logs nothing); handler tests assert returned errors and mock calls with fixed uuid strings.
- No time, network or random dependence; each test builds its own state.

## Verify

1. `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run` pass; `go.mod` and `go.sum` unchanged. Coverage uses `GOFLAGS=-mod=mod go test -count=1 -coverprofile` (the local vendor directory can differ from go.mod) and the exclusions above; hand-written total >= 80.5%, with an intermediate measurement after the dbhandler tests (the intermediate numbers go into the PR body; the dbhandler candidates alone are expected to add about 28 statements; the handler, constructor and listenhandler tests add about 11 more).
2. `go test -race -count=1 ./pkg/...` passes, and the new tests also pass with `-race -count=5` (the existing sqlite based tests keep their rows between repeated runs and are run once).
3. Mutation check, run on a scratch copy (the worktree product code stays untouched), record the failing test name for each; each must fail at least one new test: remove the scan error return (ChatGet), remove the close error log (ChatGet), remove the token condition (MessageList), remove the empty-input shortcut (ParticipantListByChatIDs), remove the filter error return (ChatList), remove the nil chat check (ParticipantAdd), remove the not found mapping (MessageGet), remove the typed error branch of `errorResponse`, drop one collaborator field from each of the four `New` functions.
4. `scripts/check-test-conventions.sh` passes and `git diff --stat origin/main...HEAD` lists only bin-talk-manager files.
