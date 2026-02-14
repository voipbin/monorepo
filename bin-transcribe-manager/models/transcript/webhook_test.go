package transcript

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestConvertWebhookMessage(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	transcribeID := uuid.Must(uuid.NewV4())

	tmTranscript := time.Date(2023, 1, 1, 0, 0, 1, 0, time.UTC)
	tmCreate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	tr := &Transcript{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,
		Direction:    DirectionIn,
		Message:      "test message",
		TMTranscript: &tmTranscript,
		TMCreate:     &tmCreate,
	}

	msg := tr.ConvertWebhookMessage()

	if msg.ID != id {
		t.Errorf("ConvertWebhookMessage().ID = %v, expected %v", msg.ID, id)
	}
	if msg.CustomerID != customerID {
		t.Errorf("ConvertWebhookMessage().CustomerID = %v, expected %v", msg.CustomerID, customerID)
	}
	if msg.TranscribeID != transcribeID {
		t.Errorf("ConvertWebhookMessage().TranscribeID = %v, expected %v", msg.TranscribeID, transcribeID)
	}
	if msg.Direction != DirectionIn {
		t.Errorf("ConvertWebhookMessage().Direction = %v, expected %v", msg.Direction, DirectionIn)
	}
	if msg.Message != "test message" {
		t.Errorf("ConvertWebhookMessage().Message = %v, expected %v", msg.Message, "test message")
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	transcribeID := uuid.Must(uuid.NewV4())

	tr := &Transcript{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,
		Direction:    DirectionOut,
		Message:      "test message",
	}

	data, err := tr.CreateWebhookEvent()
	if err != nil {
		t.Errorf("CreateWebhookEvent() error = %v, expected nil", err)
	}
	if data == nil {
		t.Error("CreateWebhookEvent() returned nil data")
	}

	// Verify it's valid JSON
	var msg WebhookMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Errorf("CreateWebhookEvent() produced invalid JSON: %v", err)
	}

	if msg.ID != id {
		t.Errorf("CreateWebhookEvent() ID = %v, expected %v", msg.ID, id)
	}
}
