# VOIP-1419: Explicit EventSubscriptionID on every published event type (JSON fallback removal)

- Date: 2026-08-28
- Ticket: VOIP-1419
- Status: Approved (design review loop: R1 Request Changes, R2 Request Changes, R3 Approve,
  R4 Approve -- 2 consecutive)
- Supersedes: the JSON `"id"` fallback clause of the VOIP-1404 design (§4.2 resolution order)
  and its restatement in the VOIP-1405 rollout design. Routing-key SCHEMA, exchange,
  placeholder semantics, and metrics are unchanged -- this document changes only HOW the
  subscription-id segment is resolved.

## 1. Goal and background

VOIP-1404/1405 established the global topic exchange `bin-manager.event` with routing keys
`<publisher>.<resource>.<subscription-id>.<action>`, resolved in two steps: the opt-in
`eventtopic.SubscriptionIdentifier` interface first, then a JSON fallback that extracts the
marshaled payload's top-level `"id"`.

Decision (CEO, 2026-08-28, recorded on VOIP-1419): the fallback is too implicit -- a json
tag rename silently changes routing keys, and nothing forces a new event type to think about
its subscription address. Every published event data type must implement
`EventSubscriptionID()` explicitly, enforced by the compiler, and the fallback is deleted.

Sequencing (confirmed): VOIP-1405 merged + deployed → **VOIP-1419** → VOIP-1406 (consumer
migration). The topic exchange still has zero consumers, so the resolution-mechanism swap
has no subscriber impact in this window.

### Invariance rule (the load-bearing constraint)

**No routing-key VALUE may change.** For every published type, the new explicit method must
return exactly what the two-step resolution returns for it today:

- a type with an existing override keeps its override untouched (22 types);
- a default type whose marshaled payload carries a top-level `"id"` returns that same id
  (44 of the 45 new methods -- verified: no published type has a custom `MarshalJSON`, an
  `omitempty` id, or an id-tag shadowing hazard);
- a type whose payload has no top-level `"id"` (only `*corev1.Pod` today, whose k8s
  `metadata` marshals no such field) returns `""` explicitly -- the same `-` placeholder it
  degrades to today, now stated in code instead of implied by a failed JSON lookup.

The 27 golden routing-key suites pin exact key strings per type and are the regression
fence for this rule.

## 2. Decisions

### D1 -- Webhook path: intersection interface (option b)

`PublishWebhookEvent` forwards its payload into `PublishEvent` (publish.go:32), so the
webhook path must be constrained too. Two options were analyzed:

- (a) embed `SubscriptionIdentifier` into `WebhookMessage` itself → all 55
  `CreateWebhookEvent` implementers must add methods, including 25 types never published as
  events (e.g. `availablenumber.AvailableNumber`, which has no id field at all);
- (b) leave `WebhookMessage` untouched; narrow only the parameter of `PublishWebhookEvent`
  to a new intersection interface.

**Chosen: (b).** The obligation lands exactly on the 30 webhook types that are actually
published as events; the 25 conversion-only DTOs stay out. (a)'s extra 25 methods would be
dead code with invented addresses for types that never reach the exchange.

```go
// notifyhandler/main.go
// WebhookEventMessage is the payload contract of PublishWebhookEvent: the value is both a
// webhook message (CreateWebhookEvent) and an event with an explicit subscription address
// (EventSubscriptionID). PublishWebhook alone still accepts a plain WebhookMessage.
type WebhookEventMessage interface {
	WebhookMessage
	eventtopic.SubscriptionIdentifier
}
```

### D2 -- `PublishEventRaw` keeps `[]byte`; topic path hard-codes the placeholder

A raw payload cannot implement the interface. Sole production caller is
voip-asterisk-proxy (ari_handler.go:76), which is not topic-enabled, so nothing changes in
production. The method comment documents that a topic-enabled service calling Raw publishes
under the `-` placeholder.

### D3 -- `PublishEventWithRoutingKey` unchanged

It never dual-publishes to `bin-manager.event` (caller supplies its own key; VOIP-1258
path) and its only payload is a `json.RawMessage`. Signature stays `interface{}`.

### D4 -- notifyhandler internals

- `PublishEvent(ctx context.Context, eventType string, data eventtopic.SubscriptionIdentifier)` --
  the compile-time enforcement point.
- `PublishWebhookEvent(ctx, customerID, eventType string, data WebhookEventMessage)`;
  `PublishWebhook` keeps `WebhookMessage` (webhook RPC path never touches the exchange).
- `resolveSubscriptionOverride` → `resolveSubscriptionID(data eventtopic.SubscriptionIdentifier) string`
  with a TWO-part guard, both branches returning `""` (→ placeholder):
  1. **nil interface**: `if data == nil { return "" }`. Under the narrowed signature an
     untyped `nil` argument still compiles, and for a nil interface
     `reflect.ValueOf(data).Kind()` is `Invalid` -- the reflect guard alone would miss it and
     the method call would panic. Today's code is safe on this input only because the type
     assertion fails first; deleting the assertion makes this check mandatory.
  2. **typed nil**: the existing reflect guard (`Kind() == reflect.Ptr && IsNil()`) stays --
     an interface-typed parameter still admits typed-nil pointers, which marshal to `null`.
  The `hasOverride` boolean disappears: "unconditional" means only that the
  override-vs-fallback branch is gone. The `topicEnabled` gate around resolution STAYS --
  with the option off, the pre-VOIP-1404 fanout-only path remains untouched, method call
  included (same rationale as today's publish.go:97-99 comment).
- Delete `parseSubscriptionID` and `subscriptionIDData`; drop the `hasOverride` parameter
  from the internal `publishEvent`; `publishTopicEvent` uses the passed subscriptionID
  as-is (placeholder predicate + metrics unchanged). `PublishEventRaw` passes `""`.
- The delayed path (`delay > 0`) returns before any topic publish -- unaffected.
- `requesthandler.CallPublishEvent` bypasses notifyhandler entirely (direct sock publish to
  a subscribe queue, never the exchange) -- audited, out of scope.
- Rewrite the `eventtopic/identifier.go` doc comment for the new contract as a whole -- not
  only the inverted method-set rationale sentence (deferred from VOIP-1405 because that PR
  froze bin-common-handler), but also the now-false "the default (no implementation) is the
  top-level `id` of the marshaled event payload" clause: post-change, implementation is
  mandatory (non-implementation does not compile) and an empty return → placeholder is the
  only degrade path. The corresponding VOIP-1409 checklist item closes with this PR.

## 3. The 45 new methods

Method template (default types -- address is the resource's own id):

```go
// EventSubscriptionID returns the subscription address of this type on the global topic
// exchange `bin-manager.event`: the resource's own id (VOIP-1404 §4.2, VOIP-1419).
func (h *Flow) EventSubscriptionID() string {
	return h.ID.String()
}
```

Placement rule: next to the type declaration, EXCEPT when the type lives in
`models/<entity>/webhook.go` -- then the method goes in a sibling file (e.g.
`subscription.go`), keeping webhook.go a pure wire-shape file and avoiding the RST
pre-commit hook (precedent: bin-ai-manager `models/message/subscription.go`, VOIP-1405).
Each type also gets a compile-time assertion -- **in the package's sibling `_test.go` file,
matching all 26 existing assertions in the repo** (no production model file imports
eventtopic today, and the narrowed call-site signature is itself the production compile
check; the assertion's job is to survive even if a publish site is later removed):

```go
var _ eventtopic.SubscriptionIdentifier = (*Flow)(nil)
```

Special cases:

| Case | Treatment |
|---|---|
| `customer.Customer` | Does not embed `commonidentity.Identity`; has its own `ID uuid.UUID` field. Method returns `h.ID.String()` from that field -- identical key value. |
| `customer.CustomerCreatedEvent` (wrapper embedding `*Customer`) | Method on the WRAPPER: `if h.Customer == nil { return "" }` then `return h.Customer.ID.String()`. The nil-embed guard is mandatory because the receiver chain is computed before any promoted access -- the exact panic class that ruled out Identity-level promotion. The customer golden suite already pins both branches (promoted-id key and nil-embed placeholder key). |
| `*corev1.Pod` (bin-sentinel-manager) | External type; cannot add a method. New wrapper TYPE in the EXISTING `models/pod` package, embedding `*corev1.Pod` anonymously -- empirically verified (compiled against the pinned k8s.io/api version): the marshaled bytes are identical to the bare `*corev1.Pod` (anonymous pointer embed inlines; Pod has no MarshalJSON), a bare Pod carries no top-level `"id"` (identity lives under `"metadata"`), and a nil embed marshals to `{}` without panic. Method returns `""` -- placeholder-by-design, preserving the VOIP-1405 invariant `placeholder_total ≈ publish_total{ok}`. Publish sites (monitoringhandler/run.go:105,117) wrap. **The pod golden file's "Do NOT wrap the payload" header comment is rewritten in the same change**: its rationale (payload-shape change for fanout consumers) does not apply to this shape-preserving embed, and leaving it would forbid the shipped implementation. |
| `contact.WebhookMessage` (contacthandler/event.go:33) | Webhook DTO published as an event since the VOIP-1405 []byte fix. Method returns its own `ID` -- same value the fallback extracts today. Lives in webhook.go → method goes to a sibling file per the placement rule. |
| confbridge event structs (`EventConfbridgeJoined` / `EventConfbridgeLeaved`) | Verified: both embed `Confbridge` BY VALUE, so their marshaled top-level `"id"` is the confbridge id, and the call-manager golden suite pins exactly those keys. Once `(*Confbridge).EventSubscriptionID()` exists, both wrappers would already satisfy the interface via value-embed promotion returning the identical value; explicit methods on the wrappers are nevertheless added deliberately (the ticket's explicitness principle, and insurance against a future same-depth embed silently dropping the promoted method). The one WRONG implementation -- returning `JoinedCallID`/`LeavedCallID` -- is what the golden suite guards against. |

Full to-do set (45; ✅22 existing overrides untouched): agent.Agent; billing.Billing,
billing/account.Account; confbridge.Confbridge, confbridge.EventConfbridgeJoined,
confbridge.EventConfbridgeLeaved; call.Call, groupcall.Groupcall, recording.Recording;
ai.AI, aicall.AIcall, summary.Summary, team.Team; campaign.Campaign; conference.Conference;
contact.WebhookMessage; conversation.Conversation, conversation/account.Account;
customer.Customer, customer.CustomerCreatedEvent, accesskey.Accesskey; direct.Direct;
email.Email; flow.Flow, activeflow.Activeflow; message.Message (message-manager);
number.Number; outdial.Outdial; pipecatcall.Pipecatcall; queue.Queue;
extension.Extension, trunk.Trunk; route.Route, provider.Provider,
providercall.ProviderCall; schedule.Schedule; pod wrapper (sentinel);
file.File, storage/account.Account; tag.Tag; talk chat.Chat; transcribe.Transcribe;
tts/streaming.Streaming, tts/speaking.Speaking; webchat session.Session.

## 4. Tests

- **Golden suites (27 files)**: the `resolveSubscriptionID` helper drops its JSON half and
  mirrors the new production form (interface + typed-nil guard). Every `expect` key string
  stays byte-identical -- that is the point of the change. The "must NOT implement
  SubscriptionIdentifier" negative assertions are inverted into `var _` compile assertions.
- **Per-type behavioral tests**: each new method gets a test in its OWN package asserting
  the returned address equals the type's id field (mutation-checked: change the field, the
  test fails). Wrapper/nil-guard cases (CustomerCreatedEvent, pod.Event) additionally
  assert the nil-embed path returns `""` without panicking.
- **bin-common-handler unit tests**: existing fixtures in notifyhandler/main_test.go update
  to the new signatures; add a case proving `PublishEventRaw` topic-publishes under the
  placeholder when topic-enabled; typed-nil test stays; add a **nil-interface** case
  (`PublishEvent(ctx, t, nil)`) proving it resolves to the placeholder without panicking --
  the mutation lock for D4's guard branch 1.
- **Mock regeneration**: notifyhandler mock regenerates (PublishEvent/PublishWebhookEvent
  signatures). `EXPECT()` recorder calls take `any` and survive; the known typed callback
  (bin-conversation-manager messagehandler/create_case_id_test.go:60,
  `notifyhandler.WebhookMessage`) updates to `WebhookEventMessage`.

## 5. Docs

- `docs/reference/rabbitmq-queues-reference.md`: resolution description updates (fallback
  clause removed; explicit-method contract stated; per-publisher address tables unchanged).
  **The "Deliberate non-overrides (do not 'fix' these)" block (~line 291) is rewritten, not
  left standing** -- it forbids exactly what this PR ships (it names `customer.Customer`,
  `activeflow.Activeflow`, `billing.Billing`, `accesskey.Accesskey`, `recording.Recording`,
  and sentinel's `*corev1.Pod`, all of which now get methods or a wrapper). The customer
  promotion-trap rationale is restated in terms of the wrapper's shadowing method with its
  nil guard, and sentinel's placeholder-by-design now flows through the wrapper's explicit
  `""` return instead of a failed JSON lookup.
- One-line supersession pointers in the VOIP-1404/1405 design docs' resolution sections.
- Service docs: methods change no domain entities; docs/domain.md untouched (the
  check-service-docs hook may warn on models edits -- no doc drift is introduced, but any
  hook warning is resolved per-service during implementation).

## 6. Verification and rollout

- Per-service verification workflow (tidy/vendor/generate/test/lint) for every touched
  service; full 38-module compile sweep because bin-common-handler's public surface changed.
- Wire behavior is unchanged (keys identical, metrics identical), so rollout needs no
  special sequencing: normal image build + redeploy of bin-* services. No consumer exists
  on the exchange; the fanout path is untouched.
- Post-deploy check: `<ns>_topic_publish_total{result="error"}` flat, placeholder counters
  unchanged in character (sentinel remains the only by-design source once it has a runtime).

## 7. Non-goals

- No new event types, resources, or key-schema changes.
- No consumer-side work (VOIP-1406).
- No fanout-path changes; no delayed-event topic semantics.
- No convention-doc overhaul beyond the reference update (VOIP-1409 remains the umbrella
  for conventions, minus the identifier.go item closed here).
