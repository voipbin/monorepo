# Customer-Configured MCP Tool Integration (Phase 1: text sessions)

Status: Approved (7 review rounds: CR -> CR -> CR -> CR -> CR -> APPROVE -> APPROVE)
Service: bin-ai-manager (+ bin-openapi-manager, bin-api-manager thin passthrough)
Date: 2026-09-11

## 1. Problem statement

Every VoIPBin AI's tool set today is a fixed catalog defined in Go source
(`bin-ai-manager/pkg/toolhandler/definitions.go`, `models/tool/main.go`). A
customer can only pick a subset of VoIPBin's own built-in tools
(`connect_call`, `send_email`, `search_knowledge`, ...) via
`AI.ToolNames`. There is no way for a customer to expose their OWN systems
(internal ticketing, CRM lookups, inventory, scheduling) to the LLM during a
conversation. Every customer's integration need is different, and VoIPBin
cannot pre-build a connector per customer.

MCP (Model Context Protocol) has become the de facto standard for exposing
an external system's capabilities to an LLM as callable tools over a thin
HTTP/JSON-RPC contract. This design lets a customer register their own
remote MCP server(s) and have VoIPBin's LLM sessions call them exactly like
a built-in tool.

## 2. Decisions locked (2026-09-11, CEO sign-off)

1. **Scope**: Phase 1 covers all Go-only (non-realtime-voice) AI sessions.
   Gate: **`AIcall.ConfbridgeID == uuid.Nil`** (Round 5 correction --
   see the note below; `PipecatcallID` was the originally stated gate and
   is factually wrong). This correctly includes text-only
   `conversation`/`contact_case` (Insight) AIcalls and excludes
   `call`-reference AIcalls that always run through pipecat, regardless
   of `ReferenceType`.

   **Round 5 correction, verified against live code:** the field
   originally named here as the scoping gate, `AIcall.PipecatcallID`, is
   NOT usable as one -- `CreateByMessaging` (`db.go:146`) sets
   `PipecatcallID` from a freshly generated UUID
   (`h.utilHandler.UUIDCreate()`) for every messaging-path AIcall too
   (`start.go:1159`), exactly as `startAIcallByRealtime`/`Create` do for
   the realtime path (`start.go:1102,1108-1109`). **Every AIcall, of
   every reference type, has a non-nil `PipecatcallID` from the moment
   of creation.** A literal `PipecatcallID == uuid.Nil` filter selects
   zero rows and cannot distinguish Phase 1 (messaging) sessions from
   out-of-scope (realtime) ones at all -- it would be a silent no-op
   guard if implemented as originally stated, a landmine for Phase 2.
   The field that actually distinguishes the two paths is
   `ConfbridgeID`: `CreateByMessaging` hardcodes `ConfbridgeID: uuid.Nil`
   (`db.go:145`), while `startAIcallByRealtime` passes through a real,
   caller-supplied `confbridgeID` (`start.go:1088,1108-1109`). All uses
   of "the `PipecatcallID` gate" elsewhere in this document (§4's
   diagram) refer to `ConfbridgeID` instead; §9.1's actual
   implementation was never affected by this error (it wires
   `resolveTools` by call-site, not by reading this field at runtime),
   but the stated rationale needed correcting before it misleads a
   future Phase 2 reader.
2. **Registration unit**: Customer-level. An `McpServer` resource belongs to
   a `customer_id`, not to a single `AI`. Each `AI` opts a registered server
   in/out via a new whitelist field (mirrors the existing `ToolNames`
   pattern), so one registration is reusable across every AI the customer
   owns.
3. **Authentication**: Phase 1 ships full secret storage, not "public
   servers only". A customer can store one bearer-token/API-key credential
   per `McpServer` row.
4. **Secret storage**: DB-level envelope encryption (AES-256-GCM), key
   material from a new `bin-ai-manager` config flag /
   env var (`mcp_secret_encryption_key` / `MCP_SECRET_ENCRYPTION_KEY`),
   following the existing `engine_key_chatgpt` config pattern. No external
   Vault dependency -- keeps VoIPBin self-hosting deploys unchanged.

## 3. Non-goals (Phase 1)

- Realtime voice sessions (pipecat-routed AIcalls). MCP round-trip latency
  inside the STT->LLM->TTS loop needs its own latency budget and timeout/
  fallback design; deferred to Phase 2.
- MCP OAuth 2.1 / dynamic client registration flows. Phase 1 supports a
  single static bearer token or API key header per server, matching what
  the large majority of self-hosted/internal MCP servers actually expose
  today.
- stdio-transport MCP servers (local process spawn). Only remote
  Streamable-HTTP MCP servers are supported -- VoIPBin cannot spawn
  arbitrary customer binaries inside its own pods.
- Tool result caching / cross-session MCP connection pooling beyond a
  per-call HTTP client with a bounded timeout.
- Per-tool cost/rate governance beyond the existing per-AI tool whitelist
  and a blanket per-call timeout. Detailed spend controls are a Phase 2+
  discussion once real usage data exists.

**Residual risk, named for the record (Round 7 finding, MINOR, explicitly
acknowledged rather than silently deferred):** a customer-registered MCP
server is a more adversarial-capable actor than any built-in tool --
per-call timeout and the per-AI `McpServerIDs` whitelist bound a SINGLE
call, but nothing in Phase 1 bounds aggregate outbound call
volume/concurrency toward one customer's server, nor total outbound
capacity spent across all customers' MCP servers combined. A malicious or
compromised customer-registered server could still be hit at whatever
rate the LLM's own tool-calling cadence produces. This is a known,
deliberately deferred Phase 1 tradeoff (consistent with the doc's overall
minimal-Phase-1 posture), not an oversight -- flagged explicitly here so
the CEO sign-off record shows it was seen, and so Phase 2's spend/rate
governance work (above) has a concrete starting point.

## 4. Architecture overview

```
Customer                                bin-ai-manager                 Customer's MCP server
--------                                --------------                 ----------------------
POST /mcpservers  -------------------->  mcpserverhandler.Create
  {name, url, auth_type, secret}          - validates URL (SSRF guard)
                                           - encrypts secret
                                           - stores row

AI.tool_names += ["mcp:<server_id>"]  ->  ai/tool_validation.go
  (per-AI opt-in, existing pattern)        validates the reference exists
                                           and belongs to the same customer

AIcall session starts (text-only)    ->  aicallHandler.resolveTools
                                           - existing GetByNames(ToolNames)
                                           - NEW: for every "mcp:<id>" entry,
                                             mcpToolHandler.ListTools(ctx, id)
                                             (cached, short TTL) and merges
                                             the remote tool defs into the
                                             LLM's tool list, namespaced
                                             mcp_<server_slug>_<tool_name>

LLM emits a function call for a          aicallHandler.ToolHandle
namespaced mcp_* tool name           ->    - existing mapFunctions dispatch
                                             miss falls through to a NEW
                                             prefix check: "mcp_" -> routes
                                             to mcpToolHandler.CallTool
                                                                          -->  POST <server_url>/mcp
                                                                               (JSON-RPC tools/call,
                                                                                Authorization header
                                                                                decrypted server-side)
                                           - bounded timeout, single retry  <-- 200 OK / tool result
                                           - result wrapped into the same
                                             messageContent shape as every
                                             other tool (Result/Message/...)
```

Key point: this reuses the EXISTING dispatch seam
(`aicallHandler.ToolHandle`'s `mapFunctions` map + the tool-result message
persistence path) rather than building a parallel pipeline. MCP tools are a
new *source* of tool definitions and a new *dispatch branch*, not a new
subsystem alongside `ToolHandle`. Opt-in is via the NEW `AI.McpServerIDs`
field (see §5), kept separate from the existing closed-enum
`AI.ToolNames` whitelist.

## 5. New domain model: `McpServer`

New package `bin-ai-manager/models/mcpserver`, new table `ai_mcp_servers`.

```go
package mcpserver

type AuthType string

const (
    AuthTypeNone   AuthType = ""        // no Authorization header sent
    AuthTypeBearer AuthType = "bearer"  // Authorization: Bearer <secret>
    AuthTypeAPIKey AuthType = "api_key" // <APIKeyHeader>: <secret>
)

type Status string

const (
    StatusActive Status = "active"
    // Disabled = customer explicitly deactivated; excluded from ListTools/CallTool.
    StatusDisabled Status = "disabled"
)

type McpServer struct {
    identity.Identity // ID, CustomerID

    Name   string `json:"name,omitempty" db:"name"`
    Detail string `json:"detail,omitempty" db:"detail"`

    URL    string `json:"url,omitempty" db:"url"` // Streamable-HTTP MCP endpoint, https only
    Status Status `json:"status,omitempty" db:"status"`

    AuthType     AuthType `json:"auth_type,omitempty" db:"auth_type"`
    APIKeyHeader string   `json:"api_key_header,omitempty" db:"api_key_header"` // only meaningful for AuthTypeAPIKey, e.g. "X-API-Key"

    // SecretCiphertext is the AES-256-GCM-encrypted bearer token / API key.
    // NEVER included in WebhookMessage or any GET response body -- see §8.
    SecretCiphertext []byte `json:"-" db:"secret_ciphertext"`
    SecretNonce      []byte `json:"-" db:"secret_nonce"`
    // KeyVersion records which entry of MCP_SECRET_ENCRYPTION_KEYS encrypted
    // this row's secret, so rotating the configured key set never breaks
    // decryption of rows encrypted under a still-configured older version.
    // See §6 "Key rotation story". Phase 1 always writes the single
    // configured current version; the column exists so Phase 2's rotation
    // sweep needs no migration.
    KeyVersion int    `json:"-" db:"key_version"`
    HasSecret  bool   `json:"has_secret" db:"-"` // derived, exposed instead of the secret itself

    TMCreate *time.Time `json:"tm_create" db:"tm_create"`
    TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
    TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}
```

`ai_mcp_servers` table (mirrors `ai_teams` shape):

```sql
create table ai_mcp_servers(
  id                binary(16),
  customer_id       binary(16),

  name              varchar(255),
  detail            text,

  url               varchar(2048),
  status            varchar(16),

  auth_type         varchar(16),
  api_key_header    varchar(255),
  secret_ciphertext varblob,
  secret_nonce      binary(12),
  key_version       smallint,

  tm_create datetime(6),
  tm_update datetime(6),
  tm_delete datetime(6),

  primary key(id)
);

create index idx_ai_mcp_servers_create on ai_mcp_servers(tm_create);
create index idx_ai_mcp_servers_customer_id on ai_mcp_servers(customer_id);
```

### AI whitelist extension (revised after Round 1 review, see §16 M1/C2)

**Round 1 finding (accepted):** overloading `AI.ToolNames []tool.ToolName`
with synthetic `mcp:<id>` entries would corrupt a closed, statically-
enumerable type. `tool.ToolName` and `ai.AllowedToolNames`/
`ai.ValidateToolNames` are a compile-time-fixed membership set
(`models/tool/main.go`'s `AllToolNames`/`AllInsightToolNames` const
slices), and that same set is also read by bin-pipecat-manager's runtime
tool expansion (`tool_validation.go:21` doc comment) -- injecting a
dynamic, per-customer, per-row value family into it would mean every other
consumer of `ToolNames` (today and Phase 2's realtime path) has to learn
to recognize and skip a `mcp:` prefix it doesn't understand, silently.

**Revised design:** a NEW, separate field, `AI.McpServerIDs []uuid.UUID`
(`json:"mcp_server_ids,omitempty" db:"mcp_server_ids,json"`), added
alongside `ToolNames` on the `AI` struct and `ai_ais` table (new nullable
JSON column, same storage shape as `ToolNames`). This keeps "which
built-in tools" (`ToolNames`, closed enum, unchanged) and "which customer-
registered MCP servers" (`McpServerIDs`, open per-row references) as two
independently-validated concerns instead of conflating them:

```go
type AI struct {
    // ... existing fields unchanged ...
    ToolNames    []tool.ToolName `json:"tool_names,omitempty" db:"tool_names,json"`
    McpServerIDs []uuid.UUID     `json:"mcp_server_ids,omitempty" db:"mcp_server_ids,json"`
}
```

Validation is a NEW function, not a branch inside the existing pure
`ai.ValidateToolNames(t Type, toolNames []tool.ToolName) error`
(`models/ai/tool_validation.go:72` -- verified signature, no `ctx`, no db
handle; it is called from `pkg/aihandler/chatbot.go:52,155` and stays
untouched by this design):

```go
// ValidateMcpServerIDs lives in pkg/aihandler (has db access), not
// models/ai (which must stay a pure, DB-free package -- mirrors why
// ai.ValidateToolNames itself takes no ctx/db today).
func (h *aiHandler) ValidateMcpServerIDs(ctx context.Context, customerID uuid.UUID, ids []uuid.UUID) error {
    for _, id := range ids {
        srv, err := h.db.McpServerGet(ctx, id) // or a dedicated existence+ownership query
        if err != nil || srv == nil || srv.CustomerID != customerID {
            return fmt.Errorf("mcp_server_id %s is not accessible", id)
        }
    }
    return nil
}
```

Called from `pkg/aihandler`'s `Create`/`Update` (the same layer that
already owns the AI row's write path), NOT from `models/ai`. This is a
larger touch point than the original "gains a branch" framing implied --
it is a new handler-layer function plus wiring into `aihandler.Create`/
`Update`, and does not touch `chatbot.go`'s existing calls to
`ai.ValidateToolNames` at all (Type/AllowedToolNames semantics for
built-in tools are completely unaffected).

**Phase 2 / bin-pipecat-manager note (M1):** because `McpServerIDs` is a
field the realtime path does not yet read, Phase 1 leaves bin-pipecat-
manager's tool-expansion untouched by construction -- there is no `mcp:`
string for it to misinterpret, closing the latent landmine the original
draft would have created.

## 6. Secret encryption

New package `bin-ai-manager/pkg/secrethandler` (or a small helper inside
`mcpserverhandler` if usage stays single-consumer -- per
`bin-common-handler`'s 3-service admission rule this does NOT belong in the
shared library).

- AES-256-GCM, 32-byte key from `MCP_SECRET_ENCRYPTION_KEY` (base64,
  decoded at startup; `internal/config.Validate()` gains a check that the
  key decodes to exactly 32 bytes when any MCP feature flag is enabled --
  fail closed at boot, same posture as the existing listen-config validator).
- Nonce is random 12 bytes per encryption, stored alongside ciphertext
  (`secret_nonce` column) -- never reused, so no server-side nonce counter
  to maintain.
- Encrypt on write (`mcpserverhandler.Create`/`Update`), decrypt only at the
  point of building the outbound HTTP request in `mcpToolHandler.CallTool`.
  Decrypted plaintext never touches a log line, a DB read outside that one
  call site, or the WebhookMessage/GET response mapping.
- `HasSecret bool` is the only secret-derived field ever serialized back to
  the customer (GET /mcpservers/:id shows whether a secret is configured,
  never its value, not even masked/truncated -- masking a token is still a
  partial leak).

This mirrors the "single encryption helper, single decrypt call site"
shape recommended for any Phase-1 credential store; it deliberately does
NOT introduce a generic multi-tenant secrets-manager abstraction (YAGNI --
today's only consumer is `McpServer.SecretCiphertext`).

**Deliberate divergence from `AI.EngineKey` (Round 1 finding MN1):**
`AI.EngineKey` (`models/ai/main.go:67`) is stored in plaintext today (no
encryption annotation) AND is included verbatim in `WebhookMessage`
(`models/ai/webhook.go:29,68`) -- i.e. it is exposed both at rest and via
GET/webhook. `McpServer`'s secret intentionally gets stricter treatment on
BOTH axes (encrypted at rest, never serialized even as `HasSecret`-style
masking would still be a partial value leak -- only the boolean is
exposed). This is a deliberate raise of the bar for a NEW credential type,
not a retroactive claim that `EngineKey`'s existing behavior is wrong;
tightening `EngineKey` itself is out of scope for this design and would be
its own follow-up if judged worth the churn (it is a long-shipped field
with existing external consumers of the webhook payload).

**Key rotation story (Round 2 finding, MAJOR):** a single static key with
no versioning means rotating `MCP_SECRET_ENCRYPTION_KEY` would make every
previously stored secret permanently undecryptable, with zero migration
path -- a real operational landmine for a self-hosted product where key
rotation is a normal, expected event (compliance, leak response, employee
offboarding). Adding a `key_version smallint` column to `ai_mcp_servers`
now, while the table is new, is cheap and forecloses the problem:

- Config becomes a small ordered list, not a single value: e.g.
  `MCP_SECRET_ENCRYPTION_KEYS` = comma-separated `<version>:<base64-key>`
  pairs. `key_version` on each row records which key encrypted it.
- Decrypt always looks up the key by the row's own `key_version`, so
  OLD rows keep decrypting under their original key indefinitely.
- Rotation procedure: append a new `<version>:<key>` entry (do not remove
  the old one yet), configure it as the "current" version for NEW writes,
  and optionally run a background re-encrypt sweep (`Update` each row
  under the old version to re-encrypt under the new one, `key_version`
  bumped) at the operator's convenience. Only after every row has been
  swept is it safe to drop the old key from the config list.
- This is a Phase 1 schema decision (one extra column, cheap) with a
  Phase 2 sweep-job implementation deferred -- Phase 1 ships with
  exactly one configured key version and no sweep, but the column exists
  so Phase 2 doesn't need a migration to retrofit it.

## 7. SSRF guard (mandatory, day one)

`McpServer.URL` is a customer-supplied outbound target that a VoIPBin pod
will call using pod-level network credentials/position. Both at
create/update time AND at call time:

- Scheme must be `https` (reject `http`, `file`, `gopher`, etc.).
- Resolve the hostname and reject if it resolves to a private/link-local/
  loopback/multicast range (RFC1918, 127.0.0.0/8, 169.254.0.0/16 incl. the
  cloud metadata address 169.254.169.254, ::1, fc00::/7) -- re-resolve
  and re-check at CALL time too, not only at registration time, to close
  the DNS-rebinding gap (a hostname can resolve to a public IP at
  registration and a private one minutes later). **Implementation
  requirement (Round 2 finding, MAJOR):** do NOT hand-check literal CIDR
  ranges against the raw resolved address string -- an IPv4-mapped IPv6
  address (e.g. `::ffff:169.254.169.254`) will silently bypass a naive
  string/CIDR comparison written against the IPv6 forms alone, a classic
  SSRF blocklist bypass. Use Go's `net.IP` built-in classifiers
  (`IsPrivate()`, `IsLoopback()`, `IsLinkLocalUnicast()`,
  `IsLinkLocalMulticast()`), which correctly unwrap IPv4-mapped IPv6 via
  `To4()` internally, rather than a bespoke CIDR-list implementation.
  **Dial-time pinning requirement (Round 3 finding, MAJOR):** validating
  a resolved IP and then letting `http.Client` independently re-resolve
  and dial is NOT sufficient -- a multi-A-record host could pass
  validation on one IP while the client dials a different (unvalidated)
  one, and a TTL=0 DNS record can flip between the validation lookup and
  the actual dial even within a single "call time" window. The client
  MUST validate and dial the SAME address: construct the `http.Client`
  with a custom `net.Dialer.Control` (or an equivalent `DialContext`
  hook) that runs the `net.IP` classifier checks against the actual
  socket peer address the dialer is about to connect to, rejecting the
  dial itself if it fails -- not a separate pre-flight `LookupHost` call
  whose result the transport is free to ignore. Every resolved IP for a
  hostname must be subject to this check, not just the first one a
  `LookupHost` call happens to return first. **TLS/SNI note (Round 4
  verification):** `net.Dialer.Control` is the correct primitive
  specifically because it inspects the resolved peer address without
  altering what address `http.Transport` dials or what hostname it uses
  for the TLS `ServerName` (SNI) -- `Control` runs as a callback during
  the dial `Transport` already performs, so certificate validation
  against the original hostname is unaffected. If a `DialContext` hook
  is used instead (the noted alternative), it MUST preserve this same
  property: do NOT manually substitute a literal resolved IP into the
  dial address inside `DialContext` without also explicitly keeping
  `tls.Config.ServerName` set to the ORIGINAL hostname -- a naive
  literal-IP substitution silently breaks TLS certificate validation
  (fails closed, not a security bypass, but a correctness landmine).
  `net.Dialer.Control` is the simpler, footgun-free choice and is the
  RECOMMENDED implementation; `DialContext` is noted only as an
  equivalent-but-more-error-prone alternative.
- Reject if the URL's host matches any internal VoIPBin service hostname
  pattern used inside the cluster (defense in depth beyond the IP check).
- Enforce a response size cap and the timeout below regardless of the
  above checks, so even an allowed host cannot be used to exhaust a pod.

This check is NOT a new abstraction bolted onto the whole platform -- it
belongs alongside `McpServer` create/update validation and the call-time
HTTP client construction, both single call sites.

## 8. `WebhookMessage` / GET response

`McpServer`'s `WebhookMessage` (and the REST GET/List response, same type
per the monorepo's style-A direct-marshal convention) excludes
`SecretCiphertext`/`SecretNonce` entirely and includes `HasSecret` instead:

```go
type WebhookMessage struct {
    identity.Identity
    Name, Detail string
    URL          string
    Status       Status
    AuthType     AuthType
    APIKeyHeader string
    HasSecret    bool
    TMCreate, TMUpdate, TMDelete *time.Time
}
```

## 9. Runtime flow detail

### 9.1 Tool-list resolution (session start)

`aicallHandler.resolveTools` (new helper, called from wherever the current
session assembles `[]tool.Tool` for the LLM -- `start.go`/`chat.go`, exact
call site confirmed during implementation against the live `GetByNames`
call path):

1. Built-ins: existing `toolHandler.GetByNames(AI.ToolNames)`, unchanged --
   `AI.ToolNames` is untouched by this design (§5 revision).
2. For each id in the NEW `AI.McpServerIDs` (already ownership-validated
   at save time by `aihandler.ValidateMcpServerIDs`, §5): fetch the
   `McpServer` row and **skip it if `Status != active`** (Round 2
   finding, MAJOR -- §5's doc comment claims disabled servers are
   "excluded from ListTools/CallTool" but the original draft never wired
   that exclusion into the runtime steps; this is the concrete
   enforcement point). Only for rows that pass the Status check, call
   `mcpToolHandler.ListTools(ctx, serverID)`.
   - Sends MCP `tools/list` JSON-RPC request to the server, with the
     decrypted Authorization header if configured. **This outbound call
     goes through the SAME SSRF guard as `CallTool` (§7) -- re-resolve
     and re-check the host at this call time too, not only at
     registration time.** (Round 1 finding M3: `ListTools` is the more
     frequent code path -- every session start on a cache miss -- so it
     cannot be exempted from the DNS-rebinding re-check that `CallTool`
     gets.)
   - Result cached in Redis, key `mcp:tools:<server_id>`, short TTL
     (config `mcp_tools_list_cache_ttl_seconds`, default 60) -- avoids one
     extra external HTTP round trip on every session start while still
     picking up the customer's tool changes within a minute.
   - On failure (timeout, non-2xx, malformed JSON-RPC): log + skip that
     server's tools for this session (degrade, don't fail the whole
     AIcall start over one broken customer MCP server) and increment a
     Prometheus counter labeled by server id's customer_id (NOT server id
     itself -- unbounded cardinality guard).
3. Each remote tool's name is namespaced `mcp_<first-8-hex-of-server-id>_<tool_name>`
   before being merged into the LLM's tool list, so a collision between two
   different customers' identically-named remote tools (or a remote tool
   name colliding with a VoIPBin built-in name) is negligible in practice
   within realistic scope (a single `AI`'s `McpServerIDs` list is
   realistically single digits to low tens of servers, and the 8-hex-char
   prefix carries 32 bits of entropy -- collision probability is
   negligible but not structurally zero, since it truncates a 128-bit
   UUID). **Failure mode if it ever did occur (Round 3 finding, MINOR,
   called out explicitly rather than asserted away):** a prefix collision
   between two servers whitelisted on the SAME `AI` would silently
   overwrite one server's entry in the Metadata tool-name map with the
   other's -- the last-registered tool for that prefix+name wins, a
   potential cross-server tool misroute WITHIN the same customer's own
   configuration (never cross-customer, since `McpServerIDs` is itself
   customer-scoped). Not a security boundary violation; acceptable
   Phase-1 risk given the scope, but not a zero-probability claim. The mapping
   from namespaced name back to (server_id, original_tool_name) is written
   into the AIcall's `Metadata` blob (mirrors `MetaKeyPromptSnapshots`),
   scoped for Phase 1 (Go-only, non-realtime sessions, no cross-pod
   routing constraint). **Write discipline (Round 1 finding M2):**
   `Metadata` is a single shared JSON column also holding
   `MetaKeyListenTranscribeID`/`MetaKeyListenOwnsTranscribe`/prompt
   snapshots etc; the write here MUST be a read-modify-write that merges
   only the new `MetaKeyMcpToolMap` key into the existing map (fetch
   current `AIcall.Metadata`, set one key, write back), never a blind
   overwrite of the whole blob -- mirrors how every other Metadata writer
   in this codebase already has to coexist with the others. This needs a
   single shared "upsert one Metadata key" helper if one does not already
   exist, rather than each writer reimplementing its own merge.

**AIcall reuse and Metadata staleness (Round 3 finding, MAJOR; correction
in Round 4 after the doc's own Round-3 fix was found to only cover ONE of
TWO reuse paths):** `bin-ai-manager` has TWO independent AIcall-reuse
code paths, not one, both gated by idle-window config, and BOTH are
genuine reopen boundaries for the Metadata tool-name mapping:

1. `startReferenceTypeContactCase` (contact_case/Insight AIcalls) --
   idle-expired check via `isAIcallIdleExpired`/`AIcallInsightSessionIdleMinutes`,
   reuse path calls `refreshInsightSessionIfIdle` (`insight_session.go`).
2. `startReferenceTypeConversation` (plain conversation AIcalls,
   `pkg/aicallhandler/start.go:288-366`) -- **verified against the live
   code (Round 4):** reuse is decided by `h.isAIcallReusable(res)`, gated
   by the SAME `AIcallConversationIdleTimeoutHours` config, at
   `start.go:319-320`. On reuse (`start.go:340-366`) it calls
   `interruptPreviousPipecatcall` + `UpdatePipecatcallIDAndActiveflowID`
   (+ a team-engine resolve for `AssistanceTypeTeam`) -- **and nothing
   else**. This branch does NOT call `refreshInsightSessionIfIdle` (that
   function is reached only from the contact_case path,
   `start.go:575-584`) and does NOT touch Metadata at all. The Round 3
   fix, which wired `resolveTools` only into
   `refreshInsightSessionIfIdle`'s trigger, therefore left this second
   reuse path completely unaddressed -- a conversation-type AIcall reused
   after an idle gap keeps its original `McpServerIDs`-derived tool list
   and `MetaKeyMcpToolMap` indefinitely, the same staleness bug Round 3
   described, just on the path the first fix didn't touch.

**Corrected resolution (Round 4):** rather than hooking `resolveTools`
into one reference-type-specific refresh function, it is called
explicitly from every branch that can produce a "this AIcall is now
serving a session" outcome -- both true creation AND reuse, for every
reference type:

- `startAIcallByMessaging` (`start.go:1137-1192`, the shared TRUE-CREATION
  helper both `startReferenceTypeConversation`'s non-reuse branch AND
  `startReferenceTypeContactCase`'s non-reuse branch call, PLUS
  `startReferenceTypeNone` and `StartTask` -- Round 5 verification
  confirmed all four callers, and since `resolveTools` is wired INSIDE
  `startAIcallByMessaging` itself rather than duplicated per caller,
  every one of them is covered by construction, not by individual
  enumeration): `resolveTools`
  runs here, alongside the existing `buildPromptSnapshots` call that
  already populates `MetaKeyPromptSnapshots` in the same `metadata` map
  literal (`start.go:1160-1164`) -- one extra key in the SAME map, no
  extra read-modify-write.
- `startReferenceTypeConversation`'s reuse branch
  (`start.go:340-366`): `resolveTools` runs here too, as a NEW step in
  that branch (it currently does not touch Metadata at all), using the
  read-modify-write discipline above.
- `startReferenceTypeContactCase`'s reuse branch: unchanged from the
  original Round 3 fix -- wired into `refreshInsightSessionIfIdle`'s
  trigger, sharing that function's existing read-modify-write pass.

This makes "AIcall now serving a session" (create OR reuse, any
reference type) the single, uniform trigger for the MCP tool list to be
current, rather than depending on which reference-type-specific
refresh function happened to already exist for an unrelated reason.

### 9.2 Tool-call dispatch

In `aicallHandler.ToolHandle` (`tool.go`), the existing `mapFunctions`
lookup gains the `mcp_`-prefix branch shown in full below (§9.2).

`toolHandleMcpCall`:
1. Resolve namespaced name -> `(server_id, original_tool_name)` from the
   AIcall's Metadata mapping written in §9.1.
2. Re-verify `server_id` is still in the AIcall's resolved AI's
   `McpServerIDs` whitelist (defense against a stale/forged tool name from
   a compromised LLM context -- the whitelist check MUST happen at call
   time, not only at list time). **This is necessary but NOT sufficient
   (Round 2 finding, MAJOR):** membership in `McpServerIDs` only proves
   the AI is still configured to use that server id -- it does NOT prove
   the server is still `active`. A server disabled mid-session (after
   `ListTools` already cached its tools into the AIcall's Metadata map,
   §9.1) must be re-checked for `Status == active` in step 3 below,
   the same way §9.1 already gates on it -- a stale, disabled server's
   tool must fail here even though it still passes the `McpServerIDs`
   membership check.
3. Fetch the `McpServer` row, **fail closed with `fillFailed` if
   `Status != active`** (the enforcement point named above), otherwise
   decrypt the secret using the row's own `KeyVersion` (§6), build the MCP
   `tools/call` JSON-RPC request with the LLM-supplied arguments passed
   through unchanged (same "LLM args -> JSON-RPC body" pass-through
   pattern as every built-in tool's `json.Unmarshal(tool.Function.Arguments, ...)`).
4. POST with a bounded timeout (config `mcp_tool_call_timeout_seconds`,
   default 10 -- matches the Python pipecat-side tool HTTP client's
   existing 10s timeout for consistency) and the SSRF re-check from §7
   (using the `net.IP`-classifier approach, not a hand-rolled CIDR list).
5. On success: `fillSuccess` with the JSON-RPC result's `content` field
   (MCP tool results are typically `[{type:"text", text:"..."}]`; join text
   blocks into the `Message` field, same shape every other tool already
   returns).
6. On failure (timeout, non-2xx, JSON-RPC error object, malformed
   response): `fillFailed` with a generic message ("MCP tool call failed")
   -- do NOT forward the customer's MCP server's raw error text to the LLM
   verbatim without size-capping it (a misbehaving remote server should not
   be able to inject arbitrarily large content into the conversation).

**Dispatch integration (revised after Round 1 review, finding C1):** the
original draft's pseudocode collapsed the "not mcp_-prefixed, genuinely
unknown" branch into a comment, which -- if implemented literally --
would fall through to `tmpMessageContent = fn(ctx, c, tool)` with `fn` a
nil map value (since `!exists` means `fn == nil`), a nil-func-value call
that panics. Verified against the ACTUAL current code
(`pkg/aicallhandler/tool.go:135-150`):

```go
default:
    fn, exists := mapFunctions[tool.Function.Name]
    if !exists {
        if strings.HasPrefix(string(tool.Function.Name), "mcp_") {
            tmpMessageContent = h.toolHandleMcpCall(ctx, c, tool)
        } else {
            log.Debugf("unknown tool call: %s", tool.Function.Name)
            errMsg := fmt.Sprintf("unknown tool call: %s", tool.Function.Name)
            failContent := newToolResult(tool.ID)
            fillFailed(failContent, stderrors.New(errMsg))
            if _, errRecord := h.toolCreateResultMessage(ctx, c, tool, failContent, toolCallActiveAIID, rowOrigin); errRecord != nil {
                log.WithError(errRecord).Error("could not record the failure result message for the unknown tool call")
            }
            return nil, stderrors.New(errMsg)
        }
    } else {
        tmpMessageContent = fn(ctx, c, tool)
    }
```

This preserves the EXISTING unknown-tool-call `return nil, err` path
byte-for-byte (it must still exit `ToolHandle` early, exactly as today --
that return is load-bearing per the existing code's own comment about
unpaired tool-call messages, VOIP-1460) and adds the `mcp_` branch as a
genuinely parallel `else if`, not a fallthrough.

## 10. API surface (new endpoints, bin-ai-manager + OpenAPI)

Admin/manager top-level resource, per the monorepo's API convention
(`/service_agents/*` is for agent-facing reduced scope; this is a
customer-admin resource, same tier as `/ais`):

- `POST   /mcpservers` -- create (name, detail, url, auth_type, api_key_header, secret)
- `GET    /mcpservers` -- list (customer-scoped, paginated like every other List)
- `GET    /mcpservers/:id` -- get
- `PUT    /mcpservers/:id` -- update. **Wire semantics (Round 1 finding
  MN2):** the request DTO's `secret` field must be `*string` (pointer),
  not `string`, because Go's `encoding/json` cannot otherwise distinguish
  "field absent from the JSON body" (leave existing encrypted secret
  untouched) from "field present with an empty string" (explicitly clear
  the secret). `nil` pointer = untouched; non-nil pointing at `""` =
  clear; non-nil pointing at a value = re-encrypt and replace.
- `DELETE /mcpservers/:id` -- soft delete (tm_delete), same pattern as `AI`

`bin-openapi-manager` gets a new `openapi/paths/mcpservers/` spec section;
`bin-api-manager` is a thin RPC passthrough to bin-ai-manager's
`listenhandler`, following the existing `/ais` wiring exactly (no new
pattern).

## 11. Observability

- `promMcpToolListTotal{result="success|failure"}` -- ListTools calls.
- `promMcpToolCallTotal{result="success|failure"}` -- CallTool calls.
- `promMcpToolCallDurationSeconds` (histogram) -- external round-trip time,
  the metric that will tell us in Phase 2 whether MCP-in-the-realtime-loop
  is even viable latency-wise.
- No per-server-id or per-customer-id label on the histogram/counters
  beyond what's noted in §9.1 (cardinality guard); per-customer debugging
  goes through Loki log lines (`mcp_server_id` field), not Prometheus
  labels.

## 12. Testing strategy

- `mcpserverhandler`: standard CRUD unit tests (gomock reqHandler/db),
  encryption round-trip test (encrypt then decrypt matches plaintext),
  SSRF validator table test (private IPs, non-https, metadata IP all
  rejected; public https accepted, DNS-rebinding re-check exercised via a
  fake resolver).
- `mcpToolHandler`: ListTools/CallTool against a `httptest.Server` mock MCP
  endpoint -- success, timeout, non-2xx, malformed JSON-RPC, JSON-RPC error
  object.
- `aicallHandler.ToolHandle`: table test extending the existing tool
  dispatch tests with an `mcp_*`-prefixed function name, asserting it
  routes to `toolHandleMcpCall`, AND a regression test asserting a
  genuinely unknown (non-`mcp_`-prefixed) tool name still hits the
  existing early-`return nil, err` failure path unchanged (locks in the
  §9.2 dispatch fix so it cannot silently regress back to the Round-1
  nil-func-call bug).
- `aihandler.ValidateMcpServerIDs`: table test -- valid same-customer
  reference accepts, cross-customer/nonexistent reference rejects.
- Integration-level smoke (manual, documented in PR description, not
  automated in Phase 1): one real text AIcall session against a throwaway
  local MCP test server, confirming the LLM can discover and call a custom
  tool end to end.

## 13. Rollout / config flags

New `bin-ai-manager` config (Cobra+Viper, 3-edit pattern: flag registration
+ bindings map + `LoadGlobalConfig` struct field):

| Flag | Env | Default | Purpose |
|---|---|---|---|
| `mcp_secret_encryption_keys` | `MCP_SECRET_ENCRYPTION_KEYS` | (required if any McpServer row exists) | comma-separated `<version>:<base64-32-byte-key>` pairs; the highest version is used for new writes, all listed versions remain available for decrypting existing rows (§6 key rotation) |
| `mcp_tools_list_cache_ttl_seconds` | `MCP_TOOLS_LIST_CACHE_TTL_SECONDS` | 60 | Redis cache TTL for a server's `tools/list` result |
| `mcp_tool_call_timeout_seconds` | `MCP_TOOL_CALL_TIMEOUT_SECONDS` | 10 | Bounded HTTP timeout for `tools/call` |

No feature flag gating the feature on/off globally -- the whitelist opt-in
(`mcp:<id>` in `ToolNames`) is itself the gate; a customer who never
registers an `McpServer` sees zero behavior change.

## 14. Open questions for review

1. **Resolved (Round 3):** the exact call site for `resolveTools`, and
   its interaction with the codebase's EXISTING long-idle AIcall reuse
   path, must be pinned down as part of this design, not left open --
   see §9.1's new "AIcall reuse and Metadata staleness" subsection below.
2. Should `McpServer.Status=disabled` be exposed as a customer-facing
   toggle (PATCH-style partial update) separate from full soft-delete? Included
   in the model (§5) but the API surface (§10) currently only lists full
   CRUD; a dedicated enable/disable endpoint may be worth adding in the
   same PR if trivial, otherwise Phase 2.
3. Confirm whether `bin-openapi-manager`'s existing `/ais` OpenAPI spec
   section is the right structural template to copy for `/mcpservers`, or
   whether the newer `/aiaudits` section (more recently added) is now the
   canonical template.

## 15. Section numbering note

Section numbers 1-14 and 16-17 are intentional; there is no separate §15
(the Phase 2 out-of-scope content that would have been §15 was folded
into §3 "Non-goals" during the initial draft and the numbering was never
backfilled). Noted here rather than silently renumbering, since §16/§17
are cross-referenced by number from the review log and elsewhere.

## 16. Implementation completeness notes (added after Round 1 review)

Filled in per the review's "completeness gaps" finding, so the plan doc
that follows this design does not have to rediscover these from scratch:

- **`dbhandler` layer**: `pkg/dbhandler` (existing package, e.g.
  `dbhandler/ai.go`) gains standard `McpServerCreate`/`McpServerGet`/
  `McpServerList`/`McpServerUpdate`/`McpServerDelete` functions, table-
  driven CRUD over `ai_mcp_servers`, mirroring `dbhandler/ai.go`'s shape.
- **Cache invariant**: unlike `ai_ais` (which `bin-ai-manager`'s CLAUDE.md
  requires refreshing via `aiUpdateToCache` after every write, because
  `AI` is read on the hot per-AIcall-start path from cache), `McpServer`
  is read-through-DB with NO cache in Phase 1. This is a deliberate,
  stated decision, not an omission: `McpServer` rows are read once per
  session (§9.1's `ListTools`, which already has its own short Redis TTL
  cache for the tools/list RESULT, not the row) plus once per
  `toolHandleMcpCall` invocation (§9.2 step 3) -- call volume is bounded
  by conversation turn count, not by the AIcall-start hot path `AI` itself
  sits on, so a second cache layer over the `McpServer` row itself is not
  justified in Phase 1. Revisit only if profiling shows it matters.
- **Migration location**: the `CREATE TABLE ai_mcp_servers` statement in
  §5 is illustrative of the target shape; the actual migration MUST be
  authored via `bin-dbscheme-manager`'s Alembic workflow
  (`alembic -c alembic.ini revision -m "..."`), per the repo-wide CLAUDE.md
  mandatory rule. No raw SQL file is committed outside that path.
- **`mcpToolHandler` interface**: new package `bin-ai-manager/pkg/mcptoolhandler`,
  mirroring the existing `toolhandler`/`teamhandler` shape --
  `//go:generate mockgen` + interface + concrete struct + `NewMcpToolHandler`
  constructor taking an HTTP client and the decrypt helper as
  constructor args (so it is mockable in `aicallHandler` tests the same
  way `reqHandler`/`messageHandler` already are).
- **`chatbot.go` callers unaffected**: `pkg/aihandler/chatbot.go:52,155`
  (the two call sites of `ai.ValidateToolNames`) are NOT touched by this
  design -- confirmed by the §5 revision, since `ValidateMcpServerIDs` is a
  new, separate function wired into `aihandler.Create`/`Update`
  independently of the existing `ValidateToolNames` call sites.

## 17. Review log

- **Round 1** (2026-09-11, independent subagent, adversarial review):
  VERDICT: CHANGES_REQUESTED. Findings: C1 (dispatch pseudocode would
  panic on genuinely unknown tools), C2/M1 (`mcp:` entries inside the
  closed-enum `ToolNames` field), M2 (Metadata read-modify-write
  discipline), M3 (SSRF re-check missing from `ListTools`), MN1 (EngineKey
  divergence unexplained), MN2 (PUT wire semantics for optional secret).
  All CRITICAL/MAJOR findings addressed by revising §5 (new
  `AI.McpServerIDs` field replacing `mcp:` prefix overloading), §9.1
  (SSRF re-check + Metadata merge discipline), §9.2 (corrected dispatch
  code verified against the live `tool.go`), §6 (EngineKey divergence
  callout), §10 (PUT pointer-field semantics), plus new §16 covering the
  completeness gaps (dbhandler, cache invariant, migration location, mock
  interface, chatbot.go non-impact).
- **Round 2** (2026-09-11, independent subagent, fresh adversarial
  re-review): independently re-verified all Round 1 fixes against live
  code (confirmed genuine, not just asserted) -- C1/C2/M1/M2 all
  correctly closed. Surfaced 3 NEW findings: encryption key rotation
  story was entirely absent (rotating the key would permanently brick
  every stored secret); the `Status=disabled` exclusion claimed in §5's
  doc comment was never wired into the §9.1/§9.2 runtime steps (the
  `McpServerIDs` membership re-check in §9.2 step 2 is NOT the same
  check as "server still active," so a mid-session-disabled server's
  tool would still succeed); the §7 SSRF range list as originally
  written invited a hand-rolled CIDR check vulnerable to the classic
  IPv4-mapped-IPv6 (`::ffff:169.254.169.254`) blocklist bypass.
  VERDICT: CHANGES_REQUESTED. Addressed by: §6 key-versioning scheme
  (`KeyVersion` column + `MCP_SECRET_ENCRYPTION_KEYS` multi-key config,
  §13), explicit `Status == active` enforcement added to both §9.1 (skip
  non-active servers before ListTools) and §9.2 step 3 (fail closed on
  a disabled server even if still whitelist-listed), and §7's guard
  requirement rewritten to mandate `net.IP`'s built-in classifier
  methods (`IsPrivate`/`IsLoopback`/`IsLinkLocalUnicast`/
  `IsLinkLocalMulticast`) instead of a bespoke CIDR-list implementation.
- **Round 3** (2026-09-11, independent subagent, fresh adversarial
  re-review): independently re-verified all Round 2 fixes against live
  doc text and code (confirmed genuine) -- KeyVersion/rotation scheme
  coherent, `Status == active` gates present and effective in both
  §9.1 and §9.2, `AI.McpServerIDs` confirmed to genuinely leave
  `ToolNames`/`chatbot.go` untouched. Surfaced 2 NEW MAJOR findings plus
  1 MINOR: (a) the §7 SSRF guard validated a resolved IP via a
  pre-flight lookup but never required the ACTUAL dial to be pinned to
  that validated address, leaving a multi-A-record / TTL=0
  DNS-rebinding gap even within a single call-time window; (b) the
  codebase's PRE-EXISTING long-idle AIcall reuse path
  (`AIcallConversationIdleTimeoutHours`, `refreshInsightSessionIfIdle`)
  was never connected to the design's Metadata tool-name mapping --
  §14's original Open Question 1 (exact `resolveTools` call site) was
  the right question but left unresolved, and a reused AIcall could
  carry a stale tool list/mapping indefinitely; (c) the tool-name
  hex-prefix collision was asserted as impossible when it is merely
  negligible (32-bit truncation of a 128-bit UUID). VERDICT:
  CHANGES_REQUESTED. Addressed by: §7's dial-time IP-pinning requirement
  (custom `net.Dialer.Control`/`DialContext` hook validating the actual
  socket peer, not a detached pre-flight lookup, checked against EVERY
  resolved IP), §9.1's new "AIcall reuse and Metadata staleness"
  subsection resolving Open Question 1 (re-run `resolveTools` on the
  same idle-refresh trigger `refreshInsightSessionIfIdle` already uses,
  single merged Metadata write), and §9.1 step 3's collision-risk
  wording softened to "negligible in practice" with the failure mode
  (same-customer cross-server tool misroute, not a security-boundary
  violation) stated explicitly instead of asserted away.
- **Round 4** (2026-09-11, independent subagent, fresh adversarial
  re-review, explicitly asked to assess convergence): independently
  re-verified Round 3's SSRF dial-pinning fix as technically sound
  (`net.Dialer.Control` correctly leaves SNI/cert validation intact) --
  no new SSRF issue. BUT found the Round 3 "AIcall reuse" fix was only
  HALF correct: it wired `resolveTools` into
  `refreshInsightSessionIfIdle`'s trigger, which is reachable ONLY from
  the `startReferenceTypeContactCase` path -- the doc's own claim that
  plain `conversation`-referenced AIcalls have "no reopen boundary" was
  independently verified FALSE against live code
  (`start.go:288-366`'s `startReferenceTypeConversation` has its own,
  separate reuse branch gated by the SAME idle-timeout config, and that
  branch touches zero Metadata). This was flagged as a materially worse
  signal than a plain omission (a stated resolution containing an
  incorrect factual claim about the codebase) -- explicit convergence
  assessment: NOT YET CONVERGING as of Round 3's headline fix.
  VERDICT: CHANGES_REQUESTED. Addressed by: §9.1's "AIcall reuse and
  Metadata staleness" subsection rewritten to name and fix BOTH
  independent reuse paths explicitly (`startAIcallByMessaging` for
  true creation of either type, `startReferenceTypeConversation`'s own
  reuse branch as a new explicit step, and `startReferenceTypeContactCase`'s
  reuse branch retaining the original fix) rather than depending on one
  reference-type-specific refresh function to cover every case; plus a
  one-paragraph TLS/SNI implementation note added to §7 clarifying that
  a `DialContext`-based alternative (if chosen over the recommended
  `net.Dialer.Control`) must explicitly preserve the original hostname
  as `tls.Config.ServerName`.
- **Round 5** (2026-09-11, independent subagent, explicitly instructed
  not to manufacture findings just to keep the loop going): independently
  re-verified the Round 4 reuse-path fix as complete and correct --
  confirmed `startAIcallByMessaging` is genuinely the shared creation
  helper for all non-realtime paths (including two callers,
  `startReferenceTypeNone`/`StartTask`, the doc's prose hadn't named,
  though the fix covers them by construction since `resolveTools` lives
  inside the shared helper) and confirmed no other reuse path exists
  unaddressed. Independently confirmed §7's SSRF/TLS fix, §6's secret
  encryption, and §9.2's dispatch fix all still hold. Found ONE genuine,
  previously-unflagged MAJOR: §2 Decision 1's stated Phase-1 scoping
  gate, `AIcall.PipecatcallID == uuid.Nil`, is factually FALSE against
  live code -- `PipecatcallID` is set to a freshly generated, non-nil
  UUID for EVERY AIcall regardless of reference type or messaging-vs-
  realtime path (`CreateByMessaging` at `db.go:146` and
  `startAIcallByRealtime`/`Create` at `start.go:1102,1108-1109` both set
  it), so a literal implementation of this stated gate would select zero
  rows -- a landmine specifically for a future Phase 2 reader relying on
  this document's own stated rule. Also noted one cosmetic NIT (section
  numbering skips §15). VERDICT: CHANGES_REQUESTED. Addressed by:
  correcting §2's stated gate to `AIcall.ConfbridgeID == uuid.Nil` (the
  field that actually distinguishes the messaging path, which hardcodes
  `ConfbridgeID: uuid.Nil`, from the realtime path, which always passes a
  real confbridge id) with the verified code citations inline, and adding
  §15 as an explicit numbering note rather than silently renumbering
  (§16/§17 are cross-referenced by number elsewhere in this log). No
  functional/security change resulted -- §9.1's actual implementation
  was never driven by this field at runtime, only the stated rationale
  needed correcting.
- **Round 6** (2026-09-11, independent subagent, explicit convergence
  check): first APPROVED verdict. Independently re-verified the §2
  `ConfbridgeID` gate is stable across BOTH creation and every reuse
  path -- confirmed no AIcall field-updater (`UpdatePipecatcallIDAndActiveflowID`,
  `UpdateCurrentMemberID`, or any other) ever touches `ConfbridgeID`
  post-creation, closing the one residual question Round 5 left
  unchecked. Fresh full read of §1-17 found no cross-round contradiction
  and no new finding. VERDICT: APPROVED.
- **Round 7** (2026-09-11, independent subagent, second consecutive
  review needed to close the loop per the 2-in-a-row rule): independently
  re-verified §2/§9.2/§9.1's most safety-critical claims against live
  code from scratch (not deferring to Round 6's account) -- all held.
  Confirmed zero residual code artifacts from this not-yet-implemented
  design (`grep` for `resolveTools`/`McpServerIDs`/`mcptoolhandler`
  returns zero hits, as expected for a design doc). Raised one MINOR,
  explicitly framed as a named-for-the-record acknowledgment rather than
  a blocking gap: Phase 1's deferred rate/concurrency governance (§3)
  bounds a single call but not aggregate outbound volume toward one
  customer's MCP server, worth stating explicitly for the sign-off
  record. Also two NITs (pre-existing debug-log argument exposure parity
  with the `EngineKey` precedent; PII/consent framing parity with
  VoIPBin's existing customer-configured webhook delivery, architecturally
  identical risk shape, no new consent model needed). VERDICT: APPROVED.
  **Two consecutive APPROVED verdicts (Rounds 6-7) -- review loop closed.**
  §3's non-goal now carries the Round 7 residual-risk acknowledgment
  verbatim.
