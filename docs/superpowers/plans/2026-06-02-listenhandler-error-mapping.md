# Listenhandler Error Mapping (Phase 0 + Phase 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the four `bin-*-manager` services named in [#955](https://github.com/voipbin/monorepo/issues/955) return the correct HTTP status (notably `404` for a non-existent UUID) instead of `500`, by mapping handler-call errors at the listenhandler call site with the existing `errorResponse` helper.

**Architecture:** Each listenhandler endpoint currently swallows every handler error with `return simpleResponse(500), nil`, discarding the typed `cerrors.NotFound` / raw `dbhandler.ErrNotFound` the handler layer already produces. The fix replaces the *handler-call-error* swallow with `return errorResponse(err), nil`. `errorResponse` (already present in all four services) maps typed `*cerrors.VoipbinError`→true status, `dbhandler.ErrNotFound` (through `errors.Wrap`)→404, everything else→500. Response-`json.Marshal` failures keep `simpleResponse(500)`; request-parse failures keep their local `simpleResponse(400)`. No central-dispatch-tail changes; no handler-layer changes.

**Tech Stack:** Go, gomock, RabbitMQ RPC (`sock.Request`/`sock.Response`), `monorepo/bin-common-handler/models/errors` (`cerrors`).

**Scope:** This plan is **Phase 0** (convention doc) + **Phase 1** (the four issue managers). Phase 2 (monorepo audit) and Phase 3+ (remaining services) are separate plans, written when reached. Within Phase 1, edits cover the four **issue resources'** listenhandler files (`v1_conversations.go`, `messages.go`, `v1_numbers.go`, `v1_groupcalls.go`); other resources hosted in the same managers (e.g. call-manager `calls`/`confbridges`, conversation-manager `accounts`) are deferred to the audit. **call-manager's central tail is intentionally NOT wired** here (see spec) — only the at-site `errorResponse` change is applied.

**Reference spec:** `docs/superpowers/specs/2026-06-02-listenhandler-error-mapping-design.md`

---

## File Structure

| File | Responsibility | Change |
|---|---|---|
| `docs/conventions/listenhandler-error-mapping.md` | Canonical listenhandler error-mapping convention | Create (Task 0) |
| `bin-conversation-manager/pkg/listenhandler/v1_conversations.go` | conversation by-id/collection endpoints | Modify (Task 1) |
| `bin-conversation-manager/pkg/listenhandler/v1_conversations_test.go` | conversation endpoint tests | Modify (Task 1) |
| `bin-message-manager/pkg/listenhandler/messages.go` | message endpoints | Modify (Task 2) |
| `bin-message-manager/pkg/listenhandler/messages_test.go` | message endpoint tests | Modify (Task 2) |
| `bin-number-manager/pkg/listenhandler/v1_numbers.go` | number endpoints | Modify (Task 3) |
| `bin-number-manager/pkg/listenhandler/v1_numbers_test.go` | number endpoint tests | Modify (Task 3) |
| `bin-call-manager/pkg/listenhandler/v1_groupcalls.go` | groupcall endpoints | Modify (Task 4) |
| `bin-call-manager/pkg/listenhandler/v1_groupcalls_test.go` | groupcall endpoint tests | Modify (Task 4) |

**The canonical edit (apply at every handler-call-error site):**

```go
// BEFORE — handler-call error swallowed as 500
tmp, err := h.<xHandler>.<Method>(ctx, ...)
if err != nil {
    log.<...>("... err: %v", err)
    return simpleResponse(500), nil
}

// AFTER — map at the site
tmp, err := h.<xHandler>.<Method>(ctx, ...)
if err != nil {
    log.<...>("... err: %v", err)
    return errorResponse(err), nil
}
```

Leave the `if err != nil { return simpleResponse(500), nil }` block that follows `data, err := json.Marshal(...)` **unchanged** (a marshal failure is genuinely internal). `errorResponse` lives in each service's `main.go` (same package) — no new imports needed in the endpoint files.

---

### Task 0: Phase 0 — Convention document

**Files:**
- Create: `docs/conventions/listenhandler-error-mapping.md`

- [ ] **Step 1: Create the convention doc**

Create `docs/conventions/listenhandler-error-mapping.md` with this content:

````markdown
# Listenhandler Error Mapping

> Backend `bin-*-manager` services map errors to RPC `sock.Response` status codes
> in their `pkg/listenhandler`. This is the canonical convention. Reference
> implementation: `bin-flow-manager/pkg/listenhandler`.

## Rule

Every endpoint function maps a **handler-call error** at the call site with the
shared `errorResponse` helper:

```go
tmp, err := h.xHandler.Method(ctx, ...)
if err != nil {
    return errorResponse(err), nil
}
```

`errorResponse(err)` maps:
- `*cerrors.VoipbinError` → its true HTTP status (via `cerrors.ToResponse`)
- `dbhandler.ErrNotFound` (matched through `errors.Wrap`/`%w`) → `404`
- anything else → `500`

Keep these **local** (do not route through `errorResponse`):
- request URI/`json.Unmarshal` parse failures → `simpleResponse(400)`
- response `json.Marshal` failures → `simpleResponse(500)`

## Do NOT

- Do **not** swallow handler errors with `return simpleResponse(500), nil` — that
  discards the typed not-found the handler layer produces and yields 500 for a
  missing resource (bug class of #955, #953).
- Do **not** flip the central dispatch tail default from `400` to `500`: several
  endpoints return bare `(nil, err)` for parse-class failures that rely on the
  `400` default; flipping regresses them. Map at the site instead.
- Do **not** wire an unwired central tail as part of a de-swallow change if it
  would silently reclassify unrelated bare-`(nil, err)` endpoints (e.g.
  call-manager). Track tail-wiring as its own enumerated + tested change.

## Helper location

`errorResponse` and `simpleResponse` live in each service's
`pkg/listenhandler/main.go`. Helper-level behavior is covered by
`pkg/listenhandler/error_response_test.go`.
````

- [ ] **Step 2: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping
git add docs/conventions/listenhandler-error-mapping.md
git commit -m "NOJIRA-Fix-listenhandler-error-mapping

- docs: Add listenhandler error-mapping convention (Phase 0)"
```

---

### Task 1: conversation-manager

**Files:**
- Modify: `bin-conversation-manager/pkg/listenhandler/v1_conversations.go` (handler-call swallows at lines ~70, ~114, ~150)
- Test: `bin-conversation-manager/pkg/listenhandler/v1_conversations_test.go`

Note: `processV1ConversationsIDPut` (line ~211) already uses `errorResponse(err)` and already returns 404 for a missing id — leave it unchanged (no edit, no test required).

- [ ] **Step 1: Add not-found tests**

Append to `bin-conversation-manager/pkg/listenhandler/v1_conversations_test.go`. The package already imports `reflect`, `testing`, `gomock`, `sockhandler`, `conversationhandler`, `sock`, `uuid`; add the dbhandler import if not present:
`"monorepo/bin-conversation-manager/pkg/dbhandler"`.

```go
func Test_processV1ConversationsID_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID
	}{
		{
			name: "GET non-existent conversation returns 404",
			request: &sock.Request{
				URI:    "/v1/conversations/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodGet,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConversation := conversationhandler.NewMockConversationHandler(mc)
			h := &listenHandler{
				sockHandler:         mockSock,
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().Get(gomock.Any(), tt.id).Return(nil, dbhandler.ErrNotFound)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test — expect FAIL**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping/bin-conversation-manager
go test ./pkg/listenhandler/ -run Test_processV1ConversationsID_notFound -v
```
Expected: FAIL — `StatusCode mismatch. expected: 404, got: 500` (the GET site still swallows to 500).

- [ ] **Step 3: Apply the canonical edit at the handler-call swallows**

In `bin-conversation-manager/pkg/listenhandler/v1_conversations.go`, change the **handler-call-error** `return simpleResponse(500), nil` to `return errorResponse(err), nil` in these functions (the one immediately after the `h.conversationHandler.<Method>` call — NOT the one after `json.Marshal`):

- `processV1ConversationsGet` — after `h.conversationHandler.List(...)` (~line 70)
- `processV1ConversationsPost` — after `h.conversationHandler.Create(...)` (~line 114)
- `processV1ConversationsIDGet` — after `h.conversationHandler.Get(...)` (~line 150)

Example (IDGet):

```go
	tmp, err := h.conversationHandler.Get(ctx, id)
	if err != nil {
		log.Debugf("Could not get a conversation. conversation_id: %s, err: %v", id, err)
		return errorResponse(err), nil
	}
```

- [ ] **Step 4: Run the not-found test — expect PASS**

```bash
go test ./pkg/listenhandler/ -run Test_processV1ConversationsID_notFound -v
```
Expected: PASS.

- [ ] **Step 5: Run the full listenhandler package tests — expect PASS**

```bash
go test ./pkg/listenhandler/ -v
```
Expected: PASS (existing tests assert success-path responses, which are unchanged).

- [ ] **Step 6: Run the mandatory verification workflow**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping/bin-conversation-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: all green.

- [ ] **Step 7: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping
git add bin-conversation-manager/pkg/listenhandler/v1_conversations.go bin-conversation-manager/pkg/listenhandler/v1_conversations_test.go bin-conversation-manager/go.mod bin-conversation-manager/go.sum
git commit -m "NOJIRA-Fix-listenhandler-error-mapping

- bin-conversation-manager: Map handler errors at the listenhandler call site (errorResponse) so non-existent conversation IDs return 404 instead of 500 (#955)"
```

---

### Task 2: message-manager

**Files:**
- Modify: `bin-message-manager/pkg/listenhandler/messages.go` (handler-call swallows at lines ~54, ~89, ~124, ~159)
- Test: `bin-message-manager/pkg/listenhandler/messages_test.go`

- [ ] **Step 1: Add not-found tests**

Append to `bin-message-manager/pkg/listenhandler/messages_test.go`. Add the dbhandler import if not present:
`"monorepo/bin-message-manager/pkg/dbhandler"`.

```go
func Test_processV1MessagesID_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID
		setup   func(m *messagehandler.MockMessageHandler, id uuid.UUID)
	}{
		{
			name: "GET non-existent message returns 404",
			request: &sock.Request{
				URI:    "/v1/messages/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodGet,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
			setup: func(m *messagehandler.MockMessageHandler, id uuid.UUID) {
				m.EXPECT().Get(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)
			},
		},
		{
			name: "DELETE non-existent message returns 404",
			request: &sock.Request{
				URI:    "/v1/messages/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodDelete,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
			setup: func(m *messagehandler.MockMessageHandler, id uuid.UUID) {
				m.EXPECT().Delete(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			h := &listenHandler{
				sockHandler:    mockSock,
				messageHandler: mockMessage,
			}
			tt.setup(mockMessage, tt.id)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test — expect FAIL**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping/bin-message-manager
go test ./pkg/listenhandler/ -run Test_processV1MessagesID_notFound -v
```
Expected: FAIL — both subtests `expected: 404, got: 500`.

- [ ] **Step 3: Apply the canonical edit at the handler-call swallows**

In `bin-message-manager/pkg/listenhandler/messages.go`, change the handler-call-error `return simpleResponse(500), nil` to `return errorResponse(err), nil` in:

- `processV1MessagesGet` — after `h.messageHandler.List(...)` (~line 54)
- `processV1MessagesPost` — after `h.messageHandler.Send(...)` (~line 89)
- `processV1MessagesIDGet` — after `h.messageHandler.Get(...)` (~line 124)
- `processV1MessagesIDDelete` — after `h.messageHandler.Delete(...)` (~line 159)

Leave each post-`json.Marshal` `simpleResponse(500)` unchanged.

- [ ] **Step 4: Run the not-found test — expect PASS**

```bash
go test ./pkg/listenhandler/ -run Test_processV1MessagesID_notFound -v
```
Expected: PASS.

- [ ] **Step 5: Run the full listenhandler package tests — expect PASS**

```bash
go test ./pkg/listenhandler/ -v
```

- [ ] **Step 6: Run the mandatory verification workflow**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping/bin-message-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

- [ ] **Step 7: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping
git add bin-message-manager/pkg/listenhandler/messages.go bin-message-manager/pkg/listenhandler/messages_test.go bin-message-manager/go.mod bin-message-manager/go.sum
git commit -m "NOJIRA-Fix-listenhandler-error-mapping

- bin-message-manager: Map handler errors at the listenhandler call site (errorResponse) so non-existent message IDs return 404 instead of 500 (#955)"
```

---

### Task 3: number-manager

**Files:**
- Modify: `bin-number-manager/pkg/listenhandler/v1_numbers.go` (handler-call swallows at lines ~44, ~80, ~116, ~165, ~217, ~264, ~298)
- Test: `bin-number-manager/pkg/listenhandler/v1_numbers_test.go`

- [ ] **Step 1: Add not-found tests**

Append to `bin-number-manager/pkg/listenhandler/v1_numbers_test.go`. Add the dbhandler import if not present:
`"monorepo/bin-number-manager/pkg/dbhandler"`.

```go
func Test_processV1NumbersID_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID
		setup   func(m *numberhandler.MockNumberHandler, id uuid.UUID)
	}{
		{
			name: "GET non-existent number returns 404",
			request: &sock.Request{
				URI:    "/v1/numbers/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodGet,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
			setup: func(m *numberhandler.MockNumberHandler, id uuid.UUID) {
				m.EXPECT().Get(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)
			},
		},
		{
			name: "DELETE non-existent number returns 404",
			request: &sock.Request{
				URI:    "/v1/numbers/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodDelete,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
			setup: func(m *numberhandler.MockNumberHandler, id uuid.UUID) {
				m.EXPECT().Delete(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)
			h := &listenHandler{
				sockHandler:   mockSock,
				numberHandler: mockNumber,
			}
			tt.setup(mockNumber, tt.id)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test — expect FAIL**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping/bin-number-manager
go test ./pkg/listenhandler/ -run Test_processV1NumbersID_notFound -v
```
Expected: FAIL — both subtests `expected: 404, got: 500`.

- [ ] **Step 3: Apply the canonical edit at the handler-call swallows**

In `bin-number-manager/pkg/listenhandler/v1_numbers.go`, change the handler-call-error `return simpleResponse(500), nil` to `return errorResponse(err), nil` in:

- `processV1NumbersPost` — after the `h.numberHandler.Create/CreateVirtual(...)` block (~line 44)
- `processV1NumbersIDDelete` — after `h.numberHandler.Delete(...)` (~line 80)
- `processV1NumbersIDGet` — after `h.numberHandler.Get(...)` (~line 116)
- `processV1NumbersIDPut` — after `h.numberHandler.Update(...)` (~line 165)
- `processV1NumbersGet` — after `h.numberHandler.List(...)` (~line 217)
- `processV1NumbersIDFlowIDsPut` — after `h.numberHandler.Update(...)` (~line 264)
- `processV1NumbersRenewPost` — after `h.numberHandler.RenewNumbers(...)` (~line 298)

Leave each post-`json.Marshal` `simpleResponse(500)` unchanged.

- [ ] **Step 4: Run the not-found test — expect PASS**

```bash
go test ./pkg/listenhandler/ -run Test_processV1NumbersID_notFound -v
```
Expected: PASS.

- [ ] **Step 5: Run the full listenhandler package tests — expect PASS**

```bash
go test ./pkg/listenhandler/ -v
```

- [ ] **Step 6: Run the mandatory verification workflow**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping/bin-number-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

- [ ] **Step 7: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping
git add bin-number-manager/pkg/listenhandler/v1_numbers.go bin-number-manager/pkg/listenhandler/v1_numbers_test.go bin-number-manager/go.mod bin-number-manager/go.sum
git commit -m "NOJIRA-Fix-listenhandler-error-mapping

- bin-number-manager: Map handler errors at the listenhandler call site (errorResponse) so non-existent number IDs return 404 instead of 500 (#955)"
```

---

### Task 4: call-manager (groupcalls)

**Files:**
- Modify: `bin-call-manager/pkg/listenhandler/v1_groupcalls.go` (handler-call swallows at lines ~55, ~108, ~144, ~180, ~216, ~258, ~294, ~330)
- Test: `bin-call-manager/pkg/listenhandler/v1_groupcalls_test.go`

Note: do **not** touch `bin-call-manager/pkg/listenhandler/main.go` (the central tail stays unwired — see spec).

- [ ] **Step 1: Add not-found tests**

Append to `bin-call-manager/pkg/listenhandler/v1_groupcalls_test.go`. The package already imports `sockhandler`, `callhandler`, `groupcallhandler`, `sock`, `uuid`, `gomock`; add the dbhandler import if not present:
`"monorepo/bin-call-manager/pkg/dbhandler"`.

```go
func Test_processV1GroupcallsID_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID
		setup   func(m *groupcallhandler.MockGroupcallHandler, id uuid.UUID)
	}{
		{
			name: "GET non-existent groupcall returns 404",
			request: &sock.Request{
				URI:    "/v1/groupcalls/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodGet,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
			setup: func(m *groupcallhandler.MockGroupcallHandler, id uuid.UUID) {
				m.EXPECT().Get(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)
			},
		},
		{
			name: "DELETE non-existent groupcall returns 404",
			request: &sock.Request{
				URI:    "/v1/groupcalls/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodDelete,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
			setup: func(m *groupcallhandler.MockGroupcallHandler, id uuid.UUID) {
				m.EXPECT().Delete(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)
			},
		},
		{
			name: "POST hangup non-existent groupcall returns 404",
			request: &sock.Request{
				URI:    "/v1/groupcalls/00000000-0000-0000-0000-000000000099/hangup",
				Method: sock.RequestMethodPost,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
			setup: func(m *groupcallhandler.MockGroupcallHandler, id uuid.UUID) {
				m.EXPECT().Hangingup(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)
			h := &listenHandler{
				sockHandler:      mockSock,
				callHandler:      mockCall,
				groupcallHandler: mockGroupcall,
			}
			tt.setup(mockGroupcall, tt.id)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test — expect FAIL**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping/bin-call-manager
go test ./pkg/listenhandler/ -run Test_processV1GroupcallsID_notFound -v
```
Expected: FAIL — all three subtests `expected: 404, got: 500`.

- [ ] **Step 3: Apply the canonical edit at the handler-call swallows**

In `bin-call-manager/pkg/listenhandler/v1_groupcalls.go`, change the handler-call-error `return simpleResponse(500), nil` to `return errorResponse(err), nil` in:

- `processV1GroupcallsGet` — after `h.groupcallHandler.List(...)` (~line 55)
- `processV1GroupcallsPost` — after `h.groupcallHandler.Start(...)` (~line 108)
- `processV1GroupcallsIDGet` — after `h.groupcallHandler.Get(...)` (~line 144)
- `processV1GroupcallsIDDelete` — after `h.groupcallHandler.Delete(...)` (~line 180)
- `processV1GroupcallsIDHangupPost` — after `h.groupcallHandler.Hangingup(...)` (~line 216)
- `processV1GroupcallsIDAnswerGroupcallIDPost` — after `h.groupcallHandler.AnswerGroupcall(...)` (~line 258)
- `processV1GroupcallsIDHangupGroupcallPost` — after `h.groupcallHandler.HangupGroupcall(...)` (~line 294)
- `processV1GroupcallsIDHangupCallPost` — after `h.groupcallHandler.HangupCall(...)` (~line 330)

Leave each post-`json.Marshal` `simpleResponse(500)` unchanged.

- [ ] **Step 4: Run the not-found test — expect PASS**

```bash
go test ./pkg/listenhandler/ -run Test_processV1GroupcallsID_notFound -v
```
Expected: PASS (all three subtests).

- [ ] **Step 5: Run the full listenhandler package tests — expect PASS**

```bash
go test ./pkg/listenhandler/ -v
```

- [ ] **Step 6: Run the mandatory verification workflow**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping/bin-call-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

- [ ] **Step 7: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping
git add bin-call-manager/pkg/listenhandler/v1_groupcalls.go bin-call-manager/pkg/listenhandler/v1_groupcalls_test.go bin-call-manager/go.mod bin-call-manager/go.sum
git commit -m "NOJIRA-Fix-listenhandler-error-mapping

- bin-call-manager: Map groupcall handler errors at the listenhandler call site (errorResponse) so non-existent groupcall IDs return 404 instead of 500 (#955)"
```

---

## Final verification (whole-program)

- [ ] **Confirm the four packages build and pass together**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Fix-listenhandler-error-mapping
for d in bin-conversation-manager bin-message-manager bin-number-manager bin-call-manager; do
  echo "=== $d ===" && (cd $d && go test ./pkg/listenhandler/...) || exit 1
done
```
Expected: each prints `ok`.

- [ ] **Confirm no central-tail / main.go changes leaked in**

```bash
git diff --name-only main...HEAD | grep 'listenhandler/main.go' && echo "UNEXPECTED main.go change" || echo "OK: no main.go changes"
```
Expected: `OK: no main.go changes`.

## Notes for PR / handoff

- The spec recommends **one PR per manager** for reviewable, revertible diffs. The
  four commits above are independent; they can be split onto per-manager branches
  or shipped as a single issue PR titled `NOJIRA-Fix-listenhandler-error-mapping`.
  Decide with the user before pushing. Do **not** merge without explicit
  authorization (squash merge only).
- **api-validator:** the 9 endpoints are already covered by the existing
  api-validator suite (the failing tests cited in #955). No new api-validator
  tests are required; the suite turns green after deploy. Verify post-deploy, not
  as part of this plan.
- **Out of scope (deferred to Phase 2/3):** other resources in these managers
  (call-manager `calls`/`confbridges`/`recordings`, conversation-manager
  `accounts`/conversation-messages, number-manager `metadata`/`available_numbers`),
  and call-manager central-tail wiring.
