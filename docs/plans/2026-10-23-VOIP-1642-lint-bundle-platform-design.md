# VOIP-1642 lint cleanup: customer, direct, tag, api-manager

## 1. Premise
Measured on d2dcae915 (the 4 modules are unchanged up to origin/main 10fe47a7a) with golangci-lint 2.14.0 and .golangci-errorlint.yml minus gocyclo/nestif,
issue limits off: customer 34, direct 25, tag 22, api-manager 38 (+268 excluded ServerInterface method-name revive).
Goal: 0 issues with that config (except the 268). Lint only: no config change, style linters, or refactoring.

## 2. Changes
- errorlint (85 fixed, 1 suppressed): `--fix` (%v to %w; auth_delegate becomes `%w: %w`, go 1.20+ is met). Consumers: the errors wrapped are
  squirrel/sql/scan/filter/uuid/rand/queue errors with no ErrNotFound or VoipbinError inside, so no errors.Is/As branch
  changes; api-manager streamhandler errors surface after the websocket upgrade (no HTTP status can change).
  One exception: accesskeyhandler.Delete re-reads the row it just soft-deleted (no tm_delete filter); ErrNotFound is reachable only by a
  physical-delete race and %w would turn a 500 into a 404. To keep behavior it stays `%v` with `//nolint:errorlint // <reason>`.
- gosec G114 (metrics listeners in 3 cmd/*/main.go and api-manager internal/config/main.go:266): `//nolint:gosec // <reason>`;
  a server timeout would change behavior.
- contextcheck: `NewServiceHandler(ctx, ...)` (only caller main.go; ctx is main's Background, same value);
  `go runListenHTTP` and the ai_builder DoChan closure get `//nolint:contextcheck // <reason>` (request ctx comes from gin;
  the shared RPC must not inherit one caller's cancellation); run_external_media.go uses `WithCancel(ctx)` like
  subscription.go (the method has no callers, so no runtime effect).
- noctx: `net.Dial` to `(&net.Dialer{}).DialContext(ctx, ...)`. The method has no callers and no test is added;
  evidence is compile + lint.
- gocritic (4): else-if, `== 0`, if-else chain to a tagless switch in websockhandler/etc.go; same logic.
- nolintlint (2, cachehandler/handler.go): `//nolint:unused // reserved`, preceded by a bare `//` line so gofmt accepts it.
- revive var-naming (18): rename 17 method parameters and flows.go `onCompleteFlowId` to the Go form. Where a local of the new
  name already exists (historyID, billingID, noteID, targetID, sourceID, participantID), the parameter gets an `IDParam` suffix
  (e.g. `historyIDParam`, type-neutral) and the body keeps its names; the others get plain `xID`. The 17 declaration lines also carry the excluded generated method name, so CI's new-line
  gate would report it: add `//nolint:revive // method name must match the generated ServerInterface` on those lines.

New `//nolint` total: gosec 4, contextcheck 2, errorlint 1, revive 17 (generated method names), unused 2 (reformatted, not new).
`NewServiceHandler` takes `ctx` as its first parameter.

## 3. Verify
1. Temporary config per module: 0 issues (api-manager: only the 251 method names not on a changed line; 17 are on nolint lines).
2. CI commands per module: `golangci-lint run -c .golangci-errorlint.yml --new-from-merge-base=origin/main` and the
   standard `golangci-lint run`, both 0 (gocyclo/nestif on changed lines included); `gofmt -l` empty.
3. Root workflow per module (tidy, vendor, generate, `go build ./...`, `go test ./...`); `git diff --numstat`; no non-ASCII change.
4. No new test: the change set keeps behavior (compile, lint and the existing suites are the evidence).
