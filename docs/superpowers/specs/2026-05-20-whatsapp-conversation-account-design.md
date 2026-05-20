# WhatsApp Conversation Account — Design Spec

**Date:** 2026-05-20  
**Branch:** NOJIRA-Add-whatsapp-conversation-account  
**Scope:** Add `whatsapp` as a new conversation account type using Meta's WhatsApp Business Cloud API. Text messages only (no media, no templates).

---

## 1. Context

`bin-conversation-manager` currently supports two account types:

| Type | Provider | Notes |
|------|----------|-------|
| `line` | LINE Messaging API | HMAC webhook verification, sends via channel access token |
| `sms` | bin-message-manager | Delegates to Telnyx/MessageBird |

Each provider has its own handler package (`linehandler`, `smshandler`) implementing a uniform interface. Adding WhatsApp follows the same extension pattern, with three important differences from LINE:

1. **Inbound conversation lifecycle**: WhatsApp has no opt-in/"follow" event. The first inbound text message IS the first contact. The provider handler creates the conversation on first message (matching how `linehandler.hookEventTypeFollow` creates conversations), rather than waiting for a separate event.
2. **Webhook signature**: Meta's HMAC-SHA256 signature (`X-Hub-Signature-256`) must be forwarded from `bin-hook-manager` to conversation-manager alongside the raw body.
3. **Webhook challenge**: Meta sends a GET request to verify the endpoint. The existing pipeline must support returning the challenge string back to the HTTP caller.

---

## 2. Credential Model

### 2.1 Existing account fields (unchanged)

| Field | WhatsApp usage |
|-------|---------------|
| `token` | System user access token (Bearer token for Cloud API calls) |
| `secret` | Webhook verify token (echoed back during Meta's hub challenge) |

### 2.2 New `provider_data` JSON column

A new `provider_data JSON` column is added to `conversation_accounts`. It holds provider-specific extras that do not generalize across account types.

For WhatsApp, `provider_data` contains:

```json
{
  "phone_number_id": "123456789012345",
  "app_secret": "..."
}
```

| Key | Purpose |
|-----|---------|
| `phone_number_id` | Required path parameter for every Cloud API send call |
| `app_secret` | Required for HMAC-SHA256 webhook signature verification — **absence causes fail-closed rejection of inbound webhooks** |

LINE and SMS accounts store JSON `null` in `provider_data` (see §4 for NULL semantics note).

A Go struct provides typed access:

```go
type WhatsAppProviderData struct {
    PhoneNumberID string `json:"phone_number_id"`
    AppSecret     string `json:"app_secret"`
}
```

**Security:** `provider_data` MUST be stripped from `account.WebhookMessage` (§8.2). It contains `app_secret` and must not appear in API responses or webhooks.

---

## 3. Model Changes

### 3.1 `models/account/account.go`

```go
TypeWhatsApp Type = "whatsapp"

ProviderData json.RawMessage `json:"provider_data,omitempty" db:"provider_data,json"`

FieldProviderData Field = "provider_data"

type WhatsAppProviderData struct {
    PhoneNumberID string `json:"phone_number_id"`
    AppSecret     string `json:"app_secret"`
}
```

### 3.2 `models/conversation/conversation.go`

```go
TypeWhatsApp Type = "whatsapp"
```

### 3.3 `models/message/message.go`

```go
ReferenceTypeWhatsApp ReferenceType = "whatsapp"
```

### 3.4 `bin-common-handler/models/address/address.go`

```go
TypeWhatsApp Type = "whatsapp"
```

This is a shared-library change. Full verification across all services importing `bin-common-handler` is required.

### 3.5 `bin-conversation-manager/internal/convtitle/build.go`

Add `TypeWhatsApp` to the `channelLabel` switch:

```go
case conversation.TypeWhatsApp:
    return "WhatsApp"
```

Also add `commonaddress.TypeWhatsApp` to `humanReadableTarget` returning `true` (WhatsApp `wa_id` is a human-readable phone number):

```go
case commonaddress.TypeWhatsApp:
    return true
```

---

## 4. Database Migration

Alembic migration in `bin-dbscheme-manager` (generated via `alembic revision -m "..."`, never hand-authored):

```sql
-- upgrade
ALTER TABLE conversation_accounts
    ADD COLUMN provider_data JSON COMMENT 'Provider-specific credentials (JSON). WhatsApp: phone_number_id, app_secret.';

-- downgrade
ALTER TABLE conversation_accounts
    DROP COLUMN provider_data;
```

**NULL semantics note:** The `,json` mapper in `bin-common-handler/pkg/databasehandler/mapping.go` marshals a nil `json.RawMessage` as the JSON literal `null` (4 bytes), not SQL NULL. LINE and SMS accounts store JSON `null`. Do not use `WHERE provider_data IS NULL` in queries.

Migration is applied before any code deployment that reads `provider_data`.

---

## 5. New Package: `whatsapphandler`

Located at `bin-conversation-manager/pkg/whatsapphandler/`.  
Mock generated via `//go:generate mockgen -package whatsapphandler -destination ./mock_whatsapphandler.go -source main.go -build_flags=-mod=mod`.

Struct `whatsappHandler` contains `reqHandler requesthandler.RequestHandler`, injected at construction (matching `linehandler`).

### 5.1 Interface

```go
type WhatsAppHandler interface {
    Setup(ctx context.Context, ac *account.Account) error
    Teardown(ctx context.Context, ac *account.Account) error
    Send(ctx context.Context, cv *conversation.Conversation, ac *account.Account, text string) (wamid string, err error)
    Hook(ctx context.Context, ac *account.Account, rawData []byte, signature string) ([]*HookResult, error)
    VerifyWebhook(ctx context.Context, ac *account.Account, mode string, verifyToken string, challenge string) (string, error)
}

type HookResult struct {
    Conversation *conversation.Conversation
    Message      *message.Message
}
```

`Send` returns the `wamid` (WhatsApp message ID) for storage in `Message.TransactionID`.

### 5.2 Setup

Validates that `provider_data` is parseable and `phone_number_id` is non-empty. No remote call is made (programmatic webhook registration is out of scope — see §11). Returns error if `phone_number_id` is missing.

### 5.3 Teardown

No-op. Meta does not provide a programmatic webhook deregistration endpoint.

### 5.4 Send

Cloud API version pinned as a package constant: `const graphAPIVersion = "v19.0"` — verify the current stable version against Meta's API changelog before implementation.

```
POST https://graph.facebook.com/{graphAPIVersion}/{phone_number_id}/messages
Authorization: Bearer {ac.Token}
Content-Type: application/json

{
  "messaging_product": "whatsapp",
  "to": "{cv.DialogID}",
  "type": "text",
  "text": { "body": "{text}" }
}
```

`cv.DialogID` is the peer's `wa_id` (the E.164-normalized phone number set at conversation creation — see §5.5).

Parse the response: extract `messages[0].id` and return it as `wamid`. Return error on non-2xx response.

### 5.5 Hook (inbound message processing)

**Signature verification (fail-closed, mirrors Paddle in `bin-hook-manager/pkg/servicehandler/billing.go`):**

1. Parse `provider_data` to extract `app_secret`.
2. If `app_secret` is empty, **return error** — do not process (fail-closed).
3. Compute `HMAC-SHA256(key=app_secret, data=rawData)`, compare against the `sha256=...` value in `signature`. Return error on mismatch.

**Payload processing:**

1. Parse `entry[].changes[].value` from the JSON payload.
2. For each `messages[]` entry where `type == "text"`:
   a. Extract `wa_id` (= `from` field, the sender's phone number used as `DialogID`) and `id` (= `wamid`).
   b. Extract `contacts[].profile.name` for the peer display name.
   c. Extract `metadata.display_phone_number` for the self address target.
   d. **Deduplicate:** call `reqHandler.ConversationV1MessageList` filtered by `FieldTransactionID=wamid`. If any result exists, skip this message.
   e. **Look up or create conversation:** filter by `FieldType=whatsapp, FieldDialogID=wa_id` via `reqHandler.ConversationV1ConversationList`. If none found, create in two steps:
      1. Call `reqHandler.ConversationV1ConversationCreate` (signature has no `AccountID` param):
         ```
         CustomerID: ac.CustomerID
         Type:       conversation.TypeWhatsApp
         DialogID:   wa_id
         Self: commonaddress.Address{
             Type:   commonaddress.TypeWhatsApp,
             Target: metadata.display_phone_number,
         }
         Peer: commonaddress.Address{
             Type:       commonaddress.TypeWhatsApp,
             Target:     wa_id,
             TargetName: contacts[].profile.name,
         }
         Name, Detail: convtitle.Build(conversation.TypeWhatsApp, peer)
         ```
      2. Immediately call `reqHandler.ConversationV1ConversationUpdate(ctx, cv.ID, {conversation.FieldAccountID: ac.ID})` to set `AccountID`. This is a two-RPC sequence (non-atomic). A crash between the two leaves a conversation with `AccountID=uuid.Nil`, which will cause outbound sends to fail until corrected. This matches the current codebase's support for `FieldAccountID` in the update path.
      Use the conversation returned by the update call for subsequent steps.
   f. Create inbound `Message` record via `reqHandler.ConversationV1MessageCreate`:
      ```
      Direction:     message.DirectionIncoming
      Status:        message.StatusDone
      ReferenceType: message.ReferenceTypeWhatsApp
      TransactionID: wamid
      Text:          messages[].text.body
      ```
   g. Append `HookResult{Conversation: cv, Message: msg}`.
3. Continue past per-message errors (log and continue, matching `linehandler` pattern).
4. Return all successfully processed results. Caller always returns 200 to Meta (Meta retries on non-200).

### 5.6 VerifyWebhook (GET hub challenge)

```go
func (h *whatsappHandler) VerifyWebhook(ctx, ac, mode, verifyToken, challenge string) (string, error)
```

1. Confirm `mode == "subscribe"`.
2. Compare `verifyToken` against `ac.Secret`.
3. If matched, return `challenge`.
4. If not matched, return error (caller responds HTTP 403).

---

## 6. Webhook Routing Changes

### 6.1 `bin-hook-manager/models/hook/hook.go` — extend model

```go
type Hook struct {
    ReceviedURI       string `json:"received_uri"`        // existing (typo preserved)
    ReceivedData      []byte `json:"received_data"`       // existing
    ReceivedMethod    string `json:"received_method"`     // new: "GET" or "POST"
    ReceivedSignature string `json:"received_signature"`  // new: X-Hub-Signature-256 header value
}
```

**Impact:** `hmhook.Hook` is a shared model. Verify all consumers after this change (additive fields, backward-compatible at JSON level).

### 6.2 `bin-hook-manager/pkg/servicehandler/conversation.go` — extend forwarder

The `ServiceHandler` interface (`pkg/servicehandler/main.go`) changes `Conversation` from `(ctx, r) error` to `(ctx, r) (string, error)`. Regenerate `mock_servicehandler.go` and update `conversation_test.go` call sites after this change.

```go
func (h *serviceHandler) Conversation(ctx context.Context, r *http.Request) (string, error) {
    data, _ := io.ReadAll(r.Body)

    req := &hmhook.Hook{
        ReceviedURI:       r.Host + r.URL.RequestURI(), // was r.URL.Path — now includes query string
        ReceivedData:      data,
        ReceivedMethod:    r.Method,
        ReceivedSignature: r.Header.Get("X-Hub-Signature-256"),
    }

    if r.Method == http.MethodGet {
        challenge, err := h.reqHandler.ConversationV1HookGet(ctx, req)
        if err != nil {
            return "", err
        }
        return challenge, nil
    }

    // POST: existing fire-and-forget path (existing LINE POST hooks unaffected by
    // the added query string in ReceviedURI, since account-ID parsing uses URL path only)
    return "", h.reqHandler.ConversationV1Hook(ctx, req)
}
```

### 6.3 `bin-hook-manager/api/v1.0/conversation/conversation.go` — method branching

The route is already registered as `conversation.Any("*any", conversationPOST)` — `.Any()` already handles GET, no routing change needed. The existing `conversationPOST` handler body is updated to branch on method:

```go
func conversationPOST(c *gin.Context) {
    ctx := context.Background()
    serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

    challenge, err := serviceHandler.Conversation(ctx, c.Request)
    if err != nil {
        c.AbortWithStatus(500)
        return
    }

    if c.Request.Method == http.MethodGet && challenge != "" {
        c.String(200, challenge)
        return
    }
    c.AbortWithStatus(200)
}
```

### 6.4 `bin-hook-manager/pkg/requesthandler` — new `ConversationV1HookGet`

```go
// ConversationV1HookGet sends a GET hook and returns the challenge string from the response body.
func (r *requestHandler) ConversationV1HookGet(ctx context.Context, hm *hmhook.Hook) (string, error) {
    // marshals hm, sends RPC via sendRequestConversation with GET method
    // parses response and returns Data as string (the challenge)
}
```

The existing `ConversationV1Hook` (POST, fire-and-forget) is unchanged.

### 6.5 `bin-conversation-manager/pkg/listenhandler/v1_hooks.go` — GET route

Add `processV1HooksGet` alongside the existing `processV1HooksPost`. The route table in `listenhandler/main.go` adds:

```go
case regV1Hooks.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
    return h.processV1HooksGet(ctx, m)
```

`processV1HooksGet` implementation:

```go
func (h *listenHandler) processV1HooksGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
    var req request.V1DataHooksPost // reuses same model — ReceviedURI contains the full URI with query string
    if err := json.Unmarshal(m.Data, &req); err != nil {
        return simpleResponse(400), nil
    }

    // hub.* params live in req.ReceviedURI query string (the forwarded external URL),
    // NOT in m.URI (which is the internal RPC path "/v1/hooks")
    u, err := url.Parse(req.ReceviedURI)
    if err != nil {
        return simpleResponse(400), nil
    }
    q := u.Query()
    mode      := q.Get("hub.mode")
    token     := q.Get("hub.verify_token")
    challenge := q.Get("hub.challenge")

    result, err := h.conversationHandler.HookVerify(ctx, req.ReceviedURI, mode, token, challenge)
    if err != nil {
        return simpleResponse(403), nil
    }

    return &sock.Response{
        StatusCode: 200,
        DataType:   "text/plain",
        Data:       []byte(result),
    }, nil
}
```

### 6.6 `conversationhandler` — `HookVerify` and updated `Hook` interface

Add `HookVerify` to `ConversationHandler`:

```go
HookVerify(ctx context.Context, uri string, mode string, verifyToken string, challenge string) (string, error)
```

Implementation: parse `account_id` from `uri` path → load account → confirm `type == whatsapp` → delegate to `h.whatsappHandler.VerifyWebhook(ctx, ac, mode, verifyToken, challenge)`. The naming split (`ConversationHandler.HookVerify` → `WhatsAppHandler.VerifyWebhook`) is intentional: `HookVerify` is the handler-layer method; `VerifyWebhook` is the provider-layer method.

**`Hook` interface signature change:**

```go
// before
Hook(ctx context.Context, uri string, data []byte) error

// after
Hook(ctx context.Context, uri string, method string, signature string, data []byte) error
```

`processV1HooksPost` passes `req.ReceivedMethod` and `req.ReceivedSignature`. `hookLine` receives `signature` but ignores it (LINE uses its own HMAC verification internally via the LINE SDK).

Both the `Hook` signature change and the new `HookVerify` method require regenerating `mock_conversationhandler.go` (via `go generate ./...`).

---

## 7. Wiring Changes

### 7.1 `accounthandler/setup.go`

```go
case account.TypeWhatsApp:
    err = h.whatsappHandler.Setup(ctx, ac)
// teardown: no-op for whatsapp
```

### 7.2 `messagehandler/send.go`

```go
case conversation.TypeWhatsApp:
    return h.sendWhatsApp(ctx, cv, text, medias)
```

`sendWhatsApp` implementation:

1. Load account via `h.accountHandler.Get`.
2. Create message record (direction=outgoing, status=progressing).
3. Call `h.whatsappHandler.Send(ctx, cv, ac, text)` — returns `(wamid, error)`.
4. On error: update status to failed, return error.
5. On success: call `h.UpdateStatus` to done; update `TransactionID = wamid` via a separate update call (or combine if `UpdateStatus` accepts additional fields).

### 7.3 `conversationhandler/hook.go`

```go
case account.TypeWhatsApp:
    if errHook := h.hookWhatsApp(ctx, ac, data, signature); errHook != nil { ... }
```

`hookWhatsApp` iterates `whatsappHandler.Hook` results and runs execute-mode dispatch (agent/flow) — same pattern as `hookLine`. Conversation and message creation happens **inside `whatsappHandler.Hook`** (§5.5), so `hookWhatsApp` only receives fully populated `HookResult` pairs.

### 7.4 Dependency injection

`whatsappHandler` is injected into `accountHandler`, `conversationHandler`, and `messageHandler`. The service entrypoint (`cmd/conversation-manager/main.go`) instantiates `whatsapphandler.NewWhatsAppHandler(reqHandler)`.

---

## 8. API Changes

### 8.1 Request model (`listenhandler/models/request/v1_accounts.go`)

```go
type V1DataAccountsPost struct {
    ...
    ProviderData json.RawMessage `json:"provider_data"`
}

type V1DataAccountsIDPut struct {
    ...
    ProviderData json.RawMessage `json:"provider_data"`
}
```

**`V1DataHooksPost` requires no change.** The existing type in `listenhandler/models/request/v1_hook.go` embeds `hmhook.Hook`:
```go
type V1DataHooksPost struct {
    hmhook.Hook
}
```
The `ReceivedMethod` and `ReceivedSignature` fields added to `hmhook.Hook` in §6.1 are automatically promoted — `processV1HooksPost` and `processV1HooksGet` access them via `req.ReceivedMethod` and `req.ReceivedSignature` with no change to this file.

### 8.2 `models/account/webhook.go`

`ConvertWebhookMessage` must **not** include `ProviderData`. It already strips `Secret` and `Token`; `ProviderData` follows the same pattern — add to the internal `Account` struct only, not to `WebhookMessage`.

### 8.3 OpenAPI spec

- Add `"whatsapp"` to `account.type` enum.
- Add `provider_data` (type: object, nullable, **write-only** — never returned) to Account create/update schema.
- Document `WhatsAppProviderData`: `phone_number_id` (string, required), `app_secret` (string, required for inbound hooks).

### 8.4 RST docs (`bin-api-manager/docsdev/source/`)

- Update conversation account overview: add `whatsapp` type, describe `provider_data` as a write-only field.
- RST struct docs reflect `WebhookMessage` only — `provider_data`, `secret`, `token` do not appear in response struct docs.
- Document WhatsApp setup steps: Meta app creation, phone number registration, webhook URL (`https://hook.voipbin.net/v1.0/conversation/accounts/{account_id}`), verify token.
- Rebuild: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`, then `git add -f bin-api-manager/docsdev/build/`.

---

## 9. Testing

### 9.1 `whatsapphandler` unit tests

| Test | Cases |
|------|-------|
| `Hook` | valid signature + text message; wrong signature (rejected); missing app_secret (fail-closed); duplicate wamid (skipped); batch of multiple messages; non-text message type (skipped) |
| `VerifyWebhook` | correct verify_token → returns challenge; wrong token → error; wrong mode → error |
| `Send` | success → wamid returned; non-2xx API response → error |
| `Setup` | valid provider_data; missing phone_number_id → error |

### 9.2 Integration touch-points

- `accounthandler/setup_test.go`: WhatsApp Setup/Teardown cases.
- `conversationhandler/hook_test.go`: WhatsApp hook routing, HookVerify routing, updated `Hook` interface signature.
- `messagehandler/send_test.go`: WhatsApp send case.
- `listenhandler`: GET `/v1/hooks` routing test; `processV1HooksGet` with correct/wrong verify_token.
- `bin-hook-manager/pkg/servicehandler/conversation_test.go`: GET forwarding, signature capture, URI query-string preservation.

**Mock regeneration required after interface changes:**
- `bin-conversation-manager/pkg/conversationhandler/mock_conversationhandler.go` — `Hook` signature change + new `HookVerify` method.
- `bin-hook-manager/pkg/servicehandler/mock_servicehandler.go` — `Conversation` return type change.
- `bin-conversation-manager/pkg/whatsapphandler/mock_whatsapphandler.go` — new package, generated fresh.

Run `go generate ./...` in each affected service directory after updating the interfaces.

### 9.3 Manual verification

Using Meta's webhook test tool in Meta Business Manager after deployment.

---

## 10. Rollout Order

1. Deploy Alembic migration (`provider_data` column — backward-compatible add).
2. Deploy updated `bin-common-handler` (`TypeWhatsApp` address type).
3. Deploy updated `bin-hook-manager` (GET forwarding, signature forwarding, query-string in URI).
4. Deploy updated `bin-conversation-manager` (new account type, handler, routes, interface changes).
5. Create a WhatsApp account via API: `POST /v1/accounts` with `type=whatsapp`, `token`, `secret`, `provider_data={"phone_number_id":"...","app_secret":"..."}`.
6. Configure the webhook URL in Meta Business Manager to `https://hook.voipbin.net/v1.0/conversation/accounts/{account_id}`.
7. Trigger Meta's webhook verification — confirm `hub.challenge` is echoed back correctly.
8. Send a test inbound WhatsApp message — confirm conversation and message records are created.
9. Send a test outbound message — confirm delivery via Cloud API and `wamid` stored in `TransactionID`.

---

## 11. Out of Scope

- Media messages (images, documents, audio, video)
- WhatsApp template messages (business-initiated, outside 24h window)
- Delivery status webhooks (read receipts, delivered/read status)
- Multiple WhatsApp phone numbers per account
- Automated WABA subscription registration (`subscribed_apps`)
- Outbound phone number E.164 normalization beyond using the inbound `wa_id` verbatim
