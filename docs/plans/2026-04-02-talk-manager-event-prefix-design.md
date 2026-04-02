# Talk-Manager Event Prefix Design

## Problem

Talk-manager events lack a service prefix, making them ambiguous. For example, `message_created` could belong to talk-manager, message-manager, or ai-manager. Other services already use prefixed names (`call_created`, `agent_created`, `aimessage_created`, `conversation_message_created`).

## Rename Mapping

| Current | New | Constant Name (unchanged) |
|---------|-----|--------------------------|
| `message_created` | `chatmessage_created` | `EventTypeMessageCreated` |
| `message_deleted` | `chatmessage_deleted` | `EventTypeMessageDeleted` |
| `message_reaction_updated` | `chatmessage_reaction_updated` | `EventTypeMessageReactionUpdated` |
| `chat_created` | *(unchanged)* | `EventTypeChatCreated` |
| `chat_updated` | *(unchanged)* | `EventTypeChatUpdated` |
| `chat_deleted` | *(unchanged)* | `EventTypeChatDeleted` |
| `participant_added` | `chatparticipant_added` | `EventParticipantAdded` |
| `participant_removed` | `chatparticipant_removed` | `EventParticipantRemoved` |

The `chat_*` events already have a natural "chat" prefix and remain unchanged.

## Files to Change

### bin-talk-manager (constant definitions + tests)

1. `models/message/event.go` — change 3 string values
2. `models/message/event_test.go` — update 3 expected strings
3. `models/participant/event.go` — change 2 string values
4. `models/participant/event_test.go` — update 2 expected strings

### bin-api-manager (test hardcoded strings)

5. `pkg/subscribehandler/talk_test.go` — update hardcoded event type strings and derived topic assertion strings

### RST documentation

6. `docsdev/source/talk_overview.rst` — update event table and WebSocket diagram to reflect new names. Also correct pre-existing inaccuracies (RST lists `message_updated`, `reaction_added`, `participant_joined` which don't match actual code events).

## Files NOT Requiring Changes

- **bin-api-manager Go code** — uses constants by reference (`talkmessage.EventTypeMessageCreated`), picks up new values at compile time
- **bin-talk-manager handler code** — uses constants, not raw strings
- **OpenAPI spec** — no event type enums defined for talk events
- **bin-flow-manager / bin-conversation-manager** — have vendor copies but don't use talk-manager event constants in their code
- **bin-webhook-manager** — generic pass-through structure, no talk-specific logic

## ZMQ Topic Impact

Old-format topics change (accepted):
- `agent_id:UUID:message:MSG_ID` → `agent_id:UUID:chatmessage:MSG_ID`
- `agent_id:UUID:participant:ID` → `agent_id:UUID:chatparticipant:ID`

New-format topics change:
- `agent_id:UUID:talk:message_created:MSG_ID` → `agent_id:UUID:talk:chatmessage_created:MSG_ID`
- `agent_id:UUID:talk:participant_added:ID` → `agent_id:UUID:talk:chatparticipant_added:ID`

## External Webhook Impact

Webhook payloads pass through raw event type strings. The `type` field in webhook HTTP POST bodies changes (e.g., `"message_created"` → `"chatmessage_created"`). This is a breaking change for external consumers filtering on these event types.

## Deployment

Both `bin-talk-manager` and `bin-api-manager` must be deployed together to avoid a mismatch window where events are unrouted.

---

## Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add "chat" service prefix to talk-manager message and participant event names for disambiguation.

**Architecture:** Change string constant values in talk-manager event definitions. All consumers reference these constants, so only tests with hardcoded strings need updating. RST docs also need correction.

**Tech Stack:** Go, Sphinx RST

---

### Task 1: Update message event constants

**Files:**
- Modify: `bin-talk-manager/models/message/event.go`

**Step 1: Update the three constant string values**

```go
// Event type constants for webhook events
const (
	EventTypeMessageCreated         = "chatmessage_created"
	EventTypeMessageDeleted         = "chatmessage_deleted"
	EventTypeMessageReactionUpdated = "chatmessage_reaction_updated"
)
```

**Step 2: Update the test expected values**

Modify `bin-talk-manager/models/message/event_test.go`:

```go
{"event_type_message_created", EventTypeMessageCreated, "chatmessage_created"},
{"event_type_message_deleted", EventTypeMessageDeleted, "chatmessage_deleted"},
{"event_type_message_reaction_updated", EventTypeMessageReactionUpdated, "chatmessage_reaction_updated"},
```

**Step 3: Run tests to verify**

```bash
cd bin-talk-manager && go test ./models/message/...
```

Expected: PASS

---

### Task 2: Update participant event constants

**Files:**
- Modify: `bin-talk-manager/models/participant/event.go`

**Step 1: Update the two constant string values**

```go
const (
	// EventParticipantAdded is published when a participant is added to a talk
	EventParticipantAdded = "chatparticipant_added"

	// EventParticipantRemoved is published when a participant is removed from a talk
	EventParticipantRemoved = "chatparticipant_removed"
)
```

**Step 2: Update the test expected values**

Modify `bin-talk-manager/models/participant/event_test.go`:

```go
{"event_participant_added", EventParticipantAdded, "chatparticipant_added"},
{"event_participant_removed", EventParticipantRemoved, "chatparticipant_removed"},
```

**Step 3: Run tests to verify**

```bash
cd bin-talk-manager && go test ./models/participant/...
```

Expected: PASS

---

### Task 3: Run full bin-talk-manager verification

**Step 1: Run full verification workflow**

```bash
cd bin-talk-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All steps pass. No code changes needed in handlers (they use constants).

---

### Task 4: Update bin-api-manager test hardcoded strings

**Files:**
- Modify: `bin-api-manager/pkg/subscribehandler/talk_test.go`

**Step 1: Update `Test_extractResource` test cases**

Update the `"message_created"` and `"participant_added"` test cases to use the new prefixed event names:

```go
{
    name:      "chatmessage_created",
    eventType: "chatmessage_created",
    expected:  "chatmessage",
},
{
    name:      "chatparticipant_added",
    eventType: "chatparticipant_added",
    expected:  "chatparticipant",
},
```

**Step 2: Run tests to verify**

```bash
cd bin-api-manager && go test ./pkg/subscribehandler/... -run Test_extractResource -v
```

Expected: PASS

---

### Task 5: Run full bin-api-manager verification

**Step 1: Run full verification workflow**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All steps pass.

---

### Task 6: Update RST documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/talk_overview.rst`

**Step 1: Update the WebSocket diagram (lines 368-371)**

Replace:
```
       |<==== message_created ========|
       |<==== message_updated ========|
       |<==== participant_joined =====|
       |<==== reaction_added =========|
```

With (corrected to match actual code events):
```
       |<==== chatmessage_created ====|
       |<==== chat_created ===========|
       |<==== chatparticipant_added ==|
```

**Step 2: Update the Event Types table (lines 376-392)**

Replace the entire table with corrected event names matching code:

```rst
+--------------------------------------+------------------------------------------------+
| Event                                | When it fires                                  |
+======================================+================================================+
| chat_created                         | New chat session created                       |
+--------------------------------------+------------------------------------------------+
| chat_updated                         | Chat session details updated                   |
+--------------------------------------+------------------------------------------------+
| chat_deleted                         | Chat session deleted                           |
+--------------------------------------+------------------------------------------------+
| chatmessage_created                  | New message sent to chat                       |
+--------------------------------------+------------------------------------------------+
| chatmessage_deleted                  | Message removed from chat                      |
+--------------------------------------+------------------------------------------------+
| chatmessage_reaction_updated         | Reaction added or removed on a message         |
+--------------------------------------+------------------------------------------------+
| chatparticipant_added                | Participant joined the chat                    |
+--------------------------------------+------------------------------------------------+
| chatparticipant_removed              | Participant left the chat                      |
+--------------------------------------+------------------------------------------------+
```

**Step 3: Rebuild HTML documentation**

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

Expected: Build succeeds with no errors.

---

### Task 7: Commit all changes

**Step 1: Stage and commit**

```bash
git add bin-talk-manager/models/message/event.go \
        bin-talk-manager/models/message/event_test.go \
        bin-talk-manager/models/participant/event.go \
        bin-talk-manager/models/participant/event_test.go \
        bin-api-manager/pkg/subscribehandler/talk_test.go \
        bin-api-manager/docsdev/source/talk_overview.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Add-talk-manager-event-prefix

Add chat service prefix to talk-manager event names for disambiguation
from message-manager and ai-manager events.

- bin-talk-manager: Rename message events to chatmessage_created/deleted/reaction_updated
- bin-talk-manager: Rename participant events to chatparticipant_added/removed
- bin-api-manager: Update test hardcoded event strings in subscribehandler
- bin-api-manager: Update RST docs event table and WebSocket diagram to match code"
```

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-Add-talk-manager-event-prefix
```
