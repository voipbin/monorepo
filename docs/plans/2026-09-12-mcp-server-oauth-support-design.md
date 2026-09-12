# MCP server OAuth 2.1 support (GitHub + Linear pilot)

Status: APPROVED (5-round design review, 2 consecutive APPROVE at R4/R5) -- 대표님 승인 2026-09-12. Implementation continues in this same PR/branch per 대표님's standing instruction (design and implementation are no longer split into separate PRs).
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
  Registration only (Linear's *interactive* setup flow, per its own docs,
  uses DCR; GitHub does not support DCR for its classic OAuth Apps flow
  -- confirmed via GitHub's public MCP-server host-integration
  discussion. Since GitHub alone rules out a DCR-only approach for this
  pilot, and Linear also works fine with a pre-registered app, a
  DCR-only implementation would additionally underserve any future
  vendor that doesn't support DCR -- narrower future coverage for no
  simplification today, since both pilot vendors need real per-vendor
  app credentials regardless per §3).
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
new columns apply). Types match the EXACT existing `secret_ciphertext`/
`secret_nonce` column types in this table
(`bin-dbscheme-manager/bin-manager/main/versions/9b0ad37e0360_ai_mcp_servers_create_table.py`)
rather than introducing a different convention: `blob` for ciphertext
(unbounded, matches `secret_ciphertext blob`), `binary(12)` for the nonce
(AES-GCM's nonce is always exactly 12 bytes -- both existing and new
nonce columns are fixed-size, not `VARBINARY`):

```sql
ALTER TABLE ai_mcp_servers
  ADD COLUMN oauth_vendor              VARCHAR(64)   NULL,
  ADD COLUMN access_token_ciphertext   BLOB          NULL,
  ADD COLUMN access_token_nonce        BINARY(12)    NULL,
  ADD COLUMN access_token_expires_at   DATETIME(6)   NULL,
  ADD COLUMN refresh_token_ciphertext  BLOB          NULL,
  ADD COLUMN refresh_token_nonce       BINARY(12)    NULL;
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
(§7 step 9) or, failing that, by a periodic sweep of `tm_expire < now()`
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

**Revised in Round 2** (see §7a's Layer 2 correction): the public
callback is now a thin, no-side-effect redirect relay; the actual token
exchange moved to an authenticated endpoint (`/oauth/complete`) called
by square-admin after the browser returns.

```
Customer (square-admin)      api-manager                ai-manager      Vendor AS
        |                          |                        |             |
1. POST /mcpservers/oauth/start (authenticated, JWT)         |             |
   { vendor: "github" }            |                        |             |
        |------------------------->|                        |             |
        |                          | 2. generate state+PKCE,|             |
        |                          |    RPC AIV1McpOAuthStart|            |
        |                          |----------------------->|             |
        |                          |                        | insert      |
        |                          |                        | ai_mcp_     |
        |                          |                        | oauth_states|
        |                          |                        | (customer_id|
        |                          |                        |  from JWT)  |
        |                          |<-----------------------|             |
        |  { authorize_url,        |                        |             |
        |    link_token }          |                        |             |
        |<-------------------------|                        |             |
1b. square-admin stores link_token in sessionStorage,                     |
    then does a full-page redirect to authorize_url                      |
        |------------------------------------------------------------->  |
                                                       2b. Customer authorizes
        |  <-- redirect: GET /mcpservers/oauth/callback?code=...&state=...|
        |<-------------------------------------------------------------  |
        |                          |                        |             |
        |                    3. GET /mcpservers/oauth/callback (PUBLIC)   |
        |                       thin relay -- validates `state` EXISTS   |
        |                       in ai_mcp_oauth_states (no customer_id   |
        |                       check here, no token exchange, no DB    |
        |                       write beyond confirming existence)      |
        |                          |                        |             |
        |  4. 302 redirect to admin.voipbin.net/#/resources/mcpservers/  |
        |     oauth-return?state=...&code=...                            |
        |<-------------------------|                        |             |
        |                          |                        |             |
5. oauth-return page: read state from URL, compare against              |
   sessionStorage's link_token (client-side fast-fail, §7a)             |
        |                          |                        |             |
        | 6. POST /mcpservers/oauth/complete (authenticated, JWT)         |
        |    { state, code }       |                        |             |
        |------------------------->|                        |             |
        |                          | 7. RPC AIV1McpOAuthComplete           |
        |                          |----------------------->|             |
        |                          |                        | 8. lookup   |
        |                          |                        |    state,   |
        |                          |                        |    verify   |
        |                          |                        |    row.     |
        |                          |                        |    Customer |
        |                          |                        |    ID ==    |
        |                          |                        |    JWT's    |
        |                          |                        |    customer |
        |                          |                        |    _id      |
        |                          |                        |    (§7a's   |
        |                          |                        |    REAL     |
        |                          |                        |    security |
        |                          |                        |    boundary)|
        |                          |                        | 9. delete   |
        |                          |                        |    state row|
        |                          |                        |    (single- |
        |                          |                        |    use)     |
        |                          |                        | 10. POST    |
        |                          |                        |    token    |
        |                          |                        |    endpoint |
        |                          |                        |------------>|
        |                          |                        | 11. access +|
        |                          |                        |    refresh  |
        |                          |                        |    token    |
        |                          |                        |<------------|
        |                          |                        | 12. encrypt,|
        |                          |                        |    upsert   |
        |                          |                        |    McpServer|
        |                          |                        |    (create  |
        |                          |                        |    if new,  |
        |                          |                        |    update if|
        |                          |                        |    reconnect)|
        |                          |<-----------------------|             |
        |  { mcp_server: {...} }   |                        |             |
        |<-------------------------|                        |             |
13. square-admin navigates to the returned server's detail page          |
```

### 7a. Why the callback is public but still safe

`GET /mcpservers/oauth/callback` cannot carry a JWT (the browser redirect
comes straight from github.com/linear.app, no way to attach VoIPBin
auth headers). Its security has TWO independent layers -- (1) defends
against callback forgery/replay, (2) defends against a session-fixation
style "account-linking hijack" that layer (1) alone does not cover
(Round 1 design review finding, RFC 6749 §10.12's exact warning class):

**Layer 1 -- the `state` parameter** (defends against forgery/replay):

- generated server-side in step 2 (crypto-random, 32 bytes, base64url --
  same RNG already used for `mcpserverhandler`'s secret nonces)
- bound to `customer_id` in `ai_mcp_oauth_states` at generation time (not
  trusted from the callback request itself)
- single-use: the `/oauth/complete` handler deletes the row immediately
  after the ownership check succeeds (diagram step 9), BEFORE the token
  exchange -- never the public callback, which per §10 is a strictly
  read-only existence check with no mutation. A replayed callback
  request (browser back-button, retried redirect) simply re-confirms
  existence and re-redirects; it is the SECOND `/oauth/complete` call
  for the same `state` (whether from a genuine retry or a replay
  attempt) that fails with "state not found or already used", because
  the first successful call already deleted the row.
- time-bounded: `tm_expire` (10 minutes, matching typical OAuth
  authorization-code lifetimes) rejects stale flows
- this is the exact same trust model `POST /auth/password-reset` already
  uses in this monorepo (a public endpoint whose security is a
  single-use, time-bounded, server-generated token in the URL, not a
  JWT) -- no new authentication pattern, reusing an established one

**Layer 2 -- a browser-bound linking token, NOT a cookie** (defends
against account-linking hijack): `state` alone binds the authorization
to a `customer_id`, but says nothing about WHICH BROWSER is completing
the flow. Without this second layer, an attacker could call `POST
/mcpservers/oauth/start` under their OWN VoIPBin session (getting back an
`authorize_url` whose `state` is bound to the attacker's `customer_id`),
then trick a victim into opening that URL and approving vendor consent
with the VICTIM's GitHub/Linear account -- the callback would then
silently attach the victim's vendor account to the attacker's VoIPBin
McpServer row. This is the exact "OAuth login CSRF" pattern RFC 6749
§10.12 warns about.

**Round 2 design review finding, corrected here:** the first revision of
this design proposed a `Set-Cookie` on the `/mcpservers/oauth/start`
response. That does not work with this API's actual deployment: `square-
admin` (`admin.voipbin.net`) calls `api-manager` (`api.voipbin.net`) as a
genuine cross-origin fetch (confirmed: `square-admin`'s API client uses
`https://api.voipbin.net` as its base URL, a different origin from where
the app is served), and `cmd/api-manager/main.go`'s global CORS config
is `AllowCredentials: false` with `AllowOrigins: ["*"]`. A cross-origin
fetch cannot both set/send cookies AND use a wildcard origin -- the
browser silently drops the `Set-Cookie` (or the fetch itself is blocked)
unless CORS is reconfigured to `AllowCredentials: true` with a specific
(non-wildcard) origin. Opening credentialed CORS for this one new
feature is not worth widening the *entire* API's CORS surface (every
other endpoint would gain a cookie-based credential path it doesn't need
today, an unrelated regression in blast radius for the sake of one
flow). Instead:

- `POST /mcpservers/oauth/start` returns the linking token in the
  **response body**, not a cookie: `{ authorize_url: string, link_token:
  string }` (`link_token` is the same value as `state`, just also
  surfaced to the JS caller instead of stashed in a cookie the browser
  manages).
- square-admin stores `link_token` in `sessionStorage` (tab-scoped,
  cleared on tab close -- appropriately short-lived for a flow that
  completes in the same tab within minutes) keyed by `mcp_oauth_link`,
  then does the full-page redirect to `authorize_url`.
- On return, the vendor's redirect lands on
  `/mcpservers/oauth/callback` (api-manager, public, §10), which does
  NOT check a cookie -- it only validates `state` against
  `ai_mcp_oauth_states` (Layer 1) and immediately 302-redirects to
  `admin.voipbin.net/#/resources/mcpservers/oauth-return?state=<state>`
  (or `mcp_server_id=<id>` on success) WITHOUT completing the token
  exchange itself.
- **The actual Layer 2 check happens on a NEW square-admin-triggered
  call**, not in the public callback: the `oauth-return` page reads
  `state` from the URL, reads its own `link_token` back out of
  `sessionStorage`, and if they match, calls a new AUTHENTICATED
  endpoint `POST /mcpservers/oauth/complete { state }` (v1.0 group, JWT
  auth like every other authenticated endpoint -- no CORS change needed,
  this is the exact same auth model every other `POST` in this API
  already uses). **The real security boundary is server-side, not the
  sessionStorage check**: `AIV1McpOAuthComplete`'s ai-manager handler
  MUST look up `ai_mcp_oauth_states` by `state` and verify
  `row.CustomerID == <the JWT-authenticated caller's customer_id>`
  before proceeding -- rejecting with the same generic "state not found
  or already used" on mismatch (not a more specific error, §7a Layer 1's
  existing anti-enumeration posture). This single check is what actually
  defeats the hijack even in the worst case (victim never sees
  square-admin's sessionStorage at all, e.g. they only ever interact
  with the raw `authorize_url` link an attacker sent them): the attacker
  generated `state` bound to THEIR OWN `customer_id`, so the only
  account that can ever successfully call `/oauth/complete` for that
  `state` is one authenticated as the attacker -- a victim who is not
  simultaneously logged into VoIPBin as the attacker structurally cannot
  complete it, regardless of which browser/tab clicks through the vendor
  consent screen. The `sessionStorage` `link_token` check is then a
  UX-layer improvement on top (fails fast, client-side, with a clearer
  message than a 403 from the server), not the actual trust boundary.
  On success, `/oauth/complete` performs the actual token exchange (§7
  steps 9-12, moved here from the public callback) and returns the
  created/updated McpServer. If `sessionStorage` has no matching
  `link_token` (different browser, different tab, or an attacker who
  never had it), the frontend never calls `/oauth/complete` and the flow
  simply fails client-side with "this authorization wasn't started in
  this browser" -- the server-side `ai_mcp_oauth_states` row is still
  there but nothing ever claims it, and it expires normally per
  `tm_expire` (§5).
- This reframing means `GET /mcpservers/oauth/callback` becomes a THIN,
  public, no-side-effect redirect relay (it never touches vendor token
  endpoints or the database beyond a read to confirm the `state` exists,
  moved to `/oauth/complete`) -- and the actual sensitive work (§7 steps
  6-8: exchanging `code` for tokens, encrypting, writing McpServer) now
  happens on an authenticated endpoint that inherently proves "this is
  the same customer session that clicked the catalog card", without any
  new cookie/CORS surface. §5, §7 (flow diagram), §9, §10, §12 below are
  updated to reflect this split.

### 7b. PKCE

VoIPBin generates `code_verifier` (43-128 char random string) and
`code_challenge = base64url(sha256(code_verifier))` at step 2 for EVERY
vendor, using the `S256` method exclusively (never `plain`) -- both
GitHub (where `code_challenge` is "strongly recommended", `S256`-only:
GitHub's own OAuth docs do not support the `plain` transform) and Linear
(where PKCE is a stated requirement) accept `S256`, so there is no
vendor branch needed on this. Sending an unused `code_challenge` to a
vendor that does not require it is harmless per the OAuth 2.1 spec and
keeps `mcpoauthhandler`'s flow code vendor-uniform.

**Open implementation risk, not a design blocker** (flagged during
Round 1 review): whether GitHub's OFFICIAL remote MCP server
(`api.githubcopilot.com/mcp/`) actually accepts a classic-OAuth-App
`gho_*` access token as its Bearer credential was not independently
confirmed against GitHub's own docs during this design -- available
search results only confirmed PAT / GitHub-App-user-token usage
examples for that specific server. Implementation MUST smoke-test this
with a real GitHub OAuth App token against the real
`api.githubcopilot.com/mcp/` endpoint before considering the GitHub pilot
done; if it turns out that server requires a GitHub App (not an OAuth
App) token, §6's GitHub `vendorConfig` needs to switch from an OAuth App
to a GitHub App registration (same DB/crypto/flow shape either way, only
the app-registration type and resulting token prefix changes).

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
5. **Refresh-call failure (Round 1 review, minor/non-blocking):** if
   step 3's POST to the vendor's token endpoint itself fails (e.g. the
   vendor has revoked the refresh token server-side, `invalid_grant`),
   `GetValidAccessToken` returns an error for this call -- it does
   NOT clear the stored refresh token or mark the row unrefreshable, so
   the very next tool call (which may happen moments later, e.g. a
   multi-turn AIcall) re-attempts the same doomed refresh. For 2 pilot
   vendors at expected-low tool-call volume this is not a design
   blocker, but implementation should add a short in-memory failure
   backoff (skip re-attempting a refresh for e.g. 60s after an
   `invalid_grant`-class failure on the same server ID) to avoid
   hammering the vendor's token endpoint during an active conversation
   with a stale connection -- a bounded, local optimization, not a new
   subsystem.

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
  accepts an optional `mcp_server_id` field. **Ownership check (Round 1
  design review finding, IDOR): when `mcp_server_id` is present,
  `AIV1McpOAuthStart`'s ai-manager-side handler MUST first
  `McpServerGet(mcp_server_id)` and verify `res.CustomerID ==
  customerID` (the JWT-authenticated caller's customer_id, the same
  parameter `AIV1McpServerCreate`/`Update` already take -- see §10 for
  `AIV1McpOAuthStart`'s exact signature), returning `cerrors.NotFound`
  (the existing `MCP_SERVER_NOT_FOUND`, matching `Get`/`Update`/
  `Delete`'s existing not-found response for cross-customer access -- do
  not leak "found but not yours" via a different error) before creating
  the state row.** Without this check, any authenticated customer could
  pass another customer's `mcp_server_id` and overwrite that row's OAuth
  connection with their own vendor account on callback -- a
  cross-customer data corruption / DoS, not merely an access-control
  gap, since the row's owner would then silently lose their own
  connection. This mirrors the ownership check every other
  `mcpserverhandler`/`servicehandler` method already performs by
  construction (the `id` a customer can reference is scoped through
  `McpServerGetsByCustomerID`-style listing) -- OAuth start is the one
  new entrypoint that accepts a bare `mcp_server_id` without that
  scoping, so it must do the check explicitly. When present and owned,
  the state row carries that ID (§5's `mcp_server_id` column), and
  `AIV1McpOAuthComplete` (§7 step 7-12) does `McpServerUpdate` on the
  existing row instead of `McpServerCreate` -- same name/URL/vendor,
  fresh tokens. This is the only way `auth_type: oauth` rows get their
  tokens replaced; there is no "PUT new access token" path (consistent
  with §4's "only the OAuth endpoints can produce an oauth-authenticated
  row").
- Changing `oauth_vendor` on an existing row is not supported (§4) --
  the customer deletes and re-adds instead. Two vendors, low value in
  supporting an in-place vendor swap that would need to re-derive
  name/URL from the new vendor's catalog entry anyway.

## 10. api-manager: new endpoints

**Revised in Round 2**: three endpoints instead of two (§7a's Layer 2
correction split the old single callback into a thin public relay plus
an authenticated completion call).

`bin-api-manager/server/mcpservers_oauth.go` (new file, same package/
conventions as `mcpservers.go`):

```
POST /mcpservers/oauth/start      (authenticated, v1.0 group, JWT)
  body: { vendor: "github"|"linear", mcp_server_id?: uuid }
  -> { authorize_url: string, link_token: string }
     (link_token == state, §7a Layer 2 -- returned in the body, NOT a
     cookie, to avoid the credentialed-CORS problem Round 2 caught)

GET  /mcpservers/oauth/callback   (PUBLIC, registered next to
                                    /auth/password-reset in main.go --
                                    same public-with-token-security
                                    pattern, §7a Layer 1)
  query: ?code=...&state=...  (or ?error=... on vendor-side denial)
  -> thin relay: confirms `state` exists in ai_mcp_oauth_states (does
     NOT delete it, does NOT exchange the code, does NOT touch McpServer
     -- that all happens in /oauth/complete below), then 302 redirects
     to admin.voipbin.net/#/resources/mcpservers/oauth-return?state=...
     &code=... (or &error=... passthrough). If `state` does NOT exist
     (expired, already consumed by a prior /oauth/complete call, or
     simply invalid), the relay still 302-redirects to the same
     `oauth-return` route but WITHOUT `code` -- e.g.
     `?state=...&error=invalid_state` -- rather than rendering its own
     error page (this endpoint has no UI of its own; `oauth-return`
     owns all user-facing error presentation, consistent with the
     vendor-denial `?error=...` passthrough case already handled the
     same way). `oauth-return` then shows the SAME generic "this
     authorization wasn't started in this browser (or already
     completed/expired)" message it would show for any other Layer 2
     mismatch (§7a) -- not a more specific one, for the same
     anti-enumeration reason Layer 1 already avoids specific errors.

POST /mcpservers/oauth/complete   (authenticated, v1.0 group, JWT)
  body: { state: string, code: string }
  -> { mcp_server: {...} } on success. This is where the REAL security
     boundary lives (§7a): AIV1McpOAuthComplete's handler verifies the
     state row's customer_id matches the JWT's customer_id before doing
     anything else, then performs the token exchange, encrypts, and
     creates/updates the McpServer row (§7 steps 9-12: delete state row,
     exchange code, encrypt, upsert).
```

**Rate limiting (Round 1 design review finding):** every existing public
route group in `cmd/api-manager/main.go` has a dedicated
`middleware.RateLimit(...)` -- `auth` (`auth_public`), `provisioning`
(`provisioning_public`). `GET /mcpservers/oauth/callback` MUST follow the
same convention: registered in its own `app.Group("/mcpservers/oauth")`
with a new `mcp_oauth_callback_public` rate limit config flag
(`RateLimitMcpOAuthCallbackPublicRPS`/`...Burst`, same naming shape as
`RateLimitAuthPublicRPS`/`...Burst`), NOT left unprotected. `POST
/mcpservers/oauth/start` and `POST /mcpservers/oauth/complete` are both
authenticated and already covered by the existing `v1.0` group's
`CustomerRateLimit` -- no additional rate-limit config needed for
either.

Both public and authenticated routes are added to
`bin-openapi-manager/openapi/paths/mcpservers/` (new `oauth_start.yaml`,
`oauth_callback.yaml`, `oauth_complete.yaml`) and regenerate both
`bin-api-manager/gens/openapi_server/gen.go` AND
`bin-openapi-manager/gens/models/gen.go` together (the exact drift this
monorepo's CI caught during the PUT partial-update PR #1291 -- noted
here explicitly so implementation doesn't repeat that CI failure).

RPC signatures (`bin-common-handler/pkg/requesthandler`, following the
existing `AIV1McpServerCreate`/`AIV1McpServerUpdate` naming/parameter
convention -- explicit here per a Round 2 review request for the same
level of concreteness §4/§8 already have):

```go
// AIV1McpOAuthStart generates a state+PKCE pair and, when mcpServerID is
// non-nil, verifies customerID owns that row (§9) before persisting the
// state. Returns the vendor's authorize_url and the link_token (== state,
// §7a Layer 2) for the caller to store client-side.
AIV1McpOAuthStart(ctx context.Context, customerID uuid.UUID, vendor string, mcpServerID *uuid.UUID) (authorizeURL string, linkToken string, err error)

// AIV1McpOAuthComplete verifies the state row's CustomerID matches
// customerID (§7a's real security boundary), exchanges code for tokens,
// encrypts them, and creates/updates the McpServer row (§7 steps 9-12).
AIV1McpOAuthComplete(ctx context.Context, customerID uuid.UUID, state string, code string) (*mcpserver.McpServer, error)
```

Both are internal-only RPCs -- api-manager is still the only public HTTP
surface; nothing here creates a second public entrypoint into
ai-manager. `GET /mcpservers/oauth/callback`'s thin relay does not need
its own RPC at all -- a direct `dbhandler`-style existence check
(reusing `ai-manager`'s existing internal DB access pattern via a very
small `AIV1McpOAuthStateExists(ctx, state string) (bool, error)` RPC) is
sufficient, since it performs no mutation.

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
instead it immediately calls `POST /mcpservers/oauth/start`, stores the
returned `link_token` in `sessionStorage` (§7a Layer 2), and redirects
the browser to the returned `authorize_url`. A new route,
`/resources/mcpservers/oauth-return`, handles the vendor's return trip
(§7 steps 4-6): it reads `state`/`code` (or `error`) from the URL,
**immediately calls `history.replaceState()` to strip `code`/`state`
from the visible URL/browser history** (Round 3 review recommendation --
`code` is a one-time authorization code; while PKCE means it cannot be
exchanged without the server-side `code_verifier`, minimizing where it
sits in plaintext, e.g. shoulder-surfable address bars or browser
history sync, is a defense-in-depth improvement with no cost), then
compares `state` against the stored `link_token` (client-side fast-fail
per §7a), and on a match calls `POST /mcpservers/oauth/complete { state,
code }`. On success, it navigates to the returned server's detail page.
On a `link_token` mismatch or `POST /oauth/complete` failure, an
`ActionFeedback` error banner surfaces on that same route (not a
redirect loop back to the catalog -- the customer stays on
`oauth-return` and sees the failure with a way back to the MCP servers
list).

The MCP server detail page (`mcpservers_detail.js`) gains an
`oauth_vendor`-aware view: when `auth_type === 'oauth'`, it shows
"Connected to GitHub" (or Linear) with a "Reconnect" button (calls
`POST /mcpservers/oauth/start` with this server's ID, §9, reusing the
same `oauth-return` route/flow above) instead of the existing
secret/api_key_header edit fields, which do not apply to oauth rows.

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
- The OAuth `state`/`code` parameters are never logged. **Revised in
  Round 2**: since the token exchange (and the `code` value) moved to
  `POST /mcpservers/oauth/complete` (an authenticated JSON body, not a
  query string -- request bodies are not written to gin's access log the
  way query strings are), the public `GET /mcpservers/oauth/callback`'s
  exposure is now limited to `state`/`code` appearing in its OWN query
  string and in the 302 `Location` header it emits (both access-logged
  by default). This is still sensitive enough to warrant the same
  `SkipPaths` treatment as `/provisioning/extension` (below) -- `code`
  is a one-time authorization code, and while it alone cannot be
  exchanged without PKCE's `code_verifier` (which never leaves
  `ai_mcp_oauth_states`), it should not be gratuitously logged anyway.
  `POST /mcpservers/oauth/complete`'s handler must still apply the
  existing `truncateForError`-style discipline (`mcptoolhandler/
  client.go` precedent) to its own error-path log lines, since a logged
  error message containing the raw `code` would defeat the point.
- `/mcpservers/oauth/start`, `/mcpservers/oauth/callback`, and
  `/mcpservers/oauth/complete` are all added to `runListenHTTP`'s
  access-log `SkipPaths` in `bin-api-manager/cmd/api-manager/main.go`
  (same treatment as `/provisioning/extension`, §2's existing
  precedent) -- covering all three keeps the policy uniform across the
  whole OAuth flow rather than relying on the body-vs-query-string
  distinction alone to protect `/complete`.
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

## 15. Round 1 design review disposition

Independent adversarial review found 4 items requiring changes (all
fixed in this revision) plus 2 minor/non-blocking notes (also addressed):

- **[FIXED, security-critical]** §7a lacked a defense against OAuth
  "account-linking hijack" (RFC 6749 §10.12): the `state` param alone
  binds to `customer_id` but not to a specific browser, letting an
  attacker share their own `authorize_url` with a victim. Added Layer 2:
  a `SameSite=Lax`/`HttpOnly`/`Secure` linking cookie set at
  `/mcpservers/oauth/start` and re-checked at the callback.
- **[FIXED, security-critical]** §9's reconnect path accepted a bare
  `mcp_server_id` with no ownership check -- an IDOR letting any customer
  hijack another customer's MCP server row's OAuth connection. Added an
  explicit `McpServerGet` + `CustomerID` match requirement before
  creating the state row.
- **[FIXED]** §10 omitted rate limiting for the new public callback
  endpoint, breaking this monorepo's existing convention (every public
  route group has one). Added `mcp_oauth_callback_public` rate limit
  config, matching `auth_public`/`provisioning_public`'s naming.
- **[FIXED]** §5's DB column types (`VARBINARY(512)`/`VARBINARY(32)`)
  did not match the ACTUAL existing `secret_ciphertext blob` /
  `secret_nonce binary(12)` columns in `ai_mcp_servers` (verified against
  the real Alembic migration file). Corrected to `blob`/`binary(12)`.
- **[FIXED, minor]** §2's claim that "GitHub and Linear's interactive
  flow uses DCR" was imprecise -- GitHub does not support DCR at all;
  only Linear does. Corrected the reasoning to not overstate GitHub's
  DCR support while keeping the same conclusion (pre-registered apps for
  both, since GitHub requires it either way).
- **[FIXED, minor]** §8 did not address refresh-call failure caching --
  a failed refresh (e.g. vendor-revoked refresh token) would be
  re-attempted on every subsequent tool call with no backoff. Added a
  note that implementation should add a short in-memory failure backoff.
- **[NOTED, not fixed -- flagged as an open implementation risk in §7b]**
  whether GitHub's official remote MCP server actually accepts a
  classic-OAuth-App token was not independently confirmed by the
  reviewer's available search results (only PAT/GitHub-App-user-token
  usage examples were found). This is a factual question that can only
  be resolved by an actual smoke test against the live endpoint with a
  real registered OAuth App -- not resolvable via further research, and
  explicitly called out in §7b as a mandatory pre-"done" smoke test for
  implementation, with a documented fallback (switch to a GitHub App
  registration) if it turns out to be required.

## 16. Round 2 design review disposition

Independent adversarial review re-confirmed all 6 Round 1 fixes against
actual code (RPC signatures, DB migration file, rate-limit middleware,
config naming -- all matched), then found 1 new BLOCKING issue introduced
by Round 1's own fix, plus 1 minor completeness gap:

- **[FIXED, blocking]** Round 1's Layer 2 mitigation (a `Set-Cookie` on
  `/mcpservers/oauth/start`) does not work with this API's actual
  deployment: square-admin calls api-manager cross-origin, and
  `main.go`'s global CORS is `AllowCredentials: false` + wildcard
  origin, which is fundamentally incompatible with cookie-based
  auth/linking without either breaking the flow outright or forcing a
  much larger CORS regression (credentialed CORS API-wide). **Redesigned
  §7/§7a/§9/§10/§11/§12**: the linking value moves from a cookie to a
  response-body `link_token` stored client-side in `sessionStorage`; the
  public callback becomes a thin, no-mutation redirect relay; the actual
  token exchange and (critically) the REAL security check --
  `state.CustomerID == JWT customer_id` -- move to a new AUTHENTICATED
  endpoint, `POST /mcpservers/oauth/complete`, which needs no CORS
  changes at all (same JWT auth model as every other authenticated
  endpoint in this API). This also has a side benefit noted during the
  fix: the security boundary is now structurally simpler to reason about
  (an authenticated ownership check on a server-generated token) than a
  client-supplied cookie ever was.
- **[FIXED, minor]** §9's ownership-check description referenced
  "the authenticated caller's customer_id passed in the RPC" without
  showing the actual parameter; §10 now includes explicit Go signatures
  for `AIV1McpOAuthStart`/`AIV1McpOAuthComplete`, matching the
  concreteness already present in §4/§8's code blocks.
- Reconnect ownership-check logic itself, DB schema, and rate-limiting
  additions from Round 1 were independently re-verified against the
  actual codebase this round and confirmed unchanged/correct -- no
  further changes needed there.

## 17. Round 3 design review disposition

Independent adversarial review confirmed the 3-endpoint architecture
itself (thin public relay + authenticated `/oauth/complete` doing the
real ownership check) is sound, but found that Round 2's rewrite left
ONE paragraph in §7a's "Layer 1" section un-updated from the original
(Round 0) design, where the callback itself deleted the state row --
directly contradicting Round 2's own "callback is a no-mutation relay"
contract, plus inconsistent step numbers scattered across §5/§7a/§10:

- **[FIXED]** §7a Layer 1's "the callback handler deletes the row..."
  sentence was stale Round-0 text. Corrected to explicitly state
  `/oauth/complete` deletes the row (diagram step 9), never the public
  callback, and clarified that a replayed CALLBACK request is harmless
  (read-only) while a replayed COMPLETE request is what actually gets
  rejected.
- **[FIXED]** Three inconsistent step-number references to the
  delete/exchange block (§5's "step 5", §7a's "steps 6-8", §10/RPC
  comment's "steps 8-12") unified to "step 9" (delete) / "steps 9-12"
  (delete, exchange, encrypt, upsert) throughout.
- **[FIXED, minor]** §11 now specifies that the `oauth-return` page
  calls `history.replaceState()` to strip `code`/`state` from the
  visible URL immediately on load, reducing where the one-time
  authorization code sits in plaintext (address bar, browser history).
- **[FIXED, minor]** §10's `GET /mcpservers/oauth/callback` entry now
  specifies its behavior when `state` does not exist (expired/already
  consumed/invalid): redirects to `oauth-return` with a generic
  `error=invalid_state` rather than rendering its own error UI, and
  `oauth-return` shows the same generic message it already uses for a
  `sessionStorage` `link_token` mismatch -- no new error-handling
  surface, and no information disclosure about WHY the state was
  rejected.
