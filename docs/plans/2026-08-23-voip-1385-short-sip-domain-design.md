# VOIP-1385: Short customer SIP domain — design comparison

Date: 2026-08-23
Ticket: VOIP-1385 (Support large SIP INVITE over UDP)
Status: DRAFT for review

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
label is the only server-side lever with meaningful savings.

Byte math (label 36 chars -> 8 chars, suffix unchanged):
- Occurrences: R-URI, From, To, digest uri (4x28B = 112B) + digest realm (28B when
  realm is also short) = up to 140B.
- 1,556B -> ~1,416B (headroom ~84B). Suffix shortening (phase 3 candidate,
  `registrar` -> shorter label) could add ~30-45B more but requires new wildcard
  cert + DNS; out of scope here.

## 2. Dependency audit (what the current format is load-bearing for)

Full audit performed 2026-08-23 across monorepo + monorepo-voip (non-vendor).

### 2.1 Forward generation (single source)
`bin-registrar-manager/models/common/common.go` — `GenerateRealmExtension`
(`<customerID>.<DOMAIN_NAME_EXTENSION>`) and `GenerateEndpointExtension`
(`<ext>@<realm>`). Called ONLY inside bin-registrar-manager:
- extensionhandler/extension.go:55,58 — realm and aorID at extension create
- contacthandler/contact.go:18,36 — rebuilds endpoint string for `ps_contacts`
  lookups and Redis contact-cache keys

### 2.2 Reverse parsing (the fragile part) — 3 sites, all in bin-call-manager
1. `pkg/callhandler/start_incoming_domain_type_registrar.go:30` —
   `customerID = uuid.FromStringOrNil(TrimSuffix(domain, ".registrar.<base>"))`
   where domain = Kamailio's `VB-Domain` header via Stasis. This customerID
   drives ALL downstream authorization for incoming registrar calls (agent
   ownership, conference ownership, flow tenancy, extension ownership).
2. `models/common/domain.go:22` `ParseSIPURI` + caller
   `pkg/arieventhandler/ari_contact.go:23` — parses `<ext>@<uuid>.registrar...`
   from ARI ContactStatusChange to refresh the contact cache. Silently yields
   uuid.Nil on non-uuid labels (stale cache, no error).
3. `pkg/callhandler/start_incoming_domain_type_trunk.go:37` — trunk analogue,
   but ALREADY uses a lookup (`RegistrarV1TrunkGetByDomainName`) instead of
   parsing an id out of the label. This is the pattern to adopt.

Suffix-only classification (`getDomainTypeIncomingCall`, start.go:424) tolerates
any label; it must keep the `.registrar.<base>` / `.trunk.<base>` hierarchy.

### 2.3 Asterisk identity coupling
All of `ps_endpoints.id/aors/auth`, `ps_aors.id`, `ps_auths.id` share the single
string `<ext>@<uuid-domain>`; `ps_auths.realm` = `<uuid-domain>`. Asterisk
registrar runs realtime with `identify_by` unset (default `username,ip`), so the
From `user@domain` IS the endpoint identity for REGISTER. `ps_domain_aliases`
is wired in sorcery.conf but unpopulated/unused (no code manages that table).
No code path rewrites ps_* ids or realm after creation — a domain
change for an existing extension requires delete+recreate (existing code path).

### 2.4 Digest auth coupling
Kamailio `validate_authentication` challenges with realm = `$fd` (From domain)
and verifies via `auth_check` against `registrar_sip_auths` with
`domain_column=realm`, `calculate_ha1=1` (plaintext password, HA1 computed at
runtime with the realm string from the client's header). Therefore: the realm
string is a free variable cryptographically, but `registrar_sip_auths.realm`
must byte-match the From domain the client uses.

### 2.5 Client-visible contract
`Extension.DomainName` is documented in the public API (bin-openapi-manager/openapi/openapi.yaml:6515) as
"Domain name, same as the customer_id", and the code stores the bare uuid
string. Note a PRE-EXISTING inconsistency: the RST docs
(extension_tutorial.rst:35, extension_struct_extension.rst:19,33) show
`domain_name` as the FULL domain (`<uuid>.registrar.voipbin.net`). This design
must settle the semantic (see Option B change list).

### 2.5b Frontend constructs the domain client-side (critical)
`monorepo-javascript/square-talk/src/config.js:13-22` (agent webphone) builds
the WSS registrar URL and the SIP URI from the customer uuid directly
(`wss://<customerId>.<registrar_host>`, `sip:<user>@<customerId>.<...>`), used
for real registration in `src/lib/webphone.js:170-171`. square-admin does the
same for the customer-facing "Registration Address"
(`src/views/extensions/extensions_detail.js:209`:
`${extension}@${customer_id}.${getRegistrarHostname()}` — "Use this address to
register your SIP device"); only the Domain Name display at :196 follows the
API value. Any change to the domain format MUST land BOTH frontend fixes
(consume the authoritative domain from the API instead of constructing it)
FIRST, or new-customer webphones break and admins are shown non-working
registration addresses. Other constructors found (cosmetic/configurable):
monorepo-monitoring/sip-validator/sip_config.py:14 (env-overridable default),
sandbox/scripts/setup_test_customer.sh:210.

### 2.6 TLS / DNS
Wildcard cert `*.registrar.voipbin.net` and wildcard A record match exactly one
label: a SHORTER label is fully covered; changing the hierarchy is not. No DNS
or cert work needed for either option below.

## 3. Option A — alias + edge normalization (Kamailio rewrites)

Every customer keeps the canonical uuid domain internally. A short alias domain
per customer exists only at the network edge. Kamailio, on ingress, maps the
short label -> customer uuid (Redis lookup provisioned by registrar-manager) and
rewrites the SIP message to the canonical uuid domain BEFORE auth. Everything
behind Kamailio (auth rows, Asterisk ps_*, call-manager parsing, ARI events)
sees only uuid domains — zero Go changes, zero data migration.

What must be rewritten per message:
- R-URI domain (covers `VB-Domain`, dispatch, dialplan)
- To domain (REGISTER AOR identity; INVITE callee context)
- From domain (Asterisk endpoint identification `user@domain` for REGISTER;
  auth realm derivation `$fd`)

Consequences of From/To rewriting:
- REGISTER is stateless per transaction — header rewrite is safe (textops).
- INVITE creates a dialog: From/To URIs are dialog state. The UAS copies the
  REWRITTEN From/To into responses and the client receives From/To differing
  from what it sent. Kamailio's `uac` module (`uac_replace_from/to`) exists
  precisely for this and restores originals on responses and in-dialog requests
  via Record-Route parameters, but it adds per-dialog restore logic on the
  hottest path, with known corner cases (strict UAs, REFER/transfer flows, WSS,
  retransmissions during restore, interplay with our topology).
- Digest: since From is rewritten to uuid BEFORE auth, challenge realm = uuid
  domain; existing `registrar_sip_auths` rows match; NO new auth rows needed.
  Cost: the client's Proxy-Authorization carries the long uuid realm again, so
  savings drop to ~112B (result ~1,444B).

Ops footprint: new Redis mapping keyspace on the ingress path (lookup for every
request with a non-uuid label, before auth — i.e., unauthenticated traffic can
trigger lookups; needs negative-caching), plus the uac restore state.

Pros:
- No Go/service changes, no DB migration, no API semantic change.
- All customers (existing included) can adopt the alias immediately by
  reconfiguring devices; old domain keeps working side by side.

Cons:
- SIP header rewriting with dialog-state restore on 100% of aliased traffic —
  highest protocol risk category we have (the exact path where subtle bugs cost
  production calls). uac_replace_from is proven but not free of edge cases.
- Permanent second domain system: every future feature must consider two
  domains per customer forever.
- Kamailio gains a Redis dependency in the ingress hot path.
- Smaller savings (~1,444B; only ~56B of headroom for fatter SDPs).
- Reverse-parsing fragility in call-manager remains untouched (debt stays).

## 4. Option B — canonical short domain (recommended)

The customer's domain simply becomes short. registrar-manager generates a
random short label (8 chars, base36 lowercase, collision-checked, stored) and
uses it in `GenerateRealmExtension`; everything downstream (ps_*, auth rows,
VB-Domain, ARI events, API responses) consistently carries the short domain.
No Kamailio changes at all, no header rewriting, full ~140B savings (~1,416B).

Required code changes (from the audit):
0. Frontend (PREREQUISITE, must land before phase 1 — see Rollout 0b):
   - `monorepo-javascript/square-talk`: stop constructing the domain from
     customer uuid in config.js; consume the authoritative `domain_name`
     (full realm) from the extension API.
   - `monorepo-javascript/square-admin`: same fix for the "Registration
     Address" constructor (extensions_detail.js:209).
1. `bin-registrar-manager`:
   - New per-customer `domain_name` (short label) storage + generation
     (collision-checked) and inclusion in extension/trunk provisioning.
   - New RPC `RegistrarV1CustomerGetByRealm` (mirroring the existing trunk
     lookup pattern), backed by `registrar_extensions` — the table that holds
     BOTH `customer_id` and `realm` per row, indexed by `idx_extensions_realm`
     / `idx_registrar_extensions_realm`. (`registrar_sip_auths` has NO
     customer_id column and cannot back this lookup without a join.)
     Phase-1 semantics: a realm resolves only if the customer has at least one
     non-deleted extension — safe, because reaching the registrar call path
     requires digest auth, which requires a sipauth row, which only exists
     alongside an extension.
   - Invariant restated: `Extension.Realm` / `Trunk.Realm` columns keep their
     existing meaning (the literal realm string Kamailio byte-matches — the
     "DO NOT CHANGE" comments in extension.go:28 / trunk.go:21 stay honored);
     only the generated VALUE changes for new customers.
2. `bin-call-manager` — replace the 3 reverse-parse sites with the lookup:
   - `start_incoming_domain_type_registrar.go:30` -> lookup by realm (adopt the
     trunk pattern one file over). Add negative-result handling (unknown realm
     -> reject call) — an IMPROVEMENT over today's silent uuid.Nil.
   - `ParseSIPURI` + `ari_contact.go` -> resolve customer via the same lookup
     (or via extension lookup by endpoint string).
   - `getDomainTypeIncomingCall` unchanged (suffix classification intact).
3. `bin-dbscheme-manager`: migration adding the short-label column (customer- or
   extension-level realm already stored; only the new label store + index).
4. API/docs — SETTLED DECISION: `Extension.DomainName` becomes the FULL
   authoritative realm (`<label>.registrar.<base>`) for ALL customers
   (existing customers serve their stored realm, i.e. the uuid-labeled full
   domain). Rationale: (a) consume-verbatim frontends require a full domain
   for every customer — a "new customers only" variant would leave existing
   customers' value a bare uuid and break verbatim consumers for them, so it
   is incompatible with the Phase 0 design and rejected; (b) this fixes the
   pre-existing code-vs-RST inconsistency (openapi.yaml:6515 says "same as
   customer_id" and code returns the bare uuid, while RST already documents
   the full domain — RST wins); (c) known first-party consumers are safe
   (square-admin display :196 shows it verbatim; square-talk display_name
   fallback webphone.js:172); the residual blast radius is third-party API
   consumers who relied on domain_name == customer_id equality — called out
   in release notes; the uuid remains extractable from the legacy label.
   Touch points: openapi.yaml description + gens regeneration
   (bin-openapi-manager/gens/models/gen.go embeds the stale description;
   bin-api-manager embeds it via openapi_redoc/api.html — both regenerate from
   openapi.yaml), `WebhookMessage` (models/extension/webhook.go:19 — webhook
   payloads carry domain_name too; also DELETE the stale comment "used by the
   kamailio's INVITE validation" — Kamailio never reads domain_name), RST
   files (extension_overview/tutorial/struct, quickstart_extension,
   trunk_overview_domain_name, direct_hash_overview, self_hosting_envvars),
   SDK regeneration (voipbin-go, python-sdk, vn CLI, MCP — type is plain
   string so non-breaking; regen for the description only) + release notes.
   api-validator verified no-impact (tests/models/extension.py:12 has no
   format assertion).
   Phase 0a NULL-realm rule: legacy extension rows (pre-Feb-2024) have NULL
   realm; when serving DomainName, fall back to computing
   `<customer_id>.registrar.<base>` for NULL realm so the API never emits an
   empty domain (these rows cannot digest-auth anyway, so this is
   representation-only).
5. Caching: lookup result cached in call-manager/registrar-manager Redis (realm
   -> customer_id), invalidated on domain change.

Rollout (strict ordering — 0a before 0b before 1):
- Phase 0a (backend API, no domain-format change): `DomainName` starts serving
  the FULL realm for all customers (change list item 4). Existing devices are
  unaffected (their configured domains do not change); only the API
  representation changes.
- Phase 0b (frontend): square-talk AND square-admin stop constructing the
  domain from the customer uuid and consume `domain_name` verbatim. Safe only
  AFTER 0a is live everywhere. Belt-and-braces for rollback windows: the
  frontends may keep a fallback "value contains no dot -> append
  `.registrar.<base>`" until 0a is confirmed stable, then remove it.
- Deploying 0b before 0a would break ALL existing webphones (they would build
  `wss://<bare-uuid>` with no suffix) — this ordering is a hard gate, stated
  here so the implementation plan cannot invert it.
- Phase 1: NEW customers get short domains. Existing customers unaffected:
  uuid domains keep working through the SAME lookup (realm rows for uuid
  domains resolve identically; the TrimSuffix code is deleted, both formats go
  through the lookup uniformly). Legacy note: extension rows created before
  the realm column existed (Feb 2024) have NULL realm and would fail the new
  lookup — no regression, since their identically-NULL sipauth realm already
  fails Kamailio's byte-match digest auth today.
- Phase 2: opt-in regeneration for existing customers ("regenerate SIP domain"
  operation = recreate extensions with new realm via the existing
  delete+create path; customer reconfigures devices; brief re-registration
  window). Per-customer flag day, self-serve or support-driven.
- Phase 3 (optional, separate): shorten the `.registrar.` suffix level —
  requires new wildcard cert + DNS + Kamailio tls.cfg; only if we need the
  extra ~30-45B.

Pros:
- Zero SIP protocol risk: no header rewriting, no dialog-state restore, no
  Kamailio change, no new ingress dependency.
- Pays down existing debt: the fragile TrimSuffix+uuid.FromStringOrNil parsing
  (which today silently produces uuid.Nil on any anomaly) is replaced by an
  explicit lookup with error handling, unifying with the trunk pattern.
- Full savings (~1,416B), and realm is short too.
- One domain per customer — no permanent dual-domain system.
- Better UX: short domains are typeable; portal/docs get simpler.

Cons:
- Go changes across 2 services + DB migration + API semantic change (docs).
- Existing customers need opt-in regeneration (device reconfig + short
  re-registration gap) to benefit; until then they must use TCP/TLS as the
  workaround for the UDP issue.
- The domain is no longer self-describing: one extra (cached) lookup per
  incoming registrar call and per contact-status event.

## 5. Comparison summary

| Axis | A: alias + edge rewrite | B: canonical short domain |
|---|---|---|
| INVITE size | ~1,444B (realm stays long) | ~1,416B (full savings) |
| SIP protocol risk | High (From/To rewrite + dialog restore on hot path) | None |
| Backend code changes | None | registrar-manager + call-manager (3 sites) + migration |
| Frontend changes | None (uuid stays canonical) | square-talk + square-admin domain-construction fixes (prerequisite) |
| Kamailio changes | Significant (mapping, rewriting, uac restore) | None |
| Provisioning work | Alias lifecycle (create/regen/delete, collision) — duplicates most of B's registrar-manager work anyway | Label lifecycle (same class of work) |
| Data migration | None | Column add only (phase 1); per-customer regen (phase 2) |
| Existing customers | Alias available immediately, dual domains forever | Opt-in regeneration (flag day per customer) |
| Architectural debt | Adds permanent dual-domain + ingress Redis dependency | Removes existing reverse-parse fragility |
| API contract | Unchanged | DomainName semantic settled (fixes existing code-vs-docs inconsistency) |
| Blast radius on bug | Live SIP dialogs (worst case) | Call setup lookup path (fails closed) |

## 6. Recommendation

Option B. The decisive factors:
1. Risk asymmetry: A concentrates risk in SIP dialog-state manipulation on the
   production hot path, where failures are live-call-affecting and hard to
   reproduce. B's risk is ordinary data-plumbing, fails closed, and is fully
   unit-testable.
2. B removes existing fragility (silent uuid.Nil parsing) instead of layering a
   second domain system on top of it.
3. B achieves the full byte savings; A leaves only ~56B headroom.
4. The main advantage of A (immediate coverage of existing customers) is weaker
   than it appears: existing customers must reconfigure devices to benefit under
   EITHER option; B's phase 2 covers them with modest additional work.

## 7. Risks and mitigations (Option B)

- Lookup failure/latency on incoming calls: cache realm->customer_id in Redis
  with invalidation on regen; unknown realm rejects the call (fail closed,
  logged). Today's behavior on anomaly is worse (silent uuid.Nil).
- Collision of short labels: 36^8 ~ 2.8e12 space; generation retries on the
  unique index. uuid-format ambiguity is impossible (uuid labels are 36 chars).
  Reserved-label exclusion (`pstn`, `sip`, `echo`, `registrar`, `trunk`) is
  NOT needed in phases 1-2 (the `.registrar.` level keeps them from colliding
  with Kamailio's `pstn.<base>` etc. routes) — it becomes relevant only if
  phase 3 flattens the suffix; kept as a cheap future-proofing rule anyway.
- Mixed formats during transition: both uuid and short realms resolve through
  the same lookup path — no format branching in call-manager after the change.
- Regeneration (phase 2) drops registrations until devices re-register with the
  new domain: communicated as part of the opt-in flow.
- RFC 3261 hard limit remains: messages can still exceed MTU with fat SDPs
  (video, many codecs). TCP/TLS stays available and documented as the fallback;
  this design widens the UDP-safe envelope, it cannot make it unbounded.

## 8. Verification plan

1. Unit: registrar-manager label generation (format, collision retry, reserved
   labels), lookup RPC; call-manager incoming-call resolution for short realm,
   uuid realm (compat), unknown realm (reject), ARI contact refresh for both
   formats. Aggressive coverage per house rules.
2. Sandbox E2E: provision customer with short domain -> REGISTER (Asterisk
   identity `<ext>@<short-domain>`), inbound + outbound calls, contact-status
   cache refresh, digest auth over UDP with a real softphone. Frontend E2E:
   square-talk webphone registration AND the square-admin extension detail
   page (Registration Address shows the API-provided domain) for a
   short-domain customer, plus an EXISTING uuid-domain customer after
   phase 0a/0b (verbatim consumption serves the uuid-labeled full realm).
3. Byte verification: capture the authenticated INVITE from Linphone against a
   short-domain account; assert wire size < 1,500B and call completes over UDP
   on the office network where the incident reproduced.
4. Regression: existing uuid-domain customer flows unchanged (E2E on a legacy
   account).
