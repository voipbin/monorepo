package streaming

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
)

func Test_NewSpeech(t *testing.T) {
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

	// ID must be a new non-nil UUID
	if speech.ID == uuid.Nil {
		t.Error("expected non-nil UUID for Speech.ID")
	}
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
	if speech.TMEvent != &now {
		t.Errorf("expected TMEvent to point to provided time")
	}
	if speech.TMCreate != &now {
		t.Errorf("expected TMCreate to equal TMEvent")
	}
}

func Test_ConvertWebhookMessage(t *testing.T) {
	now := time.Now()
	speech := &Speech{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("s0000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
		},
		StreamingID:  uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
		TranscribeID: uuid.FromStringOrNil("t0000000-0000-0000-0000-000000000001"),
		Language:     "en-US",
		Direction:    transcript.DirectionIn,
		Message:      "test message",
		TMEvent:      &now,
		TMCreate:     &now,
	}

	msg := speech.ConvertWebhookMessage()

	if msg.ID != speech.ID {
		t.Errorf("expected ID %s, got %s", speech.ID, msg.ID)
	}
	if msg.CustomerID != speech.CustomerID {
		t.Errorf("expected CustomerID %s, got %s", speech.CustomerID, msg.CustomerID)
	}
	if msg.StreamingID != speech.StreamingID {
		t.Errorf("expected StreamingID %s, got %s", speech.StreamingID, msg.StreamingID)
	}
	if msg.TranscribeID != speech.TranscribeID {
		t.Errorf("expected TranscribeID %s, got %s", speech.TranscribeID, msg.TranscribeID)
	}
	if msg.Direction != speech.Direction {
		t.Errorf("expected Direction %s, got %s", speech.Direction, msg.Direction)
	}
	if msg.Message != speech.Message {
		t.Errorf("expected Message %s, got %s", speech.Message, msg.Message)
	}
	if msg.TMEvent != speech.TMEvent {
		t.Errorf("expected TMEvent to match")
	}
	if msg.TMCreate != speech.TMCreate {
		t.Errorf("expected TMCreate to match")
	}
}

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

	data, err := speech.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("CreateWebhookEvent failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("CreateWebhookEvent returned invalid JSON: %v", err)
	}

	if parsed["streaming_id"] != st.ID.String() {
		t.Errorf("expected streaming_id %s in JSON, got %v", st.ID, parsed["streaming_id"])
	}
	if parsed["transcribe_id"] != st.TranscribeID.String() {
		t.Errorf("expected transcribe_id %s in JSON, got %v", st.TranscribeID, parsed["transcribe_id"])
	}
	if parsed["message"] != "hello world" {
		t.Errorf("expected message 'hello world' in JSON, got %v", parsed["message"])
	}
	if parsed["direction"] != string(transcript.DirectionIn) {
		t.Errorf("expected direction 'in' in JSON, got %v", parsed["direction"])
	}
	// Language should NOT appear in the webhook JSON
	if _, exists := parsed["language"]; exists {
		t.Error("expected language to be omitted from webhook JSON")
	}
	// tm_event and tm_create should be present
	if _, exists := parsed["tm_event"]; !exists {
		t.Error("expected tm_event in webhook JSON")
	}
	if _, exists := parsed["tm_create"]; !exists {
		t.Error("expected tm_create in webhook JSON")
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
