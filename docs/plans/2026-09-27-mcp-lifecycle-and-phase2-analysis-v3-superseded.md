# Analysis: MCP server lifecycle correctness + Phase 2 tool exposure (v3)

Date: 2026-09-27
Author: CPO (Hermes)
Status: v3, Analysis Review Loop round 3 pending
Supersedes:
- `2026-09-27-mcp-server-customer-gap-analysis-v1-superseded.md` (§2.4 factually wrong)
- `2026-09-27-mcp-server-revocation-and-docs-analysis-v2-superseded.md` (D1 severity overstated; missed F1/F2/F3)

## 0. CEO decisions locked 2026-09-27

Asked as Q1..Q4, answered verbatim:

| # | Question | Decision |
|---|---|---|
| Q1 | Implement the documented "MCP tools presented to the LLM" behavior, or correct the docs to match the code? | **Implement.** "Q1. 구현하자." |
| Q2 | How much of the lifecycle scope lands? | **All of it.** "Q2. 전부." |
| Q3 | Zero stored credentials on delete? | **Yes.** "Q3. 지우자. 그게 맞지." |
| Q4 | Dead `mcp_tools_list_cache_ttl_seconds` flag: implement or remove? | Clarified in §2.6 below; **remove the flag, no cache** (CPO recommendation, accepted: "좋아. 진행해.") |

PR split approved in the same turn: PR A (lifecycle+docs), PR B (Phase 2 tool
exposure), PR C (square-admin). Rationale in §3.

## 1. Issue statement

Seven defects plus one unimplemented-but-documented capability in the shipped
customer-configured MCP tool integration.

**Lifecycle correctness (PR A):**
- **D1** Soft-delete does not revoke tool access. A deleted server keeps
  `status = "active"`, every read path ignores `tm_delete`, so the AI tool path
  keeps making credentialed outbound `ListTools` calls to it forever.
- **D2** `McpServerUpdate` has no `tm_delete` predicate and never clears it, and
  `status` is PUT-settable, so **a deleted server is mutable and can be flipped
  back to `active` with a new URL and a new secret.**
- **D3** OAuth start/complete reach a deleted server; `Complete` has *no*
  ownership/existence re-check by explicit design comment, so tokens get written
  onto a `tm_delete`-populated row.
- **D4** `McpServerDelete` retains `SecretCiphertext`, `AccessTokenCiphertext`,
  `RefreshTokenCiphertext` indefinitely.
- **D5** Published docs give a 404 endpoint path (`/mcp_servers` vs `/mcpservers`)
  plus two broken `:ref:` targets.
- **D6** `mcp_tools_list_cache_ttl_seconds` is a dead flag; the design doc
  justifies a *separate* design decision by citing this nonexistent cache.

**Documented-but-unimplemented (PR B):**
- **D7** `ai_struct_ai.rst:69` and `openapi.yaml:2242-2243` promise that a
  whitelisted server's tools are merged into the AI's tool list and presented to
  the LLM. **No code path does this.** The merged list `resolveTools` returns is
  discarded by all three callers.

**Frontend parity (PR C):**
- **D8** `mcp_server_ids` is settable only on the AI detail page, though the
  backend accepts it on `POST /ais` too.

## 2. Validity re-check (code-verified)

### 2.1 D1: revocation does not revoke. CONFIRMED.

| Step | File:line | Finding |
|---|---|---|
| Soft delete | `bin-ai-manager/pkg/dbhandler/mcpserver.go:163-182` | sets ONLY `tm_update` + `tm_delete`; `status` untouched, stays `active` |
| DB read | same `:53-59` | `Where(sq.Eq{"id"})` only, no `tm_delete` predicate |
| DB read entry | same `:84-86` | `McpServerGet` → `mcpserverGetFromDB` |
| Handler read | `pkg/mcpserverhandler/handler.go:93-107` | `Get` maps `ErrNotFound`→404, no `TMDelete` check |
| Resolution | `pkg/aicallhandler/mcp_tool.go:72-88` | gates on `err != nil` and `Status != StatusActive` only |
| Dispatch | `pkg/aicallhandler/mcp_tool.go:151-168` | whitelist + `Status` only |
| Transport | `pkg/mcptoolhandler/client.go:190-196` (`ListTools`), `:213-219` (`CallTool`) | both call `db.McpServerGet` directly; check NEITHER `Status` NOR `TMDelete` |
| Whitelist validation | `pkg/aihandler/mcpserver_validation.go:37` | validates via `db.McpServerGet`; a deleted server passes |
| Grep proof | `TMDelete\|tm_delete` over `pkg/aicallhandler/mcp_tool.go`, `pkg/mcpserverhandler/`, `pkg/mcptoolhandler/` | **zero matches**, even including tests |
| Prune on delete | `McpServerIDs` in `pkg/` (excl. tests/mocks) | 24 hits, none prunes a deleted id from `ai.mcp_server_ids` |

**Live exposure today (corrected from v2, see §2.5):** on every AIcall/insight
session start, `resolveTools` calls `ListTools` (`mcp_tool.go:84`) against every
whitelisted server including deleted ones, which decrypts the stored secret and
issues an authenticated outbound HTTP request to the customer's URL. That is the
real, ongoing defect. `CallTool` dispatch is currently unreachable in production
(§2.5), so v2's "tools keep being called" was an overstatement.

Contradicted UI promise:
`square-admin/src/views/mcpservers/mcpservers_detail.js:496` — "Any AI still
whitelisting it will stop being able to call it." False today.

### 2.2 D2: a deleted server is mutable and resurrectable. CONFIRMED.

- `pkg/dbhandler/mcpserver.go:146-149`: `sq.Update(mcpserverTable).SetMap(...)
  .Where(sq.Eq{"id": id.Bytes()})`. No `tm_delete` predicate. It sets `tm_update`
  and **never clears `tm_delete`**.
- `pkg/mcpserverhandler/handler.go:168-169`: `if status != nil { fields[FieldStatus] = *status }` — `status` is PUT-settable.
- Net: `PUT /mcpservers/{id}` on a deleted server succeeds, can set
  `status = active`, and can replace `url` and the secret.

**This retracts v2's claim that "`tm_delete` is terminal."** On the write paths it
is not, which is precisely why D2 must be fixed alongside D1: gating reads while
leaving writes open leaves a resurrection path.

### 2.3 D3: OAuth reaches a deleted server, and Complete has no re-check. CONFIRMED.

- `pkg/mcpoauthhandler/start.go:37` `h.db.McpServerGet(ctx, *mcpServerID)` — the
  reconnect ownership check. `McpServerGet` ignores `tm_delete`, so an OAuth
  reconnect can be **started** against a deleted server, creating a
  `mcpoauthstate` row for it.
- `pkg/mcpoauthhandler/complete.go:112-133` then calls `McpServerUpdate` on that
  id. The comment at `:113-115` states verbatim: "ownership was already verified
  in Start before the state row was created, so this is a plain update, not a
  second ownership check." So **there is no existence or deletion re-check at
  all**, and a server deleted *between* Start and Complete (window = `stateTTL`)
  receives fresh `access_token`/`refresh_token` ciphertext on a deleted row.
  A gate in `Start` alone does not close this; `Complete` needs its own.
- `pkg/mcpoauthhandler/access_token.go:37-112` `GetValidAccessToken` receives an
  already-fetched `*mcpserver.McpServer` from `pkg/mcptoolhandler/client.go:97`
  and persists rotated tokens via `McpServerUpdate:108`. It has NO independent
  read, so a transport gate at `client.go:191/214` also closes the refresh path —
  no separate gate needed there. But until that lands, `ListTools` against a
  deleted OAuth server keeps refreshing and re-persisting vendor tokens on it.
- **Verified clean, do NOT over-gate:** the public unauthenticated
  `GET /mcpservers/oauth/callback` (`bin-api-manager/cmd/api-manager/main.go:310-321`)
  calls `CallbackExists` (`pkg/mcpoauthhandler/callback.go:17-31`), which touches
  only `McpOAuthStateGet` and never `McpServer`. No deleted-server exposure.

### 2.4 D4: credentials survive delete. CONFIRMED.

`McpServerDelete` (`pkg/dbhandler/mcpserver.go:163-182`) writes only `tm_update`
and `tm_delete`. `SecretCiphertext`, `SecretNonce`, `AccessTokenCiphertext`,
`AccessTokenNonce`, `RefreshTokenCiphertext`, `RefreshTokenNonce`, `KeyVersion`
all remain on the row indefinitely. Per Q3 these must be zeroed on delete.

Open sub-question for design (§4 Q2): zeroing is irreversible and interacts with
the validator's GET-after-DELETE-returns-200 contract — `has_secret` will flip to
false on the returned deleted row. Confirm that is acceptable (it is truthful, and
arguably more correct than reporting a secret that is about to be unusable).

### 2.5 D7: the merged tool list is DISCARDED. CONFIRMED — this reframes severity.

`resolveTools` (`pkg/aicallhandler/mcp_tool.go:52`) returns
`([]tool.Tool, map[string]aicall.McpToolRef, error)`. **All three production
callers throw away the first return value:**

```
pkg/aicallhandler/start.go:1169            _, mcpToolMap, errTools := h.resolveTools(ctx, a)
pkg/aicallhandler/insight_session.go:225   _, mcpToolMap, errTools := h.resolveTools(ctx, a)
pkg/aicallhandler/mcp_tool.go:284          _, mcpToolMap, err := h.resolveTools(ctx, a)
```

`grep -rn 'resolveTools' pkg/ --exclude=*_test.go` returns exactly those three
call sites plus the definition and comments. Nothing consumes the `[]tool.Tool`.

The tool list actually sent to a model is built elsewhere and knows nothing about MCP:

- **pipecat (realtime voice):** `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:150`
  `tools = h.toolHandler.GetByNames(ai.Type, ai.ToolNames)`; that handler
  (`bin-pipecat-manager/pkg/toolhandler/main.go:91-108`) iterates only `h.tools`,
  loaded from the static `toolDefinitions` via `AIV1ToolList`. The list is handed
  to `h.pythonRunner.Start(...)` at `runner.go:174-189`.
  `grep -rni 'mcp' bin-pipecat-manager/ --include=*.go` (excluding `vendor/`) =
  **zero matches**.
- **non-pipecat OpenAI engine:** `pkg/engine_openai_handler/streaming_send.go:50`
  sends the package-level static `tools` var, which is
  `[]openai.Tool{toolConnect, toolMessageSend}` (`tool.go:8-11`). Two tools, both
  built-in.

**Consequence:** the LLM is never told any `mcp_<8hex>_<tool>` name exists, so it
cannot emit one, so `toolHandleMcpCall` is unreachable in production.

This is per design, not an accident: `docs/plans/2026-09-11-mcp-tool-integration-design.md:72-74`
explicitly non-goals the realtime path ("deferred to Phase 2") and `:290-293`
states pipecat is left untouched by construction.

**But the docs shipped ahead of the code.** `ai_struct_ai.rst:69` publishes that
tools are namespaced "when presented to the LLM", and `openapi.yaml:2242-2243`
says "The server's tools are merged into this AI's tool list." Both describe
behavior no code path performs. Per Q1 the resolution is to implement, not to
retract the docs.

**Good news for PR B scope — the receiving half is already complete:**
`pkg/aicallhandler/tool.go:141-142` already routes any unknown tool name carrying
the `mcpToolNamePrefix` to `toolHandleMcpCall`, and `toolHandleMcpCall` re-reads
live state rather than trusting stored metadata: `mcp_tool.go:144` re-resolves the
AI, `:151` re-checks whitelist membership against the fresh `McpServerIDs`, `:157`
re-fetches the server row, `:163` re-checks `Status`. So PR B's work is confined
to **advertising** the tools, and a `tm_delete` gate placed at resolution/dispatch
(sites (a)/(b)) does correctly stop in-flight calls.

**Ordering consequence:** PR B is what makes `CallTool` reachable. Landing it
before PR A's gates would turn today's `ListTools`-only exposure into a live
"deleted server's tools are callable by the LLM" exposure. **PR A must precede
PR B.** This is the ordering dependency justifying the split.

### 2.6 D6: `mcp_tools_list_cache_ttl_seconds` is a dead flag. CONFIRMED.

Every reference, repo-wide (`--include=*.go --include=*.yaml`, excluding `vendor/`):

```
internal/config/main.go:84    struct field declaration
internal/config/main.go:149   f.Int(..., 60, "Redis cache TTL (seconds) for an MCP server's tools/list result")
internal/config/main.go:199   env mapping MCP_TOOLS_LIST_CACHE_TTL_SECONDS
internal/config/main.go:276   viper.GetInt(...)
```

Four hits, all inside config. **Zero consumers.** Contrast the sibling flag that
does work: `McpToolCallTimeoutSeconds` appears at `config/main.go:85`, `:277`, and
**`cmd/ai-manager/main.go:156`** where `NewMcpToolHandler(db,
cfg.McpSecretEncryptionKeys, cfg.McpToolCallTimeoutSeconds, mcpOAuthHandler)`
consumes it. `NewMcpToolHandler` (`pkg/mcptoolhandler/main.go:74`) has no cache-TTL
parameter at all. `grep -rni 'mcp' pkg/cachehandler/` = zero matches.
`pkg/mcptoolhandler/client.go:190-207` `ListTools` goes straight to DB + HTTP.

The design doc asserts the cache exists in three places:
`2026-09-11-mcp-tool-integration-design.md:466-467` ("Result cached in Redis, key
`mcp:tools:<server_id>`, short TTL"), `:705` (config table row), and most
damagingly **`:750-754`**, which justifies a *different* decision ("a second cache
layer over the `McpServer` row itself is not needed") by citing this nonexistent
cache as already-existing mitigation.

**Decision: remove the flag and correct the design doc. Do not implement the cache.**
Two reasons: (i) no measured performance signal — one call per session start, no
threshold-approach alarm, so building a cache layer now is speculative
infrastructure; (ii) a cache actively conflicts with the revocation requirement —
it would serve a deleted server's tools for up to the TTL after deletion,
requiring invalidation design on top of the gates. Its absence is an asset here.

### 2.7 D5: published docs give a 404 path. CONFIRMED.

`bin-api-manager/docsdev/source/ai_struct_mcpserver.rst`:
- `:29` `POST /mcp_servers`, `GET /mcp_servers` — wrong
- `:45` `POST /mcp_servers`, `PUT /mcp_servers/{id}` — wrong
- `:49` `PUT /mcp_servers/{id}` — wrong
- `:35`, `:100` correctly `/mcpservers/oauth/...` — internally inconsistent

Those three are the only `mcp_servers` hits in the whole RST tree. Authoritative
paths: `bin-openapi-manager/openapi/openapi.yaml:8569-8578`, five routes, all
`/mcpservers`.

Plus two broken references in the same file:
- `:29` `:ref:` `mcp_server_ids <ai-struct-ai-tool_names>` points at the
  `tool_names` anchor (`ai_struct_ai.rst:340`); no `mcp_server_ids` anchor exists
  anywhere in the tree.
- `:100` self-references `oauth_vendor <mcpserver-struct-mcpserver-mcpserver>`,
  the containing section rather than a field anchor.

### 2.8 What the docs ALREADY have (v1's false claim, retained as a guard)

`grep -ric mcp *.rst` → `ai_overview.rst:2`, `ai.rst:1`, `ai_struct_ai.rst:2`,
`ai_struct_mcpserver.rst:28`. Four files. `ai_struct_mcpserver.rst` is 101 lines
on `origin/main`, in the `ai.rst:15` toctree, reachable from `index.rst:83`, and
built to `docsdev/build/html/ai_struct_mcpserver.html`. The CLAUDE.md
`*_struct_*.rst` obligation IS satisfied. Genuinely missing: narrative coverage
(no `mcpserver_overview.rst`, no `mcpserver_tutorial.rst`; `rag` has both).

### 2.9 Runtime semantics (for doc accuracy)

`pkg/aicallhandler/mcp_tool.go`: prefix const `mcp_` at `:21`; `mcpServerIDShort`
`:114-120` strips dashes, first 8 hex; `resolveTools` `:52` appends MCP after
built-ins **(and the merged result is currently discarded by all callers, §2.5)`;
per-server best-effort skip on `Get` failure `:73-77`, non-active `:79-82`,
`ListTools` failure `:84-88`; `toolHandleMcpCall` `:127-178` fails closed. Reverse
map key `MetaKeyMcpToolMap = "mcp_tool_map"` at `models/aicall/main.go:67`, pruned
on session start/reuse via `refreshMcpToolMap:279-302`. `mcp_tool_call_timeout_seconds`
default 10 at `internal/config/main.go:150` (this one IS consumed).

### 2.10 Wire contract (for doc accuracy)

Source of truth: `bin-ai-manager/models/mcpserver/webhook.go:14-45`, NOT the
internal struct. 13 fields: `id`, `customer_id` (embedded
`commonidentity.Identity`), `name`, `detail`, `url`, `status`, `auth_type`,
`api_key_header`, `oauth_vendor`, `has_secret`, `tm_create`, `tm_update`,
`tm_delete`. Never document `SecretCiphertext`, `SecretNonce`, `KeyVersion`.

`auth_type` deliberately omits `,omitempty` (`:32`) because `AuthTypeNone` is the
empty string and is meaningful, so it is ALWAYS on the wire. `oauth_vendor` DOES
carry `,omitempty` (`:38`), so it is ABSENT, not empty, when unset —
`ai_struct_mcpserver.rst:37` says "empty otherwise" (imprecise) and the example at
`:60-73` omits the field entirely.

Enums (`models/mcpserver/main.go:13-45`): `auth_type` = `""` | `bearer` |
`api_key` | `oauth`; `status` = `active` | `disabled`. OAuth vendors wired:
GitHub, Linear only (`internal/config/main.go:151-154`).

### 2.11 D8: square-admin gap is frontend-only. CONFIRMED.

Backend accepts `mcp_server_ids` on BOTH create and update, four layers verified:
`openapi/paths/ais/main.yaml:93-102`; `gens/models/gen.go` `PostAisJSONBody.McpServerIds`
`:10609` and `PutAisIdJSONBody` `:10667`; `bin-api-manager/server/ais.go:74-80`
(POST) and `:293-299` (PUT); `bin-ai-manager/pkg/listenhandler/v1_ais.go:117-125`
(POST) and `:272-280` (PUT). **No backend or OpenAPI change needed for D8.**

Frontend hits exist ONLY in `square-admin/src/views/ais/ais_detail.js` (`:103`
state, `:242` hydrate, `:374` list fetch, `:382` toggle, `:421` PUT body,
`:552-553` dirty check, `:1239-1276` UI) plus its test. Zero hits (case-insensitive) in:
`views/ais/ais_create.js` (438 lines; body `:114`, POST `:154`),
`views/ais/AIEngineFields.js` (659 lines; the SHARED component both pages use),
`views/teamgraph/sidebar.js` (1769 lines; AI create body `:514-534`, AI edit body
`:753-782`), `src/types/api.ts` (`tool_names?` at `:503`).

`square-admin/CLAUDE.md:161-181` Field Sync Points names three required sites; MCP
satisfies one. `src/types/api.ts` is a de-facto FOURTH site (the checklist's step 1
says "Add type to src/types/api.ts") but is absent from the table — the table needs
a row.

### 2.12 Other surfaces

| Surface | Finding |
|---|---|
| api-validator, repo `~/gitvoipbin/monorepo-monitoring/api-validator/tests/scenarios/` (NOT `monorepo/monitoring`, which holds only CLAUDE.md + Grafana dashboards) | `test_mcpservers_lifecycle.py` + `test_mcpservers_oauth.py` (9 tests) cover mcpserver CRUD. `grep -rn mcp_server_ids api-validator/` → exactly one hit, a docstring at `test_mcpservers_lifecycle.py:36`. **Zero assertions** that `POST /ais` or `PUT /ais/{id}` accepts/persists/rejects `mcp_server_ids`, and **no OAuth-against-deleted-server test**. `:243-274` is the GET-after-DELETE-returns-200 test, whose docstring warns a 404 would be a platform-wide convention change |
| `square-main/public/skill.md`, `llms.txt` | `skill.md:414` / `llms.txt:49` enumerate AI creation fields incl. `tool_names`, no `mcp_server_ids`. Separately `skill.md:774-795` / `llms.txt:11` document "MCP Server" meaning `uvx voipbin-mcp` (VoIPBin AS an MCP server) — same terminology collision as `ai_overview.rst:681`, on a second customer-visible surface |
| `ai_overview.rst:677-685` | links `github.com/nrjchnd/voipbin-mcp`, a third-party fork, while `skill.md:794` links the official `github.com/voipbin/mcp` |
| `openapi.yaml:2239-2244` | the `mcp_server_ids` description states namespacing without the "8 hex chars" detail, says nothing about best-effort skip, and (per §2.5) promises LLM merging that does not happen. Redoc is customer-facing, so this is an editable surface |
| `square-talk`, `square-meet`, `square-dev`, `square-admin_new` | zero MCP / AI-tool-config references (`square-admin_new` has no `src/`). Genuinely clear |

## 3. Proceed decision and PR plan

### PROCEED. Three PRs, ordering-constrained. CEO-approved split.

**PR A — `voipbin/monorepo` — MCP server lifecycle correctness + docs.**
D1, D2, D3, D4, D5, D6. One logical unit: "a deleted MCP server must be inert
everywhere, and the docs/config must stop describing things that are not true."

**PR B — `voipbin/monorepo` — Phase 2: expose MCP tools to the LLM.**
D7. Depends on PR A: this PR is what makes `CallTool` reachable, so it must not
land before the revocation gates (§2.5). Touches `bin-pipecat-manager`, so it is
also the largest review surface.

**PR C — `voipbin/monorepo-javascript` — square-admin parity.**
D8 + the `types/api.ts` field + the CLAUDE.md Field Sync Points table row +
re-verification of the `mcpservers_detail.js:496` delete-dialog copy (PR A makes it
true). Independent of A and B; may proceed in parallel.

The split is by **ordering dependency and repository boundary**, not by
convenience: A→B is a hard sequence, C is a different repo.

### Why this is not overengineering

D1..D4 are correctness/security fixes on a shipped feature with code-verified
failure modes. D5/D6 remove published falsehoods. D7 makes a shipped, documented,
UI-exposed capability actually function. D8 completes it. The one thing explicitly
NOT built is the Redis cache (§2.6), because there is no measured signal for it.

### Non-goals

- **Tool-level (per-tool) whitelisting.** `resolveTools:91-105` adds every tool a
  server returns, no per-tool filter. Deferred: no demand signal, and
  `mcp_server_ids []uuid.UUID` extends additively. **The deferral is not
  consequence-free:** server-level whitelisting grants all present AND FUTURE
  tools that server exposes, i.e. a third party can silently expand its own scope.
  Mitigation is one explicit sentence in the docs, not building the feature.
- Changing the soft-delete READ contract (GET after DELETE returns 200). It is
  intentional and validator-tested (§2.12).
- The Redis tools/list cache (§2.6).
- Additional OAuth vendors beyond GitHub/Linear.
- In-app help content (`src/views/help/content/navHelpManifest.js:39`, explicit
  `helpTopicId: null`).
- Resolving the `uvx voipbin-mcp` naming collision platform-wide (§4 Q4).

## 4. Decisions to lock before the design docs

1. **PR A gate placement.** Mandatory: transport (`client.go:190,213`), whitelist
   validation (`mcpserver_validation.go:35-60`), `McpServerUpdate`
   (`dbhandler/mcpserver.go:146-149`), OAuth `start.go:37` AND `complete.go:112`
   independently. Defense in depth: resolution (`mcp_tool.go:72-88`) and dispatch
   (`:151-168`). **Explicitly forbidden:** gating inside
   `dbhandler.McpServerGet` — `mcpServerHandler.Delete` calls it at
   `handler.go:230` AFTER `McpServerDelete` to return the row and publish
   `EventTypeDeleted`, and `Update` reads back at `:210`; gating there breaks
   Delete's own success path and the validator's 200 contract. Confirm, and
   confirm whether a deleted server is skipped silently (matching the existing
   non-active pattern) or logged at WARN.
2. **Credential zeroing mechanics (Q3).** Zero in the same UPDATE as the delete
   timestamps, or a separate statement? And confirm `has_secret` flipping to false
   on the returned deleted row is acceptable.
3. **Does PR A prune stale ids from `ai.mcp_server_ids` on delete?** Recommend NO:
   consume-time gating is sufficient and idempotent, and pruning means a fan-out
   write across every AI of that customer on every delete. The stale id becomes
   inert. But PR C should not render an id that resolves to nothing.
4. **Naming collision** (`uvx voipbin-mcp` = VoIPBin-as-MCP-server vs
   customer-registered MCP server): rename the `ai_overview.rst:681` heading only,
   or also `skill.md`/`llms.txt`? And should `ai_overview.rst:685` keep pointing at
   the third-party fork rather than `github.com/voipbin/mcp`?
5. **api-validator coverage** (a third repo, `monorepo-monitoring`, therefore a
   fourth PR): `mcp_server_ids` on `POST /ais`, revocation behavior after DELETE,
   and OAuth start/complete against a deleted server. Add now or defer?
6. **PR B's advertising mechanism.** `resolveTools` returns
   `[]tool.Tool` (`bin-ai-manager/models/tool`), while pipecat builds
   `[]aitool.Tool` via its own `toolHandler.GetByNames` and ships it to
   `pythonRunner.Start`. The design must decide whether ai-manager passes the
   merged list across to pipecat (changing the pipecat tool-source contract) or
   pipecat gains an MCP-aware path. This is PR B's central design question, not
   PR A's.
7. **Does `square-admin/CLAUDE.md` Field Sync Points gain `src/types/api.ts` as a
   fourth row?** Recommend yes.

## 5. Risk table

| Risk | Severity | Mitigation |
|---|---|---|
| PR B lands before PR A → deleted servers' tools become LLM-callable, converting a `ListTools`-only exposure into full tool access | **High** | Hard ordering A→B (§3); state the dependency in both PR bodies |
| Gate placed in `dbhandler.McpServerGet` breaks `Delete`/`Update` read-back and the validator's GET-after-DELETE 200 contract | High | §4 Q1 forbids it explicitly; run the full api-validator mcpserver suite |
| Credential zeroing is irreversible; a mis-fire destroys a live server's secret | High | Zero only on the delete path, never on update; unit test that update paths never clear ciphertext columns |
| Published docs promise LLM-facing MCP tool merging the code never delivers | High (live) | D7 / PR B. Until PR B lands, PR A must not add NEW claims of the same kind |
| Published docs give a 404 endpoint path | High (live) | D5 |
| A deleted server can be resurrected to `active` with a new URL + secret | High | D2 |
| OAuth Complete has no re-check (TOCTOU between Start and Complete) | Medium | D3; gate both independently |
| Dead config flag + design doc justifying another decision by citing a nonexistent cache | Medium | D6; remove flag, correct `2026-09-11-…:466-467`, `:705`, `:750-754` |
| `mcp_server_ids` grants all present and FUTURE tools of a server | Medium | One explicit docs sentence (§3 non-goals) |
| Coverage for `mcp_server_ids` on `POST /ais` is Go-unit-only, no end-to-end | Medium | §4 Q5 |
| PR A "breaks" a customer whose AI depends on a deleted-but-working server | **Low** (downgraded from v2) | No customer AI can be *calling* those tools today (§2.5); breakage is limited to stopping the unwanted `ListTools` round trip. Still note it in the PR body |
| Field Sync Points drift recurs | Low | Update all four sites + amend the table in PR C |
| Teamgraph AI panels are inline, not the shared component | Low | Accept the duplication to match the file's pattern; do not refactor mid-task |
| PR C lands in the dead nav/layout generation | Low | No nav change needed; `/resources/mcpservers/*` routes already live (`routes.js:333-336`, `_nav.js:239`). Changes are inside existing AI forms only |

## 6. Verification plan

**PR A (`monorepo`):** `go mod tidy && go mod vendor && go generate ./... && go
test ./... && golangci-lint run -v --timeout 5m` in `bin-ai-manager`. New unit
tests proving a soft-deleted server is: skipped in resolution, refused in
dispatch, refused in transport, rejected by `ValidateMcpServerIDs`, rejected by
`McpServerUpdate`, rejected by OAuth `Start`, rejected by OAuth `Complete`; and
that delete zeroes all secret/token ciphertext while update paths never do. Clean
Sphinx rebuild; `grep -n 'mcp_servers' docsdev/source/*.rst` returns nothing.
Full api-validator mcpserver suite green (the GET-after-DELETE 200 contract must
still hold).

**PR B (`monorepo`):** same Go gates in both `bin-ai-manager` and
`bin-pipecat-manager`; an end-to-end check that a whitelisted server's tool
actually reaches the model and that a deleted server's does not.

**PR C (`monorepo-javascript`):** baseline-vs-branch test comparison per CLAUDE.md
test gate; `npm run build`; production build served and verified in a real browser
(reviews and RTL do not catch unmount races).

## 7. Retrospective

**v1** asserted "exactly ONE grep hit" and "no `mcpserver*.rst` files exist." Both
false; root cause was a case-sensitive `grep mcp` that missed uppercase `MCP`,
written up as evidence without re-running it. The whole v1 priority ordering
rested on that one unverified claim.

**v2** fixed that but overstated D1 ("still resolved and dispatched … indefinitely")
without checking whether the resolved list is ever used, and missed D2, D3, D4,
D6, D7 entirely. Root cause: treating `resolveTools`' existence as proof of its
effect, rather than following the return value to a consumer.

Both failure modes are the same shape: **a claim about behavior inferred from the
existence of code, instead of traced to the code that consumes it.** For every
"X happens" claim, find the consumer.

Lessons now in effect: use `grep -i` for content/convention greps; for any
producer function, grep its callers and check whether the return value is
discarded (`_,`).
