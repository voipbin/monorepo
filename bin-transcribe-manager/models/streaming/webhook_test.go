package streaming

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
)

func Test_Speech_CreateWebhookEvent(t *testing.T) {
	now := time.Now()
	st := &Streaming{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
		},
		TranscribeID: uuid.FromStringOrNil("t0000000-0000-0000-0000-000000000001"),
		Language:     "en-US",
		Direction:    transcript.DirectionIn,
	}

	speech := st.NewSpeech("hello world", &now)

	if speech.CustomerID != st.CustomerID {
		t.Errorf("expected CustomerID %s, got %s", st.CustomerID, speech.CustomerID)
	}
	if speech.StreamingID != st.ID {
		t.Errorf("expected StreamingID %s, got %s", st.ID, speech.StreamingID)
	}
	if speech.TranscribeID != st.TranscribeID {
		t.Errorf("expected TranscribeID %s, got %s", st.TranscribeID, speech.TranscribeID)
	}
	if speech.Language != "en-US" {
		t.Errorf("expected Language en-US, got %s", speech.Language)
	}
	if speech.Direction != transcript.DirectionIn {
		t.Errorf("expected direction in, got %s", speech.Direction)
	}
	if speech.Message != "hello world" {
		t.Errorf("expected message 'hello world', got '%s'", speech.Message)
	}

	// verify ConvertWebhookMessage omits Language
	msg := speech.ConvertWebhookMessage()
	if msg.StreamingID != st.ID {
		t.Errorf("expected WebhookMessage.StreamingID %s, got %s", st.ID, msg.StreamingID)
	}

	// verify CreateWebhookEvent returns valid JSON via the interface
	data, err := speech.CreateWebhookEvent()
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
	// Language should NOT appear in the webhook JSON
	if _, exists := parsed["language"]; exists {
		t.Error("expected language to be omitted from webhook JSON")
	}
}

func Test_Speech_CreateWebhookEvent_EmptyMessage(t *testing.T) {
	now := time.Now()
	st := &Streaming{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
		},
		TranscribeID: uuid.FromStringOrNil("t0000000-0000-0000-0000-000000000001"),
		Language:     "en-US",
		Direction:    transcript.DirectionOut,
	}

	speech := st.NewSpeech("", &now)
	data, err := speech.CreateWebhookEvent()
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
