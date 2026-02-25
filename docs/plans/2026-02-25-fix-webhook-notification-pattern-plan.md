# Fix Webhook Notification Pattern Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix three call sites that incorrectly pre-convert to WebhookMessage before calling PublishWebhookEvent, causing internal event queue to lose data.

**Architecture:** The notifyhandler.PublishWebhookEvent function sends data to two destinations: an internal event queue (json.Marshal) and a customer webhook (CreateWebhookEvent). Services must pass the internal struct so internal events get full data and customer webhooks get filtered data. Three call sites pre-convert, breaking the internal event path.

**Tech Stack:** Go, RabbitMQ, gomock

---

### Task 1: Fix bin-talk-manager messagehandler — remove pre-conversion

**Files:**
- Modify: `bin-talk-manager/pkg/messagehandler/message.go:185-207`

**Step 1: Replace publishMessageCreatedEvent and publishMessageDeletedEvent**

Replace the two publish helper functions at the bottom of the file (lines 185-207) with versions that pass the internal `*message.Message` directly:

```go
// publishMessageCreatedEvent publishes a webhook event for message creation
func (h *messageHandler) publishMessageCreatedEvent(ctx context.Context, msg *message.Message) {
	h.notifyHandler.PublishWebhookEvent(ctx, msg.CustomerID, message.EventTypeMessageCreated, msg)
}

// publishMessageDeletedEvent publishes a webhook event for message deletion
func (h *messageHandler) publishMessageDeletedEvent(ctx context.Context, msg *message.Message) {
	h.notifyHandler.PublishWebhookEvent(ctx, msg.CustomerID, message.EventTypeMessageDeleted, msg)
}
```

This removes the `ConvertWebhookMessage()` calls. The `logrus` import may become unused — remove it if so.

**Step 2: Run tests**

Run: `cd bin-talk-manager && go test ./pkg/messagehandler/...`
Expected: PASS (tests already use `gomock.Any()` for the data argument)

**Step 3: Commit**

```bash
git add bin-talk-manager/pkg/messagehandler/message.go
git commit -m "NOJIRA-fix-webhook-notification-pattern

- bin-talk-manager: Pass internal Message struct directly to PublishWebhookEvent in messagehandler"
```

---

### Task 2: Fix bin-talk-manager reactionhandler — remove pre-conversion

**Files:**
- Modify: `bin-talk-manager/pkg/reactionhandler/reaction.go:137-147`

**Step 1: Replace publishReactionUpdated**

Replace the function at lines 137-147:

```go
// publishReactionUpdated publishes a single webhook event for both add and remove
func (h *reactionHandler) publishReactionUpdated(ctx context.Context, m *message.Message) {
	h.notifyHandler.PublishWebhookEvent(ctx, m.CustomerID, message.EventTypeMessageReactionUpdated, m)
}
```

This removes the `ConvertWebhookMessage()` call. The `logrus` import may become unused — remove it if so.

**Step 2: Run tests**

Run: `cd bin-talk-manager && go test ./pkg/reactionhandler/...`
Expected: PASS (tests already use `gomock.Any()` for the data argument)

**Step 3: Commit**

```bash
git add bin-talk-manager/pkg/reactionhandler/reaction.go
git commit -m "NOJIRA-fix-webhook-notification-pattern

- bin-talk-manager: Pass internal Message struct directly to PublishWebhookEvent in reactionhandler"
```

---

### Task 3: Remove unused CreateWebhookEvent on *WebhookMessage in talk-manager

**Files:**
- Modify: `bin-talk-manager/models/message/message.go:112-120`

**Step 1: Remove the CreateWebhookEvent method on *WebhookMessage**

Delete lines 112-120 (the `CreateWebhookEvent` method on `*WebhookMessage`). Keep the one on `*Message` (lines 97-110) — that's the correct implementation used by PublishWebhookEvent.

After removal, the file should end at line 110 (after `*Message`'s `CreateWebhookEvent`). The `WebhookMessage` struct itself stays — it's still used by `ConvertWebhookMessage()`.

**Step 2: Run full verification for bin-talk-manager**

Run: `cd bin-talk-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All steps PASS

**Step 3: Commit**

```bash
git add bin-talk-manager/
git commit -m "NOJIRA-fix-webhook-notification-pattern

- bin-talk-manager: Remove unused CreateWebhookEvent on WebhookMessage struct"
```

---

### Task 4: Create Speech struct in bin-transcribe-manager

**Files:**
- Create: `bin-transcribe-manager/models/streaming/speech.go`

**Step 1: Create the Speech struct file**

Create `bin-transcribe-manager/models/streaming/speech.go`:

```go
package streaming

import (
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"
)

// Speech represents a speech recognition event from a streaming session.
// This is the internal struct passed to PublishWebhookEvent.
// PublishEvent serializes all fields (including Language) for the internal queue.
// PublishWebhook calls CreateWebhookEvent() which filters to WebhookMessage.
type Speech struct {
	commonidentity.Identity

	StreamingID  uuid.UUID           `json:"streaming_id"`
	TranscribeID uuid.UUID           `json:"transcribe_id"`
	Language     string              `json:"language"`
	Direction    transcript.Direction `json:"direction"`

	Message string     `json:"message,omitempty"`
	TMEvent *time.Time `json:"tm_event"`

	TMCreate *time.Time `json:"tm_create"`
}
```

**Step 2: Verify it compiles**

Run: `cd bin-transcribe-manager && go build ./models/streaming/...`
Expected: PASS

**Step 3: Commit**

```bash
git add bin-transcribe-manager/models/streaming/speech.go
git commit -m "NOJIRA-fix-webhook-notification-pattern

- bin-transcribe-manager: Add Speech struct for speech recognition events"
```

---

### Task 5: Add NewSpeech constructor to Streaming

**Files:**
- Modify: `bin-transcribe-manager/models/streaming/streaming.go`

**Step 1: Add NewSpeech method**

Add the following method after the `Streaming` struct definition:

```go
// NewSpeech creates a Speech event from the streaming session and per-event data.
func (h *Streaming) NewSpeech(message string, tmEvent *time.Time) *Speech {
	return &Speech{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: h.CustomerID,
		},
		StreamingID:  h.ID,
		TranscribeID: h.TranscribeID,
		Language:     h.Language,
		Direction:    h.Direction,
		Message:      message,
		TMEvent:      tmEvent,
		TMCreate:     tmEvent,
	}
}
```

Add `"time"` to the imports block.

**Step 2: Verify it compiles**

Run: `cd bin-transcribe-manager && go build ./models/streaming/...`
Expected: PASS

**Step 3: Commit**

```bash
git add bin-transcribe-manager/models/streaming/streaming.go
git commit -m "NOJIRA-fix-webhook-notification-pattern

- bin-transcribe-manager: Add NewSpeech constructor to Streaming"
```

---

### Task 6: Update webhook.go — move methods to Speech, update WebhookMessage

**Files:**
- Modify: `bin-transcribe-manager/models/streaming/webhook.go`

**Step 1: Rewrite webhook.go**

Replace the entire content of `bin-transcribe-manager/models/streaming/webhook.go` with:

```go
package streaming

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines the customer-facing webhook payload for speech events
type WebhookMessage struct {
	commonidentity.Identity

	StreamingID  uuid.UUID           `json:"streaming_id"`
	TranscribeID uuid.UUID           `json:"transcribe_id"`
	Direction    transcript.Direction `json:"direction"`
	Message      string              `json:"message,omitempty"`
	TMEvent      *time.Time          `json:"tm_event"`

	TMCreate *time.Time `json:"tm_create"`
}

// ConvertWebhookMessage converts a Speech to the customer-facing WebhookMessage
func (h *Speech) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity:     h.Identity,
		StreamingID:  h.StreamingID,
		TranscribeID: h.TranscribeID,
		Direction:    h.Direction,
		Message:      h.Message,
		TMEvent:      h.TMEvent,
		TMCreate:     h.TMCreate,
	}
}

// CreateWebhookEvent implements notifyhandler.WebhookMessage interface
func (h *Speech) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()
	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return m, nil
}
```

This removes:
- `ConvertWebhookMessage()` from `*Streaming` (replaced by `NewSpeech()`)
- `CreateWebhookEvent()` from `*WebhookMessage` (moved to `*Speech`)

And adds `StreamingID` and `TMCreate` to `WebhookMessage`.

**Step 2: Verify it compiles**

Run: `cd bin-transcribe-manager && go build ./models/streaming/...`
Expected: PASS

**Step 3: Commit**

```bash
git add bin-transcribe-manager/models/streaming/webhook.go
git commit -m "NOJIRA-fix-webhook-notification-pattern

- bin-transcribe-manager: Move webhook methods from Streaming/WebhookMessage to Speech"
```

---

### Task 7: Update result.go to use NewSpeech

**Files:**
- Modify: `bin-transcribe-manager/pkg/streaminghandler/result.go:52-70`

**Step 1: Replace ConvertWebhookMessage calls with NewSpeech**

In the `process` method, replace the three `ConvertWebhookMessage` calls:

Line 52-54 (speech_started):
```go
			now := time.Now()
			evt := rp.st.NewSpeech("", &now)
			rp.notifyHandler.PublishWebhookEvent(ctx, rp.st.CustomerID, streaming.EventTypeSpeechStarted, evt)
```

Line 58-60 (speech_interim):
```go
		now := time.Now()
		evt := rp.st.NewSpeech(r.message, &now)
		rp.notifyHandler.PublishWebhookEvent(ctx, rp.st.CustomerID, streaming.EventTypeSpeechInterim, evt)
```

Line 68-70 (speech_ended):
```go
		now := time.Now()
		evt := rp.st.NewSpeech("", &now)
		rp.notifyHandler.PublishWebhookEvent(ctx, rp.st.CustomerID, streaming.EventTypeSpeechEnded, evt)
```

Variable name changes from `webhookMsg` to `evt`.

**Step 2: Run tests**

Run: `cd bin-transcribe-manager && go test ./pkg/streaminghandler/...`
Expected: PASS (tests use `gomock.Any()` for the data argument)

**Step 3: Commit**

```bash
git add bin-transcribe-manager/pkg/streaminghandler/result.go
git commit -m "NOJIRA-fix-webhook-notification-pattern

- bin-transcribe-manager: Use NewSpeech instead of ConvertWebhookMessage in result processor"
```

---

### Task 8: Run full verification for bin-transcribe-manager

**Step 1: Run full verification workflow**

Run: `cd bin-transcribe-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: All steps PASS

**Step 2: Commit any generated changes**

If `go generate` or `go mod vendor` produced changes:
```bash
git add bin-transcribe-manager/
git commit -m "NOJIRA-fix-webhook-notification-pattern

- bin-transcribe-manager: Vendor and generated file updates"
```

---

### Task 9: Final verification and conflict check

**Step 1: Fetch latest main and check for conflicts**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

Expected: No conflicts

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-fix-webhook-notification-pattern
```

Create PR with title `NOJIRA-fix-webhook-notification-pattern` and body describing all changes.
