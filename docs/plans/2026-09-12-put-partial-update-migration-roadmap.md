# PUT endpoint partial-update migration: platform-wide roadmap

Status: DRAFT (Round 0)
Author: CPO, per 대표님 approval ("점진적으로 수정하는 방향으로 진행하자")
Related: `docs/plans/2026-09-12-mcp-server-put-partial-update-design.md` (the
already-merged reference implementation this roadmap generalizes, PR #1291,
merged as `81a9a6b98`).

## 1. Problem statement

An audit of every `PUT` path under `bin-openapi-manager/openapi/paths/**/*.yaml`
(**66** total `put:` operations, re-verified via `grep -rlE '^put:'`) found
the platform has no single PUT convention. Endpoints split as follows:

- **32 endpoints** already accept fully partial bodies (every request-body
  field optional, or the endpoint is single-field by design, e.g.
  `/agents/{id}/status`) — this is the residual bucket (66 total − 31
  strict − 3 hybrid = 32), independently recounted, not re-derived from
  the earlier (incorrect) "35" figure.
- **31 endpoints** declare `required:` on **every** field in the PUT
  request body ("strict full-replace"). A client must resend the entire
  current state to change one field, or risk silently wiping fields it
  didn't intend to touch, exactly the class of bug fixed in
  `PUT /mcpservers/:id` (PR #1291): some full-replace endpoints fail loudly
  on omission (400), others -- worse -- fail silently, writing empty
  string / zero value over existing data with a `200 OK` response.
- **3 endpoints** (`PUT /providers/{id}`, `PUT /teams/{id}`,
  `PUT /flows/{id}`) are "hybrid" -- all but one body field is required
  (`codecs`, `parameter`, `on_complete_flow_id` respectively are already
  optional). These are included in this migration's scope (their required
  fields have the same silent-wipe risk as strict full-replace endpoints)
  but are called out separately because the fix is smaller: only the
  already-required fields need migrating, not the whole schema.

31 strict + 3 hybrid = **34 endpoints in scope** for this migration (the
figure used throughout the rest of this document); "full-replace" below
refers to this combined set of 34 unless a section explicitly says
"strict full-replace."

This roadmap does not re-litigate whether to fix this (대표님 already
approved "점진적으로" — phased). It defines: the inventory, the risk-based
phase ordering, the reusable technical pattern, and the per-phase scope and
exit criteria, so each phase can run through this monorepo's standard
design-review-loop and PR-review-loop independently rather than needing one
34-endpoint mega design doc.

## 2. Full inventory of the 34 in-scope endpoints (31 strict full-replace + 3 hybrid), risk-ranked

Risk model (same reasoning as PR #1291 §1): rank by (a) whether a
field's zero-value is a *silent, valid-looking* state (worst — e.g.
`auth_type: ""` is a real enum member) vs. one that fails loudly (400,
self-limiting blast radius) vs. cosmetic; and (b) whether the field is
credential/webhook/routing-critical (customer-facing breakage or security
exposure) vs. purely descriptive (`name`/`detail`).

| # | Endpoint | Service | Fields (required=all) | Highest-risk field(s) | Risk |
|---|---|---|---|---|---|
| 1 | `PUT /customer` | bin-customer-manager | name, detail, email, phone_number, address, webhook_method, webhook_uri | `webhook_uri`/`webhook_method` — omission wipes the customer's only webhook target; every backend event silently stops delivering with no error | **HIGH** |
| 2 | `PUT /customers/{id}` | bin-customer-manager | same 7 fields (admin surface) | same as #1 | **HIGH** |
| 3 | `PUT /trunks/{id}` | bin-registrar-manager | name, detail, auth_types, username, password, allowed_ips | `password`/`allowed_ips` — omission wipes trunk auth; SIP trunk stops authenticating or, worse, `allowed_ips` cleared opens it | **HIGH** |
| 4 | `PUT /extensions/{id}` | bin-registrar-manager | name, detail, password | `password` — omission wipes extension credential, SIP registration breaks silently (this is the exact sibling bug to the already-fixed `secret` field on mcpservers) | **HIGH** |
| 5 | `PUT /routes/{id}` | bin-route-manager | name, detail, provider_id, priority, target | `provider_id`/`target` — wrong/empty value can silently misroute live outbound traffic | **HIGH** |
| 6 | `PUT /providers/{id}` | bin-route-manager | type, hostname, tech_prefix, tech_postfix, tech_headers, name, detail (hybrid: `codecs` already optional) | `hostname`/`tech_prefix`/`tech_postfix` are routing-determining, comparable to `routes/{id}`'s `provider_id`/`target` (#5, HIGH); rated MEDIUM here on narrower blast radius (a provider misconfiguration affects calls routed through that one provider, not the route table's overall dispatch), with low migration effort as a separate phasing tiebreaker, not part of the risk rating itself | MEDIUM |
| 7 | `PUT /queues/{id}` | bin-queue-manager | name, detail, routing_method, tag_ids, wait_flow_id, wait_timeout, service_timeout | `wait_flow_id`/`tag_ids` — wrong flow id can silently break the wait experience; tag_ids omission would declaw routing filters | MEDIUM |
| 8 | `PUT /conferences/{id}` | bin-conference-manager | name, detail, data, timeout, pre_flow_id, post_flow_id | `pre_flow_id`/`post_flow_id` — omission silently detaches conference hooks | MEDIUM |
| 9 | `PUT /numbers/{id}` | bin-number-manager | call_flow_id, message_flow_id, name, detail | `call_flow_id`/`message_flow_id` — a live phone number's call flow could be wiped | MEDIUM |
| 10 | `PUT /numbers/{id}/flow_id` | bin-number-manager | call_flow_id, message_flow_id | same as #9, narrower single-purpose PUT | MEDIUM |
| 11 | `PUT /campaigns/{id}` | bin-campaign-manager | name, detail, type, service_level, end_handle | `end_handle` — omission could alter live campaign completion behavior | MEDIUM |
| 12 | `PUT /outplans/{id}/dial_info` | bin-campaign-manager (`pkg/outplanhandler/`) | source, dial_timeout, try_interval, max_try_count_0..4 (8 fields) | dial timing fields — wrong/omitted values change live autodialer pacing | MEDIUM |
| 13 | `PUT /teams/{id}` | bin-ai-manager (`pkg/teamhandler/`) | name, detail, start_member_id, members (hybrid: `parameter` already optional) | `members`/`start_member_id` — wipe risk on team roster | MEDIUM |
| 14 | `PUT /flows/{id}` | bin-flow-manager | name, detail, actions (hybrid: `on_complete_flow_id` already optional) | `name`/`detail` are plausible-to-omit and migrate in Phase 7; `actions` is permanently exempt (§3.1) — its risk tier is N/A, not assessed, since it will never become optional | LOW (name/detail); N/A — exempt, see §3.1 (actions) |
| 15 | `PUT /outdials/{id}` | bin-outdial-manager | name, detail | cosmetic only | LOW |
| 16 | `PUT /outdials/{id}/campaign_id` | bin-outdial-manager | campaign_id | single-field PUT, no partial-update concept applies | LOW (exempt, see §3.2) |
| 17 | `PUT /outdials/{id}/data` | bin-outdial-manager | data | single-field PUT | LOW (exempt) |
| 18 | `PUT /outplans/{id}` | bin-campaign-manager (`pkg/outplanhandler/`) | name, detail | cosmetic only | LOW |
| 19 | `PUT /tags/{id}` | bin-tag-manager | name, detail | cosmetic only | LOW |
| 20 | `PUT /service_agents/me` | bin-agent-manager | name, detail, ring_method | self-service, agent immediately notices if `ring_method` silently resets; sorts with MEDIUM for phase-ordering purposes (agent-visible behavior change, though not credential/routing-critical) | MEDIUM (self-service, agent-visible) |
| 21 | `PUT /service_agents/me/password` | bin-agent-manager | password | single-field PUT | LOW (exempt) |
| 22 | `PUT /service_agents/me/addresses` | bin-agent-manager | addresses | single-field PUT (array = the whole resource) | LOW (exempt) |
| 23 | `PUT /service_agents/me/status` | bin-agent-manager | status | single-field PUT | LOW (exempt) |
| 24 | `PUT /contact_cases/{id}` | bin-contact-manager | contact_id | single-field PUT | LOW (exempt) |
| 25 | `PUT /service_agents/contact_cases/{id}` | bin-contact-manager | contact_id | single-field PUT | LOW (exempt) |
| 26 | `PUT /customer/billing_account_id` | bin-customer-manager | billing_account_id | single-field PUT | LOW (exempt) |
| 27 | `PUT /customers/{id}/billing_account_id` | bin-customer-manager | billing_account_id | single-field PUT | LOW (exempt) |
| 28 | `PUT /campaigns/{id}/actions` | bin-campaign-manager | actions | single-field PUT (array = the whole resource, like flow actions) | LOW (exempt) |
| 29 | `PUT /campaigns/{id}/next_campaign_id` | bin-campaign-manager | next_campaign_id | single-field PUT | LOW (exempt) |
| 30 | `PUT /campaigns/{id}/resource_info` | bin-campaign-manager | outplan_id, outdial_id, queue_id, next_campaign_id | 4 fields but conceptually one atomic "resource binding" that should always move together; permanently exempt (§3.1) — risk tier is N/A, not assessed, since these fields will never become independently optional | N/A — exempt, see §3.1 |
| 31 | `PUT /campaigns/{id}/service_level` | bin-campaign-manager | service_level | single-field PUT | LOW (exempt) |
| 32 | `PUT /campaigns/{id}/status` | bin-campaign-manager | status | single-field PUT | LOW (exempt) |
| 33 | `PUT /queues/{id}/routing_method` | bin-queue-manager | routing_method | single-field PUT | LOW (exempt) |
| 34 | `PUT /queues/{id}/tag_ids` | bin-queue-manager | tag_ids | single-field PUT (array = the whole resource) | LOW (exempt) |

### 3.1 Endpoints excluded from this migration by design, not oversight

Some multi-field "full replace" endpoints are not bugs and should stay
full-replace:

- **`PUT /flows/{id}`'s `actions`**: a flow's action list is edited as a
  whole document by the flow editor (add/remove/reorder actions is a
  client-side operation on the full array, then the full array is saved).
  There is no natural "omit `actions` to leave it unchanged" use case for
  a PUT that also changes `name`/`detail` — the caller either is renaming
  a flow (in which case sending the current `actions` array back is not
  onerous, it's already loaded in the editor) or is editing actions (in
  which case there's nothing to "omit"). Converting `actions` to
  optional/nil-means-unchanged would only invite an accidental clear if a
  client's array serialization bug ever produced `null`/empty. **Decision:
  keep `name`/`detail`/`on_complete_flow_id` migrated to optional
  (they ARE plausible to omit — e.g. a "rename only" action), but leave
  `actions` required.** This makes `PUT /flows/{id}` a hybrid, not a full
  migration — call this out explicitly in that endpoint's own phase design
  doc.
- **`PUT /campaigns/{id}/resource_info`**: `outplan_id`/`outdial_id`/
  `queue_id`/`next_campaign_id` are bound together as one atomic
  reassignment operation (rebinding a campaign to a different outplan
  almost always means rebinding its outdial and queue together too,
  per `bin-campaign-manager`'s domain model). Partial update here risks
  a campaign left pointing at a queue from its old outplan. **Decision:
  exempt, keep full-replace**, unless a future concrete use case
  (re-verified against the domain, not assumed here) demonstrates
  independent field updates are needed.

### 3.2 Single-field PUTs are structurally exempt

Any endpoint whose body has exactly one field (15 of the 34, marked
"exempt" above) has no partial-vs-full distinction to fix: "the field" and
"the whole resource" are the same thing. Making it "optional" would only
create a meaningless no-op PUT. No design/implementation work needed for
these; they are listed here only so the inventory accounts for all 34 and
a future audit doesn't rediscover them as "unaddressed."

## 3. The reusable technical pattern (established, not re-designed)

PR #1291 already designed and shipped the canonical pattern; every phase
below applies it, it is not reinvented per phase:

1. OpenAPI request-body schema: remove the field from `required:`; the
   type itself does not change (already generates as a pointer in Go via
   `oapi-codegen` for a non-required object property).
2. `bin-api-manager/server/<resource>.go`: stop dereferencing
   `if req.X != nil { x = *req.X } else { x = "" }` down to a plain value;
   pass the generated `*T` straight through.
3. `bin-api-manager/pkg/servicehandler/<resource>.go`: accept pointers.
4. `bin-common-handler/pkg/requesthandler/<resource>.go`: accept pointers,
   marshal into the RabbitMQ-hop request DTO as pointers too (mandatory —
   §4.4 of the mcpservers design doc explains why skipping this
   reintroduces the bug one hop later).
5. Owning service's `pkg/listenhandler/models/request/<resource>.go`: wire
   DTO fields become pointers with `omitempty`.
6. Owning service's `pkg/listenhandler/v1_<resource>.go`: pass pointers
   through unchanged.
7. Owning service's business handler (`pkg/<resource>handler/handler.go`):
   this is the actual fix site — build the SQL field map conditionally,
   `if field != nil { fields[...] = *field }`, mirroring PR #1291 §4.1
   verbatim. Any field-specific validation (enum validity, URL format,
   etc.) also becomes conditional on non-nil.
8. `len(fields) == 0` short-circuits to the existing `Get` (reuse its
   `ErrNotFound` mapping), per PR #1291 §4.1's documented rationale.
9. OpenAPI `description` per field updated to state the
   omitted-means-unchanged contract.
10. RST docs (`bin-api-manager/docsdev/source/<resource>_struct*.rst` or
    `*_overview.rst`) updated with an Implementation Hint note per this
    monorepo's mandatory RST-sync rule, clean Sphinx rebuild committed.
11. Tests at every layer per PR #1291 §7's structure: business-handler
    table tests (per-field-omitted cases, all-omitted no-op case,
    zero-value-explicitly-set-is-still-a-real-value case for any enum
    field whose zero value is valid), listenhandler end-to-end test, and
    critically the `bin-api-manager/server/<resource>_test.go` HTTP-layer
    test (PR #1291's Round 1 review caught that this layer is where the
    dereference-to-zero-value bug actually lives and is invisible to every
    test below it).

No phase's design doc needs to re-derive this pattern; each phase's design
doc should be a short "apply the established pattern to endpoint(s) X, Y,
Z" doc that names the endpoint-specific risk (per §2's table) and any
endpoint-specific deviation (e.g. §3.1's `flows`/`resource_info` partial
exemptions), not a full re-justification.

## 4. Phasing strategy

Ship HIGH risk first (biggest silent-data-loss exposure), batch by owning
service so one phase = one or two services = one worktree = one PR,
keeping each PR reviewable and each service's verification workflow
(`go mod tidy && go mod vendor && go generate ./... && go test ./... &&
golangci-lint run`) scoped to what actually changed.

| Phase | Endpoints | Services touched | Rationale for grouping |
|---|---|---|---|
| **Phase 1** | `PUT /extensions/{id}`, `PUT /trunks/{id}` | bin-registrar-manager | Both HIGH risk, both carry a credential field (`password`) with the exact same silent-wipe shape as the already-fixed `mcpservers.secret`/other fields bug — most direct precedent, same owning service, smallest safe first step to validate the pattern generalizes cleanly outside bin-ai-manager. |
| **Phase 2** | `PUT /customer`, `PUT /customers/{id}` | bin-customer-manager | Both HIGH risk, same webhook_uri/webhook_method exposure, same owning service, same schema (self-service vs admin surface of the same resource) — natural single PR. |
| **Phase 3** | `PUT /routes/{id}` | bin-route-manager | HIGH risk (live traffic misroute), single endpoint, isolated service. |
| **Phase 4** | `PUT /queues/{id}`, `PUT /providers/{id}` | bin-queue-manager, bin-route-manager | MEDIUM risk; `providers` is nearly-already-partial (only 1 field away) so low incremental effort. Two separate PRs (per this monorepo's one-PR-per-repo-boundary rule) landed in the same phase/timeframe, same as Phase 5/6 below. |
| **Phase 5** | `PUT /conferences/{id}`, `PUT /numbers/{id}`, `PUT /numbers/{id}/flow_id` | bin-conference-manager, bin-number-manager | MEDIUM risk, both involve flow-id wipe risk (same bug shape), natural conceptual pairing even though different services — still two separate PRs (per this monorepo's one-PR-per-repo-boundary rule), same phase/timeframe. |
| **Phase 6** | `PUT /campaigns/{id}` (bin-campaign-manager), `PUT /outplans/{id}/dial_info` (bin-campaign-manager), `PUT /teams/{id}` (bin-ai-manager) | bin-campaign-manager, bin-ai-manager | MEDIUM risk, remaining multi-field endpoints not yet covered. `campaigns`/`outplans` share bin-campaign-manager and can land in one PR; `teams` lives in bin-ai-manager and is a separate PR per this monorepo's one-PR-per-repo-boundary rule (same caveat as Phase 5). |
| **Phase 7** | `PUT /flows/{id}` (hybrid, §3.1 — `name`/`detail` migrate, `actions` stays required), `PUT /service_agents/me` | bin-flow-manager, bin-agent-manager | LOW (flows name/detail) to MEDIUM (service_agents/me) risk, includes the one hybrid-scope endpoint requiring its own explicit non-goal statement. Two separate PRs (different services). |
| **Phase 8 (optional, low priority)** | `PUT /outdials/{id}` (bin-outdial-manager), `PUT /outplans/{id}` (bin-campaign-manager), `PUT /tags/{id}` (bin-tag-manager) | bin-outdial-manager, bin-campaign-manager, bin-tag-manager | LOW risk, cosmetic-only fields (`name`/`detail`). Three separate PRs (three different services). Defer indefinitely unless a concrete client complaint surfaces — do not build ahead of signal (오버엔지니어링 지양 원칙). |

Single-field-exempt endpoints (§3.2, 15 of them) require no phase — this
roadmap's inventory work is their final disposition, closing them as
"reviewed, no change needed." `PUT /campaigns/{id}/resource_info` (§3.1)
is likewise phase-less and closed as "reviewed, kept full-replace by
design." `PUT /flows/{id}`'s `actions` field specifically (not the whole
endpoint — `name`/`detail` DO migrate, in Phase 7) is the one field-level
exemption from §3.1.

Each phase gets its own short design doc (per §3's note: apply the
established pattern, state endpoint-specific risk, no full re-derivation)
and its own worktree/branch/PR, going through this monorepo's standard
design-review-loop (min 2 rounds, 2 consecutive APPROVE) and PR-review-loop
(min 3 rounds, 2 consecutive APPROVE) independently. Phases do not block
each other — Phase 2 can start before Phase 1's PR merges if capacity
allows, though sequential is simpler to track and is the default unless
대표님 asks to parallelize.

## 5. Per-phase exit criteria

A phase is complete when:
- OpenAPI spec, all Go layers, and RST docs are updated per §3's pattern.
- Full verification workflow passes for every touched service.
- Table tests exist and were confirmed to fail pre-fix (revert-and-rerun,
  per PR #1291 §7's standing practice) for at least the highest-risk field
  in that phase.
- PR merged (squash, after 대표님's explicit "merge" instruction).
- This roadmap doc's phase table (§4) updated to mark the phase done, with
  the merged PR number, in the same follow-up commit style used elsewhere
  in this monorepo's plans (no separate tracking system needed — this doc
  IS the tracker, avoiding a new measurement/tracking tool per the
  오버엔지니어링 지양 principle of reusing existing tools).

## 6. Non-goals

- No retroactive data remediation for rows that may have already been
  silently wiped by the pre-fix full-replace behavior in production
  (mirrors PR #1291 §8's same non-goal) — raise as a separate question to
  대표님 if a specific customer complaint surfaces, not addressed here.
- No frontend (square-admin/square-talk) changes bundled into backend
  phases unless a phase's own design doc identifies a concrete behavior
  change requiring one (mirrors PR #1291's frontend-scope discipline,
  §2/§2.1 of that doc).
- No new PATCH method, no HTTP verb redesign — every endpoint stays PUT
  with optional-field partial-update semantics, consistent with the
  already-established `mcp_server_ids`/`secret`/(now) all-mcpservers-fields
  precedent.
- Phase 8 (LOW-risk cosmetic-only endpoints) is explicitly deferred, not
  scheduled — do not build it preemptively without a concrete trigger
  (repeats 대표님's standing 오버엔지니어링 지양 instruction).

## 7. Round 1 review disposition

Independent review (`deleg_17c33ddd`) verdict: REQUEST CHANGES. Findings
and fixes:

| # | Finding | Severity | Fix location |
|---|---|---|---|
| 1 | §1's headline audit numbers didn't match the repo: actual scan found 66 total `put:` operations (not 69), and only 31 endpoints have literally every body field in `required:` (not 34) | **CRITICAL** | §1 rewritten: 66 total confirmed via `grep -rlE '^put:'`; introduced "31 strict full-replace + 3 hybrid = 34 in scope" framing |
| 2 | §2 rows for `providers`, `teams`, `flows` were counted as full-replace despite each having one non-required field (`codecs`, `parameter`, `on_complete_flow_id`), contradicting §1's own definition | **MAJOR** | §1 and §2 now explicitly label these 3 as "hybrid" with the specific already-optional field named per row |
| 3 | §2/§4 attributed `outplans` to a nonexistent "bin-outplan-manager" and `teams` to a nonexistent "bin-team-manager"; actual owners are `bin-campaign-manager` (`pkg/outplanhandler/`) and `bin-ai-manager` (`pkg/teamhandler/`), confirmed by directory listing | **MAJOR** | §2 rows 12/13/18 and §4 Phase 6/8 corrected to real service names; Phase 6/8 grouping rationale updated to state the resulting multi-PR split explicitly |
| 4 | §3.2 said "14 of the 34" single-field-exempt endpoints; actual count of rows tagged exempt is 15 | **MINOR** | §3.2 corrected to 15; §4 closing paragraph updated to match |
| 5 | §4's closing paragraph implied the whole `flows` endpoint is phase-less via §3.1, but `PUT /flows/{id}` (`name`/`detail`) is scheduled in Phase 7 — only its `actions` field is exempt | **MINOR** | §4 closing paragraph rewritten to distinguish the field-level `actions` exemption from the endpoint-level Phase 7 scheduling |

Not changed (reviewer confirmed correct as originally written): §3.1's
exemption reasoning for `resource_info` and the `flows`/`actions` split;
the technical pattern in §3 (verified byte-accurate against the shipped
PR #1291 implementation); the document's scope (roadmap-only, deferring
per-phase design docs, appropriate under this monorepo's design-first
workflow); Korean-language usage and register.

## 8. Round 2 review disposition

Independent review (`deleg_985a29a8`) verdict: REQUEST CHANGES. A fresh
recount (not anchored to Round 1's disposition table) found the same
failure mode recurring in the adjacent, un-re-verified figure, plus two
risk-model consistency nits:

| # | Finding | Severity | Fix location |
|---|---|---|---|
| 1 | §1's complementary "already-partial" bucket was still stated as 35, which is inconsistent with the corrected 66-total/31-strict/3-hybrid figures (66 − 31 − 3 = 32, not 35) — Round 1's fix corrected the 66/34 side but never re-derived this adjacent figure | **CRITICAL** | §1 first bullet corrected to 32, with the subtraction shown explicitly so it can't silently drift again |
| 2 | §2 row 6 (`providers/{id}`)'s risk justification cited "low incremental effort" as part of the MEDIUM rating, conflating implementation cost with the risk dimensions §2's own preamble defines (silent-vs-loud failure, credential/routing criticality) | MINOR | §2 row 6 rewritten to justify MEDIUM on risk grounds alone (narrower blast radius than `routes/{id}`), with effort explicitly separated out as a phasing tiebreaker, not a risk input |
| 3 | §2 rows 14 (`flows/{id}`'s `actions` sub-field) and 30 (`resource_info`) were tagged with a risk tier ("LOW") despite being permanently exempt from the migration (§3.1) — conflating "risk-assessed as low" with "structurally out of scope, never assessed" | MINOR | Both rows' risk-tier column changed to "N/A — exempt, see §3.1" |

Not changed (reviewer confirmed correct via independent re-derivation, not
just re-checking Round 1's claims): the 66/31/3/34 core inventory numbers;
all service-ownership entries in rows 15-34 (re-verified against
`bin-api-manager/pkg/servicehandler` import aliases and target-service
`pkg/` directory listings); §3.1/§3.2/§4 tally (15 single-field exempts +
1 endpoint-level exempt `resource_info` + 18 phased endpoints = 34, no
endpoint missing or double-counted); §4's phase-table service-ownership
splits (Phase 6, Phase 8); §7's Round-1-disposition table's own honesty
(it does not overclaim fixes it didn't make).

## 9. Round 3 review disposition

Independent review (`deleg_920917d7`) verdict: REQUEST CHANGES. A fresh
recount (independently re-deriving all of §1's headline numbers, not
reusing Round 2's arithmetic) found no further error in the core 66/31/3/32
figures, but surfaced three new minor issues neither Round 1 nor Round 2
caught:

| # | Finding | Severity | Fix location |
|---|---|---|---|
| 1 | §2 row 12 (`outplans/{id}/dial_info`) annotated its field count as "(9 fields)" but the schema has 8 fields (`source`, `dial_timeout`, `try_interval`, `max_try_count_0..4` = 5, total 8), verified directly against `outplans/id_dial_info.yaml` | MINOR | Row 12 corrected to "(8 fields)"; headline §1 tally unaffected (row 12 was already correctly counted as 1 of the 31 strict endpoints regardless of its internal field count) |
| 2 | §4 Phase 4 (bin-queue-manager + bin-route-manager) omitted the "two separate PRs, per this monorepo's one-PR-per-repo-boundary rule" clarification that Phase 5 and Phase 6 both state explicitly for their own multi-service groupings — inconsistent signaling could mislead a future implementer into planning one cross-service PR for Phase 4 | MINOR | Phase 4 row's rationale text updated to state the same two-separate-PRs caveat |
| 3 | §2 row 20 (`service_agents/me`) used a "LOW-MEDIUM" risk-tier label outside the four-value vocabulary (HIGH/MEDIUM/LOW/N/A-exempt) used everywhere else in §2, and §4 Phase 7's rationale echoed the same ad hoc label — no phase-ordering guidance existed for what a hybrid tier means | MINOR | Row 20 changed to "MEDIUM (self-service, agent-visible)" with justification text explaining why it sorts as MEDIUM despite not being credential/routing-critical; Phase 7's rationale reworded to state the LOW-to-MEDIUM range per endpoint instead of inventing a new tier label |

Not changed (reviewer confirmed correct via full independent re-derivation
of every input number, not just the final arithmetic): the 66 total /
31 strict / 3 hybrid / 32 already-partial figures (all four independently
recomputed from the YAML, including resolving the four `$ref`-based
schemas and confirming `ais/id.yaml` and `webchat_widgets/id.yaml` are
correctly bucketed as already-partial); essentially all 34 §2 rows'
required-field lists and owning services (spot-checked beyond the
required minimum); no other risk-tier row found inconsistent with its own
justification; §3.1/§3.2/§4/§2's 34-row tally reconfirmed independently;
§7 and §8's disposition claims all verified accurate against current
§§1-6 text.
