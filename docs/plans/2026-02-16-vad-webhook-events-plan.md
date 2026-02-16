# VAD Webhook Events Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Publish VAD webhook events (speech_started, speech_interim, speech_ended) derived from STT provider responses in bin-transcribe-manager.

**Architecture:** Track a per-stream `speaking` boolean in the GCP/AWS result-processing goroutines. When interim results arrive, derive speech_started/speech_interim events. When final results arrive, derive speech_ended. Publish via existing `notifyHandler.PublishWebhookEvent()`. No DB, no new API endpoints.

**Tech Stack:** Go, GCP Speech-to-Text streaming API, AWS Transcribe Streaming, RabbitMQ webhooks via notifyhandler

**Worktree:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-Publish-VAD-webhook-events/`

**All file paths below are relative to:** `bin-transcribe-manager/`

---

### Task 1: Add VAD event type constants

**Files:**
- Modify: `models/streaming/event.go`

**Step 1: Add three new event type constants**

Open `models/streaming/event.go` and add three constants to the existing const block:

```go
package streaming

// list of streaming event types
const (
	EventTypeStreamingStarted string = "streaming_started"
	EventTypeStreamingStopped string = "streaming_stopped"

	EventTypeSpeechStarted string = "transcribe_speech_started"
	EventTypeSpeechInterim string = "transcribe_speech_interim"
	EventTypeSpeechEnded   string = "transcribe_speech_ended"
)
```

**Step 2: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Publish-VAD-webhook-events/bin-transcribe-manager && go build ./...`
Expected: no errors

**Step 3: Commit**

```bash
git add models/streaming/event.go
git commit -m "NOJIRA-Publish-VAD-webhook-events

- bin-transcribe-manager: Add VAD event type constants (speech_started, speech_interim, speech_ended)"
```

---

### Task 2: Create streaming WebhookMessage

**Files:**
- Create: `models/streaming/webhook.go`

**Step 1: Write the test**

Create `models/streaming/webhook_test.go`:

```go
package streaming

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
)

func Test_WebhookMessage_CreateWebhookEvent(t *testing.T) {
	now := time.Now()
	st := &Streaming{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
		},
		TranscribeID: uuid.FromStringOrNil("t0000000-0000-0000-0000-000000000001"),
		Direction:    transcript.DirectionIn,
	}

	msg := st.ConvertWebhookMessage("hello world", &now)

	if msg.ID != st.ID {
		t.Errorf("expected ID %s, got %s", st.ID, msg.ID)
	}
	if msg.CustomerID != st.CustomerID {
		t.Errorf("expected CustomerID %s, got %s", st.CustomerID, msg.CustomerID)
	}
	if msg.TranscribeID != st.TranscribeID {
		t.Errorf("expected TranscribeID %s, got %s", st.TranscribeID, msg.TranscribeID)
	}
	if msg.Direction != transcript.DirectionIn {
		t.Errorf("expected direction in, got %s", msg.Direction)
	}
	if msg.Message != "hello world" {
		t.Errorf("expected message 'hello world', got '%s'", msg.Message)
	}
	if msg.TMEvent != &now {
		t.Errorf("expected TMEvent to match")
	}

	// verify CreateWebhookEvent returns valid JSON
	data, err := st.CreateWebhookEvent("test message", &now)
	if err != nil {
		t.Fatalf("CreateWebhookEvent failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("CreateWebhookEvent returned invalid JSON: %v", err)
	}

	if parsed["transcribe_id"] != st.TranscribeID.String() {
		t.Errorf("expected transcribe_id %s in JSON, got %v", st.TranscribeID, parsed["transcribe_id"])
	}
	if parsed["message"] != "test message" {
		t.Errorf("expected message 'test message' in JSON, got %v", parsed["message"])
	}
}

func Test_WebhookMessage_CreateWebhookEvent_EmptyMessage(t *testing.T) {
	now := time.Now()
	st := &Streaming{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
		},
		TranscribeID: uuid.FromStringOrNil("t0000000-0000-0000-0000-000000000001"),
		Direction:    transcript.DirectionOut,
	}

	data, err := st.CreateWebhookEvent("", &now)
	if err != nil {
		t.Fatalf("CreateWebhookEvent failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// message should be omitted when empty (omitempty)
	if _, exists := parsed["message"]; exists {
		t.Error("expected message to be omitted when empty")
	}
}
```

**Step 2: Run the test to verify it fails**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Publish-VAD-webhook-events/bin-transcribe-manager && go test ./models/streaming/...`
Expected: FAIL — `ConvertWebhookMessage` and `CreateWebhookEvent` not defined

**Step 3: Implement WebhookMessage**

Create `models/streaming/webhook.go`:

```go
package streaming

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines the webhook payload for VAD events
type WebhookMessage struct {
	commonidentity.Identity

	TranscribeID uuid.UUID            `json:"transcribe_id"`
	Direction    transcript.Direction `json:"direction"`
	Message      string               `json:"message,omitempty"`
	TMEvent      *time.Time           `json:"tm_event"`
}

// ConvertWebhookMessage converts a Streaming to a WebhookMessage
func (h *Streaming) ConvertWebhookMessage(message string, tmEvent *time.Time) *WebhookMessage {
	return &WebhookMessage{
		Identity:     h.Identity,
		TranscribeID: h.TranscribeID,
		Direction:    h.Direction,
		Message:      message,
		TMEvent:      tmEvent,
	}
}

// CreateWebhookEvent generates a WebhookEvent JSON payload
func (h *Streaming) CreateWebhookEvent(message string, tmEvent *time.Time) ([]byte, error) {
	e := h.ConvertWebhookMessage(message, tmEvent)

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
```

**Important note:** The standard `notifyhandler.WebhookMessage` interface expects `CreateWebhookEvent() ([]byte, error)` with no arguments. However, `Streaming` is not a persisted model with all data — message and tmEvent vary per event. So we need a **wrapper type** that holds the data and implements the interface. Update the implementation:

```go
package streaming

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines the webhook payload for VAD events
type WebhookMessage struct {
	commonidentity.Identity

	TranscribeID uuid.UUID            `json:"transcribe_id"`
	Direction    transcript.Direction `json:"direction"`
	Message      string               `json:"message,omitempty"`
	TMEvent      *time.Time           `json:"tm_event"`
}

// ConvertWebhookMessage converts a Streaming to a WebhookMessage
func (h *Streaming) ConvertWebhookMessage(message string, tmEvent *time.Time) *WebhookMessage {
	return &WebhookMessage{
		Identity:     h.Identity,
		TranscribeID: h.TranscribeID,
		Direction:    h.Direction,
		Message:      message,
		TMEvent:      tmEvent,
	}
}

// CreateWebhookEvent implements notifyhandler.WebhookMessage interface
func (h *WebhookMessage) CreateWebhookEvent() ([]byte, error) {
	m, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}

	return m, nil
}
```

**Step 4: Update the test to match the interface pattern**

The test should call `ConvertWebhookMessage()` to get a `*WebhookMessage`, then call `CreateWebhookEvent()` on that:

```go
package streaming

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
)

func Test_WebhookMessage_CreateWebhookEvent(t *testing.T) {
	now := time.Now()
	st := &Streaming{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
		},
		TranscribeID: uuid.FromStringOrNil("t0000000-0000-0000-0000-000000000001"),
		Direction:    transcript.DirectionIn,
	}

	msg := st.ConvertWebhookMessage("hello world", &now)

	if msg.ID != st.ID {
		t.Errorf("expected ID %s, got %s", st.ID, msg.ID)
	}
	if msg.CustomerID != st.CustomerID {
		t.Errorf("expected CustomerID %s, got %s", st.CustomerID, msg.CustomerID)
	}
	if msg.TranscribeID != st.TranscribeID {
		t.Errorf("expected TranscribeID %s, got %s", st.TranscribeID, msg.TranscribeID)
	}
	if msg.Direction != transcript.DirectionIn {
		t.Errorf("expected direction in, got %s", msg.Direction)
	}
	if msg.Message != "hello world" {
		t.Errorf("expected message 'hello world', got '%s'", msg.Message)
	}

	// verify CreateWebhookEvent returns valid JSON via the interface
	data, err := msg.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("CreateWebhookEvent failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("CreateWebhookEvent returned invalid JSON: %v", err)
	}

	if parsed["transcribe_id"] != st.TranscribeID.String() {
		t.Errorf("expected transcribe_id %s in JSON, got %v", st.TranscribeID, parsed["transcribe_id"])
	}
	if parsed["message"] != "hello world" {
		t.Errorf("expected message 'hello world' in JSON, got %v", parsed["message"])
	}
}

func Test_WebhookMessage_CreateWebhookEvent_EmptyMessage(t *testing.T) {
	now := time.Now()
	st := &Streaming{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
		},
		TranscribeID: uuid.FromStringOrNil("t0000000-0000-0000-0000-000000000001"),
		Direction:    transcript.DirectionOut,
	}

	msg := st.ConvertWebhookMessage("", &now)
	data, err := msg.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("CreateWebhookEvent failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// message should be omitted when empty (omitempty)
	if _, exists := parsed["message"]; exists {
		t.Error("expected message to be omitted when empty")
	}
}
```

**Step 5: Run tests to verify they pass**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Publish-VAD-webhook-events/bin-transcribe-manager && go test ./models/streaming/...`
Expected: PASS

**Step 6: Commit**

```bash
git add models/streaming/webhook.go models/streaming/webhook_test.go
git commit -m "NOJIRA-Publish-VAD-webhook-events

- bin-transcribe-manager: Add streaming WebhookMessage for VAD events with tests"
```

---

### Task 3: Add VAD event publishing to GCP handler

**Files:**
- Modify: `pkg/streaminghandler/gcp.go`

**Step 1: Modify gcpProcessResult to track speaking state and publish VAD events**

Replace the entire `gcpProcessResult` function in `pkg/streaminghandler/gcp.go`:

```go
// gcpProcessResult handles transcript result from the google stt
func (h *streamingHandler) gcpProcessResult(ctx context.Context, cancel context.CancelFunc, st *streaming.Streaming, streamClient speechpb.Speech_StreamingRecognizeClient) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "gcpProcessResult",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})
	log.Debugf("Starting gcpProcessResult. transcribe_id: %s", st.TranscribeID)

	defer func() {
		log.Debugf("Finished gcpProcessResult. transcribe_id: %s", st.TranscribeID)
		cancel()
	}()

	speaking := false
	t1 := time.Now()
	for {
		if ctx.Err() != nil {
			log.Debugf("Context has finsished. transcribe_id: %s, streaming_id: %s", st.TranscribeID, st.ID)
			return
		}

		tmp, err := streamClient.Recv()
		if err != nil {
			log.Debugf("Could not received the result. Consider this hangup. err: %v", err)
			return
		} else if len(tmp.Results) == 0 {
			// no result
			continue
		}

		if !tmp.Results[0].IsFinal {
			// interim result — publish VAD events
			message := ""
			if len(tmp.Results[0].Alternatives) > 0 {
				message = tmp.Results[0].Alternatives[0].Transcript
			}

			if !speaking {
				speaking = true
				now := time.Now()
				webhookMsg := st.ConvertWebhookMessage("", &now)
				h.notifyHandler.PublishWebhookEvent(ctx, st.CustomerID, streaming.EventTypeSpeechStarted, webhookMsg)
				log.Debugf("Published speech_started. transcribe_id: %s, direction: %s", st.TranscribeID, st.Direction)
			}

			now := time.Now()
			webhookMsg := st.ConvertWebhookMessage(message, &now)
			h.notifyHandler.PublishWebhookEvent(ctx, st.CustomerID, streaming.EventTypeSpeechInterim, webhookMsg)
			log.Debugf("Published speech_interim. transcribe_id: %s, direction: %s, message: %s", st.TranscribeID, st.Direction, message)
			continue
		}

		// final result — publish speech_ended if was speaking
		if speaking {
			speaking = false
			now := time.Now()
			webhookMsg := st.ConvertWebhookMessage("", &now)
			h.notifyHandler.PublishWebhookEvent(ctx, st.CustomerID, streaming.EventTypeSpeechEnded, webhookMsg)
			log.Debugf("Published speech_ended. transcribe_id: %s, direction: %s", st.TranscribeID, st.Direction)
		}

		// get transcript message and create transcript
		message := tmp.Results[0].Alternatives[0].Transcript
		if len(message) == 0 {
			continue
		}
		log.Debugf("Received transcript message. transcribe_id: %s, direction: %s, message: %s", st.TranscribeID, st.Direction, message)

		t2 := time.Now()
		t3 := t2.Sub(t1)
		tmGap := time.Time{}.Add(t3)

		// create transcript
		ts, err := h.transcriptHandler.Create(ctx, st.CustomerID, st.TranscribeID, st.Direction, message, &tmGap)
		if err != nil {
			log.Errorf("Could not create transript. err: %v", err)
			break
		}
		log.WithField("transcript", ts).Debugf("Created transcript. transcribe_id: %s, direction: %s", ts.TranscribeID, ts.Direction)
	}
}
```

Key changes from original:
1. Added `speaking` boolean before the loop
2. Removed `time.Sleep(100ms)` on interim results
3. On interim: if not speaking → publish `speech_started` + `speech_interim`; if already speaking → publish `speech_interim` only
4. On final: if was speaking → publish `speech_ended`, then existing transcript logic unchanged

**Step 2: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Publish-VAD-webhook-events/bin-transcribe-manager && go build ./...`
Expected: no errors

**Step 3: Run all existing tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Publish-VAD-webhook-events/bin-transcribe-manager && go test ./...`
Expected: PASS (no existing gcp tests are affected)

**Step 4: Commit**

```bash
git add pkg/streaminghandler/gcp.go
git commit -m "NOJIRA-Publish-VAD-webhook-events

- bin-transcribe-manager: Add VAD event publishing to GCP streaming handler"
```

---

### Task 4: Add VAD event publishing to AWS handler

**Files:**
- Modify: `pkg/streaminghandler/aws.go`

**Step 1: Modify awsProcessResult to track speaking state and publish VAD events**

Replace the entire `awsProcessResult` function in `pkg/streaminghandler/aws.go`:

```go
// awsProcessResult handles transcript results from AWS Transcribe
func (h *streamingHandler) awsProcessResult(ctx context.Context, cancel context.CancelFunc, st *streaming.Streaming, streamClient *transcribestreaming.StartStreamTranscriptionOutput) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "awsProcessResult",
		"streaming_id":  st.ID,
		"transcribe_id": st.TranscribeID,
	})
	log.Debugf("Starting awsProcessResult. transcribe_id: %s", st.TranscribeID)

	defer func() {
		log.Debugf("Finished awsProcessResult. transcribe_id: %s", st.TranscribeID)
		cancel()
	}()

	stream := streamClient.GetStream()
	defer func() {
		_ = stream.Close()
	}()

	speaking := false
	t1 := time.Now()
	for {
		select {
		case <-ctx.Done():
			log.Debug("Context canceled, stopping awsProcessResult.")
			return

		case event, ok := <-stream.Events():
			if !ok {
				log.Debug("TranscriptResultStream closed.")
				return
			}

			transcriptEvent, ok := event.(*types.TranscriptResultStreamMemberTranscriptEvent)
			if !ok {
				continue
			}

			for _, result := range transcriptEvent.Value.Transcript.Results {
				if len(result.Alternatives) == 0 {
					continue
				}

				if result.IsPartial {
					// partial result — publish VAD events
					message := ""
					if result.Alternatives[0].Transcript != nil {
						message = *result.Alternatives[0].Transcript
					}

					if !speaking {
						speaking = true
						now := time.Now()
						webhookMsg := st.ConvertWebhookMessage("", &now)
						h.notifyHandler.PublishWebhookEvent(ctx, st.CustomerID, streaming.EventTypeSpeechStarted, webhookMsg)
						log.Debugf("Published speech_started. transcribe_id: %s, direction: %s", st.TranscribeID, st.Direction)
					}

					now := time.Now()
					webhookMsg := st.ConvertWebhookMessage(message, &now)
					h.notifyHandler.PublishWebhookEvent(ctx, st.CustomerID, streaming.EventTypeSpeechInterim, webhookMsg)
					log.Debugf("Published speech_interim. transcribe_id: %s, direction: %s, message: %s", st.TranscribeID, st.Direction, message)
					continue
				}

				// final result — publish speech_ended if was speaking
				if speaking {
					speaking = false
					now := time.Now()
					webhookMsg := st.ConvertWebhookMessage("", &now)
					h.notifyHandler.PublishWebhookEvent(ctx, st.CustomerID, streaming.EventTypeSpeechEnded, webhookMsg)
					log.Debugf("Published speech_ended. transcribe_id: %s, direction: %s", st.TranscribeID, st.Direction)
				}

				message := *result.Alternatives[0].Transcript
				if len(message) == 0 {
					continue
				}
				log.Debugf("Received transcript message. transcribe_id: %s, direction: %s, message: %s", st.TranscribeID, st.Direction, message)

				t2 := time.Now()
				t3 := t2.Sub(t1)
				tmGap := time.Time{}.Add(t3)

				// create transcript
				ts, err := h.transcriptHandler.Create(ctx, st.CustomerID, st.TranscribeID, st.Direction, message, &tmGap)
				if err != nil {
					log.Errorf("Could not create transript. err: %v", err)
					break
				}
				log.WithField("transcript", ts).Debugf("Created transcript. transcribe_id: %s, direction: %s", ts.TranscribeID, ts.Direction)
			}
		}
	}
}
```

Key changes from original:
1. Added `speaking` boolean before the loop
2. Removed the `if result.IsPartial` skip — now processes partial results for VAD events
3. On partial: if not speaking → publish `speech_started` + `speech_interim`; if already speaking → publish `speech_interim` only
4. On non-partial: if was speaking → publish `speech_ended`, then existing transcript logic unchanged

**Step 2: Verify it compiles**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Publish-VAD-webhook-events/bin-transcribe-manager && go build ./...`
Expected: no errors

**Step 3: Run all existing tests**

Run: `cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Publish-VAD-webhook-events/bin-transcribe-manager && go test ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add pkg/streaminghandler/aws.go
git commit -m "NOJIRA-Publish-VAD-webhook-events

- bin-transcribe-manager: Add VAD event publishing to AWS streaming handler"
```

---

### Task 5: Run full verification workflow

**Step 1: Run the complete verification**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Publish-VAD-webhook-events/bin-transcribe-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: all steps pass

**Step 2: Fix any lint or test issues**

If golangci-lint reports issues, fix them and re-run the workflow.

**Step 3: Final commit if any fixes were needed**

Only commit if there were fixes from step 2.

---

### Task 6: Push and create PR

**Step 1: Push branch**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-Publish-VAD-webhook-events && \
git push -u origin NOJIRA-Publish-VAD-webhook-events
```

**Step 2: Create PR**

```bash
gh pr create --title "NOJIRA-Publish-VAD-webhook-events" --body "$(cat <<'EOF'
Publish VAD (Voice Activity Detection) webhook events derived from STT provider
responses in bin-transcribe-manager. Events are fired during real-time
transcription when speech activity is detected via interim/partial results from
GCP and AWS STT providers.

- bin-transcribe-manager: Add VAD event type constants (transcribe_speech_started, transcribe_speech_interim, transcribe_speech_ended)
- bin-transcribe-manager: Add streaming WebhookMessage for VAD webhook payloads
- bin-transcribe-manager: Publish VAD events from GCP streaming handler on interim/final results
- bin-transcribe-manager: Publish VAD events from AWS streaming handler on partial/final results
EOF
)"
```
