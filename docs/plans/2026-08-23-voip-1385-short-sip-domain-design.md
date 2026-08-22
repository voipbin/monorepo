# VOIP-1385: Short customer SIP domain — design (rev 2, full-cutover)

Date: 2026-08-23 (rev 2, same day)
Ticket: VOIP-1385 (Support large SIP INVITE over UDP)
Status: rev 2 for review. Rev 1 (option comparison, phased opt-in rollout)
passed its design review loop and was then superseded by CEO/CTO decisions;
the option analysis is retained in section 3 as the decision record.

## 0. Final decisions (CEO/CTO, 2026-08-23)

1. Option B (canonical short domain) — approved.
2. FULL immediate cutover: existing customers' DB rows migrate uuid -> short
   label in one batch. No opt-in phase, no transition deadline.
3. Mapping owned by a NEW table `registrar_customer_domains` in
   bin-registrar-manager.
4. Suffix also changes: `registrar.voipbin.net` -> `reg.voipbin.net`,
   bundled into the same cutover (single device-reconfiguration event).
5. Label: 4-char random base36 (lowercase), auto-generated. Customer-chosen
   vanity labels: explicitly OUT of scope (future candidate).

Resulting domain: `<4 chars>.reg.voipbin.net` = 20 chars (vs 58 today).
Byte effect on the incident INVITE: 38 chars x 5 occurrences = 190B;
1,556B -> ~1,366B wire (~134B headroom under MTU 1,500).

## 1. Background

Customer extension SIP domains are `<customer-uuid>.registrar.voipbin.net` (58
chars). This string appears 5 times in an authenticated INVITE (R-URI, From, To,
digest realm, digest uri), contributing ~290 bytes. A live incident proved that
an authenticated INVITE at 1,556 bytes (wire) exceeds MTU 1,500, gets
IP-fragmented, and the fragments are dropped before reaching our server
(verified: no fragment arrives at bm-nyc-01's NIC; single-packet 1,320B messages
pass). Result: digest auth over UDP fails for affected clients.

Goal (CEO/CTO directive): keep supporting UDP. Shrink our contribution to the
message size so typical INVITEs stay under MTU. Shortening the customer domain
is the only server-side lever with meaningful savings.

## 2. Dependency audit (unchanged from rev 1, all claims code-verified)

### 2.1 Forward generation (single source)
`bin-registrar-manager/models/common/common.go` — `GenerateRealmExtension`
(`<customerID>.<DOMAIN_NAME_EXTENSION>`) and `GenerateEndpointExtension`
(`<ext>@<realm>`). Called ONLY inside bin-registrar-manager:
- extensionhandler/extension.go:55,58 — realm and aorID at extension create
- contacthandler/contact.go:18,36 — rebuilds endpoint string for `ps_contacts`
  lookups and Redis contact-cache keys

### 2.2 Reverse parsing — 3 sites, all in bin-call-manager
1. `pkg/callhandler/start_incoming_domain_type_registrar.go:30` —
   `customerID = uuid.FromStringOrNil(TrimSuffix(domain, ".registrar.<base>"))`
   where domain = Kamailio's `VB-Domain` header via Stasis; drives ALL
   downstream authorization for incoming registrar calls.
2. `models/common/domain.go:22` `ParseSIPURI` + caller
   `pkg/arieventhandler/ari_contact.go:23` — parses `<ext>@<uuid>.registrar...`
   from ARI ContactStatusChange for contact-cache refresh; silently yields
   uuid.Nil on non-uuid labels.
3. `pkg/callhandler/start_incoming_domain_type_trunk.go:37` — trunk analogue,
   ALREADY a lookup (`RegistrarV1TrunkGetByDomainName`); the pattern to adopt.

Suffix-only classification (`getDomainTypeIncomingCall`, start.go:424) uses
hardcoded `".registrar." + PROJECT_BASE_DOMAIN` / `".trunk." + ...` suffixes
(`bin-call-manager/pkg/projectconfig/main.go:56-57`) — the suffix change
touches this code (section 6).

### 2.3 Asterisk identity coupling
`ps_endpoints.id/aors/auth`, `ps_aors.id`, `ps_auths.id` all share
`<ext>@<uuid-domain>`; `ps_auths.realm` = `<uuid-domain>`. Asterisk registrar
runs realtime with `identify_by` unset (default `username,ip`): the From
`user@domain` IS the endpoint identity for REGISTER. `ps_domain_aliases` is
wired in sorcery.conf but unpopulated/unused. No code path rewrites ps_* ids
or realm after creation; a domain change requires deleting and recreating the
ps_* rows — the migration batch does exactly that at the ps_*/sipauth level
while PRESERVING the extension entity itself (see section 7; the extension
Delete+Create handlers are deliberately NOT used).

### 2.4 Digest auth coupling
Kamailio challenges with realm = `$fd` and verifies via `auth_check` against
`registrar_sip_auths` (`domain_column=realm`, `calculate_ha1=1`, plaintext
password, HA1 computed at runtime). The realm string is cryptographically a
free variable, but `registrar_sip_auths.realm` must byte-match the From
domain the client uses. REGISTER is NOT authenticated at Kamailio (auth runs
only for new INVITEs, kamailio.cfg:1082); Asterisk performs REGISTER digest
against `ps_auths`. Two credential stores, both provisioned by
registrar-manager with the same realm — the single generation point keeps
them consistent.

### 2.5 Client-visible contract
`Extension.DomainName` documented (bin-openapi-manager/openapi/openapi.yaml:6515)
as "same as the customer_id"; code returns the bare uuid. RST docs already
show the FULL domain (extension_tutorial.rst:35, extension_struct_extension.rst)
— pre-existing inconsistency, settled in section 5 (full realm wins).

### 2.5b Frontends construct the domain client-side (hard prerequisite)
- `monorepo-javascript/square-talk/src/config.js:13-22` builds the WSS URL and
  SIP URI from the customer uuid (used in webphone.js:170-171).
- `monorepo-javascript/square-admin/src/views/extensions/extensions_detail.js:209`
  builds the customer-facing "Registration Address" the same way (only the
  Domain Name display at :196 follows the API).
Both must consume the API's `domain_name` verbatim BEFORE any format change.
Other constructors (cosmetic/configurable): monorepo-monitoring
sip-validator/sip_config.py:14, sandbox/scripts/setup_test_customer.sh:210.

### 2.6 TLS / DNS
Wildcard cert and wildcard A record match exactly one label. The suffix
change to `reg.voipbin.net` therefore needs a NEW wildcard cert
(`*.reg.voipbin.net`) and a new wildcard A record — section 6.

## 3. Decision record: option comparison (rev 1 summary)

Option A (alias + Kamailio edge normalization: short domain rewritten to
canonical uuid at ingress) was REJECTED: it concentrates risk in SIP
dialog-state From/To rewriting (uac_replace_from restore logic) on the
production hot path, adds a permanent dual-domain system and an ingress Redis
dependency, keeps the long realm in Proxy-Authorization (smaller savings),
and leaves the existing reverse-parse fragility untouched. Option B
(canonical short domain) was chosen: zero SIP protocol risk, replaces the
silent-uuid.Nil parsing debt with an explicit fail-closed lookup, maximum
savings. Rev 1's phased opt-in rollout was then superseded by the
full-cutover decision (section 0), which REMOVES the need for opt-in
regeneration machinery entirely.

## 4. New table: `registrar_customer_domains` (bin-registrar-manager)

| column | type | notes |
|---|---|---|
| customer_id | binary(16) | PK |
| domain_label | varchar(64) | unique index; 4-char base36 generated |
| realm | varchar(255) | unique index; `<label>.reg.<base>` full string |
| tm_create, tm_update | datetime(6) | standard |

Deviation note: no `tm_delete` (hard delete on `customer_deleted`) — a pure
mapping row has no soft-delete consumer; deliberate deviation from sibling
registrar tables.

- Owner: bin-registrar-manager (realm generation already lives there).
  Rejected alternatives: a column on customer records (leaks SIP concerns
  into bin-customer-manager, adds cross-service RPC on hot lookups);
  deriving from `registrar_extensions.realm` rows (mapping must exist before
  the first extension; label must survive delete-all-extensions).
- Lifecycle: rows created on `customer_created` (registrar-manager already
  subscribes to customer-manager events for `customer_deleted` cascade —
  same subscription gains creation), deleted on `customer_deleted`.
  Lazy-create fallback at first extension creation for robustness.
- Label generation: 4-char base36 lowercase, crypto-random, retry on unique
  violation (36^4 = 1,679,616 space — see section 9 for the enumeration
  trade-off, accepted by decision 5). Reserved labels excluded — with
  exactly-4-char labels only `pstn` and `echo` can genuinely collide; the
  full list (`pstn`, `sip`, `echo`, `reg`, `www`, `api`) is kept as cheap
  future-proofing for any later length change.
- `GenerateRealmExtension(customerID)` changes from a pure format function
  to a table lookup. Since `models/common` must stay DB-free (monorepo
  convention), the function MOVES to the handler layer with a ctx+error
  signature; `GenerateEndpointExtension` follows. Both call sites
  (extensionhandler, contacthandler) already hold dbBin. Trunk generation
  unchanged.
- The realm->customer lookup RPC (`RegistrarV1CustomerGetByRealm`) is backed
  by THIS table's unique realm index (supersedes rev 1's
  registrar_extensions-backed lookup): exactly one row per customer, exists
  even with zero extensions. Result cached in Redis (realm -> customer_id),
  invalidated on migration/regeneration.
- Migration prep backfill: one row per existing customer with the CURRENT
  uuid realm (pure additive, zero risk). After code cutover reads this table,
  the batch (section 7) updates each row to the short realm.

## 5. Code changes

0. Frontend (PREREQUISITE — deploy order in section 8):
   - square-talk: consume API `domain_name` verbatim; the WSS URL becomes
     `wss://<domain_name>`. Delete the `registrar.` constructors in
     src/config/hostname.js:35,41 (the actual literal source; config.js and
     webphone.js consume them). Scope note: this is a small API-consumption
     refactor (plumb the extension's domain_name into webphone.js), not a
     one-line constant edit — config.js currently receives only customerId.
   - square-admin: same for the Registration Address
     (extensions_detail.js:209) and its hostname helper
     (src/config/hostname.js:35).
1. bin-registrar-manager:
   - New table (section 4), event-driven lifecycle, label generator.
   - `GenerateRealmExtension` -> table lookup.
   - New RPC `RegistrarV1CustomerGetByRealm` backed by the table.
   - Migration batch command in registrar-control (section 7).
   - Invariant honored: `Extension.Realm`/`Trunk.Realm` column semantics
     unchanged (the literal string Kamailio byte-matches; "DO NOT CHANGE"
     comments at extension.go:28 / trunk.go:21); only the generated value
     changes.
2. bin-call-manager — replace all 3 reverse-parse sites with the lookup:
   - start_incoming_domain_type_registrar.go:30 -> lookup by realm; unknown
     realm rejects the call (fail closed; improves on today's silent
     uuid.Nil).
   - ParseSIPURI + ari_contact.go -> same lookup.
   - projectconfig suffix constants: `DomainRegistrarSuffix` accepts BOTH
     `".reg."+base` (primary) and `".registrar."+base` (legacy, kept for the
     migration window and rollback; removable later).
3. bin-dbscheme-manager: Alembic migration adding `registrar_customer_domains`
   (schema only; the data batch is application-level, section 7).
4. API/docs — SETTLED: `Extension.DomainName` = FULL realm for ALL customers.
   Touch points: openapi.yaml:6515 description (+ delete the stale "used by
   the kamailio's INVITE validation" clause — Kamailio never reads
   domain_name; same stale comment in models/extension/webhook.go:19),
   bin-openapi-manager gens regeneration (gens/models/gen.go embeds the
   description; bin-api-manager via gens/openapi_redoc/api.html), RST files
   (extension_overview/tutorial/struct, quickstart_extension,
   trunk_overview_domain_name, direct_hash_overview, self_hosting_envvars),
   SDK regeneration (voipbin-go, python-sdk, vn CLI, MCP — plain string type,
   non-breaking) + release notes. api-validator verified no-impact
   (tests/models/extension.py:12, no format assertion). NULL-realm rule:
   legacy pre-Feb-2024 rows with NULL realm are covered by the migration
   batch itself (customer domain row + re-provisioned realm), so no serving
   fallback is needed post-cutover; during the pre-batch window DomainName
   falls back to the computed legacy value for NULL realm.

## 6. Suffix change: `registrar.voipbin.net` -> `reg.voipbin.net`

Bundled into the same cutover so customers reconfigure devices exactly once.

1. DNS: add wildcard A `*.reg.voipbin.net` -> 199.127.61.42 (Cloudflare).
2. TLS: issue wildcard cert `*.reg.voipbin.net` via the existing certbot
   DNS-01 path; deliver via Komodo Variables (VOIP-1375 mechanism); add a
   server_name profile in Kamailio tls.cfg.
3. Kamailio template: `validate_authentication` gate regex accepts BOTH
   `.reg.BASE_DOMAIN` and `.registrar.BASE_DOMAIN` (legacy acceptance kept
   as rollback safety net; removal is a later cleanup).
4. registrar-manager env: `DOMAIN_NAME_EXTENSION=reg.voipbin.net` (config
   only — generation code reads the base from env).
5. call-manager: dual-suffix acceptance (section 5 item 2).
6. Old `*.registrar.voipbin.net` DNS + cert + acceptance stay live as the
   rollback safety net until cleanup. During the dual window tls.cfg's
   `[server:default]` remains the legacy cert: a non-SNI SIP-TLS client
   connecting to `<label>.reg.voipbin.net` would be served the legacy cert
   until cleanup flips the default (browsers/WSS always send SNI; only rare
   non-SNI native TLS clients are affected).
7. Trunk suffix (`.trunk.`) unchanged.

`sip.` was considered as the suffix (semantically broader) and rejected:
`sip.voipbin.net` is an existing service domain matched exactly in
`getDomainTypeIncomingCall`, and overloading it invites classification
confusion. `reg.` has no collision.

## 7. Migration batch (existing customers, uuid -> short, immediate)

- Implemented as a registrar-control command in bin-registrar-manager,
  NOT raw SQL (registrar-manager mirrors every ps_* write into Redis; raw
  SQL would leave stale mirrors).
- CRITICAL: the batch does NOT reuse the extension Delete+Create handlers.
  Those would (a) generate a NEW extension UUID (Create calls UUIDCreate),
  breaking every persisted reference to the extension id — agent address
  bindings (bin-agent-manager agent.go:398 validates stored extension uuids),
  flow actions targeting extensions (call-manager
  parseAddressTypeExtension), and direct hashes (Delete removes the direct
  hash, Create mints a new one -> every `sip:direct.<hash>@...` URI silently
  changes); (b) fire extension_deleted/extension_created webhook storms and
  re-run billing limit checks; (c) lose the extension if interrupted
  between delete and create.
- Instead: a dedicated RE-PROVISION path that PRESERVES extension id,
  password, and direct hash. Per extension: UPDATE `registrar_extensions`
  in place (realm, domain_name, aor_id/auth_id/endpoint_id strings);
  delete+create ONLY the `ps_endpoints`/`ps_aors`/`ps_auths` rows and the
  `registrar_sip_auths` row via the existing dbhandler methods (these
  maintain the Redis mirrors); publish a single extension_updated event
  (no deleted/created lifecycle events, no billing re-check).
- Per customer: generate label + upsert `registrar_customer_domains` row ->
  re-provision every extension (above) -> purge stale `ps_contacts` rows for
  the old endpoint strings (NEW dbhandler method: row delete + contact-cache
  invalidation — no such method exists today) -> invalidate the
  realm->customer Redis cache entries. (Mirror scope note: the ps_*
  dbhandler methods maintain Redis mirrors; `registrar_sip_auths` has no
  Redis mirror — Kamailio reads it from MySQL directly — so no sipauth
  cache work exists.)
- The stored `registrar_extensions.domain_name` column is rewritten to the
  FULL new realm by the batch (and the Create path writes the full realm for
  new extensions), keeping the stored column, the API value, and the RST
  docs consistent — no compute-on-read needed post-cutover.
- Rollback: the batch logs old_realm <-> new_realm per customer; a reverse
  run re-provisions back to the uuid realm. `registrar_customer_domains`
  keeps only the current value; the log is the rollback source.
- Effect on live devices: external SIP devices registered under the uuid
  domain lose registration at their customer's migration moment and need
  manual reconfiguration (portal notice + guide ship with the cutover).
  square-talk webphones SELF-HEAL: after the step-2 frontend fix they fetch
  the domain from the API on load, so a reload re-registers correctly.
- Runbook notes: run in a low-traffic window (each extension has a
  sub-second ps_* delete+create gap for incoming INVITEs, subsumed by the
  accepted outage decision). During the window Asterisk may emit
  ContactStatusChange events carrying OLD-realm endpoint strings for
  just-migrated customers; the fail-closed lookup rejects them with a log
  line — expected noise, not an alarm condition (the batch purges stale
  ps_contacts anyway).
- Execution: Claude authors the batch + Alembic schema migration; the batch
  run against production is executed by the CEO/CTO (same policy as Alembic
  upgrades).

## 8. Deploy order (hard gates)

1. Backend code — internally sequenced (the call-manager cutover fails
   CLOSED on unknown realms, so its prerequisites are hard sub-gates):
   1a. Alembic: create `registrar_customer_domains`.
   1b. registrar-manager deploy: lookup RPC + event lifecycle +
       generation-from-table (+ pre-batch NULL-realm serving fallback).
   1c. Backfill current realms; reconcile row count against customer count
       (100% coverage required before 1d).
   1d. call-manager deploy: lookup cutover (TrimSuffix deleted; unknown
       realm rejects) + dual-suffix acceptance.
   (Everything still uuid-domain; zero behavior change for clients — both
   uuid and future short realms resolve through the same lookup.)
2. Frontend: square-talk + square-admin consume `domain_name` verbatim.
   (Gate: MUST follow step 1 — deploying frontends first would break all
   existing webphones, which would build `wss://<bare-uuid>`.)
3. Infra: DNS wildcard + `*.reg.voipbin.net` cert + Kamailio tls.cfg/regex
   deploy (accepting both suffixes; no traffic on `.reg.` yet).
4. registrar-manager env flip: new provisioning uses `reg.voipbin.net`.
5. Migration batch run (CEO/CTO executes): all existing customers ->
   `<4ch>.reg.voipbin.net`. Portal notice + reconfiguration guide go out
   simultaneously.
6. Later cleanup (separate change, after stability): remove `.registrar.`
   acceptance, DNS, cert; remove legacy-suffix code path.

## 9. Risks

- Device outage at migration: accepted by decision 2 (immediate cutover);
  bounded by comms + self-healing webphones; rollback via section 7 log.
- Lookup failure on incoming calls: fail closed (reject + log); Redis-cached;
  strictly better than today's silent uuid.Nil.
- 4-char label enumeration: 36^4 = 1.68M combinations IS sweepable by a
  determined scanner (flagged during design; accepted by CEO/CTO decision 5).
  Mitigations: the Kamailio pike/ban ingress gate rate-limits and bans
  scanning sources; a discovered domain still requires digest auth for every
  INVITE and REGISTER; domains are not treated as secrets. Revisit label
  length if scanning telemetry shows targeted enumeration.
- Wrong-order deploy: hard gates in section 8; the step-2 gate is documented
  in both this doc and the implementation plan.
- RFC 3261 hard limit remains: fat SDPs (video, many codecs) can exceed MTU
  regardless; TCP/TLS stays documented as the fallback. This design widens
  the UDP-safe envelope (~134B headroom on the incident INVITE), it cannot
  make it unbounded.
- api-validator / sip-validator configs reference uuid domains
  (env-overridable defaults) — update alongside step 5.

## 10. Verification plan

1. Unit (aggressive coverage per house rules): label generation (charset,
   length, reserved list, collision retry), table lifecycle on customer
   events, lookup RPC (hit, miss, cache), call-manager resolution (short
   realm, legacy uuid realm via lookup, unknown realm rejects), ARI contact
   refresh both formats, dual-suffix classification, AND the re-provision
   path itself: asserts extension `id`, `password`, `direct_id` are
   unchanged across a re-provision, ps_*/sipauth rows carry the new realm,
   exactly one extension_updated event is published (zero deleted/created),
   no billing limit RPC is called, and interruption mid-extension leaves a
   recoverable state (idempotent re-run).
2. Sandbox E2E: full migration dry-run — provision uuid-domain customer
   (with an agent bound to an extension and a flow targeting it), run the
   batch, verify: REGISTER under `<label>.reg.<base>` (Asterisk identity),
   inbound + outbound calls, contact cache, digest auth over UDP, stale
   ps_contacts purged. IDENTITY INVARIANT (the round-1 MAJOR defect class):
   across the batch, each extension's `id`, `password`, and
   `direct_id`/direct hash are byte-identical, `sip:direct.<hash>@...`
   still resolves, the agent's extension address binding still validates,
   and the event stream contains ONLY extension_updated (no
   extension_deleted/extension_created). Rollback run restores the uuid
   realm with the SAME identity assertions.
3. Frontend E2E: square-talk webphone self-heal after migration (reload ->
   re-register on new domain); square-admin Registration Address shows the
   API value.
4. Byte verification: capture the authenticated INVITE from Linphone against
   a migrated account; assert wire size < 1,500B (expected ~1,366B) and the
   call completes over UDP on the office network where the incident
   reproduced.
5. TLS: WSS + TLS registration against `<label>.reg.voipbin.net` validates
   against the new wildcard cert.
