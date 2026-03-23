# Add MessageFlowID to Conversation Account

## Problem

When a LINE message is received, the conversation-manager creates a message record but does not trigger an activeflow. Unlike SMS — where the phone number has a `MessageFlowID` that triggers flow execution — LINE accounts have no equivalent field. This means LINE messages cannot be routed to AI/Pipecat for processing.

## Approach

Add a `MessageFlowID` field to the conversation `Account` model. When a LINE message arrives, `conversationhandler.hookLine()` checks the account's `MessageFlowID` and creates/executes an activeflow if set — reusing the same `MessageExecuteActiveflow` logic the SMS path uses.

We chose a single `MessageFlowID` (not `CallFlowID` + `MessageFlowID`) because LINE accounts only receive messages — there's no call scenario.

## Changes

### 1. Account Model (`bin-conversation-manager/models/account/account.go`)

- Add `MessageFlowID uuid.UUID` field with `json:"message_flow_id,omitempty" db:"message_flow_id,uuid"` tags
- Add `FieldMessageFlowID Field = "message_flow_id"` constant

### 2. Account WebhookMessage (`bin-conversation-manager/models/account/webhook.go`)

- Add `MessageFlowID uuid.UUID` to `WebhookMessage` struct
- Map it in `ConvertWebhookMessage()`

### 3. Database Migration (`bin-dbscheme-manager`)

```sql
-- upgrade
ALTER TABLE conversation_accounts ADD COLUMN message_flow_id BINARY(16) NOT NULL DEFAULT (UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''))) AFTER token;

-- downgrade
ALTER TABLE conversation_accounts DROP COLUMN message_flow_id;
```

`NOT NULL` with nil UUID default ensures `ScanRow` with `,uuid` tag never encounters NULL. Existing rows get nil UUID (no flow triggered).

### 3a. Test DB Schema (`bin-conversation-manager/scripts/database_scripts_test/table_conversation_accounts.sql`)

Add `message_flow_id` column to the test schema. Without this, all `Test_Account*` DB integration tests will fail because `PrepareFields`/`GetDBFields`/`ScanRow` reference the column.

```sql
message_flow_id binary(16) NOT NULL DEFAULT (UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''))),
```

### 4. Refactor `MessageExecuteActiveflow` (`bin-conversation-manager/pkg/conversationhandler/message.go`)

Change signature from:
```go
func (h *conversationHandler) MessageExecuteActiveflow(ctx context.Context, cv *conversation.Conversation, m *message.Message, num *nmnumber.Number) (*fmactiveflow.Activeflow, error)
```
to:
```go
func (h *conversationHandler) MessageExecuteActiveflow(ctx context.Context, cv *conversation.Conversation, m *message.Message, messageFlowID uuid.UUID) (*fmactiveflow.Activeflow, error)
```

The SMS caller passes `num.MessageFlowID` instead of `num`.

### 5. LINE Hook Handler Return Value (`bin-conversation-manager/pkg/linehandler/`)

Introduce a `HookResult` struct and change `Hook()` to return a slice:

```go
type HookResult struct {
    Conversation *conversation.Conversation
    Message      *message.Message
}

func Hook(ctx, ac, data) ([]*HookResult, error)
```

- `hookEventHandle()` returns `(*HookResult, error)`
- `hookEventTypeMessage()` returns `&HookResult{cv, m}, nil`
- `hookEventTypeFollow()` returns `nil, nil` (no result, no error)
- `Hook()` collects all non-nil results into a slice, preserving current behavior of processing ALL events

Update `LineHandler` interface accordingly.

### 6. `hookLine()` in `conversationhandler` (`bin-conversation-manager/pkg/conversationhandler/hook.go`)

After calling `lineHandler.Hook()`, iterate results and trigger activeflow for each:
```go
results, err := h.lineHandler.Hook(ctx, ac, data)
if err != nil {
    return err
}

if ac.MessageFlowID == uuid.Nil {
    return nil
}

for _, r := range results {
    if r.Conversation == nil || r.Message == nil {
        continue
    }

    af, err := h.MessageExecuteActiveflow(ctx, r.Conversation, r.Message, ac.MessageFlowID)
    if err != nil {
        return errors.Wrapf(err, "Could not execute the activeflow")
    }
    log.WithField("activeflow", af).Debugf("Executed activeflow. activeflow_id: %s", af.ID)
}
```

### 7. Account Create API (`bin-conversation-manager/pkg/listenhandler/`)

- Add `MessageFlowID uuid.UUID` to `V1DataAccountsPost` request struct in `pkg/listenhandler/models/request/v1_accounts.go`
- Update `accountHandler.Create()` signature and implementation to accept `messageFlowID uuid.UUID`
- Update `processV1AccountsPost` caller to pass `req.MessageFlowID`

### 8. Account Update API (`bin-conversation-manager/pkg/listenhandler/v1_accounts.go`)

Add `string(account.FieldMessageFlowID)` to the `allowedItems` list in `processV1AccountsIDPut()` so that `message_flow_id` in PUT requests is not silently dropped.

### 9. bin-common-handler RequestHandler (`bin-common-handler/pkg/requesthandler/`)

The shared `RequestHandler` interface has `ConversationV1AccountCreate` which builds `V1DataAccountsPost` and sends the RPC. Both the interface (in `main.go`) and implementation (in `conversation_accounts.go`) need the new `messageFlowID uuid.UUID` parameter.

- Add `messageFlowID uuid.UUID` param to `ConversationV1AccountCreate` in `RequestHandler` interface (`main.go`)
- Add `messageFlowID uuid.UUID` param to the implementation and set `MessageFlowID: messageFlowID` in the `V1DataAccountsPost` struct literal (`conversation_accounts.go`)
- Regenerate mock: `go generate ./pkg/requesthandler/...`

**Note:** `bin-common-handler` is used by 30+ services. Changing the `RequestHandler` interface requires regenerating mocks for any service that mocks `ConversationV1AccountCreate`. Currently only `bin-api-manager` and `bin-conversation-manager` call this function. Other services import `RequestHandler` but don't call this specific method, so their mocks won't break — only callers need updating.

### 10. bin-api-manager servicehandler (`bin-api-manager/pkg/servicehandler/conversation_account.go`)

`ConversationAccountCreate` is the API gateway handler that accepts the HTTP request and calls `reqHandler.ConversationV1AccountCreate`. It needs to:

- Add `messageFlowID uuid.UUID` param to `ConversationAccountCreate` method signature (both interface in `main.go` and implementation)
- Pass it through to `h.reqHandler.ConversationV1AccountCreate(..., messageFlowID)`

### 10a. bin-api-manager server layer (`bin-api-manager/server/conversation_accounts.go`)

`PostConversationAccounts` is a **hand-written** handler (not auto-generated) that parses the OpenAPI-generated request body and calls `serviceHandler.ConversationAccountCreate`. After OpenAPI regeneration adds `MessageFlowId` to `PostConversationAccountsJSONBody`, this handler must be **manually updated** to:

- Parse `req.MessageFlowId` (string from generated type) and convert to `uuid.UUID`
- Pass it to `h.serviceHandler.ConversationAccountCreate(..., messageFlowID)`

### 11. conversation-control CLI (`bin-conversation-manager/cmd/conversation-control/main.go`)

`runAccountCreate` calls `accountHandler.Create()` directly. It needs:

- Add `--message-flow-id` flag (string, parsed to `uuid.UUID`)
- Pass the parsed UUID to `accountHandler.Create(..., messageFlowID)`

### 12. OpenAPI Schema (`bin-openapi-manager/openapi/openapi.yaml`)

Add `message_flow_id` (type: string, format: uuid) to:
- `ConversationManagerAccount` response schema
- POST `/v1/accounts` request body
- PUT `/v1/accounts/{id}` request body

### 13. Regenerate

- `bin-openapi-manager`: `go generate ./...`
- `bin-api-manager`: `go generate ./...`
- `bin-common-handler`: `go generate ./pkg/requesthandler/...`

## Services Affected

| Service | What changes |
|---------|-------------|
| `bin-conversation-manager` | Account model, webhook, linehandler, conversationhandler, listenhandler, accounthandler, CLI tool |
| `bin-common-handler` | RequestHandler interface + implementation for ConversationV1AccountCreate |
| `bin-dbscheme-manager` | Alembic migration |
| `bin-openapi-manager` | OpenAPI schema |
| `bin-api-manager` | Regenerated server code + servicehandler ConversationAccountCreate |

## Trade-offs

- **Single `MessageFlowID` vs dual flow IDs**: Chose single since LINE has no call scenario. Can add `CallFlowID` later if needed (YAGNI).
- **Refactoring `MessageExecuteActiveflow`**: Breaking the `*nmnumber.Number` dependency makes the function reusable for any message source, at the cost of changing the SMS caller too.
- **`Hook()` return slice**: Returning `[]*HookResult` from `Hook()` preserves current behavior of processing all events in a webhook payload, while keeping activeflow logic centralized in `conversationhandler`.
