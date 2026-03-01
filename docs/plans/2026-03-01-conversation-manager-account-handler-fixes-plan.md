# Conversation-Manager Account Handler Fixes — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix 9 issues in the conversation-manager account handler: credential exposure, inconsistent events, error codes, missing teardown, ExecContext, debug logs, dead code, WebhookMessage bypass, and OpenAPI annotation.

**Architecture:** All changes are in `bin-conversation-manager/` except one OpenAPI annotation in `bin-openapi-manager/`. The `accountHandler` interface is unchanged — only internals, the listenhandler boundary, and the `lineHandler`/`DBHandler` interfaces change.

**Tech Stack:** Go, gomock, squirrel SQL builder, LINE Bot SDK v7, OpenAPI 3.0

**Worktree:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/`

**All paths below are relative to:** `bin-conversation-manager/`

---

### Task 1: Strip credentials from WebhookMessage (Fix 1)

**Files:**
- Modify: `models/account/webhook.go`

**Step 1: Edit WebhookMessage struct — remove Secret and Token fields**

In `models/account/webhook.go`, remove lines 19-20 (`Secret` and `Token` fields) from the `WebhookMessage` struct:

```go
// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	Type Type `json:"type,omitempty"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}
```

**Step 2: Edit ConvertWebhookMessage — remove Secret and Token assignments**

In the same file, update `ConvertWebhookMessage()` to remove lines 37-38:

```go
func (h *Account) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Type: h.Type,

		Name:   h.Name,
		Detail: h.Detail,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}
```

**Step 3: Remove unused `"encoding/json"` import if needed**

Check if `encoding/json` is still needed — yes, it's used by `CreateWebhookEvent()`. Keep it.

**Step 4: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-conversation-manager && go build ./...`
Expected: Compiles successfully.

---

### Task 2: Remove unused DBHandler.AccountSet (Fix 7)

Do this early because it changes the DBHandler interface, and `go generate` must run before later tasks depend on updated mocks.

**Files:**
- Modify: `pkg/dbhandler/main.go` (remove from interface)
- Modify: `pkg/dbhandler/account.go` (remove implementation)
- Modify: `pkg/dbhandler/account_test.go` (remove Test_AccountSet)

**Step 1: Remove AccountSet from DBHandler interface**

In `pkg/dbhandler/main.go`, delete line 28:
```
	AccountSet(ctx context.Context, id uuid.UUID, name string, detail string, secret string, token string) error
```

The interface should now read:
```go
type DBHandler interface {
	AccountCreate(ctx context.Context, ac *account.Account) error
	AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error)
	AccountUpdate(ctx context.Context, id uuid.UUID, fields map[account.Field]any) error
	AccountList(context.Context, uint64, string, map[account.Field]any) ([]*account.Account, error)
	AccountDelete(ctx context.Context, id uuid.UUID) error

	ConversationCreate(ctx context.Context, cv *conversation.Conversation) error
	// ... rest unchanged
```

**Step 2: Remove AccountSet implementation**

In `pkg/dbhandler/account.go`, delete lines 203-213 (the entire `AccountSet` function):

```go
// DELETE THIS ENTIRE FUNCTION:
// AccountSet returns sets the account info
func (h *handler) AccountSet(ctx context.Context, id uuid.UUID, name string, detail string, secret string, token string) error {
	...
}
```

**Step 3: Remove Test_AccountSet from tests**

In `pkg/dbhandler/account_test.go`, delete lines 122-208 (the entire `Test_AccountSet` function).

**Step 4: Fix ExecContext in AccountUpdate (Fix 5)**

While we're in `pkg/dbhandler/account.go`, fix line 238. Change:
```go
	if _, err := h.db.Exec(sqlStr, args...); err != nil {
```
To:
```go
	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
```

**Step 5: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-conversation-manager && go build ./...`
Expected: Compiles successfully.

---

### Task 3: Add Teardown to LineHandler interface and implementation (Fix 4 — part 1)

**Files:**
- Modify: `pkg/linehandler/main.go` (add to interface)
- Create: `pkg/linehandler/teardown.go` (implementation)

**Step 1: Add Teardown to LineHandler interface**

In `pkg/linehandler/main.go`, add after line 18 (`Setup` method):
```go
	Teardown(ctx context.Context, ac *account.Account) error
```

Full interface:
```go
type LineHandler interface {
	Setup(ctx context.Context, ac *account.Account) error
	Teardown(ctx context.Context, ac *account.Account) error
	Send(ctx context.Context, cv *conversation.Conversation, ac *account.Account, text string, medias []media.Media) error
	Hook(ctx context.Context, ac *account.Account, data []byte) error

	GetPeer(ctx context.Context, ac *account.Account, userID string) (*commonaddress.Address, error)
}
```

**Step 2: Create teardown.go**

Create new file `pkg/linehandler/teardown.go`:

```go
package linehandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/account"
)

// Teardown removes the LINE webhook for the given account
func (h *lineHandler) Teardown(ctx context.Context, ac *account.Account) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Teardown",
		"account_id": ac.ID,
	})

	c, err := h.getClient(ctx, ac)
	if err != nil {
		log.Errorf("Could not get client. err: %v", err)
		return err
	}

	_, err = c.SetWebhookEndpointURL("").WithContext(ctx).Do()
	if err != nil {
		log.Errorf("Could not remove webhook uri. err: %v", err)
		return err
	}

	return nil
}
```

**Step 3: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-conversation-manager && go build ./...`
Expected: Compiles successfully.

---

### Task 4: Regenerate mocks

Both `DBHandler` (removed `AccountSet`) and `LineHandler` (added `Teardown`) interfaces changed.

**Step 1: Regenerate all mocks**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-conversation-manager && go generate ./...`
Expected: Mocks regenerated successfully. Files updated:
- `pkg/dbhandler/mock_main.go` (no more `AccountSet` mock)
- `pkg/linehandler/mock_linehandler.go` (new `Teardown` mock)

**Step 2: Verify it compiles and tests pass so far**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-conversation-manager && go build ./... && go test ./...`
Expected: Build OK. Tests FAIL because:
- `Test_Create` expects `PublishEvent` but code still calls `PublishEvent` (not yet changed)
- That's OK — we'll fix tests in later tasks

Actually, wait — we haven't changed the event publishing yet. Current tests should still pass because the code still calls `PublishEvent` for Create and `PublishEvent` for Delete. Let's verify.

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-conversation-manager && go test ./...`
Expected: All tests PASS (no behavioral changes yet, only interface + dead code removal).

---

### Task 5: Add teardown dispatch + restructure Delete (Fix 4 — part 2, Fix 2, Fix 6)

This is the main logic task. Changes to `pkg/accounthandler/`.

**Files:**
- Modify: `pkg/accounthandler/setup.go` (add `teardown()` method)
- Modify: `pkg/accounthandler/db.go` (restructure Delete, fix event publishing, add debug logs)

**Step 1: Add teardown() to setup.go**

In `pkg/accounthandler/setup.go`, add after the `setup()` function:

```go
// teardown tears down the account (best-effort, logs warning on failure)
func (h *accountHandler) teardown(ctx context.Context, ac *account.Account) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "teardown",
		"account_id": ac.ID,
	})

	var err error
	switch ac.Type {
	case account.TypeLine:
		err = h.lineHandler.Teardown(ctx, ac)

	case account.TypeSMS:
		// nothing to do

	default:
		// unknown type, nothing to tear down
	}
	if err != nil {
		log.Warnf("Could not teardown the account. err: %v", err)
	}
}
```

Note: `teardown` returns nothing — it's best-effort. Failures are logged as warnings and do not block deletion.

**Step 2: Fix Create — change PublishEvent to PublishWebhookEvent (Fix 2)**

In `pkg/accounthandler/db.go`, change line 61 from:
```go
	h.notifyHandler.PublishEvent(ctx, account.EventTypeAccountCreated, res)
```
To:
```go
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, account.EventTypeAccountCreated, res)
```

**Step 3: Add debug log to Get (Fix 6)**

In `pkg/accounthandler/db.go`, in the `Get` function, add after line 77 (before `return res, nil`):
```go
	log.WithField("account", res).Debugf("Retrieved account info. account_id: %s", id)
```

**Step 4: Add debug log to List (Fix 6)**

In `pkg/accounthandler/db.go`, in the `List` function, add after line 93 (before `return res, nil`):
```go
	log.WithField("accounts", res).Debugf("Retrieved account list. count: %d", len(res))
```

**Step 5: Restructure Delete — add teardown + fix event publishing (Fix 4 + Fix 2)**

Replace the entire `Delete` function in `pkg/accounthandler/db.go` with:

```go
// Delete deletes the account and return the deleted account
func (h *accountHandler) Delete(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Delete",
		"account_id": id,
	})

	// get the account first for teardown
	ac, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get account info for teardown. err: %v", err)
		return nil, errors.Wrap(err, "could not get account info")
	}

	// teardown external resources (best-effort)
	h.teardown(ctx, ac)

	if errDelete := h.db.AccountDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete account info. err: %v", errDelete)
		return nil, errors.Wrap(errDelete, "could not delete account info")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted account info")
		return nil, errors.Wrap(err, "could not get deleted account info")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, account.EventTypeAccountDeleted, res)

	return res, nil
}
```

**Step 6: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-conversation-manager && go build ./...`
Expected: Compiles successfully.

---

### Task 6: Update accounthandler tests

**Files:**
- Modify: `pkg/accounthandler/db_test.go`
- Modify: `pkg/accounthandler/setup_test.go`

**Step 1: Update Test_Create — expect PublishWebhookEvent instead of PublishEvent**

In `pkg/accounthandler/db_test.go`, replace line 93:
```go
			mockNotify.EXPECT().PublishEvent(ctx, account.EventTypeAccountCreated, tt.responseAccount)
```
With:
```go
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAccount.CustomerID, account.EventTypeAccountCreated, tt.responseAccount)
```

**Step 2: Update Test_Delete — add lineHandler mock, extra Get call, teardown, and expect PublishWebhookEvent**

Replace the entire `Test_Delete` function with:

```go
func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseAccount *account.Account
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("74a879e6-fe49-11ed-98e7-576bc17c7b79"),

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("74a879e6-fe49-11ed-98e7-576bc17c7b79"),
				},
				Type: account.TypeLine,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)

			h := &accountHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}
			ctx := context.Background()

			// Get for teardown
			mockDB.EXPECT().AccountGet(ctx, tt.id).Return(tt.responseAccount, nil)
			// Teardown (LINE type)
			mockLine.EXPECT().Teardown(ctx, tt.responseAccount).Return(nil)
			// DB delete
			mockDB.EXPECT().AccountDelete(ctx, tt.id).Return(nil)
			// Get after delete
			mockDB.EXPECT().AccountGet(ctx, tt.id).Return(tt.responseAccount, nil)
			// Publish
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAccount.CustomerID, account.EventTypeAccountDeleted, tt.responseAccount)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAccount) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAccount, res)
			}

		})
	}
}
```

**Step 3: Add teardown tests to setup_test.go**

In `pkg/accounthandler/setup_test.go`, add after the `Test_setup` function:

```go
func Test_teardown(t *testing.T) {

	tests := []struct {
		name string

		account *account.Account
	}{
		{
			name: "type is line",

			account: &account.Account{
				Type: account.TypeLine,
			},
		},
		{
			name: "type is sms",

			account: &account.Account{
				Type: account.TypeSMS,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)

			h := accountHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}
			ctx := context.Background()

			switch tt.account.Type {
			case account.TypeLine:
				mockLine.EXPECT().Teardown(ctx, tt.account).Return(nil)
			}

			h.teardown(ctx, tt.account)
		})
	}
}
```

**Step 4: Run accounthandler tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-conversation-manager && go test -v ./pkg/accounthandler/...`
Expected: All tests PASS.

---

### Task 7: Fix listenhandler — WebhookMessage conversion + 404 (Fix 3, Fix 8)

**Files:**
- Modify: `pkg/listenhandler/v1_accounts.go`

**Step 1: Fix processV1AccountsGet — convert list to WebhookMessage**

Replace lines 57-58:
```go
	data, err := json.Marshal(tmps)
```
With:
```go
	webhookMessages := make([]*account.WebhookMessage, len(tmps))
	for i, t := range tmps {
		webhookMessages[i] = t.ConvertWebhookMessage()
	}

	data, err := json.Marshal(webhookMessages)
```

**Step 2: Fix processV1AccountsPost — convert to WebhookMessage**

Replace line 92:
```go
	data, err := json.Marshal(tmp)
```
With:
```go
	data, err := json.Marshal(tmp.ConvertWebhookMessage())
```

**Step 3: Fix processV1AccountsIDGet — convert to WebhookMessage + 404**

Replace line 125:
```go
		return simpleResponse(500), nil
```
With:
```go
		return simpleResponse(404), nil
```

Replace line 128:
```go
	data, err := json.Marshal(tmp)
```
With:
```go
	data, err := json.Marshal(tmp.ConvertWebhookMessage())
```

**Step 4: Fix processV1AccountsIDPut — convert to WebhookMessage + 404**

Replace line 185:
```go
		return simpleResponse(500), nil
```
With:
```go
		return simpleResponse(404), nil
```

Replace line 188:
```go
	data, err := json.Marshal(tmp)
```
With:
```go
	data, err := json.Marshal(tmp.ConvertWebhookMessage())
```

**Step 5: Fix processV1AccountsIDDelete — convert to WebhookMessage + 404**

Replace line 222:
```go
		return simpleResponse(500), nil
```
With:
```go
		return simpleResponse(404), nil
```

Replace line 225:
```go
	data, err := json.Marshal(tmp)
```
With:
```go
	data, err := json.Marshal(tmp.ConvertWebhookMessage())
```

**Step 6: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-conversation-manager && go build ./...`
Expected: Compiles successfully.

---

### Task 8: Update listenhandler tests

**Files:**
- Modify: `pkg/listenhandler/v1_accounts_test.go`

The response JSON data has changed because `WebhookMessage` no longer includes `secret` and `token`. Since the test accounts don't have secret/token set (they're empty strings), and `omitempty` skips empty strings, the response JSON is actually the same. But we should verify.

**Current test response data pattern:** `{"id":"...","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`

This pattern has no `secret` or `token` because they're empty strings with `omitempty`. The WebhookMessage conversion won't change this. So the existing response assertions should still pass.

**Step 1: Run listenhandler tests to verify**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-conversation-manager && go test -v ./pkg/listenhandler/...`
Expected: All tests PASS (response JSON unchanged because test data has empty credentials).

---

### Task 9: OpenAPI writeOnly annotation (Fix 9)

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Add writeOnly to secret field**

In `bin-openapi-manager/openapi/openapi.yaml`, find the `ConversationManagerAccount` schema's `secret` property (around line 2855) and add `writeOnly: true`:

```yaml
        secret:
          type: string
          writeOnly: true
          description: Webhook secret for signature verification. Write-only.
          example: "whsec_...redacted..."
```

**Step 2: Add writeOnly to token field**

Similarly for the `token` property:

```yaml
        token:
          type: string
          writeOnly: true
          description: API token for the messaging platform. Write-only.
          example: "xoxb_...redacted..."
```

**Step 3: Regenerate OpenAPI types**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-openapi-manager && go generate ./...`
Expected: `gens/models/gen.go` regenerated. `writeOnly` does not change generated Go types.

---

### Task 10: Full verification workflow

**Step 1: Run full verification for bin-conversation-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-conversation-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: All steps pass.

**Step 2: Run verification for bin-openapi-manager**

Run:
```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler/bin-openapi-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: All steps pass.

---

### Task 11: Commit

**Step 1: Stage all changed files**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler
git add \
  docs/plans/ \
  bin-conversation-manager/models/account/webhook.go \
  bin-conversation-manager/pkg/accounthandler/db.go \
  bin-conversation-manager/pkg/accounthandler/db_test.go \
  bin-conversation-manager/pkg/accounthandler/setup.go \
  bin-conversation-manager/pkg/accounthandler/setup_test.go \
  bin-conversation-manager/pkg/dbhandler/main.go \
  bin-conversation-manager/pkg/dbhandler/account.go \
  bin-conversation-manager/pkg/dbhandler/account_test.go \
  bin-conversation-manager/pkg/dbhandler/mock_main.go \
  bin-conversation-manager/pkg/linehandler/main.go \
  bin-conversation-manager/pkg/linehandler/teardown.go \
  bin-conversation-manager/pkg/linehandler/mock_linehandler.go \
  bin-conversation-manager/pkg/listenhandler/v1_accounts.go \
  bin-openapi-manager/openapi/openapi.yaml \
  bin-openapi-manager/gens/models/gen.go
```

Also add any vendor changes:
```bash
git add bin-conversation-manager/vendor/ bin-openapi-manager/vendor/
```

**Step 2: Commit**

```bash
git commit -m "NOJIRA-fix-conversation-manager-account-handler

Fix 9 issues in conversation-manager account handler identified during code review.

- bin-conversation-manager: Strip Secret/Token from WebhookMessage (security fix)
- bin-conversation-manager: Use PublishWebhookEvent for all account CRUD events (consistency)
- bin-conversation-manager: Return 404 instead of 500 for account GET/PUT/DELETE errors
- bin-conversation-manager: Add LINE webhook teardown on account delete
- bin-conversation-manager: Fix ExecContext in AccountUpdate for context cancellation
- bin-conversation-manager: Add debug logs after account Get and List
- bin-conversation-manager: Remove unused DBHandler.AccountSet interface method
- bin-conversation-manager: Return WebhookMessage from listenhandler instead of raw Account
- bin-openapi-manager: Add writeOnly annotation to account secret/token fields"
```

**Step 3: Verify clean state**

Run: `git status`
Expected: Clean working tree.

---

### Task 12: Push and create PR

**Step 1: Push branch**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-fix-conversation-manager-account-handler
git push -u origin NOJIRA-fix-conversation-manager-account-handler
```

**Step 2: Create PR**

```bash
gh pr create \
  --title "NOJIRA-fix-conversation-manager-account-handler" \
  --body "Fix 9 issues in conversation-manager account handler identified during code review.

- bin-conversation-manager: Strip Secret/Token from WebhookMessage (security fix — corrects existing OpenAPI 'Write-only' spec violation)
- bin-conversation-manager: Use PublishWebhookEvent for all account CRUD events (Create/Delete were missing customer webhooks)
- bin-conversation-manager: Return 404 instead of 500 for account GET/PUT/DELETE errors (matches monorepo convention)
- bin-conversation-manager: Add LINE webhook teardown on account delete (removes dangling webhook URL from LINE API)
- bin-conversation-manager: Fix ExecContext in AccountUpdate for proper context cancellation
- bin-conversation-manager: Add debug logs after account Get and List (CLAUDE.md requirement)
- bin-conversation-manager: Remove unused DBHandler.AccountSet interface method
- bin-conversation-manager: Return WebhookMessage from listenhandler instead of raw Account struct
- bin-openapi-manager: Add writeOnly annotation to account secret/token fields"
```
