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
