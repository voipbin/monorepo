# bin-timeline-manager: add `peer_events` table (peer/local address-searchable event log)

- Issue: NOJIRA
- Class: new ClickHouse table + subscribe-path insert extension + shared helper
- Service: bin-timeline-manager (+ bin-common-handler for the shared derive helper)
- Date: 2026-07-24
- Author: Lux (CPO), reviewed with 대표님 in #voipbin-cpo

## 1. Problem

`bin-timeline-manager` already stores all 27 subscribed services' events verbatim
in ClickHouse (`events` table), keyed by `(event_type, timestamp)`. That table has
no per-party structure: to find "everything that happened with this phone number
or email," a consumer would have to parse `data` (raw JSON) for every row and
apply publisher-specific field paths.

Separately, `bin-contact-manager` already solved a similar-looking problem with
`contact_interactions` (VOIP-1204), but that table requires a `contact_id`
identity match (peer address -> contact_addresses -> contact) and applies a CRM
eligibility filter that drops internal-resource noise (agent/AI/conference/SIP
legs). That is the right shape for CRM identity resolution, but it is a
different use case: it discards events with no resolvable contact, and it is
scoped to only two publishers already.

The new requirement (confirmed with 대표님, see decision log below) is a
**third**, independent shape: a raw, unfiltered, address-searchable peer/local
event log, scoped to call-manager and conversation-manager only, living in
ClickHouse next to `events`, with **no identity resolution and no eligibility
filter**. It is explicitly NOT a replacement for `contact_interactions` (that
stays as-is) and NOT a widening of the 27-queue subscription (call and
conversation are already subscribed).

## 2. Decision log (from #voipbin-cpo discussion, 2026-07-24)

1. Scope: call-manager + conversation-manager events only. No new queue
   subscription needed — both are already in `subscribeTargets`
   (`pkg/subscribehandler/main.go`).
2. This is additive. `events` (ClickHouse), `timeline_analyses` (MySQL), and
   `contact_interactions`/`contact_addresses`/`contact_resolutions`
   (contact-manager, MySQL) are all left untouched.
3. Name: `peer_events` (paired with existing `events`).
4. No `crmIneligiblePeerTypes` filter — every event that has a
   source/destination or self/peer pair is inserted, without exception. This is
   a deliberate reversal of the contact-manager precedent load-bearing
   assumption; see §7.1.
5. No `contact_id` resolution. The peer/local pair is stored raw, exactly as
   contact-manager's `peer_target`/`local_target` are, but there is no join to
   `contact_addresses` at write time. Identity is entirely a read-time concern
   for whichever consumer queries this table.
6. Search axis: full address-set search (a caller holds N addresses registered
   for a contact and queries "show me every event touching any of these N
   addresses"), not point lookup. This drives the ClickHouse `ORDER BY` choice
   (§4).
7. Columns: include both the derived peer/local pair AND the raw `data` payload
   (duplicate storage vs. `events`, deliberate — avoids a cross-table refetch on
   every read; see §7.2).
8. TTL: 1 year, matching `events`.
9. Event-type scope: ALL event types from call-manager and conversation-manager
   (not just `*_created`). A single call or conversation_message therefore
   produces multiple `peer_events` rows across its lifecycle (§7.3).
   **ROUND 1 CLARIFICATION** (see §5): "all event types" means all event
   types whose payload shape actually carries a peer/local pair (the
   `call.WebhookMessage`, `message.WebhookMessage`, and
   `conversation.WebhookMessage` shapes) — NOT literally every event a
   service ever publishes. `call-manager` also publishes `groupcall_*`,
   `recording_*`, `confbridge_*`, and a raw-map whitelist-rejection event on
   the SAME queue/publisher, none of which carry a compatible payload; these
   are out of scope by construction (§5's explicit allowlist), not by
   omission.

## 3. What already exists (verified) and what does not

Verified in `bin-contact-manager/pkg/contacthandler/interaction.go`:

- `deriveEndpoints(direction string, source, dest commonaddress.Address) (peer, local commonaddress.Address)`
  is the exact peer/local derivation rule needed here (see §6). It is currently
  a private, unexported function local to contact-manager.
- `EventCallCreated` / `EventConversationMessageCreated` are the two existing
  wired subscribers for these two publishers' creation events — but they only
  handle the `_created` event type, apply the CRM eligibility filter, and
  resolve to `contact_id`. None of that is reused as-is; `peer_events` needs its
  own subscribe path inside timeline-manager's existing subscribehandler, not a
  new consumer of contact-manager's handler.

Verified in `bin-call-manager/models/call/webhook.go` and
`bin-conversation-manager/models/message/webhook.go`:

- `call.WebhookMessage` carries `Source`, `Destination` (absolute,
  `commonaddress.Address` value type) and `Direction` (`incoming`/`outgoing`).
- `conversation.message.WebhookMessage` carries the same shape (`Source`,
  `Destination`, `Direction`) — added by VOIP-1215 specifically so per-message
  peer/local derivation is possible without a conversation refetch.
- `conversation.WebhookMessage` (the parent Conversation event, not the
  message) carries `Self`/`Peer` directly (relative axis, no direction needed).

Verified: `Call.Source`/`Destination` are set at call creation and are not
rewritten by `transfer-manager` (transfers create a new call/groupcall
resource; the original call's Source/Destination fields are untouched). So
peer/local is stable across every event in a single call's lifecycle — the
precondition for decision §2.9 (all event types, not just `_created`) holds
for `call`. This must be spot-checked again for `conversation` at
implementation time (conversation's parent Self/Peer are documented immutable
post-create in the VOIP-1215 design, §5 of that doc).

Does NOT exist yet, and is new work in this design:

- A ClickHouse table `peer_events`.
- A shared (non-contact-manager-private) `DeriveEndpoints`-equivalent helper.
- A subscribe-path branch in timeline-manager's `flushBatch` that additionally
  projects call/conversation_message events into `peer_events`.
- A `PeerEventBatchInsert` dbhandler method (mirrors `EventBatchInsert`).

## 4. Schema

```sql
CREATE TABLE IF NOT EXISTS peer_events (
    timestamp     DateTime64(3),
    customer_id   UUID,
    publisher     LowCardinality(String),   -- "call" | "conversation_message" | "conversation"
    event_type    LowCardinality(String),   -- e.g. call_hangup, conversation_message_created
    reference_id  UUID,                     -- call_id / conversation_message_id / conversation_id

    direction     LowCardinality(String),   -- incoming / outgoing / "" (conversation parent has none)

    peer_type     LowCardinality(String),
    peer_target   String,

    local_type    LowCardinality(String),
    local_target  String,

    data          String                    -- raw webhook payload, verbatim (same bytes as events.data)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (customer_id, peer_type, peer_target, timestamp)
TTL toDateTime(timestamp) + INTERVAL 1 YEAR;
```

Migration file: `migrations/000005_create_peer_events_table.up.sql` /
`.down.sql` (golang-migrate, sequential after existing `000004_*`). Also update
`scripts/database_scripts_test/` with the matching test-schema file
(pattern used by `table_timeline_analyses.sql` for the MySQL side; ClickHouse
test fixtures are separately verified at implementation — confirm the exact
test-harness convention for ClickHouse tables in this repo before writing
tests, since the existing test fixture directory currently only shows a MySQL
example).

**`ORDER BY` rationale (§2.6):** the confirmed primary query shape is
"customer's full registered address set, searched at once" — i.e.
`WHERE customer_id = ? AND (peer_type, peer_target) IN [(...), (...), ...]`.
Putting `customer_id, peer_type, peer_target` ahead of `timestamp` in the
sparse primary index lets ClickHouse skip-scan per address-set member instead
of scanning a customer's entire event history and filtering in memory. This
mirrors the existing `events` table's convention of leading with the highest-
cardinality-useful filter column (`event_type`) before `timestamp`.

**Why `data` is duplicated from `events` (§2.7):** avoids a second lookup
(`peer_events` row -> `events` row by some correlating key) on every read; the
two tables have no natural join key today (`events` has no `reference_id`
column, only `publisher`/`event_type`/`data_type`/`data`), so a join-based
dedup would need `events` schema changes out of scope here. Storage cost is
double the payload bytes for exactly the call/conversation_message/conversation
event volume (already the smallest 2 of 27 publishers by count), accepted
trade-off per 대표님's explicit choice in §2.7.

## 5. Ingestion path (extends existing subscribe pipeline, no new queue)

**ROUND 1 CORRECTION (BLOCKER fix):** the original draft filtered on
`publisher in {"call", "conversation_message", "conversation"}`. This is
wrong and would match nothing: `Publisher` is fixed per-SERVICE at
`NewNotifyHandler` construction time (verified
`bin-call-manager/cmd/call-manager/main.go:143` passes `common.Servicename`;
verified `bin-common-handler/models/outline/servicename.go:17,21` — the real
values are `ServiceNameCallManager = "call-manager"` and
`ServiceNameConversationManager = "conversation-manager"`). There is no
`"conversation_message"` or `"call"` publisher value anywhere. Worse, a
single service's `Publisher` value covers MULTIPLE structurally different
webhook shapes: `call-manager` alone publishes `call.WebhookMessage`,
`groupcall.WebhookMessage` (verified `bin-call-manager/models/groupcall/webhook.go` —
`Source *Address` pointer + `Destinations []Address` PLURAL, no `Direction`
field at all), `recording.WebhookMessage`, and `confbridge.WebhookMessage`
(verified: neither carries `Source`/`Destination`/`Direction`), all on the
same `QueueNameCallEvent` queue. A publisher-only filter would either produce
`peer_type=""` garbage rows or fail to unmarshal for every non-call
call-manager event.

**Corrected design: filter on the `(Publisher, EventType)` PAIR, against an
explicit allowlist**, mirroring the proven pattern already in
`bin-contact-manager/pkg/subscribehandler/main.go:147,155`
(`case m.Publisher == publisherCallManager && m.Type == call.EventTypeCallCreated:`).
Only event types whose payload is verified (§3) to carry
`Source`/`Destination`/`Direction` (call) or `Self`/`Peer` (conversation
parent) are in scope — NOT "every event type this service ever publishes."

**Allowlist (exhaustive, verified against each service's `models/*/event.go`):**

| Publisher | Event types (all share the `call.WebhookMessage` shape) |
|---|---|
| `call-manager` | `call_created`, `call_updated`, `call_deleted`, `call_dialing`, `call_ringing`, `call_progressing`, `call_terminating`, `call_canceling`, `call_hangup` (verified `bin-call-manager/models/call/event.go:5-14`) |

| Publisher | Event types (all share `message.WebhookMessage` shape, `Source`/`Destination`/`Direction`) |
|---|---|
| `conversation-manager` | `conversation_message_created`, `conversation_message_updated`, `conversation_message_deleted` (verified `bin-conversation-manager/models/message/event.go:5-7`) |

| Publisher | Event types (share the `conversation.WebhookMessage` shape, `Self`/`Peer`, no direction) |
|---|---|
| `conversation-manager` | `conversation_created`, `conversation_updated`, `conversation_deleted` (verified `bin-conversation-manager/models/conversation/event.go:5-7`) |

**Explicitly EXCLUDED from `peer_events` (same publisher, incompatible or
non-webhook-message payload):** `call-manager`'s `groupcall_*`,
`recording_*`, `confbridge_*` events (different struct shape per publisher
census in §3), and `call.outbound_whitelist_rejected` (verified
`bin-call-manager/pkg/callhandler/outgoing_call.go:213` publishes a raw
`map[string]interface{}`, not `call.WebhookMessage` — cannot be unmarshaled
into the derivation path at all). This narrows §2.9's "ALL event types"
decision to mean "all event types of the two Webhook*Message shapes that
actually carry peer/local data," not literally every event on the queue —
confirmed as the correct reading of 대표님's intent (the goal was "no
`_created`-only restriction," not "ingest structurally incompatible
payloads").

```go
// eligiblePeerEvents is the exhaustive (Publisher, EventType) allowlist for
// projection into peer_events. Any (Publisher, EventType) pair not in this
// set is left in `events` only — never attempted for peer/local derivation.
var eligiblePeerEvents = map[string]map[string]struct{}{
    string(commonoutline.ServiceNameCallManager): {
        call.EventTypeCallCreated:     {},
        call.EventTypeCallUpdated:     {},
        call.EventTypeCallDeleted:     {},
        call.EventTypeCallDialing:     {},
        call.EventTypeCallRinging:     {},
        call.EventTypeCallProgressing: {},
        call.EventTypeCallTerminating: {},
        call.EventTypeCallCanceling:   {},
        call.EventTypeCallHangup:      {},
    },
    string(commonoutline.ServiceNameConversationManager): {
        convmessage.EventTypeMessageCreated:      {}, // message.WebhookMessage shape
        convmessage.EventTypeMessageUpdated:      {},
        convmessage.EventTypeMessageDeleted:      {},
        convconversation.EventTypeConversationCreated: {}, // conversation.WebhookMessage shape (different struct!)
        convconversation.EventTypeConversationUpdated: {},
        convconversation.EventTypeConversationDeleted: {},
    },
}

func (h *subscribeHandler) flushBatch(entries []eventEntry) {
    // existing: build `rows []dbhandler.EventRow` for the `events` table (unchanged)

    // NEW: additionally build peer rows, filtered by the (Publisher, EventType) allowlist above
    peerRows := buildPeerEventRows(entries)   // new pure function, see §6
    if len(peerRows) > 0 {
        if err := h.dbHandler.PeerEventBatchInsert(ctx, peerRows); err != nil {
            log.Errorf("Could not batch insert peer events into ClickHouse. count: %d, err: %v", len(peerRows), err)
            // does NOT block/fail the primary events insert — peer_events is
            // an additive projection, same non-fatal-on-secondary-write
            // posture as the rest of this handler's error handling.
        }
    }
}
```

`buildPeerEventRows` checks each entry's `(event.Publisher, event.Type)`
against `eligiblePeerEvents` FIRST; only matching entries are unmarshaled.
Because `message.WebhookMessage` (per-message, absolute axis) and
`conversation.WebhookMessage` (parent, relative axis) are two DIFFERENT Go
struct shapes under the same publisher, the event-type match also selects
which struct to unmarshal into and whether to call `DeriveEndpoints` at all
(message path) or map `Self`/`Peer` directly (conversation-parent path, §6).
Malformed/unparseable payloads within an eligible `(Publisher, EventType)`
pair are logged and skipped — they still land in `events` (unaffected), just
not in `peer_events`.

Two independent ClickHouse batch inserts per flush cycle (existing `events`
batch + new `peer_events` batch) — not a combined statement, since they are
different tables. This keeps the existing `events` insert path completely
unmodified (§2.2 additive constraint) and isolates any `peer_events`-path
failure from the audit-log write. Both run sequentially inside the single
`flushWorker` goroutine (verified `pkg/subscribehandler/main.go:178-218` —
`flushWorker` is never invoked concurrently with itself), so there is no
ordering/race concern between the two inserts.

## 6. Peer/local derivation (shared helper, not contact-manager-private)

The existing `deriveEndpoints` in `bin-contact-manager/pkg/contacthandler/interaction.go`
is unexported and contact-manager-local. Two consumers now need the identical
rule (contact-manager's CRM projection, timeline-manager's new `peer_events`
projection), so this design promotes it to a shared location:

**Decision: move to `bin-common-handler/models/address` (or a new
`bin-common-handler/pkg/addresshandler` if a free function without an
existing type-anchor is preferred) as an exported function**, e.g.:

```go
// DeriveEndpoints maps an absolute (source, destination) pair to a relative
// (peer, local) pair using direction. Single shared authority for this rule —
// see VOIP-1215 §3.0a for the original contract definition.
func DeriveEndpoints(direction string, source, destination Address) (peer, local Address) {
    switch direction {
    case "incoming":
        return source, destination
    case "outgoing":
        return destination, source
    default:
        return Address{}, Address{}
    }
}
```

`bin-contact-manager` then deletes its private copy and imports the shared one
(one-line change to `interaction.go`); `bin-timeline-manager` imports the same
function for `buildPeerEventRows`. This closes the "two services drifting on
the same rule" risk flagged earlier in this thread.

**ROUND 1 CLARIFICATION:** only the bare direction-switch (`DeriveEndpoints`
itself) moves to `bin-common-handler`. `crmIneligiblePeerTypes` and
`isCRMEligiblePeer` (`bin-contact-manager/pkg/contacthandler/interaction.go:34-77`)
are CRM-specific eligibility judgment, not part of the derivation rule, and
stay contact-manager-private — this is what makes §2.4's "no eligibility
filter for peer_events" decision correct without any code duplication of the
filter logic (there is nothing to duplicate; timeline-manager simply never
imports it).

For the `conversation` parent event (`conversation_created` /
`_updated`/`_deleted`), there is no `direction` field — `Self`/`Peer` are
already stored in the relative axis. `buildPeerEventRows` maps these directly
(`peer = Peer`, `local = Self`) without calling `DeriveEndpoints`.

**Scope note on the shared-helper move:** this is a cross-service change
(bin-common-handler + bin-contact-manager + bin-timeline-manager) and must run
the full verification workflow in all three services, not just
timeline-manager, since bin-common-handler is a shared dependency.

## 7. Open questions for review round 2 (round 1 findings resolved above)

**Round 1 verdict: CHANGES_REQUESTED.** Two BLOCKERs found and fixed in §5/§2.9
above (publisher-vs-event-type conflation; call-manager's multiple incompatible
webhook shapes on one queue). One MAJOR clarified in §6 (eligibility filter
does not travel with the shared helper). Remaining open items carried into
round 2:

### 7.1 No eligibility filter — confirmed noise implication

Per 대표님's explicit decision (§2.4), `peer_events` will contain agent
extensions, AI resources, conference legs, and other internal-resource peer
types that `contact_interactions` deliberately excludes. This is intentional
(raw log, no identity judgment), but any future consumer building an
address-set search UI on top of `peer_events` must apply its own filtering if
it wants a "customer-only" view — this table does not do that. Flagging as an
explicit non-goal, not a gap.

### 7.2 `conversation` (parent) vs `conversation_message` — is publisher `conversation` in scope?

The original discussion scoped this to "call-manager, conversation-manager
events." bin-conversation-manager publishes both parent `conversation_*`
events (self/peer, account-level) and `conversation_message_*` events
(per-message, source/destination). Both are the same publisher
(`conversation-manager`) on the same subscribed queue. This design includes
both (§4 schema's publisher enum lists all three), since excluding the parent
event but keeping the child would be an arbitrary asymmetry within "all event
types" (§2.9). Confirm this reading is correct before implementation.

### 7.3 Event-type-scope duplication is expected, not a bug

Per §2.9, a single call produces one `peer_events` row per lifecycle event
(`call_dialing`, `call_ringing`, `call_progressing`, `call_hangup`, etc.), all
sharing the same `reference_id` and (per §3's transfer-manager verification)
the same `peer_type`/`peer_target`. An address-set search will therefore
return multiple rows per underlying call/message. This is accepted as
consistent with `events`' own "store everything verbatim" philosophy, not
deduplicated at write time.

### 7.5 No existing address-keyed ClickHouse table to validate ORDER BY against (Round 1 MINOR)

`events` (the only existing ClickHouse table in this service) uses
`ORDER BY (event_type, timestamp)` — a type-scoped access pattern, not an
address-scoped one. There is no precedent in this codebase for the
`(customer_id, peer_type, peer_target, timestamp)` ordering proposed in §4.
The reasoning in §4 is sound on ClickHouse sparse-index mechanics generally,
but should be spot-checked with an `EXPLAIN`/query-plan pass once a
throwaway table is populated at implementation time, not just asserted from
first principles.

### 7.4 ClickHouse test-fixture convention

`scripts/database_scripts_test/` currently only contains a MySQL fixture
(`table_timeline_analyses.sql`) for the `analysishandler`'s MySQL-backed
table. There is no existing ClickHouse test-fixture example in this service
to follow for `peer_events`. Needs a quick check at implementation time of how
`dbhandler` ClickHouse tests are currently set up (likely an in-memory/mock
`conn`, per the `mock_main.go` pattern already in `pkg/dbhandler`), rather
than a new SQL fixture file.

## 8. Out of scope

- Any change to `contact_interactions`, `contact_addresses`,
  `contact_resolutions`, or their read API. Untouched.
- Any change to the `events` table schema or its existing insert path.
- A read/query API for `peer_events` (RPC + REST). Not requested yet; this
  design covers ingestion only. A follow-up ticket is needed before any
  consumer can actually query this table.
- Retroactive backfill. Same precedent as VOIP-1204 M2 and VOIP-1215: from
  cutover only, no replay of historical events.
- Widening beyond call-manager/conversation-manager to any other publisher.

## 9. Test plan

- Unit: `buildPeerEventRows` — table-driven per publisher (call, conversation,
  conversation_message), per direction (incoming/outgoing/unknown), asserting
  correct peer/local swap and that conversation parent events skip
  `DeriveEndpoints` entirely.
- Unit: `DeriveEndpoints` (moved to bin-common-handler) — same cases as the
  existing contact-manager test, moved/kept in sync.
- Unit: `PeerEventBatchInsert` dbhandler — mirrors `EventBatchInsert` tests
  (batch construction, ClickHouse connection nil-check, empty-batch no-op).
- Integration/manual: verify a flush cycle with a mixed batch (call events +
  conversation events + other-publisher events) produces the right row count
  in `peer_events` and leaves `events` row count unaffected.
- Regression: bin-contact-manager's existing `interaction_test.go` /
  `interaction_crm_eligibility_test.go` must still pass unchanged after the
  `deriveEndpoints` call site is repointed to the shared helper (behavior
  identical, only the import changes).
- Full verification workflow (`go mod tidy && go mod vendor && go generate
  ./... && go test ./... && golangci-lint run`) in bin-common-handler,
  bin-contact-manager, AND bin-timeline-manager (three services touched).
