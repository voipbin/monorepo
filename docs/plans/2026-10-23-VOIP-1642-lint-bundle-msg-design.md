# VOIP-1642 lint cleanup: conversation, email, message, webhook, hook, talk, webchat

Goal: zero findings of `.golangci-errorlint.yml` (without gocyclo and nestif, no issue caps) on the seven
modules, with no change to behavior observed outside the services. Measured at d2dcae915 with golangci-lint
2.13.2 (CI pins 2.14.0): 130 findings.

## Decisions

- errorlint (95) and usestdlibvars (5): `%v` to `%w` and `http.Method*`, auto fixed. Every Errorf sits in an
  `if err != nil` branch. Message text is unchanged, only the chain grows. The listenhandlers branch on
  `errors.Is(dbhandler.ErrNotFound)` and `errors.As(*VoipbinError)`; no wrapped site returns those (ErrNotFound
  is returned bare). RPC errors are wrapped in conversation `setVariables*` (callers only log or already use
  `errors.Wrapf`) and webhook `SendWebhookToCustomer` (its only caller answers a fixed 500). Future note:
  wrapping an ErrNotFound or `*VoipbinError` source with `%w` on a listen path changes 500 to the typed code.
- noctx: hook-control and messagebird use `NewRequestWithContext` (messagebird takes its `ctx` parameter; every
  listenhandler ctx is `context.Background()`). webhook: `DialContext` and `CheckRedirect` use the request ctx
  through `net.DefaultResolver.LookupIP`, so client timeouts now also bound those two lookups (the pre-send
  validation lookup keeps no deadline, as before); `sendMessage` and
  `validateWebhookURL` take a ctx and the three delivery goroutines pass `context.WithoutCancel(ctx)`.
- gosec G114 (7): `//nolint:gosec` with one common reason (internal Prometheus listener). Server timeouts are a
  behavior change, left for a separate change.
- email `go h.Send(context.Background(), res)` (contextcheck, G118): `//nolint`, the send outlives the
  request and existing tests pin `context.Background()`. hook-manager `log.Fatalf` after `defer` (gocritic):
  `//nolint`, a startup failure must still exit.
- email wastedassign (3): `var x string`. email revive: `SgMessageID`/`SgEventID`, json tags unchanged.
- talk rowserrcheck (8, the only sites without a check): `rows.Err()` after list loops and before not-found.
  A mid-iteration failure now returns the error (propagated, answered 500) instead of a partial or not-found
  result. Callers stay safe: chat creation still continues after a lookup error, a participant delete is
  reported failed (retryable) instead of "already deleted", and permission checks stay fail-closed.

## Verify

- Measured config on each module: 0 issues. Full workflow per module (tidy, vendor, generate, test, lint).
- New tests that fail on the old code: talk `Test_rowsErrIsReturned`, webhook
  `Test_validateWebhookURL_canceledContext`, message `Test_MessagebirdSendMessage_canceledContext`.
- No non-ASCII added. CI-pinned golangci-lint 2.14.0 is the final check on the PR.
