# VOIP-1642: lint cleanup of bin-common-handler

## 1. Ticket review
Measured on origin/main d2dcae915 with golangci-lint 2.13.2 (CI pins 2.14.0) using
.golangci-errorlint.yml minus gocyclo/nestif and without issue caps: 59 findings
(errorlint 26, usestdlibvars 23, contextcheck 6, gocritic 1, nilerr 1, noctx 1, revive 1).
Goal: 0 findings with the same config. Behavior must not change.

## 2. Per-finding decision
- usestdlibvars (23, requesthandler talk_*, rag_rags, pipecat_ping): 200/201/204 -> http.Status*. Same values.
- errorlint (26, notifyhandler/publish.go, rabbitmqhandler consume/publish/queue/topic): %v -> %w.
  Message text is identical. Only the errors.Is/As reach changes. Consumers: grep of errors.Is/As
  across the repo finds no amqp/rabbit error matching; the only context.DeadlineExceeded/Canceled
  matches (ai-manager builderhandler, aicallhandler/ping, api-manager authenticate/ai_builder/
  error_translate) test errors that already wrap ctx errors. Every changed Errorf sits inside an
  `err != nil` branch (verified from the diff), so no %!w(<nil>).
- noctx databasehandler/main.go Connect: Ping() -> PingContext(context.Background()). Ping() is
  defined as exactly that, so no behavior change. Connect has no ctx parameter and its signature is
  public, so it is not changed.
- gocritic ifElseChain databasehandler/main.go: rewrite as tagless switch, same conditions and order.
- revive var-naming: HttpStatusErrorMap -> HTTPStatusErrorMap in requesthandler/common.go (+ test
  comment). Code references: only common.go. bin-api-manager error_translate.go:35 mentions it in a
  comment only, so no compat alias and no cross-module change. The comments at common.go:63,134 and
  common_test.go:136 are updated with the rename (stale comment is for the api-manager PR).
- nilerr models/errors/rpc.go FromResponse: documented contract "nil on unmarshal failure"
  -> //nolint:nilerr with reason.
- contextcheck (6), all intentional and documented in code:
  notifyhandler publishEvent (x2 call sites) and PublishEventWithRoutingKey pubCtx (publish.go:245): events are fire-and-forget and
  must outlive the caller ctx; rabbitmqhandler publish.go PublishRPC: publish uses Background on
  purpose, ctx governs the response wait; consume.go ConsumeMessage/ConsumeRPC -> startConsumers:
  worker lifecycle is owned by the connection, not by the Consume call ctx.
  Passing the caller ctx would drop events when the request ctx is cancelled, so
  //nolint:contextcheck // <reason> on each line.

## 3. Verify
- golangci-lint with the measurement config: 0 findings; project .golangci.yml and
  .golangci-errorlint.yml --new-from-merge-base: 0.
- go build/vet/test ./... in bin-common-handler; git diff has no non-ASCII changes.
- go build ./... in dependants (api-manager, call-manager, ai-manager) for the rename.
- Existing tests cover the touched paths; no new tests needed for message-identical changes.
  Add one test that errors.Is reaches the wrapped cause on a rabbitmqhandler path.
- Re-run the measurement with golangci-lint 2.14.0 (CI pin) before handoff.
- nolint form: `//nolint:contextcheck // <reason>` on the exact reported line.
