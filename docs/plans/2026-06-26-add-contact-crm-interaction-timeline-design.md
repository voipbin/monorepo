# Contact-axis unified interaction timeline (lightweight CRM)

- Issue: VOIP-1204
- Related issue (split out): VOIP-1205 (contact-manager soft-delete filter + stale CLAUDE.md)
- Class: new read-model + schema migration (encapsulated in bin-contact-manager)
- Date: 2026-06-26

## 1. Problem Statement

VoIPBin records every channel event (Voice/SMS/Email/Chat/Social) inside its own
manager (call-manager, message-manager, email, conversation-manager, ai-manager),
keyed by that channel's own identifiers. There is no single place that answers the
question a contact-center operator actually asks:

> "Show me everything that happened with this one end customer, across all
> channels, in time order."

This is not a full CRM. Deals, pipelines, and sales automation are explicitly out
of scope for v1. The goal is a thin layer that unifies scattered per-channel
interactions onto a single end-customer (Contact) timeline.

### Terminology

`customer` in VoIPBin already means the tenant (account holder). The CRM end user
is called a **Contact** to avoid polluting the data model. The same word for two
concepts corrupts the schema quickly.

## 2. Unification axis: contact (address), not activeflow

Code-verified rejection of activeflow as the unifying axis:

1. SMS messages carry no `ActiveflowID` field.
2. Direct (non-flow) sends create no activeflow.
3. conversation creation does not create an activeflow.

So activeflow is a strict subset of customer interactions and cannot be the join
key. The common denominator that every channel does carry is the **peer address**
(the remote endpoint of the interaction). Therefore the axis is:

```
peer address  →  contact_addresses (identity mapping)  →  contact
```

## 3. Data Model (3 new tables)

All inside bin-contact-manager. VoIPBin conventions apply: `binary(16)` ids, no
physical FKs (logical references, integrity enforced by handlers/events), `_type`
suffix for type columns, `tm_create/tm_update/tm_delete` triples.

### 3.1 contact_addresses (permanent identifier mapping)

Merges the existing `contact_phone_numbers` + `contact_emails` child tables into a
single normalized address table.

```
id            binary(16)   PK
customer_id   binary(16)
contact_id    binary(16)   nullable (NULL = unresolved)
type          varchar      tel / email / sip / line / whatsapp ...
target        varchar      normalized identifier (join key)
target_name   varchar      display name (optional)
is_primary    bool         one per type
tm_create, tm_update

unique(customer_id, type, target)   -- identifier dedup (matches agent_addresses)
```

- Holds **only re-identifiable permanent identifiers**. `web_session` is NOT an
  address (see §6) and is excluded.
- **No `tm_delete`; addresses are hard-deleted.** This follows the existing
  `agent_addresses` convention (bin-dbscheme-manager, 2026-06-21), which solved the
  identical problem (JSON→child table, normalized address, by-address hot-path
  lookup): `agent_addresses` has columns `tm_create/tm_update` only, a
  `unique(customer_id, type, target)`, and removes rows on delete so the unique
  slot is freed immediately. contact_addresses mirrors this. Rationale: an address
  mapping ("this number belongs to this contact") is a correctable assertion, not
  an immutable fact, so a wrong/removed mapping is deleted rather than tombstoned.
  This is the opposite of contact_interactions, which IS an immutable fact and is
  append-only. Hard-delete also avoids the MySQL "NULL is distinct in UNIQUE" trap
  that a `tm_delete`-in-unique scheme creates (active rows would not actually be
  deduped).
- **is_primary uniqueness.** Because there is no soft-delete, "one primary per
  type" is enforced over active rows directly (all rows are active), via a
  generated-column UNIQUE on `(customer_id, type)` restricted to `is_primary=true`.
  No `tm_delete` interaction to reason about.
- Late-binding note: hard-deleting an address breaks automatic re-matching of past
  interactions to that address. This is not a loss, because `peer_target` is stored
  raw on each interaction; re-registering the address re-establishes the match at
  read time (§5.1).
- REST backward compatibility: `phone_numbers[]` / `emails[]` responses are kept as
  a reverse-projection from `contact_addresses`; `addresses[]` is also exposed so
  sip/line/whatsapp are not hidden.

### 3.2 contact_interactions (immutable append-only fact log)

```
id              binary(16)   PK
customer_id     binary(16)
direction       varchar      inbound / outbound
peer_type       varchar      raw remote endpoint type  (match key)
peer_target     varchar      raw remote endpoint, normalized  (match key)
reference_type  varchar      origin channel (= channel discriminator)
reference_id    binary(16)   origin record id (state/body fetched at read time)
tm_interaction  datetime     origin event time (display sort / sessionize basis)
tm_create       datetime     projection insert time (pagination cursor)

unique(reference_type, reference_id, peer_target)
```

This is the only table with no `tm_delete`: create-only projection means no
soft-delete in the normal lifecycle, so the dedup discriminator `tm_delete` that
other tables carry is not applicable here. The unique key is the bare
`(reference_type, reference_id, peer_target)`.

**Forward-compatibility for deletion.** Deletion will eventually be required
(contact-manager already subscribes to `customer_deleted` for cascade cleanup; GDPR
/ data-removal requests also apply). When that lands, the path is expand-contract:
add a `tm_delete` column and extend the unique to
`(reference_type, reference_id, peer_target, tm_delete)` so a tombstoned
(ref, ref, peer) can be re-created. The deletion stays a **soft-delete (tombstone),
not a physical delete**, preserving the append-only spirit; cascade is a
customer-scoped bulk tombstone. Keeping the v1 unique as the bare triple is the
forward-compatible choice that lets this extension be a clean expand later.

Design decisions baked in:

- **`peer_*` is a raw fact, not identity.** `(peer_type, peer_target)` is the
  remote endpoint as the event carried it. Identity ("who is this") is computed at
  read time by matching against `contact_addresses`, never stored here. See §5.
- **No `contact_id`, no `address_id`, no `thread_id` columns.** All derived;
  computed at read time. Storing them would resurrect backfill/sweeper/stale
  debt.
- **No `channel_type` column.** Derived from `reference_type`; not stored.
- **No `status` / `preview` snapshot.** create-only projection (see §4) would
  freeze a stale value ("ended call shown as progressing"). Current state/body is
  fetched at read time via `(reference_type, reference_id)`.
- **No `tm_update` / `tm_delete` in v1.** Subscribing only to the creation signal
  (§4) means no in-place mutation and no soft-delete in the normal lifecycle.
  (`tm_delete` is added later only if/when deletion is required; see the
  forward-compatibility note above).

### 3.3 contact_resolutions (manual attribution, append-only)

Automatic attribution (peer→address match) cannot cover three real cases: a
borrowed phone, an anonymous session later identified, and a wrong automatic match
that must be suppressed. Those need an explicit human (or rule) judgment, which is
itself a new immutable fact.

```
id                binary(16)   PK
customer_id       binary(16)
contact_id        binary(16)   the contact this interaction is attributed to
interaction_id    binary(16)   the interaction being attributed (single-row grain)
resolution_type   varchar      positive (attach) / negative (suppress)
resolved_by_type  varchar      agent / system / rule
resolved_by_id    binary(16)   the deciding actor (nil for system)
tm_create         datetime     attribution time (immutable fact)
tm_update         datetime     (triple kept by convention; no real mutation)
tm_delete         datetime     soft delete = attribution retraction (correction)

index(customer_id, contact_id, tm_delete)
index(customer_id, interaction_id, tm_delete)
```

- **`resolution_type` gives the layer both polarity directions.** A positive row
  attaches an interaction to a contact that automatic matching missed (borrowed
  phone attributed to the borrower, anonymous session attributed once identified).
  A negative row suppresses an interaction that automatic matching wrongly attached
  (the borrowed-phone call automatically matches the phone *owner* via peer-IN;
  a negative row for that owner removes it from the owner's timeline). Without the
  negative direction, resolution could only add, never correct an over-match, so
  the borrowed-phone case would stay permanently mis-attributed to the owner.
- Single-interaction grain. Each resolution row attributes/suppresses exactly one
  interaction. Session-grained channels (call, aicall) are one interaction per
  session, so one row covers the session; message-grained channels (SMS, LINE) are
  one interaction per message (§4), so attributing a whole thread is N rows. The
  grain is always one interaction, not "one session".
- Correction = `tm_delete` (retract) + new row (re-attribute). The interaction
  itself is never touched.

### 3.4 Entity relationships

```
contacts (existing)
   1 : N  contact_addresses        (contact owns permanent identifiers)
   1 : N  contact_interactions     (via read-time peer match, derived, not FK)
   1 : N  contact_resolutions      (manual attributions)

contact_interactions
   1 : N  contact_resolutions      (an interaction accumulates attribution rows:
                                     retract via tm_delete + re-attribute appends)
```

threads (episodes) are NOT a table. They are computed at read time by sessionizing
the interaction stream (§5).

## 4. Projection handler (channel events → interactions)

- **Subscribe to each channel's creation signal only (no state/lifecycle
  follow-ups).** The interaction is appended once, when the origin record is born;
  later state transitions (call ended, AIcall terminated) never rewrite it. This
  realizes pure append-only and dissolves LWW/sweeper/tombstone debt. The exact
  subscribed event differs per manager (not every manager emits a literal
  `*_created`):

  | reference_type | subscribed event              | grain          | reference_id            | peer source            |
  |----------------|-------------------------------|----------------|-------------------------|------------------------|
  | call           | call create event             | session (1)    | call_id                 | call peer              |
  | message (SMS)  | `message_created`             | message / recipient | message_id (fan-out) | each `Targets[]` dest  |
  | conversation (LINE) | `conversation_message_created` | message    | conversation_message_id | message peer           |
  | aicall (web)   | `aicall_status_initializing`  | session (1)    | aicall_id               | web_session            |

- **AIcall has no `aicall_created` event.** It emits per-message
  (`aimessage_created`) and lifecycle state (`aicall_status_initializing → ... →
  terminated`) events. To keep the session grain agreed in §6 (one web AIcall
  session = one interaction), the projection subscribes to the **first state event
  (`aicall_status_initializing`)** as the session-creation signal and does NOT
  subscribe to `aimessage_created`. Individual AIcall messages are fetched at read
  time via the reference (e.g. `get_aicall_messages`), consistent with the
  "body fetched at read time" rule. This also keeps `reference_id = aicall_id`
  unique per session, so the idempotency unique never collides across the session's
  messages.
- **Grain is defined by the subscribed event, not by the channel abstractly.**
  call and aicall subscribe to a once-per-session event, so they are session-grained
  (one interaction per session). SMS and LINE subscribe to a per-message event, so
  they are message-grained. A long SMS/LINE thread renders as many interactions;
  visual grouping is read-time sessionize (§5).
- **SMS fan-out:** message-manager stores one `message` with `Targets[]` (N
  recipients) and publishes `message_created` once with the full `Targets` payload.
  The handler expands `Targets` into N interactions (same `reference_id`, distinct
  `peer_target`). No `target_index` column is needed; `peer_target` itself
  distinguishes the fan-out rows.
- **direction → peer extraction:** outbound takes peer from the destination;
  inbound takes peer from the source. Per-channel detail specified at
  implementation. Multi-party cases (conference, duplicate recipients in
  `Targets[]`) are noted as an open item (§9).
- **Idempotency (at-least-once):** `unique(reference_type, reference_id,
  peer_target)` (see §3.2). `(reference_type, reference_id)` alone collides on
  fan-out.

## 5. Read API

The storage was made thin and stateless; the cost moved to the read path. The
server emits facts in order; identity and grouping are computed at read time or by
the consumer.

### 5.1 Endpoints

```
GET /v1/interactions?peer_type=&peer_target=   # natural key, direct
GET /v1/interactions?contact_id={id}           # expand address set, IN-match ∪ resolutions
GET /v1/interactions?address_id={id}           # single address only
GET /v1/interactions/unresolved?since=7d        # unresolved queue (web_session excluded)
```

`?contact_id=` resolution at read time:

```
1. expand the contact's address set:
   SELECT type, target FROM contact_addresses WHERE contact_id = ?
2. automatic match:
   SELECT * FROM contact_interactions
   WHERE (peer_type, peer_target) IN (address set)
3. manual attribution (active rows only):
   SELECT interaction_id, resolution_type FROM contact_resolutions
   WHERE contact_id = ? AND tm_delete IS NULL
4. combine:
   ( automatic (2) ∪ positive manual (3) )  MINUS  negative manual (3)
   de-dup by interaction id
```

contact_addresses has no soft-delete (§3.1), so step 1 needs no `tm_delete` filter.
contact_resolutions IS append-only with soft-delete retraction, so step 3 keeps
`AND tm_delete IS NULL`.

**Precedence is a fixed decision, not LWW.** When an active positive and an active
negative exist for the same `(contact_id, interaction_id)`, **negative always
wins** (the set-MINUS in step 4 is the authoritative semantics). Implementations
MUST NOT resolve this by "latest `tm_create` polarity wins" (LWW), which would
diverge from the set-MINUS and make the same input produce different reads. Negative
is a hard suppression of a wrong attribution; it is not time-ordered against
positive.

The negative set (step 3 `resolution_type='negative'`) suppresses interactions
that automatic matching wrongly attached to this contact (e.g. a borrowed-phone
call that peer-matches the phone owner). This is what lets a human override an
over-match; positive-only attribution could never remove a wrong automatic row.

### 5.2 Response shape: pure flat

- Server returns a flat `interactions[]` array. No thread nesting.
- Server responsibility: union + sort + page slice. No sessionize.
- Consumer responsibility: sessionize (visual grouping by time gap) + display
  sort.
- Rejected: grouped/nested response. A thread crossing a page boundary collides
  with pagination (sessionize is sequentially dependent; a thread of N interactions
  cannot be cleanly sliced into fixed-size pages). flat keeps interactions atomic,
  so any slice point is safe.
- Cross-consumer consistency: the gap threshold is pinned as a documented SDK
  constant so clients sessionize identically (not code-enforced in v1).

### 5.3 Pagination

- Cursor sort key = `interaction.tm_create DESC` (+ `id` as tie-breaker).
  `tm_create` (insert time) is near-monotonic, so no past-dated row inserts into an
  already-served page (no infinite-scroll gap). `tm_interaction` (event time) would
  not be safe as a cursor because a late-attached past interaction can appear with
  an old `tm_interaction`.
- Display order (`tm_interaction`, event order) is re-sorted by the consumer
  (convention: "mutable secondary sort lives in the consumer layer").
- A resolution-attached interaction still sorts by `interaction.tm_create`;
  resolution acts only as a visibility filter, never participates in sort, so the
  same interaction lands in the same position regardless of attribution path
  (deterministic).
- **Late-binding visibility caveat (both directions).** The cursor is gap-free for
  the *projection insert* path (new rows always carry a fresh `tm_create` at the
  head). It is NOT gap-free for late-binding edits, which move rows in BOTH
  directions within an already-served page:
  - appearance: registering an address or adding a positive resolution makes a
    past interaction (old `tm_create`) newly visible in a `?contact_id=` result.
  - disappearance: adding a negative resolution, or retracting (`tm_delete`) a
    positive one, removes a previously visible row.
  Either way the change can land in a page already served during an in-progress
  infinite scroll, so a single scroll session may miss or retain a row until a full
  refetch. This is a transient per-session inconsistency, not data loss (a fresh
  query always returns the correct set). Accepted for v1; if it bites in dogfood,
  the client refetches the head on a "contact identity changed" signal.

### 5.4 Unresolved queue and negative-suppressed orphans

`GET /v1/interactions/unresolved` must surface two distinct populations, or a
negative resolution would create an invisible orphan:

1. **Never matched.** No `contact_addresses` row matches `(peer_type, peer_target)`
   and no positive resolution exists. (web_session is excluded, per §6.)
2. **Negatively suppressed without a positive home.** An interaction whose
   automatic peer-match was removed by a negative resolution for that contact, and
   which has no active positive resolution to any other contact. Without this
   branch, the borrowed-phone case becomes invisible: the call is suppressed from
   the owner's timeline (negative) but, until the borrower is identified with a
   positive resolution, it belongs to no one. The unresolved queue is exactly where
   such an interaction should wait for a human to assign its real owner.

So the queue predicate is: an interaction is unresolved if, after applying all
active resolutions, it resolves to **zero contacts** (no surviving automatic match
and no active positive). A negative resolution that strips the only (automatic)
attribution returns the interaction to this queue. This closes the orphan path that
polarity would otherwise open.

## 6. web_session handling

`web_session` is the session id of a web-direct AIcall messaging conversation
(reference_type=aicall, web-originated). AIcall idle timeout fragments sessions, so
the same person gets a new `web_session` id on each visit. It is a one-time handle,
not a permanent identifier. Therefore:

- web_session is excluded from `contact_addresses`.
- The projection subscribes to `aicall_status_initializing` (one per session, §4),
  so a web AIcall session is exactly one interaction with `peer_target=web_session`.
  Attributing that session reduces to a single-row `contact_resolutions` row. No
  endpoint-grained attribution machinery is required.
- Note: an AIcall whose `reference_type` routes its messages through
  conversation-manager could in principle also surface as conversation interactions.
  Pinning the canonical reference for an interaction (so the same logical event is
  not double-counted across aicall and conversation) is the same concern as SMS
  double-counting and is tracked as an open item (§9).

## 7. Migration

### 7.1 M1: contact_phone_numbers + contact_emails → contact_addresses

Code+DB verified (production, GKE `bin_manager`):

- Existing structure is already-normalized child tables (NOT a JSON embed), so no
  JSON-to-child work is required.
- Scale: phone 781, email 1022, distinct e164 86, orphan rows 0 (integrity good).
  Under 2k rows total (no chunking, no downtime needed); a single transactional
  `INSERT ... SELECT`.

Mapping:

```
phone_numbers → contact_addresses:
  type='tel',   target=number_e164,    target_name=NULL, is_primary, contact_id,
  customer_id,  tm_create
emails        → contact_addresses:
  type='email', target=lower(address), is_primary, contact_id, customer_id,
  tm_create
```

Decisions:

1. **No tm_delete on contact_addresses; hard-delete** (matches agent_addresses,
   §3.1). Addresses are a correctable mapping, not an immutable fact.
2. **2 unnormalizable rows deleted** before migration. Both had non-numeric raw
   `number` (`+15550009c24`, `+155****4567`, alphabet/asterisk, no recoverable
   E.164, dummy/test data) and empty `number_e164`. Production DELETE was done
   id-scoped with before/after count verification (783 → 781, bad 0). Remaining
   data is 100% normalizable; the migration needs no exception branch.
3. **Big-bang cutover** (replace, not parallel). phone/email tables retired,
   contact_addresses becomes the single source.

Code rewire (the real weight of M1, all in contact-manager dbhandler):

- caller-ID lookup → `contact_addresses` query.
- Redis cache keys `(customer_id, phone_e164)` / `(customer_id, email)` →
  unified `(customer_id, type, target)`.
- REST `phone_numbers[]` / `emails[]` kept as reverse-projection from
  contact_addresses.

Big-bang risk: caller-ID lookup is the inbound-call hot path. Ordering must be
**data migration → verify → code deploy** within the same deploy window. A rollback
trigger (caller-ID match-failure rate spike) is recorded in the deploy notes.

### 7.2 M2: interaction backfill (NOT done)

- From cutover, interactions accumulate via projection only. No retroactive load.
- Rationale: past events already exist as origin records in each manager.
  Retroactive replay would resurrect idempotency / past-normalization-divergence /
  deleted-origin debt, exactly the debt this design avoids. In the dogfood stage,
  past history has low value relative to the forward-accumulating unified timeline.
- Accepted limitation: the timeline is empty for history at the moment CRM is
  switched on. If past history proves necessary, a separate backfill ticket (small
  data set, runnable in one pass) is opened.

## 8. Guiding principle

Store immutable facts; compute derived values at read time. This single rule drove
every reduction: `channel_type`, `address_id`, `contact_id`, `thread`,
`status/preview` were all removed as derived, and the model kept getting thinner.
interaction = "what happened" (immutable); resolution = "who it was" (append-only
judgment). Separating the event from its interpretation means a wrong
interpretation never damages the event.

## 9. Open items

1. Sessionize gap thresholds per channel (tuned during dogfood on real data;
   cross-channel uses the previous interaction's channel as the gap basis).
2. contact_addresses is hard-deleted (no tm_delete, §3.1), so the M1 active-row
   question applies to the parent `contact_contacts` rows, not addresses: the
   migration must only carry addresses whose owning contact is active, consistent
   with the VOIP-1205 soft-delete decision (active contact = `tm_delete IS NULL`).
3. Normalization authority for non-tel/email types (sip/line/whatsapp): existing
   `ValidateTarget` validates but does not normalize for these; whatsapp currently
   errors in ValidateTarget. "Who normalizes" must be specified before those types
   flow into contact_addresses. This is also a **matching-key invariant**:
   late-binding works only if the projection normalizes `peer_target` with the same
   function that address registration normalizes `target`. If the two diverge by
   even one byte, the IN-match silently fails and the interaction stays unresolved
   forever (a missed match, not a wrong one). The single shared normalization
   authority must be pinned before those types ship.
4. Interaction deletion (cascade on `customer_deleted`, GDPR/data-removal): not in
   v1. When required, add `tm_delete` to contact_interactions, extend the unique to
   `(reference_type, reference_id, peer_target, tm_delete)`, and implement deletion
   as a soft-delete tombstone (customer-scoped bulk tombstone for cascade), per the
   forward-compatibility note in §3.2.
5. Canonical reference pinning (double-count avoidance): a single logical event
   that surfaces in two managers (SMS via message + conversation; web AIcall via
   aicall + conversation) must be pinned to one canonical `reference_type` so it
   produces one interaction, not two. Resolve which manager is authoritative per
   overlap before projection ships.
6. Multi-party / duplicate-peer grain: conference or multi-leg calls (one session,
   many peers) and duplicate recipients in `Targets[]` (same peer, one unique slot)
   can drop peers under the current single-peer-per-row + unique model. Specify
   whether multi-party produces one row per peer and how duplicate peers are
   de-duplicated, before those flows are projected.

## 10. Scope out (v1)

- Deals / pipelines / sales automation.
- Interaction notes / annotations (a mutable concern; future
  `contact_interaction_notes`, deliberately not folded into resolutions).
- Auto-merge of past anonymous sessions on identity confirmation (accepted
  limitation).
