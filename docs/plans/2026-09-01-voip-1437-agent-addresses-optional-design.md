# VOIP-1437: Make agent `addresses` optional again on `POST /agents`

## Background

`bin-api-manager/server/agents.go:114-121` rejects `POST /agents` with `400 INVALID_ARGUMENT` when
`addresses` is an empty array. This was added by VOIP-1307 (commit `3be80108d`, 2026-08-05) as a
mechanical fix to make 4 endpoints' handler code match their OpenAPI spec's `required` lists
(Gin's `BindJSON` doesn't enforce presence on slice/string fields, so a spec-required field can be
silently accepted empty unless the handler checks explicitly).

User-reported issue: chat-only agents don't need a dial address (phone/SIP/extension), and this
validation blocks creating them entirely.

## Update after round-1 design review (blocking items resolved empirically, not just argued)

Round 1 review flagged two blocking gaps, both resolved by actually running the change, not by
further argument:

1. **The "oapi-codegen regeneration blocker."** `server/outdials.go`'s VOIP-1307 comment
   (`:341-343`) says relaxing a `required` field is "tracked in VOIP-1308 to relax the spec once
   the oapi-codegen regeneration blocker is resolved" — but that comment is about
   `outdials/{id}/targets`'s 5-sibling-required-fields case, which genuinely has no clean
   declarative expression (needs "at least 1 of 5", which is `oneOf`/`anyOf` territory, not
   solvable by just deleting entries from `required`). It is **not** a generic blocker on removing
   any field from `required`. Proof: `docs/plans/2026-06-15-openapi-optional-flow-fields-design.md`
   already did exactly this for `POST /groupcalls`'s `flow_id`/`actions` and `POST
   /transcribes`'s `on_end_flow_id` — implemented in `f5298304e`
   ("NOJIRA-Fix-openapi-optional-flow-fields", #992) — and it worked, producing pointer-typed
   generated fields with nil-checks in the handler (`server/groupcalls.go:51-52`,
   `if req.FlowId != nil { ... *req.FlowId }`). The closer precedent is the very next lines,
   `:54-58` — `actions` is an array field made required → optional, handled with
   `if req.Actions != nil { for _, v := range *req.Actions { ... } }` — structurally identical to
   this ticket's `addresses` change (single array field, required → optional, no cross-field "at
   least one of N" semantics), not the outdials
   shape.
   - Empirically confirmed in this session: edited `paths/agents/main.yaml`, ran
     `bin-openapi-manager`'s `go generate ./...` (succeeds), ran `bin-api-manager`'s
     `go generate ./...` (succeeds — see note on redoc bundler warnings below), and the resulting
     diff is exactly `Addresses []CommonAddress` → `Addresses *[]CommonAddress` in both
     `bin-openapi-manager/gens/models/gen.go` and `bin-api-manager/gens/openapi_server/gen.go`. No
     blocker encountered.
   - The redoc generation sub-step (`config_redoc/generate.go`, a *separate* `go:generate` line
     from the actual server-type codegen) prints ~2373 "Can't resolve $ref" warnings from
     `@redocly/cli` when bundling per-path YAML files standalone, then finishes anyway ("Errors
     ignored because of --force") and still produces `gens/openapi_redoc/api.html` /
     `openapi.json`. This is unrelated to the `addresses` change: the warnings come from
     `@redocly/cli`'s own resolution of unrelated `$ref`s across the whole spec tree (e.g.
     `id_tag_ids.yaml`'s `400`/`401` response refs, per the sample output above), not from
     anything this ticket edited, and the `--force` flag means the tool already treats this as a
     known, tolerated condition rather than a hard failure — not the "blocker" the outdials
     comment refers to (that one is about the server-type codegen actually failing to produce a
     type, which did not happen here).
2. **Scope 2 needed the pointer dereference fix**, confirmed by the actual compile: after
   `addresses` leaves `required`, `req.Addresses` becomes `*[]CommonAddress`, so
   `server/agents.go:108-113`'s `for _, v := range req.Addresses` no longer compiles as-is — it
   needs `range *req.Addresses` (guarded by the existing `req.Addresses != nil` check, unchanged).
   `go build ./...` in `bin-api-manager` is clean after this one-line fix.

Also incorporated: `bin-openapi-manager/CLAUDE.md` rule 5 ("`minItems: 1` on required arrays") was
never satisfied for `addresses` even while it was required (no `minItems` in the schema,
confirmed) — meaning VOIP-1307's guard enforced a stronger constraint than the spec itself ever
declared. This is an additional, independent argument for reverting: the guard over-enforced
relative to the spec's own authoring rules, not just relative to actual product need.

RST citation corrected: the accurate evidence isn't `agent_tutorial.rst:49` (that's the *response*
body's `"addresses": []`) but `:26-37`, the *request* example, which omits the `addresses` key
entirely — meaning the documented example request would have been rejected with 400 for the
entire 4-week VOIP-1307 window. `:13`'s "(For address assignment) Contact addresses..." phrasing
already reads as conditional/optional, reinforcing the same conclusion.

RST sync (per this repo's mandatory rule for user-visible behavior changes): added one sentence to
the existing "AI Implementation Hint" note in `agent_tutorial.rst` stating `addresses` is optional
and explaining the chat-vs-voice-queue implication, then did a clean Sphinx rebuild
(`rm -rf build && python3 -m sphinx -M html source build` — succeeded). `build/` is force-added
per the RST sync rule.

Scope of investigation, stated explicitly (round-1 review noted this wasn't bounded before):
this ticket investigated `bin-agent-manager`, `bin-api-manager`, `bin-queue-manager`,
`bin-call-manager`, and `bin-talk-manager`/`bin-conversation-manager`/`bin-webchat-manager` inside
this monorepo. Frontends (`square-admin`, `square-talk`) and out-of-repo SDKs (`voipbin-go`,
`python-sdk`) were not inspected. Risk is judged low for frontends (the server becomes strictly
more permissive; a client that always sends `addresses` is unaffected) but the Go SDK — generated
from the same `openapi.yaml` — will see the same `[]CommonAddress` → `*[]CommonAddress` source-level
type change on its request struct, a source-compatibility break for any SDK consumer that
constructs that struct field directly. Flagged here, not fixed (out of repo, no access from this
session) — worth a heads-up to SDK maintainers when this ships.

## Root cause of the underlying requirement (investigated, not assumed)

- `addresses` has been in this endpoint's `required` list since the very first commit that created
  this spec file — `27d1402c7` ("VOIP-966-Add_openapi_spec", 2025-01-26, verified via
  `git log --follow --diff-filter=A` and `git show 27d1402c7:.../agents/main.yaml`; the file lived
  at `bin-api-manager/openapi/paths/agents/main.yaml` until `91c41b73e`
  ("VOIP-975-Add_openapi_manager") moved the whole spec tree into the now-dedicated
  `bin-openapi-manager` service) — about 18.3 months before VOIP-1307 (`3be80108d`, 2026-08-05). No
  commit message or code comment documents a business reason; it reads as a blanket "mark all the
  fields required" choice made during initial scaffolding, not a deliberate product decision.
- VOIP-1307 did not revisit that choice — it only made 4 handlers (including this one) match
  whatever their spec already said, explicitly as a spec-compliance pass. Tellingly, for 2 of the
  other 3 endpoints in that same commit, the author *did* deviate from a literal reading of the
  spec where it didn't match the domain's actual needs (`outdials/{id}/targets`: spec lists 5
  required destination fields, commit accepts "at least 1 of 5" instead, with a comment noting
  "the spec's `required` list overstates the actual constraint"). `addresses` didn't get that same
  scrutiny at the time.

## Why chat-only agents genuinely don't need this

- `bin-talk-manager` (the chat/conversation backend) associates an agent with a chat via
  `Participant.owner_id` (`owner_type="agent"`) — a direct agent-ID reference. It never reads
  `addresses`.
- `bin-conversation-manager` and `bin-webchat-manager` have no agent concept at all.
- `addresses` is exclusively a voice-routing concern: `bin-call-manager`'s
  `getDialDestinationsAddressAndRingMethodTypeAgent` (`pkg/groupcallhandler/dial.go:218-251`)
  fetches the agent and returns `ag.Addresses` as the literal dial targets for `ring_method`
  `ringall`/`linear`. An agent that's never assigned to a voice queue never has this code path
  invoked at all.

## Why reverting is safe (investigated, not assumed)

- `PUT /agents/{id}/addresses` already allows replacing an agent's addresses with an empty array
  (`bin-api-manager/server/agents.go:289-294` has no length check) — an addressless agent is
  already a reachable state in production today, created via a path VOIP-1307 never touched.
  Reverting the `POST` guard doesn't introduce a new class of state, it just allows reaching an
  already-possible state through a second door.
- No code in `bin-agent-manager`, `bin-api-manager`, `bin-queue-manager`, or `bin-call-manager`
  indexes into `Agent.Addresses` assuming at least one element (`Addresses[0]` etc.) — every
  consumer either ranges over it or checks length first. Reverting cannot introduce a panic.
- The domain layer (`bin-agent-manager/pkg/agenthandler/agent.go`'s `Create`, and the DB layer
  `pkg/dbhandler/agent_address.go:54`) never required non-empty `addresses` in the first place —
  the *only* enforcement anywhere in the stack is the 8-line guard this ticket removes.
- The RST tutorial's create-agent request example (`agent_tutorial.rst:26-37`) has never included
  an `addresses` key at all — meaning that exact documented example has been rejected with `400`
  for the entire ~4-week VOIP-1307 window, silently inaccurate. Reverting makes the existing
  example work as documented again. (RST is still updated in this ticket — see Scope — to add an
  explicit sentence about the optionality and its voice-queue implication; "the example already
  matches the new behavior" is why no *correction* was needed, not a reason to skip RST entirely.)

## Scope

1. `bin-openapi-manager/openapi/paths/agents/main.yaml`: remove `addresses` from `POST /agents`'s
   request body `required` list. (Spec-first, per this repo's OpenAPI-first workflow — the handler
   change follows from the spec change, not the other way around.)
2. `bin-api-manager/server/agents.go`: remove the 8-line guard (`:114-121`) added by VOIP-1307,
   **and** change `for _, v := range req.Addresses` to `for _, v := range *req.Addresses` (guarded
   by the existing `req.Addresses != nil` check) — required because `req.Addresses` becomes
   `*[]CommonAddress` once `addresses` leaves `required` (confirmed via actual codegen, not
   assumed).
3. Regenerate: `cd bin-openapi-manager && go generate ./...`, then
   `cd bin-api-manager && go generate ./...` (regenerates `gens/openapi_server/gen.go` from the
   updated spec). Confirmed via actual run: only `Addresses`'s type changes
   (`[]CommonAddress` → `*[]CommonAddress`, `omitempty` added), no unrelated churn.
4. `bin-api-manager/server/agents_test.go`: the two test cases VOIP-1307 added
   (renamed `"empty body -- addresses optional, defaults to empty"` and
   `"empty addresses array is accepted"`, both previously expecting `400`) updated to expect `200`
   with `AgentCreate` called with an empty `[]commonaddress.Address{}` (not `nil` — gomock's
   default matcher is `reflect.DeepEqual`, so `expectedAddresses`/`expectedTagIDs` must both be
   empty-slice-typed, matching what the handler already produces for `tag_ids` today).
5. `bin-api-manager/docsdev/source/agent_tutorial.rst`: add one sentence to the existing "AI
   Implementation Hint" note stating `addresses` is optional and that an addressless agent can't
   receive voice-queue calls until one is added — per this repo's mandatory RST-sync rule (this is
   a user-visible `400` → `200` behavior change). Clean Sphinx rebuild
   (`rm -rf build && python3 -m sphinx -M html source build`), force-add `build/`.
6. Full verification workflow for `bin-openapi-manager` and `bin-api-manager` per this repo's
   `CLAUDE.md`: `go mod tidy && go mod vendor && go generate ./... && go test ./... &&
   golangci-lint run -v --timeout 5m`, in both service directories. Confirmed via actual run: both
   clean (0 lint issues, all tests pass, including the 3 updated `Test_PostAgents` subtests).

## Explicitly out of scope (confirmed with the user)

- `bin-call-manager`'s asymmetric silent-failure bug (an agent with zero addresses assigned to a
  `ringall`/`linear` voice queue produces no error and no dialed leg — extension destinations get
  an explicit error in the same function, agent destinations don't) is tracked separately as
  [VOIP-1436](https://voipbin.atlassian.net/browse/VOIP-1436). This ticket makes addressless agents
  a normally-created, supported state instead of an edge case only reachable via `PUT`, so
  VOIP-1436's failure mode becomes meaningfully more likely to occur in practice, not just
  marginally — the user explicitly chose to scope it out of this ticket and file it separately, but
  its priority is worth revisiting once this ships.
- The other 3 endpoints VOIP-1307 touched (`outdials`, `outdials/{id}/targets`,
  `campaigns/{id}/next_campaign_id`) are unrelated and untouched by this ticket.
- Pre-existing, unrelated spec/handler mismatch noticed while editing this exact schema block:
  `paths/agents/main.yaml:83` declares the success response as `'201'`, but the handler returns
  `c.JSON(200, res)` (matching this ticket's and the existing test's `200` expectation). Not fixed
  here to avoid scope creep on an unrelated pre-existing inconsistency — noted for a separate
  follow-up if it's worth fixing.

## Acceptance criteria

- `POST /agents` with no `addresses` field, and with `addresses: []`, both succeed (`200`),
  creating an agent with zero addresses.
- `POST /agents` with a populated `addresses` array continues to work unchanged.
- `go test ./...` passes in both `bin-openapi-manager` and `bin-api-manager`.
- `golangci-lint run` is clean in both.
- No other test in the monorepo asserts on the removed 400 behavior (checked via repo-wide grep
  for `addresses is required` / `addresses.*must not be empty` during implementation).
