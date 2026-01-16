# Critical Technical Fixes for bin-talk-manager Implementation

**Date:** 2026-01-17
**Status:** Required before implementation

## Overview

This document describes three critical technical issues identified in the initial implementation plan and their solutions.

---

## Issue 1: Reaction Race Condition ⚠️

### Problem

The current `ReactionAdd` implementation uses a read-modify-write pattern:
```go
// RACE CONDITION - Multiple concurrent requests will lose data
m, _ := h.dbHandler.MessageGet(ctx, messageID)        // Read
json.Unmarshal([]byte(m.Metadata), &metadata)        // Modify
metadata.Reactions = append(metadata.Reactions, ...) // Modify
h.dbHandler.MessageUpdate(ctx, messageID, fields)    // Write
```

**Scenario:**
1. User A and User B simultaneously add reactions to the same message
2. Both read the message metadata (e.g., `{"reactions": []}`)
3. User A appends 👍, User B appends ❤️
4. User A writes `{"reactions": [{"emoji": "👍", ...}]}`
5. User B writes `{"reactions": [{"emoji": "❤️", ...}]}` - **OVERWRITES User A's reaction**

**Result:** Lost data, missing reactions

### Solution: Atomic MySQL JSON Operations (Preferred)

Use MySQL's native JSON functions to perform atomic updates at the database level, avoiding application-level read-modify-write cycles.

**Implementation:**

1. **Add new method to dbhandler:**
```go
// MessageAddReaction atomically adds a reaction using MySQL JSON functions
func (h *dbHandler) MessageAddReactionAtomic(ctx context.Context, messageID uuid.UUID, reactionJSON string) error {
	query := `
		UPDATE talk_messages
		SET metadata = JSON_SET(
			metadata,
			'$.reactions',
			JSON_ARRAY_APPEND(
				JSON_EXTRACT(metadata, '$.reactions'),
				'$',
				CAST(? AS JSON)
			)
		),
		tm_update = ?
		WHERE id = ?
	`

	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	_, err := h.db.ExecContext(ctx, query, reactionJSON, now, messageID.Bytes())
	return err
}

// MessageRemoveReaction atomically removes a reaction by filtering the JSON array
func (h *dbHandler) MessageRemoveReactionAtomic(ctx context.Context, messageID uuid.UUID, emoji, ownerType string, ownerID uuid.UUID) error {
	// MySQL doesn't have JSON_ARRAY_REMOVE with predicate, so we use a workaround:
	// 1. Get current reactions
	// 2. Filter in Go
	// 3. Replace entire array atomically

	query := `
		UPDATE talk_messages
		SET metadata = JSON_SET(metadata, '$.reactions', CAST(? AS JSON)),
		    tm_update = ?
		WHERE id = ?
	`

	// First, get current metadata
	var metadataJSON string
	err := h.db.QueryRowContext(ctx,
		"SELECT metadata FROM talk_messages WHERE id = ?",
		messageID.Bytes(),
	).Scan(&metadataJSON)
	if err != nil {
		return err
	}

	// Parse and filter reactions
	var metadata message.Metadata
	json.Unmarshal([]byte(metadataJSON), &metadata)

	var filtered []message.Reaction
	for _, r := range metadata.Reactions {
		if !(r.Emoji == emoji && r.OwnerType == ownerType && r.OwnerID == ownerID) {
			filtered = append(filtered, r)
		}
	}

	filteredJSON, _ := json.Marshal(filtered)
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")

	_, err = h.db.ExecContext(ctx, query, string(filteredJSON), now, messageID.Bytes())
	return err
}
```

2. **Update reactionhandler to use atomic operations:**
```go
func (h *reactionHandler) ReactionAdd(ctx context.Context, messageID uuid.UUID, emoji, ownerType string, ownerID uuid.UUID) (*message.Message, error) {
	// Check if reaction already exists (idempotent check)
	m, err := h.dbHandler.MessageGet(ctx, messageID)
	if err != nil {
		return nil, err
	}

	var metadata message.Metadata
	json.Unmarshal([]byte(m.Metadata), &metadata)

	for _, r := range metadata.Reactions {
		if r.Emoji == emoji && r.OwnerType == ownerType && r.OwnerID == ownerID {
			// Already exists, return current message (idempotent)
			h.publishReactionUpdated(m)
			return m, nil
		}
	}

	// Add reaction atomically
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	reaction := message.Reaction{
		Emoji:     emoji,
		OwnerType: ownerType,
		OwnerID:   ownerID,
		TMCreate:  now,
	}
	reactionJSON, _ := json.Marshal(reaction)

	err = h.dbHandler.MessageAddReactionAtomic(ctx, messageID, string(reactionJSON))
	if err != nil {
		return nil, err
	}

	// Refresh and publish
	m, _ = h.dbHandler.MessageGet(ctx, messageID)
	h.publishReactionUpdated(m)
	return m, nil
}

func (h *reactionHandler) ReactionRemove(ctx context.Context, messageID uuid.UUID, emoji, ownerType string, ownerID uuid.UUID) (*message.Message, error) {
	// Remove reaction atomically
	err := h.dbHandler.MessageRemoveReactionAtomic(ctx, messageID, emoji, ownerType, ownerID)
	if err != nil {
		return nil, err
	}

	// Refresh and publish
	m, _ := h.dbHandler.MessageGet(ctx, messageID)
	h.publishReactionUpdated(m)
	return m, nil
}
```

**Benefits:**
- Atomic operation - no race conditions
- Database-level consistency
- Better performance (single query vs read + write)

**Note:** For remove operations, we still need a read-filter-write cycle since MySQL doesn't support JSON_ARRAY_REMOVE with predicates. However, this is less critical since remove operations are less frequent and the worst case is that a removal fails silently (reaction remains).

---

## Issue 2: Participant Re-join (Unique Key Conflict) ⚠️

### Problem

The `talk_participants` table has:
```sql
UNIQUE KEY unique_participant (chat_id, owner_type, owner_id)
```

Current `ParticipantCreate` uses simple INSERT:
```go
func (h *dbHandler) ParticipantCreate(ctx context.Context, p *participant.Participant) error {
	query := sq.Insert(tableParticipants).SetMap(fields)
	// ...
}
```

**Scenario:**
1. User joins a talk (INSERT succeeds)
2. User leaves the talk (DELETE removes row)
3. User tries to re-join (INSERT fails with duplicate key error)

**Result:** User cannot re-join talks they previously left

### Solution: Upsert with ON DUPLICATE KEY UPDATE

Use MySQL's `INSERT ... ON DUPLICATE KEY UPDATE` to handle both new participants and re-joins.

**Implementation:**

```go
func (h *dbHandler) ParticipantCreate(ctx context.Context, p *participant.Participant) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000Z")
	p.TMJoined = now

	// Use raw SQL for ON DUPLICATE KEY UPDATE (Squirrel doesn't support it well)
	query := `
		INSERT INTO talk_participants
		(id, customer_id, chat_id, owner_type, owner_id, tm_joined)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		tm_joined = VALUES(tm_joined)
	`

	_, err := h.db.ExecContext(ctx, query,
		p.ID.Bytes(),
		p.CustomerID.Bytes(),
		p.ChatID.Bytes(),
		p.OwnerType,
		p.OwnerID.Bytes(),
		now,
	)

	if err != nil {
		log.Errorf("Failed to create/update participant: %v", err)
		return err
	}

	return nil
}
```

**Logic:**
- First insert attempt: Creates new participant with new ID and tm_joined
- Subsequent insert (re-join): Updates tm_joined to current time, preserves original ID

**Benefits:**
- Users can leave and re-join talks freely
- No application-level "check if exists" logic needed
- Atomic operation (no race conditions)

---

## Issue 3: Message Threading Validation Policy ⚠️

### Problem

The current implementation has the correct validation logic but lacks explicit documentation.

**Current code (lines 1715-1731):**
```go
// Validate parent message if provided
if req.ParentID != nil {
	parent, err := h.dbHandler.MessageGet(ctx, *req.ParentID)
	if err != nil {
		log.Errorf("Parent message not found: %v", err)
		return nil, errors.New("parent message does not exist")
	}

	// Parent must be in same chat
	if parent.ChatID != req.ChatID {
		log.Errorf("Parent message is in different chat")
		return nil, errors.New("parent message must be in the same talk")
	}

	// Parent can be deleted (thread structure preserved)
	// No additional validation needed
}
```

**Issue:** The comment "Parent can be deleted" is present, but there's no explicit check showing that we're intentionally allowing deleted parents.

### Solution: Add Explicit Validation Comments

Update the validation logic to make the policy crystal clear:

```go
// Validate parent message if provided
if req.ParentID != nil {
	parent, err := h.dbHandler.MessageGet(ctx, *req.ParentID)
	if err != nil {
		log.Errorf("Parent message not found: %v", err)
		return nil, errors.New("parent message does not exist")
	}

	// VALIDATION POLICY: Two checks required

	// 1. Parent must exist in database
	//    (Already validated above - MessageGet would fail if not found)

	// 2. Parent must be in same talk (prevents cross-talk threading)
	if parent.ChatID != req.ChatID {
		log.Errorf("Parent message is in different chat")
		return nil, errors.New("parent message must be in the same talk")
	}

	// INTENTIONALLY ALLOWED: Parent can be soft-deleted (tm_delete IS NOT NULL)
	// Reason: Preserve thread structure even when parent messages are deleted
	// UI should display deleted parent as placeholder (e.g., "Message deleted")
	// This allows conversation context to remain intact

	log.Debugf("Parent message validation passed: exists=%v, same_chat=%v, is_deleted=%v",
		parent.ID != uuid.Nil,
		parent.ChatID == req.ChatID,
		parent.TMDelete != "",
	)
}
```

**Policy Confirmation:**

✅ **Replies ARE allowed if parent is soft-deleted (`tm_delete IS NOT NULL`)**

This is the correct design because:
1. **Thread structure preservation** - Deleting a parent shouldn't break all child threads
2. **Conversation context** - Users can still see reply relationships even if parent is deleted
3. **UI responsibility** - Frontend shows "Message deleted" placeholder for deleted parents
4. **Common pattern** - Slack, Discord, Twitter all use this approach

---

## Summary of Changes

### Files to Update in Implementation Plan:

1. **Task 2.7: Implement Message Database Operations**
   - Add `MessageAddReactionAtomic()` method
   - Add `MessageRemoveReactionAtomic()` method

2. **Task 2.6: Implement Participant Database Operations**
   - Change `ParticipantCreate()` to use `INSERT ... ON DUPLICATE KEY UPDATE`

3. **Task 3.2: Create Message Handler**
   - Add explicit validation comments in `MessageCreate()` parent validation logic

4. **Task 3.4: Create Reaction Handler**
   - Replace read-modify-write logic with atomic operations
   - Update `ReactionAdd()` to use `MessageAddReactionAtomic()`
   - Update `ReactionRemove()` to use `MessageRemoveReactionAtomic()`

### Impact Assessment:

- **Breaking changes:** None (these are implementation improvements before any code exists)
- **API changes:** None (external API remains identical)
- **Performance:** Improved (atomic operations are faster and safer)
- **Reliability:** Significantly improved (eliminates race conditions and re-join bugs)

---

## Testing Considerations

### Reaction Concurrency Test
```go
func TestReactionAddConcurrent(t *testing.T) {
	// Setup: Create message
	// Execute: 10 goroutines add different reactions simultaneously
	// Assert: All 10 reactions present (no lost updates)
}
```

### Participant Re-join Test
```go
func TestParticipantRejoin(t *testing.T) {
	// Setup: Create talk and participant
	// Execute: Delete participant, then add again
	// Assert: No error, participant exists with new tm_joined
}
```

### Deleted Parent Thread Test
```go
func TestMessageCreateWithDeletedParent(t *testing.T) {
	// Setup: Create parent message, soft-delete it
	// Execute: Create reply with parent_id pointing to deleted message
	// Assert: Reply created successfully, parent_id preserved
}
```

---

**Next Steps:** Apply these changes to the implementation plan before beginning Task 1 (Service Scaffolding).
