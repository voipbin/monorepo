# Team Member Switch Notification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Record team member switches during AI calls as notification messages in the conversation transcript, with full audit details and automatic webhook events.

**Architecture:** Python Pipecat transition handler sends HTTP notification to Go pipecat-manager, which publishes a RabbitMQ event. AI Manager subscribes to the event and creates a notification message in the database. The existing `aimessage_created` webhook fires automatically.

**Tech Stack:** Go (ai-manager, pipecat-manager), Python (pipecat team_flow.py), RabbitMQ events, Gin HTTP router

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification`

---

### Task 1: Add RoleNotification to ai-manager message model

**Files:**
- Modify: `bin-ai-manager/models/message/main.go:32-39`
- Modify: `bin-ai-manager/models/message/main_test.go:82-127`

**Step 1: Write the failing test**

Add a test case for the new `RoleNotification` constant in `bin-ai-manager/models/message/main_test.go`. Insert into the `TestRoleConstants` table:

```go
{
    name:     "role_notification",
    constant: RoleNotification,
    expected: "notification",
},
```

**Step 2: Run test to verify it fails**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification/bin-ai-manager
go test -v ./models/message/ -run TestRoleConstants
```

Expected: FAIL — `RoleNotification` undefined.

**Step 3: Write minimal implementation**

Add to the role constants in `bin-ai-manager/models/message/main.go:32-39`, after `RoleTool`:

```go
RoleNotification Role = "notification"
```

**Step 4: Run test to verify it passes**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification/bin-ai-manager
go test -v ./models/message/ -run TestRoleConstants
```

Expected: PASS

**Step 5: Also add a test case for RoleNotification in TestMessage**

Add to `bin-ai-manager/models/message/main_test.go` TestMessage table:

```go
{
    name:      "creates_message_with_notification_role",
    aicallID:  uuid.Must(uuid.NewV4()),
    direction: DirectionOutgoing,
    role:      RoleNotification,
    content:   `{"type":"member_switched","transition_function_name":"transfer_to_sales"}`,
},
```

Run: `go test -v ./models/message/ -run TestMessage`
Expected: PASS

**Step 6: Commit**

```bash
git add bin-ai-manager/models/message/main.go bin-ai-manager/models/message/main_test.go
git commit -m "NOJIRA-Add-team-member-switch-notification

- bin-ai-manager: Add RoleNotification constant to message model
- bin-ai-manager: Add test coverage for notification role"
```

---

### Task 2: Add MemberSwitchedEvent to pipecat-manager message models

**Files:**
- Modify: `bin-pipecat-manager/models/message/event.go:1-9`
- Create: `bin-pipecat-manager/models/message/member_switched.go`
- Modify: `bin-pipecat-manager/models/message/event_test.go:1-40`
- Create: `bin-pipecat-manager/models/message/member_switched_test.go`

**Step 1: Write the failing test for the new event type constant**

Add to `bin-pipecat-manager/models/message/event_test.go` TestEventTypes table:

```go
{
    name:     "team member switched",
    constant: EventTypeTeamMemberSwitched,
    want:     "team_member_switched",
},
```

**Step 2: Run test to verify it fails**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification/bin-pipecat-manager
go test -v ./models/message/ -run TestEventTypes
```

Expected: FAIL — `EventTypeTeamMemberSwitched` undefined.

**Step 3: Add the event type constant**

Add to `bin-pipecat-manager/models/message/event.go`:

```go
EventTypeTeamMemberSwitched string = "team_member_switched"
```

**Step 4: Run test to verify it passes**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification/bin-pipecat-manager
go test -v ./models/message/ -run TestEventTypes
```

Expected: PASS

**Step 5: Write the MemberSwitchedEvent and MemberInfo structs**

Create `bin-pipecat-manager/models/message/member_switched.go`:

```go
package message

import (
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

// MemberSwitchedEvent is the event payload published when
// a team member transition occurs during an AI call.
type MemberSwitchedEvent struct {
	PipecatcallID            uuid.UUID                 `json:"pipecatcall_id,omitempty"`
	PipecatcallReferenceType pipecatcall.ReferenceType `json:"pipecatcall_reference_type,omitempty"`
	PipecatcallReferenceID   uuid.UUID                 `json:"pipecatcall_reference_id,omitempty"`
	TransitionFunctionName   string                    `json:"transition_function_name,omitempty"`
	FromMember               MemberInfo                `json:"from_member"`
	ToMember                 MemberInfo                `json:"to_member"`
}

// MemberInfo holds non-sensitive details about a team member.
type MemberInfo struct {
	ID          uuid.UUID `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	EngineModel string    `json:"engine_model,omitempty"`
	TTSType     string    `json:"tts_type,omitempty"`
	TTSVoiceID  string    `json:"tts_voice_id,omitempty"`
	STTType     string    `json:"stt_type,omitempty"`
}
```

**Step 6: Write tests for MemberSwitchedEvent**

Create `bin-pipecat-manager/models/message/member_switched_test.go`:

```go
package message

import (
	"encoding/json"
	"testing"

	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

func TestMemberSwitchedEvent_JSONRoundTrip(t *testing.T) {
	evt := MemberSwitchedEvent{
		PipecatcallID:            uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001"),
		PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000002"),
		TransitionFunctionName:   "transfer_to_sales",
		FromMember: MemberInfo{
			ID:          uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001"),
			Name:        "Reception",
			EngineModel: "openai.gpt-4o",
			TTSType:     "cartesia",
			TTSVoiceID:  "voice-123",
			STTType:     "deepgram",
		},
		ToMember: MemberInfo{
			ID:          uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000002"),
			Name:        "Sales Agent",
			EngineModel: "openai.gpt-4o",
			TTSType:     "elevenlabs",
			TTSVoiceID:  "voice-456",
			STTType:     "deepgram",
		},
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var got MemberSwitchedEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if got.TransitionFunctionName != evt.TransitionFunctionName {
		t.Errorf("TransitionFunctionName = %q, want %q", got.TransitionFunctionName, evt.TransitionFunctionName)
	}
	if got.FromMember.Name != evt.FromMember.Name {
		t.Errorf("FromMember.Name = %q, want %q", got.FromMember.Name, evt.FromMember.Name)
	}
	if got.ToMember.Name != evt.ToMember.Name {
		t.Errorf("ToMember.Name = %q, want %q", got.ToMember.Name, evt.ToMember.Name)
	}
}
```

**Step 7: Run tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification/bin-pipecat-manager
go test -v ./models/message/...
```

Expected: PASS

**Step 8: Commit**

```bash
git add bin-pipecat-manager/models/message/
git commit -m "NOJIRA-Add-team-member-switch-notification

- bin-pipecat-manager: Add EventTypeTeamMemberSwitched constant
- bin-pipecat-manager: Add MemberSwitchedEvent and MemberInfo structs
- bin-pipecat-manager: Add tests for new event type and structs"
```

---

### Task 3: Add HTTP endpoint and handler in pipecat-manager Go side

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/main.go:24-48` (add `RunnerMemberSwitchedHandle` to interface)
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` (add handler implementation)
- Modify: `bin-pipecat-manager/pkg/httphandler/main.go:46-47` (add route)

**Step 1: Add the interface method**

In `bin-pipecat-manager/pkg/pipecatcallhandler/main.go`, add to `PipecatcallHandler` interface after `RunnerToolHandle`:

```go
RunnerMemberSwitchedHandle(id uuid.UUID, c *gin.Context) error
```

**Step 2: Write the handler implementation**

Add to `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go`, after the `RunnerToolHandle` method (after line 374):

```go
func (h *pipecatcallHandler) RunnerMemberSwitchedHandle(id uuid.UUID, c *gin.Context) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "RunnerMemberSwitchedHandle",
		"pipecatcall_id": id,
	})
	ctx := c.Request.Context()

	se, err := h.SessionGet(id)
	if err != nil {
		return fmt.Errorf("could not get pipecatcall session: %w", err)
	}
	log.WithField("session", se).Debugf("Pipecatcall session retrieved. pipecatcall_id: %s", id)

	pc, err := h.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("could not get pipecatcall: %w", err)
	}
	log.WithField("pipecatcall", pc).Debugf("Pipecatcall retrieved. pipecatcall_id: %s", id)

	if pc.ReferenceType != pipecatcall.ReferenceTypeAICall {
		return fmt.Errorf("pipecatcall reference type is not ai-call. reference_type: %s", pc.ReferenceType)
	}

	request := struct {
		TransitionFunctionName string         `json:"transition_function_name"`
		FromMember             message.MemberInfo `json:"from_member"`
		ToMember               message.MemberInfo `json:"to_member"`
	}{}
	if errBind := c.BindJSON(&request); errBind != nil {
		return fmt.Errorf("could not bind member-switched request JSON: %w", errBind)
	}

	evt := message.MemberSwitchedEvent{
		PipecatcallID:            pc.ID,
		PipecatcallReferenceType: pc.ReferenceType,
		PipecatcallReferenceID:   pc.ReferenceID,
		TransitionFunctionName:   request.TransitionFunctionName,
		FromMember:               request.FromMember,
		ToMember:                 request.ToMember,
	}

	h.notifyHandler.PublishEvent(ctx, message.EventTypeTeamMemberSwitched, evt)
	log.WithField("event", evt).Debugf("Published team member switched event.")

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
	return nil
}
```

**Step 3: Add the HTTP route**

In `bin-pipecat-manager/pkg/httphandler/main.go`, after line 47 (`router.POST("/:id/tools", h.toolHandle)`), add:

```go
router.POST("/:id/member-switched", h.memberSwitchedHandle)
```

**Step 4: Add the HTTP handler function**

Add to `bin-pipecat-manager/pkg/httphandler/main.go`, after `toolHandle` method (after line 106):

```go
func (h *httpHandler) memberSwitchedHandle(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func": "memberSwitchedHandle",
	})

	id := uuid.FromStringOrNil(c.Param("id"))
	if id == uuid.Nil {
		log.Errorf("Invalid pipecatcall ID: %s", c.Param("id"))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if errHandle := h.pipecatcallHandler.RunnerMemberSwitchedHandle(id, c); errHandle != nil {
		log.Errorf("Could not handle member-switched request. pipecatcall_id: %s, err: %v", id, errHandle)
		c.JSON(http.StatusBadRequest, gin.H{"error": errHandle.Error()})
		return
	}
}
```

**Step 5: Regenerate mocks**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification/bin-pipecat-manager
go generate ./pkg/pipecatcallhandler/...
```

**Step 6: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification/bin-pipecat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: PASS

**Step 7: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/ bin-pipecat-manager/pkg/httphandler/
git commit -m "NOJIRA-Add-team-member-switch-notification

- bin-pipecat-manager: Add RunnerMemberSwitchedHandle to PipecatcallHandler interface
- bin-pipecat-manager: Implement member-switched HTTP handler that publishes event
- bin-pipecat-manager: Add POST /:id/member-switched route in httphandler"
```

---

### Task 4: Add event subscription and message creation in ai-manager

**Files:**
- Modify: `bin-ai-manager/pkg/subscribehandler/main.go:167-176` (add case for new event)
- Modify: `bin-ai-manager/pkg/subscribehandler/pipecat_message.go` (add handler function)
- Modify: `bin-ai-manager/pkg/messagehandler/main.go:20-37` (add interface method)
- Modify: `bin-ai-manager/pkg/messagehandler/event.go` (add handler implementation)

**Step 1: Add the MessageHandler interface method**

In `bin-ai-manager/pkg/messagehandler/main.go`, add to `MessageHandler` interface after `EventPMMessageUserLLM`:

```go
EventPMTeamMemberSwitched(ctx context.Context, evt *pmmessage.MemberSwitchedEvent)
```

**Step 2: Implement the message handler method**

Add to `bin-ai-manager/pkg/messagehandler/event.go`, after `EventPMMessageUserLLM`:

```go
func (h *messageHandler) EventPMTeamMemberSwitched(ctx context.Context, evt *pmmessage.MemberSwitchedEvent) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "EventPMTeamMemberSwitched",
		"event": evt,
	})

	contentMap := map[string]any{
		"type":                     "member_switched",
		"transition_function_name": evt.TransitionFunctionName,
		"from_member": map[string]any{
			"id":   evt.FromMember.ID,
			"name": evt.FromMember.Name,
			"ai": map[string]any{
				"engine_model": evt.FromMember.EngineModel,
				"tts_type":     evt.FromMember.TTSType,
				"tts_voice_id": evt.FromMember.TTSVoiceID,
				"stt_type":     evt.FromMember.STTType,
			},
		},
		"to_member": map[string]any{
			"id":   evt.ToMember.ID,
			"name": evt.ToMember.Name,
			"ai": map[string]any{
				"engine_model": evt.ToMember.EngineModel,
				"tts_type":     evt.ToMember.TTSType,
				"tts_voice_id": evt.ToMember.TTSVoiceID,
				"stt_type":     evt.ToMember.STTType,
			},
		},
	}

	contentBytes, err := json.Marshal(contentMap)
	if err != nil {
		log.Errorf("Could not marshal notification content. err: %v", err)
		return
	}

	tmp, err := h.Create(ctx, evt.PipecatcallReferenceID, evt.PipecatcallReferenceID, message.DirectionOutgoing, message.RoleNotification, string(contentBytes), nil, "")
	if err != nil {
		log.Errorf("Could not create the notification message. err: %v", err)
		return
	}
	log.WithField("message", tmp).Debugf("Created member-switched notification message.")
}
```

**Important note:** The first arg of `h.Create` is `customerID`. We don't have `customerID` in the `MemberSwitchedEvent` — but looking at how `EventPMMessageUserTranscription` works (line 22 of event.go), it uses `evt.CustomerID` from the `pmmessage.Message` which inherits `identity.Identity`. The `MemberSwitchedEvent` does NOT have `identity.Identity`. We need to either: (a) add `CustomerID` to `MemberSwitchedEvent`, or (b) look up the aicall to get the customerID.

**Resolution:** Add `CustomerID` to `MemberSwitchedEvent`. The Go pipecat-manager handler already has `pc` (the pipecatcall) which carries `CustomerID`. Update `MemberSwitchedEvent` to include it:

In `bin-pipecat-manager/models/message/member_switched.go`, add to `MemberSwitchedEvent`:
```go
CustomerID               uuid.UUID                 `json:"customer_id,omitempty"`
```

And in `RunnerMemberSwitchedHandle` in runner.go, set it from the pipecatcall:
```go
evt := message.MemberSwitchedEvent{
    CustomerID:               pc.CustomerID,
    // ... rest of fields
}
```

Then the `EventPMTeamMemberSwitched` Create call becomes:
```go
tmp, err := h.Create(ctx, evt.CustomerID, evt.PipecatcallReferenceID, ...)
```

**Step 3: Add the subscribe handler case**

In `bin-ai-manager/pkg/subscribehandler/main.go`, add a new case after the `pmpipecatcall.EventTypeInitialized` case (after line 175):

```go
case m.Publisher == string(commonoutline.ServiceNamePipecatManager) && m.Type == string(pmmessage.EventTypeTeamMemberSwitched):
    err = h.processEventPMTeamMemberSwitched(ctx, m)
```

**Step 4: Add the subscribe handler function**

Add to `bin-ai-manager/pkg/subscribehandler/pipecat_message.go`, after `processEventPMMessageBotLLM`:

```go
func (h *subscribeHandler) processEventPMTeamMemberSwitched(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventPMTeamMemberSwitched",
		"event": m,
	})
	log.Debugf("Received the pipecat-manager's team_member_switched event.")

	var evt pmmessage.MemberSwitchedEvent
	if err := json.Unmarshal([]byte(m.Data), &evt); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	h.messageHandler.EventPMTeamMemberSwitched(ctx, &evt)

	return nil
}
```

**Step 5: Regenerate mocks**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification/bin-ai-manager
go generate ./pkg/messagehandler/...
```

**Step 6: Add `encoding/json` import to event.go if not already present**

In `bin-ai-manager/pkg/messagehandler/event.go`, add `"encoding/json"` to imports.

**Step 7: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: PASS

**Step 8: Commit**

```bash
git add bin-ai-manager/pkg/messagehandler/ bin-ai-manager/pkg/subscribehandler/ bin-pipecat-manager/models/message/
git commit -m "NOJIRA-Add-team-member-switch-notification

- bin-ai-manager: Add EventPMTeamMemberSwitched to MessageHandler interface
- bin-ai-manager: Implement notification message creation from member switch events
- bin-ai-manager: Subscribe to team_member_switched events in subscribehandler
- bin-pipecat-manager: Add CustomerID to MemberSwitchedEvent"
```

---

### Task 5: Add Python HTTP notification in team_flow.py

**Files:**
- Modify: `bin-pipecat-manager/scripts/pipecat/team_flow.py`

**Step 1: Add current_state tracking and pipecatcall_id to build_team_flow**

In `build_team_flow()`, after `member_nodes = {}` (line 49), add:

```python
current_state = {"active_member_id": resolved_team["start_member_id"]}
```

Update `_create_transition_handler` call (lines 74-80) to pass additional args:

```python
handler=_create_transition_handler(
    transition["next_member_id"],
    member_nodes,
    routing_llm,
    routing_tts,
    routing_stt,
    current_state,
    pipecatcall_id,
    resolved_team,
    transition["function_name"],
),
```

**Step 2: Update _create_transition_handler to accept new params and notify**

Replace the `_create_transition_handler` function (lines 116-139):

```python
def _create_transition_handler(
    next_member_id: str,
    member_nodes: dict,
    routing_llm,
    routing_tts,
    routing_stt,
    current_state: dict,
    pipecatcall_id: str,
    resolved_team: dict,
    function_name: str,
):
    """Create a FlowsFunctionSchema handler for member transitions."""
    async def handler(args: FlowArgs, flow_manager: FlowManager):
        # Validate target member exists before switching any services
        next_node = member_nodes.get(next_member_id)
        if next_node is None:
            logger.error(f"No NodeConfig for member {next_member_id}")
            return {"error": f"unknown member {next_member_id}"}, None

        from_member_id = current_state["active_member_id"]

        routing_llm.set_active_member(next_member_id)
        if routing_tts:
            routing_tts.set_active_member(next_member_id)
        if routing_stt:
            routing_stt.set_active_member(next_member_id)

        current_state["active_member_id"] = next_member_id

        # Fire-and-forget notification to Go
        asyncio.create_task(_notify_member_switched(
            pipecatcall_id, from_member_id, next_member_id,
            function_name, resolved_team,
        ))

        logger.info(f"Transition to member {next_member_id} successful")
        return {"status": "transferred"}, next_node
    return handler
```

**Step 3: Add _notify_member_switched function**

Add after `_create_transition_handler`, before `_http_session` (before line 142):

```python
def _find_member(resolved_team: dict, member_id: str) -> dict | None:
    """Find a member dict by ID in the resolved team."""
    for member in resolved_team.get("members", []):
        if member["id"] == member_id:
            return member
    return None


def _build_member_info(member: dict) -> dict:
    """Build a MemberInfo dict from a resolved member, excluding sensitive fields."""
    ai = member.get("ai", {})
    return {
        "id": member["id"],
        "name": member.get("name", ""),
        "engine_model": ai.get("engine_model", ""),
        "tts_type": ai.get("tts_type", ""),
        "tts_voice_id": ai.get("tts_voice_id", ""),
        "stt_type": ai.get("stt_type", ""),
    }


async def _notify_member_switched(
    pipecatcall_id: str,
    from_member_id: str,
    to_member_id: str,
    function_name: str,
    resolved_team: dict,
):
    """Fire-and-forget HTTP notification to Go about a member switch."""
    from_member = _find_member(resolved_team, from_member_id)
    to_member = _find_member(resolved_team, to_member_id)

    if from_member is None or to_member is None:
        logger.warning(
            f"[team_flow] Could not find member details for notification. "
            f"from={from_member_id} to={to_member_id}"
        )
        return

    http_url = f"{common.PIPECATCALL_HTTP_URL}/{pipecatcall_id}/member-switched"
    http_body = {
        "transition_function_name": function_name,
        "from_member": _build_member_info(from_member),
        "to_member": _build_member_info(to_member),
    }

    try:
        session = await _get_http_session()
        async with session.post(http_url, json=http_body) as response:
            if response.status >= 400:
                text = await response.text()
                logger.warning(f"[team_flow][member-switched] HTTP {response.status}: {text[:500]}")
            else:
                logger.debug(f"[team_flow][member-switched] Notification sent successfully")
    except Exception as e:
        logger.warning(f"[team_flow][member-switched] Failed to notify: {e}")
```

**Step 4: Run Python linting (if available)**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification/bin-pipecat-manager/scripts/pipecat
python3 -c "import ast; ast.parse(open('team_flow.py').read()); print('Syntax OK')"
```

Expected: `Syntax OK`

**Step 5: Commit**

```bash
git add bin-pipecat-manager/scripts/pipecat/team_flow.py
git commit -m "NOJIRA-Add-team-member-switch-notification

- bin-pipecat-manager: Track current active member in team flow
- bin-pipecat-manager: Send fire-and-forget HTTP notification on member switch
- bin-pipecat-manager: Add helper functions to extract member info from resolved team"
```

---

### Task 6: Run full verification for both services

**Files:** None (verification only)

**Step 1: Verify bin-pipecat-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification/bin-pipecat-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: PASS

**Step 2: Verify bin-ai-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification/bin-ai-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: PASS

**Step 3: Check for conflicts with main**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

Expected: No output (no conflicts)

---

### Task 7: Push branch and create PR

**Step 1: Push branch**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Add-team-member-switch-notification
git push -u origin NOJIRA-Add-team-member-switch-notification
```

**Step 2: Create PR**

Title: `NOJIRA-Add-team-member-switch-notification`

Body:
```
Add system notification messages for AI assistant switching in team-based AI calls.
When an LLM triggers a team member transition, a notification message is now recorded
in the AI call transcript with full audit details (from/to member, AI config, trigger function).

- bin-ai-manager: Add RoleNotification constant to message model
- bin-ai-manager: Subscribe to team_member_switched events from pipecat-manager
- bin-ai-manager: Create notification messages with full member switch audit details
- bin-pipecat-manager: Add MemberSwitchedEvent and MemberInfo model structs
- bin-pipecat-manager: Add POST /:id/member-switched HTTP endpoint
- bin-pipecat-manager: Publish team_member_switched events via notifyHandler
- bin-pipecat-manager: Track current active member in Python team flow
- bin-pipecat-manager: Send fire-and-forget HTTP notification on member transitions
- docs: Add design document for team member switch notification feature
```
