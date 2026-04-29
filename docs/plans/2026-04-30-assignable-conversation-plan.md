# Assignable Conversation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wire manual conversation-to-agent assignment end-to-end so that admins can assign a conversation to a specific agent, and once assigned, new inbound messages on that conversation skip the registered flow trigger.

**Architecture:** Reuse the existing `commonidentity.Owner` (`OwnerType` + `OwnerID`) embedded in `Conversation` (no schema change). Introduce an `ExecuteMode` dispatch in conversation-manager's inbound message handlers that branches on the loaded `cv.OwnerType`. Expose assignment through the existing `PUT /v1.0/conversations/<id>` endpoint as a partial update on `owner_id`, with field-level permission gating in api-manager (admin/manager assigns; owning agent self-unassigns).

**Tech Stack:** Go 1.x, gomock, Squirrel SQL builder, RabbitMQ RPC, Sphinx RST documentation. Tests use table-driven gomock-based unit tests per the repo's testing convention.

**Reference design:** [`docs/plans/2026-04-30-assignable-conversation-design.md`](./2026-04-30-assignable-conversation-design.md) (already merged). Read it before starting; this plan implements it task-by-task.

---

## Phase 0: Pre-flight reading

Before touching any code, read these files end-to-end. They are the foundation for every task in this plan:

- `docs/plans/2026-04-30-assignable-conversation-design.md` — the spec being implemented
- `docs/conventions/error-handling.md` — especially §4.5 (`err<SpecificName>` for if-init blocks)
- `docs/conventions/testing.md` — gomock + table-driven patterns
- `bin-conversation-manager/CLAUDE.md` — service-specific conventions
- `bin-api-manager/CLAUDE.md` — if it exists; otherwise the root CLAUDE.md

Required-skills cross-references:
- @superpowers:test-driven-development — enforced for every task in this plan
- @superpowers:verification-before-completion — required before each commit

---

## Phase A: conversation-manager core changes

All file paths are relative to `bin-conversation-manager/`.

### Task A1: Refactor `MessageExecuteActiveflow` to error-only signature

**Why:** The existing function returns `(*activeflow.Activeflow, error)` but no caller uses the activeflow. Per design §4.1, change the signature to `error` only and rename to `executeActiveflow` (lower-case — internal helper). The existing log-line on `af.ID` moves inside the function.

**Files:**
- Modify: `pkg/conversationhandler/message.go:44-79` (function body, signature, rename to lowercase)
- Modify: `pkg/conversationhandler/hook.go:84-94` (caller updated to use new signature)
- Modify: `pkg/conversationhandler/message.go:140-150` (caller updated to use new signature; ALSO fix the existing buggy short-circuit — see Step 5b)
- Create: `pkg/conversationhandler/message_test.go` test function `Test_executeActiveflow` (no test currently exists for `MessageExecuteActiveflow`)
- Modify: `pkg/conversationhandler/hook_test.go` (update existing tests that mock the call to `MessageExecuteActiveflow`)

Note: `MessageExecuteActiveflow` is **not** on the `ConversationHandler` interface (only a method on the concrete `conversationHandler` struct). It does not appear in `mock_conversationhandler.go`. No interface modification or mock regeneration is required.

**Step 1: Read the current shape**

Run: `grep -n "MessageExecuteActiveflow" pkg/conversationhandler/*.go`
Expected: ~6 occurrences — definition (`message.go`), doc comment, log-field constant (if any), two call sites (`hook.go`, `message.go`), and a test name in `hook_test.go`. Mock file (`mock_conversationhandler.go`) does **not** contain it.

**Step 2: Create the new test for the helper**

There is no existing `Test_MessageExecuteActiveflow`. Create a new `Test_executeActiveflow` table-driven test in `pkg/conversationhandler/message_test.go` (alongside the existing `Test_MessageSend`). Mock `reqHandler.FlowV1ActiveflowCreate`, `reqHandler.FlowV1VariableSetVariable`, `reqHandler.FlowV1ActiveflowExecute` for the happy path. Add the "no flow configured" case:

```go
{
    name:    "no flow configured — returns nil error and does not call RPCs",
    flowID:  uuid.Nil,
    // no mockReq.EXPECT() calls
    wantErr: false,
},
```

**Step 3: Run the tests — expect FAIL**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_executeActiveflow -v`
Expected: FAIL — `executeActiveflow` (lowercase) does not yet exist; the existing function is named `MessageExecuteActiveflow` and has a different signature.

**Step 4: Refactor `MessageExecuteActiveflow` body into the new shape**

In `pkg/conversationhandler/message.go`, replace the function:

```go
// executeActiveflow creates and executes an activeflow for the given conversation, given a non-nil flowID.
// Returns nil if flowID is uuid.Nil (no flow configured for this conversation source).
func (h *conversationHandler) executeActiveflow(ctx context.Context, cv *conversation.Conversation, m *message.Message, flowID uuid.UUID) error {
    log := logrus.WithFields(logrus.Fields{
        "func":            "executeActiveflow",
        "conversation_id": cv.ID,
        "message_id":      m.ID,
        "flow_id":         flowID,
    })

    if flowID == uuid.Nil {
        log.Debugf("No flow configured. Skipping activeflow.")
        return nil
    }

    af, errCreate := h.reqHandler.FlowV1ActiveflowCreate(
        ctx,
        uuid.Nil,
        m.CustomerID,
        flowID,
        fmactiveflow.ReferenceTypeConversation,
        m.ConversationID,
        uuid.Nil,
    )
    if errCreate != nil {
        return errors.Wrapf(errCreate, "could not create activeflow. flow_id: %s", flowID)
    }

    if errVariable := h.setVariables(ctx, af.ID, cv, m); errVariable != nil {
        return errors.Wrapf(errVariable, "could not set variables. activeflow_id: %s", af.ID)
    }

    if errExecute := h.reqHandler.FlowV1ActiveflowExecute(ctx, af.ID); errExecute != nil {
        return errors.Wrapf(errExecute, "could not execute activeflow. activeflow_id: %s", af.ID)
    }

    log.WithField("activeflow_id", af.ID).Debugf("Executed activeflow.")
    return nil
}
```

**Step 5a: Update both call sites to drop the `*Activeflow` discard**

`pkg/conversationhandler/hook.go` — replace lines around 89–94:

```go
if errExecute := h.executeActiveflow(ctx, r.Conversation, r.Message, ac.MessageFlowID); errExecute != nil {
    return errors.Wrapf(errExecute, "could not execute activeflow. account_id: %s", ac.ID)
}
```

`pkg/conversationhandler/message.go` — replace lines around 146–150 inside `MessageEventReceived`:

```go
if errExecute := h.executeActiveflow(ctx, cv, m, num.MessageFlowID); errExecute != nil {
    return errors.Wrapf(errExecute, "could not execute activeflow. message_id: %s, number_id: %s", m.ID, num.ID)
}
```

**Step 5b: Fix the existing multi-target short-circuit bug in `MessageEventReceived`**

The current code at `message.go:140-143` has:

```go
if num.MessageFlowID == uuid.Nil {
    // nothing to do. has no message flow id
    return nil
}
```

This `return nil` exits the **entire function**, skipping any remaining targets in the `m.Targets` loop. Since `executeActiveflow` now handles the `flowID == uuid.Nil` case internally (returns nil without side effects), **delete this short-circuit block entirely** so the loop correctly iterates over remaining targets. This is a behavior fix the refactor enables.

**Step 6: Mock regeneration check (no expected changes)**

Run: `cd bin-conversation-manager && go generate ./...`
Expected: no diff in `pkg/conversationhandler/mock_conversationhandler.go` because `MessageExecuteActiveflow` was never on the interface. If `go generate` reports a diff, investigate before continuing.

**Step 7: Run the full test suite**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -v`
Expected: all tests pass, including the new "no flow configured" case.

**Step 8: Commit**

```bash
cd <worktree-root>
git add bin-conversation-manager/pkg/conversationhandler/message.go \
        bin-conversation-manager/pkg/conversationhandler/hook.go \
        bin-conversation-manager/pkg/conversationhandler/message_test.go \
        bin-conversation-manager/pkg/conversationhandler/hook_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-Conversation-agent-assignment executeActiveflow refactor

- bin-conversation-manager: Rename MessageExecuteActiveflow to internal executeActiveflow with error-only signature; the *Activeflow return value was unused at every call site
- bin-conversation-manager: Move the af.ID log line inside executeActiveflow; flowID == uuid.Nil now returns nil instead of being checked at each caller
- bin-conversation-manager: Delete the buggy multi-target short-circuit block in MessageEventReceived; the executeActiveflow helper now handles the no-flow-configured case so the loop iterates remaining targets correctly
- bin-conversation-manager: Update both call sites (hook.go, message.go) to drop the discarded return and use errExecute naming per docs/conventions/error-handling.md §4.5
EOF
)"
```

---

### Task A2: Add `ExecuteMode` type and `getExecuteMode`

**Why:** Design §4. Introduces the dispatch primitive without wiring it into the inbound paths yet.

**Files:**
- Modify: `pkg/conversationhandler/main.go` (add `ExecuteMode` type and constants)
- Create: `pkg/conversationhandler/execute_mode.go` (the `getExecuteMode` function)
- Create: `pkg/conversationhandler/execute_mode_test.go`

**Step 1: Write the failing test**

Create `pkg/conversationhandler/execute_mode_test.go`:

```go
package conversationhandler

import (
    "testing"

    "github.com/gofrs/uuid"

    commonidentity "monorepo/bin-common-handler/models/identity"
    "monorepo/bin-conversation-manager/models/conversation"
)

func Test_getExecuteMode(t *testing.T) {
    tests := []struct {
        name string
        cv   *conversation.Conversation
        want ExecuteMode
    }{
        {
            name: "unassigned conversation -> flow mode",
            cv: &conversation.Conversation{
                Owner: commonidentity.Owner{
                    OwnerType: commonidentity.OwnerTypeNone,
                    OwnerID:   uuid.Nil,
                },
            },
            want: ExecuteModeFlow,
        },
        {
            name: "agent owner with non-nil owner id -> agent mode",
            cv: &conversation.Conversation{
                Owner: commonidentity.Owner{
                    OwnerType: commonidentity.OwnerTypeAgent,
                    OwnerID:   uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
                },
            },
            want: ExecuteModeAgent,
        },
        {
            name: "agent owner with nil owner id -> flow mode (defensive against malformed state)",
            cv: &conversation.Conversation{
                Owner: commonidentity.Owner{
                    OwnerType: commonidentity.OwnerTypeAgent,
                    OwnerID:   uuid.Nil,
                },
            },
            want: ExecuteModeFlow,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            h := &conversationHandler{}
            got := h.getExecuteMode(tt.cv)
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

**Step 2: Run the test to verify it fails**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_getExecuteMode -v`
Expected: FAIL — `ExecuteMode` undefined; `getExecuteMode` undefined.

**Step 3: Add the type and constants to `main.go`**

In `pkg/conversationhandler/main.go`, after the existing variable section:

```go
// ExecuteMode defines how an inbound message on a conversation should be processed.
type ExecuteMode string

const (
    ExecuteModeNone  ExecuteMode = ""      // reserved; not currently produced by getExecuteMode
    ExecuteModeAgent ExecuteMode = "agent" // conversation owned by an agent — skip flow trigger
    ExecuteModeFlow  ExecuteMode = "flow"  // default — trigger the registered flow per cv.Type
)
```

**Step 4: Add `getExecuteMode` in a new file**

Create `pkg/conversationhandler/execute_mode.go`:

```go
package conversationhandler

import (
    "github.com/gofrs/uuid"

    commonidentity "monorepo/bin-common-handler/models/identity"
    "monorepo/bin-conversation-manager/models/conversation"
)

// getExecuteMode reads the conversation's Owner snapshot and returns the dispatch mode.
// See docs/plans/2026-04-30-assignable-conversation-design.md §3.1: callers MUST NOT re-fetch
// the Conversation in the dispatch path; the snapshot already loaded by the inbound handler is authoritative.
func (h *conversationHandler) getExecuteMode(cv *conversation.Conversation) ExecuteMode {
    if cv.OwnerType == commonidentity.OwnerTypeAgent && cv.OwnerID != uuid.Nil {
        return ExecuteModeAgent
    }
    return ExecuteModeFlow
}
```

**Step 5: Run the test to verify it passes**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_getExecuteMode -v`
Expected: PASS — three test cases.

**Step 6: Commit**

```bash
git add bin-conversation-manager/pkg/conversationhandler/main.go \
        bin-conversation-manager/pkg/conversationhandler/execute_mode.go \
        bin-conversation-manager/pkg/conversationhandler/execute_mode_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-Conversation-agent-assignment add ExecuteMode and getExecuteMode

- bin-conversation-manager: Add ExecuteMode type with constants None/Agent/Flow per design §4
- bin-conversation-manager: Add getExecuteMode helper that branches on cv.OwnerType / cv.OwnerID; reserved no-op None value for future owner types
- bin-conversation-manager: Add table-driven test covering unassigned, assigned-to-agent, and defensive owner-id-nil cases
EOF
)"
```

---

### Task A3: Add `runExecuteModeAgent` no-op handler

**Why:** Design §4. The agent UI receives the new message via the existing `message_created` event; this handler just logs and returns nil.

**Files:**
- Modify: `pkg/conversationhandler/execute_mode.go`
- Modify: `pkg/conversationhandler/execute_mode_test.go`

**Step 1: Write the failing test**

Add to `pkg/conversationhandler/execute_mode_test.go`:

```go
func Test_runExecuteModeAgent_isNoop(t *testing.T) {
    mc := gomock.NewController(t)
    defer mc.Finish()

    mockReq := requesthandler.NewMockRequestHandler(mc)
    h := &conversationHandler{reqHandler: mockReq}

    cv := &conversation.Conversation{
        Identity: commonidentity.Identity{
            ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
            CustomerID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
        },
    }
    m := &message.Message{
        Identity: commonidentity.Identity{
            ID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
        },
    }

    // mockReq.EXPECT() — no expectations: any RPC call will fail the test.

    err := h.runExecuteModeAgent(context.Background(), cv, m)
    if err != nil {
        t.Errorf("expected nil error, got: %v", err)
    }
}
```

(Add the necessary imports for `gomock`, `requesthandler`, `context`, `message`.)

**Step 2: Run the test to verify it fails**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_runExecuteModeAgent -v`
Expected: FAIL — `runExecuteModeAgent` undefined.

**Step 3: Implement the no-op**

Append to `pkg/conversationhandler/execute_mode.go`:

```go
// runExecuteModeAgent handles inbound messages on conversations owned by an agent.
// The agent UI learns of new messages via the existing `message_created` event filtered on cv.OwnerID.
// No new event is published; no flow is triggered. Logging only.
func (h *conversationHandler) runExecuteModeAgent(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
    log := logrus.WithFields(logrus.Fields{
        "func":            "runExecuteModeAgent",
        "conversation_id": cv.ID,
        "message_id":      m.ID,
        "owner_id":        cv.OwnerID,
    })
    log.Debugf("Conversation owned by agent. Skipping flow trigger.")
    return nil
}
```

(Add imports: `context`, `logrus`, `monorepo/bin-conversation-manager/models/message`.)

**Step 4: Run the test to verify it passes**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_runExecuteModeAgent -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add bin-conversation-manager/pkg/conversationhandler/execute_mode.go \
        bin-conversation-manager/pkg/conversationhandler/execute_mode_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-Conversation-agent-assignment add runExecuteModeAgent

- bin-conversation-manager: Add runExecuteModeAgent — no-op handler that logs and returns nil; the agent UI receives the message via the existing message_created event
- bin-conversation-manager: Add test asserting no flow RPCs are invoked when the handler runs
EOF
)"
```

---

### Task A4: Add `runExecuteModeFlow` and per-type runners

**Why:** Design §4. Per-conversation-type dispatch (LINE vs Message), each runner fetches its source (account / number) and calls `executeActiveflow`.

**Files:**
- Modify: `pkg/conversationhandler/execute_mode.go`
- Modify: `pkg/conversationhandler/execute_mode_test.go`

**Step 1: Write the failing tests**

Add to `pkg/conversationhandler/execute_mode_test.go`:

```go
func Test_runExecuteModeFlowLine(t *testing.T) {
    accountID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
    flowID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
    convID := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
    custID := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
    msgID := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")
    afID := uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666")

    tests := []struct {
        name     string
        cv       *conversation.Conversation
        m        *message.Message
        setup    func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler)
        wantErr  bool
    }{
        {
            name: "valid line conversation with flow id -> executeActiveflow called",
            cv: &conversation.Conversation{
                Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
                Type:      conversation.TypeLine,
                AccountID: accountID,
            },
            m: &message.Message{
                Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
                ConversationID: convID,
            },
            setup: func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {
                mockAccount.EXPECT().Get(gomock.Any(), accountID).Return(&account.Account{
                    Identity:      commonidentity.Identity{ID: accountID, CustomerID: custID},
                    MessageFlowID: flowID,
                }, nil)
                mockReq.EXPECT().FlowV1ActiveflowCreate(
                    gomock.Any(), uuid.Nil, custID, flowID,
                    fmactiveflow.ReferenceTypeConversation, convID, uuid.Nil,
                ).Return(&fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: afID}}, nil)
                mockReq.EXPECT().FlowV1VariableSetVariable(gomock.Any(), afID, gomock.Any()).Return(nil)
                mockReq.EXPECT().FlowV1ActiveflowExecute(gomock.Any(), afID).Return(nil)
            },
            wantErr: false,
        },
        {
            name: "line conversation with nil account id -> short-circuit, no fetch",
            cv: &conversation.Conversation{
                Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
                Type:      conversation.TypeLine,
                AccountID: uuid.Nil,
            },
            m: &message.Message{
                Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
                ConversationID: convID,
            },
            setup:   func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {},
            wantErr: false,
        },
        {
            name: "line conversation, account fetch fails -> error wrapped and returned",
            cv: &conversation.Conversation{
                Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
                Type:      conversation.TypeLine,
                AccountID: accountID,
            },
            m: &message.Message{
                Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
                ConversationID: convID,
            },
            setup: func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {
                mockAccount.EXPECT().Get(gomock.Any(), accountID).Return(nil, errors.New("db down"))
            },
            wantErr: true,
        },
        {
            name: "line conversation, account has no flow id -> no activeflow created, no error",
            cv: &conversation.Conversation{
                Identity:  commonidentity.Identity{ID: convID, CustomerID: custID},
                Type:      conversation.TypeLine,
                AccountID: accountID,
            },
            m: &message.Message{
                Identity:       commonidentity.Identity{ID: msgID, CustomerID: custID},
                ConversationID: convID,
            },
            setup: func(mockAccount *accounthandler.MockAccountHandler, mockReq *requesthandler.MockRequestHandler) {
                mockAccount.EXPECT().Get(gomock.Any(), accountID).Return(&account.Account{
                    Identity:      commonidentity.Identity{ID: accountID, CustomerID: custID},
                    MessageFlowID: uuid.Nil,
                }, nil)
            },
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mc := gomock.NewController(t)
            defer mc.Finish()

            mockAccount := accounthandler.NewMockAccountHandler(mc)
            mockReq := requesthandler.NewMockRequestHandler(mc)
            h := &conversationHandler{
                accountHandler: mockAccount,
                reqHandler:     mockReq,
            }
            tt.setup(mockAccount, mockReq)

            err := h.runExecuteModeFlowLine(context.Background(), tt.cv, tt.m)
            if (err != nil) != tt.wantErr {
                t.Errorf("got err = %v, wantErr = %v", err, tt.wantErr)
            }
        })
    }
}
```

Add an analogous `Test_runExecuteModeFlowMessage` with `cv.Self.Target` and `NumberGet` mocking.

Add a top-level `Test_runExecuteModeFlow` switch test:

```go
func Test_runExecuteModeFlow_typeDispatch(t *testing.T) {
    tests := []struct {
        name     string
        cvType   conversation.Type
        wantNoop bool // true means dispatch should hit the default arm and return nil without calling sub-runners
    }{
        {name: "line type", cvType: conversation.TypeLine, wantNoop: false},
        {name: "message type", cvType: conversation.TypeMessage, wantNoop: false},
        {name: "unsupported type", cvType: "unknown", wantNoop: true},
    }
    // For wantNoop=false cases, we'd need full mocks; for wantNoop=true the handler returns nil with no mock setup.
    // For brevity, exercise wantNoop=true here. Per-type runners are tested separately.
    for _, tt := range tests {
        if !tt.wantNoop {
            continue
        }
        t.Run(tt.name, func(t *testing.T) {
            h := &conversationHandler{}
            cv := &conversation.Conversation{Type: tt.cvType}
            m := &message.Message{}
            err := h.runExecuteModeFlow(context.Background(), cv, m)
            if err != nil {
                t.Errorf("expected nil for unsupported type, got: %v", err)
            }
        })
    }
}
```

**Step 2: Run the tests to verify they fail**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_runExecuteModeFlow -v`
Expected: FAIL — `runExecuteModeFlow`, `runExecuteModeFlowLine`, `runExecuteModeFlowMessage` undefined.

**Step 3: Implement the dispatcher and per-type runners**

Append to `pkg/conversationhandler/execute_mode.go`:

```go
// runExecuteModeFlow dispatches by conversation type. Each per-type runner fetches the type-specific
// flow source (account for LINE, number for SMS) and calls executeActiveflow with the resolved flow id.
func (h *conversationHandler) runExecuteModeFlow(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
    switch cv.Type {
    case conversation.TypeLine:
        return h.runExecuteModeFlowLine(ctx, cv, m)
    case conversation.TypeMessage:
        return h.runExecuteModeFlowMessage(ctx, cv, m)
    default:
        logrus.WithFields(logrus.Fields{
            "func":            "runExecuteModeFlow",
            "conversation_id": cv.ID,
            "type":            cv.Type,
        }).Debugf("Unsupported conversation type for flow execution. Skipping.")
        return nil
    }
}

func (h *conversationHandler) runExecuteModeFlowLine(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
    if cv.AccountID == uuid.Nil {
        return nil
    }
    ac, errGet := h.accountHandler.Get(ctx, cv.AccountID)
    if errGet != nil {
        return errors.Wrapf(errGet, "could not get account. account_id: %s", cv.AccountID)
    }
    if errExecute := h.executeActiveflow(ctx, cv, m, ac.MessageFlowID); errExecute != nil {
        return errors.Wrapf(errExecute, "could not execute activeflow. account_id: %s", ac.ID)
    }
    return nil
}

func (h *conversationHandler) runExecuteModeFlowMessage(ctx context.Context, cv *conversation.Conversation, m *message.Message) error {
    num, errGet := h.NumberGet(ctx, cv.Self.Target)
    if errGet != nil {
        return errors.Wrapf(errGet, "could not get number. number: %s", cv.Self.Target)
    }
    if errExecute := h.executeActiveflow(ctx, cv, m, num.MessageFlowID); errExecute != nil {
        return errors.Wrapf(errExecute, "could not execute activeflow. number_id: %s", num.ID)
    }
    return nil
}
```

**Step 4: Run the tests to verify they pass**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_runExecuteModeFlow -v`
Expected: PASS — all three test functions.

**Step 5: Commit**

```bash
git add bin-conversation-manager/pkg/conversationhandler/execute_mode.go \
        bin-conversation-manager/pkg/conversationhandler/execute_mode_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-Conversation-agent-assignment add flow-mode dispatchers

- bin-conversation-manager: Add runExecuteModeFlow that switches on cv.Type and delegates to runExecuteModeFlowLine (account.MessageFlowID) or runExecuteModeFlowMessage (number.MessageFlowID); unsupported types are a logged no-op
- bin-conversation-manager: Each per-type runner fetches its source and calls the shared executeActiveflow helper; matches the dispatch shape from design §4
- bin-conversation-manager: Add tests covering happy path, nil account id, fetch error, and missing flow id for the LINE runner; analogous tests for the message runner; a default-arm test for the top-level dispatcher
EOF
)"
```

---

### Task A5: Wire the dispatch into `hookLine`

**Why:** Design §4. Existing call to `MessageExecuteActiveflow` is replaced with the mode dispatch. Behavior change: when `cv` is owned by an agent, no flow is triggered.

**Files:**
- Modify: `pkg/conversationhandler/hook.go`
- Modify: `pkg/conversationhandler/hook_test.go`

**Step 1: Add a failing test for the assigned-conversation case**

In `pkg/conversationhandler/hook_test.go`, add a table case to the existing `Test_hookLine` (or analogous) where `r.Conversation.OwnerType = OwnerTypeAgent` and `r.Conversation.OwnerID != uuid.Nil`. Set up the mocks so that `FlowV1ActiveflowCreate` is **not** called. Assert no error.

Also keep an existing case where the conversation is unassigned and `FlowV1ActiveflowCreate` IS called — this confirms behavior is preserved for the default path.

**Step 2: Run the tests to verify the new case fails**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_hookLine -v`
Expected: FAIL — the assigned-conversation case will incorrectly call `FlowV1ActiveflowCreate` because the dispatch isn't wired in yet.

**Step 3: Replace the direct flow trigger with the mode dispatch**

In `pkg/conversationhandler/hook.go`, replace the block that calls `executeActiveflow` (the renamed `MessageExecuteActiveflow` from Task A1) with:

```go
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
```

Remove the now-redundant `if ac.MessageFlowID == uuid.Nil { return nil }` short-circuit that lives **before the `for _, r := range results` loop** in `hookLine` (around lines 79-81 today) — `runExecuteModeFlowLine` handles that case internally.

**Note on per-message account fetch:** the new `runExecuteModeFlowLine` re-fetches the account by `cv.AccountID` even though `hookLine` already has `ac` loaded from the URL. Since `cv.AccountID` is the same account that received the webhook (by construction — the conversation was created with that account), the re-fetch is wasteful but behaviorally equivalent. The cost is one extra cache/db hit per inbound LINE message; the design accepts this as the trade for keeping `runExecuteModeFlowLine`'s signature `(cv, m)` consistent with the SMS runner. If profiling later shows this matters, `ac` can be threaded through; for now, simplicity wins.

**Step 4: Run the tests to verify they pass**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_hookLine -v`
Expected: PASS — both the assigned-conversation case and the unassigned-conversation case.

**Step 5: Commit**

```bash
git add bin-conversation-manager/pkg/conversationhandler/hook.go \
        bin-conversation-manager/pkg/conversationhandler/hook_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-Conversation-agent-assignment wire dispatch into hookLine

- bin-conversation-manager: Replace direct executeActiveflow call in hookLine with the ExecuteMode dispatch; assigned conversations now skip flow trigger as specified in design §4
- bin-conversation-manager: Add table case for the assigned-conversation path that asserts no flow RPCs are made; existing unassigned-conversation behavior preserved
EOF
)"
```

---

### Task A6: Wire the dispatch into `MessageEventReceived`

**Why:** Design §4. SMS path mirrors the LINE path.

**Files:**
- Modify: `pkg/conversationhandler/message.go`
- Modify: `pkg/conversationhandler/message_test.go` or `event_test.go` (whichever holds the existing tests)

**Step 1: Add a failing test for the assigned-conversation SMS case**

Mirror Task A5 step 1, but for the SMS path. The test asserts no flow RPCs are made when `cv.OwnerType == OwnerTypeAgent`.

**Step 2: Run the tests to verify the new case fails**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_MessageEventReceived -v`
Expected: FAIL on the new case.

**Step 3: Replace the direct flow trigger with the mode dispatch**

In `pkg/conversationhandler/message.go::MessageEventReceived`, replace the `NumberGet` + `MessageFlowID` check + `executeActiveflow` block with:

```go
mode := h.getExecuteMode(cv)
switch mode {
case ExecuteModeAgent:
    if errAgent := h.runExecuteModeAgent(ctx, cv, m); errAgent != nil {
        return errors.Wrapf(errAgent, "could not run agent mode. message_id: %s", m.ID)
    }
case ExecuteModeFlow:
    if errFlow := h.runExecuteModeFlow(ctx, cv, m); errFlow != nil {
        return errors.Wrapf(errFlow, "could not run flow mode. message_id: %s", m.ID)
    }
case ExecuteModeNone:
    // reserved; no-op
default:
    return fmt.Errorf("unknown execute mode: %s", mode)
}
```

The `NumberGet` call moves into `runExecuteModeFlowMessage` (already implemented in Task A4).

**Step 4: Run the tests to verify they pass**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_MessageEventReceived -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add bin-conversation-manager/pkg/conversationhandler/message.go \
        bin-conversation-manager/pkg/conversationhandler/message_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-Conversation-agent-assignment wire dispatch into MessageEventReceived

- bin-conversation-manager: Replace direct flow-trigger block in MessageEventReceived with the ExecuteMode dispatch; assigned SMS conversations now skip flow trigger
- bin-conversation-manager: Move NumberGet + flow id resolution into runExecuteModeFlowMessage; the inbound entry point becomes a thin dispatch
- bin-conversation-manager: Add test case asserting no flow RPCs are made on assigned conversations
EOF
)"
```

---

### Task A7: `owner_type` derivation in `Update`

**Why:** Design §5.3. Conversation-manager always derives `owner_type` from `owner_id`, regardless of caller-supplied value.

**Files:**
- Modify: `pkg/conversationhandler/db.go` (or wherever `Update` lives — likely the same file as `Create`)
- Modify: `pkg/conversationhandler/db_test.go` (or appropriate test file)

**Step 1: Locate the Update function**

Run: `grep -n "func (h \*conversationHandler) Update" pkg/conversationhandler/`
Expected: one location.

**Step 2: Write failing tests for derivation**

Add table cases:
- input fields: `{owner_id: <non-nil-uuid>}` → expect both `owner_id` AND `owner_type=agent` written to DB
- input fields: `{owner_id: uuid.Nil}` → expect `owner_id=uuid.Nil` AND `owner_type=""` written to DB
- input fields: `{owner_id: <uuid>, owner_type: "something-else"}` → expect server-derived `owner_type=agent` written, ignoring caller-supplied value
- input fields: `{name: "x"}` (no owner_id present) → expect no owner_type or owner_id written

**Step 3: Run the tests to verify they fail**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_Update -v`
Expected: FAIL — derivation not implemented.

**Step 4: Implement derivation**

In `Update()`, before the DB call, inspect the `fields` map:

```go
if v, ok := fields[conversation.FieldOwnerID]; ok {
    ownerID, okType := v.(uuid.UUID)
    if !okType {
        return nil, cerrors.InvalidArgument(fmt.Sprintf("invalid owner_id type: %T", v))
    }
    if ownerID == uuid.Nil {
        fields[conversation.FieldOwnerType] = commonidentity.OwnerTypeNone
    } else {
        fields[conversation.FieldOwnerType] = commonidentity.OwnerTypeAgent
    }
}
```

Apply this **before** any validation step (Task A8) and before the DB write.

**Note on type-assertion handling:** `ConvertStringMapToFieldMap` (`models/conversation/convert.go`) already converts string UUIDs to `uuid.UUID` via reflection, so a type-assertion failure should be unreachable in practice. But the explicit type check + `cerrors.InvalidArgument` rejection is defensive — silently treating a malformed input as `uuid.Nil` would be an unintended unassignment.

**Note on error type:** Use `cerrors.InvalidArgument(...)` (from `bin-common-handler/pkg/cerrors`) so that api-manager's edge can map this to **400** per the design's contract. Plain `fmt.Errorf` would surface as 500. Verify the exact constructor name in the cerrors package before implementing.

**Step 5: Run the tests to verify they pass**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_Update -v`
Expected: PASS — all four cases.

**Step 6: Commit**

```bash
git add bin-conversation-manager/pkg/conversationhandler/db.go \
        bin-conversation-manager/pkg/conversationhandler/db_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-Conversation-agent-assignment owner_type derivation in Update

- bin-conversation-manager: Derive owner_type from owner_id whenever owner_id is present in the partial-update fields; non-nil uuid yields owner_type=agent, nil uuid yields owner_type=""
- bin-conversation-manager: Always override caller-supplied owner_type with the derived value, matching design §5.3 — clients never need to send owner_type
- bin-conversation-manager: Add table cases covering assignment, unassignment, override, and no-owner-id payloads
EOF
)"
```

---

### Task A8: Agent existence and customer-match validation

**Why:** Design §5.4. Reject the update when `owner_type=agent` and the agent doesn't exist or belongs to a different customer.

**Implementation note — divergence from design §5.4 error messages:** The design specified two distinct error strings ("agent not found" vs "agent customer mismatch"). However, `bin-agent-manager`'s current `processV1AgentsIDGet` (`pkg/listenhandler/v1_agents.go`) collapses `dbhandler.ErrNotFound` into HTTP 500 with no typed envelope, so `errors.Is(err, ErrNotFound)` cannot reliably distinguish "not found" from "transport error" across the RPC boundary. Rather than introducing a coupled change in agent-manager's GET path, we use **`AgentV1AgentGets`** (the list endpoint, filterable by `id` + `customer_id`) and infer existence-and-customer-match from the result length: empty list → reject; non-empty list → proceed. This collapses the two cases into a single combined 400 ("could not validate agent. owner_id: <uuid>") with the same canonical-format prefix. The "not found" / "customer mismatch" semantic distinction was always cosmetic; the combined rejection still tells the client their `owner_id` is invalid for this conversation. If a future need requires the distinction, agent-manager can be updated to surface a typed 404 and this validation can be reworked to use `AgentV1AgentGet`.

**Files:**
- Modify: `pkg/conversationhandler/db.go` (Update function — adds explicit `cv` fetch + validation)
- Modify: `pkg/conversationhandler/db_test.go`

**Step 1: Verify `AgentV1AgentGets` (or equivalent list-by-filter RPC) exists**

Run: `grep -n "AgentV1AgentGets\|AgentV1AgentList" bin-common-handler/pkg/requesthandler/agent_*.go`

Inspect the actual signature and the list filter shape. If only `AgentV1AgentGets` (or an equivalent) accepts a filter map, use it with `id=<owner_id>` AND `customer_id=<cv.CustomerID>` AND `deleted=false`. If the available list endpoint cannot filter by id at the RPC layer, fall back to `AgentV1AgentGet(ctx, ownerID)` followed by `ag.CustomerID != cv.CustomerID` — both cases (RPC error or customer mismatch) reject with the same combined error, which still satisfies the implementation note above.

**Step 2: Add failing tests**

Add table cases (assumes the chosen RPC is `AgentV1AgentGets` with a filter):

- valid agent (list returns single match) → DB write occurs, no error
- agent doesn't exist or belongs to different customer (list returns empty) → reject with `cerrors.InvalidArgument("could not validate agent. owner_id: <uuid>")`; DB not written; no event fires
- agent-manager RPC fails (transport error) → wrapped error returned; DB not written; no event fires (api-manager surfaces 500)
- unassign (`owner_id=uuid.Nil`) → no agent RPC attempted; DB write occurs
- `Update` is called for a conversation where the conversation-manager `Get(ctx, id)` (the explicit pre-fetch added in step 3) fails → error wrapped and returned; no agent RPC attempted

**Step 3: Run the tests to verify they fail**

Expected: FAIL — validation not implemented; `cv` not loaded in `Update`.

**Step 4: Implement validation in `Update`**

The current `Update` signature is `(ctx, id, fields) -> (*conversation.Conversation, error)` and it does **not** load the conversation first. Validation needs `cv.CustomerID`, so add an explicit pre-fetch.

After Task A7's derivation block, before the DB write:

```go
if v, ok := fields[conversation.FieldOwnerID]; ok {
    ownerID, okType := v.(uuid.UUID)
    if !okType {
        return nil, cerrors.InvalidArgument(fmt.Sprintf("invalid owner_id type: %T", v))
    }
    if ownerID != uuid.Nil {
        // Need cv.CustomerID for the validation filter — fetch the existing conversation.
        cv, errGet := h.Get(ctx, id)
        if errGet != nil {
            return nil, errors.Wrapf(errGet, "could not load conversation for validation. id: %s", id)
        }

        // Validate agent existence + same-customer constraint via list filter.
        agents, errList := h.reqHandler.AgentV1AgentGets(ctx, /* filters: id=ownerID, customer_id=cv.CustomerID, deleted=false */)
        if errList != nil {
            return nil, errors.Wrapf(errList, "could not validate agent. owner_id: %s", ownerID)
        }
        if len(agents) == 0 {
            return nil, cerrors.InvalidArgument(fmt.Sprintf("could not validate agent. owner_id: %s", ownerID))
        }
    }
}
```

**Implementation hints:**
- Verify the exact `AgentV1AgentGets` signature and filter format before writing the call. The exact filter constants (`id`, `customer_id`, `deleted`) may differ — check `bin-agent-manager/pkg/listenhandler/v1_agents.go::processV1AgentsGet` for the supported filter keys.
- `cerrors.InvalidArgument` is the typed envelope api-manager uses to surface 400. Double-check the constructor name in `bin-common-handler/pkg/cerrors/`.
- `h.Get(ctx, id)` already exists on the handler and uses cache + DB — it is the correct way to load `cv` here.

**Step 5: Run the tests to verify they pass**

Run: `cd bin-conversation-manager && go test ./pkg/conversationhandler/ -run Test_Update -v`
Expected: PASS — all five cases.

**Step 6: Commit**

```bash
git add bin-conversation-manager/pkg/conversationhandler/db.go \
        bin-conversation-manager/pkg/conversationhandler/db_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-Conversation-agent-assignment agent validation on assignment

- bin-conversation-manager: Validate agent existence + same-customer constraint as a pre-check in Update before any DB write; failures abort with no event fired
- bin-conversation-manager: Use AgentV1AgentGets list-with-filter to detect existence + customer match in one RPC; bin-agent-manager does not surface a typed 404 today, so the not-found and customer-mismatch cases collapse into a single combined "could not validate agent" rejection — semantic distinction was cosmetic
- bin-conversation-manager: Use cerrors.InvalidArgument typed envelope so api-manager surfaces 400 per design §5.4
- bin-conversation-manager: Add explicit cv fetch via h.Get before validation — Update did not previously load the conversation
- bin-conversation-manager: Unassignment (owner_id=uuid.Nil) skips validation entirely
- bin-conversation-manager: Add table cases for valid, invalid (combined), RPC-failure, unassign, and conversation-not-found paths
EOF
)"
```

---

### Phase A verification

Run the full conversation-manager verification workflow:

```bash
cd bin-conversation-manager && \
  go mod tidy && \
  go mod vendor && \
  go generate ./... && \
  go test ./... && \
  golangci-lint run -v --timeout 5m
```

All five steps must pass before moving to Phase B. If any step fails, fix and re-run.

---

## Phase B: api-manager exposure

### Task B1: Field-level permission gate for `owner_id`

**Why:** Design §5.2. Today's `ConversationUpdate` only checks admin/manager. Owning-agent self-unassign is net-new.

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/conversation.go::ConversationUpdate`
- Modify: `bin-api-manager/pkg/servicehandler/conversation_test.go`

**Step 1: Identify the current shape**

Run: `grep -n "func (h \*serviceHandler) ConversationUpdate" bin-api-manager/pkg/servicehandler/conversation.go`
Read the function. Note the current single permission check on `PermissionCustomerAdmin | PermissionCustomerManager`.

**Step 2: Write failing tests for the new permission rules**

For each row in the design §5.2 permission table, add a test case:

| Auth | Payload | Expected |
|---|---|---|
| admin/manager | `{owner_id: <uuid>}` | forwards to conversation-manager; 200 |
| admin/manager | `{owner_id: nil-UUID}` | forwards; 200 |
| admin/manager | `{name: "x"}` | forwards; 200 |
| owning agent | `{owner_id: nil-UUID}` (self-unassign) | forwards; 200 |
| owning agent | `{owner_id: <other-uuid>}` (try to assign) | 403; not forwarded |
| owning agent | `{owner_id: <self-uuid>}` (try to assign self) | 403; not forwarded |
| owning agent | `{name: "x"}` | 403; not forwarded |
| owning agent | `{owner_id: nil-UUID, name: "x"}` (combined; name is denied) | 403; not forwarded |
| non-owning agent | `{owner_id: nil-UUID}` | 403; not forwarded |
| non-owning agent | `{name: "x"}` | 403; not forwarded |

**Step 3: Run the tests to verify they fail**

Run: `cd bin-api-manager && go test ./pkg/servicehandler/ -run Test_ConversationUpdate -v`
Expected: FAIL — many cases will currently 403 the admin path or 200 the agent path incorrectly.

**Step 4: Implement the per-field gate**

The new logic. Note that `ConversationUpdate` already loads `c, err := h.conversationGet(ctx, conversationID)` early (around line 154 today) — the gate runs **after** that load so `c.OwnerID` is available, and **before** the call to `reqHandler.ConversationV1ConversationUpdate`. By the time `ConversationUpdate` is reached, `data` has already been transformed from raw JSON into `fields map[cvconversation.Field]any` (server transforms in `bin-api-manager/server/conversations.go`).

```go
// Already-existing guard from today's code; kept explicit here:
if a.IsDirect() {
    return ErrDirectAccessNotSupported
}

// Existing path: admin/manager retains full access.
if h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
    // proceed with the existing forward to conversation-manager
} else {
    // Net-new agent self-unassign path — see design §5.2.
    // Only an authenticated agent identity can take this path.
    if !a.IsAgent() || a.Agent == nil {
        return ErrPermissionDenied
    }
    // Must be the currently-owning agent.
    if c.OwnerType != commonidentity.OwnerTypeAgent || c.OwnerID != a.Agent.ID {
        return ErrPermissionDenied
    }
    // Payload must be EXACTLY a self-unassign — one key, owner_id, value uuid.Nil.
    if !payloadIsExactlySelfUnassign(fields) {
        return ErrPermissionDenied
    }
    // Permitted: fall through to forward.
}
```

Where `payloadIsExactlySelfUnassign(fields map[cvconversation.Field]any) bool` is true iff:
- `len(fields) == 1`, AND
- `fields[cvconversation.FieldOwnerID]` exists and equals `uuid.Nil` (compare as `uuid.UUID`, not as a string).

Any second key in the map (even `name=""`) makes this false → permission denied for the agent. Use the actual error type the package returns for permission denials (`ErrPermissionDenied` or equivalent — verify the symbol in `bin-api-manager/pkg/servicehandler/`).

**Implementation note:** The `IsAgent` / `Agent` shape on `auth.AuthIdentity` is verified to support `TypeAgent`, `TypeAccesskey`, `TypeDirect`. The guard above prevents an accesskey-only caller (with customer scope but no agent identity) from triggering a nil-pointer panic when the code reads `a.Agent.ID`.

**Step 5: Run the tests to verify they pass**

Run: `cd bin-api-manager && go test ./pkg/servicehandler/ -run Test_ConversationUpdate -v`
Expected: PASS — all permission table cases.

**Step 6: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/conversation.go \
        bin-api-manager/pkg/servicehandler/conversation_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-Conversation-agent-assignment field-level permission gate

- bin-api-manager: Extend ConversationUpdate to permit owning agents to self-unassign by sending {owner_id: nil-UUID}; admin/manager retain unrestricted access; combined or non-self-unassign payloads from agents are rejected with 403
- bin-api-manager: Introduce payloadIsExactlySelfUnassign helper; the rule is intentionally strict — any second field in the payload disqualifies the agent path
- bin-api-manager: Add tests for every cell in the design §5.2 permission table
EOF
)"
```

---

### Task B2: Verify body decode preserves empty strings

**Why:** Design §5.1 — confirm that `PutConversationsIdJSONBody` round-trip preserves `{"name": ""}` end-to-end.

**Files:**
- Modify: `bin-api-manager/server/conversations_test.go` (or create if it doesn't exist)

**Step 1: Write a test asserting the empty-string survives round-trip**

```go
func Test_PutConversationsId_emptyStringPreserved(t *testing.T) {
    body := []byte(`{"name": ""}`)
    var req openapi_server.PutConversationsIdJSONBody
    if errBind := json.Unmarshal(body, &req); errBind != nil {
        t.Fatalf("unmarshal failed: %v", errBind)
    }
    raw, errFilter := structToFilteredMap(req)
    if errFilter != nil {
        t.Fatalf("filter failed: %v", errFilter)
    }
    if v, ok := raw["name"]; !ok {
        t.Errorf("expected name key in filtered map, got: %v", raw)
    } else if v != "" {
        t.Errorf("expected name='', got: %q", v)
    }
}
```

**Step 2: Run the test**

Run: `cd bin-api-manager && go test ./server/ -run Test_PutConversationsId_emptyStringPreserved -v`
Expected: PASS — pointer-with-omitempty already preserves the semantic. If it fails, that's a regression to address now.

**Step 3: Commit (if test passed without code change)**

```bash
git add bin-api-manager/server/conversations_test.go
git commit -m "$(cat <<'EOF'
NOJIRA-Conversation-agent-assignment empty-string round-trip test

- bin-api-manager: Add regression test asserting that {"name": ""} survives the PutConversationsIdJSONBody decode + structToFilteredMap pipeline; pointer-with-omitempty preserves the absent-vs-empty distinction
EOF
)"
```

---

### Phase B verification

```bash
cd bin-api-manager && \
  go mod tidy && \
  go mod vendor && \
  go generate ./... && \
  go test ./... && \
  golangci-lint run -v --timeout 5m
```

All five steps must pass before moving to Phase C.

---

## Phase C: RST documentation

All paths under `bin-api-manager/docsdev/source/`.

**Important:** Per the root CLAUDE.md "RST docs sync" rule, **RST source and rebuilt HTML must be committed together**. Do **not** commit RST source in C1, C2, or C3 individually. Instead, accumulate the RST source edits across C1–C3 (no commits), then in C4 do a clean HTML rebuild and commit everything (RST + HTML + changelog) in one commit. This is the only way to keep `docsdev/build/` in sync with `docsdev/source/` per repo convention.

### Task C1: Expand `owner_type` / `owner_id` field descriptions

**File:** `conversation_struct_conversation.rst`

**Step 1:** Locate the existing field descriptions at lines 36-37 (`owner_type` and `owner_id`).

**Step 2:** Replace the brief descriptions with expanded versions noting that, when populated, the conversation is currently assigned to that agent and inbound messages skip the registered flow trigger. Cross-reference the assignment overview section that Task C2 will add.

**Step 3:** Do NOT commit yet — see the "Important" note at the top of Phase C. Continue to Task C2.

---

### Task C2: Add "Assigning a Conversation to an Agent" overview

**File:** `conversation_overview.rst`

**Step 1:** Add a new section after the existing structure/operations sections covering:
- The `PUT /v1.0/conversations/<id>` partial-update with `owner_id`.
- Unassign payload: `{"owner_id": "00000000-0000-0000-0000-000000000000"}`.
- Permission semantics: admin/manager assigns or reassigns; owning agent self-unassigns; cross-agent or cross-customer attempts return 403; agent not found / customer mismatch return 400.
- Behavior change: when assigned, the registered flow is not triggered for new inbound messages; already-running activeflows are unaffected.
- The list filter: `GET /v1.0/conversations?owner_id=<id>`.

**Step 2:** Do NOT commit yet — accumulate edits and commit together in Task C4.

---

### Task C3: Add walkthrough tutorial

**File:** `conversation_tutorial.rst`

**Step 1:** Add a section walking through:
1. Admin assigns: `PUT /v1.0/conversations/<id>` with `{"owner_id": "<agent-uuid>"}`.
2. Agent receives the webhook update with the new owner_id.
3. Inbound message arrives: no activeflow created.
4. Agent replies via the existing `POST /v1.0/conversations/<id>/messages`.
5. Agent self-unassigns: `PUT /v1.0/conversations/<id>` with `{"owner_id": "00000000-0000-0000-0000-000000000000"}`.
6. Next inbound message: registered flow resumes.

Include curl examples and example webhook payloads.

**Step 2:** Do NOT commit yet — accumulate edits and commit together in Task C4.

---

### Task C4: Changelog note + clean rebuild

**Files:**
- Modify: `conversation_overview.rst` (top-of-page changelog note)
- Rebuild: `bin-api-manager/docsdev/build/`

**Step 1:** Add a one-line changelog note near the top of `conversation_overview.rst` mentioning that `owner_type` / `owner_id` in webhook payloads will start carrying real values for assigned conversations as of this release; existing unassigned conversations continue to read empty values. Additive change; not breaking.

**Step 2:** Clean rebuild:

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

Expected: clean build with no warnings about missing references.

**Step 3:** Force-add the rebuilt HTML and commit:

```bash
git add bin-api-manager/docsdev/source/conversation_struct_conversation.rst \
        bin-api-manager/docsdev/source/conversation_overview.rst \
        bin-api-manager/docsdev/source/conversation_tutorial.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "$(cat <<'EOF'
NOJIRA-Conversation-agent-assignment RST documentation

- bin-api-manager: Expand owner_type and owner_id descriptions in conversation_struct_conversation.rst to cover their meaning for agent assignment
- bin-api-manager: Add Assigning a Conversation to an Agent section to conversation_overview.rst — covers the PUT partial-update, unassign payload, permission semantics, behavior change for inbound messages, and the list filter
- bin-api-manager: Add walkthrough tutorial in conversation_tutorial.rst — admin assigns, agent receives webhook, agent replies, agent self-unassigns, flow resumes
- bin-api-manager: Add additive-change changelog note to conversation_overview.rst about webhook payload values
- bin-api-manager: Clean rebuild of docsdev/build/ via sphinx and force-add per repo convention
EOF
)"
```

---

## Phase D: api-validator integration tests

### Task D1: Add the assign/unassign integration flow

**Why:** Per the api-validator workflow rule. Read-only and conversation/agent CRUD are safe; no calls/SMS/email-send.

**Files:**
- Create or modify: `~/gitvoipbin/monorepo-monitoring/api-validator/<appropriate-file>` (verify the location of conversation tests in that repo).

**Step 1:** Check api-validator's directory structure for existing conversation tests.

**Step 2:** Add an integration test that:
1. Creates an agent (or reuses a fixture).
2. Lists conversations for the customer (sets a baseline).
3. Picks an existing conversation (or creates one if necessary via supported test fixtures).
4. PUTs `{owner_id: <agent-id>}`. Asserts 200 and webhook reflects new owner_id.
5. GETs `?owner_id=<agent-id>`. Asserts the conversation appears.
6. PUTs `{owner_id: nil-UUID}`. Asserts 200.
7. GETs `?owner_id=<agent-id>`. Asserts the conversation no longer appears.

**Step 3:** Run the validator against the dev cluster.

**Step 4:** Commit per the api-validator repo's conventions (separate repo from monorepo).

---

## Phase E: PR and merge

**Important — branch and worktree distinction:**

This plan document was committed on the **plan branch** (`NOJIRA-Conversation-agent-assignment`) and merged to `main` separately. The implementation tasks A1–D1 happen on a **separate, new** worktree on branch `NOJIRA-Conversation-agent-assignment` (no `-plan` suffix). Before starting Phase A, create that worktree from `main`:

```bash
cd ~/gitvoipbin/monorepo
git fetch origin main && git pull origin main
git worktree add ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Conversation-agent-assignment -b NOJIRA-Conversation-agent-assignment
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Conversation-agent-assignment
```

All commit messages in Tasks A1–D1 use title `NOJIRA-Conversation-agent-assignment` (matching the branch name per the root CLAUDE.md rule). The plan branch's existing PR is unrelated; do not push implementation commits to the plan branch.

**Step 1: Pre-PR conflict check**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Conversation-agent-assignment
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)" || echo "no conflicts"
```

If conflicts exist, rebase, resolve, and re-run **all** Phase A and Phase B verification workflows.

**Step 2: Final monorepo-wide verification**

For each service touched (`bin-conversation-manager`, `bin-api-manager`):
```bash
cd <service-dir> && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Push and open PR**

```bash
git push -u origin NOJIRA-Conversation-agent-assignment
gh pr create --title "NOJIRA-Conversation-agent-assignment" --body "$(cat <<'BODY'
Implement manual conversation-to-agent assignment per docs/plans/2026-04-30-assignable-conversation-design.md.

- bin-conversation-manager: Refactor MessageExecuteActiveflow to internal error-only executeActiveflow helper
- bin-conversation-manager: Add ExecuteMode dispatch (None/Agent/Flow) with getExecuteMode and per-conversation-type runners (LINE via account.MessageFlowID, SMS via number.MessageFlowID)
- bin-conversation-manager: Wire ExecuteMode dispatch into hookLine and MessageEventReceived; assigned conversations skip flow trigger
- bin-conversation-manager: Derive owner_type from owner_id in Update; always override caller-supplied owner_type
- bin-conversation-manager: Validate agent existence and same-customer constraint as a pre-check on assignment; canonical error strings for the two 400 cases
- bin-api-manager: Add field-level permission gate on ConversationUpdate; admin/manager retain full access, owning agents may self-unassign only
- bin-api-manager: Add round-trip regression test confirming empty-string field updates survive the PutConversationsIdJSONBody decode pipeline
- bin-api-manager: Update conversation_struct_conversation.rst, conversation_overview.rst, and conversation_tutorial.rst; add additive-change changelog note; clean rebuild of docsdev/build/
BODY
)"
```

**Step 4: Run the review-and-fix loop**

Per the project workflow, run a code-review loop on the PR (e.g., via `pr-review-toolkit:review-pr`) and address all CRITICAL and HIGH severity issues before requesting merge.

**Step 5: Wait for explicit user authorization to merge**

Per the repo CLAUDE.md, do NOT merge without explicit user instruction. When authorized:

```bash
gh pr merge <pr-number> --squash --delete-branch
cd ~/gitvoipbin/monorepo && git pull origin main
git worktree remove ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Conversation-agent-assignment
```

---

## Definition of done

- [ ] Tasks A1–A8 complete; conversation-manager full verification passes.
- [ ] Tasks B1–B2 complete; api-manager full verification passes.
- [ ] Tasks C1–C4 complete; RST docs rebuilt and committed.
- [ ] Task D1 complete; api-validator regression flow lands and passes against dev.
- [ ] PR opened, code-review loop run to APPROVED state.
- [ ] PR squash-merged with explicit user authorization.
- [ ] `main` synced locally; worktree cleaned.

---

## Notes on rollback

Per design §10, rollback is `git revert` of the implementation PR(s) and redeploy. No schema migration. Leftover `owner_type='agent'` / `owner_id=<uuid>` values in DB are inert post-rollback (the dispatch is gone). Roll-forward re-activates pre-existing assignments unless ops bulk-clears first.
