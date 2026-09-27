# Analysis: MCP server lifecycle correctness + square-admin parity (v5, PR A/C only)

Date: 2026-09-27
Author: CPO (Hermes)
Status: v14, Analysis Review Loop round 10 pending
Round 5: fact-check **APPROVED**; adversarial CHANGES_REQUESTED (7 findings → D15-D19).
Round 6: fact-check **APPROVED**; adversarial CHANGES_REQUESTED (3 blocking → D20).
Round 7: fact-check **APPROVED** (3 consecutive); adversarial CHANGES_REQUESTED
(1 blocking, PR A ↔ PR C boundary → D21 + §5 Q11).
Round 8: **BOTH tracks CHANGES_REQUESTED.** Fact-check refuted v10's "the PUT path
performs no pre-write `AIGet`"; adversarial found the **fourth instance** of the
recurring failure mode in Q5 (D22).
Round 9: **BOTH tracks CHANGES_REQUESTED.** Adversarial found the **fifth instance**,
inside the v12 material written to close the fourth (D23): a type flip omitting
`mcp_server_ids` bypassed the gate and produced a permanently frozen Insight AI.
Fact-check found a missed third transition site and a FALSE api-validator citation
(`test_ai_lifecycle.py` is skipped at `:4`).
**v14 — 대표님 REVERSED the underlying policy: "인사이트도 mcp 사용가능하도록 해."**
Insight AIs may use MCP servers. No AI-type gate is added anywhere, which **voids
D13, D22 and D23 outright** (retained in §3.12b/§3.12e as recorded instances of the
failure mode, marked VOIDED and out of scope). Five rounds of gate design collapse into
one docs sentence. D18 is re-scoped: it no longer rides on a signature rewrite.
v9: 대표님 resolved Q11 and Q6. v10: Q5 and Q9. v12: Q5(i)/(ii), now withdrawn.
**v14 status: readiness NOT claimed.** §5 Q1/Q3/Q4/Q7/Q8 still carry "Recommend …"
items never confirmed by 대표님; those are listed for a single decision pass. v10 and
v12 both claimed readiness prematurely and both were refuted, so this header will not
claim it until a round returns clean AND those five are closed.
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
| Q7 | D21: how to handle an AI whose whitelist already holds a deleted id? | **Skip already-stored ids, reject newly added ones.** "skip-not-reject 방식으로 하자." + "이후에 AI 업데이트를 할 때 ... 이때는 reject 를 해야하지 않을까?" |
| Q8 | Add a prune loop that strips deleted ids from every affected AI? | **No.** "비쌀려나?" → no transaction on delete, un-indexed JSON column, one `ai_updated` webhook per affected AI, and skip-not-reject already resolves D21 alone |
| Q9 | D13: add an AI-type parameter to `ValidateMcpServerIDs`? | **REVERSED in v14 — No.** "인사이트도 mcp 사용가능하도록 해." Insight AIs may use MCP servers. No type gate anywhere; D13/D22/D23 are voided. PR A owes one docs sentence stating the behavior |
| Q10 | D17: reject `auth_type: "oauth"` on POST/PUT? | **Yes, reject.** "좋아, 네 제안대로 가자" Gate the write handlers, not `IsValid()`; reject the transition INTO oauth, not the value |
| Q11 | ~~Where does the Insight AI-type gate live?~~ | **MOOT (v14).** No gate exists. The v12 answer (both write and consume path) is withdrawn |
| Q12 | ~~Does PR C clear `mcpServerIds` on the Insight flip?~~ | **MOOT (v14).** No gate, so nothing to clear and nothing to hide. `mcpServerIds` survives a type flip and the MCP card stays visible for every type |

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
- ~~**D13**~~ **VOIDED in v14.** `ValidateMcpServerIDs` takes no AI type. 대표님
  confirmed this is intended ("인사이트도 mcp 사용가능하도록 해"): Insight AIs may use
  MCP servers, so the asymmetry with `tool_names` is deliberate, not a defect. PR A adds
  no type gate; it owes only one docs sentence stating the behavior (see §5 Q5).
- **D14** The three mcpserver webhook event types are undocumented, and
  `ai.McpServerIDs` carries `,omitempty` while the RST says it defaults to `[]`.
- **D15** `McpServerUpdate` discards `RowsAffected`, so **the `tm_delete` predicate
  D2 requires would turn a rejected PUT into a silent 200** with unchanged fields
  and a spurious `EventTypeUpdated`. Gating alone is not enough; the function must
  report "no row matched".
- **D16** `McpServerDelete` has no `tm_delete` predicate and ignores `RowsAffected`,
  so **delete is not idempotent**: a second DELETE re-stamps `tm_delete` with a
  fresh timestamp and re-publishes `EventTypeDeleted`.
- **D17** `auth_type: "oauth"` can be set directly via POST/PUT with a
  customer-supplied secret, which `ai_struct_mcpserver.rst:35` says is impossible
  and which no code enforces.
- **D18** `ValidateMcpServerIDs` runs **after** the AI write has already committed,
  so a rejected whitelist leaves an orphaned/mutated AI behind and still returns 400.
- **D19** `ai_struct_mcpserver.rst` documents a `9999-01-01` timestamp sentinel this
  resource does not use (it uses `null`), plus a bogus "(enum string)" label on
  `oauth_vendor`.
- **D20** `mcpserverhandler.Update` **short-circuits to `h.Get` before touching the
  DB** when every field is omitted, so a `PUT {}` on a deleted server returns 200
  with the full row — a path NO dbhandler-level gate can reach.
- **D21** An AI holding a deleted MCP server id must stay saveable. Write-time
  validation **skips ids already in the stored whitelist and rejects only newly added
  ones** (§5 Q11, 대표님 확정). Without the skip, every save of that AI 400s forever,
  for any unrelated edit; without the reject, attaching a deleted server fails
  silently. This is a PR A ↔ PR C cross-boundary defect that neither PR sees alone.
- ~~**D22**~~ **VOIDED in v14** by the Q5 reversal — no AI-type gate exists, so a gate
  that under-enforces and freezes AIs cannot occur. Retained in §3.12b as the fourth
  recorded instance of the recurring failure mode. Not PR A or PR C scope.
- ~~**D23**~~ **VOIDED in v14** by the Q5 reversal — no gate to bypass, no row to
  freeze; its three open decisions are withdrawn. Retained in §3.12e as the fifth
  recorded instance, created by the fix for the fourth. Not PR A or PR C scope.

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

### 3.7a D15: gating `McpServerUpdate` silently succeeds. CONFIRMED. BLOCKER for §5 Q1.

```go
// pkg/dbhandler/mcpserver.go:154-156 — result discarded
if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
	return fmt.Errorf("McpServerUpdate: could not execute. err: %v", err)
}
return nil
```

Adding `tm_delete IS NULL` to the `Where` makes the statement match zero rows and
return `nil`. `mcpserverHandler.Update` then continues:

```go
// pkg/mcpserverhandler/handler.go:206-217
if err := h.db.McpServerUpdate(ctx, id, fields); err != nil { ... }
res, err := h.db.McpServerGet(ctx, id)      // does NOT filter tm_delete
...
h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, mcpserver.EventTypeUpdated, res)
return res, nil
```

So a PUT against a deleted server would return **200 with unchanged fields and a
spurious `EventTypeUpdated` webhook**. Today's visible resurrection becomes a
silent success, which is arguably worse: the customer believes the update applied.

The same mechanic defeats the OAuth `Complete` gate, since `complete.go:126` writes
through `McpServerUpdate`.

**Therefore §7's promised tests ("rejected by `McpServerUpdate`", "rejected by
OAuth `Complete`") are unachievable by gating alone.** PR A must additionally make
`McpServerUpdate` inspect `RowsAffected` and return `ErrNotFound` on zero. That is a
behavior change affecting **every** existing caller, including
`mcpoauthhandler/access_token.go:101-108`'s token rotation — see §6's new risk row.

This is the same mechanic §5 Q2 already established for credential zeroing; v5
failed to carry it into Q1.

### 3.7b D16: delete is not idempotent. CONFIRMED.

`McpServerDelete` (`pkg/dbhandler/mcpserver.go:163-182`) has no `tm_delete`
predicate and also ignores `RowsAffected`, so a second DELETE **re-stamps
`tm_delete` with a fresh timestamp** and `handler.Delete:242` re-publishes
`EventTypeDeleted`.

This is unavoidable for PR A because §5 Q2 mandates zeroing credentials **in this
exact statement**, so PR A is rewriting it regardless. Interacts with the
api-validator cleanup fixture's documented repeat-DELETE tolerance
(`test_mcpservers_lifecycle.py:249-252`), so the decision must preserve a 200 on
repeat DELETE even if the timestamp stops moving. Recommend: add the predicate,
keep the handler's response 200 (idempotent), stop re-publishing the webhook.

### 3.7c D17: `auth_type: "oauth"` is settable directly, contradicting the docs. CONFIRMED.

`ai_struct_mcpserver.rst:35` states `oauth` "can never be set directly via
POST/PUT." Nothing enforces that:
- `AuthType.IsValid()` (`models/mcpserver/main.go:23-30`) **accepts**
  `AuthTypeOAuth`.
- `mcpserverhandler.Create:48-50` and `Update:154-156` check only `IsValid()`.
- `bin-api-manager/server/mcpservers.go:91-94` (POST) and `:185-189` (PUT) cast the
  raw string straight through.
- **There is no OpenAPI request-validator middleware.** `grep -rn
  'OapiRequestValidator\|openapi3filter\|ValidateRequest' bin-api-manager`
  (excluding `vendor/`) → **zero hits**. The chain at
  `cmd/api-manager/main.go:366-378` is RequestID / RateLimit / Authenticate /
  EnforceAccountStatus / DirectResourceScope only. So the generated enum
  (`gens/openapi_server/gen.go:3550-3566`) and the spec enums
  (`paths/mcpservers/main.yaml:57`, `id.yaml:74`) are **decorative**.

Result: `POST /mcpservers {"auth_type":"oauth","secret":"x"}` succeeds, stores a
customer-supplied secret on an `oauth` row, and reports `has_secret: true`, while
`buildAuthHeader`'s oauth branch (`mcptoolhandler/client.go:90-101`) ignores
`SecretCiphertext` and fails on an empty `AccessTokenCiphertext`
(`access_token.go:38`). A silently broken server the customer cannot diagnose.

Same defect class as D2 (a missing write-time gate), on the same docs sentence D5
edits. **Decision needed (§5 Q9): add the gate, or correct the sentence.**

### 3.7d D18: whitelist validation runs after the AI write commits. CONFIRMED.

```go
// pkg/listenhandler/v1_ais.go:92-128 (POST) — Create FIRST
tmp, err := h.aiHandler.Create(ctx, req.CustomerID, ... req.ToolNames, ...)
if err != nil { ... }

if req.McpServerIDs != nil {
	if err := h.aiHandler.ValidateMcpServerIDs(ctx, tmp.CustomerID, *req.McpServerIDs); err != nil {
		return errorResponse(err), nil        // AI already exists
	}
	tmp, err = h.aiHandler.UpdateMcpServerIDs(ctx, tmp.ID, *req.McpServerIDs)
```

`mcp_server_ids` is not even a `Create` parameter; it is applied by a **second**
call. So a rejected whitelist returns 400 while the AI **has already been
created** (POST) or mutated (PUT `:247-283`), leaving an orphan. Note `tool_names`
IS a `Create` parameter and IS validated inside the write.

**v14:** the earlier argument here — that D13's type gate rewrites this signature
anyway, making the reordering nearly free — is withdrawn with D13 (§5 Q5). D18 now
stands alone: the orphan path is still real, but PR A must justify the reordering on
its own merits. Q11's diff logic touches the same two blocks, which is the remaining
reason to do it here.

### 3.7e D19: the RST documents a timestamp sentinel this resource does not use. CONFIRMED.

`ai_struct_mcpserver.rst:53` asserts that `tm_delete` = `9999-01-01
00:00:00.000000` means "not deleted," and the example block at `:71-72` prints that
value for **both** `tm_update` and `tm_delete`. The API cannot produce that output:
- `models/mcpserver/main.go:98` is `TMDelete *time.Time` (pointer, so `null` on the
  wire).
- `ai_mcp_servers` was created 2026-09-11
  (`9b0ad37e0360_ai_mcp_servers_create_table.py:22-42`), **after**
  `071504ef41d0_timestamp_sentinel_to_null.py` retired the sentinel, and the table
  is absent from that migration's `TABLE_COLUMNS` (`:26-71`).
- api-validator confirms the real contract: `test_mcpservers_lifecycle.py:266,274`
  assert `tm_delete is not None` rather than the sentinel comparison older
  resources use (`test_ai_lifecycle.py:66`).
- Correct precedent in the same docs tree: `talk_struct_talk.rst:37`,
  `customer_struct_customer.rst:67`.

Also in the same file: `:37` labels `oauth_vendor` "(enum string)", but the model
field is a plain `string` (`models/mcpserver/main.go:82`) populated from the
runtime-configurable vendor catalog (`internal/config/main.go:151-154`) with no
`IsValid`. And neither the Status table (`:79-86`) nor the delete note states what
DELETE does to tool access — which becomes the headline customer-visible change once
PR A lands, so it belongs in this file.

All of the above sit in the same note block and example that D5 already edits, so
PR A fixes them in the same pass.

### 3.7f D20: the empty-PUT path bypasses every DB-level gate. CONFIRMED. BLOCKER for §5 Q1.

`mcpserverhandler.Update` builds a field map from nil-able pointers and then:

```go
// pkg/mcpserverhandler/handler.go:194-204
if len(fields) == 0 {
	// A PUT with every field omitted … is a client no-op, not a server error.
	// … Reuses Get's existing ErrNotFound -> cerrors.NotFound mapping …
	return h.Get(ctx, id)
}

if err := h.db.McpServerUpdate(ctx, id, fields); err != nil { ... }
```

A `PUT /mcpservers/{id}` with an empty or all-null body yields `len(fields) == 0` —
every field is a nil pointer end to end
(`bin-api-manager/server/mcpservers.go:179-191` →
`requesthandler/ai_mcpservers.go:122-130` → `listenhandler/v1_mcpservers.go:187-197`)
— so it **never calls `db.McpServerUpdate`** and returns 200 with the deleted row's
full body. An existing test pins this: `pkg/mcpserverhandler/handler_partial_update_test.go:115-148`
(`Test_Update_AllFieldsOmitted_IsANoOp`) asserts `db.McpServerUpdate` is not called.

**Consequence: §5 Q1's mandated fix does not close PUT-against-a-deleted-server.**
The `tm_delete IS NULL` predicate plus the D15 `RowsAffected` check both live below
a branch this request never enters, and Q1's own forbid-clause rules out the only
other interception point (`dbhandler.McpServerGet`). After PR A lands exactly as
v6 wrote it, `PUT {}` on a deleted server still answers 200 with the row.

Fix: an explicit existence gate inside `mcpserverhandler.Update` **ahead of** the
`len(fields) == 0` branch — and, per §3.7g, ahead of `ValidateURL` too.

This is the same failure shape as D15 one layer higher: **a prescribed fix that does
not reach the path it was meant to close.**

### 3.7g D20 corollary: validation precedes existence, so the error contract is inconsistent.

`ValidateURL` runs at the very top of `Update` (`handler.go:146-150`), before any
field assembly and far before the DB write:

```go
// pkg/mcpserverhandler/handler.go:146-150
if url != nil {
	if err := ValidateURL(*url); err != nil {
		return nil, cerrors.InvalidArgument(..., "INVALID_MCP_SERVER_URL", ...)
	}
}
```

`status.IsValid()` (`:151-153`) and `authType.IsValid()` (`:154-156`) follow, also
before existence is known. So on a **deleted** server: a PUT with a malformed URL
returns **400 `INVALID_MCP_SERVER_URL`** while the same PUT with a valid URL would
return 404 once gated. That is both inconsistent and a weak existence oracle (a
caller can probe which ids exist by varying only the URL's validity).

The gate must precede all three validations.

**Error code choice (must be decided, §5 Q10).** Both candidates terminate at 404,
so there is no 500 anywhere — verified plumbing:
- bare `dbhandler.ErrNotFound` → `listenhandler/main.go:194-195` →
  `requesthandler.ErrNotFound` → `server/error_translate.go:83-84` → **404
  `RESOURCE_NOT_FOUND`**
- typed `cerrors.NotFound` → `listenhandler/main.go:184-189` → typed passthrough
  (`error_translate.go:52-56`) → **404 `MCP_SERVER_NOT_FOUND`**

`handler.Get:97-101` already emits `MCP_SERVER_NOT_FOUND` for a nonexistent id, so
using the same code for "deleted" keeps the two indistinguishable — which is the
correct privacy posture, but it must be a deliberate choice, not an accident of
which error type the implementer happens to return.

### 3.7h What CANNOT be gated: `mcpserverhandler.Get` has four consumers pulling opposite ways.

§5 Q1 forbids gating `dbhandler.McpServerGet` and explains why. The **handler-level**
`Get` (`handler.go:93-107`) is a different function and equally ungateable, for a
different reason — its four non-test consumers have contradictory requirements:

| Consumer | Requirement |
|---|---|
| `pkg/listenhandler/v1_mcpservers.go:142` | customer GET — **must keep returning 200** for a deleted row (§4 non-goals, validator-tested) |
| `pkg/aicallhandler/mcp_tool.go:73` (resolution) | must **fail closed** |
| `pkg/aicallhandler/mcp_tool.go:157` (dispatch) | must **fail closed** |
| `pkg/mcpserverhandler/handler.go:203` | D20's empty-PUT short-circuit — must fail closed |

So the defense-in-depth gates §5 Q1 lists must be written **in `mcp_tool.go`
itself**, not in the shared `Get`. The design doc has to say this outright: an
implementer who "helpfully" gates `Get` breaks the GET-after-DELETE 200 contract,
which is §6's top risk row one layer up.

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
path. **RETRACTED in v8:** v5-v7 called the residual UI behavior "a PR C display
decision, not a data problem." That was true of today's code and is **false once PR
A's whitelist gate lands** — see §3.12a (D21).

### 3.12a D21: PR A's whitelist gate freezes any AI holding a deleted id. CONFIRMED. BLOCKER.

Three prescriptions in this document, each defensible alone, combine into a
customer-facing dead end. Traced forward through square-admin:

1. **The stale id is still there.** §5 Q6 decides NOT to prune
   `ai.mcp_server_ids` on delete. `ais_detail.js:242` hydrates `mcpServerIds` from
   the GET response, which still contains the deleted id.
2. **The user cannot see or remove it.** The picker at `ais_detail.js:1252-1271`
   iterates `mcpServersList.map(...)` and derives `selected` from
   `mcpServerIds.includes(server.id)`. Since the list comes through the
   `deleted:"false"` filter (`servicehandler/mcpserver.go:67-70`), the deleted server
   is **absent from the iteration entirely** — no row, no checkbox, no
   `toggleMcpServer` affordance. The id is invisible and unremovable from the UI.
3. **Every save re-submits it.** `ais_detail.js:406-422` builds the PUT body with
   `mcp_server_ids: mcpServerIds` **unconditionally** (verified: the field sits in
   the literal alongside `tool_names`, with no conditional wrapper).
4. **PR A then rejects it.** A non-nil array reaches `server/ais.go:294` →
   `v1_ais.go:272-276` always calls the now-gated `ValidateMcpServerIDs` →
   `INVALID_MCP_SERVER_ID` → **400 on every save of that AI, forever**, including
   edits to completely unrelated fields (name, prompt, engine, TTS).
5. **The user gets no hint why.** `ais_detail.js:551-553` computes dirtiness by
   comparing old vs new whitelist, so the dead id never marks the whitelist section
   dirty. The failure surfaces while editing something else, as the generic
   "Could not update the AI configuration" at `:445`.
6. **D18 compounds it.** `aiHandler.Update` at `v1_ais.go:247` has **already
   committed** before validation runs, so the unrelated edits persist while the user
   is told the save failed. If §5 Q5's recommendation (move validation ahead of the
   write) is adopted, nothing persists and the AI is simply frozen.

The same trap propagates to `teamgraph/sidebar.js:753-782` the moment PR C adds the
field there, and `sidebar.js` has no `mcpservers` fetch of its own (§4).

**This is the third instance of this document's recurring failure mode** (after D15
and D20): a fix traced only to its own layer, never forward to the surface that
consumes it. Here the two layers are in **different repositories**, which is why
five rounds of review missed it.

**Resolution is a decision, not an implementation detail — RESOLVED in §5 Q11:**
skip ids already in the stored whitelist, reject ids newly added. The diff is free
because `tmp.McpServerIDs` at `v1_ais.go:272` is still the unmodified stored list
(`aiHandler.Update` at `:247` does not take `McpServerIDs`). With that, none of the
six steps above fires: step 4 no longer rejects, so steps 5 and 6 never trigger.

square-admin reads `has_secret` only from GET
responses (`mcpservers_list.js:86`, `mcpservers_detail.js:266`) and does not consume
mcpserver webhooks, and it already handles `auth_type: 'oauth'`
(`mcpservers_detail.js:38`), so §5 Q2's `has_secret` flip breaks nothing.

### 3.12b D22: VOIDED in v14 — Q5's Insight gate enforces nothing where it matters and freezes AIs elsewhere

**STATUS: VOIDED, not fixed.** 대표님 reversed the underlying policy in v14 ("인사이트도
mcp 사용가능하도록 해") — Insight AIs may use MCP servers, so no AI-type gate is added
and this defect cannot occur. The analysis below is retained verbatim because it is the
fourth recorded instance of this document's recurring failure mode and its reasoning is
what led to the reversal: the gate could not be placed anywhere that both enforced the
policy and left customers able to save. **Nothing here is PR A or PR C scope.**

**This is the FOURTH instance of this document's recurring failure mode** (after D15,
D20, D21), and it was NOT the instance the author self-flagged. Q5 prescribes a
write-time AI-type gate. Traced forward in both directions, the write-time layer is
the one layer that cannot deliver the policy.

**(a) It does not enforce the policy for existing rows.** The consume path has NO type
check. `grep -n 'Type' pkg/aicallhandler/mcp_tool.go` returns exactly one line,
`:144` (`resolveAI`), and never `a.Type`. Resolution iterates the whitelist blind to
type:

```go
// pkg/aicallhandler/mcp_tool.go:72
for _, serverID := range a.McpServerIDs {
	server, err := h.mcpServerHandler.Get(ctx, serverID)
```

Insight sessions reach this through `insight_session.go:104` → `:143`. Combined with
**Q6's decision never to prune**, any Insight AI that already stores
`mcp_server_ids` — or any Normal AI later flipped to Insight — keeps resolving and
`ListTools`-calling those servers forever. §7's promised test ("`TypeInsight` cannot
whitelist an MCP server") would pass while the policy is unenforced in production.

**(b) It reintroduces D21 verbatim, and Q11's skip does NOT cover it.** Q11 exempts
already-stored ids from the **deleted-server** predicate only. The type predicate is
different, so the exemption does not apply. In square-admin:

```javascript
// ais_detail.js:738-743 — switching to insight clears TOOLS but not MCP servers
onValueChange={(value) => {
  setAiType(value);
  if (value === 'insight') {
    setSelectedTools([]);
  }
}}
```

`mcpServerIds` is never cleared, `:421` puts it unconditionally, and the MCP Servers
card at `:1238` is rendered with **no `aiType` condition** — unlike
`AIEngineFields.js:301`/`:320`, which do gate tool sets on `aiType === 'insight'`. So a
customer flipping an existing AI to Insight sends `type: "insight"` plus a non-empty
`mcp_server_ids` and receives a hard 400 behind the generic "Could not update the AI
configuration" (`:445`). The dead end is identical to D21 and lands in the repo PR A
does not touch.

Resolution: **none needed — VOIDED in v14.** 대표님 reversed the policy, so no gate is
built and this defect cannot occur.

### 3.12c Q9's transition gate needs a read that does not exist, and its frontend safety is accidental

**The pre-image is missing.** `mcpserverhandler.Update` (`handler.go:138-192`) builds
its field map purely from incoming pointers; the only `McpServerGet` is the
**post-write** read-back at `:210`. So "validate the transition, not the value"
(§5 Q9) requires a pre-write row read that does not exist today. **It is the same read
D20's existence gate needs** (§5 Q1) and the same one §3.7g's precedence fix needs.
The design doc must state that **one** pre-write `McpServerGet` serves all three —
otherwise three implementers add three reads, or one adds none and silently falls back
to a value gate, reproducing the freeze Q9 explicitly warns about.

**The frontend is compatible, but by accident.** §3.12a's earlier claim that
square-admin "already handles `auth_type: 'oauth'`" is true and materially incomplete.
`mcpservers_detail.js:156` re-submits `auth_type: buildAuthTypeWireValue(authType)`
**unconditionally** on every save, and `:38`/`:91` hydrate `'oauth'`, so an OAuth row's
PUT always carries `auth_type: "oauth"`. A value gate would 400 every edit of a
Connected server. The transition gate survives only because `AUTH_TYPE_OPTIONS`
(`:31-35`) omits `oauth`, so the form cannot move a row INTO oauth; and `:359-366`
renders oauth as a read-only badge, so it cannot move a row OUT of oauth either. **Both
are load-bearing UI facts that PR A's tests do not pin.** If either changes, the gate
breaks. Record them, and assert the transition semantics in a backend test rather than
relying on the form's option list.

### 3.12d Q11 implementation constraints (non-blocking, but must be written down)

- **The diff is SET membership, not sequence comparison.** `mcpServerIds` is
  order-dependent in the frontend (`ais_detail.js:382-386` appends on toggle; `:553`
  sorts only for the dirty check, never for submission). A `slices.Equal`-style
  comparison would misclassify a reordered list as "newly added." Build a
  `map[uuid.UUID]struct{}` of the stored list and test membership. Duplicates in the
  incoming list and the explicit empty-slice clear (pinned by
  `ais_detail.test.js:690-700`) both fall out correctly from set membership; no extra
  decision is needed.
- **POST has no diff code path at all.** `mcp_server_ids` is not a `Create` parameter
  (`chatbot.go:20-39`), so the POST rule is simply "reject all deleted ids." An
  implementer writing one shared helper must not have it read a stored list that does
  not exist.
- **Both internal oauth writes are exempt, not one.** §5 Q9 cites `complete.go:136-158`
  (new row). The **reconnect branch also writes it**: `complete.go:112-133` sets
  `FieldAuthType: AuthTypeOAuth` at `:117` via `McpServerUpdate` at `:126`. Neither
  passes through `mcpserverhandler`, so the decision is unchanged, but §7 must assert
  both survive the gate.

### 3.12e D23: VOIDED in v14 — the Insight gate is unreachable by the request that creates the violating row

**STATUS: VOIDED, not fixed.** Same reversal as D22: there is no AI-type gate, so there
is nothing to bypass and nothing to freeze. Its three open decisions (flip semantics,
empty-list exemption, on-load cleanup) are **withdrawn**. Retained verbatim as the fifth
recorded instance of the recurring failure mode — it was created by the fix for the
fourth, which is the strongest evidence in this document that the gate was the wrong
shape. **Nothing here is PR A or PR C scope.**

**FIFTH instance of this document's recurring failure mode** (after D15, D20, D21,
D22), found in the v12 material that was written to close the fourth. v12 was verified
for *reachability of the gate points*; it was not verified for *reachability of the
gate by the offending request*.

**(a) A type flip that OMITS the whitelist bypasses the gate entirely.** Both call
sites of `ValidateMcpServerIDs` sit inside `if req.McpServerIDs != nil`
(`v1_ais.go:117` and `:272`) — Q11's own table codifies "field omitted → no
validation". So:

```
PUT /ais/{id}  {"type": "insight"}     // mcp_server_ids omitted
```

flips the type (`chatbot.go:145-149` resolves and writes it), never enters the
`req.McpServerIDs != nil` block, and leaves the stored non-empty whitelist untouched
(Q6 never prunes; `aiHandler.Update`'s parameter list excludes `McpServerIDs`).
**Result: an Insight AI holding a non-empty whitelist, created AFTER the deploy,
through the public API, with nothing rejected.** Q5(iii)'s production count of zero
does not cover this — it is not a migration artifact, it is a new row.

**(b) That row is then permanently frozen, and Q5(ii)'s cleanup does not reach it.**
`ais_detail.js:197` hydrates `aiType = 'insight'` and `:242` hydrates
`mcpServerIds = [X]` on load. Q5(ii)'s clear lives in `onValueChange`
(`:738-743`), which fires only when the user **changes** the select — **not on load of
an already-Insight AI.** Meanwhile Q5(ii)'s other half hides the MCP card for Insight,
so there is no picker and no `toggleMcpServer` to remove `[X]`. `:421` still submits
`mcp_server_ids` unconditionally (pinned by `__tests__/ais_detail.test.js:690-726`), so
every subsequent save is a non-empty whitelist on an Insight AI → 400 → generic "Could
not update the AI configuration" (`:445`). Save is not dirty-gated (`:831`, `:840`
disable only on `isSaving`/`isDeleting`), so any unrelated edit hits it. **This is D21's
freeze reconstructed by the fix for D22.**

**(c) The gate's predicate is unspecified, and the obvious reading 400s every Insight
AI save.** `ais_detail.js:421` sends `mcp_server_ids` on every save; for an Insight AI
that value is `[]`. §7's wording ("`TypeInsight` cannot whitelist an MCP server") and
§5 Q5's "denied outright" read as "type is Insight → reject". Implemented literally,
`ValidateMcpServerIDs(…, TypeInsight, [])` rejects and **every save of every Insight
AI 400s**, including AIs that never had MCP data, and including the POST from
`ais_create.js` once PR C adds the field. **The empty list must be explicitly
exempt:** reject only a NON-EMPTY whitelist for Insight.

**(d) A third Insight-transition site exists.** Q5(ii) names two;
`ais_create.js:92` (`setAiType(template.type || 'normal')` in `handleTemplateSelect`)
is a third, reachable for `type: 'insight'` via `prompt_templates.js:659`
(`insight_case_assistant`, `:653`) and re-triggerable after the form is filled by the
"Change Template" button at `:186`. It clears tools but not MCP ids.

**Open decisions this creates (not implementation details):**
1. **What happens on a type flip to Insight while a whitelist is stored?** Options:
   (a) reject the flip while a non-empty whitelist is stored, telling the customer to
   clear it first; (b) auto-clear the whitelist server-side as part of the flip (one
   write, no webhook storm, unlike Q8's rejected bulk loop); (c) validate on the flip
   even when the field is omitted, which requires reading the stored list. Recommend
   **(b)** — it is the only option that cannot freeze a row, and the flip is already a
   deliberate destructive act on the tool set.
2. **Confirm the empty-list exemption** in both the gate and §7's assertions.
3. **PR C: clear `mcpServerIds` on load when the fetched AI is already Insight**, not
   only in `onValueChange`, and add the third transition site (`ais_create.js:92`).

### 3.13a Verified clean in round 6 — recorded so PR A does not over-scope

| Area | Finding |
|---|---|
| SSRF / URL validation | `ValidateURL` IS applied on both write paths: `mcpserverhandler/handler.go:38` (Create) and `:146-150` (Update, under `if url != nil`). Literal private/loopback/link-local addresses are rejected (`ssrf.go:31-57`, `rejectDisallowedIP:63-77`), and the DNS-rebinding case is closed at dial time by `controlRejectDisallowedAddr` (`ssrf.go:117-133`) via the shared guarded client (`mcptoolhandler/client.go:140-144`). **Nothing for PR A to add** beyond §5 Q10's precedence fix |
| Key rotation | **No rotation or re-encryption job exists anywhere in the repo.** Rotation is config-side and decrypt-by-row-version (`mcpserverhandler/secret.go:123-130`, `NewSecretCrypto:79-89`); nothing iterates rows, so zeroed rows would be encountered by no job. Credential zeroing is safe on this axis |
| Caller-set completeness | Full non-test, non-mock enumeration. `db.McpServerUpdate`: exactly 3 callers (`mcpserverhandler/handler.go:206`, `mcpoauthhandler/access_token.go:108`, `mcpoauthhandler/complete.go:126`) — all named in §5 Q1/§6. `db.McpServerDelete`: exactly 1 (`handler.go:226`) — named. `db.McpServerGet`: 10; the three not named in this analysis (`handler.go:82`, `complete.go:129`, `complete.go:161`) are post-write read-backs of a row the same function just wrote, harmless once `:126` is gated. **No caller of consequence is unmentioned** |
| Concurrency | No transaction or row lock on any mcpserver path — `McpServerDelete` is a bare UPDATE, unlike `dbhandler/ai.go:243`+`:294` and `aipromptproposal.go:227`+`:253` which use `BeginTx` + `FOR UPDATE`. For delete-vs-tool-call, the fail-closed re-read per call (`client.go:191`, `:214`) is **sufficient**: the residual window is at most one already-dispatched outbound request. **No transaction warranted.** State this bound in the PR body, since `mcpservers_detail.js:496` promises immediacy |
| Tool-path error surface | `toolHandleMcpCall` converts every failure into a generic `fillFailed(...)` tool result (`mcp_tool.go:137-174`), so gating never leaks a status code to a customer through the AI path |

### 3.14 Other surfaces

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
D1, D2, D3, D4, D5, D6, D12, D14, **D15, D16, D17, D18, D19, D20, D21**.
(D13, D22, D23 voided in v14 — Insight AIs may use MCP servers.) One
logical unit: "a deleted or
foreign MCP server must be inert everywhere, write paths must reject rather than
silently succeed, and the docs/config must stop describing things that are not true."

**PR C — `voipbin/monorepo-javascript` — square-admin parity.**
D8 + `types/api.ts` + the CLAUDE.md Field Sync Points table row + re-verification
of the `mcpservers_detail.js:496` delete-dialog copy (PR A makes it true).
**Per the Field Sync Points rule this covers FOUR form bodies, not one:**
`ais_create.js:114-129`, and `teamgraph/sidebar.js` AI-create `:514-528` and
AI-edit `:753-768` (both verified to carry `tool_names` and omit
`mcp_server_ids`), plus `ais_detail.js`'s existing implementation left intact.
`sidebar.js` has no `mcpservers` fetch of its own, so the
`ProviderGet('mcpservers?page_size=100')` call at `ais_detail.js:373-377` (a bare
`useEffect`, token-scoped, no extra customer context needed) must be reused there.
**The Insight-transition cleanup prescribed in v12 is WITHDRAWN (v14).** With no
AI-type gate, `mcpServerIds` must SURVIVE a type flip and the MCP Servers card
(`ais_detail.js:1238`) must stay visible for every AI type — including Insight, since
Insight AIs may now use MCP servers. PR C must not add `aiType === 'insight'`
conditions around MCP state or the MCP card.
**Independent of A again; may proceed in parallel.**

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
   **RESOLVED sub-decision (D15): gating `McpServerUpdate` is necessary but NOT
   sufficient.** Because it discards `RowsAffected`
   (`dbhandler/mcpserver.go:154-156`), a predicate alone yields a silent 200 plus a
   spurious `EventTypeUpdated` (§3.7a). PR A must make `McpServerUpdate` inspect
   `RowsAffected` and return `ErrNotFound` on zero. **Open:** whether that applies
   to every caller uniformly, notably `access_token.go:101-108`'s token rotation,
   where "no row matched" now surfaces as an error instead of being swallowed.
   **Also open (D16):** does `McpServerDelete` gain the same predicate? Recommend
   yes, keeping the handler's repeat-DELETE response at 200 (idempotent) and
   dropping the duplicate `EventTypeDeleted`, to stay compatible with the validator
   cleanup fixture (`test_mcpservers_lifecycle.py:249-252`).
   **MANDATORY ADDITION (D20): an existence gate inside `mcpserverhandler.Update`,
   placed ahead of BOTH the `len(fields)==0` short-circuit (`handler.go:194-204`)
   AND the `ValidateURL`/`status.IsValid`/`authType.IsValid` block
   (`handler.go:146-156`).** Without it, `PUT {}` on a deleted server still returns
   200 with the row (§3.7f) and a malformed-URL PUT returns 400 instead of 404
   (§3.7g). The existing test `handler_partial_update_test.go:115-148` pins the
   no-op behavior and will need a deleted-row counterpart.
   **ALSO EXPLICITLY FORBIDDEN (D20 corollary): gating `mcpserverhandler.Get`
   (`handler.go:93-107`).** Its four consumers have contradictory requirements
   (§3.7h): `v1_mcpservers.go:142` must stay 200, while `mcp_tool.go:73`/`:157` and
   `handler.go:203` must fail closed. The defense-in-depth gates therefore live in
   `mcp_tool.go` itself.
2. **Credential zeroing mechanics. RESOLVED: same statement as the delete
   timestamps.** A separate `dbhandler.McpServerUpdate` after `McpServerDelete`
   would be **silently no-op'd** by the `tm_delete IS NULL` predicate Q1 mandates,
   because `McpServerUpdate` ignores `RowsAffected` (`dbhandler/mcpserver.go:154-156`).
   Confirmed acceptable side effect: `has_secret` flips to false on the
   `EventTypeDeleted` payload (`mcpserverhandler/handler.go:242`) and on the
   GET-after-DELETE response; no validator test asserts it (§3.14), no square-admin
   consumer breaks (§3.12), no documented contract exists (§3.7), and it is truthful. **Sub-decision (v7): `key_version` is zeroed alongside the ciphertext
 columns.** §3.4 lists it among the retained columns and Q2 previously omitted it.
 Precedent exists in-file: the explicit secret-clear path already sets
 `FieldKeyVersion = 0` next to the nil ciphertext/nonce
 (`mcpserverhandler/handler.go:178-182`). Delete should match.
3. **Naming collision** (`uvx voipbin-mcp` = VoIPBin-as-MCP-server vs
   customer-registered MCP server): rename the `ai_overview.rst:681` heading only,
   or also `skill.md`/`llms.txt`? And should `ai_overview.rst:685` keep pointing at
   the third-party fork rather than `github.com/voipbin/mcp`?
4. **api-validator coverage** (third repo `monorepo-monitoring`, therefore a fourth
   PR): `mcp_server_ids` on `POST /ais`, revocation behavior after DELETE, OAuth
   start/complete against a deleted server, `has_secret` false after DELETE. Add now
   or defer?
5. **D13 policy: are MCP servers permitted on `TypeInsight` AIs?**
   (v10 confirmed "add the AI-type gate" — "Q5. 추가하도록 하자." — then)
   **REVERSED in v14 — 대표님 확정: "인사이트도 mcp 사용가능하도록 해."** Insight AIs
   MAY use MCP servers. **No AI-type gate is added anywhere**, so:
   - `ValidateMcpServerIDs` does NOT gain a `Type` parameter, and neither call site
     changes signature.
   - **No consume-path type gate.** `mcp_tool.go:72` and `:151` keep only D1's
     `tm_delete`/status gates.
   - **No PR C Insight-transition cleanup and no card hiding.** `mcpServerIds` survives
     a type flip, and the MCP Servers card stays visible for every AI type.

   **Three defects are VOIDED by this decision, not fixed:**
   - **D13** (Insight AIs may whitelist arbitrary MCP servers) — no longer a defect;
     it is the intended behavior.
   - **D22** (§3.12b: a write-time-only gate enforces nothing and freezes flipped AIs)
     — moot; there is no gate.
   - **D23** (§3.12e: a whitelist-omitting type flip bypasses the gate and permanently
     freezes the row) — moot; there is no gate. The three open decisions D23 raised
     (flip semantics, empty-list exemption, on-load cleanup) are all withdrawn.

   **What the decision means, stated honestly.** `AllowedToolNames(TypeInsight)`
   returns `tool.AllInsightToolNames`, and `models/tool/main.go:74-88` requires every
   entry to have "no side effects outside the session's own message/expression
   surface". An arbitrary customer MCP server carries no such guarantee, so MCP tools
   on an Insight AI are NOT bound by the Insight tool contract. That asymmetry is now
   deliberate: **the built-in tool set stays curated; MCP servers are the customer's
   own trust decision on their own AI, whatever its type.** PR A must NOT silently
   reintroduce the restriction, and §7 must not assert it.

   **PR A owes one docs sentence** (D14/§4 docs scope): state plainly that
   `mcp_server_ids` applies to every AI type including Insight, and that MCP tools are
   outside the curated Insight tool set. Without it, a reader comparing
   `AllInsightToolNames`' comment against observed behavior will reasonably file this
   as a bug.

   **Deferred, not decided against:** per-server or per-tool capability limits (already
   a §4 non-goal). If Insight-specific restriction is ever wanted, that is the honest
   mechanism — not the AI-type gate this decision reverses.

   **Coupled decision (D18): does validation move AHEAD of the AI write?** Today
   `ValidateMcpServerIDs` runs after `aiHandler.Create`/`Update` has committed
   (`v1_ais.go:92-128`, `:247-283`), so a rejected whitelist leaves an orphaned or
   already-mutated AI and still returns 400. **Re-scoped in v14:** the Q5 reversal
   removes the signature rewrite that made this "nearly free", so D18 now stands on its
   own merits. It is still worth doing — an orphaned AI on a rejected whitelist is a
   real defect — but it is now an independent change, and the design doc may legitimately
   defer it. Recommend still fixing it in PR A, since D21/Q11's diff logic touches the
   same two blocks anyway.

   **Where the validation must move, if D18 is taken.** `aihandler/chatbot.go:135-139`
   pre-fetches unconditionally ("Pre-fetch unconditionally so all three branches can
   detect changes", `preUpdateAI, errGet := h.db.AIGet(ctx, id)`) before any `AIUpdate`,
   so reordering costs no extra read **provided validation moves INTO
   `aiHandler.Update`**, where `preUpdateAI.McpServerIDs` is in hand. Merely swapping
   the two statements in `v1_ais.go` does not work: that frame has no row read, so
   Q11's diff would lose the stored whitelist it compares against and force a second
   `AIGet`. D18 also destroys `tmp.CustomerID` (passed today at `v1_ais.go:118`,
   `:273`), so the POST path must use the request's customer id and the PUT path must
   take it from `preUpdateAI`.

   **v14 note on a now-moot hazard.** v11-v13 warned at length that an omitted `type`
   decodes to `TypeNone` (`listenhandler/models/request/ais.go:17`, `:56`) and would be
   denied by a deny-by-default `default:` branch, 400ing every type-omitting PUT. **That
   hazard disappears with the Q5 reversal** — no type is passed to
   `ValidateMcpServerIDs` at all. Retained here only so a future reader who revisits
   AI-type gating knows the trap exists: resolution (`TypeNone` → stored →
   `TypeNormal`) happens at `chatbot.go:145-149`, i.e. after the `AIGet`, so any
   type-keyed validation must live downstream of that point.
   **Also corrected in v13 and still worth recording:** v11/v12 cited
   `api-validator/tests/scenarios/test_ai_lifecycle.py:38-48`/`:96-106` as live proof of
   type-omitting traffic. The citation was FALSE — the module is skipped at
   `test_ai_lifecycle.py:4`. **No api-validator test breaks under any gate in this
   document**: `test_mcpservers_lifecycle.py` and `test_mcpservers_oauth.py` never set
   `auth_type: "oauth"` and never touch `mcp_server_ids`.

9. **D17: is `auth_type: "oauth"` allowed on POST/PUT?** **RESOLVED — 대표님 확정:
   reject it (option a).** ("좋아, 네 제안대로 가자")

   The code already declares this the correct behavior and only the implementation
   failed to follow: `models/mcpserver/main.go:17-19` says "never set directly via
   POST/PUT with a customer-supplied secret," and the OpenAPI spec enumerates only
   `["", "bearer", "api_key"]` (`paths/mcpservers/main.yaml:57`, `id.yaml:74`). But
   `validAuthTypes` at `:23-25` includes `AuthTypeOAuth: true`, the write paths check
   only `IsValid()` (`handler.go:48-50`, `:154-156`), and there is no OpenAPI
   request-validator middleware in `bin-api-manager` to fall back on (§3.7c). Option
   (b) — relaxing the docs to match — would deliberately make an already-correct
   document wrong.

   **What the defect produces today:** the row saves with 200 and lists as `active`,
   but the oauth branch resolves its credential through
   `oauthHandler.GetValidAccessToken` (`client.go:98-101`), which fails on an empty
   access token. The customer sees a healthy-looking server whose tools never fire,
   with nothing on the API surface explaining why. **Same class as Q11's
   silent-attach:** a write that succeeds and does nothing.

   **Implementation constraints for the design doc:**
   - Gate `mcpserverhandler.Create`/`Update`, **not** `AuthType.IsValid()` — the
     OAuth completion path legitimately writes `oauth` internally
     (`mcpoauthhandler/complete.go:136-158`) and must keep passing model validation.
   - Follow Q11's shape: **validate the transition, not the value.** A server that
     legitimately completed OAuth must stay editable — a PUT changing only `name`
     that re-submits `auth_type: "oauth"` unchanged must NOT 400. Reject only when the
     request *moves* a row into `oauth`. Without this the gate reproduces D21's freeze
     on OAuth-backed servers.
   - Decide whether `Create` must also reject a `secret` when `auth_type` is `oauth`
     (the secret would be stored and then ignored by `client.go:90-101`).

   **Migration risk: none measured.** All 100 active production MCP servers are
   api-validator leftovers with `auth_type` `bearer` (75) or `""` (25) — **zero rows
   carry a customer-set `oauth`**, so no existing row is frozen by this gate. (Source:
   full cursor walk of `GET /v1.0/mcpservers` on `api.voipbin.net`, 2026-09-27; the
   leftover accumulation itself is tracked separately as ETC-18.)

   **Scope note:** the unenforced assertion appears TWICE in the same file, `:35`
   and again in the Auth Type table at `:100` ("Never set directly by the customer"),
   plus as a code comment at `models/mcpserver/main.go:17-19`. All three now become
   true statements and must be verified together rather than edited.
10. **D20/D15 error contract: which 404 do gated writes return?** Both candidates
   terminate at 404, so there is no 500 risk (§3.7g verified the plumbing), but the
   customer-visible code differs: bare `dbhandler.ErrNotFound` → `RESOURCE_NOT_FOUND`,
   typed `cerrors.NotFound` → `MCP_SERVER_NOT_FOUND`. Recommend `MCP_SERVER_NOT_FOUND`
   for consistency with `handler.Get:97-101`, which already returns it for a
   nonexistent id — keeping "nonexistent" and "deleted" indistinguishable is the right
   privacy posture, but it must be deliberate. Also confirm the gate precedes
   `ValidateURL` (`handler.go:146-150`) so a deleted row never answers 400.
   **v8 correction:** the recommended code is NOT what the D15 path currently
   produces. `handler.go:206-208` wraps with a plain `errors.Wrapf`, so a bare
   `dbhandler.ErrNotFound` from a `RowsAffected`-gated `McpServerUpdate` resolves via
   `listenhandler/main.go:194-195` to `RESOURCE_NOT_FOUND`. PR A must translate it at
   `handler.go:206` (mirroring `Get:96-101`), or the gate path and the TOCTOU race
   path return two different codes for the same condition.
6. **Does PR A prune stale ids from `ai.mcp_server_ids` on delete?** **RESOLVED: no.**
   Q11's skip-not-reject removes the reason pruning existed (D21's frozen AI), and the
   loop's own costs (no transaction on `McpServerDelete`, un-indexed JSON column, one
   `ai_updated` webhook per affected AI) are real. See Q11's "Rejected alternatives."
11. **D21 (RESOLVED — 대표님 확정): what happens when an AI's whitelist already
   contains a deleted id?** **Decision: skip what is already there, reject what is
   newly added.** ("skip-not-reject 방식으로 하자." + "이후에 AI 업데이트를 할 때
   ... 이때는 reject 를 해야하지 않을까?")

   | Path | Behavior | Rationale |
   |---|---|---|
   | POST `/ais` | reject ALL deleted ids | no prior list exists, so every id is new |
   | PUT with field omitted (`req.McpServerIDs == nil`) | no validation | current behavior, unchanged |
   | PUT, id already in the stored whitelist | **skip** | already-held; rejecting freezes the AI (D21) |
   | PUT, id newly added to the list | **reject** (`INVALID_MCP_SERVER_ID`) | attaching a deleted server is a user error and must not fail silently |

   **The diff costs nothing — both sides are already in hand at the gate.**
   `v1_ais.go:272` runs with `tmp` returned by `aiHandler.Update` at `:247`, and
   `Update`'s argument list (`:249-266`) does NOT include `McpServerIDs`. Therefore
   `tmp.McpServerIDs` is the **currently stored, not-yet-modified** whitelist, and the
   incoming list is `*req.McpServerIDs`. Computing "which ids are new" needs **zero
   extra DB reads and no signature change**. (An earlier draft of this document
   claimed an extra parameter was required; that was wrong.)

   **Why the diff is mandatory, not an optimization.** `ais_detail.js:406-440` always
   puts `mcp_server_ids` in the PUT body, so `req.McpServerIDs` is non-nil on every
   square-admin save. Without the diff, square-admin traffic must pass the gate
   unconditionally, which is exactly the D21 freeze. The "field omitted" branch only
   ever applies to direct API callers.

   **Why plain skip-everything was rejected.** It would silently accept "attach this
   deleted server to my AI now." The consume-time gate keeps that inert, so there is no
   security incident, but the customer believes the server is connected and never
   learns why its tools never fire — a silent failure. Rejecting only the newly added
   ids reports the error at the moment it is made.

   **Pointer semantics already support this** and are deliberate:
   `server/ais.go:68-74` documents that nil means "field omitted, leave the whitelist
   untouched" while a non-nil pointer to an empty slice means "explicitly clear it."
   PR A must not collapse that distinction.

   **Rejected alternatives.** (b) Prune stale ids on MCP delete: `McpServerDelete`
   (`dbhandler/mcpserver.go:163+`) is a bare UPDATE with no transaction, so a
   mid-loop failure leaves dead ids behind and the AI still frozen; `mcp_server_ids`
   is an un-indexed JSON column (`d8e342656cf0…:22`) with no reverse lookup, so the
   loop must read every AI of the customer; and each rewrite fires
   `ai.EventTypeUpdated` (`mcpserver_validation.go:86`), so deleting one MCP server
   emits a webhook per affected AI — the customer is told N AIs changed when they
   deleted a server. It also cannot clean rows that already exist without a
   migration. Since skip-not-reject fully resolves D21 on its own, the loop adds
   webhook noise, partial-failure handling, and a migration while solving nothing
   that remains. (c) PR C rendering removable "unavailable" entries: not needed for
   correctness; the stale id disappears naturally the next time the customer saves
   that AI, since the form submits only currently-valid ids.

   Applies equally to `teamgraph/sidebar.js:753-782` once PR C adds the field (§4).
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
| Insight AIs can whitelist arbitrary MCP servers while denied most built-ins | n/a | **NOT A RISK (v14).** 대표님 confirmed this is intended. MCP tools are outside the curated Insight tool set by design; PR A documents the asymmetry rather than gating it (§5 Q5) |
| ~~An Insight AI created by a type flip becomes permanently un-saveable~~ | n/a | **VOIDED (v14).** D23 depended on the AI-type gate, which no longer exists |
| Q9's transition gate has no pre-write row to compare against, and survives square-admin only because `AUTH_TYPE_OPTIONS` omits oauth and `:359-366` renders it read-only | Medium | §3.12c — one pre-write `McpServerGet` must serve D20, §3.7g and Q9 together; assert transition semantics in a backend test, not via the form's option list |
| `mcp_server_ids` grants all present and FUTURE tools of a server | Medium | One explicit docs sentence (§4 non-goals) |
| An AI whose whitelist holds a deleted id becomes un-saveable for ANY edit (400 on every PUT, no UI affordance to clear it, no dirty-state hint) | **High** | D21 (§3.12a) — **RESOLVED by §5 Q11**: skip already-stored ids, reject only newly added ones. Both directions must be unit-tested (§7) |
| Gating OAuth `Complete` discards a freshly-minted vendor grant: `complete.go:84` deletes the state row and `:88` completes the token exchange BEFORE `:126`, so VoIPBin holds a live GitHub/Linear token and drops it unrevoked | Medium | Preferable to writing onto a deleted row, but the orphaned vendor-side grant must be acknowledged in the PR body (and revocation considered) |
| `PUT {}` on a deleted server returns 200 with the row because `mcpserverhandler.Update` short-circuits to `h.Get` before any DB write; NO dbhandler gate reaches it | **High** | D20 (§3.7f). Gate inside `mcpserverhandler.Update` ahead of the `len(fields)==0` branch AND ahead of `ValidateURL` |
| An implementer gates `mcpserverhandler.Get` to fix the above and breaks the GET-after-DELETE 200 contract | **High** | §3.7h names its four contradictory consumers; §5 Q1 forbids it explicitly; defense-in-depth gates go in `mcp_tool.go` |
| Validation precedes existence, so a deleted row answers 400 `INVALID_MCP_SERVER_URL` on a malformed URL — inconsistent, and a weak existence oracle | Medium | D20 corollary (§3.7g) / §5 Q10 |
| Gating `McpServerUpdate` yields a SILENT 200 + spurious `EventTypeUpdated` instead of a rejection, because `RowsAffected` is discarded (`dbhandler/mcpserver.go:154-156`); the same defeats the OAuth Complete gate | **High** | D15 (§3.7a). PR A must return `ErrNotFound` on zero rows; §7's rejection tests are otherwise unachievable |
| Making `McpServerUpdate` honor `RowsAffected` changes behavior for EVERY existing caller, notably `access_token.go:101-108` token rotation | **High** | §5 Q1; enumerate all callers and add a test per caller for the new error path |
| `auth_type: "oauth"` settable via POST/PUT with a secret the oauth path ignores, producing an undiagnosable broken server; no OpenAPI validator middleware exists to catch it | **High** | D17 (§3.7c) / §5 Q9 |
| A rejected `mcp_server_ids` leaves an orphaned (POST) or already-mutated (PUT) AI behind and still returns 400 | Medium | D18 (§3.7d); fix ordering alongside D13's signature change |
| Repeat DELETE re-stamps `tm_delete` and re-publishes `EventTypeDeleted`; PR A is rewriting that exact statement for credential zeroing | Medium | D16 (§3.7b); keep repeat DELETE at 200 for the validator cleanup fixture |
| Docs assert a `9999-01-01` sentinel and an "(enum string)" type this resource does not have, and say nothing about what DELETE does to tool access | Medium | D19 (§3.7e); same note block D5 edits |
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
refused in transport, rejected by `ValidateMcpServerIDs` **when newly added and
skipped when already stored** (§5 Q11 — assert BOTH directions: a PUT adding a
deleted id 400s, and a PUT that merely re-submits an already-stored deleted id
succeeds), rejected by
`McpServerUpdate`, rejected by OAuth `Start`, rejected by OAuth `Complete`; that
delete zeroes all secret/token ciphertext in the SAME statement while update paths
never do; that dispatch refuses a server owned by another customer (D12); that the
team path validates the CURRENT member's whitelist; and `GetValidAccessToken`'s new
error when rotating against a row deleted mid-flight. **Q5 (D13) REVERSED in v14 — assert the OPPOSITE:** a test that an
Insight AI CAN hold and use `mcp_server_ids` (`ValidateMcpServerIDs` accepts it, and
`resolveTools` returns that server's tools for a `TypeInsight` AI). This is a
regression guard in the literal sense: it fails if anyone reintroduces the type gate
this decision removes. No `TypeInsight`-denial test, no unknown-`Type` test, and no
consume-path type-gate test — v12's three prescribed assertions are withdrawn.
**Q9 (D17) now confirmed:** that
POST/PUT setting `auth_type: "oauth"` is rejected; that a PUT re-submitting an
unchanged `auth_type: "oauth"` on an OAuth-completed row still succeeds (transition
gate, not a value gate — a value gate 400s every save from
`mcpservers_detail.js:156`); and that BOTH internal oauth writers still succeed —
`complete.go:136-158` (new row) and the reconnect branch `complete.go:112-133` (`:117`
via `McpServerUpdate` at `:126`).
**New in v6:** that a PUT against a deleted server returns an
error rather than a silent 200 with no `EventTypeUpdated` published (D15 — assert
BOTH the status and the absence of the webhook); that every other
`McpServerUpdate` caller still behaves correctly under the new `RowsAffected`
check, `access_token.go:101-108` included; that a repeat DELETE does not move
`tm_delete` or re-publish `EventTypeDeleted` while still answering 200 (D16); that
`POST`/`PUT` with `auth_type: "oauth"` is rejected (D17, §5 Q9 resolved);
and that a rejected `mcp_server_ids` leaves no AI behind on POST (D18). **New in v7
(D20):** that `PUT {}` (every field omitted) against a deleted server is rejected
rather than returning 200 with the row — the existing
`handler_partial_update_test.go:115-148` no-op test needs a deleted-row counterpart;
that a malformed-URL PUT on a deleted row returns the gate's 404, not 400
`INVALID_MCP_SERVER_URL`; and that `mcpserverhandler.Get` is NOT gated, i.e.
`v1_mcpservers.go:142`'s customer GET still answers 200 while `mcp_tool.go:73`/`:157`
fail closed. **New in v8 (D21), per §5 Q11:** a test that an
existing AI holding a deleted MCP server id can still be saved when editing an
unrelated field (name/prompt/engine) — this is the regression that freezes customer
AIs, and it spans both repos, so PR C must exercise it in a real browser against a
seeded deleted id. Clean Sphinx
rebuild; `grep -n 'mcp_servers' docsdev/source/*.rst` returns nothing **and
`grep -n '9999-01-01' docsdev/source/ai_struct_mcpserver.rst` returns nothing**
(D19). Full 21-test api-validator mcpserver suite green (GET-after-DELETE 200 must
still hold, and the cleanup fixture's repeat-DELETE tolerance must not regress).

**PR C (`monorepo-javascript`):** baseline-vs-branch test comparison per CLAUDE.md
test gate; `npm run build`; production build served and verified in a real browser
(reviews and RTL do not catch unmount races — 대표님 has repeatedly found real bugs
this way after green reviews). Verified no existing test breaks: `ais_create.test.js:157`
is a blanket `ProviderGet.mockResolvedValue`, `:320/:380/:398/:431` use
`expect.objectContaining`, `:446` uses `stringMatching(/rags/)`; `ais_detail.test.js:690-726`
asserts only the PUT body. All four form bodies (§4) must be exercised.

## 8. Retrospective (five rounds)

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

**v5 (this document's first split version)** passed the fact-check reviewer with no
errors found, but the adversarial reviewer found a NEW failure mode: **prescribing a
fix without verifying the fix produces the promised observable.** v5 mandated a
`tm_delete` predicate on `McpServerUpdate` and promised a test asserting rejection —
without checking that `McpServerUpdate` discards `RowsAffected`, which makes the
predicate a silent no-op returning `nil`. v5 had already established that exact
mechanic one section earlier, for credential zeroing, and failed to carry it across.
v5 also under-scoped PR C (its scope line listed one form body while its own §3.12
and §6 named three) and missed four defects in the very file it was editing
(`ai_struct_mcpserver.rst`: the sentinel note, the example block, the `oauth_vendor`
label, the unenforced oauth sentence).

**v6** fixed all of that and was approved on facts a second time, but the adversarial
reviewer found the **same failure mode one layer higher**: v6 prescribed a
`McpServerUpdate` gate plus a `RowsAffected` check, and `PUT {}` never reaches
either, because `mcpserverhandler.Update` short-circuits to `h.Get` before the DB
write (D20). v6 had even quoted the surrounding function twice without noticing the
branch above the line it quoted. It also missed that validation runs before
existence (so a deleted row answers 400), and that the handler-level `Get` is as
ungateable as the dbhandler one, for a different reason.

**v7** was approved on facts a third time, and the adversarial reviewer found the
**third instance of the same shape — this time spanning two repositories.** v7's own
prescriptions (gate `ValidateMcpServerIDs` on `tm_delete`, and don't prune stale ids)
combine with square-admin's unconditional `mcp_server_ids` PUT and its
`deleted=false`-filtered picker to make any affected AI un-saveable forever (D21).
v7 had even documented the picker behavior and called it "a display decision, not a
data problem" — a judgement that was correct for the code as it stands and wrong for
the code v7 was prescribing. Five rounds missed it because the cause and the effect
live in different repositories, and each repo's section was reviewed against its own
code.

**v10** resolved Q5 and Q9 and declared the document ready for the design doc. Round 8
refuted both halves of that. The fact-check track found that v10 had **fabricated a
constraint**: it asserted the PUT path performs no pre-write `AIGet` when
`chatbot.go:134-138` does exactly that, unconditionally, with a comment saying so. The
adversarial track found the **fourth instance** of the recurring failure mode inside
the newly resolved Q5 itself — and worse, Q5's write-time gate is the one layer that
can neither enforce the Insight policy (the consume path has no type check) nor avoid
freezing customers (square-admin clears tools but not MCP servers on the Insight
flip). The author had diagnosed this exact shape three times, written two lessons
about it, self-flagged a possible fourth instance — and then created a fifth in the
very decision meant to close the loop.

Lessons now in effect:
- **When a gate needs five rounds of design, question the policy, not the gate.**
  D13 → D22 → D23 was three escalating attempts to place one AI-type gate; each fix
  created the next defect. The author never stepped back to ask whether the restriction
  itself was wanted. 대표님 answered that in one sentence ("인사이트도 mcp 사용가능하도록
  해") and three defects evaporated. **Repeated implementation difficulty is evidence
  about the requirement, not just the implementation.**
- **Surface the policy question early, with its cost.** The author treated "are Insight
  AIs allowed MCP servers?" as settled by an inferred analogy to `AllowedToolNames`,
  and spent five rounds on mechanics. Asking "do we actually want this restriction?"
  before designing the gate would have saved all of it.
- **A gate is only as reachable as its enclosing condition.** v12 verified that both
  new gate points *hold the AI object*, and concluded the gate was sound. It never
  asked which requests *enter* the block the gate sits in — both call sites are inside
  `if req.McpServerIDs != nil`, so the one request that creates the violation (a type
  flip omitting the field) walks past it (D23). Verifying that a gate CAN run is not
  verifying that it WILL run for the case it targets.
- **Name the request that violates the new rule, then trace whether it reaches the
  gate.** Not "is the gate correct" but "what is the cheapest request that breaks this
  rule, and what happens to it line by line".
- **State the predicate, not the policy.** "Insight AIs are denied MCP servers" is a
  policy; "reject a NON-EMPTY whitelist when the resolved type is Insight" is a
  predicate. The policy left the empty list ambiguous, and the literal reading would
  have 400'd every Insight AI save (D23c). Every prescribed rejection needs its exact
  predicate and its exempt cases written down.
- **A frontend cleanup keyed on an event only covers rows that fire that event.**
  Clearing state in `onValueChange` does nothing for a row that loads already in the
  target state — and hiding the control removes the only affordance to fix it.
- **Verify a test citation is live before using it as evidence.** v11/v12 cited two
  api-validator tests as proof of real traffic; the whole module is skipped at `:4`.
  A skipped test proves nothing and cannot regress.
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
- **When prescribing a fix, verify the fix produces the observable you promise to
  test.** A `WHERE` predicate only rejects if the caller inspects `RowsAffected`; an
  enum in a spec only validates if a validator runs. Trace the fix forward the same
  way you trace a defect.
- **When editing a file for one defect, read it in full and fix every defect in it.**
  A second PR touching the same paragraph is wasted review.
- **When quoting a function to justify a gate, read the WHOLE function from its first
  line, not the neighborhood of the line you care about.** Early returns, guard
  clauses, and short-circuits above your quote can make the gate unreachable (D20).
- **For every gate, name the request that bypasses it.** If you cannot think of one,
  you have not looked hard enough at the branches.
- **When a backend gate starts rejecting data the frontend already holds and
  re-submits, the frontend is part of the change.** Trace every new rejection to the
  client that will hit it, across repository boundaries. "Not our repo" is how a fix
  becomes an outage (D21).
- **A judgement about existing behavior ("this is only cosmetic") expires the moment
  you prescribe a change that touches it.** Re-evaluate every such judgement against
  the post-fix code, not the current code.
- **Never assert that code does NOT do something without grepping for it.** v10 claimed
  the PUT path performs no pre-write read; it performs one, with an explanatory
  comment. A negative claim about code needs the same evidence as a positive one, and
  is easier to get wrong because nothing contradicts it on the screen you are reading.
- **A new policy needs an enforcement point, and the write path is rarely it.**
  Before prescribing a validation gate, ask which layer actually decides the behavior
  at runtime. If the consume path does not check the predicate, a write-time gate is
  documentation, not enforcement — it only constrains rows created after the deploy
  (D22).
- **Every new rejection predicate needs its own frontend trace; an exemption written
  for one predicate does not cover another.** Q11's skip exempts already-stored ids
  from the deleted-server check, and that exemption gave false comfort about the type
  check, which is a different predicate hitting the same unconditional PUT body.
