# Analysis: MCP server revocation gap + docs/UI correctness (v2)

Date: 2026-09-27
Author: CPO (Hermes)
Status: v2, Analysis Review Loop round 2 pending
Supersedes: `2026-09-27-mcp-server-customer-gap-analysis-v1-superseded.md`
(v1 §2.4 was factually wrong; retrospective in §7)

## 1. Issue statement

Three defects in the shipped customer-configured MCP tool integration, in
descending severity:

- **D1 (behavior, security-relevant):** deleting an MCP server does NOT revoke
  its tools. A soft-deleted server keeps `status = "active"`, is still returned by
  `McpServerGet`, and is still resolved and dispatched by the AI tool path
  indefinitely, using the stored encrypted secret. The shipped square-admin
  delete dialog explicitly promises the opposite.
- **D2 (published docs, correctness):** `ai_struct_mcpserver.rst` documents the
  registration endpoint as `/mcp_servers` (underscore). The real path is
  `/mcpservers`. A customer copying the documented path gets a 404. The same file
  uses the correct path elsewhere, so it is internally inconsistent.
- **D3 (frontend parity):** `mcp_server_ids` is settable only on the AI detail
  page in square-admin, though the backend accepts it on `POST /ais` too.

## 2. Validity re-check (code-verified)

### 2.1 D1: revocation does not revoke. CONFIRMED.

Full chain, every link verified by reading the file:

| Step | File:line | Finding |
|---|---|---|
| Soft delete | `bin-ai-manager/pkg/dbhandler/mcpserver.go:163-182` | `McpServerDelete` sets ONLY `tm_update` + `tm_delete`. `status` untouched, stays `active` |
| DB read | same file `:53-59` | `mcpserverGetFromDB` builds `SELECT ... WHERE id = ?` with NO `tm_delete` predicate |
| DB read entry point | same file `:84-86` | `McpServerGet` delegates straight to `mcpserverGetFromDB` |
| Handler read | `pkg/mcpserverhandler/handler.go:93-107` | `Get` maps `ErrNotFound` to 404 but adds no `TMDelete` check |
| Tool resolution | `pkg/aicallhandler/mcp_tool.go:72-88` | Gates on `err != nil` and `server.Status != StatusActive` only |
| Tool dispatch | `pkg/aicallhandler/mcp_tool.go:151-168` | Gates on whitelist membership and `Status` only |
| Transport | `pkg/mcptoolhandler/client.go:190-196` (`ListTools`), `:213-219` (`CallTool`) | Both call `db.McpServerGet` directly, check NEITHER `Status` NOR `TMDelete` |
| Grep proof | `grep -rn 'TMDelete\|tm_delete'` over `pkg/aicallhandler/mcp_tool.go`, `pkg/mcpserverhandler/`, `pkg/mcptoolhandler/` (excl. tests) | **zero matches** |
| Prune on delete | `grep -rn 'McpServerIDs' pkg/ --include=*.go` (excl. tests/mocks) | no code path prunes a deleted server's id from any `ai.mcp_server_ids` |

Two distinct consequences:

1. **Stale grant keeps working.** A deleted server still sits in some AI's
   `mcp_server_ids`, still passes `resolveTools`, so `ListTools` keeps being
   called against the customer's endpoint and `CallTool` keeps succeeding.
2. **A deleted server can be newly whitelisted.** `ValidateMcpServerIDs`
   (`pkg/aihandler/mcpserver_validation.go:37`) validates via `db.McpServerGet`,
   which ignores `tm_delete`. So `POST /ais` or `PUT /ais/{id}` carrying a deleted
   server's id is accepted as valid input.

**Contradicted UI promise:**
`square-admin/src/views/mcpservers/mcpservers_detail.js:496` delete dialog reads
"Any AI still whitelisting it will stop being able to call it." False today.

### 2.2 Where the gate must go, and where it must NOT go

**It must NOT go in `dbhandler.McpServerGet`.** Decisive reason:
`mcpServerHandler.Delete` (`pkg/mcpserverhandler/handler.go:221-245`) calls
`h.db.McpServerGet(ctx, id)` AFTER `McpServerDelete` in order to return the
deleted record and publish `EventTypeDeleted`. Filtering `tm_delete` at the
dbhandler layer would make `Delete` fail with `ErrNotFound` on its own success
path. `mcpServerHandler.Update` has the same read-back shape at `:210`.

Additionally `api-validator/tests/scenarios/test_mcpservers_lifecycle.py:244-274`
asserts `GET /mcpservers/{id}` returns 200 after DELETE, so the soft-delete
read-through is an intentional, tested API contract. Changing it is out of scope
and would break the validator.

**It must go on the TOOL CONSUMPTION path.** Candidate choke points:

| Site | File:line | Role |
|---|---|---|
| (a) resolution | `pkg/aicallhandler/mcp_tool.go:72-88` | stops a deleted server's tools from ever reaching the LLM |
| (b) dispatch | `pkg/aicallhandler/mcp_tool.go:151-168` | fails closed if a stale namespaced name is called anyway |
| (c) transport | `pkg/mcptoolhandler/client.go:190,213` | narrowest choke point; both (a) and (b) ultimately funnel through it |
| (d) whitelist write | `pkg/aihandler/mcpserver_validation.go:35-60` | rejects whitelisting an already-deleted server on POST/PUT `/ais` |

Recommendation: gate at **(c) + (d)** as mandatory, plus (a)/(b) for defense in
depth. Rationale: (c) is the single place every outbound MCP call passes through,
so it cannot be bypassed by a future caller; (a)/(b) give an early, well-logged
skip matching the existing best-effort pattern instead of surfacing a transport
error; (d) closes the input-validation hole. Exact placement and whether (a)/(b)
reuse the existing `Status` skip branch or get their own log line is a DESIGN
decision, deliberately not fixed here.

`Status` and `TMDelete` are semantically different and both are needed:
`disabled` is a reversible customer choice, `tm_delete` is terminal. A gate on one
does not imply the other.

### 2.3 D2: published docs give a 404 path. CONFIRMED.

`bin-api-manager/docsdev/source/ai_struct_mcpserver.rst`:

- `:29` `POST /mcp_servers`, `GET /mcp_servers` — wrong
- `:45` `POST /mcp_servers`, `PUT /mcp_servers/{id}` — wrong
- `:49` `PUT /mcp_servers/{id}` — wrong
- `:35`, `:100` `POST /mcpservers/oauth/...` — correct, hence internally inconsistent

Authoritative paths, `bin-openapi-manager/openapi/openapi.yaml:8569-8578`: five
routes, all `/mcpservers`.

Two further defects in the same file:
- `:29` `:ref:` `mcp_server_ids <ai-struct-ai-tool_names>` points at the
  `tool_names` anchor (`ai_struct_ai.rst:340`). No `mcp_server_ids` anchor exists
  anywhere in the tree.
- `:100` self-references `oauth_vendor <mcpserver-struct-mcpserver-mcpserver>`,
  the containing section rather than a field anchor.

### 2.4 What the docs ALREADY have (correcting v1's false claim)

v1 claimed no MCP docs exist. Wrong: a case-sensitive `grep mcp` missed uppercase
`MCP`. Verified present on `origin/main`:

- `ai_struct_mcpserver.rst` — 101 lines: wire-field table, Status enum table,
  Auth Type enum table, implementation notes, worked example. Built and tracked at
  `docsdev/build/html/ai_struct_mcpserver.html`.
- `ai.rst:15` — already in the AI toctree, so reachable from `index.rst:83`.
- `ai_struct_ai.rst:39,69` — `mcp_server_ids` documented, INCLUDING the
  `mcp_<8-hex>_<tool>` namespacing and the non-active-server silent-skip.
- `grep -ric mcp *.rst` → `ai_overview.rst:2`, `ai.rst:1`, `ai_struct_ai.rst:2`,
  `ai_struct_mcpserver.rst:28`. Four files, not one.

So the CLAUDE.md `*_struct_*.rst` sync obligation IS already satisfied; v1's
"unmet mandatory requirement" framing is withdrawn. What is genuinely missing is
narrative coverage: no `mcpserver_overview.rst`, no `mcpserver_tutorial.rst`, no
endpoint table. (`rag` has both overview and tutorial; MCP has neither.)

### 2.5 Runtime semantics (for doc accuracy)

`pkg/aicallhandler/mcp_tool.go`: prefix const `mcp_` at `:21`; `mcpServerIDShort`
at `:114-120` strips dashes and takes the first 8 hex chars; `resolveTools` at
`:52` appends MCP tools after built-ins; per-server best-effort skip on `Get`
failure `:73-77`, non-active `:79-82`, `ListTools` failure `:84-88`;
`toolHandleMcpCall` at `:127-178` fails closed. Reverse map key
`MetaKeyMcpToolMap = "mcp_tool_map"` at `models/aicall/main.go:67`. Config
`mcp_tools_list_cache_ttl_seconds` default 60 and `mcp_tool_call_timeout_seconds`
default 10 at `internal/config/main.go:149-150`.

### 2.6 Wire contract (for doc accuracy)

Source of truth is `bin-ai-manager/models/mcpserver/webhook.go:14-45`, NOT the
internal struct. 13 fields: `id`, `customer_id` (embedded
`commonidentity.Identity`), `name`, `detail`, `url`, `status`, `auth_type`,
`api_key_header`, `oauth_vendor`, `has_secret`, `tm_create`, `tm_update`,
`tm_delete`. Never document `SecretCiphertext`, `SecretNonce`, `KeyVersion`.

`auth_type` deliberately omits `,omitempty` (`:32`) because `AuthTypeNone` is the
empty string and is meaningful, so the field is ALWAYS present on the wire.
`oauth_vendor` DOES carry `,omitempty` (`:38`), so it is ABSENT, not empty, when
unset — `ai_struct_mcpserver.rst:37` says "empty otherwise", which is imprecise,
and the example block at `:60-73` omits the field entirely.

Enums, `models/mcpserver/main.go:13-45`: `auth_type` = `""` | `bearer` |
`api_key` | `oauth`; `status` = `active` | `disabled`. OAuth vendors wired today:
GitHub and Linear only (`internal/config/main.go:151-154`).

### 2.7 D3: square-admin gap. CONFIRMED, and it is frontend-only.

Backend accepts `mcp_server_ids` on BOTH create and update, verified at four
layers:

- `bin-openapi-manager/openapi/paths/ais/main.yaml:93-102` declares it on the
  POST body
- `bin-openapi-manager/gens/models/gen.go` `PostAisJSONBody.McpServerIds
  *[]string` (:10609), `PutAisIdJSONBody.McpServerIds` (:10667)
- `bin-api-manager/server/ais.go` POST converts at `:74-80`, PUT at `:293-299`
- `bin-ai-manager/pkg/listenhandler/v1_ais.go` POST validates+writes at
  `:117-125`, PUT at `:272-280`

=> **no backend or OpenAPI change needed for D3.**

Frontend hits for `mcp_server_ids` / `mcpServerIds` exist ONLY in
`square-admin/src/views/ais/ais_detail.js` (:103 state, :242 hydrate, :374 list
fetch, :382 toggle, :421 PUT body, :552-553 dirty check, :1238-1276 UI) plus its
test file. Zero hits in:

- `views/ais/ais_create.js` (438 lines; `body` at :114, POST at :154)
- `views/ais/AIEngineFields.js` (659 lines; the SHARED component both pages use)
- `views/teamgraph/sidebar.js` (1769 lines; AI create body :514-534, AI edit body
  :753-782)
- `src/types/api.ts` (`tool_names?` at :503, no MCP field)

`square-admin/CLAUDE.md:161-181` Field Sync Points names three required update
sites; MCP satisfies one. `src/types/api.ts` is a de-facto FOURTH site (the
checklist's step 1 says "Add type to src/types/api.ts") but is absent from the
table — the table itself needs a row.

### 2.8 Other surfaces enumerated (v1 omitted these)

| Surface | Finding |
|---|---|
| api-validator, repo `~/gitvoipbin/monorepo-monitoring` (NOT `monorepo/monitoring`, which holds only CLAUDE.md + Grafana dashboards) | `tests/scenarios/test_mcpservers_lifecycle.py` + `test_mcpservers_oauth.py` cover mcpserver CRUD. `grep -rn mcp_server_ids` hits only a docstring at `test_mcpservers_lifecycle.py:36` — ZERO assertions that `POST /ais` or `PUT /ais/{id}` accepts/persists/rejects `mcp_server_ids`, despite `test_ai_lifecycle.py` and `test_ai_errors.py` existing. §2.7's proof rests on Go unit tests with no end-to-end coverage |
| `square-main/public/skill.md`, `llms.txt` | `skill.md:414` / `llms.txt:49` enumerate AI creation fields incl. `tool_names`, no `mcp_server_ids`. Separately `skill.md:774-795` / `llms.txt:11` document "MCP Server" meaning `uvx voipbin-mcp` (VoIPBin AS an MCP server for Claude Code/Cursor) — the SAME terminology collision as `ai_overview.rst:681`, on a second customer-visible surface |
| `ai_overview.rst:677-685` | The existing "External AI Agent Integration → MCP Server" block links `github.com/nrjchnd/voipbin-mcp`, a third-party fork, while `skill.md:794` links the official `github.com/voipbin/mcp` |
| `openapi.yaml:2240-2245` | The `mcp_server_ids` description says namespacing is `mcp_<server_id_prefix>_<tool_name>` without stating 8 hex chars, and says nothing about best-effort skip. Redoc is customer-facing, so this is an editable surface, not just evidence |
| `square-talk`, `square-meet`, `square-dev`, `square-admin_new` | Checked: zero MCP / AI-tool-config references. Genuinely clear |

## 3. Proceed decision

### PROCEED. Priority order (revised from v1):

1. **D1 + D2 together, one PR in `voipbin/monorepo`.** One logical unit: "MCP
   lifecycle semantics are wrong in code and mis-documented." D2's fix text
   depends on D1's outcome (what the Status/deletion section says). Per root
   CLAUDE.md, one PR per logical change per repo.
2. **D3, one PR in `voipbin/monorepo-javascript`.** Includes re-verifying the
   delete-dialog copy (D1 makes it true), the `types/api.ts` field, and the
   CLAUDE.md Field Sync Points table amendment.
3. **Narrative docs (`mcpserver_overview` / `_tutorial`)** — same PR as (1) if it
   stays reviewable, otherwise a follow-up. Decide at design time.

### CEO decision locked 2026-09-27

Option A: **soft-delete MUST revoke tool access.** Verbatim: "A로 가자. 당연히
삭제했으면 차단하는게 맞아." So D1 is a backend fix, not a docs-only
reinterpretation, and the shipped UI copy becomes true rather than being
rewritten.

### Why this is not overengineering

D1 is a correctness/security fix on a shipped feature with a concrete,
code-verified failure mode. D2 is a published 404. D3 completes an
already-shipped capability. None introduces new infrastructure or speculative
capacity.

### Non-goals

- **Tool-level (per-tool) whitelisting.** `resolveTools:91-105` adds every tool a
  server returns, no per-tool filter. Deferred: no demand signal, and
  `mcp_server_ids []uuid.UUID` is additively extensible so a future
  `mcp_tool_allowlist` can land without breaking it. **The deferral is not
  consequence-free:** documenting server-level-only commits us to the property
  that whitelisting a server grants all present AND FUTURE tools that server
  exposes, i.e. a third party can silently expand its own scope. Mitigation is one
  explicit sentence in the docs, not building the feature.
- Changing the soft-delete READ contract (`GET` after `DELETE` returns 200).
  Intentional and validator-tested (§2.2).
- Additional OAuth vendors beyond GitHub/Linear.
- In-app help content (`navHelpManifest.js:39`, explicit `helpTopicId: null`).
- Resolving the `uvx voipbin-mcp` vs customer-MCP-server naming collision
  platform-wide. Flag it, decide separately (§4 Q4).

## 4. Decisions to lock before the design doc

1. **Gate placement for D1.** Recommend (c) transport + (d) whitelist-write as
   mandatory, (a)/(b) as defense in depth. Confirm; and confirm whether a deleted
   server is skipped silently (matching the existing non-active pattern) or logged
   at WARN.
2. **Does D1's PR also prune stale ids out of `ai.mcp_server_ids` on delete?**
   Recommend NO: gating at consume time is sufficient, idempotent, and avoids a
   fan-out write across every AI of that customer on every delete. The stale id
   becomes inert. But the UI should not render an id that resolves to nothing —
   needs a display decision in D3's PR.
3. **`/mcp_servers` → `/mcpservers`:** fold into D1's PR (recommended; a 3-line
   docs fix) or ship standalone immediately?
4. **Naming collision** (`uvx voipbin-mcp` = VoIPBin-as-MCP-server vs
   customer-registered MCP server): rename the `ai_overview.rst:681` heading only,
   or also `skill.md`/`llms.txt`? And should `ai_overview.rst:685` keep pointing at
   the third-party fork rather than `github.com/voipbin/mcp`?
5. **api-validator coverage** for `mcp_server_ids` on `POST /ais` and for D1's
   revocation behavior: add now in `monorepo-monitoring` (a THIRD repo, third PR —
   the 2-PR plan does not account for it) or defer?
6. **Does `square-admin/CLAUDE.md` Field Sync Points gain `src/types/api.ts` as a
   fourth row?** Recommend yes.

## 5. Risk table

| Risk | Severity | Mitigation |
|---|---|---|
| Gate placed in `dbhandler.McpServerGet` breaks `Delete`/`Update` read-back and the validator's GET-after-DELETE contract | High | §2.2 forbids it explicitly; design doc must state the chosen sites; run the full api-validator mcpserver suite |
| D1 fix breaks a customer whose AI depends on a deleted-but-working server | Medium | That dependency IS the bug. Deletion is customer-initiated and the UI already promised revocation. No migration needed, but call it out in the PR body |
| Documenting an endpoint path that 404s (already shipped, live now) | High | D2 |
| `mcp_server_ids` grants all present and FUTURE tools of a server | Medium | One explicit docs sentence (§3 non-goals) |
| Coverage for `mcp_server_ids` on `POST /ais` is Go-unit-only, no end-to-end | Medium | §4 Q5 |
| Field Sync Points drift recurs | Low | Update all four sites + amend the table in D3's PR |
| Teamgraph AI panels are inline, not the shared component | Low | Accept the duplication to match the file's established pattern; do not refactor mid-task |
| D3 lands in the dead nav/layout generation | Low | No nav change needed; `/resources/mcpservers/*` routes already live (`routes.js:333-336`, `_nav.js:239`). Changes are inside existing AI forms only |

## 6. Verification plan (per PR)

D1+D2 (`monorepo`): `go mod tidy && go mod vendor && go generate ./... && go test
./... && golangci-lint run -v --timeout 5m` in `bin-ai-manager`; new unit tests
proving a soft-deleted server is skipped in resolution, refused in dispatch,
refused in transport, and rejected by `ValidateMcpServerIDs`; clean Sphinx
rebuild; `grep -n 'mcp_servers' docsdev/source/*.rst` returns nothing.

D3 (`monorepo-javascript`): baseline-vs-branch test comparison per CLAUDE.md test
gate; `npm run build`; production build served and verified in a real browser
(reviews and RTL do not catch unmount races).

## 7. Retrospective: why v1 was wrong

v1 §2.4 asserted "exactly ONE grep hit" and "no `mcpserver*.rst` files exist."
Both false. Root cause: a case-sensitive `grep mcp` that missed uppercase `MCP`,
and the result was written into the analysis as evidence without re-running it
case-insensitively. The entire v1 priority ordering rested on that one unverified
claim, and both independent reviewers caught it. Preserved here rather than
silently rewritten so the failure mode stays visible.

Lesson for the loop: the section nobody thinks to re-verify is the one the whole
recommendation rests on. Always use `grep -i` for convention/content greps.
