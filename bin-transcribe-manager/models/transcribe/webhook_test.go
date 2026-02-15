package transcribe

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
	activeflowID := uuid.Must(uuid.NewV4())
	onEndFlowID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	tmCreate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	tr := &Transcribe{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		ActiveflowID:  activeflowID,
		OnEndFlowID:   onEndFlowID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
		Status:        StatusProgressing,
		Language:      "en-US",
		Direction:     DirectionBoth,
		TMCreate:      &tmCreate,
		TMUpdate:      nil,
		TMDelete:      nil,
	}

	msg := tr.ConvertWebhookMessage()

	if msg.ID != id {
		t.Errorf("ConvertWebhookMessage().ID = %v, expected %v", msg.ID, id)
	}
	if msg.CustomerID != customerID {
		t.Errorf("ConvertWebhookMessage().CustomerID = %v, expected %v", msg.CustomerID, customerID)
	}
	if msg.ActiveflowID != activeflowID {
		t.Errorf("ConvertWebhookMessage().ActiveflowID = %v, expected %v", msg.ActiveflowID, activeflowID)
	}
	if msg.OnEndFlowID != onEndFlowID {
		t.Errorf("ConvertWebhookMessage().OnEndFlowID = %v, expected %v", msg.OnEndFlowID, onEndFlowID)
	}
	if msg.ReferenceType != ReferenceTypeCall {
		t.Errorf("ConvertWebhookMessage().ReferenceType = %v, expected %v", msg.ReferenceType, ReferenceTypeCall)
	}
	if msg.ReferenceID != referenceID {
		t.Errorf("ConvertWebhookMessage().ReferenceID = %v, expected %v", msg.ReferenceID, referenceID)
	}
	if msg.Status != StatusProgressing {
		t.Errorf("ConvertWebhookMessage().Status = %v, expected %v", msg.Status, StatusProgressing)
	}
	if msg.Language != "en-US" {
		t.Errorf("ConvertWebhookMessage().Language = %v, expected %v", msg.Language, "en-US")
	}
	if msg.Direction != DirectionBoth {
		t.Errorf("ConvertWebhookMessage().Direction = %v, expected %v", msg.Direction, DirectionBoth)
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	tr := &Transcribe{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   uuid.Must(uuid.NewV4()),
		Status:        StatusProgressing,
		Language:      "en-US",
		Direction:     DirectionIn,
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
