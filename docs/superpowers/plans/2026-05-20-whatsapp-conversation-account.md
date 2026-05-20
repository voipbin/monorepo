# WhatsApp Conversation Account Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `whatsapp` as a conversation account type, enabling inbound text messages via Meta's WhatsApp Business Cloud API and outbound text replies, with full webhook signature verification and hub challenge support.

**Architecture:** Follows the existing `linehandler`/`smshandler` provider pattern — a new `whatsapphandler` package implements `Setup`, `Teardown`, `Send`, `Hook`, and `VerifyWebhook`. The existing `bin-hook-manager` webhook pipeline is extended to forward HTTP method and the `X-Hub-Signature-256` header, and to support GET requests for Meta's hub challenge handshake.

**Tech Stack:** Go 1.21+, gomock (go.uber.org/mock), Squirrel SQL builder, Meta WhatsApp Business Cloud API (graph.facebook.com v19.0), Alembic (Python), RabbitMQ RPC via sockhandler.

---

## File Map

### bin-dbscheme-manager
- **Create:** `bin-manager/main/versions/<auto_id>_conversation_accounts_add_provider_data.py` — Alembic migration adding `provider_data JSON` column

### bin-common-handler
- **Modify:** `models/address/main.go` — add `TypeWhatsApp`
- **Modify:** `pkg/requesthandler/main.go` — add `ConversationV1HookGet`, update `ConversationV1AccountCreate` signature
- **Modify:** `pkg/requesthandler/conversation_hook.go` — implement `ConversationV1HookGet`
- **Modify:** `pkg/requesthandler/conversation_accounts.go` — add `providerData json.RawMessage` param to `ConversationV1AccountCreate`
- **Regenerate:** `pkg/requesthandler/mock_main.go` (via `go generate ./...`)

### bin-hook-manager
- **Modify:** `models/hook/hook.go` — add `ReceivedMethod`, `ReceivedSignature` fields
- **Modify:** `pkg/servicehandler/main.go` — change `Conversation` return type to `(string, error)`
- **Modify:** `pkg/servicehandler/conversation.go` — rewrite to handle GET/POST, forward signature and query string
- **Modify:** `api/v1.0/conversation/conversation.go` — branch on GET/POST, write challenge body
- **Modify:** `api/v1.0/conversation/conversation_test.go` — add GET test cases
- **Modify:** `pkg/servicehandler/conversation_test.go` — update to `(string, error)` signature
- **Regenerate:** `pkg/servicehandler/mock_servicehandler.go` (via `go generate ./...`)

### bin-conversation-manager
- **Modify:** `models/account/account.go` — add `TypeWhatsApp`, `ProviderData` field, `FieldProviderData`, `WhatsAppProviderData` struct
- **Modify:** `models/conversation/conversation.go` — add `TypeWhatsApp`
- **Modify:** `models/message/message.go` — add `ReferenceTypeWhatsApp`
- **Modify:** `internal/convtitle/build.go` — add WhatsApp to `channelLabel` and `humanReadableTarget`
- **Create:** `pkg/whatsapphandler/main.go` — interface + constructor
- **Create:** `pkg/whatsapphandler/setup.go` — validate provider_data, phone_number_id
- **Create:** `pkg/whatsapphandler/teardown.go` — no-op
- **Create:** `pkg/whatsapphandler/send.go` — HTTP POST to graph.facebook.com
- **Create:** `pkg/whatsapphandler/hook.go` — HMAC verify + parse + create-on-first-message
- **Create:** `pkg/whatsapphandler/verify.go` — hub challenge verification
- **Create:** `pkg/whatsapphandler/setup_test.go`
- **Create:** `pkg/whatsapphandler/send_test.go`
- **Create:** `pkg/whatsapphandler/hook_test.go`
- **Create:** `pkg/whatsapphandler/verify_test.go`
- **Regenerate:** `pkg/whatsapphandler/mock_whatsapphandler.go` (via `go generate ./...`)
- **Modify:** `pkg/conversationhandler/main.go` — add `whatsappHandler` field, add `HookVerify` to interface, update `Hook` signature
- **Modify:** `pkg/conversationhandler/hook.go` — add `hookWhatsApp`, `HookVerify` impl, update `Hook` dispatch
- **Regenerate:** `pkg/conversationhandler/mock_conversationhandler.go`
- **Modify:** `pkg/accounthandler/main.go` — add `whatsappHandler` field + `providerData` to `Create` interface
- **Modify:** `pkg/accounthandler/db.go` — pass `ProviderData` to account struct in `Create`
- **Modify:** `pkg/accounthandler/setup.go` — add WhatsApp case
- **Regenerate:** `pkg/accounthandler/mock_accounthandler.go`
- **Modify:** `pkg/messagehandler/main.go` — add `whatsappHandler` field
- **Modify:** `pkg/messagehandler/send.go` — add `sendWhatsApp`
- **Regenerate:** `pkg/messagehandler/mock_messagehandler.go`
- **Modify:** `pkg/listenhandler/main.go` — add GET route for `regV1Hooks`
- **Modify:** `pkg/listenhandler/v1_hooks.go` — add `processV1HooksGet`, update `processV1HooksPost` to pass method+signature
- **Modify:** `pkg/listenhandler/models/request/v1_accounts.go` — add `ProviderData json.RawMessage`
- **Modify:** `pkg/listenhandler/v1_accounts.go` — pass `req.ProviderData` to `accountHandler.Create` and `FieldProviderData` in update
- **Modify:** `cmd/conversation-manager/main.go` — instantiate and wire `whatsapphandler`
- **Modify:** `models/account/webhook.go` — confirm `ProviderData` is NOT in `WebhookMessage`

### bin-api-manager (if not already done)
- **Modify:** `pkg/listenhandler` or relevant file — accept `provider_data` in Account create/update HTTP body and pass to requesthandler

### docs
- **Modify:** `bin-api-manager/docsdev/source/` — update conversation_account RST docs
- **Rebuild:** `bin-api-manager/docsdev/build/` — clean rebuild and force-add

---

## Task 1: Database Migration

**Files:**
- Create: `bin-conversation-manager/.worktrees/NOJIRA-Add-whatsapp-conversation-account/bin-dbscheme-manager/bin-manager/main/versions/<auto_id>_conversation_accounts_add_provider_data.py`

- [ ] **Step 1: Generate the migration file**

  Run from the worktree root:
  ```bash
  cd bin-dbscheme-manager/bin-manager && alembic -c alembic.ini revision -m "conversation_accounts_add_provider_data"
  ```
  Alembic will create a new file under `main/versions/` with a random revision ID like `a1b2c3d4e5f6_conversation_accounts_add_provider_data.py`.

- [ ] **Step 2: Edit the migration — add upgrade/downgrade SQL**

  Open the generated file and replace the empty `upgrade()` and `downgrade()` bodies with:

  ```python
  def upgrade():
      op.execute("""
          ALTER TABLE conversation_accounts
              ADD COLUMN provider_data JSON
              COMMENT 'Provider-specific credentials (JSON). WhatsApp: phone_number_id, app_secret.'
      """)


  def downgrade():
      op.execute("""
          ALTER TABLE conversation_accounts
              DROP COLUMN provider_data
      """)
  ```

- [ ] **Step 3: Verify the migration file looks correct**

  ```bash
  cat bin-dbscheme-manager/bin-manager/main/versions/<auto_id>_conversation_accounts_add_provider_data.py
  ```

  Confirm: `revision`, `down_revision`, `upgrade()`, and `downgrade()` are all present and correct.

- [ ] **Step 4: Commit**

  ```bash
  git add bin-dbscheme-manager/bin-manager/main/versions/<auto_id>_conversation_accounts_add_provider_data.py
  git commit -m "NOJIRA-Add-whatsapp-conversation-account

  - bin-dbscheme-manager: Add Alembic migration to add provider_data JSON column to conversation_accounts"
  ```

---

## Task 2: Shared Type Constants

**Files:**
- Modify: `bin-common-handler/models/address/main.go`
- Modify: `bin-conversation-manager/models/account/account.go`
- Modify: `bin-conversation-manager/models/conversation/conversation.go`
- Modify: `bin-conversation-manager/models/message/message.go`

- [ ] **Step 1: Add `TypeWhatsApp` to bin-common-handler address model**

  In `bin-common-handler/models/address/main.go`, add `TypeWhatsApp` to the existing const block:

  ```go
  const (
      TypeNone       Type = ""
      TypeAgent      Type = "agent"
      TypeAI         Type = "ai"
      TypeAITeam     Type = "ai_team"
      TypeConference Type = "conference"
      TypeEmail      Type = "email"
      TypeExtension  Type = "extension"
      TypeLine       Type = "line"
      TypeSIP        Type = "sip"
      TypeTel        Type = "tel"
      TypeWhatsApp   Type = "whatsapp"   // add this line
  )
  ```

- [ ] **Step 2: Add WhatsApp types to account model**

  In `bin-conversation-manager/models/account/account.go`:

  2a. Add `TypeWhatsApp` to the types const block:
  ```go
  const (
      TypeLine     Type = "line"
      TypeSMS      Type = "sms"
      TypeWhatsApp Type = "whatsapp"
  )
  ```

  2b. Add `ProviderData` field to the `Account` struct (after `MessageFlowID`):
  ```go
  ProviderData json.RawMessage `json:"provider_data,omitempty" db:"provider_data,json"`
  ```

  2c. Add the field constant to the Field const block:
  ```go
  FieldProviderData Field = "provider_data"
  ```

  2d. Add the import for `encoding/json` at the top of the file (it already imports `github.com/gofrs/uuid` — add `"encoding/json"` to the imports).

  2e. Add the `WhatsAppProviderData` struct after the `Account` struct:
  ```go
  // WhatsAppProviderData holds WhatsApp-specific credentials stored in provider_data JSON.
  type WhatsAppProviderData struct {
      PhoneNumberID string `json:"phone_number_id"`
      AppSecret     string `json:"app_secret"`
  }
  ```

- [ ] **Step 3: Add TypeWhatsApp to conversation model**

  In `bin-conversation-manager/models/conversation/conversation.go`, add to the type const block:
  ```go
  const (
      TypeNone    Type = ""
      TypeMessage Type = "message"
      TypeLine    Type = "line"
      TypeWhatsApp Type = "whatsapp"
  )
  ```

- [ ] **Step 4: Add ReferenceTypeWhatsApp to message model**

  In `bin-conversation-manager/models/message/message.go`, add to the ReferenceType const block:
  ```go
  const (
      ReferenceTypeNone    ReferenceType = ""
      ReferenceTypeMessage ReferenceType = "message"
      ReferenceTypeLine    ReferenceType = "line"
      ReferenceTypeWhatsApp ReferenceType = "whatsapp"
  )
  ```

- [ ] **Step 5: Verify bin-common-handler still builds**

  ```bash
  cd bin-common-handler && go build ./...
  ```
  Expected: no errors.

- [ ] **Step 6: Verify bin-conversation-manager models compile**

  ```bash
  cd bin-conversation-manager && go build ./models/...
  ```
  Expected: no errors.

- [ ] **Step 7: Commit**

  ```bash
  git add bin-common-handler/models/address/main.go \
          bin-conversation-manager/models/account/account.go \
          bin-conversation-manager/models/conversation/conversation.go \
          bin-conversation-manager/models/message/message.go
  git commit -m "NOJIRA-Add-whatsapp-conversation-account

  - bin-common-handler: Add TypeWhatsApp to address types
  - bin-conversation-manager: Add TypeWhatsApp to account, conversation, message models
  - bin-conversation-manager: Add WhatsAppProviderData struct and ProviderData field to Account
  - bin-conversation-manager: Add ReferenceTypeWhatsApp to message reference types"
  ```

---

## Task 3: convtitle WhatsApp Support

**Files:**
- Modify: `bin-conversation-manager/internal/convtitle/build.go`
- Test: `bin-conversation-manager/internal/convtitle/build_test.go`

- [ ] **Step 1: Write the failing tests**

  Open `bin-conversation-manager/internal/convtitle/build_test.go` and add these cases to the existing table-driven tests (or add new test functions if the existing test structure does not use tables). Find the test for `channelLabel` and `humanReadableTarget` and add:

  ```go
  // In the channelLabel test table, add:
  {
      name:  "whatsapp",
      input: conversation.TypeWhatsApp,
      want:  "WhatsApp",
  },

  // In the humanReadableTarget test table, add:
  {
      name:  "whatsapp type is human-readable",
      input: commonaddress.TypeWhatsApp,
      want:  true,
  },

  // In the Build test table, add:
  {
      name:     "whatsapp with peer name and target",
      convType: conversation.TypeWhatsApp,
      peer: commonaddress.Address{
          Type:       commonaddress.TypeWhatsApp,
          Target:     "+15551234567",
          TargetName: "Alice",
      },
      wantName:   "WhatsApp · Alice (+15551234567)",
      wantDetail: "WhatsApp conversation",
  },
  ```

- [ ] **Step 2: Run tests to verify they fail**

  ```bash
  cd bin-conversation-manager && go test ./internal/convtitle/... -v -run "TestChannelLabel|TestHumanReadable|TestBuild"
  ```
  Expected: FAIL — `TypeWhatsApp` case falls through to default, returning `"whatsapp"` not `"WhatsApp"`.

- [ ] **Step 3: Implement — add WhatsApp to channelLabel**

  In `bin-conversation-manager/internal/convtitle/build.go`, update `channelLabel`:

  ```go
  func channelLabel(t conversation.Type) string {
      switch t {
      case conversation.TypeLine:
          return "LINE"
      case conversation.TypeMessage:
          return "SMS"
      case conversation.TypeWhatsApp:
          return "WhatsApp"
      default:
          return string(t)
      }
  }
  ```

- [ ] **Step 4: Implement — add WhatsApp to humanReadableTarget**

  In the same file, update `humanReadableTarget`:

  ```go
  func humanReadableTarget(t commonaddress.Type) bool {
      switch t {
      case commonaddress.TypeTel, commonaddress.TypeEmail,
          commonaddress.TypeSIP, commonaddress.TypeExtension,
          commonaddress.TypeWhatsApp:
          return true
      default:
          return false
      }
  }
  ```

- [ ] **Step 5: Run tests to verify they pass**

  ```bash
  cd bin-conversation-manager && go test ./internal/convtitle/... -v
  ```
  Expected: PASS.

- [ ] **Step 6: Commit**

  ```bash
  git add bin-conversation-manager/internal/convtitle/build.go \
          bin-conversation-manager/internal/convtitle/build_test.go
  git commit -m "NOJIRA-Add-whatsapp-conversation-account

  - bin-conversation-manager: Add WhatsApp to convtitle channelLabel and humanReadableTarget"
  ```

---

## Task 4: bin-hook-manager — Extend Hook Model and Conversation Handler

**Files:**
- Modify: `bin-hook-manager/models/hook/hook.go`
- Modify: `bin-hook-manager/pkg/servicehandler/main.go`
- Modify: `bin-hook-manager/pkg/servicehandler/conversation.go`
- Modify: `bin-hook-manager/api/v1.0/conversation/conversation.go`
- Modify: `bin-hook-manager/api/v1.0/conversation/conversation_test.go`
- Modify: `bin-hook-manager/pkg/servicehandler/conversation_test.go`
- Regenerate: `bin-hook-manager/pkg/servicehandler/mock_servicehandler.go`

- [ ] **Step 1: Extend hmhook.Hook**

  Replace the contents of `bin-hook-manager/models/hook/hook.go` with:

  ```go
  package hook

  // Hook defines
  type Hook struct {
      ReceviedURI       string `json:"received_uri"`        // typo preserved for backward compat
      ReceivedData      []byte `json:"received_data"`
      ReceivedMethod    string `json:"received_method"`     // "GET" or "POST"
      ReceivedSignature string `json:"received_signature"`  // X-Hub-Signature-256 header value
  }
  ```

- [ ] **Step 2: Update ServiceHandler interface — Conversation returns (string, error)**

  In `bin-hook-manager/pkg/servicehandler/main.go`, change the `Conversation` method signature:

  ```go
  type ServiceHandler interface {
      Email(ctx context.Context, r *http.Request) error
      Message(ctx context.Context, r *http.Request) error
      Conversation(ctx context.Context, r *http.Request) (string, error)  // was: error
      Billing(ctx context.Context, r *http.Request) error
  }
  ```

- [ ] **Step 3: Regenerate mock_servicehandler.go**

  ```bash
  cd bin-hook-manager && go generate ./...
  ```
  This updates `pkg/servicehandler/mock_servicehandler.go` with the new `(string, error)` return type for `Conversation`.

- [ ] **Step 4: Write failing tests for GET challenge forwarding**

  In `bin-hook-manager/pkg/servicehandler/conversation_test.go`, add a test case:

  ```go
  func Test_Conversation_GET_HubChallenge(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := &serviceHandler{reqHandler: mockReq}

      body := []byte(``)
      r := httptest.NewRequest(http.MethodGet, "https://hook.voipbin.net/v1.0/conversation/accounts/acct-id?hub.mode=subscribe&hub.verify_token=tok&hub.challenge=chal123", bytes.NewBuffer(body))

      mockReq.EXPECT().ConversationV1HookGet(gomock.Any(), gomock.Any()).Return("chal123", nil)

      challenge, err := h.Conversation(context.Background(), r)
      if err != nil {
          t.Fatalf("unexpected error: %v", err)
      }
      if challenge != "chal123" {
          t.Errorf("expected challenge 'chal123', got '%s'", challenge)
      }
  }

  func Test_Conversation_POST_Signature(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := &serviceHandler{reqHandler: mockReq}

      body := []byte(`{"entry":[]}`)
      r := httptest.NewRequest(http.MethodPost, "https://hook.voipbin.net/v1.0/conversation/accounts/acct-id", bytes.NewBuffer(body))
      r.Header.Set("X-Hub-Signature-256", "sha256=abc123")

      mockReq.EXPECT().ConversationV1Hook(gomock.Any(), gomock.Any()).DoAndReturn(
          func(_ context.Context, hm *hmhook.Hook) error {
              if hm.ReceivedSignature != "sha256=abc123" {
                  t.Errorf("expected signature 'sha256=abc123', got '%s'", hm.ReceivedSignature)
              }
              if hm.ReceivedMethod != http.MethodPost {
                  t.Errorf("expected method POST, got '%s'", hm.ReceivedMethod)
              }
              return nil
          },
      ).Return(nil)

      _, err := h.Conversation(context.Background(), r)
      if err != nil {
          t.Fatalf("unexpected error: %v", err)
      }
  }
  ```

- [ ] **Step 5: Run tests to verify they fail**

  ```bash
  cd bin-hook-manager && go test ./pkg/servicehandler/... -v -run "Test_Conversation_GET|Test_Conversation_POST_Sig"
  ```
  Expected: FAIL — compile error because `ConversationV1HookGet` doesn't exist yet (or method mismatch).

  Note: The tests may not even compile at this step since `ConversationV1HookGet` is not yet in requesthandler. That's OK — note the compile error and continue. The tests will pass after Task 5.

- [ ] **Step 6: Implement the new servicehandler.Conversation**

  Replace the content of `bin-hook-manager/pkg/servicehandler/conversation.go` with:

  ```go
  package servicehandler

  import (
      "context"
      "io"
      "net/http"

      "github.com/sirupsen/logrus"

      hmhook "monorepo/bin-hook-manager/models/hook"
  )

  // Conversation handles webhook receive for conversation.
  // For GET requests (Meta hub challenge verification) it returns the challenge string.
  // For POST requests (inbound events) it returns "".
  func (h *serviceHandler) Conversation(ctx context.Context, r *http.Request) (string, error) {
      log := logrus.WithFields(logrus.Fields{
          "func":   "Conversation",
          "method": r.Method,
      })

      data, err := io.ReadAll(r.Body)
      if err != nil {
          return "", fmt.Errorf("could not read request body: %w", err)
      }

      req := &hmhook.Hook{
          ReceviedURI:       r.Host + r.URL.RequestURI(), // RequestURI preserves query string
          ReceivedData:      data,
          ReceivedMethod:    r.Method,
          ReceivedSignature: r.Header.Get("X-Hub-Signature-256"),
      }

      if r.Method == http.MethodGet {
          challenge, err := h.reqHandler.ConversationV1HookGet(ctx, req)
          if err != nil {
              log.Errorf("Could not handle hub challenge. err: %v", err)
              return "", err
          }
          return challenge, nil
      }

      log.WithField("request", req).Debugf("Sending hook message.")
      if err := h.reqHandler.ConversationV1Hook(ctx, req); err != nil {
          return "", fmt.Errorf("could not send the hook: %w", err)
      }

      return "", nil
  }
  ```

  Add `"fmt"` to the import list in that file.

- [ ] **Step 7: Update the gin handler to branch on method**

  Replace `bin-hook-manager/api/v1.0/conversation/conversation.go` with:

  ```go
  package conversation

  import (
      "context"
      "net/http"

      "github.com/gin-gonic/gin"
      "github.com/sirupsen/logrus"

      "monorepo/bin-hook-manager/api/models/common"
      "monorepo/bin-hook-manager/pkg/servicehandler"
  )

  // conversationPOST handles all HTTP methods for /conversation/...
  // Named conversationPOST for historical reasons; the route is registered with .Any().
  func conversationPOST(c *gin.Context) {
      ctx := context.Background()
      log := logrus.WithFields(logrus.Fields{
          "func":   "conversationPOST",
          "method": c.Request.Method,
      })

      serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

      challenge, err := serviceHandler.Conversation(ctx, c.Request)
      if err != nil {
          log.Errorf("Could not handle the request. err: %v", err)
          c.AbortWithStatus(http.StatusInternalServerError)
          return
      }

      if c.Request.Method == http.MethodGet && challenge != "" {
          c.String(http.StatusOK, challenge)
          return
      }

      c.AbortWithStatus(http.StatusOK)
  }
  ```

- [ ] **Step 8: Update conversation_test.go for the new return type**

  In `bin-hook-manager/api/v1.0/conversation/conversation_test.go`, update all `mockSvc.EXPECT().Conversation(...)` calls that previously returned `nil` to return `("", nil)`, and update the error test to return `("", fmt.Errorf(...))`.

  The test mock calls change from:
  ```go
  mockSvc.EXPECT().Conversation(gomock.Any(), gomock.Any()).Return(nil)
  ```
  to:
  ```go
  mockSvc.EXPECT().Conversation(gomock.Any(), gomock.Any()).Return("", nil)
  ```

  Add a new GET test case:
  ```go
  func Test_conversationGET_HubChallenge(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockSvc := servicehandler.NewMockServiceHandler(mc)

      w := httptest.NewRecorder()
      _, r := gin.CreateTestContext(w)
      r.Use(func(c *gin.Context) {
          c.Set(common.OBJServiceHandler, mockSvc)
      })
      setupServer(r)

      req, _ := http.NewRequest("GET", "/v1.0/conversation/accounts/some-id?hub.mode=subscribe&hub.verify_token=tok&hub.challenge=abc123", nil)
      mockSvc.EXPECT().Conversation(gomock.Any(), gomock.Any()).Return("abc123", nil)

      r.ServeHTTP(w, req)

      if w.Code != http.StatusOK {
          t.Errorf("Wrong status. expect: %d, got: %d", http.StatusOK, w.Code)
      }
      if w.Body.String() != "abc123" {
          t.Errorf("Wrong body. expect: 'abc123', got: '%s'", w.Body.String())
      }
  }
  ```

- [ ] **Step 9: Run bin-hook-manager verification (after Task 5 adds ConversationV1HookGet)**

  This step is run after Task 5. For now, verify that the package at least compiles:
  ```bash
  cd bin-hook-manager && go build ./...
  ```
  Expected: may fail if ConversationV1HookGet is not yet in requesthandler — that's OK, continue to Task 5.

---

## Task 5: bin-common-handler requesthandler — Add ConversationV1HookGet, Update ConversationV1AccountCreate

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go`
- Modify: `bin-common-handler/pkg/requesthandler/conversation_hook.go`
- Modify: `bin-common-handler/pkg/requesthandler/conversation_accounts.go`
- Regenerate: `bin-common-handler/pkg/requesthandler/mock_main.go`

- [ ] **Step 1: Add ConversationV1HookGet to the RequestHandler interface**

  In `bin-common-handler/pkg/requesthandler/main.go`, find the existing `ConversationV1Hook` line and add immediately after it:

  ```go
  ConversationV1Hook(ctx context.Context, hm *hmhook.Hook) error
  ConversationV1HookGet(ctx context.Context, hm *hmhook.Hook) (string, error)  // new
  ```

- [ ] **Step 2: Add ConversationV1AccountCreate providerData param to interface**

  In the same `main.go`, find `ConversationV1AccountCreate` and update its signature:

  ```go
  ConversationV1AccountCreate(
      ctx context.Context,
      customerID uuid.UUID,
      accountType cvaccount.Type,
      name string,
      detail string,
      secret string,
      token string,
      messageFlowID uuid.UUID,
      providerData json.RawMessage,  // new last param
  ) (*cvaccount.Account, error)
  ```

  Ensure `"encoding/json"` is in the imports of `main.go`.

- [ ] **Step 3: Implement ConversationV1HookGet**

  In `bin-common-handler/pkg/requesthandler/conversation_hook.go`, add the new function:

  ```go
  // ConversationV1HookGet sends a GET hook to conversation-manager and returns the challenge string.
  func (r *requestHandler) ConversationV1HookGet(ctx context.Context, hm *hmhook.Hook) (string, error) {
      uri := "/v1/hooks"

      m, err := json.Marshal(hm)
      if err != nil {
          return "", err
      }

      tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/hooks-get", requestTimeoutDefault, 0, ContentTypeJSON, m)
      if err != nil {
          return "", err
      }

      var res string
      if errParse := parseResponse(tmp, &res); errParse != nil {
          return "", errParse
      }

      return res, nil
  }
  ```

- [ ] **Step 4: Update ConversationV1AccountCreate implementation**

  In `bin-common-handler/pkg/requesthandler/conversation_accounts.go`, update `ConversationV1AccountCreate` to accept and forward `providerData`:

  ```go
  func (r *requestHandler) ConversationV1AccountCreate(
      ctx context.Context,
      customerID uuid.UUID,
      accountType cvaccount.Type,
      name string,
      detail string,
      secret string,
      token string,
      messageFlowID uuid.UUID,
      providerData json.RawMessage,
  ) (*cvaccount.Account, error) {
      uri := "/v1/accounts"

      data := &cvrequest.V1DataAccountsPost{
          CustomerID:    customerID,
          Type:          accountType,
          Name:          name,
          Detail:        detail,
          Secret:        secret,
          Token:         token,
          MessageFlowID: messageFlowID,
          ProviderData:  providerData,  // new field
      }

      m, err := json.Marshal(data)
      if err != nil {
          return nil, err
      }

      tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodPost, "conversation/accounts", 30000, 0, ContentTypeJSON, m)
      if err != nil {
          return nil, err
      }

      var res cvaccount.Account
      if errParse := parseResponse(tmp, &res); errParse != nil {
          return nil, errParse
      }

      return &res, nil
  }
  ```

  Add `"encoding/json"` to imports if not already present.

- [ ] **Step 5: Regenerate mock_main.go**

  ```bash
  cd bin-common-handler && go generate ./...
  ```
  This updates `pkg/requesthandler/mock_main.go`.

- [ ] **Step 6: Verify bin-common-handler**

  ```bash
  cd bin-common-handler && go mod tidy && go build ./... && go test ./pkg/requesthandler/...
  ```
  Expected: PASS.

- [ ] **Step 7: Return to bin-hook-manager and complete verification**

  ```bash
  cd bin-hook-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
  ```
  Expected: all green. The `ConversationV1HookGet` is now available so the servicehandler tests can compile and pass.

- [ ] **Step 8: Commit**

  ```bash
  git add bin-common-handler/pkg/requesthandler/main.go \
          bin-common-handler/pkg/requesthandler/conversation_hook.go \
          bin-common-handler/pkg/requesthandler/conversation_accounts.go \
          bin-common-handler/pkg/requesthandler/mock_main.go \
          bin-hook-manager/models/hook/hook.go \
          bin-hook-manager/pkg/servicehandler/main.go \
          bin-hook-manager/pkg/servicehandler/conversation.go \
          bin-hook-manager/pkg/servicehandler/conversation_test.go \
          bin-hook-manager/pkg/servicehandler/mock_servicehandler.go \
          bin-hook-manager/api/v1.0/conversation/conversation.go \
          bin-hook-manager/api/v1.0/conversation/conversation_test.go
  git commit -m "NOJIRA-Add-whatsapp-conversation-account

  - bin-common-handler: Add ConversationV1HookGet to requesthandler
  - bin-common-handler: Add providerData param to ConversationV1AccountCreate
  - bin-hook-manager: Extend Hook model with ReceivedMethod and ReceivedSignature fields
  - bin-hook-manager: Update servicehandler.Conversation to forward method, signature, full query string
  - bin-hook-manager: Update gin handler to write challenge body on GET"
  ```

---

## Task 6: New whatsapphandler Package

**Files:**
- Create: `bin-conversation-manager/pkg/whatsapphandler/main.go`
- Create: `bin-conversation-manager/pkg/whatsapphandler/setup.go`
- Create: `bin-conversation-manager/pkg/whatsapphandler/teardown.go`
- Create: `bin-conversation-manager/pkg/whatsapphandler/send.go`
- Create: `bin-conversation-manager/pkg/whatsapphandler/hook.go`
- Create: `bin-conversation-manager/pkg/whatsapphandler/verify.go`
- Create: `bin-conversation-manager/pkg/whatsapphandler/setup_test.go`
- Create: `bin-conversation-manager/pkg/whatsapphandler/send_test.go`
- Create: `bin-conversation-manager/pkg/whatsapphandler/hook_test.go`
- Create: `bin-conversation-manager/pkg/whatsapphandler/verify_test.go`

### Sub-task 6a: Interface and Constructor

- [ ] **Step 1: Write the failing test — interface exists**

  Create `bin-conversation-manager/pkg/whatsapphandler/setup_test.go`:

  ```go
  package whatsapphandler_test

  import (
      "context"
      "encoding/json"
      "testing"

      "go.uber.org/mock/gomock"

      "monorepo/bin-common-handler/pkg/requesthandler"
      "monorepo/bin-conversation-manager/models/account"
      "monorepo/bin-conversation-manager/pkg/whatsapphandler"
  )

  func TestSetup_MissingPhoneNumberID(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := whatsapphandler.NewWhatsAppHandler(mockReq)

      ac := &account.Account{}
      ac.ProviderData = json.RawMessage(`{"phone_number_id":"","app_secret":"sec"}`)

      err := h.Setup(context.Background(), ac)
      if err == nil {
          t.Fatal("expected error for missing phone_number_id, got nil")
      }
  }

  func TestSetup_ValidData(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := whatsapphandler.NewWhatsAppHandler(mockReq)

      ac := &account.Account{}
      ac.ProviderData = json.RawMessage(`{"phone_number_id":"12345","app_secret":"secret"}`)

      err := h.Setup(context.Background(), ac)
      if err != nil {
          t.Fatalf("unexpected error: %v", err)
      }
  }
  ```

- [ ] **Step 2: Run test to verify it fails (compile error expected)**

  ```bash
  cd bin-conversation-manager && go test ./pkg/whatsapphandler/... -v 2>&1 | head -20
  ```
  Expected: FAIL — package does not exist yet.

- [ ] **Step 3: Create main.go — interface and constructor**

  Create `bin-conversation-manager/pkg/whatsapphandler/main.go`:

  ```go
  package whatsapphandler

  //go:generate mockgen -package whatsapphandler -destination ./mock_whatsapphandler.go -source main.go -build_flags=-mod=mod

  import (
      "context"

      "monorepo/bin-common-handler/pkg/requesthandler"
      "monorepo/bin-conversation-manager/models/account"
      "monorepo/bin-conversation-manager/models/conversation"
      "monorepo/bin-conversation-manager/models/message"
  )

  // HookResult contains the conversation and message produced from a WhatsApp inbound message.
  type HookResult struct {
      Conversation *conversation.Conversation
      Message      *message.Message
  }

  // WhatsAppHandler defines the WhatsApp provider operations.
  type WhatsAppHandler interface {
      Setup(ctx context.Context, ac *account.Account) error
      Teardown(ctx context.Context, ac *account.Account) error
      Send(ctx context.Context, cv *conversation.Conversation, ac *account.Account, text string) (wamid string, err error)
      Hook(ctx context.Context, ac *account.Account, rawData []byte, signature string) ([]*HookResult, error)
      VerifyWebhook(ctx context.Context, ac *account.Account, mode string, verifyToken string, challenge string) (string, error)
  }

  type whatsappHandler struct {
      reqHandler requesthandler.RequestHandler
  }

  // NewWhatsAppHandler returns a new WhatsAppHandler.
  func NewWhatsAppHandler(reqHandler requesthandler.RequestHandler) WhatsAppHandler {
      return &whatsappHandler{reqHandler: reqHandler}
  }
  ```

### Sub-task 6b: Setup

- [ ] **Step 4: Create setup.go**

  Create `bin-conversation-manager/pkg/whatsapphandler/setup.go`:

  ```go
  package whatsapphandler

  import (
      "context"
      "encoding/json"
      "fmt"

      "monorepo/bin-conversation-manager/models/account"
  )

  // Setup validates that the account's provider_data is parseable and phone_number_id is non-empty.
  // No remote call is made.
  func (h *whatsappHandler) Setup(_ context.Context, ac *account.Account) error {
      var pd account.WhatsAppProviderData
      if err := json.Unmarshal(ac.ProviderData, &pd); err != nil {
          return fmt.Errorf("whatsapphandler: invalid provider_data: %w", err)
      }
      if pd.PhoneNumberID == "" {
          return fmt.Errorf("whatsapphandler: provider_data.phone_number_id is required")
      }
      return nil
  }
  ```

- [ ] **Step 5: Run setup tests — expect PASS**

  ```bash
  cd bin-conversation-manager && go test ./pkg/whatsapphandler/... -v -run "TestSetup"
  ```
  Expected: PASS.

### Sub-task 6c: Teardown

- [ ] **Step 6: Create teardown.go**

  Create `bin-conversation-manager/pkg/whatsapphandler/teardown.go`:

  ```go
  package whatsapphandler

  import (
      "context"

      "monorepo/bin-conversation-manager/models/account"
  )

  // Teardown is a no-op. Meta does not expose a programmatic webhook deregistration endpoint.
  func (h *whatsappHandler) Teardown(_ context.Context, _ *account.Account) error {
      return nil
  }
  ```

### Sub-task 6d: Send

- [ ] **Step 7: Write the failing Send test**

  Create `bin-conversation-manager/pkg/whatsapphandler/send_test.go`:

  ```go
  package whatsapphandler_test

  import (
      "context"
      "encoding/json"
      "net/http"
      "net/http/httptest"
      "testing"

      "go.uber.org/mock/gomock"

      "monorepo/bin-common-handler/pkg/requesthandler"
      "monorepo/bin-conversation-manager/models/account"
      "monorepo/bin-conversation-manager/models/conversation"
      "monorepo/bin-conversation-manager/pkg/whatsapphandler"
  )

  func TestSend_Success(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      // Stub the Cloud API endpoint
      srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
          w.Header().Set("Content-Type", "application/json")
          _, _ = w.Write([]byte(`{"messages":[{"id":"wamid.abc123"}]}`))
      }))
      defer srv.Close()

      // Override graphAPIBase for test (see send.go for the var)
      origBase := whatsapphandler.GraphAPIBase
      whatsapphandler.GraphAPIBase = srv.URL
      defer func() { whatsapphandler.GraphAPIBase = origBase }()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := whatsapphandler.NewWhatsAppHandler(mockReq)

      ac := &account.Account{}
      ac.Token = "test-token"
      ac.ProviderData = json.RawMessage(`{"phone_number_id":"12345","app_secret":"sec"}`)

      cv := &conversation.Conversation{DialogID: "+15551234567"}

      wamid, err := h.Send(context.Background(), cv, ac, "hello world")
      if err != nil {
          t.Fatalf("unexpected error: %v", err)
      }
      if wamid != "wamid.abc123" {
          t.Errorf("expected wamid 'wamid.abc123', got '%s'", wamid)
      }
  }

  func TestSend_NonOK(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
          w.WriteHeader(http.StatusBadRequest)
          _, _ = w.Write([]byte(`{"error":{"message":"bad request"}}`))
      }))
      defer srv.Close()

      origBase := whatsapphandler.GraphAPIBase
      whatsapphandler.GraphAPIBase = srv.URL
      defer func() { whatsapphandler.GraphAPIBase = origBase }()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := whatsapphandler.NewWhatsAppHandler(mockReq)

      ac := &account.Account{}
      ac.Token = "test-token"
      ac.ProviderData = json.RawMessage(`{"phone_number_id":"12345","app_secret":"sec"}`)

      cv := &conversation.Conversation{DialogID: "+15551234567"}

      _, err := h.Send(context.Background(), cv, ac, "hello")
      if err == nil {
          t.Fatal("expected error for non-200 response, got nil")
      }
  }
  ```

- [ ] **Step 8: Run Send tests to verify they fail**

  ```bash
  cd bin-conversation-manager && go test ./pkg/whatsapphandler/... -v -run "TestSend"
  ```
  Expected: FAIL — `Send` method not implemented, `GraphAPIBase` var does not exist.

- [ ] **Step 9: Create send.go**

  Create `bin-conversation-manager/pkg/whatsapphandler/send.go`:

  ```go
  package whatsapphandler

  import (
      "bytes"
      "context"
      "encoding/json"
      "fmt"
      "net/http"
      "time"

      "github.com/sirupsen/logrus"

      "monorepo/bin-conversation-manager/models/account"
      "monorepo/bin-conversation-manager/models/conversation"
  )

  // GraphAPIBase is the base URL for the WhatsApp Business Cloud API.
  // Exported so tests can override it with a local httptest server.
  var GraphAPIBase = "https://graph.facebook.com"

  const graphAPIVersion = "v19.0"

  // Send sends a text message via the WhatsApp Business Cloud API.
  // Returns the wamid (WhatsApp message ID) on success.
  func (h *whatsappHandler) Send(ctx context.Context, cv *conversation.Conversation, ac *account.Account, text string) (string, error) {
      log := logrus.WithFields(logrus.Fields{
          "func":            "whatsapphandler.Send",
          "conversation_id": cv.ID,
      })

      var pd account.WhatsAppProviderData
      if err := json.Unmarshal(ac.ProviderData, &pd); err != nil {
          return "", fmt.Errorf("whatsapphandler.Send: invalid provider_data: %w", err)
      }

      payload := map[string]any{
          "messaging_product": "whatsapp",
          "to":                cv.DialogID,
          "type":              "text",
          "text":              map[string]string{"body": text},
      }

      body, err := json.Marshal(payload)
      if err != nil {
          return "", fmt.Errorf("whatsapphandler.Send: marshal payload: %w", err)
      }

      url := fmt.Sprintf("%s/%s/%s/messages", GraphAPIBase, graphAPIVersion, pd.PhoneNumberID)

      httpClient := &http.Client{Timeout: 30 * time.Second}
      req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
      if err != nil {
          return "", fmt.Errorf("whatsapphandler.Send: create request: %w", err)
      }
      req.Header.Set("Authorization", "Bearer "+ac.Token)
      req.Header.Set("Content-Type", "application/json")

      resp, err := httpClient.Do(req)
      if err != nil {
          return "", fmt.Errorf("whatsapphandler.Send: http request: %w", err)
      }
      defer func() { _ = resp.Body.Close() }()

      if resp.StatusCode < 200 || resp.StatusCode >= 300 {
          return "", fmt.Errorf("whatsapphandler.Send: API returned %d", resp.StatusCode)
      }

      var result struct {
          Messages []struct {
              ID string `json:"id"`
          } `json:"messages"`
      }
      if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
          return "", fmt.Errorf("whatsapphandler.Send: decode response: %w", err)
      }
      if len(result.Messages) == 0 {
          return "", fmt.Errorf("whatsapphandler.Send: no messages in response")
      }

      log.Debugf("Sent WhatsApp message. wamid: %s", result.Messages[0].ID)
      return result.Messages[0].ID, nil
  }
  ```

- [ ] **Step 10: Run Send tests — expect PASS**

  ```bash
  cd bin-conversation-manager && go test ./pkg/whatsapphandler/... -v -run "TestSend"
  ```
  Expected: PASS.

### Sub-task 6e: VerifyWebhook

- [ ] **Step 11: Write the failing VerifyWebhook tests**

  Create `bin-conversation-manager/pkg/whatsapphandler/verify_test.go`:

  ```go
  package whatsapphandler_test

  import (
      "context"
      "testing"

      "go.uber.org/mock/gomock"

      "monorepo/bin-common-handler/pkg/requesthandler"
      "monorepo/bin-conversation-manager/models/account"
      "monorepo/bin-conversation-manager/pkg/whatsapphandler"
  )

  func TestVerifyWebhook_CorrectToken(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := whatsapphandler.NewWhatsAppHandler(mockReq)

      ac := &account.Account{}
      ac.Secret = "my-verify-token"

      result, err := h.VerifyWebhook(context.Background(), ac, "subscribe", "my-verify-token", "challenge_xyz")
      if err != nil {
          t.Fatalf("unexpected error: %v", err)
      }
      if result != "challenge_xyz" {
          t.Errorf("expected challenge 'challenge_xyz', got '%s'", result)
      }
  }

  func TestVerifyWebhook_WrongToken(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := whatsapphandler.NewWhatsAppHandler(mockReq)

      ac := &account.Account{}
      ac.Secret = "my-verify-token"

      _, err := h.VerifyWebhook(context.Background(), ac, "subscribe", "wrong-token", "challenge_xyz")
      if err == nil {
          t.Fatal("expected error for wrong token, got nil")
      }
  }

  func TestVerifyWebhook_WrongMode(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := whatsapphandler.NewWhatsAppHandler(mockReq)

      ac := &account.Account{}
      ac.Secret = "my-verify-token"

      _, err := h.VerifyWebhook(context.Background(), ac, "not-subscribe", "my-verify-token", "challenge_xyz")
      if err == nil {
          t.Fatal("expected error for wrong mode, got nil")
      }
  }
  ```

- [ ] **Step 12: Run VerifyWebhook tests to verify they fail**

  ```bash
  cd bin-conversation-manager && go test ./pkg/whatsapphandler/... -v -run "TestVerify"
  ```
  Expected: FAIL — method not implemented.

- [ ] **Step 13: Create verify.go**

  Create `bin-conversation-manager/pkg/whatsapphandler/verify.go`:

  ```go
  package whatsapphandler

  import (
      "context"
      "fmt"

      "monorepo/bin-conversation-manager/models/account"
  )

  // VerifyWebhook handles Meta's GET hub challenge verification.
  // Returns the challenge string if mode is "subscribe" and verifyToken matches ac.Secret.
  func (h *whatsappHandler) VerifyWebhook(_ context.Context, ac *account.Account, mode string, verifyToken string, challenge string) (string, error) {
      if mode != "subscribe" {
          return "", fmt.Errorf("whatsapphandler: unexpected hub.mode: %q", mode)
      }
      if verifyToken != ac.Secret {
          return "", fmt.Errorf("whatsapphandler: hub.verify_token mismatch")
      }
      return challenge, nil
  }
  ```

- [ ] **Step 14: Run VerifyWebhook tests — expect PASS**

  ```bash
  cd bin-conversation-manager && go test ./pkg/whatsapphandler/... -v -run "TestVerify"
  ```
  Expected: PASS.

### Sub-task 6f: Hook (inbound message processing)

- [ ] **Step 15: Write the failing Hook tests**

  Create `bin-conversation-manager/pkg/whatsapphandler/hook_test.go`:

  ```go
  package whatsapphandler_test

  import (
      "context"
      "encoding/json"
      "testing"

      "github.com/gofrs/uuid"
      "go.uber.org/mock/gomock"

      commonaddress "monorepo/bin-common-handler/models/address"
      "monorepo/bin-common-handler/pkg/requesthandler"
      "monorepo/bin-conversation-manager/models/account"
      "monorepo/bin-conversation-manager/models/conversation"
      "monorepo/bin-conversation-manager/models/message"
      "monorepo/bin-conversation-manager/pkg/whatsapphandler"
  )

  // validSignature computes a valid HMAC-SHA256 signature for use in tests.
  func validSignature(t *testing.T, secret string, body []byte) string {
      t.Helper()
      import_("crypto/hmac"; "crypto/sha256"; "encoding/hex")
      mac := hmac.New(sha256.New, []byte(secret))
      mac.Write(body)
      return "sha256=" + hex.EncodeToString(mac.Sum(nil))
  }

  func TestHook_ValidTextMessage(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := whatsapphandler.NewWhatsAppHandler(mockReq)

      appSecret := "test-secret"
      ac := &account.Account{}
      ac.CustomerID = uuid.Must(uuid.NewV4())
      ac.ProviderData = json.RawMessage(`{"phone_number_id":"12345","app_secret":"` + appSecret + `"}`)

      rawData := []byte(`{
          "entry": [{
              "changes": [{
                  "value": {
                      "metadata": {"display_phone_number": "+10001112222"},
                      "contacts": [{"profile": {"name": "Alice"}}],
                      "messages": [{
                          "from": "+15551234567",
                          "id": "wamid.msg1",
                          "type": "text",
                          "text": {"body": "Hello"}
                      }]
                  }
              }]
          }]
      }`)

      sig := validSignature(t, appSecret, rawData)

      cvID := uuid.Must(uuid.NewV4())
      cv := &conversation.Conversation{}
      cv.ID = cvID
      cv.AccountID = ac.ID

      msg := &message.Message{}
      msg.ID = uuid.Must(uuid.NewV4())

      // Expect: dedup check (no existing message with that wamid)
      mockReq.EXPECT().ConversationV1MessageList(gomock.Any(), "", uint64(1),
          map[message.Field]any{message.FieldTransactionID: "wamid.msg1"},
      ).Return([]message.Message{}, nil)

      // Expect: look up conversation by dialog_id
      mockReq.EXPECT().ConversationV1ConversationList(gomock.Any(), "", uint64(1),
          map[conversation.Field]any{
              conversation.FieldType:     conversation.TypeWhatsApp,
              conversation.FieldDialogID: "+15551234567",
              conversation.FieldDeleted:  false,
          },
      ).Return([]conversation.Conversation{*cv}, nil)

      // Expect: create message
      mockReq.EXPECT().ConversationV1MessageCreate(
          gomock.Any(),
          uuid.Nil,
          gomock.Any(), // customerID
          cvID,
          message.DirectionIncoming,
          message.StatusDone,
          message.ReferenceTypeWhatsApp,
          uuid.Nil,
          "wamid.msg1",
          "Hello",
          gomock.Any(),
      ).Return(msg, nil)

      results, err := h.Hook(context.Background(), ac, rawData, sig)
      if err != nil {
          t.Fatalf("unexpected error: %v", err)
      }
      if len(results) != 1 {
          t.Fatalf("expected 1 result, got %d", len(results))
      }
      if results[0].Message.ID != msg.ID {
          t.Errorf("wrong message ID")
      }
  }

  func TestHook_WrongSignature(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := whatsapphandler.NewWhatsAppHandler(mockReq)

      ac := &account.Account{}
      ac.ProviderData = json.RawMessage(`{"phone_number_id":"12345","app_secret":"secret"}`)

      _, err := h.Hook(context.Background(), ac, []byte(`{}`), "sha256=wrongsig")
      if err == nil {
          t.Fatal("expected error for wrong signature, got nil")
      }
  }

  func TestHook_MissingAppSecret(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := whatsapphandler.NewWhatsAppHandler(mockReq)

      ac := &account.Account{}
      ac.ProviderData = json.RawMessage(`{"phone_number_id":"12345","app_secret":""}`)

      _, err := h.Hook(context.Background(), ac, []byte(`{}`), "sha256=whatever")
      if err == nil {
          t.Fatal("expected error for missing app_secret (fail-closed), got nil")
      }
  }

  func TestHook_DuplicateWamid(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := whatsapphandler.NewWhatsAppHandler(mockReq)

      appSecret := "dup-secret"
      ac := &account.Account{}
      ac.CustomerID = uuid.Must(uuid.NewV4())
      ac.ProviderData = json.RawMessage(`{"phone_number_id":"12345","app_secret":"` + appSecret + `"}`)

      rawData := []byte(`{
          "entry": [{
              "changes": [{
                  "value": {
                      "metadata": {"display_phone_number": "+10001112222"},
                      "contacts": [{"profile": {"name": "Bob"}}],
                      "messages": [{
                          "from": "+15559999999",
                          "id": "wamid.dup1",
                          "type": "text",
                          "text": {"body": "Already seen"}
                      }]
                  }
              }]
          }]
      }`)

      sig := validSignature(t, appSecret, rawData)

      existingMsg := message.Message{}
      existingMsg.ID = uuid.Must(uuid.NewV4())

      // Dedup check returns an existing message — should skip
      mockReq.EXPECT().ConversationV1MessageList(gomock.Any(), "", uint64(1),
          map[message.Field]any{message.FieldTransactionID: "wamid.dup1"},
      ).Return([]message.Message{existingMsg}, nil)

      results, err := h.Hook(context.Background(), ac, rawData, sig)
      if err != nil {
          t.Fatalf("unexpected error: %v", err)
      }
      if len(results) != 0 {
          t.Errorf("expected 0 results (deduplication), got %d", len(results))
      }
  }

  func TestHook_CreateOnFirstMessage(t *testing.T) {
      mc := gomock.NewController(t)
      defer mc.Finish()

      mockReq := requesthandler.NewMockRequestHandler(mc)
      h := whatsapphandler.NewWhatsAppHandler(mockReq)

      appSecret := "first-secret"
      acID := uuid.Must(uuid.NewV4())
      ac := &account.Account{}
      ac.ID = acID
      ac.CustomerID = uuid.Must(uuid.NewV4())
      ac.ProviderData = json.RawMessage(`{"phone_number_id":"12345","app_secret":"` + appSecret + `"}`)

      rawData := []byte(`{
          "entry": [{
              "changes": [{
                  "value": {
                      "metadata": {"display_phone_number": "+10001112222"},
                      "contacts": [{"profile": {"name": "Charlie"}}],
                      "messages": [{
                          "from": "+15558888888",
                          "id": "wamid.new1",
                          "type": "text",
                          "text": {"body": "First message"}
                      }]
                  }
              }]
          }]
      }`)

      sig := validSignature(t, appSecret, rawData)

      newCvID := uuid.Must(uuid.NewV4())
      newCv := &conversation.Conversation{}
      newCv.ID = newCvID
      newCv.AccountID = acID

      updatedCv := &conversation.Conversation{}
      updatedCv.ID = newCvID
      updatedCv.AccountID = acID

      msg := &message.Message{}
      msg.ID = uuid.Must(uuid.NewV4())

      // Dedup: no existing message
      mockReq.EXPECT().ConversationV1MessageList(gomock.Any(), "", uint64(1),
          map[message.Field]any{message.FieldTransactionID: "wamid.new1"},
      ).Return([]message.Message{}, nil)

      // No existing conversation for this dialog_id
      mockReq.EXPECT().ConversationV1ConversationList(gomock.Any(), "", uint64(1), gomock.Any()).
          Return([]conversation.Conversation{}, nil)

      // Create conversation (no AccountID param — separate update call)
      mockReq.EXPECT().ConversationV1ConversationCreate(
          gomock.Any(),
          ac.CustomerID,
          gomock.Any(), // name
          gomock.Any(), // detail
          conversation.TypeWhatsApp,
          "+15558888888", // dialog_id = wa_id
          commonaddress.Address{
              Type:   commonaddress.TypeWhatsApp,
              Target: "+10001112222",
          },
          commonaddress.Address{
              Type:       commonaddress.TypeWhatsApp,
              Target:     "+15558888888",
              TargetName: "Charlie",
          },
      ).Return(newCv, nil)

      // Immediately update conversation with AccountID
      mockReq.EXPECT().ConversationV1ConversationUpdate(
          gomock.Any(),
          newCvID,
          map[conversation.Field]any{conversation.FieldAccountID: acID},
      ).Return(updatedCv, nil)

      // Create message
      mockReq.EXPECT().ConversationV1MessageCreate(
          gomock.Any(),
          uuid.Nil,
          gomock.Any(),
          newCvID,
          message.DirectionIncoming,
          message.StatusDone,
          message.ReferenceTypeWhatsApp,
          uuid.Nil,
          "wamid.new1",
          "First message",
          gomock.Any(),
      ).Return(msg, nil)

      results, err := h.Hook(context.Background(), ac, rawData, sig)
      if err != nil {
          t.Fatalf("unexpected error: %v", err)
      }
      if len(results) != 1 {
          t.Fatalf("expected 1 result, got %d", len(results))
      }
  }
  ```

  Note: The `validSignature` helper uses `crypto/hmac`, `crypto/sha256`, and `encoding/hex`. Define it as a proper function at the top of the test file without import-as-strings.

- [ ] **Step 16: Run Hook tests to verify they fail**

  ```bash
  cd bin-conversation-manager && go test ./pkg/whatsapphandler/... -v -run "TestHook"
  ```
  Expected: FAIL — `Hook` not implemented.

- [ ] **Step 17: Create hook.go**

  Create `bin-conversation-manager/pkg/whatsapphandler/hook.go`:

  ```go
  package whatsapphandler

  import (
      "context"
      "crypto/hmac"
      "crypto/sha256"
      "encoding/hex"
      "encoding/json"
      "fmt"
      "strings"

      "github.com/gofrs/uuid"
      "github.com/pkg/errors"
      "github.com/sirupsen/logrus"

      commonaddress "monorepo/bin-common-handler/models/address"
      "monorepo/bin-conversation-manager/internal/convtitle"
      "monorepo/bin-conversation-manager/models/account"
      "monorepo/bin-conversation-manager/models/conversation"
      "monorepo/bin-conversation-manager/models/media"
      "monorepo/bin-conversation-manager/models/message"
  )

  // whatsappPayload is the top-level structure of a Meta WhatsApp webhook payload.
  type whatsappPayload struct {
      Entry []struct {
          Changes []struct {
              Value struct {
                  Metadata struct {
                      DisplayPhoneNumber string `json:"display_phone_number"`
                  } `json:"metadata"`
                  Contacts []struct {
                      Profile struct {
                          Name string `json:"name"`
                      } `json:"profile"`
                  } `json:"contacts"`
                  Messages []struct {
                      From string `json:"from"`
                      ID   string `json:"id"`
                      Type string `json:"type"`
                      Text struct {
                          Body string `json:"body"`
                      } `json:"text"`
                  } `json:"messages"`
              } `json:"value"`
          } `json:"changes"`
      } `json:"entry"`
  }

  // Hook processes an inbound WhatsApp webhook payload.
  // It verifies the HMAC-SHA256 signature, parses text messages,
  // deduplicates by wamid, looks up or creates the conversation,
  // and creates a message record for each new text message.
  func (h *whatsappHandler) Hook(ctx context.Context, ac *account.Account, rawData []byte, signature string) ([]*HookResult, error) {
      log := logrus.WithFields(logrus.Fields{
          "func":       "whatsapphandler.Hook",
          "account_id": ac.ID,
      })

      // Parse provider_data and verify signature (fail-closed)
      var pd account.WhatsAppProviderData
      if err := json.Unmarshal(ac.ProviderData, &pd); err != nil {
          return nil, fmt.Errorf("whatsapphandler.Hook: invalid provider_data: %w", err)
      }
      if pd.AppSecret == "" {
          return nil, fmt.Errorf("whatsapphandler.Hook: app_secret is required (fail-closed)")
      }
      if err := verifyHMAC(pd.AppSecret, rawData, signature); err != nil {
          return nil, fmt.Errorf("whatsapphandler.Hook: signature verification failed: %w", err)
      }

      // Parse payload
      var payload whatsappPayload
      if err := json.Unmarshal(rawData, &payload); err != nil {
          return nil, fmt.Errorf("whatsapphandler.Hook: unmarshal payload: %w", err)
      }

      var results []*HookResult
      for _, entry := range payload.Entry {
          for _, change := range entry.Changes {
              val := change.Value
              selfTarget := val.Metadata.DisplayPhoneNumber

              peerName := ""
              if len(val.Contacts) > 0 {
                  peerName = val.Contacts[0].Profile.Name
              }

              for _, msg := range val.Messages {
                  if msg.Type != "text" {
                      log.Debugf("Skipping non-text message type: %s", msg.Type)
                      continue
                  }

                  r, err := h.processTextMessage(ctx, ac, msg.ID, msg.From, msg.Text.Body, selfTarget, peerName)
                  if err != nil {
                      log.Errorf("Could not process message wamid=%s: %v", msg.ID, err)
                      continue
                  }
                  if r != nil {
                      results = append(results, r)
                  }
              }
          }
      }

      return results, nil
  }

  // processTextMessage handles a single inbound text message.
  func (h *whatsappHandler) processTextMessage(
      ctx context.Context,
      ac *account.Account,
      wamid string,
      waID string,
      text string,
      selfTarget string,
      peerName string,
  ) (*HookResult, error) {
      // Deduplication: skip if a message with this wamid already exists.
      existing, err := h.reqHandler.ConversationV1MessageList(ctx, "", 1,
          map[message.Field]any{message.FieldTransactionID: wamid},
      )
      if err != nil {
          return nil, errors.Wrap(err, "dedup check failed")
      }
      if len(existing) > 0 {
          return nil, nil // already processed
      }

      // Look up or create the conversation.
      cv, err := h.getOrCreateConversation(ctx, ac, waID, selfTarget, peerName)
      if err != nil {
          return nil, errors.Wrap(err, "could not get or create conversation")
      }

      // Create the inbound message record.
      m, err := h.reqHandler.ConversationV1MessageCreate(
          ctx,
          uuid.Nil,
          cv.CustomerID,
          cv.ID,
          message.DirectionIncoming,
          message.StatusDone,
          message.ReferenceTypeWhatsApp,
          uuid.Nil,
          wamid,
          text,
          []media.Media{},
      )
      if err != nil {
          return nil, errors.Wrap(err, "could not create message")
      }

      return &HookResult{Conversation: cv, Message: m}, nil
  }

  // getOrCreateConversation finds an existing conversation by (type=whatsapp, dialog_id=waID)
  // or creates a new one (with a follow-up update to set AccountID).
  func (h *whatsappHandler) getOrCreateConversation(
      ctx context.Context,
      ac *account.Account,
      waID string,
      selfTarget string,
      peerName string,
  ) (*conversation.Conversation, error) {
      cvs, err := h.reqHandler.ConversationV1ConversationList(ctx, "", 1,
          map[conversation.Field]any{
              conversation.FieldType:     conversation.TypeWhatsApp,
              conversation.FieldDialogID: waID,
              conversation.FieldDeleted:  false,
          },
      )
      if err != nil {
          return nil, errors.Wrap(err, "list conversations")
      }
      if len(cvs) > 0 {
          return &cvs[0], nil
      }

      // No existing conversation — create one.
      self := commonaddress.Address{
          Type:   commonaddress.TypeWhatsApp,
          Target: selfTarget,
      }
      peer := commonaddress.Address{
          Type:       commonaddress.TypeWhatsApp,
          Target:     waID,
          TargetName: peerName,
      }

      name, detail := convtitle.Build(conversation.TypeWhatsApp, peer)

      cv, err := h.reqHandler.ConversationV1ConversationCreate(
          ctx,
          ac.CustomerID,
          name,
          detail,
          conversation.TypeWhatsApp,
          waID,
          self,
          peer,
      )
      if err != nil {
          return nil, errors.Wrap(err, "create conversation")
      }

      // Immediately set AccountID — ConversationCreate has no AccountID param.
      updated, err := h.reqHandler.ConversationV1ConversationUpdate(
          ctx,
          cv.ID,
          map[conversation.Field]any{conversation.FieldAccountID: ac.ID},
      )
      if err != nil {
          return nil, errors.Wrap(err, "update conversation AccountID")
      }

      return updated, nil
  }

  // verifyHMAC verifies an HMAC-SHA256 signature in "sha256=<hex>" format.
  func verifyHMAC(secret string, data []byte, signature string) error {
      expected := strings.TrimPrefix(signature, "sha256=")

      mac := hmac.New(sha256.New, []byte(secret))
      mac.Write(data)
      computed := hex.EncodeToString(mac.Sum(nil))

      if !hmac.Equal([]byte(computed), []byte(expected)) {
          return fmt.Errorf("HMAC mismatch")
      }
      return nil
  }
  ```

- [ ] **Step 18: Run Hook tests — expect PASS**

  ```bash
  cd bin-conversation-manager && go test ./pkg/whatsapphandler/... -v -run "TestHook"
  ```
  Expected: PASS.

- [ ] **Step 19: Generate mock**

  ```bash
  cd bin-conversation-manager && go generate ./pkg/whatsapphandler/...
  ```
  This creates `pkg/whatsapphandler/mock_whatsapphandler.go`.

- [ ] **Step 20: Run all whatsapphandler tests**

  ```bash
  cd bin-conversation-manager && go test ./pkg/whatsapphandler/... -v
  ```
  Expected: all PASS.

- [ ] **Step 21: Commit**

  ```bash
  git add bin-conversation-manager/pkg/whatsapphandler/
  git commit -m "NOJIRA-Add-whatsapp-conversation-account

  - bin-conversation-manager: Add whatsapphandler package (Setup, Teardown, Send, Hook, VerifyWebhook)
  - bin-conversation-manager: HMAC-SHA256 fail-closed signature verification
  - bin-conversation-manager: Create-on-first-message with two-RPC AccountID persistence
  - bin-conversation-manager: wamid deduplication via TransactionID lookup"
  ```

---

## Task 7: conversationhandler Updates

**Files:**
- Modify: `bin-conversation-manager/pkg/conversationhandler/main.go`
- Modify: `bin-conversation-manager/pkg/conversationhandler/hook.go`
- Regenerate: `bin-conversation-manager/pkg/conversationhandler/mock_conversationhandler.go`

- [ ] **Step 1: Write the failing tests for WhatsApp hook dispatch and HookVerify**

  In `bin-conversation-manager/pkg/conversationhandler/hook_test.go`, add:

  ```go
  func TestHook_WhatsApp(t *testing.T) {
      // table-driven test: valid WhatsApp POST hook dispatched to hookWhatsApp
      // set up mock whatsappHandler.Hook to return a HookResult
      // assert HookResult is processed (runExecuteMode called)
      // ... (follow the pattern of the existing TestHook_Line tests in hook_test.go)
  }

  func TestHookVerify_WhatsApp(t *testing.T) {
      // set up account with type=whatsapp
      // mock whatsappHandler.VerifyWebhook to return "challenge_value"
      // call h.HookVerify(ctx, uri, "subscribe", "tok", "challenge_value")
      // assert returned string == "challenge_value"
  }

  func TestHookVerify_NonWhatsApp_ReturnsError(t *testing.T) {
      // set up account with type=line
      // call HookVerify
      // assert error: "unsupported account type for webhook verification"
  }
  ```

- [ ] **Step 2: Run tests to verify they fail**

  ```bash
  cd bin-conversation-manager && go test ./pkg/conversationhandler/... -v -run "TestHook_WhatsApp|TestHookVerify"
  ```
  Expected: FAIL — `hookWhatsApp`, `HookVerify` not defined; `Hook` interface mismatch.

- [ ] **Step 3: Update ConversationHandler interface in main.go**

  In `bin-conversation-manager/pkg/conversationhandler/main.go`:

  3a. Change the `Hook` method signature:
  ```go
  // before
  Hook(ctx context.Context, uri string, data []byte) error
  // after
  Hook(ctx context.Context, uri string, method string, signature string, data []byte) error
  ```

  3b. Add `HookVerify` method:
  ```go
  HookVerify(ctx context.Context, uri string, mode string, verifyToken string, challenge string) (string, error)
  ```

  3c. Add `whatsappHandler whatsapphandler.WhatsAppHandler` field to the `conversationHandler` struct.

  3d. Add `whatsappHandler whatsapphandler.WhatsAppHandler` parameter to `NewConversationHandler` and set it in the returned struct.

  3e. Add `"monorepo/bin-conversation-manager/pkg/whatsapphandler"` to imports.

- [ ] **Step 4: Update hook.go — Hook dispatch and HookVerify**

  In `bin-conversation-manager/pkg/conversationhandler/hook.go`:

  4a. Update `Hook` method signature to match the new interface:
  ```go
  func (h *conversationHandler) Hook(ctx context.Context, uri string, method string, signature string, data []byte) error {
  ```

  4b. In the `switch ac.Type` block, add:
  ```go
  case account.TypeWhatsApp:
      if errHook := h.hookWhatsApp(ctx, ac, data, signature); errHook != nil {
          log.Errorf("Could not handle WhatsApp hook. err: %v", errHook)
          return errHook
      }
  ```

  4c. Add `hookWhatsApp` function:
  ```go
  // hookWhatsApp handles the WhatsApp type of hook message.
  func (h *conversationHandler) hookWhatsApp(ctx context.Context, ac *account.Account, data []byte, signature string) error {
      log := logrus.WithFields(logrus.Fields{
          "func":       "hookWhatsApp",
          "account_id": ac.ID,
      })

      results, err := h.whatsappHandler.Hook(ctx, ac, data, signature)
      if err != nil {
          log.Errorf("Could not parse WhatsApp message. err: %v", err)
          return err
      }

      for _, r := range results {
          if r.Conversation == nil || r.Message == nil {
              continue
          }
          mode := h.getExecuteMode(r.Conversation)
          switch mode {
          case ExecuteModeAgent:
              if errAgent := h.runExecuteModeAgent(ctx, r.Conversation, r.Message); errAgent != nil {
                  return errors.Wrapf(errAgent, "could not run agent mode. account_id: %s", ac.ID)
              }
          case ExecuteModeFlow:
              if errFlow := h.runExecuteModeFlow(ctx, r.Conversation, r.Message); errFlow != nil {
                  return errors.Wrapf(errFlow, "could not run flow mode. account_id: %s", ac.ID)
              }
          case ExecuteModeNone:
              // reserved; no-op
          default:
              return fmt.Errorf("unknown execute mode: %s", mode)
          }
      }
      return nil
  }
  ```

  4d. Add `HookVerify` implementation:
  ```go
  // HookVerify handles GET hub challenge verification for WhatsApp webhooks.
  func (h *conversationHandler) HookVerify(ctx context.Context, uri string, mode string, verifyToken string, challenge string) (string, error) {
      log := logrus.WithFields(logrus.Fields{
          "func": "HookVerify",
          "uri":  uri,
      })

      // Parse account_id from URI path (same as Hook)
      u, err := url.Parse(uri)
      if err != nil {
          return "", err
      }
      tmpVals := strings.Split(u.Path, "/")
      if len(tmpVals) < 5 {
          return "", fmt.Errorf("could not parse account_id from uri: %s", uri)
      }
      accountID := uuid.FromStringOrNil(tmpVals[4])

      ac, err := h.accountHandler.Get(ctx, accountID)
      if err != nil {
          log.Errorf("Could not get account. err: %v", err)
          return "", errors.Wrap(err, "could not get account")
      }

      if ac.Type != account.TypeWhatsApp {
          return "", fmt.Errorf("unsupported account type for webhook verification: %s", ac.Type)
      }

      return h.whatsappHandler.VerifyWebhook(ctx, ac, mode, verifyToken, challenge)
  }
  ```

- [ ] **Step 5: Regenerate mock_conversationhandler.go**

  ```bash
  cd bin-conversation-manager && go generate ./pkg/conversationhandler/...
  ```

- [ ] **Step 6: Run conversationhandler tests**

  ```bash
  cd bin-conversation-manager && go test ./pkg/conversationhandler/... -v
  ```
  Expected: PASS. (Some existing tests may need updating for the new `Hook` signature — add `method` and `signature` args where `Hook` is called.)

- [ ] **Step 7: Fix any existing tests that use the old Hook signature**

  Search for calls to `conversationHandler.Hook` in tests and update from:
  ```go
  h.Hook(ctx, uri, data)
  ```
  to:
  ```go
  h.Hook(ctx, uri, "POST", "", data)
  ```

---

## Task 8: accounthandler Wiring

**Files:**
- Modify: `bin-conversation-manager/pkg/accounthandler/main.go`
- Modify: `bin-conversation-manager/pkg/accounthandler/db.go`
- Modify: `bin-conversation-manager/pkg/accounthandler/setup.go`
- Regenerate: `bin-conversation-manager/pkg/accounthandler/mock_accounthandler.go`

- [ ] **Step 1: Add ProviderData to AccountHandler.Create interface**

  In `pkg/accounthandler/main.go`, update the `AccountHandler` interface:
  ```go
  Create(ctx context.Context, customerID uuid.UUID, accountType account.Type, name string, detail string, secret string, token string, messageFlowID uuid.UUID, providerData json.RawMessage) (*account.Account, error)
  ```

  Add `"encoding/json"` to imports. Add `whatsappHandler whatsapphandler.WhatsAppHandler` field to `accountHandler` struct. Add it as a parameter to `NewAccountHandler` and set in constructor. Add import for `"monorepo/bin-conversation-manager/pkg/whatsapphandler"`.

- [ ] **Step 2: Update Create in db.go**

  In `pkg/accounthandler/db.go`, update `Create`:
  ```go
  func (h *accountHandler) Create(ctx context.Context, customerID uuid.UUID, accountType account.Type, name string, detail string, secret string, token string, messageFlowID uuid.UUID, providerData json.RawMessage) (*account.Account, error) {
  ```

  Set `ProviderData` on the account struct:
  ```go
  ac := &account.Account{
      Identity: commonidentity.Identity{
          ID:         id,
          CustomerID: customerID,
      },
      Type:          accountType,
      Name:          name,
      Detail:        detail,
      Secret:        secret,
      Token:         token,
      MessageFlowID: messageFlowID,
      ProviderData:  providerData,   // add this
  }
  ```

- [ ] **Step 3: Add WhatsApp to setup/teardown**

  In `pkg/accounthandler/setup.go`, add to the `setup` switch:
  ```go
  case account.TypeWhatsApp:
      err = h.whatsappHandler.Setup(ctx, ac)
  ```

  In the `teardown` switch (WhatsApp is a no-op):
  ```go
  case account.TypeWhatsApp:
      // no-op: Meta has no programmatic webhook deregistration
  ```

- [ ] **Step 4: Regenerate mock_accounthandler.go**

  ```bash
  cd bin-conversation-manager && go generate ./pkg/accounthandler/...
  ```

- [ ] **Step 5: Run accounthandler tests**

  ```bash
  cd bin-conversation-manager && go test ./pkg/accounthandler/... -v
  ```
  Update any test files that call `accountHandler.Create` to pass a final `json.RawMessage(nil)` argument.

---

## Task 9: messagehandler sendWhatsApp

**Files:**
- Modify: `bin-conversation-manager/pkg/messagehandler/main.go`
- Modify: `bin-conversation-manager/pkg/messagehandler/send.go`
- Regenerate: `bin-conversation-manager/pkg/messagehandler/mock_messagehandler.go`

- [ ] **Step 1: Write the failing test for WhatsApp send**

  In `pkg/messagehandler/send_test.go`, add:
  ```go
  func TestSend_WhatsApp_Success(t *testing.T) {
      // mock whatsappHandler.Send returning ("wamid.xxx", nil)
      // mock accountHandler.Get returning a WhatsApp account
      // mock messageHandler.Create returning a message with StatusProgressing
      // mock messageHandler.UpdateStatus returning a done message
      // assert returned message has StatusDone
      // ... (follow sendLine test pattern)
  }
  ```

- [ ] **Step 2: Run test to verify it fails**

  ```bash
  cd bin-conversation-manager && go test ./pkg/messagehandler/... -v -run "TestSend_WhatsApp"
  ```
  Expected: FAIL.

- [ ] **Step 3: Add whatsappHandler to messageHandler struct and constructor**

  In `pkg/messagehandler/main.go`:
  - Add `whatsappHandler whatsapphandler.WhatsAppHandler` field to `messageHandler` struct
  - Add it as a parameter to `NewMessageHandler`
  - Add import for `"monorepo/bin-conversation-manager/pkg/whatsapphandler"`

- [ ] **Step 4: Add sendWhatsApp to send.go**

  In `pkg/messagehandler/send.go`, add to the `Send` switch:
  ```go
  case conversation.TypeWhatsApp:
      return h.sendWhatsApp(ctx, cv, text)
  ```

  Add the `sendWhatsApp` function:
  ```go
  // sendWhatsApp sends a message via the WhatsApp Business Cloud API.
  func (h *messageHandler) sendWhatsApp(ctx context.Context, cv *conversation.Conversation, text string) (*message.Message, error) {
      log := logrus.WithFields(logrus.Fields{
          "func":            "sendWhatsApp",
          "conversation_id": cv.ID,
      })

      ac, err := h.accountHandler.Get(ctx, cv.AccountID)
      if err != nil {
          return nil, errors.Wrap(err, "could not get account")
      }

      tmp, err := h.Create(
          ctx,
          uuid.Nil,
          cv.CustomerID,
          cv.ID,
          message.DirectionOutgoing,
          message.StatusProgressing,
          message.ReferenceTypeWhatsApp,
          uuid.Nil,
          "",
          text,
          []media.Media{},
      )
      if err != nil {
          log.Errorf("Could not create message. err: %v", err)
          return nil, err
      }

      wamid, err := h.whatsappHandler.Send(ctx, cv, ac, text)
      if err != nil {
          log.Errorf("Could not send WhatsApp message. err: %v", err)
          _, _ = h.UpdateStatus(ctx, tmp.ID, message.StatusFailed)
          return nil, err
      }

      // Update message status to done and persist wamid as TransactionID.
      res, err := h.UpdateStatus(ctx, tmp.ID, message.StatusDone)
      if err != nil {
          log.Errorf("Could not update message status. err: %v", err)
          return nil, err
      }
      // Persist wamid for future deduplication and status correlation.
      if _, errUpd := h.db.MessageUpdate(ctx, res.ID, map[message.Field]any{
          message.FieldTransactionID: wamid,
      }); errUpd != nil {
          log.Warnf("Could not persist wamid. message_id: %s, wamid: %s, err: %v", res.ID, wamid, errUpd)
      }

      log.Debugf("Sent WhatsApp message. wamid: %s", wamid)
      return res, nil
  }
  ```

  Note: This uses `h.db.MessageUpdate` — verify this method exists in `dbhandler.DBHandler` interface. If not, add `MessageUpdate(ctx, id, fields) (*message.Message, error)` to the interface and implement it (similar to AccountUpdate pattern using squirrel UPDATE query). Alternatively, use a dedicated `UpdateTransactionID` if it exists.

- [ ] **Step 5: Regenerate mock_messagehandler.go**

  ```bash
  cd bin-conversation-manager && go generate ./pkg/messagehandler/...
  ```

- [ ] **Step 6: Run messagehandler tests**

  ```bash
  cd bin-conversation-manager && go test ./pkg/messagehandler/... -v
  ```
  Update existing tests to pass the new `whatsappHandler` argument to `NewMessageHandler`.

---

## Task 10: listenhandler GET Hooks Route and Request Model Updates

**Files:**
- Modify: `bin-conversation-manager/pkg/listenhandler/main.go`
- Modify: `bin-conversation-manager/pkg/listenhandler/v1_hooks.go`
- Modify: `bin-conversation-manager/pkg/listenhandler/models/request/v1_accounts.go`
- Modify: `bin-conversation-manager/pkg/listenhandler/v1_accounts.go`

- [ ] **Step 1: Write the failing test for GET hooks route**

  In `pkg/listenhandler/v1_hooks_test.go`, add a test for `processV1HooksGet`:
  ```go
  func TestProcessV1HooksGet_ValidChallenge(t *testing.T) {
      // set up mock conversationHandler.HookVerify to return ("chal123", nil)
      // send a GET /v1/hooks request with ReceviedURI containing hub.* params
      // assert response StatusCode == 200 and Data == []byte("chal123")
  }

  func TestProcessV1HooksGet_WrongToken(t *testing.T) {
      // mock HookVerify to return ("", fmt.Errorf("token mismatch"))
      // assert response StatusCode == 403
  }
  ```

- [ ] **Step 2: Run tests to verify they fail**

  ```bash
  cd bin-conversation-manager && go test ./pkg/listenhandler/... -v -run "TestProcessV1HooksGet"
  ```
  Expected: FAIL.

- [ ] **Step 3: Add GET route to processRequest switch in main.go**

  In `bin-conversation-manager/pkg/listenhandler/main.go`, add to the hooks section of the switch:
  ```go
  // GET /hooks (Meta hub challenge)
  case regV1Hooks.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
      response, err = h.processV1HooksGet(ctx, m)
      requestType = "/v1/hooks-get"
  ```

  Place this case BEFORE the existing POST case.

- [ ] **Step 4: Add processV1HooksGet and update processV1HooksPost in v1_hooks.go**

  Replace `bin-conversation-manager/pkg/listenhandler/v1_hooks.go` with:

  ```go
  package listenhandler

  import (
      "context"
      "encoding/json"
      "net/url"

      "monorepo/bin-common-handler/models/sock"

      "github.com/sirupsen/logrus"

      "monorepo/bin-conversation-manager/pkg/listenhandler/models/request"
  )

  // processV1HooksPost handles POST /v1/hooks
  func (h *listenHandler) processV1HooksPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
      log := logrus.WithFields(logrus.Fields{
          "func":    "processV1HooksPost",
          "request": m,
      })

      var req request.V1DataHooksPost
      if err := json.Unmarshal(m.Data, &req); err != nil {
          log.Debugf("Could not unmarshal data. err: %v", err)
          return simpleResponse(400), nil
      }
      log.WithField("request", req).Debugf("Received hook. uri: %s", req.ReceviedURI)

      if errHook := h.conversationHandler.Hook(ctx, req.ReceviedURI, req.ReceivedMethod, req.ReceivedSignature, req.ReceivedData); errHook != nil {
          log.Errorf("Could not process hook. err: %v", errHook)
      }

      return &sock.Response{StatusCode: 200, DataType: "application/json"}, nil
  }

  // processV1HooksGet handles GET /v1/hooks (Meta hub challenge verification).
  func (h *listenHandler) processV1HooksGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
      log := logrus.WithFields(logrus.Fields{
          "func":    "processV1HooksGet",
          "request": m,
      })

      var req request.V1DataHooksPost // reuses same model; embeds hmhook.Hook
      if err := json.Unmarshal(m.Data, &req); err != nil {
          log.Debugf("Could not unmarshal data. err: %v", err)
          return simpleResponse(400), nil
      }

      // hub.* params are in req.ReceviedURI (the forwarded external URL),
      // NOT in m.URI (the internal RPC path "/v1/hooks").
      u, err := url.Parse(req.ReceviedURI)
      if err != nil {
          log.Debugf("Could not parse ReceviedURI. err: %v", err)
          return simpleResponse(400), nil
      }

      q := u.Query()
      mode      := q.Get("hub.mode")
      token     := q.Get("hub.verify_token")
      challenge := q.Get("hub.challenge")

      result, err := h.conversationHandler.HookVerify(ctx, req.ReceviedURI, mode, token, challenge)
      if err != nil {
          log.Errorf("HookVerify failed. err: %v", err)
          return simpleResponse(403), nil
      }

      return &sock.Response{
          StatusCode: 200,
          DataType:   "text/plain",
          Data:       []byte(result),
      }, nil
  }
  ```

- [ ] **Step 5: Add ProviderData to V1DataAccountsPost and V1DataAccountsIDPut**

  In `pkg/listenhandler/models/request/v1_accounts.go`:
  ```go
  import (
      "encoding/json"
      "github.com/gofrs/uuid"
      "monorepo/bin-conversation-manager/models/account"
  )

  type V1DataAccountsPost struct {
      CustomerID    uuid.UUID        `json:"customer_id"`
      Type          account.Type     `json:"type"`
      Name          string           `json:"name"`
      Detail        string           `json:"detail"`
      Secret        string           `json:"secret"`
      Token         string           `json:"token"`
      MessageFlowID uuid.UUID        `json:"message_flow_id"`
      ProviderData  json.RawMessage  `json:"provider_data"`   // new
  }

  type V1DataAccountsIDPut struct {
      Name   string `json:"name"`
      Detail string `json:"detail"`
      Secret string `json:"secret"`
      Token  string `json:"token"`
  }
  ```

  Note: `V1DataAccountsIDPut` uses the `GetFilteredItems` approach in the handler — `ProviderData` is handled separately there.

- [ ] **Step 6: Thread ProviderData through v1_accounts.go**

  In `pkg/listenhandler/v1_accounts.go`:

  6a. In `processV1AccountsPost`, update the `Create` call to pass `req.ProviderData`:
  ```go
  tmp, err := h.accountHandler.Create(ctx, req.CustomerID, req.Type, req.Name, req.Detail, req.Secret, req.Token, req.MessageFlowID, req.ProviderData)
  ```

  6b. In `processV1AccountsIDPut`, add `string(account.FieldProviderData)` to `allowedItems`:
  ```go
  allowedItems := []string{
      string(account.FieldName),
      string(account.FieldDetail),
      string(account.FieldType),
      string(account.FieldSecret),
      string(account.FieldToken),
      string(account.FieldMessageFlowID),
      string(account.FieldProviderData),  // new
  }
  ```

  Note: Since `ProviderData` is `json.RawMessage` (not a plain string), the generic `ConvertStringMapToFieldMap` might not handle it correctly. If it fails, handle `provider_data` specially:
  ```go
  // After filtering, check for provider_data and handle as json.RawMessage
  if rawPD, ok := filteredItems[string(account.FieldProviderData)]; ok {
      if pdStr, ok := rawPD.(string); ok {
          tmpFields[account.FieldProviderData] = json.RawMessage(pdStr)
      }
  }
  ```

- [ ] **Step 7: Run listenhandler tests**

  ```bash
  cd bin-conversation-manager && go test ./pkg/listenhandler/... -v
  ```
  Update existing tests: callers of `accountHandler.Create` mock expectations need the new `json.RawMessage` parameter.

---

## Task 11: Dependency Injection in cmd/main.go

**Files:**
- Modify: `bin-conversation-manager/cmd/conversation-manager/main.go`

- [ ] **Step 1: Wire whatsappHandler into all handlers**

  In `run()` function of `cmd/conversation-manager/main.go`:

  ```go
  lineHandler := linehandler.NewLineHandler(reqHandler)
  whatsappHandler := whatsapphandler.NewWhatsAppHandler(reqHandler)  // add

  accountHandler := accounthandler.NewAccountHandler(db, reqHandler, notifyHandler, lineHandler, whatsappHandler)  // add whatsappHandler
  smsHandler := smshandler.NewSMSHandler(reqHandler, accountHandler)

  messageHandler := messagehandler.NewMessageHandler(db, notifyHandler, accountHandler, lineHandler, smsHandler, whatsappHandler)  // add whatsappHandler
  conversationHandler := conversationhandler.NewConversationHandler(db, notifyHandler, reqHandler, accountHandler, messageHandler, lineHandler, smsHandler, whatsappHandler)  // add whatsappHandler
  ```

  Add import: `"monorepo/bin-conversation-manager/pkg/whatsapphandler"`

- [ ] **Step 2: Verify the service compiles**

  ```bash
  cd bin-conversation-manager && go build ./cmd/conversation-manager/...
  ```
  Expected: no errors.

---

## Task 12: Full Verification Per Service

- [ ] **Step 1: Full verification — bin-common-handler**

  ```bash
  cd bin-common-handler
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
  ```

- [ ] **Step 2: Full verification — bin-hook-manager**

  ```bash
  cd bin-hook-manager
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
  ```

- [ ] **Step 3: Full verification — bin-conversation-manager**

  ```bash
  cd bin-conversation-manager
  go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
  ```

- [ ] **Step 4: Commit all verified changes**

  ```bash
  git add bin-conversation-manager/ bin-hook-manager/ bin-common-handler/
  git commit -m "NOJIRA-Add-whatsapp-conversation-account

  - bin-conversation-manager: Wire whatsappHandler into accounthandler, messagehandler, conversationhandler
  - bin-conversation-manager: Add processV1HooksGet for Meta hub challenge
  - bin-conversation-manager: Add ProviderData to account request models
  - bin-conversation-manager: Update Hook interface to accept method and signature
  - bin-conversation-manager: Add HookVerify to conversationhandler
  - bin-hook-manager: Full verification passed
  - bin-common-handler: Full verification passed"
  ```

---

## Task 13: RST Documentation and OpenAPI Spec

**Files:**
- Modify: `bin-api-manager/docsdev/source/` — relevant RST files for conversation accounts
- Rebuild: `bin-api-manager/docsdev/build/`

- [ ] **Step 1: Update conversation account overview RST**

  In `bin-api-manager/docsdev/source/` (find the file covering conversation accounts, likely `conversation_account_overview.rst` or similar):

  - Add `whatsapp` to the account type enum documentation
  - Document `provider_data` as a **write-only** JSON object
  - Describe `phone_number_id` and `app_secret` keys
  - Add a setup section: Meta app creation, phone number, webhook URL format (`https://hook.voipbin.net/v1.0/conversation/accounts/{account_id}`), verify token

  Example RST addition:
  ```rst
  WhatsApp Account
  ^^^^^^^^^^^^^^^^

  Type: ``whatsapp``

  Required fields:

  - ``token``: System user access token (Bearer token for Meta Cloud API calls)
  - ``secret``: Webhook verify token (echoed back during Meta hub challenge)
  - ``provider_data``: JSON object (write-only, never returned in responses):

    .. code-block:: json

       {
         "phone_number_id": "123456789012345",
         "app_secret": "..."
       }

  Webhook URL: ``https://hook.voipbin.net/v1.0/conversation/accounts/{account_id}``

  Setup steps:

  1. Create a Meta Business Account and WhatsApp Business App
  2. Register the phone number on the WhatsApp Business Platform
  3. Create a VoIPbin conversation account (``POST /v1/accounts``) with ``type=whatsapp``
  4. In Meta Business Manager, set the webhook URL to the VoIPbin hook URL above
  5. Set the verify token to match the ``secret`` field
  6. Trigger Meta's webhook verification to confirm the challenge is echoed correctly
  ```

- [ ] **Step 2: Update RST struct docs**

  In `*_struct*.rst` files covering Account — verify that `provider_data`, `secret`, and `token` are NOT listed in the response struct documentation. These fields are stripped by `ConvertWebhookMessage()` and must not appear.

- [ ] **Step 3: Clean rebuild RST docs**

  ```bash
  cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
  ```
  Expected: Sphinx outputs HTML with no errors.

- [ ] **Step 4: Force-add build output**

  ```bash
  git add -f bin-api-manager/docsdev/build/
  git add bin-api-manager/docsdev/source/
  ```

- [ ] **Step 5: Update OpenAPI spec**

  In `bin-openapi-manager` or wherever the OpenAPI YAML/JSON lives, add:
  - `"whatsapp"` to the `account.type` enum
  - `provider_data` field (type: object, nullable, **writeOnly: true**) to Account create/update schema
  - Sub-schema for `WhatsAppProviderData`: `phone_number_id` (string, required), `app_secret` (string)

- [ ] **Step 6: Commit docs changes**

  ```bash
  git commit -m "NOJIRA-Add-whatsapp-conversation-account

  - bin-api-manager: Document WhatsApp conversation account type in RST docs
  - bin-api-manager: Document provider_data as write-only field
  - bin-openapi-manager: Add whatsapp type and provider_data to OpenAPI spec"
  ```

---

## Task 14: Final End-to-End Verification Checklist

These steps are performed manually after deployment (see spec §10 Rollout Order).

- [ ] **1. Apply Alembic migration** — Human deploys via `alembic upgrade` in `bin-dbscheme-manager` after VPN access.

- [ ] **2. Deploy bin-common-handler** — Publish updated Docker image.

- [ ] **3. Deploy bin-hook-manager** — Updated with GET forwarding and signature capture.

- [ ] **4. Deploy bin-conversation-manager** — Updated with WhatsApp handler.

- [ ] **5. Create WhatsApp account via API:**
  ```bash
  curl -X POST https://api.voipbin.net/v1/accounts \
    -H "Authorization: Bearer <api-key>" \
    -H "Content-Type: application/json" \
    -d '{
      "type": "whatsapp",
      "name": "My WhatsApp Account",
      "token": "<system-user-access-token>",
      "secret": "<your-verify-token>",
      "provider_data": {
        "phone_number_id": "<phone-number-id>",
        "app_secret": "<app-secret>"
      }
    }'
  ```
  Expected: 200 with account JSON (provider_data NOT returned).

- [ ] **6. Configure webhook in Meta Business Manager:**
  Set webhook URL to `https://hook.voipbin.net/v1.0/conversation/accounts/{account_id}`

- [ ] **7. Trigger Meta hub verification** — Click "Verify" in Meta Business Manager.
  Expected: `hub.challenge` is returned correctly; Meta shows "Active".

- [ ] **8. Send test inbound WhatsApp message** from a test phone to the registered number.
  Expected: conversation and message records created in VoIPbin.

- [ ] **9. Send test outbound message via API:**
  ```bash
  curl -X POST https://api.voipbin.net/v1/messages \
    -H "Authorization: Bearer <api-key>" \
    -d '{"conversation_id": "<cv-id>", "text": "Hello from VoIPbin"}'
  ```
  Expected: message delivered, `wamid` stored in `transaction_id`.

---

## Self-Review Notes

After writing this plan, checked against the spec:

1. **§3.4 address.TypeWhatsApp** → Task 2 Step 1 ✅
2. **§3.5 convtitle** → Task 3 ✅
3. **§4 Alembic migration** → Task 1 ✅
4. **§5 whatsapphandler** → Task 6 ✅ (all sub-sections covered)
5. **§6.1 hmhook.Hook extension** → Task 4 Step 1 ✅
6. **§6.2 servicehandler.Conversation signature change** → Task 4 Steps 2-6 ✅
7. **§6.3 gin handler GET/POST branching** → Task 4 Step 7 ✅
8. **§6.4 ConversationV1HookGet** → Task 5 Step 3 ✅
9. **§6.5 processV1HooksGet** → Task 10 Steps 3-4 ✅
10. **§6.6 conversationhandler HookVerify + Hook signature** → Task 7 ✅
11. **§7.1 accounthandler setup** → Task 8 Steps 3 ✅
12. **§7.2 messagehandler sendWhatsApp** → Task 9 ✅
13. **§7.3 conversationhandler hookWhatsApp** → Task 7 Step 4c ✅
14. **§7.4 DI wiring** → Task 11 ✅
15. **§8.1 request models** → Task 10 Step 5 ✅
16. **§8.2 provider_data NOT in WebhookMessage** → model webhook.go confirmed unchanged in Task 2 (ProviderData not added to WebhookMessage) ✅
17. **§8.3 OpenAPI spec** → Task 13 Step 5 ✅
18. **§8.4 RST docs** → Task 13 Steps 1-2 ✅

**Placeholder scan:** No TBD/TODO in code blocks. All function signatures are complete. All test expectations are concrete.

**Type consistency:**
- `WhatsAppHandler.Send` returns `(string, error)` ← used in `messagehandler.sendWhatsApp`
- `WhatsAppHandler.Hook` takes `(ctx, ac, rawData, signature)` ← matches `hookWhatsApp` call
- `ConversationHandler.Hook` takes `(ctx, uri, method, signature, data)` ← matches `processV1HooksPost`
- `ConversationHandler.HookVerify` takes `(ctx, uri, mode, verifyToken, challenge)` ← matches `processV1HooksGet`
- `accounthandler.Create` takes `providerData json.RawMessage` as last param ← matches all call sites
- `GraphAPIBase` var exported from `whatsapphandler/send.go` ← used in `send_test.go` for override ✅
