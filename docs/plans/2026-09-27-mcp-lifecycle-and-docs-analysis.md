# Analysis: MCP server lifecycle correctness + square-admin parity (v5, PR A/C only)

Date: 2026-09-27
Author: CPO (Hermes)
Status: v5, Analysis Review Loop round 5 pending
Scope: **PR A and PR C only.** Phase 2 LLM tool exposure (formerly PR B) is split
out to `2026-09-27-mcp-phase2-tool-exposure-analysis.md` and is NOT in scope here.

Supersedes (all retained for the failure-mode record):
- `…-mcp-server-customer-gap-analysis-v1-superseded.md` (§2.4 factually wrong)
- `…-mcp-server-revocation-and-docs-analysis-v2-superseded.md` (D1 severity overstated)
- `…-mcp-lifecycle-and-phase2-analysis-v3-superseded.md` (PR B scope understated)
- `…-mcp-lifecycle-and-phase2-analysis-v4-superseded.md` (PR B still unconverged; four citation defects)

## 0. Why this document was split

Rounds 1-4 of the review loop all returned CHANGES_REQUESTED. The pattern changed
at round 4: rounds 1-3 found factual errors in the lifecycle analysis, round 4
found only **PR B scope discoveries** (a fourth AIcall path missing the tool map,
a whitelist-reintroduction hazard, an asymmetric sanitizer in the team path) while
confirming every lifecycle claim as correct.

CEO decision (2026-09-27): split the analysis. Verbatim: "권고대로 진행해" in
response to the recommendation that PR A/C analysis is converged and should not
wait on PR B's, since PR A fixes live security defects and is PR B's prerequisite
anyway.

This document therefore covers only the converged half. PR B keeps iterating
separately.

## 1. CEO decisions locked 2026-09-27

| # | Question | Decision |
|---|---|---|
| Q1 | Implement the documented "MCP tools presented to the LLM" behavior, or correct the docs? | **Implement** (belongs to PR B, out of scope here). "Q1. 구현하자." |
| Q2 | How much of the lifecycle scope lands? | **All of it.** "Q2. 전부." |
| Q3 | Zero stored credentials on delete? | **Yes.** "Q3. 지우자. 그게 맞지." |
| Q4 | Dead `mcp_tools_list_cache_ttl_seconds` flag: implement or remove? | **Remove the flag, no cache.** |
| Q5 | PR B as one PR or split? | **One PR.** "하나로 가자." (PR B doc) |
| Q6 | Split the analysis so A/C can proceed without waiting on B? | **Yes.** "권고대로 진행해" |

## 2. Issue statement (PR A/C scope)

**PR A — lifecycle correctness + docs (`voipbin/monorepo`):**
- **D1** Soft-delete does not revoke tool access. A deleted server keeps
  `status = "active"`, every read path ignores `tm_delete`, so the AI tool path
  keeps making credentialed outbound `ListTools` calls to it forever.
- **D2** `McpServerUpdate` has no `tm_delete` predicate and never clears it, and
  `status` is PUT-settable, so **a deleted server is mutable and can be flipped
  back to `active` with a new URL and a new secret.**
- **D3** OAuth start/complete reach a deleted server; `Complete` has *no*
  ownership/existence re-check by explicit design comment, so tokens get written
  onto a `tm_delete`-populated row.
- **D4** `McpServerDelete` retains `SecretCiphertext`,
  `AccessTokenCiphertext`, `RefreshTokenCiphertext` indefinitely.
- **D5** Published docs give a 404 endpoint path (`/mcp_servers` vs `/mcpservers`)
  plus two broken `:ref:` targets.
- **D6** `mcp_tools_list_cache_ttl_seconds` is a dead flag; the design doc
  justifies a *separate* design decision by citing this nonexistent cache.
- **D12** `toolHandleMcpCall` never asserts `server.CustomerID == c.CustomerID`,
  and on team AIcalls it validates the whitelist against the **start** member while
  the session runs the **current** member.
- **D13** `ValidateMcpServerIDs` takes no AI type, so `mcp_server_ids` is not
  type-validated, asymmetric with `tool_names`.
- **D14** The three mcpserver webhook event types are undocumented, and
  `ai.McpServerIDs` carries `,omitempty` while the RST says it defaults to `[]`.

**PR C — square-admin parity (`voipbin/monorepo-javascript`):**
- **D8** `mcp_server_ids` is settable only on the AI detail page, though the
  backend accepts it on `POST /ais` too.

Deferred to the PR B document: D7 (tools never advertised to the LLM), D9
(`inputSchema` tag), D10 (`AllowedToolNames` interaction), D11 (name validation).

## 3. Validity re-check (code-verified; unchanged and re-confirmed in round 4)

### 3.1 D1: revocation does not revoke. CONFIRMED.

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

**Live exposure today.** On every AIcall/insight session start that calls
`resolveTools`, `ListTools` (`mcp_tool.go:84`) fires against every whitelisted
server including deleted ones, decrypting the stored secret and issuing an
authenticated outbound HTTP request to the customer's URL. That is the real,
ongoing defect in PR A's scope.

`CallTool` dispatch is currently **unreachable in production** because the merged
tool list `resolveTools` returns is discarded by every caller (`start.go:1169`,
`insight_session.go:225`, `mcp_tool.go:284` all use `_,`), and the lists actually
sent to models are built elsewhere with no MCP awareness. That is D7's subject
matter and belongs to the PR B document; it is stated here only to keep PR A's
severity claim honest and to explain why PR A must precede PR B.

Contradicted UI promise:
`square-admin/src/views/mcpservers/mcpservers_detail.js:496` — "Any AI still
whitelisting it will stop being able to call it." False today.

### 3.2 D2: a deleted server is mutable and resurrectable. CONFIRMED.

- `pkg/dbhandler/mcpserver.go:146-149`: `sq.Update(mcpserverTable).SetMap(...)
  .Where(sq.Eq{"id": id.Bytes()})`. No `tm_delete` predicate. It sets `tm_update`
  and **never clears `tm_delete`**. `RowsAffected` is ignored (`:154-156`) — see
  §5 Q2.
- `pkg/mcpserverhandler/handler.go:168-169`: `status` is PUT-settable.
- Net: `PUT /mcpservers/{id}` on a deleted server succeeds, can set
  `status = active`, and can replace `url` and the secret.

This retracts any claim that "`tm_delete` is terminal": on the write paths it is
not, which is why gating reads alone leaves a resurrection path.

### 3.3 D3: OAuth reaches a deleted server; Complete has no re-check. CONFIRMED.

- `pkg/mcpoauthhandler/start.go:37` `h.db.McpServerGet(ctx, *mcpServerID)` — the
  reconnect ownership check, which ignores `tm_delete`, so an OAuth reconnect can
  be **started** against a deleted server, creating a `mcpoauthstate` row.
- `pkg/mcpoauthhandler/complete.go:112-133` then calls `McpServerUpdate` on that
  id. The comment at `:113-115` states verbatim that ownership "was already
  verified in Start … so this is a plain update, not a second ownership check."
  **No existence or deletion re-check at all**, so a server deleted *between*
  Start and Complete (window = `stateTTL`) receives fresh
  `access_token`/`refresh_token` ciphertext on a deleted row. Gating `Start` alone
  does not close this.
- `pkg/mcpoauthhandler/access_token.go:37-112` `GetValidAccessToken` receives an
  already-fetched `*mcpserver.McpServer` from `pkg/mcptoolhandler/client.go:97`
  and persists rotated tokens via `McpServerUpdate:108`. It has NO independent
  read, so a transport gate at `client.go:191/214` also closes the refresh path —
  no separate gate needed. Until then, `ListTools` against a deleted OAuth server
  keeps refreshing and re-persisting vendor tokens on it.
- **Verified clean, do NOT over-gate:** the public unauthenticated
  `GET /mcpservers/oauth/callback`
  (`bin-api-manager/cmd/api-manager/main.go:310-321`) calls `CallbackExists`
  (`pkg/mcpoauthhandler/callback.go:17-31`), which touches only
  `McpOAuthStateGet` and never `McpServer`. No deleted-server exposure.

### 3.4 D4: credentials survive delete. CONFIRMED, and schema-safe to zero.

`McpServerDelete` (`pkg/dbhandler/mcpserver.go:163-182`) writes only `tm_update`
and `tm_delete`. `SecretCiphertext`, `SecretNonce`, `AccessTokenCiphertext`,
`AccessTokenNonce`, `RefreshTokenCiphertext`, `RefreshTokenNonce`, `KeyVersion`
all remain indefinitely.

**Verified no schema obstacle:** `bin-dbscheme-manager` migration
`9b0ad37e0360_ai_mcp_servers_create_table.py:35-37` declares `secret_ciphertext`
(blob), `secret_nonce` (binary(12)), `key_version` (smallint) with **no NOT NULL**,
and `62c10f986f07_ai_mcp_servers_add_oauth_columns.py:30-34` declares all six OAuth
columns explicitly nullable. Zeroing is safe.

### 3.5 D12: no ownership check; team validates the wrong member. CONFIRMED.

`pkg/aicallhandler/mcp_tool.go:127-178` has exactly three gates:
`lookupMcpToolRef` (`:137`), whitelist membership (`:151`), and
`server.Status != StatusActive` (`:163`). **There is no
`server.CustomerID == c.CustomerID` assertion anywhere in the dispatch path.** The
only ownership check in the system is write-time `ValidateMcpServerIDs`
(`pkg/aihandler/mcpserver_validation.go:50`, `srv.CustomerID != customerID`) — a
single-layer defense.

The whitelist check also validates the wrong AI on team calls. `mcp_tool.go:144`
calls `h.resolveAI(...)`, whose team branch (`pkg/aicallhandler/start.go:75`)
resolves **`t.StartMemberID`**:

```go
a, memberID, err := h.resolveTeamMemberAI(ctx, t, t.StartMemberID)
```

while the running session tracks `CurrentMemberID`
(`bin-pipecat-manager/pkg/pipecatcallhandler/run.go:150-163`,
`resolveAIFromAIcall` `:218-224`). On a team call that has transitioned members,
the whitelist check runs against the START member's `McpServerIDs`. Two failure
directions: a tool the current member legitimately whitelists is refused, and a
tool only the start member whitelists is ACCEPTED for the current member.

**Corrected blast radius (round 4).** The same mismatch affects
`start.go:1169` and `mcp_tool.go:284` (`refreshMcpToolMap`, called from
`start.go:361` with the `a` resolved at `start.go:187`). It does **NOT** affect
`insight_session.go:225`: `writeInsightSessionMetadata` is reached only from
`insight_session.go:143` inside `runInsightSessionRefresh`, which returns early at
`:68-72` when `AssistanceType != AssistanceTypeAI`, and its `a` comes from
`h.aiHandler.Get(...)` at `:104`, not from `resolveAI`'s team branch. Teams never
reach `:225`.

### 3.6 D13: `mcp_server_ids` is not AI-type validated. CONFIRMED.

```go
// pkg/aihandler/mcpserver_validation.go:35 — no AI type parameter
func (h *aiHandler) ValidateMcpServerIDs(ctx context.Context, customerID uuid.UUID, ids []uuid.UUID) error

// models/ai/tool_validation.go:72 — takes the type
func ValidateToolNames(t Type, toolNames []tool.ToolName) error
```

`ValidateMcpServerIDs` checks only existence and customer ownership. Both call
sites (`pkg/listenhandler/v1_ais.go:118`, `:273`) pass only `CustomerID`. Meanwhile
`tool_names` IS type-validated against `AllowedToolNames`
(`models/ai/tool_validation.go:77`), and the Insight catalog
(`AllInsightToolNames`, `models/tool/main.go:90`) is deliberately narrower than
Normal's.

So an Insight AI may whitelist arbitrary MCP servers while being denied most
built-in tools. Whether that is intended is a **policy decision** (§5 Q5). It is a
write-time gate, so if the answer is "not unconditionally," it belongs in PR A next
to the other gates. If the answer is "MCP is type-agnostic by design," PR A should
record that explicitly so the asymmetry stops looking like an oversight.

### 3.7 D14: undocumented webhook events + an omitempty/docs contradiction. CONFIRMED.

- `bin-ai-manager/models/mcpserver/event.go:5-7` declares
  `mcp_server_created`, `mcp_server_updated`, `mcp_server_deleted`. Grep across
  `bin-api-manager/docsdev/source/` and `bin-openapi-manager/openapi/` returns
  **zero** occurrences of all three. They are published
  (`mcpserverhandler/handler.go:242` for deleted) but undocumented.
  Silver lining for §5 Q2: no documented contract breaks when `has_secret` flips.
- `bin-ai-manager/models/ai/webhook.go:47`
  `McpServerIDs []uuid.UUID \`json:"mcp_server_ids,omitempty"\`` — so an
  explicitly cleared whitelist is **absent** from the AI webhook payload, while
  `ai_struct_ai.rst:69` says it "Defaults to `[]`". Same defect class as
  `oauth_vendor` in §3.10.

### 3.8 D5: published docs give a 404 path. CONFIRMED.

`bin-api-manager/docsdev/source/ai_struct_mcpserver.rst`:
- `:29` `POST /mcp_servers`, `GET /mcp_servers` — wrong
- `:45` `POST /mcp_servers`, `PUT /mcp_servers/{id}` — wrong
- `:49` `PUT /mcp_servers/{id}` — wrong
- `:35`, `:100` correctly `/mcpservers/oauth/...` — internally inconsistent

Those three are the only `mcp_servers` hits in the whole RST tree. Authoritative:
`bin-openapi-manager/openapi/openapi.yaml:8569-8578`, five routes, all
`/mcpservers`.

Two broken references in the same file:
- `:29` `:ref:` `mcp_server_ids <ai-struct-ai-tool_names>` points at the
  `tool_names` anchor (`ai_struct_ai.rst:340`); no `mcp_server_ids` anchor exists.
- `:100` self-references `oauth_vendor <mcpserver-struct-mcpserver-mcpserver>`,
  the containing section rather than a field anchor.

### 3.9 D6: `mcp_tools_list_cache_ttl_seconds` is a dead flag. CONFIRMED.

Every reference, repo-wide (excluding `vendor/`):

```
internal/config/main.go:84    struct field declaration
internal/config/main.go:149   f.Int(..., 60, "Redis cache TTL (seconds) for an MCP server's tools/list result")
internal/config/main.go:199   env mapping MCP_TOOLS_LIST_CACHE_TTL_SECONDS
internal/config/main.go:276   viper.GetInt(...)
```

Four hits, all inside config. **Zero consumers.** Contrast the sibling that works:
`McpToolCallTimeoutSeconds` at `config/main.go:85`, `:277`, and
**`cmd/ai-manager/main.go:156`** where `NewMcpToolHandler(db,
cfg.McpSecretEncryptionKeys, cfg.McpToolCallTimeoutSeconds, mcpOAuthHandler)`
consumes it. `NewMcpToolHandler` (`pkg/mcptoolhandler/main.go:74`) has no
cache-TTL parameter. `grep -rni 'mcp' pkg/cachehandler/` = zero matches.
`pkg/mcptoolhandler/client.go:190-207` `ListTools` goes straight to DB + HTTP.

The design doc asserts the cache exists in three places:
`docs/plans/2026-09-11-mcp-tool-integration-design.md:466-467`, `:705` (config
table), and most damagingly **`:750-754`**, which justifies a *different* decision
("a second cache layer over the `McpServer` row itself is not needed") by citing
this nonexistent cache as existing mitigation.

**Decision: remove the flag and correct the design doc. Do not implement a cache.**
(i) No measured signal — one call per session start, no threshold alarm, so a cache
layer now is speculative infrastructure. (ii) A cache actively conflicts with
revocation — it would serve a deleted server's tools for up to the TTL, requiring
invalidation design on top of the gates.

### 3.10 Wire contract (for doc accuracy)

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

### 3.11 What the docs ALREADY have (v1's false claim, retained as a guard)

`grep -ric mcp *.rst` → `ai_overview.rst:2`, `ai.rst:1`, `ai_struct_ai.rst:2`,
`ai_struct_mcpserver.rst:28`. Four files. `ai_struct_mcpserver.rst` is 101 lines on
`origin/main`, in the `ai.rst:15` toctree, reachable from `index.rst:83`, built to
`docsdev/build/html/ai_struct_mcpserver.html`. The CLAUDE.md `*_struct_*.rst`
obligation IS satisfied. Genuinely missing: narrative coverage (no
`mcpserver_overview.rst`, no `mcpserver_tutorial.rst`; `rag` has both).

### 3.12 D8: square-admin gap is frontend-only. CONFIRMED.

Backend accepts `mcp_server_ids` on BOTH create and update, four layers:
`openapi/paths/ais/main.yaml:93-102`; `gens/models/gen.go`
`PostAisJSONBody.McpServerIds` `:10609` and `PutAisIdJSONBody` `:10667`;
`bin-api-manager/server/ais.go:74-80` (POST) and `:293-299` (PUT);
`bin-ai-manager/pkg/listenhandler/v1_ais.go:117-128` (POST) and `:272-283` (PUT).
**No backend or OpenAPI change needed for D8.**

Frontend hits exist ONLY in `square-admin/src/views/ais/ais_detail.js` (`:103`
state, `:242` hydrate, `:374` list fetch, `:382` toggle, `:421` PUT body,
`:552-553` dirty check, `:1239-1276` UI) plus its test. Zero hits
(case-insensitive) in: `views/ais/ais_create.js` (438 lines; body `:114`, POST
`:154`), `views/ais/AIEngineFields.js` (659 lines; the SHARED component both pages
use), `views/teamgraph/sidebar.js` (1769 lines; AI create body `:514-534`, AI edit
body `:753-782`), `src/types/api.ts` (`tool_names?` at `:503`).

`square-admin/CLAUDE.md:161-181` Field Sync Points names three required sites; MCP
satisfies one. `src/types/api.ts` is a de-facto FOURTH site (the checklist's step 1
says "Add type to src/types/api.ts") but is absent from the table — the table needs
a row.

**Deleted servers are never rendered.** `bin-api-manager/pkg/servicehandler/mcpserver.go:67-70`
hardcodes `filters{"deleted":"false"}` and `ais_detail.js:374` fetches through that
path. The residual UI issue is only that a selected-but-unlisted id gets no
checkbox at `ais_detail.js:1252-1270` while `:421` still PUTs it back — a PR C
display decision, not a data problem. square-admin reads `has_secret` only from GET
responses (`mcpservers_list.js:86`, `mcpservers_detail.js:266`) and does not consume
mcpserver webhooks, and it already handles `auth_type: 'oauth'`
(`mcpservers_detail.js:38`), so §5 Q2's `has_secret` flip breaks nothing.

### 3.13 Other surfaces

| Surface | Finding |
|---|---|
| api-validator, `~/gitvoipbin/monorepo-monitoring/api-validator/tests/scenarios/` (NOT `monorepo/monitoring`, which holds only CLAUDE.md + Grafana dashboards) | `test_mcpservers_lifecycle.py` (12 tests) + `test_mcpservers_oauth.py` (9) = **21 tests**, no parametrize; CRUD/OAuth coverage is substantial. The gap is specific: `grep -rn mcp_server_ids api-validator/` → one hit, a docstring at `test_mcpservers_lifecycle.py:36`. **Zero assertions** on `mcp_server_ids` via `POST /ais` or `PUT /ais/{id}`, **no OAuth-against-deleted-server test**, and **no test asserting `has_secret` after DELETE** (`:244-274` asserts only 200 + `tm_delete is not None`; `has_secret` only on create/update at `:71`, `:95`, `:202`, `:224`) — so credential zeroing does not break the current suite |
| `square-main/public/skill.md`, `llms.txt` | `skill.md:414` / `llms.txt:49` enumerate AI creation fields incl. `tool_names`, no `mcp_server_ids`. Separately `skill.md:774-795` / `llms.txt:11` document "MCP Server" meaning `uvx voipbin-mcp` (VoIPBin AS an MCP server) — same terminology collision as `ai_overview.rst:681`, on a second customer-visible surface |
| `ai_overview.rst:677-685` | links `github.com/nrjchnd/voipbin-mcp`, a third-party fork, while `skill.md:794` links the official `github.com/voipbin/mcp` |
| `openapi.yaml:2239-2244` | the `mcp_server_ids` description omits the "8 hex chars" detail, says nothing about best-effort skip, and promises LLM merging that does not happen (D7, PR B). Redoc is customer-facing, so this is an editable surface — but the merging text must not be "corrected" here, because PR B will make it true |
| `square-talk`, `square-meet`, `square-dev`, `square-admin_new` | zero MCP / AI-tool-config references (`square-admin_new` has no `src/`). Genuinely clear |

## 4. Proceed decision and PR plan

### PROCEED with PR A and PR C.

**PR A — `voipbin/monorepo` — MCP server lifecycle correctness + docs.**
D1, D2, D3, D4, D5, D6, D12, D13 (pending §5 Q5), D14. One logical unit: "a
deleted or foreign MCP server must be inert everywhere, and the docs/config must
stop describing things that are not true."

**PR C — `voipbin/monorepo-javascript` — square-admin parity.**
D8 + `types/api.ts` + the CLAUDE.md Field Sync Points table row + re-verification
of the `mcpservers_detail.js:496` delete-dialog copy (PR A makes it true).
Independent of A; may proceed in parallel.

**PR B — deferred to its own analysis.** PR A remains its hard prerequisite: PR B
is what makes `CallTool` reachable, so landing it before PR A's gates would turn
today's `ListTools`-only exposure into live "deleted server's tools are callable"
exposure.

**Possible fourth PR** in `monorepo-monitoring` for api-validator coverage (§5 Q4).

### Why this is not overengineering

D1..D4, D12 are correctness/security fixes on a shipped feature with code-verified
failure modes. D5/D6/D14 remove published falsehoods and dead configuration. D8
completes an already-shipped capability. The one thing explicitly NOT built is the
Redis cache (§3.9), because there is no measured signal for it.

### Non-goals

- Everything in the PR B document (D7, D9, D10, D11).
- **Tool-level (per-tool) whitelisting.** `resolveTools:91-105` adds every tool a
  server returns, no per-tool filter. Deferred: no demand signal, and
  `mcp_server_ids []uuid.UUID` extends additively. **The deferral is not
  consequence-free:** server-level whitelisting grants all present AND FUTURE tools
  that server exposes, i.e. a third party can silently expand its own scope.
  Mitigation is one explicit sentence in the docs, not building the feature.
- Changing the soft-delete READ contract (GET after DELETE returns 200) —
  intentional and validator-tested.
- The Redis tools/list cache (§3.9).
- Additional OAuth vendors beyond GitHub/Linear.
- In-app help content (`src/views/help/content/navHelpManifest.js:39`, explicit
  `helpTopicId: null`).
- Narrative docs (`mcpserver_overview` / `_tutorial`) — desirable but not required
  to fix a live 404; decide at PR A design time whether they fit without bloating
  review.
- Resolving the `uvx voipbin-mcp` naming collision platform-wide (§5 Q3).

## 5. Decisions to lock before the PR A design doc

1. **Gate placement.** Mandatory: transport (`client.go:190,213`), whitelist
   validation (`mcpserver_validation.go:35-60`), `McpServerUpdate`
   (`dbhandler/mcpserver.go:146-149`), OAuth `start.go:37` AND `complete.go:112`
   independently, plus the D12 ownership assertion at `mcp_tool.go:157-167`.
   Defense in depth: resolution (`mcp_tool.go:72-88`) and dispatch (`:151-168`).
   **Explicitly forbidden:** gating inside `dbhandler.McpServerGet` —
   `mcpServerHandler.Delete` calls it at `handler.go:230` AFTER `McpServerDelete`
   to return the row and publish `EventTypeDeleted`, and `Update` reads back at
   `:210`; gating there breaks Delete's own success path and the validator's 200
   contract. Confirm, and confirm whether a deleted server is skipped silently
   (matching the existing non-active pattern) or logged at WARN.
2. **Credential zeroing mechanics. RESOLVED: same statement as the delete
   timestamps.** A separate `dbhandler.McpServerUpdate` after `McpServerDelete`
   would be **silently no-op'd** by the `tm_delete IS NULL` predicate Q1 mandates,
   because `McpServerUpdate` ignores `RowsAffected` (`dbhandler/mcpserver.go:154-156`).
   Confirmed acceptable side effect: `has_secret` flips to false on the
   `EventTypeDeleted` payload (`mcpserverhandler/handler.go:242`) and on the
   GET-after-DELETE response; no validator test asserts it (§3.13), no square-admin
   consumer breaks (§3.12), no documented contract exists (§3.7), and it is truthful.
3. **Naming collision** (`uvx voipbin-mcp` = VoIPBin-as-MCP-server vs
   customer-registered MCP server): rename the `ai_overview.rst:681` heading only,
   or also `skill.md`/`llms.txt`? And should `ai_overview.rst:685` keep pointing at
   the third-party fork rather than `github.com/voipbin/mcp`?
4. **api-validator coverage** (third repo `monorepo-monitoring`, therefore a fourth
   PR): `mcp_server_ids` on `POST /ais`, revocation behavior after DELETE, OAuth
   start/complete against a deleted server, `has_secret` false after DELETE. Add now
   or defer?
5. **D13 policy: are MCP servers permitted on `TypeInsight` AIs?** If conditional,
   `ValidateMcpServerIDs` gains an AI-type parameter and the gate lands in PR A. If
   unconditional by design, PR A records the rationale so the asymmetry with
   `tool_names` is deliberate rather than accidental. **This must be answered before
   PR A's design doc**, because it changes a function signature and both call sites.
6. **Does PR A prune stale ids from `ai.mcp_server_ids` on delete?** Recommend NO:
   consume-time gating is sufficient and idempotent; pruning means a fan-out write
   across every AI of that customer on every delete. The stale id becomes inert, and
   deleted servers are never rendered (§3.12).
7. **Does `square-admin/CLAUDE.md` Field Sync Points gain `src/types/api.ts` as a
   fourth row?** Recommend yes (PR C).
8. **D14 scope in PR A:** document the three webhook event types now, or only fix
   the `ai_struct_ai.rst:69` "Defaults to `[]`" vs `,omitempty` contradiction?
   Recommend fixing the contradiction in PR A and documenting the event types in the
   same PR if it stays reviewable.

## 6. Risk table

| Risk | Severity | Mitigation |
|---|---|---|
| Gating inside `dbhandler.McpServerGet` breaks `Delete`/`Update` read-back and the validator's GET-after-DELETE 200 contract | **High** | §5 Q1 forbids it explicitly; run the full 21-test api-validator mcpserver suite |
| Credential zeroing as a separate UPDATE is silently no-op'd by the new `tm_delete` predicate (`RowsAffected` ignored at `dbhandler/mcpserver.go:154-156`) | **High** | §5 Q2 resolved: same statement; test that ciphertext columns are actually null after delete |
| Credential zeroing is irreversible; a mis-fire destroys a live server's secret | **High** | Zero only on the delete path, never on update; unit test that update paths never clear ciphertext columns |
| No customer-ownership assertion in dispatch; team path validates the START member's whitelist | **High** | D12 (§3.5) |
| A deleted server can be resurrected to `active` with a new URL + secret | **High** | D2 |
| Published docs give a 404 endpoint path | **High** (live) | D5 |
| OAuth Complete has no re-check (TOCTOU between Start and Complete) | Medium | D3; gate both independently |
| PR A "corrects" the `openapi.yaml`/`ai_struct_ai.rst` LLM-merging text that PR B will make true | Medium | Leave D7's text alone in PR A; only fix the `,omitempty`/"Defaults to `[]`" contradiction (§5 Q8) |
| Dead config flag + design doc justifying another decision by citing a nonexistent cache | Medium | D6; correct `2026-09-11-…:466-467`, `:705`, `:750-754` |
| Insight AIs can whitelist arbitrary MCP servers while denied most built-ins | Medium | D13 / §5 Q5 — decide before the design doc |
| `mcp_server_ids` grants all present and FUTURE tools of a server | Medium | One explicit docs sentence (§4 non-goals) |
| Coverage for `mcp_server_ids` on `POST /ais` is Go-unit-only, no end-to-end | Medium | §5 Q4 |
| `GetValidAccessToken`'s error surface changes once `McpServerUpdate` is gated | Low | Desired behavior; needs an explicit test (§7) |
| PR A "breaks" a customer whose AI depends on a deleted-but-working server | Low | No customer AI can be *calling* those tools today (§3.1); breakage is limited to stopping the unwanted `ListTools` round trip. Note it in the PR body |
| Field Sync Points drift recurs | Low | Update all four sites + amend the table in PR C |
| Teamgraph AI panels are inline, not the shared component | Low | Accept the duplication to match the file's pattern; do not refactor mid-task |
| PR C lands in the dead nav/layout generation | Low | No nav change needed; `/resources/mcpservers/*` routes already live (`routes.js:333-336`, `_nav.js:239`). Changes are inside existing AI forms only |

## 7. Verification plan

**PR A (`monorepo`):** `go mod tidy && go mod vendor && go generate ./... && go
test ./... && golangci-lint run -v --timeout 5m` in `bin-ai-manager`. New unit
tests proving a soft-deleted server is: skipped in resolution, refused in dispatch,
refused in transport, rejected by `ValidateMcpServerIDs`, rejected by
`McpServerUpdate`, rejected by OAuth `Start`, rejected by OAuth `Complete`; that
delete zeroes all secret/token ciphertext in the SAME statement while update paths
never do; that dispatch refuses a server owned by another customer (D12); that the
team path validates the CURRENT member's whitelist; and `GetValidAccessToken`'s new
error when rotating against a row deleted mid-flight. Plus D13's type gate if §5 Q5
says conditional. Clean Sphinx rebuild; `grep -n 'mcp_servers' docsdev/source/*.rst`
returns nothing. Full 21-test api-validator mcpserver suite green (GET-after-DELETE
200 must still hold).

**PR C (`monorepo-javascript`):** baseline-vs-branch test comparison per CLAUDE.md
test gate; `npm run build`; production build served and verified in a real browser
(reviews and RTL do not catch unmount races — 대표님 has repeatedly found real bugs
this way after green reviews).

## 8. Retrospective (four rounds, four different failure modes)

**v1** asserted "exactly ONE grep hit" and "no `mcpserver*.rst` files exist." Both
false. Root cause: a case-sensitive `grep mcp` that missed uppercase `MCP`, written
up as evidence without re-running it.

**v2** fixed that but overstated D1 ("still resolved and dispatched … indefinitely")
without checking whether the resolved list is ever used, and missed D2, D3, D4, D6,
D7. Root cause: treating `resolveTools`' existence as proof of its effect.

**v3** understated PR B by an order of magnitude: missed the `inputSchema` tag bug,
the `AllowedToolNames` filter, unvalidated tool names, and the missing ownership
check; and invented a type-conversion problem that does not exist. Root cause:
having traced `resolveTools` to its callers, I stopped, never tracing forward to the
python consumer or backward to the MCP wire format.

**v4** still mis-cited the team registration mechanism (`tools.py:113-116`, which
the team path never calls — the real site is `team_flow.py:63-70`, a file absent
from v4 entirely), missed a FOURTH AIcall path that never writes the tool map
(`startAIcallByRealtime:1087`), cited the wrong `GetByNames` (ai-manager has its own
aiType-blind one), and presented a dead code path (`StreamingSend` has no callers)
as a live advertising surface. Root cause: assuming symmetry — that two paths doing
"the same thing" do it the same way.

All four are the same shape: **a claim about behavior inferred from the existence or
apparent symmetry of code, instead of traced end-to-end.**

Lessons now in effect:
- Use `grep -i` for content/convention greps.
- For any producer function, grep its callers and check whether the return value is
  discarded (`_,`).
- For any struct unmarshalling an external wire format, verify the json tag against
  the actual specification, not local naming convention.
- When a value crosses a process/language boundary, read the receiving side's
  parsing code before asserting the contract holds.
- Before declaring scope, enumerate EVERY path that creates the same resource, and
  verify each one independently — never assume two paths are symmetric.
- Before citing a function, confirm which implementation the caller actually binds
  to (same name, different package, different signature).
- Before citing a code path as live, grep for its callers.
