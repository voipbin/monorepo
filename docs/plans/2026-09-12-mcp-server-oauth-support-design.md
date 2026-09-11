# MCP server OAuth 2.1 support (GitHub + Linear pilot)

Status: DRAFT (Round 0)
Author: CPO design, per 대표님 "1번가자" (vendor-fixed OAuth catalog, GitHub +
Linear pilot, api-manager owns the public callback)
Depends on: docs/plans/2026-09-11-mcp-tool-integration-design.md (bearer/
api_key/none auth, the McpServer entity, SSRF guard, mcptoolhandler),
docs/plans/2026-09-12-mcp-server-catalog-presets-design.md (square-admin
quick-start catalog UX, merged as monorepo-javascript PR #470)

## 1. Problem

The MCP server integration shipped with three auth types: `none`, `bearer`,
`api_key`. Most popular hosted MCP servers (GitHub, Linear, Notion,
Atlassian, Slack, Asana, HubSpot...) require OAuth 2.1 instead -- there is
no static secret a customer can paste. Customers cannot register these
without an OAuth flow VoIPBin drives on their behalf.

## 2. Scope decision (대표님, this session)

- **Vendor-fixed catalog, not arbitrary OAuth.** VoIPBin pre-registers one
  OAuth application per supported vendor (holds `client_id`/`client_secret`
  for that vendor). The customer clicks "Connect with GitHub" / "Connect
  with Linear" and authorizes -- no client_id/secret entry by the customer.
  Rejected: (a) customer-supplied client_id/secret (spreads OAuth app
  registration burden onto every customer, worse UX, was explicitly the
  UX problem this whole catalog effort is solving), (b) Dynamic Client
  Registration only (GitHub and Linear's *interactive* flow uses DCR, but
  a DCR-only implementation would silently fail for any vendor that
  doesn't support DCR -- narrower future coverage for no simplification
  today, since both pilot vendors need real per-vendor app credentials
  regardless per §3).
- **Pilot vendors: GitHub + Linear.** Both allow self-service OAuth app
  creation in their own developer console with no review/approval wait,
  unlike e.g. Notion (public integration listing review) or HubSpot (app
  marketplace review). Fastest to ship, fits VoIPBin's developer/PM-heavy
  target audience.
- **api-manager owns the public OAuth callback**, per the existing
  "api-manager = sole public HTTP gateway" rule. ai-manager (which owns
  the McpServer entity) is never publicly reachable; api-manager calls it
  over the existing internal RPC (`bin-common-handler/pkg/requesthandler`)
  the same way every other MCP server RPC already works.

## 3. Per-vendor OAuth facts (verified against vendor docs this session)

| | GitHub | Linear |
|---|---|---|
| MCP server URL | `https://api.githubcopilot.com/mcp/` (GitHub's official remote MCP server; confirmed via GitHub's own `github-mcp-server` host-integration doc) | `https://mcp.linear.app/mcp` (Linear's official docs, linear.app/docs/mcp) |
| Authorization endpoint | `https://github.com/login/oauth/authorize` | `https://linear.app/oauth/authorize` (Linear Developers OAuth 2.0 doc) |
| Token endpoint | `https://github.com/login/oauth/access_token` | `https://api.linear.app/oauth/token` |
| PKCE | Not required by GitHub's classic OAuth Apps flow, but MCP spec mandates it for the *MCP client* leg regardless -- VoIPBin generates and sends PKCE `code_challenge`/`code_verifier` on every flow, vendor-agnostic (§7) | Required (Linear's docs show PKCE token-request parameters) |
| Refresh token | GitHub OAuth Apps: no refresh token by default (long-lived access token, no expiry) -- design accounts for "may not have a refresh_token" as a normal, not error, case | Yes -- `expires_in: 86399` (~24h), rotating refresh token |
| App registration surface | github.com/settings/developers -> "OAuth Apps" -> "New OAuth App", instant, no review | linear.app -> workspace settings -> API -> OAuth applications, instant, no review |
| Redirect URI registered per app | Single fixed URI (`https://api.voipbin.net/mcpservers/oauth/callback`, see §5) | Same |

Both vendors' MCP servers additionally support providing their own
`Authorization: Bearer <token>` header directly (i.e. a customer could
already paste a manually-obtained PAT via the *existing* `bearer` auth
type as a workaround) -- this design is about removing that manual step,
not about a capability that's otherwise totally unavailable today.

## 4. New auth type: `oauth`

`bin-ai-manager/models/mcpserver/main.go`:

```go
const (
    AuthTypeNone   AuthType = ""
    AuthTypeBearer AuthType = "bearer"
    AuthTypeAPIKey AuthType = "api_key"
    AuthTypeOAuth  AuthType = "oauth"  // NEW
)
```

`AuthTypeOAuth` is added to `validAuthTypes`. Unlike `bearer`/`api_key`,
`auth_type: oauth` is never set directly via `POST /mcpservers` or
`PUT /mcpservers/:id` with a customer-supplied secret -- it is set
*implicitly* by completing the OAuth flow described in §6-§7. A `POST
/mcpservers` or `PUT` request that tries to set `auth_type: oauth`
directly is rejected with 400 (`INVALID_MCP_SERVER_AUTH_TYPE`, extending
the existing validation, not a new error class) -- the only way to reach
`oauth` is via the dedicated OAuth endpoints, so the field cannot be
spoofed into a state with no actual token behind it.

New `McpServer` fields (mirrors the existing `secret_ciphertext`/
`secret_nonce`/`key_version` envelope, reusing `mcpserverhandler.SecretCrypto`
byte-for-byte -- no new crypto primitive):

```go
type McpServer struct {
    // ... existing fields unchanged ...

    // OAuthVendor identifies which entry of the vendor catalog (§6) this
    // row is bound to. Empty for non-oauth auth types. Immutable once set
    // (changing vendor requires disconnect + reconnect, not an update --
    // §9).
    OAuthVendor string `json:"oauth_vendor,omitempty" db:"oauth_vendor"`

    // AccessTokenCiphertext/-Nonce hold the encrypted OAuth access token,
    // same AES-256-GCM envelope and KeyVersion column as the existing
    // secret fields (reuses mcpserverhandler.SecretCrypto, not a new key
    // set -- MCP_SECRET_ENCRYPTION_KEYS covers both).
    AccessTokenCiphertext []byte `json:"-" db:"access_token_ciphertext"`
    AccessTokenNonce      []byte `json:"-" db:"access_token_nonce"`
    AccessTokenExpiresAt  *time.Time `json:"-" db:"access_token_expires_at"` // nil = vendor did not report an expiry (treat as long-lived, e.g. GitHub)

    // RefreshTokenCiphertext/-Nonce are nil when the vendor did not issue
    // a refresh token (GitHub, per §3) -- this is a normal, not an error,
    // state; §8's refresh path treats a nil refresh token as "cannot
    // refresh, fail the call with a clear message" rather than crashing.
    RefreshTokenCiphertext []byte `json:"-" db:"refresh_token_ciphertext"`
    RefreshTokenNonce      []byte `json:"-" db:"refresh_token_nonce"`

    // Reuses the existing KeyVersion column for all four ciphertext
    // fields above (same encryption key set, same rotation story --
    // no new column).
}
```

`HasSecret bool` is renamed in *meaning* only (not wire-broken) to also
cover oauth: `HasSecret` becomes true whenever `AccessTokenCiphertext` is
non-empty, exactly parallel to today's `len(SecretCiphertext) > 0` check
in `dbhandler/mcpserver.go`. `WebhookMessage` additionally exposes
`OAuthVendor` (never the tokens) so square-admin can render "Connected to
GitHub" / "Connected to Linear" without a secondary GET.

## 5. New DB columns (Alembic migration)

Added to `ai_mcp_servers` (all nullable, no backfill needed -- every
existing row is `auth_type IN ('', 'bearer', 'api_key')` and none of the
new columns apply):

```sql
ALTER TABLE ai_mcp_servers
  ADD COLUMN oauth_vendor              VARCHAR(64)  NULL,
  ADD COLUMN access_token_ciphertext   VARBINARY(512) NULL,
  ADD COLUMN access_token_nonce        VARBINARY(32)  NULL,
  ADD COLUMN access_token_expires_at   DATETIME(6)    NULL,
  ADD COLUMN refresh_token_ciphertext  VARBINARY(512) NULL,
  ADD COLUMN refresh_token_nonce       VARBINARY(32)  NULL;
```

A separate new table, `ai_mcp_oauth_states`, holds short-lived CSRF/PKCE
state for in-flight authorization requests (§7 step 2) -- NOT reusing
`ai_mcp_servers` for this, because a flow is initiated before the target
row necessarily exists (customer may be doing "Add GitHub MCP server"
from scratch, not editing an existing row) and because leaking this into
the long-lived entity table would need extra nullable churn on every row
for data that lives seconds:

```sql
CREATE TABLE ai_mcp_oauth_states (
  state            VARCHAR(64) NOT NULL PRIMARY KEY,   -- opaque random token, in the OAuth `state` param
  customer_id      BINARY(16)  NOT NULL,
  mcp_server_id    BINARY(16)  NULL,                   -- NULL = "create a new server on callback"; set = "attach to this existing server" (reconnect, §9)
  vendor           VARCHAR(64) NOT NULL,
  pkce_verifier    VARCHAR(128) NOT NULL,               -- PKCE code_verifier, plaintext (short-lived, single-use, never logged)
  tm_create        DATETIME(6) NOT NULL,
  tm_expire        DATETIME(6) NOT NULL                 -- tm_create + 10 minutes (§7 step 2)
);
```

No `customer_id`/`mcp_server_id` foreign keys enforced at the DB layer
(matches this monorepo's existing convention of app-layer referential
integrity for `ai_*` tables). A row is deleted the moment it's consumed
(§7 step 5) or, failing that, by a periodic sweep of `tm_expire < now()`
(reuses the existing generic expired-row cleanup pattern already present
for other short-lived state in this monorepo -- see
`references/existing-service-reconnaissance.md` convention notes; exact
cron wiring is an implementation-time detail, not a design decision).

## 6. Vendor catalog (backend, not the same as square-admin's UX catalog)

New package `bin-ai-manager/pkg/mcpoauthhandler` owns a small, hand-coded
Go map (not DB-backed -- same "static list, PR to change" posture as
square-admin's `mcpServerCatalog.js`, deliberately not overengineered
into an admin-editable table for 2 initial entries):

```go
type vendorConfig struct {
    MCPServerURL     string   // e.g. "https://api.githubcopilot.com/mcp/"
    AuthorizeURL     string
    TokenURL         string
    Scopes           []string
    ClientID         string   // from env, see below
    ClientSecret     string   // from env, see below
}

var vendors = map[string]vendorConfig{
    "github": {
        MCPServerURL: "https://api.githubcopilot.com/mcp/",
        AuthorizeURL: "https://github.com/login/oauth/authorize",
        TokenURL:     "https://github.com/login/oauth/access_token",
        Scopes:       []string{"repo", "read:org"}, // narrowest scope covering the GitHub MCP server's default toolset; revisit if customers hit permission errors
    },
    "linear": {
        MCPServerURL: "https://mcp.linear.app/mcp",
        AuthorizeURL: "https://linear.app/oauth/authorize",
        TokenURL:     "https://api.linear.app/oauth/token",
        Scopes:       []string{"read", "write"},
    },
}
```

`ClientID`/`ClientSecret` are populated from new config flags
`mcp_oauth_github_client_id` / `mcp_oauth_github_client_secret` /
`mcp_oauth_linear_client_id` / `mcp_oauth_linear_client_secret` (env
`MCP_OAUTH_GITHUB_CLIENT_ID` etc., following the exact naming convention
already used for `MCP_SECRET_ENCRYPTION_KEYS` in
`bin-ai-manager/internal/config/main.go`). These are real VoIPBin-owned
app credentials (one GitHub OAuth App, one Linear OAuth application),
stored the same way every other cross-cutting secret in this monorepo is
(SOPS-encrypted k8s/komodo secret, per
`voipbin-k8s-secret-management` conventions) -- NOT per-customer, NOT in
the database.

## 7. OAuth flow

```
Customer (square-admin)      api-manager (public)      ai-manager      Vendor AS
        |                          |                        |             |
1. POST /mcpservers/oauth/start    |                        |             |
   { vendor: "github" }            |                        |             |
        |------------------------->|                        |             |
        |                          | 2. generate state+PKCE,|             |
        |                          |    RPC AIV1McpOAuthStart|            |
        |                          |----------------------->|             |
        |                          |                        | insert      |
        |                          |                        | ai_mcp_     |
        |                          |                        | oauth_states|
        |                          |<-----------------------|             |
        |  { authorize_url }       |                        |             |
        |<-------------------------|                        |             |
2b. Browser redirects to authorize_url (GitHub/Linear consent screen)     |
        |------------------------------------------------------------->  |
                                                       3. Customer authorizes
        |  <-- redirect: GET /mcpservers/oauth/callback?code=...&state=...|
        |<-------------------------------------------------------------  |
        |                          |                        |             |
        |                     4. GET /mcpservers/oauth/callback           |
        |                          | (public, unauthenticated -- the      |
        |                          |  `state` param IS the auth, per §7a) |
        |                          | 5. RPC AIV1McpOAuthCallback           |
        |                          |----------------------->|             |
        |                          |                        | 6. validate |
        |                          |                        |    state,   |
        |                          |                        |    POST     |
        |                          |                        |    token    |
        |                          |                        |    endpoint |
        |                          |                        |------------>|
        |                          |                        | 7. access + |
        |                          |                        |    refresh  |
        |                          |                        |    token    |
        |                          |                        |<------------|
        |                          |                        | 8. encrypt, |
        |                          |                        |    upsert   |
        |                          |                        |    McpServer|
        |                          |                        |    (create  |
        |                          |                        |    if new,  |
        |                          |                        |    update if|
        |                          |                        |    reconnect)|
        |                          |<-----------------------|             |
        |  9. 302 redirect to      |                        |             |
        |     admin.voipbin.net/#  |                        |             |
        |     /resources/mcpservers/                        |             |
        |     <id>?oauth=success   |                        |             |
        |<-------------------------|                        |             |
```

### 7a. Why the callback is public but still safe

`GET /mcpservers/oauth/callback` cannot carry a JWT (the browser redirect
comes straight from github.com/linear.app, no way to attach VoIPBin
auth headers). Its security is the OAuth `state` parameter, which is:

- generated server-side in step 2 (crypto-random, 32 bytes, base64url --
  same RNG already used for `mcpserverhandler`'s secret nonces)
- bound to `customer_id` in `ai_mcp_oauth_states` at generation time (not
  trusted from the callback request itself)
- single-use: the callback handler deletes the row in the same
  transaction it reads it (step 6), so a replayed callback request
  (browser back-button, retried redirect) fails with "state not found or
  already used" rather than silently reusing a stale authorization code
- time-bounded: `tm_expire` (10 minutes, matching typical OAuth
  authorization-code lifetimes) rejects stale flows
- this is the exact same trust model `POST /auth/password-reset` already
  uses in this monorepo (a public endpoint whose security is a
  single-use, time-bounded, server-generated token in the URL, not a
  JWT) -- no new authentication pattern, reusing an established one

### 7b. PKCE

VoIPBin generates `code_verifier` (43-128 char random string) and
`code_challenge` (`base64url(sha256(code_verifier))`) at step 2 for
*every* vendor, regardless of whether that vendor's docs say PKCE is
required (§3 table) -- sending an unused `code_challenge` is harmless
per the OAuth 2.1 spec and keeps `mcpoauthhandler`'s flow code
vendor-uniform rather than branching per-vendor on this one flag.

## 8. Outbound MCP calls: token refresh

`bin-ai-manager/pkg/mcptoolhandler/client.go`'s `buildAuthHeader` gets a
new case:

```go
case mcpserver.AuthTypeOAuth:
    token, err := h.oauthHandler.GetValidAccessToken(ctx, m)
    if err != nil {
        return "", "", fmt.Errorf("could not get valid oauth access token: %w", err)
    }
    return "Authorization", "Bearer " + token, nil
```

`GetValidAccessToken` (new method on `mcpoauthhandler`):
1. Decrypt `AccessTokenCiphertext` + check `AccessTokenExpiresAt`.
2. If `AccessTokenExpiresAt` is nil (vendor never reported an expiry,
   e.g. GitHub) OR still in the future with a 60-second safety margin,
   return the decrypted token as-is -- no refresh call.
3. Otherwise, if `RefreshTokenCiphertext` is non-empty (Linear), POST to
   the vendor's token endpoint with `grant_type=refresh_token`, decrypt +
   re-encrypt the new access/refresh token pair (per OAuth 2.1's
   single-use refresh token rotation -- both GitHub and Linear rotate),
   `McpServerUpdate` the row, return the new access token.
4. Otherwise (expired, no refresh token available) return an error --
   surfaced to the LLM tool-call path the same way any other MCP call
   failure already is (`mcptoolhandler.CallTool`'s existing error
   propagation, no new error class). The customer sees a normal tool-call
   failure and, on the square-admin MCP server detail page, `HasSecret:
   true` but an `access_token_expires_at` in the past -- §10 surfaces a
   "Reconnect" action for exactly this state.

This refresh happens synchronously inline on the calling goroutine (the
same one already calling `doJSONRPCRequest`) -- no separate background
refresh job. Matches this handler's existing per-call, no-cache posture
for McpServer rows (design doc §16: "No cache layer for McpServer...
bounded by conversation turn count"), and keeps the change local to one
function rather than introducing a new scheduled job for 2 vendors.

## 9. Reconnect / disconnect

- **Disconnect**: `DELETE /mcpservers/:id` already exists and works
  unchanged (soft-delete, same as any other server) -- no new endpoint.
- **Reconnect** (e.g. the customer revoked access on GitHub's side, or
  the refresh token was itself revoked): `POST /mcpservers/oauth/start`
  accepts an optional `mcp_server_id` field. When present, step 2's state
  row carries that ID (§5's `mcp_server_id` column), and step 8's
  callback does `McpServerUpdate` on the existing row instead of
  `McpServerCreate` -- same name/URL/vendor, fresh tokens. This is the
  only way `auth_type: oauth` rows get their tokens replaced; there is no
  "PUT new access token" path (consistent with §4's "only the OAuth
  endpoints can produce an oauth-authenticated row").
- Changing `oauth_vendor` on an existing row is not supported (§4) --
  the customer deletes and re-adds instead. Two vendors, low value in
  supporting an in-place vendor swap that would need to re-derive
  name/URL from the new vendor's catalog entry anyway.

## 10. api-manager: new endpoints

`bin-api-manager/server/mcpservers_oauth.go` (new file, same package/
conventions as `mcpservers.go`):

```
POST /mcpservers/oauth/start      (authenticated, v1.0 group)
  body: { vendor: "github"|"linear", mcp_server_id?: uuid }
  -> { authorize_url: string }

GET  /mcpservers/oauth/callback   (PUBLIC, registered next to
                                    /auth/password-reset in main.go --
                                    same public-with-token-security
                                    pattern, §7a)
  query: ?code=...&state=...  (or ?error=... on vendor-side denial)
  -> 302 redirect to admin.voipbin.net with a query param indicating
     success/failure (no response body a browser would render; this is
     purely a redirect target)
```

Both are added to `bin-openapi-manager/openapi/paths/mcpservers/` (new
`oauth_start.yaml`, `oauth_callback.yaml`) and regenerate both
`bin-api-manager/gens/openapi_server/gen.go` AND
`bin-openapi-manager/gens/models/gen.go` together (the exact drift this
monorepo's CI caught during the PUT partial-update PR #1291 -- noted
here explicitly so implementation doesn't repeat that CI failure).

`POST /mcpservers/oauth/start` RPCs `AIV1McpOAuthStart` (new method on
`bin-common-handler/pkg/requesthandler`, following the existing
`AIV1McpServerUpdate` etc. naming/signature convention). The callback
RPCs `AIV1McpOAuthCallback`. Both are internal-only RPCs -- api-manager
is still the only public HTTP surface; nothing here creates a second
public entrypoint into ai-manager.

## 11. square-admin: "Connect with GitHub/Linear" UX

Extends the existing quick-start catalog (PR #470) rather than
replacing it. `mcpServerCatalog.js` gains an `authFlow` field
distinguishing `preset` (today's 4 bearer/none entries -- pre-fills the
form) from `oauth` (new):

```js
{
  id: 'github',
  name: 'GitHub',
  description: "Issues, pull requests, code search, and more.",
  authFlow: 'oauth',
  vendor: 'github',
  icon: githubIcon,
},
{
  id: 'linear',
  name: 'Linear',
  description: 'Find, create, and update issues, projects, and comments.',
  authFlow: 'oauth',
  vendor: 'linear',
  icon: linearIcon,
},
```

Clicking an `authFlow: 'oauth'` card does NOT pre-fill the manual form
(there is nothing to pre-fill -- no URL/secret the customer types) --
instead it immediately calls `POST /mcpservers/oauth/start` and redirects
the browser to the returned `authorize_url`. On return
(`?oauth=success&id=<uuid>` per §10), square-admin navigates straight to
that server's detail page. On `?oauth=error=...`, an `ActionFeedback`
error banner surfaces the vendor's denial reason.

The MCP server detail page (`mcpservers_detail.js`) gains an
`oauth_vendor`-aware view: when `auth_type === 'oauth'`, it shows
"Connected to GitHub" (or Linear) with a "Reconnect" button (calls
`POST /mcpservers/oauth/start` with this server's ID, §9) instead of the
existing secret/api_key_header edit fields, which do not apply to oauth
rows.

GitHub and Linear brand icons follow the exact same self-hosting
precedent as PR #470's Context7/Firecrawl/Apify/DeepWiki icons --
downloaded from each vendor's own official brand assets
(github.com has a public brand-assets page; Linear's press/brand page),
stored in `src/assets/mcp-icons/`, exact source URLs recorded in
`mcpServerCatalog.js`'s header comment (matches the convention
established during that PR's review loop).

## 12. Security notes

- Access/refresh tokens use the exact same AES-256-GCM envelope +
  `MCP_SECRET_ENCRYPTION_KEYS` rotation story as the existing `secret`
  field -- no new key management surface, no new crypto code path beyond
  extending `SecretCrypto`'s call sites.
- `ai_mcp_oauth_states.pkce_verifier` is stored in plaintext (not
  encrypted) because it is single-use and lives at most 10 minutes; this
  matches how short-lived nonces are already handled elsewhere in this
  monorepo (e.g. `bin-customer-manager`'s email-verify tokens) and adding
  envelope encryption to a 10-minute-lived, single-use value is not a
  meaningful security improvement, just extra code.
- The OAuth `state` parameter is never logged (existing `truncateForError`
  precedent in `mcptoolhandler/client.go` already caps/redacts response
  bodies; the callback handler must apply the same discipline to the
  `code`/`state` query params in its own log lines -- an implementation
  checklist item, not a design change).
- `/mcpservers/oauth/callback` is added to `runListenHTTP`'s access-log
  `SkipPaths` in `bin-api-manager/cmd/api-manager/main.go` (same
  treatment as `/provisioning/extension`, §2's existing precedent) --
  the `code` query param is a genuine one-time secret and must not reach
  stdout/access logs.
- Vendor app `client_secret`s (GitHub/Linear) are VoIPBin-owned
  cross-cutting secrets, not per-customer data -- provisioned via SOPS +
  komodo/k8s secret per `voipbin-k8s-secret-management` convention, never
  stored in `ai_mcp_servers` or any customer-scoped table.

## 13. Non-goals

- Dynamic Client Registration (RFC 7591) -- not needed since both pilot
  vendors get real pre-registered app credentials (§2).
- Arbitrary/customer-supplied OAuth MCP servers -- explicitly rejected in
  §2; would be a separate, larger design if ever pursued.
- Vendors beyond GitHub/Linear -- adding one is "add an entry to the
  `vendors` map (§6) + register a VoIPBin OAuth app with that vendor +
  add a square-admin catalog entry (§11)", a small, repeatable follow-up
  once this pilot is proven, not part of this design's implementation.
- Scopes/permission picker UI -- the scope list per vendor (§6) is fixed
  in code for the pilot; a customer-facing "choose which scopes to grant"
  UI is out of scope (the vendor's own consent screen already lets the
  user see/deny the requested scopes).
- Token-exchange / on-behalf-of flows, resource indicators (RFC 8707)
  beyond what's needed to satisfy each vendor's own token endpoint
  requirements -- not needed for a 2-vendor pilot with fixed, known
  audiences.
- Changing `bin-registrar-manager`'s unrelated pre-existing `astauth`
  OAuth code (Asterisk realtime auth, a completely different "oauth" by
  name coincidence only, confirmed via repo grep during this design's
  reconnaissance) -- not touched, not related.

## 14. Open questions for review

- Exact GitHub OAuth scope set (`repo`, `read:org` proposed in §6) should
  be validated against the actual tool list GitHub's MCP server exposes
  once implementation starts (may need `read:user` too, TBD at
  implementation time, not a design blocker).
- Whether `ai_mcp_oauth_states` needs a dedicated cleanup cron or can
  piggyback an existing one -- flagged in §5 as an implementation-time
  detail; if this monorepo has no existing generic "sweep expired rows"
  cron, implementation should confirm the mechanism during, not defer
  indefinitely.
