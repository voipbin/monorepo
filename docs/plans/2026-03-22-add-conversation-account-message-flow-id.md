# Add MessageFlowID to Conversation Account — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable LINE message conversations to trigger activeflows (and thus AI/Pipecat processing) by adding a `MessageFlowID` field to the conversation `Account` model.

**Architecture:** Add `MessageFlowID` to the Account model and WebhookMessage. Refactor `MessageExecuteActiveflow` to accept a `uuid.UUID` instead of `*nmnumber.Number`. Change `linehandler.Hook()` to return a slice of `HookResult` (conversation + message pairs), so `conversationhandler.hookLine()` can iterate and trigger activeflow execution for each when `MessageFlowID` is set. Update the full Create call chain: conversation-manager accountHandler → bin-common-handler RequestHandler → bin-api-manager servicehandler + server layer. Update the CLI tool and OpenAPI schema.

**Tech Stack:** Go, MySQL (Alembic migration), OpenAPI 3.0, oapi-codegen

**Design doc:** `docs/plans/2026-03-22-add-conversation-account-message-flow-id-design.md`

---

### Task 1: Add MessageFlowID to Account model

**Files:**
- Modify: `bin-conversation-manager/models/account/account.go:1-57`

**Step 1: Add uuid import and MessageFlowID field to Account struct**

Add `"github.com/gofrs/uuid"` to imports. Add field after `Token` (line 19):

```go
MessageFlowID uuid.UUID `json:"message_flow_id,omitempty" db:"message_flow_id,uuid"`
```

**Step 2: Add FieldMessageFlowID constant**

Add to the Field constants block after `FieldToken` (line 40):

```go
FieldMessageFlowID Field = "message_flow_id"
```

**Step 3: Verify build**

Run: `cd bin-conversation-manager && go build ./...`
Expected: SUCCESS

---

### Task 2: Add MessageFlowID to Account WebhookMessage

**Files:**
- Modify: `bin-conversation-manager/models/account/webhook.go:1-50`

**Step 1: Add uuid import and MessageFlowID to WebhookMessage struct**

Add `"github.com/gofrs/uuid"` to imports. Add field after `Detail` (line 17):

```go
MessageFlowID uuid.UUID `json:"message_flow_id,omitempty"`
```

**Step 2: Map MessageFlowID in ConvertWebhookMessage**

Add after `Detail: h.Detail,` (line 32):

```go
MessageFlowID: h.MessageFlowID,
```

**Step 3: Verify build**

Run: `cd bin-conversation-manager && go build ./...`
Expected: SUCCESS

---

### Task 3: Create database migration

**Files:**
- Create: `bin-dbscheme-manager/alembic/versions/<auto>_add_message_flow_id_to_conversation_accounts.py`

**Step 1: Create Alembic migration**

Run from the worktree:
```bash
cd bin-dbscheme-manager && alembic -c alembic.ini revision -m "add_message_flow_id_to_conversation_accounts"
```

**Step 2: Edit migration file**

Add SQL to the generated file:

```python
def upgrade() -> None:
    op.execute(
        "ALTER TABLE conversation_accounts "
        "ADD COLUMN message_flow_id BINARY(16) NOT NULL "
        "DEFAULT (UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''))) "
        "AFTER token"
    )

def downgrade() -> None:
    op.execute(
        "ALTER TABLE conversation_accounts DROP COLUMN message_flow_id"
    )
```

**Step 3: Do NOT run `alembic upgrade`** — migration will be applied manually.

**Step 4: Update test DB schema**

In `bin-conversation-manager/scripts/database_scripts_test/table_conversation_accounts.sql`, add the `message_flow_id` column after `token`:

```sql
  message_flow_id binary(16) NOT NULL DEFAULT (UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''))),
```

This is required because `PrepareFields`/`GetDBFields`/`ScanRow` use the model's `db:` tags — if the column doesn't exist in the test schema, all `Test_Account*` DB integration tests will fail.

---

### Task 4: Refactor MessageExecuteActiveflow signature

**Files:**
- Modify: `bin-conversation-manager/pkg/conversationhandler/message.go:45-79` (MessageExecuteActiveflow)
- Modify: `bin-conversation-manager/pkg/conversationhandler/message.go:146` (SMS caller)

**Step 1: Change MessageExecuteActiveflow signature**

Replace the `num *nmnumber.Number` parameter with `messageFlowID uuid.UUID`:

```go
func (h *conversationHandler) MessageExecuteActiveflow(ctx context.Context, cv *conversation.Conversation, m *message.Message, messageFlowID uuid.UUID) (*fmactiveflow.Activeflow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "MessageExecuteActiveflow",
		"conversation":    cv,
		"message":         m,
		"message_flow_id": messageFlowID,
	})

	res, err := h.reqHandler.FlowV1ActiveflowCreate(
		ctx,
		uuid.Nil,
		m.CustomerID,
		messageFlowID,
		fmactiveflow.ReferenceTypeConversation,
		m.ConversationID,
		uuid.Nil,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create an activeflow. message_id: %s", m.ID)
	}
	log.WithField("activeflow", res).Debugf("Created activeflow. activeflow_id: %s", res.ID)

	if errVariable := h.setVariables(ctx, res.ID, cv, m); errVariable != nil {
		return nil, errors.Wrapf(errVariable, "Could not set the variables. activeflow_id: %s", res.ID)
	}

	if errExecute := h.reqHandler.FlowV1ActiveflowExecute(ctx, res.ID); errExecute != nil {
		return nil, errors.Wrapf(errExecute, "Could not execute the activeflow. activeflow_id: %s", res.ID)
	}

	return res, nil
}
```

**Step 2: Update SMS caller in MessageEventReceived (line 146)**

Change:
```go
af, err := h.MessageExecuteActiveflow(ctx, cv, m, num)
```
to:
```go
af, err := h.MessageExecuteActiveflow(ctx, cv, m, num.MessageFlowID)
```

**Step 3: Remove `nmnumber` import if no longer used**

Check if `nmnumber` is still used elsewhere in `message.go`. `NumberGet` at line 134 returns `*nmnumber.Number`, so keep the import.

**Step 4: Verify build**

Run: `cd bin-conversation-manager && go build ./...`
Expected: SUCCESS

---

### Task 5: Add HookResult and change linehandler.Hook() to return slice

**Files:**
- Modify: `bin-conversation-manager/pkg/linehandler/main.go:17-24` (LineHandler interface)
- Modify: `bin-conversation-manager/pkg/linehandler/hook.go:22-183` (Hook, hookEventHandle, hookEventTypeFollow, hookEventTypeMessage)

**Step 1: Add HookResult struct to main.go**

Add before the interface definition:

```go
// HookResult contains the conversation and message created from a LINE webhook event.
type HookResult struct {
	Conversation *conversation.Conversation
	Message      *message.Message
}
```

**Step 2: Update LineHandler interface**

Change `Hook` signature in the interface:
```go
Hook(ctx context.Context, ac *account.Account, data []byte) ([]*HookResult, error)
```

**Step 3: Update Hook() function**

```go
func (h *lineHandler) Hook(ctx context.Context, ac *account.Account, data []byte) ([]*HookResult, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Hook",
		"account_id": ac.ID,
		"data":       data,
	})

	tmp := &struct {
		Events []*linebot.Event `json:"events"`
	}{}

	if errUnmarshal := json.Unmarshal(data, tmp); errUnmarshal != nil {
		log.Errorf("Could not unmarshal the data. err: %v", errUnmarshal)
		return nil, errUnmarshal
	}

	results := []*HookResult{}
	for _, e := range tmp.Events {
		r, err := h.hookEventHandle(ctx, ac, e)
		if err != nil {
			log.Errorf("Could not handle the message. err: %v", err)
			continue
		}
		if r != nil {
			results = append(results, r)
		}
	}

	return results, nil
}
```

**Step 4: Update hookEventHandle()**

```go
func (h *lineHandler) hookEventHandle(ctx context.Context, ac *account.Account, e *linebot.Event) (*HookResult, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "hookEventHandle",
		"customer_id": ac,
		"event":       e,
	})

	switch e.Type {
	case linebot.EventTypeFollow:
		if errHook := h.hookEventTypeFollow(ctx, ac, e); errHook != nil {
			log.Errorf("Could not handle the line event follow. err: %v", errHook)
			return nil, errHook
		}
		return nil, nil

	case linebot.EventTypeMessage:
		r, err := h.hookEventTypeMessage(ctx, ac, e)
		if err != nil {
			log.Errorf("Could not handle the line event message. err: %v", err)
			return nil, err
		}
		return r, nil

	default:
		log.Errorf("Unsupported event type. event_type: %s", e.Type)
		return nil, fmt.Errorf("unsupported event type. event_type: %s", e.Type)
	}
}
```

**Step 5: Update hookEventTypeMessage() to return HookResult**

Change signature to `(*HookResult, error)` and update the return at the end:

```go
func (h *lineHandler) hookEventTypeMessage(ctx context.Context, ac *account.Account, e *linebot.Event) (*HookResult, error) {
```

Replace the final `return nil` with:
```go
	return &HookResult{
		Conversation: cv,
		Message:      m,
	}, nil
```

And update error returns from `return errors.Wrapf(...)` to `return nil, errors.Wrapf(...)` / `return nil, fmt.Errorf(...)`.

**Step 6: Regenerate mocks**

Run: `cd bin-conversation-manager && go generate ./pkg/linehandler/...`

**Step 7: Verify build**

Run: `cd bin-conversation-manager && go build ./...`
Expected: May fail until Task 6 updates the caller.

---

### Task 6: Update hookLine() to iterate results and trigger activeflow

**Files:**
- Modify: `bin-conversation-manager/pkg/conversationhandler/hook.go:64-78` (hookLine function)

**Step 1: Update hookLine()**

```go
func (h *conversationHandler) hookLine(ctx context.Context, ac *account.Account, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "hookLine",
		"account_id": ac.ID,
	})

	// parse messages and get results back
	results, err := h.lineHandler.Hook(ctx, ac, data)
	if err != nil {
		log.Errorf("Could not parse the message. err: %v", err)
		return err
	}

	// check if account has a message flow
	if ac.MessageFlowID == uuid.Nil {
		return nil
	}
	log.Debugf("The account has message flow id. account_id: %s, message_flow_id: %s", ac.ID, ac.MessageFlowID)

	for _, r := range results {
		if r.Conversation == nil || r.Message == nil {
			continue
		}

		af, err := h.MessageExecuteActiveflow(ctx, r.Conversation, r.Message, ac.MessageFlowID)
		if err != nil {
			return errors.Wrapf(err, "Could not execute the activeflow. account_id: %s", ac.ID)
		}
		log.WithField("activeflow", af).Debugf("Executed activeflow. activeflow_id: %s", af.ID)
	}

	return nil
}
```

**Step 2: Verify build**

Run: `cd bin-conversation-manager && go build ./...`
Expected: SUCCESS

---

### Task 7: Update account Create handler to accept MessageFlowID

**Files:**
- Modify: `bin-conversation-manager/pkg/listenhandler/models/request/v1_accounts.go:12-19` (V1DataAccountsPost)
- Modify: `bin-conversation-manager/pkg/accounthandler/main.go:22` (AccountHandler interface)
- Modify: `bin-conversation-manager/pkg/accounthandler/db.go:17-64` (Create implementation)
- Modify: `bin-conversation-manager/pkg/listenhandler/v1_accounts.go:91` (caller)

**Step 1: Add MessageFlowID to V1DataAccountsPost**

```go
type V1DataAccountsPost struct {
	CustomerID    uuid.UUID    `json:"customer_id"`
	Type          account.Type `json:"type"`
	Name          string       `json:"name"`
	Detail        string       `json:"detail"`
	Secret        string       `json:"secret"`
	Token         string       `json:"token"`
	MessageFlowID uuid.UUID   `json:"message_flow_id"`
}
```

**Step 2: Update AccountHandler interface**

Change `Create` signature:
```go
Create(ctx context.Context, customerID uuid.UUID, accountType account.Type, name string, detail string, secret string, token string, messageFlowID uuid.UUID) (*account.Account, error)
```

**Step 3: Update Create implementation**

In `accounthandler/db.go:17`, add `messageFlowID uuid.UUID` param and set it in the struct:

```go
func (h *accountHandler) Create(ctx context.Context, customerID uuid.UUID, accountType account.Type, name string, detail string, secret string, token string, messageFlowID uuid.UUID) (*account.Account, error) {
```

Add to the `account.Account` struct literal (after `Token: token,`):

```go
MessageFlowID: messageFlowID,
```

**Step 4: Update processV1AccountsPost caller**

In `v1_accounts.go:91`, add `req.MessageFlowID`:

```go
tmp, err := h.accountHandler.Create(ctx, req.CustomerID, req.Type, req.Name, req.Detail, req.Secret, req.Token, req.MessageFlowID)
```

**Step 5: Regenerate mocks**

Run: `cd bin-conversation-manager && go generate ./pkg/accounthandler/...`

**Step 6: Verify build**

Run: `cd bin-conversation-manager && go build ./...`
Expected: SUCCESS

---

### Task 8: Update account Update handler allowlist

**Files:**
- Modify: `bin-conversation-manager/pkg/listenhandler/v1_accounts.go:164-170` (allowedItems)

**Step 1: Add FieldMessageFlowID to allowedItems**

```go
allowedItems := []string{
	string(account.FieldName),
	string(account.FieldDetail),
	string(account.FieldType),
	string(account.FieldSecret),
	string(account.FieldToken),
	string(account.FieldMessageFlowID),
}
```

**Step 2: Verify build**

Run: `cd bin-conversation-manager && go build ./...`
Expected: SUCCESS

---

### Task 9: Update bin-common-handler RequestHandler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go:860` (RequestHandler interface)
- Modify: `bin-common-handler/pkg/requesthandler/conversation_accounts.go:63-91` (implementation)

**Step 1: Update RequestHandler interface**

In `main.go`, change the `ConversationV1AccountCreate` signature:
```go
ConversationV1AccountCreate(ctx context.Context, customerID uuid.UUID, accountType cvaccount.Type, name string, detail string, secret string, token string, messageFlowID uuid.UUID) (*cvaccount.Account, error)
```

**Step 2: Update implementation**

In `conversation_accounts.go`, add `messageFlowID uuid.UUID` param and set it in the struct:
```go
func (r *requestHandler) ConversationV1AccountCreate(ctx context.Context, customerID uuid.UUID, accountType cvaccount.Type, name string, detail string, secret string, token string, messageFlowID uuid.UUID) (*cvaccount.Account, error) {
	uri := "/v1/accounts"

	data := &cvrequest.V1DataAccountsPost{
		CustomerID:    customerID,
		Type:          accountType,
		Name:          name,
		Detail:        detail,
		Secret:        secret,
		Token:         token,
		MessageFlowID: messageFlowID,
	}
```

**Step 3: Regenerate mock**

Run: `cd bin-common-handler && go generate ./pkg/requesthandler/...`

**Step 4: Verify build**

Run: `cd bin-common-handler && go build ./...`
Expected: SUCCESS

---

### Task 10: Update bin-api-manager servicehandler

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go:436-444` (ServiceHandler interface)
- Modify: `bin-api-manager/pkg/servicehandler/conversation_account.go:105-134` (implementation)

**Step 1: Update ServiceHandler interface**

In `main.go`, change the `ConversationAccountCreate` signature:
```go
ConversationAccountCreate(
	ctx context.Context,
	a *amagent.Agent,
	accountType cvaccount.Type,
	name string,
	detail string,
	secret string,
	token string,
	messageFlowID uuid.UUID,
) (*cvaccount.WebhookMessage, error)
```

**Step 2: Update implementation**

In `conversation_account.go`, add `messageFlowID uuid.UUID` param and pass it through:
```go
func (h *serviceHandler) ConversationAccountCreate(
	ctx context.Context,
	a *amagent.Agent,
	accountType cvaccount.Type,
	name string,
	detail string,
	secret string,
	token string,
	messageFlowID uuid.UUID,
) (*cvaccount.WebhookMessage, error) {
```

Update the RPC call (line 126):
```go
tmp, err := h.reqHandler.ConversationV1AccountCreate(ctx, a.CustomerID, accountType, name, detail, secret, token, messageFlowID)
```

**Step 3: Regenerate mock**

Run: `cd bin-api-manager && go generate ./pkg/servicehandler/...`

---

### Task 11: Update conversation-control CLI

**Files:**
- Modify: `bin-conversation-manager/cmd/conversation-control/main.go:230-288` (cmdAccountCreate + runAccountCreate)

**Step 1: Add --message-flow-id flag**

In `cmdAccountCreate()`, add after the `token` flag:
```go
flags.String("message-flow-id", "", "Message flow ID (UUID) to trigger on incoming messages")
```

**Step 2: Parse and pass message flow ID**

In `runAccountCreate()`, after the `token` validation block, add:
```go
messageFlowIDStr := viper.GetString("message-flow-id")
var messageFlowID uuid.UUID
if messageFlowIDStr != "" {
	messageFlowID, err = uuid.FromString(messageFlowIDStr)
	if err != nil {
		return fmt.Errorf("invalid message-flow-id: %v", err)
	}
}
```

Update the `accountHandler.Create` call to pass `messageFlowID`:
```go
res, err := accountHandler.Create(
	context.Background(),
	customerID,
	account.Type(accountType),
	viper.GetString("name"),
	viper.GetString("detail"),
	secret,
	token,
	messageFlowID,
)
```

**Step 3: Verify build**

Run: `cd bin-conversation-manager && go build ./...`
Expected: SUCCESS

---

### Task 12: Update tests

**Files:**
- Modify: `bin-conversation-manager/pkg/conversationhandler/hook_test.go:105` (Test_Hook mock)
- Modify: `bin-conversation-manager/pkg/conversationhandler/hook_test.go:114-202` (Test_hookLine)
- Modify: `bin-conversation-manager/pkg/listenhandler/v1_accounts_test.go:133-237` (Test_processV1AccountsPost)
- Modify: `bin-common-handler/pkg/requesthandler/conversation_accounts_test.go:163-236` (Test_ConversationV1AccountCreate)
- Modify: `bin-api-manager/pkg/servicehandler/conversation_account_test.go:168-230` (Test_ConversationAccountCreate)
- Modify: `bin-api-manager/server/conversation_accounts_test.go:164` (PostConversationAccounts test)

**Step 1: Update Test_Hook mock expectation (line 105)**

Change:
```go
mockLine.EXPECT().Hook(ctx, tt.responseAccount, tt.data).Return(nil)
```
to:
```go
mockLine.EXPECT().Hook(ctx, tt.responseAccount, tt.data).Return([]*linehandler.HookResult{}, nil)
```

**Step 2: Update Test_hookLine mock expectation (line 195)**

Change:
```go
mockLine.EXPECT().Hook(ctx, tt.account, tt.data).Return(nil)
```
to:
```go
mockLine.EXPECT().Hook(ctx, tt.account, tt.data).Return([]*linehandler.HookResult{}, nil)
```

**Step 3: Update Test_processV1AccountsPost in bin-conversation-manager**

In `bin-conversation-manager/pkg/listenhandler/v1_accounts_test.go`:
- Add `expectMessageFlowID uuid.UUID` to test struct fields (after `expectToken`)
- Update request JSON data (lines 170, 204) to include `"message_flow_id":"<uuid>"`
- Update response JSON data (line 175, 209) to include `"message_flow_id":"..."` if non-nil
- Update mock expectation (line 227) to include `tt.expectMessageFlowID`:
  ```go
  mockAccount.EXPECT().Create(gomock.Any(), tt.expectCustomerID, tt.expectType, tt.expectName, tt.expectDetail, tt.expectSecret, tt.expectToken, tt.expectMessageFlowID).Return(tt.responseAccount, nil)
  ```

**Step 4: Update Test_ConversationV1AccountCreate in bin-common-handler**

In `bin-common-handler/pkg/requesthandler/conversation_accounts_test.go`:
- Add `messageFlowID uuid.UUID` to test struct fields (after `token`)
- Update expected JSON data (line 202) to include `"message_flow_id":"<uuid>"`
- Update function call (line 226) to pass `tt.messageFlowID`

**Step 5: Update Test_ConversationAccountCreate in bin-api-manager**

In `bin-api-manager/pkg/servicehandler/conversation_account_test.go`:
- Add `messageFlowID uuid.UUID` to test struct fields
- Update mock expectation (line 229) to include `tt.messageFlowID`
- Update function call (line 230) to pass `tt.messageFlowID`

**Step 6: Update PostConversationAccounts test in bin-api-manager**

In `bin-api-manager/server/conversation_accounts_test.go`:
- Update mock expectation (line 164) to include the `messageFlowID` parameter

**Step 7: Run tests**

Run: `cd bin-common-handler && go test ./pkg/requesthandler/...`
Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/... && go test ./pkg/listenhandler/...`
Run: `cd bin-api-manager && go test ./pkg/servicehandler/... && go test ./server/...`
Expected: PASS

---

### Task 13: Update OpenAPI schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml:2976-2977` (ConversationManagerAccount schema)
- Modify: `bin-openapi-manager/openapi/paths/conversation_accounts/main.yaml:35-51` (POST request body)
- Modify: `bin-openapi-manager/openapi/paths/conversation_accounts/id.yaml:37-46` (PUT request body)

**Step 1: Add message_flow_id to ConversationManagerAccount schema**

After the `token` property (line ~2976), add:
```yaml
        message_flow_id:
          type: string
          format: uuid
          x-go-type: string
          description: "The flow ID to execute when a message is received on this account. Returned from the `GET /flows` response."
          example: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
```

**Step 2: Add message_flow_id to POST request body**

In `bin-openapi-manager/openapi/paths/conversation_accounts/main.yaml`, add to the request properties:
```yaml
            message_flow_id:
              type: string
              format: uuid
              x-go-type: string
```

**Step 3: Add message_flow_id to PUT request body**

In `bin-openapi-manager/openapi/paths/conversation_accounts/id.yaml`, add to the update properties:
```yaml
            message_flow_id:
              type: string
              format: uuid
              x-go-type: string
```

**Step 4: Regenerate OpenAPI types**

Run: `cd bin-openapi-manager && go generate ./...`

**Step 5: Regenerate API server code**

Run: `cd bin-api-manager && go generate ./...`

---

### Task 14: Update bin-api-manager server layer

**Files:**
- Modify: `bin-api-manager/server/conversation_accounts.go:60-99` (PostConversationAccounts)

After OpenAPI regeneration (Task 13), the generated `PostConversationAccountsJSONBody` type will have a `MessageFlowId` field. Update the handler to parse and pass it.

**Step 1: Parse MessageFlowId from request and pass to servicehandler**

In `PostConversationAccounts`, update the call to pass the new field:
```go
res, err := h.serviceHandler.ConversationAccountCreate(
	c.Request.Context(),
	&a,
	cvaccount.Type(req.Type),
	req.Name,
	req.Detail,
	req.Secret,
	req.Token,
	utilhandler.ConvertStringToUUID(req.MessageFlowId),
)
```

Note: `req.MessageFlowId` is a string from the generated OpenAPI type (since the schema uses `x-go-type: string`). Use `utilhandler.ConvertStringToUUID` or `uuid.FromString` to convert. Check the existing pattern in the codebase for which conversion helper is used.

**Step 2: Verify build**

Run: `cd bin-api-manager && go build ./...`
Expected: SUCCESS

---

### Task 15: Run full verification workflow

**Step 1: Verify bin-common-handler**

```bash
cd bin-common-handler && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 2: Verify bin-conversation-manager**

```bash
cd bin-conversation-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 3: Verify bin-openapi-manager**

```bash
cd bin-openapi-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 4: Verify bin-api-manager**

```bash
cd bin-api-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: All four pass with no errors.

---

### Task 16: Commit and push

**Step 1: Stage changes**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-conversation-account-message-flow-id
git add docs/plans/ bin-conversation-manager/ bin-common-handler/ bin-dbscheme-manager/ bin-openapi-manager/ bin-api-manager/
```

**Step 2: Commit**

```bash
git commit -m "NOJIRA-Add-conversation-account-message-flow-id

Add MessageFlowID to conversation account to enable LINE messages to trigger
activeflows for AI/Pipecat processing.

- bin-conversation-manager: Add MessageFlowID field to Account model and WebhookMessage
- bin-conversation-manager: Refactor MessageExecuteActiveflow to accept uuid.UUID instead of *nmnumber.Number
- bin-conversation-manager: Add HookResult struct and change linehandler.Hook() to return slice
- bin-conversation-manager: Update hookLine() to iterate results and trigger activeflow when MessageFlowID is set
- bin-conversation-manager: Update account Create handler to accept MessageFlowID parameter
- bin-conversation-manager: Add MessageFlowID to account Update handler allowlist
- bin-conversation-manager: Add --message-flow-id flag to conversation-control CLI
- bin-common-handler: Add messageFlowID param to ConversationV1AccountCreate in RequestHandler
- bin-dbscheme-manager: Add Alembic migration for message_flow_id column on conversation_accounts
- bin-openapi-manager: Add message_flow_id to ConversationManagerAccount schema and request bodies
- bin-api-manager: Update ConversationAccountCreate in servicehandler and server layer
- bin-api-manager: Regenerate server code from updated OpenAPI spec
- docs: Add design document and implementation plan"
```

**Step 3: Push**

```bash
git push -u origin NOJIRA-Add-conversation-account-message-flow-id
```

**Step 4: Create PR**

Use `gh pr create` following the repo's PR format.
