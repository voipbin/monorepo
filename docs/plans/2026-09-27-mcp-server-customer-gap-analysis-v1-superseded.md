# Analysis: MCP Server customer-facing gap (docs + square-admin)

Date: 2026-09-27
Author: CPO (Hermes)
Status: Analysis, pending Analysis Review Loop (min 2, 2x consecutive APPROVED)

## 1. Issue statement

The customer-configured MCP tool integration is fully implemented and deployed
in the backend (`bin-ai-manager`), exposed via `bin-api-manager` REST, and has a
square-admin CRUD surface for MCP server registration. However two customer
reachability gaps remain:

- **G1 (docs):** `docs.voipbin.net` has NO user-facing documentation for
  registering an MCP server or whitelisting it on an AI. The only MCP mention in
  the whole RST tree is a single external link.
- **G2 (square-admin):** `mcp_server_ids` can only be set on the AI DETAIL page.
  The AI CREATE page and the teamgraph AI create/edit panels have no MCP field,
  even though the backend accepts `mcp_server_ids` on `POST /ais`.

## 2. Validity re-check (code-verified, not from memory)

### 2.1 Backend is complete and accepts the field on BOTH create and update

| Claim | Evidence |
|---|---|
| Domain model carries the whitelist | `bin-ai-manager/models/ai/main.go:95` `McpServerIDs []uuid.UUID` (`db:"mcp_server_ids,json"`) |
| Exposed externally | `bin-ai-manager/models/ai/webhook.go:47` |
| Field constant for updates | `bin-ai-manager/models/ai/field.go:37` `FieldMcpServerIDs` |
| Ownership validation (anti-IDOR) | `bin-ai-manager/pkg/aihandler/mcpserver_validation.go:35-60` — per-id `McpServerGet` + `srv.CustomerID != customerID` → `cerrors.InvalidArgument` 400; genuine DB errors deliberately fall through to 500 |
| Dedicated write path | same file, `UpdateMcpServerIDs` (lines 72-89), publishes `ai.EventTypeUpdated` |
| Wired on POST **and** PUT | `bin-ai-manager/pkg/listenhandler/v1_ais.go:119,125` (create path) and `:274,280` (update path) |
| Explicit-clear vs absent distinguishable | `bin-ai-manager/pkg/listenhandler/models/request/ais.go:42,81` `McpServerIDs *[]uuid.UUID` |
| Regression tests exist | `v1_ais_test.go:534-557` (`{"mcp_server_ids":[]}` explicit clear), `:585` (omitted → no call), `:647` (invalid id → 400) |

**Critical correction to an earlier assumption:** the OpenAPI spec already
declares `mcp_server_ids` on the CREATE body, not just update:
`bin-openapi-manager/gens/models/gen.go` `PostAisJSONBody.McpServerIds
*[]string` (line ~10606) AND `PutAisIdJSONBody.McpServerIds` (line ~10664).
The api-manager server handler converts it on both routes:
`bin-api-manager/server/ais.go:75-77` (POST) and `:294-296` (PUT).

=> **G2 is a pure frontend gap. No backend or OpenAPI change is required.**

### 2.2 Runtime behavior (for accurate docs)

Source: `bin-ai-manager/pkg/aicallhandler/mcp_tool.go`.

- Namespacing: `mcp_<8-hex-server-id-prefix>_<tool_name>`
  (`mcpToolNamePrefix = "mcp_"` line 21; `mcpServerIDShort` lines 114-120 strips
  dashes, takes first 8 chars).
- Merge point: `resolveTools` (line 52) appends MCP tools AFTER built-ins.
- Reverse mapping persisted under `aicall.MetaKeyMcpToolMap = "mcp_tool_map"`
  (`models/aicall/main.go:67`).
- **Best-effort per server (important for docs):** a server that fails `Get`
  (line 74-77), is not `StatusActive` (line 79-82), or fails `ListTools`
  (line 84-88) is SKIPPED with a log line. It does not fail the AI session and
  does not break other servers' tools.
- Dispatch fails closed: `toolHandleMcpCall` refuses a call when the name is
  unresolvable, the server is no longer whitelisted, or is no longer active
  (lines 122-176).
- Config: `mcp_tools_list_cache_ttl_seconds` default 60,
  `mcp_tool_call_timeout_seconds` default 10
  (`bin-ai-manager/internal/config/main.go:149-150`).

### 2.3 External API surface (for the docs' endpoint table)

`bin-openapi-manager/openapi/openapi.yaml:8569-8578` registers 5 paths:
`GET/POST /mcpservers`, `GET/PUT/DELETE /mcpservers/{id}`,
`POST /mcpservers/oauth/start`, `GET /mcpservers/oauth/callback`,
`POST /mcpservers/oauth/complete`.

Wire fields (authoritative source = `bin-ai-manager/models/mcpserver/webhook.go`,
NOT the internal struct): `id`, `customer_id` (from embedded
`commonidentity.Identity`), `name`, `detail`, `url`, `status`, `auth_type`,
`api_key_header`, `oauth_vendor`, `has_secret`, `tm_create`, `tm_update`,
`tm_delete`.

Fields deliberately NOT exposed and therefore MUST NOT be documented:
`SecretCiphertext`, `SecretNonce`, `KeyVersion`.

`auth_type` intentionally omits `,omitempty` because `AuthTypeNone` is the empty
string and is itself meaningful — so `auth_type` is ALWAYS present on the wire.

Enums (`models/mcpserver/main.go:14-44`):
- `auth_type`: `""` (none) | `bearer` | `api_key` | `oauth`
- `status`: `active` | `disabled`

`oauth` is never settable directly via POST/PUT with a customer-supplied
secret; it is set implicitly by completing `POST /mcpservers/oauth/complete`.
OAuth vendors wired today: GitHub and Linear only
(`internal/config/main.go:151-154`).

### 2.4 Docs gap confirmed

- `grep -rn 'mcp' bin-api-manager/docsdev/source/*.rst` → exactly ONE hit:
  `ai_overview.rst:685`, an external link to `github.com/nrjchnd/voipbin-mcp`
  under an "External AI Agent Integration → MCP Server" heading. That heading is
  about third-party AI consuming VoIPBin media streams, which is a DIFFERENT
  feature from customer-registered MCP tool servers. Leaving it as-is next to new
  content would be actively confusing.
- No `mcpserver*.rst` files exist. `index.rst` has no MCP entry.
- Root `CLAUDE.md` mandates RST sync for "new resource types, statuses, or
  fields" — this is an unmet mandatory requirement, not a nice-to-have.

### 2.5 square-admin gap confirmed

`grep -rn 'mcp_server_ids|mcpServerIds' square-admin/src` → hits ONLY in
`views/ais/ais_detail.js` (state at :103, hydrate at :242, list fetch at :374,
toggle at :382, PUT body at :421, dirty-check at :551-553, UI at :1238-1276)
plus its test file. Zero hits in:

- `views/ais/ais_create.js` (438 lines; builds `body` at :114, POSTs at :154)
- `views/ais/AIEngineFields.js` (659 lines; the SHARED component used by both
  detail and create)
- `views/teamgraph/sidebar.js` (1769 lines; AI create `body` at :514-534, AI
  edit `body` at :753-782)
- `src/types/api.ts` (no MCP type at all)

square-admin's own `CLAUDE.md` "Field Sync Points" table names exactly three
places that must be updated for any AI field: `ais_detail.js`, `ais_create.js`,
`teamgraph/sidebar.js` (both panels). MCP satisfies only one of three.

Consequence for the user: an AI must be created first, then re-opened in detail
to attach MCP servers. A 2-step workflow with no in-product hint that step 2
exists.

Secondary gap: `views/help/content/navHelpManifest.js:39` registers the MCP
list page with `helpTopicId: null` and reason "New feature (MCP tool
integration); dedicated help content not yet written."

## 3. Proceed-or-not analysis

### Recommendation: PROCEED on both, as TWO PRs (one per repository).

Per CLAUDE.md, one-PR-per-task is the default and splitting within a repo needs
permission; work spanning separate repositories is a structural necessity, not a
split. G1 lives in `voipbin/monorepo`, G2 in `voipbin/monorepo-javascript`.

### Priority and rationale

1. **G1 (docs) first.** Highest value/cost ratio. The feature is invisible to
   customers today; no amount of UI polish helps a customer who does not know the
   capability exists. It is also a standing CLAUDE.md compliance gap.
2. **G2 (square-admin) second.** Real UX friction and a self-imposed convention
   violation, but the capability IS reachable today via the detail page. Lower
   severity than total invisibility.

### Why this is not overengineering

Both items are completion/verification work on an ALREADY-SHIPPED feature, which
the standing anti-overengineering rule explicitly exempts. Neither introduces new
infrastructure, new measurement systems, nor speculative capacity.

### Explicitly out of scope (deliberate non-goals)

- **Tool-level (not server-level) whitelisting.** Today a whitelisted server
  contributes ALL its tools. Narrowing to individual tools is a real future
  question but has no measured demand signal. Do NOT build it now.
- **Additional OAuth vendors** beyond GitHub/Linear.
- **help content** for the MCP pages (`navHelpManifest.js`). Lowest priority,
  separate concern; leave the explicit `helpTopicId: null` marker in place.
- Any backend / OpenAPI change. Verified unnecessary (§2.1).

### Risks

| Risk | Mitigation |
|---|---|
| Docs overstate reliability (a customer assumes a down MCP server fails the call) | Document the best-effort skip semantics from §2.2 explicitly |
| Docs leak secret fields | Document strictly against `mcpserver/webhook.go`, never the internal struct (§2.3) |
| square-admin change lands in the DEAD nav/layout generation | No nav change needed (the `/resources/mcpservers/*` routes already exist and are live); the change is inside existing AI forms only |
| Field Sync table drifts again | Update all three sync points in one PR, add tests per point |
| Teamgraph panels are inline (not shared component) | Accept duplication to match the existing file's established pattern rather than refactoring mid-task |

## 4. Open questions for the reviewer

1. Should the docs be a new top-level `mcpserver` section under the "AI &
   Automation" caption, or sub-pages folded into the existing `ai` section? (CPO
   lean: new top-level entry — it is its own REST resource with 5 endpoints, and
   `rag` is the established precedent for exactly this shape.)
2. Should the pre-existing `ai_overview.rst:681-685` "MCP Server" heading (about
   external AI via media stream) be renamed to remove ambiguity with the new
   content? (CPO lean: yes, rename to something media-stream-specific and
   cross-reference the new section.)
