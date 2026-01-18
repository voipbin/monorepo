# Square-Talk Client Implementation Guide

> **For Claude:** This document describes the Talk API server implementation and provides guidance for implementing the client-side integration in square-talk.

**Date:** 2026-01-17
**Server Implementation:** Complete
**Client Implementation:** Pending (square-talk)

---

## Overview

The VoIPBIN Talk API has been fully implemented on the server side. This document provides all necessary information for implementing the client-side Talk functionality in the square-talk application.

**What Talk Provides:**
- Real-time messaging between agents
- Message threading (replies to messages)
- Emoji reactions on messages
- Group conversations with participant management
- 1:1 and group talk types

---

## Server Implementation Status

### ✅ Completed Components

**Backend Services:**
- `bin-talk-manager` - Core Talk service (RabbitMQ RPC)
- `bin-api-manager` - REST API gateway (HTTP endpoints)
- `bin-common-handler` - RPC client library (13 methods)

**Database:**
- `talk_chats` - Talk sessions
- `talk_participants` - Talk membership
- `talk_messages` - Messages with threading and reactions
- Alembic migration: `f6d7d64e259e_talk_create_tables_talk_chats_talk_.py`

**Testing:**
- 12 RPC method tests (bin-common-handler)
- 12 HTTP endpoint tests (bin-api-manager)
- All tests passing

**Documentation:**
- Developer docs at `bin-api-manager/docsdev/source/talk*.rst`
- API reference in OpenAPI schema

---

## API Endpoints

All endpoints are under `/service_agents/` prefix and require agent authentication.

### Talk Management

**List Talks**
```http
GET /service_agents/talk_chats?page_size=50&page_token=<timestamp>

Response:
{
  "result": [
    {
      "id": "uuid",
      "customer_id": "uuid",
      "type": "normal",  // or "group"
      "tm_create": "timestamp",
      "tm_update": "timestamp",
      "tm_delete": ""
    }
  ],
  "next_page_token": "timestamp"
}
```

**Create Talk**
```http
POST /service_agents/talk_chats
{
  "type": "normal"  // or "group"
}

Response: Talk object
```

**Get Talk**
```http
GET /service_agents/talk_chats/{id}

Response: Talk object
```

**Delete Talk**
```http
DELETE /service_agents/talk_chats/{id}

Response: Talk object (with tm_delete set)
```

### Participant Management

**List Participants**
```http
GET /service_agents/talk_chats/{id}/participants

Response: Array of Participant objects
[
  {
    "id": "uuid",
    "customer_id": "uuid",
    "owner_type": "agent",
    "owner_id": "uuid",
    "chat_id": "uuid",
    "tm_joined": "timestamp"
  }
]
```

**Add Participant**
```http
POST /service_agents/talk_chats/{id}/participants
{
  "owner_type": "agent",
  "owner_id": "uuid"
}

Response: Participant object
```

**Remove Participant**
```http
DELETE /service_agents/talk_chats/{id}/participants/{participant_id}

Response: Participant object
```

### Message Management

**List Messages**
```http
GET /service_agents/talk_messages?page_size=50&page_token=<timestamp>

Response:
{
  "result": [
    {
      "id": "uuid",
      "customer_id": "uuid",
      "owner_type": "agent",
      "owner_id": "uuid",
      "chat_id": "uuid",
      "parent_id": "uuid",  // optional, for threading
      "type": "normal",  // or "system"
      "text": "message text",
      "medias": [],
      "metadata": {
        "reactions": [
          {
            "emoji": "👍",
            "owner_type": "agent",
            "owner_id": "uuid",
            "tm_create": "timestamp"
          }
        ]
      },
      "tm_create": "timestamp",
      "tm_update": "timestamp",
      "tm_delete": ""
    }
  ],
  "next_page_token": "timestamp"
}
```

**Send Message**
```http
POST /service_agents/talk_messages
{
  "chat_id": "uuid",
  "parent_id": "uuid",  // optional, for replies
  "type": "normal",
  "text": "message text"
}

Response: Message object
```

**Get Message**
```http
GET /service_agents/talk_messages/{id}

Response: Message object
```

**Delete Message**
```http
DELETE /service_agents/talk_messages/{id}

Response: Message object (with tm_delete set)
```

**Add Reaction**
```http
POST /service_agents/talk_messages/{id}/reactions
{
  "emoji": "👍"
}

Response: Message object (with updated reactions in metadata)
```

---

## Data Models

### Talk
```typescript
interface Talk {
  id: string;                // UUID
  customer_id: string;       // UUID
  type: "normal" | "group";  // Talk type
  tm_create: string;         // ISO timestamp
  tm_update: string;         // ISO timestamp
  tm_delete: string;         // ISO timestamp or empty
}
```

### Participant
```typescript
interface Participant {
  id: string;           // UUID
  customer_id: string;  // UUID
  owner_type: string;   // "agent"
  owner_id: string;     // UUID
  chat_id: string;      // UUID
  tm_joined: string;    // ISO timestamp
}
```

### Message
```typescript
interface Message {
  id: string;               // UUID
  customer_id: string;      // UUID
  owner_type: string;       // "agent"
  owner_id: string;         // UUID
  chat_id: string;          // UUID
  parent_id?: string;       // UUID, optional for threading
  type: "normal" | "system";
  text: string;
  medias: Media[];
  metadata: {
    reactions: Reaction[];
  };
  tm_create: string;        // ISO timestamp
  tm_update: string;        // ISO timestamp
  tm_delete: string;        // ISO timestamp or empty
}

interface Reaction {
  emoji: string;            // Emoji character
  owner_type: string;       // "agent"
  owner_id: string;         // UUID
  tm_create: string;        // ISO timestamp
}

interface Media {
  type: string;             // "file", "link", etc.
  // Additional fields based on type
}
```

---

## Client Implementation Requirements

### Core Features to Implement

**1. Talk List View**
- Display list of talks (1:1 and group)
- Show last message preview
- Show unread message count (client-side tracking)
- Pull-to-refresh for new talks
- Pagination support

**2. Talk Detail View**
- Display messages in chronological order
- Show message sender (agent name/avatar)
- Show timestamps
- Auto-scroll to bottom for new messages
- Message input field

**3. Threading Support**
- Display threaded replies indented under parent
- "Reply" button on messages
- Visual thread indicator (line connecting replies)
- Collapse/expand threads

**4. Reactions**
- Display emoji reactions on messages
- Show count per emoji type
- Show who reacted (on tap/hover)
- Reaction picker UI
- Add/remove reactions

**5. Participant Management**
- View participant list
- Add participants (for group talks)
- Remove participants (for group talks)
- Show participant status (online/offline - if available)

**6. Message Composition**
- Text input field
- Send button
- "Replying to..." indicator when threading
- Cancel reply option

**7. Real-time Updates** (CRITICAL)
- WebSocket connection for live updates
- Receive new messages in real-time
- Update reactions in real-time
- Update participant list in real-time

### Technical Considerations

**Authentication**
- All requests require agent JWT token
- Token passed in `Authorization: Bearer <token>` header

**Pagination**
- Use `page_token` for cursor-based pagination
- Token is the `tm_create` timestamp of the last item
- Default `page_size` is 100, max is 100

**WebSocket Integration**
- Connect to `/service_agents/ws` endpoint
- Subscribe to Talk events:
  - `talk_created`
  - `talk_deleted`
  - `message_created`
  - `message_deleted`
  - `message_updated` (for reactions)
  - `participant_joined`
  - `participant_left`

**Error Handling**
- 400: Bad request (validation error)
- 401: Unauthorized (invalid/expired token)
- 403: Forbidden (not a participant)
- 404: Not found (talk/message doesn't exist)

**Threading Validation**
- Parent message must exist
- Parent message must be in same talk
- Parent can be deleted (display placeholder)

**Reaction Handling**
- Multiple reactions per message
- Same agent can add multiple different emojis
- Reactions are append-only (no remove endpoint yet)

---

## Implementation Phases

### Phase 1: Basic Messaging
**Goal:** Send and receive messages in existing talks

- [ ] Create API client service for Talk endpoints
- [ ] Implement message list view (no threading)
- [ ] Implement message send functionality
- [ ] Add basic message display (sender, text, timestamp)
- [ ] Test with manual talk creation via API

**Acceptance Criteria:**
- Can view messages in a talk
- Can send new messages
- Messages appear in chronological order

### Phase 2: Talk Management
**Goal:** Create and manage talks

- [ ] Implement talk list view
- [ ] Add "New Talk" button (create normal talk)
- [ ] Add participant selection UI
- [ ] Implement participant list view
- [ ] Add/remove participants

**Acceptance Criteria:**
- Can create new 1:1 talks
- Can create new group talks
- Can add/remove participants
- Talk list shows all accessible talks

### Phase 3: Threading
**Goal:** Support threaded conversations

- [ ] Add "Reply" button to messages
- [ ] Implement reply input UI with context
- [ ] Send messages with `parent_id`
- [ ] Display threaded messages (indent/tree view)
- [ ] Add collapse/expand for threads

**Acceptance Criteria:**
- Can reply to any message
- Replies appear under parent message
- Thread structure is clear visually
- Can navigate threads easily

### Phase 4: Reactions
**Goal:** Support emoji reactions

- [ ] Display reactions on messages
- [ ] Add reaction picker UI
- [ ] Send reaction requests
- [ ] Update message UI when reactions change
- [ ] Show who reacted (tooltip/modal)

**Acceptance Criteria:**
- Can add reactions to messages
- Reactions display correctly
- Multiple reactions supported
- Can see who reacted

### Phase 5: Real-time Updates
**Goal:** Live updates via WebSocket

- [ ] Connect to WebSocket endpoint
- [ ] Subscribe to Talk events
- [ ] Handle `message_created` events
- [ ] Handle `message_updated` events (reactions)
- [ ] Handle `participant_joined/left` events
- [ ] Update UI without refresh

**Acceptance Criteria:**
- New messages appear instantly
- Reactions update in real-time
- Participant changes reflect immediately
- No manual refresh needed

### Phase 6: Polish & UX
**Goal:** Production-ready experience

- [ ] Add unread message tracking
- [ ] Implement message search
- [ ] Add typing indicators (if server supports)
- [ ] Optimize performance (virtualized lists)
- [ ] Add loading states
- [ ] Add error states with retry
- [ ] Add offline support (queue messages)

---

## Testing Strategy

**Unit Tests:**
- API client methods
- Message threading logic
- Reaction state management
- Participant list operations

**Integration Tests:**
- Create talk flow
- Send message flow
- Add reaction flow
- Thread message flow
- Participant management flow

**E2E Tests:**
- Complete conversation workflow
- Multi-participant group talk
- Threading with multiple levels
- Reaction interactions
- Real-time updates

**Manual Testing Checklist:**
- [ ] Create 1:1 talk
- [ ] Create group talk with 3+ participants
- [ ] Send messages in both talks
- [ ] Reply to messages (threading)
- [ ] Add reactions to messages
- [ ] Remove participant from group
- [ ] Add participant to group
- [ ] Delete messages
- [ ] Delete talk
- [ ] Verify real-time updates in another window

---

## Known Limitations & Future Work

**Current Limitations:**
1. No reaction removal endpoint (reactions are append-only)
2. No message editing (only delete)
3. No read receipts
4. No typing indicators
5. No message search endpoint
6. No file attachments (medias array is empty)

**Future Enhancements:**
- Message editing support
- Read receipts
- Typing indicators
- File/image attachments via medias
- Message search
- Mention notifications (@agent)
- Push notifications for new messages

---

## Reference Implementation Examples

### Creating a Talk and Sending Messages

```typescript
// 1. Create a new group talk
const talk = await talkAPI.createTalk({ type: "group" });

// 2. Add participants
await talkAPI.addParticipant(talk.id, {
  owner_type: "agent",
  owner_id: "agent-uuid-1"
});
await talkAPI.addParticipant(talk.id, {
  owner_type: "agent",
  owner_id: "agent-uuid-2"
});

// 3. Send a message
const message = await talkAPI.sendMessage({
  chat_id: talk.id,
  type: "normal",
  text: "Hello team!"
});

// 4. Reply to the message (threading)
const reply = await talkAPI.sendMessage({
  chat_id: talk.id,
  parent_id: message.id,
  type: "normal",
  text: "Great to be here!"
});

// 5. Add a reaction
const updatedMessage = await talkAPI.addReaction(message.id, {
  emoji: "👍"
});
```

### Displaying Threaded Messages

```typescript
interface MessageNode {
  message: Message;
  replies: MessageNode[];
}

function buildThreadTree(messages: Message[]): MessageNode[] {
  const messageMap = new Map<string, MessageNode>();
  const rootMessages: MessageNode[] = [];

  // Create nodes for all messages
  messages.forEach(msg => {
    messageMap.set(msg.id, { message: msg, replies: [] });
  });

  // Build tree structure
  messages.forEach(msg => {
    const node = messageMap.get(msg.id)!;
    if (msg.parent_id) {
      const parent = messageMap.get(msg.parent_id);
      if (parent) {
        parent.replies.push(node);
      } else {
        // Parent deleted or not loaded, treat as root
        rootMessages.push(node);
      }
    } else {
      rootMessages.push(node);
    }
  });

  return rootMessages;
}
```

### WebSocket Event Handling

```typescript
// Connect to WebSocket
const ws = new WebSocket('wss://api.voipbin.net/service_agents/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);

  switch (data.type) {
    case 'message_created':
      // Add new message to UI
      addMessageToTalk(data.data);
      break;

    case 'message_updated':
      // Update message (likely reactions changed)
      updateMessage(data.data);
      break;

    case 'participant_joined':
      // Update participant list
      addParticipant(data.data);
      break;

    case 'participant_left':
      // Update participant list
      removeParticipant(data.data);
      break;
  }
};
```

---

## API Client Reference

All Talk API methods are available in `bin-common-handler/pkg/requesthandler`:

**Talk Methods:**
- `TalkV1TalkCreate(ctx, customerID, type)`
- `TalkV1TalkGet(ctx, talkID)`
- `TalkV1TalkDelete(ctx, talkID)`
- `TalkV1TalkList(ctx, pageSize, pageToken)`

**Message Methods:**
- `TalkV1TalkMessageCreate(ctx, chatID, parentID, ownerType, ownerID, msgType, text)`
- `TalkV1TalkMessageGet(ctx, messageID)`
- `TalkV1TalkMessageDelete(ctx, messageID)`
- `TalkV1TalkMessageList(ctx, pageSize, pageToken)`
- `TalkV1TalkMessageReactionCreate(ctx, messageID, emoji)`

**Participant Methods:**
- `TalkV1TalkParticipantCreate(ctx, chatID, ownerType, ownerID)`
- `TalkV1TalkParticipantDelete(ctx, chatID, participantID)`
- `TalkV1TalkParticipantList(ctx, chatID)`

---

## Questions & Support

**For clarification or issues:**
1. Check API documentation: `bin-api-manager/docsdev/source/talk*.rst`
2. Review server tests: `bin-api-manager/server/service_agents_talk_test.go`
3. Check OpenAPI schema: `bin-openapi-manager/openapi/openapi.yaml` (search for "TalkManager")

**Common Issues:**

**Q: Messages not appearing in real-time?**
A: Ensure WebSocket connection is established and subscribed to correct events.

**Q: "User has no permission" error?**
A: Agent must be a participant of the talk to view/send messages.

**Q: Thread not displaying correctly?**
A: Verify parent_id exists and matches a message in the same talk.

**Q: Reaction not showing?**
A: Reactions are in `message.metadata.reactions` array, check you're parsing it correctly.

---

## Success Criteria

Client implementation is complete when:

- ✅ Agents can create and view talks
- ✅ Agents can send and receive messages
- ✅ Threading works (replies display under parent)
- ✅ Reactions work (add and display emoji reactions)
- ✅ Participants can be added/removed
- ✅ Real-time updates work via WebSocket
- ✅ UI is responsive and intuitive
- ✅ All core flows have test coverage

---

**Implementation Time Estimate:** 3-5 days for core features (Phases 1-4), 1-2 days for real-time (Phase 5), 1-2 days for polish (Phase 6).

**Priority Order:** Phase 1 → Phase 2 → Phase 4 → Phase 3 → Phase 5 → Phase 6

Good luck with the implementation! 🚀
