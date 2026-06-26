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
is_primary    bool         one per type (generated-col UNIQUE)
tm_create, tm_update, tm_delete

unique(customer_id, type, target, tm_delete)   -- identifier dedup
```

- Holds **only re-identifiable permanent identifiers**. `web_session` is NOT an
  address (see §6) and is excluded.
- `tm_delete` is newly introduced here (the old child tables had only `tm_create`),
  unifying soft-delete and supporting the dedup unique.
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
- **No `tm_update` / `tm_delete` in v1.** create-only subscription means no
  in-place mutation and no soft-delete in the normal lifecycle. (`tm_delete` is
  added later only if/when deletion is required; see the forward-compatibility
  note above).

### 3.3 contact_resolutions (manual attribution, append-only)

Automatic attribution (peer→address match) cannot cover two real cases: a borrowed
phone, and an anonymous session later identified. Those need an explicit human (or
rule) judgment, which is itself a new immutable fact.

```
id                binary(16)   PK
customer_id       binary(16)
contact_id        binary(16)   the contact this interaction is attributed to
interaction_id    binary(16)   the interaction being attributed (single-row grain)
resolved_by_type  varchar      agent / system / rule
resolved_by_id    binary(16)   the deciding actor (nil for system)
tm_create         datetime     attribution time (immutable fact)
tm_update         datetime     (triple kept by convention; no real mutation)
tm_delete         datetime     soft delete = attribution retraction (correction)

index(customer_id, contact_id, tm_delete)
index(customer_id, interaction_id, tm_delete)
```

- Single-interaction grain. Because conversation/AIcall interactions are
  session-grained (one row per session, §4), "attribute this whole session" reduces
  to "attribute this one interaction", so no endpoint-grained variant is needed.
- Correction = `tm_delete` (retract) + new row (re-attribute). The interaction
  itself is never touched.

### 3.4 Entity relationships

```
contacts (existing)
   1 : N  contact_addresses        (contact owns permanent identifiers)
   1 : N  contact_interactions     (via read-time peer match, derived, not FK)
   1 : N  contact_resolutions      (manual attributions)

contact_interactions
   N : 1  contact_resolutions      (an interaction may be manually attributed)
```

threads (episodes) are NOT a table. They are computed at read time by sessionizing
the interaction stream (§5).

## 4. Projection handler (channel events → interactions)

- **create-only subscription.** Subscribe only to each manager's `*_created`
  event. `updated` / `ended` / `terminated` are NOT subscribed. This realizes pure
  append-only and dissolves LWW/sweeper/tombstone debt. A status transition after
  creation never rewrites the interaction; current state is read-time fetched.
- **Grain (channel-dependent, by design):**
  - call, AIcall → session grain (one `*_created` per session). AIcall lifecycle
    (`initiating → progressing → terminating → terminated`, idle timeout default
    24h) provides the natural session boundary.
  - SMS/LINE (conversation-manager) → message grain
    (`conversation_message_created` fires per message).
- **SMS fan-out:** message-manager stores one `message` with `Targets[]` (N
  recipients) and publishes `message_created` once with the full `Targets` payload.
  The handler expands `Targets` into N interactions (same `reference_id`, distinct
  `peer_target`). No `target_index` column is needed; `peer_target` itself
  distinguishes the fan-out rows.
- **direction → peer extraction:** outbound takes peer from the destination;
  inbound takes peer from the source. Per-channel detail specified at
  implementation.
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
3. manual match:
   interaction_ids FROM contact_resolutions WHERE contact_id = ? AND tm_delete IS NULL
4. union (2) ∪ (3), de-dup
```

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

## 6. web_session handling

`web_session` is the session id of a web-direct AIcall messaging conversation
(reference_type=conversation, web-originated). AIcall idle timeout fragments
sessions, so the same person gets a new `web_session` id on each visit. It is a
one-time handle, not a permanent identifier. Therefore:

- web_session is excluded from `contact_addresses`.
- Because a conversation/AIcall session is one interaction (§4 session grain),
  attributing a web_session reduces to a single-row `contact_resolutions`
  attribution. No endpoint-grained attribution machinery is required.

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
  customer_id,  tm_create,             tm_delete=NULL
emails        → contact_addresses:
  type='email', target=lower(address), is_primary, contact_id, customer_id,
  tm_create,    tm_delete=NULL
```

Decisions:

1. **tm_delete introduced** on contact_addresses (soft-delete unification + dedup
   unique).
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
2. M1 active-row criterion must stay consistent with the VOIP-1205 soft-delete
   decision (active = `tm_delete IS NULL`).
3. Normalization authority for non-tel/email types (sip/line/whatsapp): existing
   `ValidateTarget` validates but does not normalize for these; whatsapp currently
   errors in ValidateTarget. "Who normalizes" must be specified before those types
   flow into contact_addresses.
4. Interaction deletion (cascade on `customer_deleted`, GDPR/data-removal): not in
   v1. When required, add `tm_delete` to contact_interactions, extend the unique to
   `(reference_type, reference_id, peer_target, tm_delete)`, and implement deletion
   as a soft-delete tombstone (customer-scoped bulk tombstone for cascade), per the
   forward-compatibility note in §3.2.

## 10. Scope out (v1)

- Deals / pipelines / sales automation.
- Interaction notes / annotations (a mutable concern; future
  `contact_interaction_notes`, deliberately not folded into resolutions).
- Auto-merge of past anonymous sessions on identity confirmation (accepted
  limitation).
