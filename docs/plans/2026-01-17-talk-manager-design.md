# bin-talk-manager Design Document

**Date:** 2026-01-17
**Status:** Design Phase
**Author:** System Design

## Overview

This document describes the design for `bin-talk-manager`, a new microservice that replaces `bin-chat-manager` with a simplified, scalable architecture. The redesign eliminates problematic data duplication, adds threading and reaction features, and provides a clean foundation for modern chat functionality.

## Problem Statement

The existing `bin-chat-manager` has architectural issues:

1. **Data Duplication:** Messages are duplicated across `chat_messagechats` and `chat_messagechatrooms` tables (N copies for N participants)
2. **Consistency Challenges:** Keeping duplicated data in sync is complex and error-prone
3. **Poor Scalability:** Storage and update costs scale linearly with participant count
4. **Complex Logic:** Two-level message model (messagechat/messagechatroom) complicates business logic
5. **Missing Features:** No support for threading, reactions, or modern chat patterns

## Design Goals

1. **Single Source of Truth:** Messages stored once, no duplication
2. **Scalability:** Support up to 200 participants per chat efficiently
3. **Modern Features:** Built-in threading (replies) and reactions
4. **Simple Architecture:** Clear separation of concerns, easy to understand
5. **No Migration Risk:** New service coexists with old, gradual adoption

## Architecture

### Core Data Model

The redesign uses three core tables:

```
talk_chats (chat sessions)
  ↓ 1:N
talk_participants (who's in each chat)

talk_messages (all messages)
  ↓ references
talk_messages.parent_id (for threading)
```

### Service Boundaries

**bin-talk-manager (new service):**
- Clean architecture with threading and reactions
- Tables: `talk_chats`, `talk_messages`, `talk_participants`
- RabbitMQ queues: `QueueNameTalkRequest`, `QueueNameTalkEvent`, `QueueNameTalkSubscribe`
- API routes: `/v1/talks`, `/v1/messages`, `/v1/participants`

**bin-chat-manager (legacy):**
- Continues serving existing clients unchanged
- No new development
- Eventually deprecated when all clients migrate

## Database Schema

### Table: talk_chats

```sql
CREATE TABLE talk_chats (
    id              BINARY(16) PRIMARY KEY,
    customer_id     BINARY(16) NOT NULL,
    type            VARCHAR(255) NOT NULL,  -- 'normal' or 'group'
    tm_create       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    tm_update       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    tm_delete       DATETIME(6) NULL,
    INDEX idx_customer_id (customer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**Fields:**
- `id`: Chat unique identifier
- `customer_id`: Tenant/customer this chat belongs to
- `type`: `"normal"` (1:1) or `"group"` (multi-participant)
- `tm_create`, `tm_update`, `tm_delete`: Standard timestamp fields

### Table: talk_participants

```sql
CREATE TABLE talk_participants (
    id              BINARY(16) PRIMARY KEY,
    chat_id         BINARY(16) NOT NULL,
    owner_type      VARCHAR(255) NOT NULL,  -- 'agent', 'customer', 'system'
    owner_id        BINARY(16) NOT NULL,
    tm_joined       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_chat_id (chat_id),
    INDEX idx_owner (owner_type, owner_id),
    UNIQUE KEY unique_participant (chat_id, owner_type, owner_id),
    FOREIGN KEY (chat_id) REFERENCES talk_chats(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**Fields:**
- `id`: Participant record identifier
- `chat_id`: References parent chat
- `owner_type`, `owner_id`: Participant identity (using monorepo Owner pattern)
- `tm_joined`: When participant joined the chat

**Key Design Decisions:**
- Tracks current participants only (rows deleted when user leaves)
- No participation history tracking (YAGNI principle)
- UNIQUE constraint prevents duplicate participant entries
- CASCADE delete removes participants when chat is deleted

### Table: talk_messages

```sql
CREATE TABLE talk_messages (
    id              BINARY(16) PRIMARY KEY,
    customer_id     BINARY(16) NOT NULL,
    chat_id         BINARY(16) NOT NULL,
    parent_id       BINARY(16) NULL,  -- For threading/replies

    -- Sender (Owner pattern)
    owner_type      VARCHAR(255) NOT NULL,
    owner_id        BINARY(16) NOT NULL,

    -- Content
    type            VARCHAR(255) NOT NULL,  -- 'normal', 'system'
    text            TEXT,
    medias          JSON,  -- Array of media attachments
    metadata        JSON,  -- Reactions and future extensions

    -- Timestamps
    tm_create       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    tm_update       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    tm_delete       DATETIME(6) NULL,

    INDEX idx_chat_id (chat_id),
    INDEX idx_parent_id (parent_id),
    INDEX idx_customer_id (customer_id),
    INDEX idx_owner (owner_type, owner_id),
    FOREIGN KEY (chat_id) REFERENCES talk_chats(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES talk_messages(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**Fields:**
- `id`: Message unique identifier
- `customer_id`: Tenant/customer
- `chat_id`: Parent chat reference
- `parent_id`: Parent message for threading (nullable)
- `owner_type`, `owner_id`: Message sender identity
- `type`: Message type (`"normal"` or `"system"`)
- `text`: Message text content
- `medias`: JSON array of media attachments (files, links, etc.)
- `metadata`: JSON object for reactions and future extensions

**Metadata Structure:**
```json
{
  "reactions": [
    {
      "emoji": "👍",
      "owner_type": "agent",
      "owner_id": "550e8400-e29b-41d4-a716-446655440000",
      "tm_create": "2026-01-17T10:30:00Z"
    },
    {
      "emoji": "❤️",
      "owner_type": "customer",
      "owner_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "tm_create": "2026-01-17T10:31:00Z"
    }
  ]
}
```

**Threading Behavior:**
- Messages with `parent_id = NULL` are top-level messages
- Replies set `parent_id` to reference parent message
- Flat reply structure (no nested hierarchy limits)
- Parent validation enforced (must exist and be in same chat)
- Parent deletion preserves `parent_id` (UI shows "message deleted" placeholder)

## API Endpoints

### Chat Endpoints

```
POST   /v1/talks
       Body: {"customer_id": "uuid", "type": "group"}
       Returns: Created talk object

GET    /v1/talks?page_size=X&page_token=Y
       Filters (in body): customer_id, deleted
       Returns: Array of talks with pagination

GET    /v1/talks/{talk-id}
       Returns: Single talk object

DELETE /v1/talks/{talk-id}
       Soft-deletes talk and all messages (cascade)
       Returns: Deleted talk object
```

### Participant Endpoints

```
POST   /v1/talks/{talk-id}/participants
       Body: {"owner_type": "agent", "owner_id": "uuid"}
       Returns: Created participant object

GET    /v1/talks/{talk-id}/participants
       Returns: Array of participant objects

DELETE /v1/talks/{talk-id}/participants/{participant-id}
       Removes participant from talk (hard delete)
       Returns: Success status
```

### Message Endpoints

```
POST   /v1/messages
       Body: {
         "chat_id": "uuid",
         "owner_type": "agent",
         "owner_id": "uuid",
         "text": "Hello world",
         "parent_id": "uuid",  // Optional - for replies
         "medias": [...]       // Optional
       }
       Returns: Created message object

GET    /v1/messages?page_size=X&page_token=Y
       Filters (in body): chat_id, customer_id, parent_id, deleted
       Returns: Array of messages with pagination

GET    /v1/messages/{message-id}
       Returns: Single message object with reactions

DELETE /v1/messages/{message-id}
       Soft-deletes message (sets tm_delete)
       Returns: Deleted message object
```

### Reaction Endpoints

```
POST   /v1/messages/{message-id}/reactions
       Body: {"emoji": "👍", "owner_type": "agent", "owner_id": "uuid"}
       Adds reaction (idempotent - duplicate reactions ignored)
       Returns: Updated message object

DELETE /v1/messages/{message-id}/reactions
       Body: {"emoji": "👍", "owner_type": "agent", "owner_id": "uuid"}
       Removes specific reaction by this user
       Returns: Updated message object
```

## Business Logic

### Handler Architecture

```
listenhandler (RabbitMQ RPC router)
  ├─ talkhandler (talk CRUD operations)
  ├─ messagehandler (message CRUD, threading validation)
  ├─ participanthandler (participant management)
  └─ reactionhandler (reaction add/remove logic)
      │
      └─ dbhandler (database operations)
          └─ cachehandler (Redis caching)
```

**Removed handlers (compared to bin-chat-manager):**
- `chatroomhandler` (eliminated)
- `messagechatroomhandler` (eliminated)

**New handlers:**
- `participanthandler` (manages talk participants)
- `reactionhandler` (manages message reactions)

### Message Creation Flow

1. Validate talk exists
2. Validate sender is a participant in the talk
3. If `parent_id` provided:
   - Validate parent message exists
   - Validate parent is in the same talk
   - Return error if validation fails
4. Create message record in `talk_messages`
5. Publish `message_created` webhook event

**Key Difference from Old System:**
- Single message record created (no duplication)
- No chatroom records to create
- Much simpler and faster

### Reaction Management

**Add Reaction:**
1. Fetch message from database
2. Parse existing `metadata.reactions` JSON array
3. Check if this user already added this emoji:
   - If yes: Idempotent operation, return success (no change)
   - If no: Append new reaction object to array
4. Update message metadata JSON in database
5. Publish `message_reaction_updated` webhook event

**Remove Reaction:**
1. Fetch message from database
2. Parse `metadata.reactions` JSON array
3. Filter out reaction matching: `emoji` AND `owner_type` AND `owner_id`
4. Update message metadata JSON in database
5. Publish `message_reaction_updated` webhook event

**Multiple Reactions Per User:**
- Users can add multiple different emojis to the same message
- Example: Same user can add both 👍 and ❤️
- Removing one emoji doesn't affect other reactions by same user

### Parent Message Validation

**On message creation with parent_id:**
1. Query parent message: `SELECT id, chat_id, tm_delete FROM talk_messages WHERE id = ?`
2. If parent not found: Return error `parent message does not exist`
3. If parent.chat_id ≠ new message.chat_id: Return error `parent must be in same talk`
4. If parent.tm_delete IS NOT NULL: Allow (parent can be deleted, thread preserved)
5. Proceed with message creation

**On parent message deletion:**
- Child messages keep their `parent_id` (references preserved)
- UI can show "Message deleted" placeholder for missing parent
- Thread structure remains intact

### Participant Management

**Add Participant:**
1. INSERT INTO talk_participants with UNIQUE constraint
2. If duplicate: Return error or treat as idempotent (implementation choice)
3. Set `tm_joined` to current timestamp
4. Publish `participant_added` event

**Remove Participant:**
1. DELETE FROM talk_participants WHERE id = ?
2. Participant row is hard-deleted (no soft delete)
3. Past messages remain visible to all participants (no history filtering)
4. Publish `participant_removed` event

## Webhook Events

### Event Types

```
talk_created            - New talk created
talk_deleted            - Talk deleted

message_created         - New message posted (includes parent_id if reply)
message_deleted         - Message soft-deleted
message_reaction_updated - Reactions added or removed

participant_added       - Participant joined talk
participant_removed     - Participant left/removed from talk
```

### Event Payload Examples

**message_created:**
```json
{
  "id": "550e8400-...",
  "customer_id": "7c9e6679-...",
  "chat_id": "8d7f5890-...",
  "parent_id": "9e8g6901-...",  // null if not a reply
  "owner_type": "agent",
  "owner_id": "abc12345-...",
  "type": "normal",
  "text": "This is a reply",
  "medias": [],
  "metadata": {"reactions": []},
  "tm_create": "2026-01-17T10:30:00Z"
}
```

**message_reaction_updated:**
```json
{
  "id": "550e8400-...",
  "customer_id": "7c9e6679-...",
  "chat_id": "8d7f5890-...",
  "metadata": {
    "reactions": [
      {
        "emoji": "👍",
        "owner_type": "agent",
        "owner_id": "abc12345-...",
        "tm_create": "2026-01-17T10:31:00Z"
      }
    ]
  },
  "tm_update": "2026-01-17T10:31:00Z"
}
```

## Integration Points

### bin-common-handler Changes

**Add queue constants (`models/outline/queuename.go`):**
```go
QueueNameTalkRequest   = "bin-manager.talk.request"
QueueNameTalkEvent     = "bin-manager.talk.event"
QueueNameTalkSubscribe = "bin-manager.talk.subscribe"
```

**Add service name constant (`models/outline/service.go`):**
```go
ServiceNameTalk = "talk"
```

**Add requesthandler methods (`pkg/requesthandler/`):**
```go
TalkV1TalkCreate(ctx context.Context, req TalkCreateRequest) (*Talk, error)
TalkV1TalkGet(ctx context.Context, talkID uuid.UUID) (*Talk, error)
TalkV1TalkList(ctx context.Context, filters map[Field]any, pageOpts PageOptions) ([]*Talk, error)

TalkV1MessageCreate(ctx context.Context, req MessageCreateRequest) (*Message, error)
TalkV1MessageList(ctx context.Context, filters map[Field]any, pageOpts PageOptions) ([]*Message, error)

TalkV1ParticipantAdd(ctx context.Context, talkID uuid.UUID, req ParticipantRequest) (*Participant, error)
TalkV1ParticipantList(ctx context.Context, talkID uuid.UUID) ([]*Participant, error)

TalkV1ReactionAdd(ctx context.Context, messageID uuid.UUID, req ReactionRequest) (*Message, error)
TalkV1ReactionRemove(ctx context.Context, messageID uuid.UUID, req ReactionRequest) (*Message, error)
```

### bin-api-manager Changes

**Add new routes (`internal/routes/`):**
```go
// Talk routes
apiV1.POST("/talks", handler.TalkCreate)
apiV1.GET("/talks", handler.TalkList)
apiV1.GET("/talks/:id", handler.TalkGet)
apiV1.DELETE("/talks/:id", handler.TalkDelete)

// Participant routes
apiV1.POST("/talks/:id/participants", handler.ParticipantAdd)
apiV1.GET("/talks/:id/participants", handler.ParticipantList)
apiV1.DELETE("/talks/:id/participants/:participant_id", handler.ParticipantRemove)

// Message routes
apiV1.POST("/messages", handler.MessageCreate)
apiV1.GET("/messages", handler.MessageList)
apiV1.GET("/messages/:id", handler.MessageGet)
apiV1.DELETE("/messages/:id", handler.MessageDelete)

// Reaction routes
apiV1.POST("/messages/:id/reactions", handler.ReactionAdd)
apiV1.DELETE("/messages/:id/reactions", handler.ReactionRemove)
```

**Permission checks:**
- Require `PermissionCustomerAdmin` OR `PermissionCustomerManager`
- Validate `agent.CustomerID == talk.CustomerID`
- Validate sender is a participant before allowing message creation

### bin-openapi-manager Changes

Add OpenAPI schema definitions for:
- `TalkManagerTalk`
- `TalkManagerMessage`
- `TalkManagerParticipant`
- `TalkManagerReaction`
- Request/response models for all endpoints

## Implementation Plan

### Phase 1: Service Scaffolding (Week 1)
1. Create `bin-talk-manager` directory structure
2. Set up go.mod with replace directives
3. Create database migration scripts in `bin-dbscheme-manager`
4. Implement configuration management (Cobra/Viper)
5. Set up RabbitMQ connection and listener
6. Add Prometheus metrics endpoint

### Phase 2: Core Models & Database (Week 1)
1. Implement models: `talk`, `message`, `participant`
2. Add db struct tags (with `,uuid` for UUID fields)
3. Implement `dbhandler` for database operations
4. Use `commondatabasehandler` pattern (PrepareFields, ScanRow, etc.)
5. Implement Redis caching via `cachehandler`
6. Write unit tests for database operations

### Phase 3: Business Logic (Week 2)
1. Implement `talkhandler` (CRUD operations)
2. Implement `messagehandler` (creation, threading validation)
3. Implement `participanthandler` (add/remove participants)
4. Implement `reactionhandler` (add/remove reactions)
5. Write unit tests with mocks
6. Implement webhook event publishing

### Phase 4: API Integration (Week 2)
1. Update `bin-common-handler` (queue names, requesthandler)
2. Add OpenAPI specs in `bin-openapi-manager`
3. Generate models: `go generate ./...`
4. Integrate routes in `bin-api-manager`
5. Add permission checks
6. Update Swagger documentation

### Phase 5: Testing & Deployment (Week 3-4)
1. Integration testing (RabbitMQ RPC flows)
2. Load testing (200-participant chats)
3. Deploy to staging environment
4. Update API documentation
5. Internal dogfooding
6. Production deployment

### Phase 6: Client Migration (Month 2+)
1. Update internal services to use bin-talk-manager
2. Migrate external API consumers
3. Monitor usage of bin-chat-manager
4. Deprecate old endpoints when usage drops to zero
5. Eventually decommission bin-chat-manager

## Testing Strategy

### Unit Tests
- Mock all handler dependencies (dbhandler, cachehandler)
- Table-driven tests for business logic
- Test reaction add/remove edge cases
- Test threading validation (parent exists, same chat, etc.)

### Integration Tests
- Test full RabbitMQ RPC flows
- Test message creation → webhook event publishing
- Test cascading deletes (talk → messages)
- Test reaction management end-to-end

### Load Tests
- Create 200-participant talk
- Send 1000 messages
- Add 50 reactions per message
- Measure query performance
- Verify no N+1 query problems

### Migration Tests
- Verify new service handles expected load
- Ensure old bin-chat-manager continues working
- Test coexistence (both services running)

## Monitoring & Observability

### Prometheus Metrics
- Request latency per endpoint
- Message creation rate
- Reaction operations per second
- Active talk count
- Participant count per talk (histogram)

### Logging
- Structured logging with logrus
- Log all RPC requests/responses
- Log validation failures (parent not found, etc.)
- Log reaction operations

### Alerting
- High error rate (>5%)
- Slow queries (>500ms)
- RabbitMQ queue backlog
- Database connection pool exhaustion

## Risks & Mitigations

### Risk: Reaction Concurrency Issues
**Scenario:** Two users add reactions simultaneously, last write wins
**Mitigation:** Use optimistic locking or row-level locking in reaction operations

### Risk: Thread Depth Explosion
**Scenario:** Users create deeply nested reply chains (100+ levels)
**Mitigation:** Client-side rendering limits, no database constraints needed (flat structure)

### Risk: Large Participant Counts
**Scenario:** Talks with >200 participants
**Mitigation:** Add participant limit validation, return error at 200 participants

### Risk: Emoji Encoding Issues
**Scenario:** Unicode emoji storage/retrieval problems
**Mitigation:** UTF8MB4 charset in MySQL, test emoji round-tripping

### Risk: Parent Message Deletion UX
**Scenario:** Threads become confusing when parent deleted
**Mitigation:** UI responsibility to show "Message deleted" placeholders

## Future Enhancements

### Post-MVP Features (Not in initial release)
1. **Per-user metadata:** Read receipts, starred messages, pinned messages
2. **Message editing:** Allow users to edit their own messages
3. **Message search:** Full-text search across message content
4. **Rich media:** Better media handling (thumbnails, previews)
5. **Typing indicators:** Real-time typing status
6. **Thread summaries:** Show reply counts, latest reply timestamp
7. **Reaction aggregation:** Group by emoji type, show counts
8. **Message mentions:** @-mention participants with notifications

### Not Planned (YAGNI)
- Message access history (user sees what they had access to)
- Per-participant message visibility
- Message snapshots/archives for removed users
- Complex thread hierarchies (tree structures)

## Success Criteria

1. **Performance:** Message creation <100ms for 200-participant talks
2. **Scalability:** Support 10,000 active talks simultaneously
3. **Reliability:** 99.9% uptime, no data loss
4. **Adoption:** 50% of clients migrated within 3 months
5. **Code Quality:** 80%+ test coverage, passing linters

## Conclusion

The `bin-talk-manager` redesign eliminates architectural complexity from `bin-chat-manager` while adding modern chat features (threading, reactions). By creating a new service instead of migrating, we avoid risk and maintain backward compatibility. The clean three-table design (chats, messages, participants) provides a solid foundation for future enhancements.

## Appendix: Comparison Table

| Feature | bin-chat-manager (old) | bin-talk-manager (new) |
|---------|------------------------|------------------------|
| Message Storage | Duplicated (N copies) | Single source of truth |
| Participant Tracking | Embedded JSON array | Separate table |
| Threading | Not supported | Built-in with parent_id |
| Reactions | Not supported | Built-in with metadata |
| Tables | 4 (chats, chatrooms, messagechats, messagechatrooms) | 3 (talks, messages, participants) |
| Scalability | Poor (O(N) writes) | Good (O(1) writes) |
| Complexity | High (dual model) | Low (single model) |
| Query Performance | Multiple JOINs needed | Simple single-table queries |
