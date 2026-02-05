package participant

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestParticipantInputStruct(t *testing.T) {
	ownerID := uuid.Must(uuid.NewV4())

	input := ParticipantInput{
		OwnerType: "agent",
		OwnerID:   ownerID,
	}

	if input.OwnerType != "agent" {
		t.Errorf("ParticipantInput.OwnerType = %v, expected %v", input.OwnerType, "agent")
	}
	if input.OwnerID != ownerID {
		t.Errorf("ParticipantInput.OwnerID = %v, expected %v", input.OwnerID, ownerID)
	}
}

func TestParticipantStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	ownerID := uuid.Must(uuid.NewV4())
	chatID := uuid.Must(uuid.NewV4())

	p := Participant{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: "agent",
			OwnerID:   ownerID,
		},
		ChatID:   chatID,
		TMJoined: timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	if p.ID != id {
		t.Errorf("Participant.ID = %v, expected %v", p.ID, id)
	}
	if p.CustomerID != customerID {
		t.Errorf("Participant.CustomerID = %v, expected %v", p.CustomerID, customerID)
	}
	if p.OwnerType != "agent" {
		t.Errorf("Participant.OwnerType = %v, expected %v", p.OwnerType, "agent")
	}
	if p.OwnerID != ownerID {
		t.Errorf("Participant.OwnerID = %v, expected %v", p.OwnerID, ownerID)
	}
	if p.ChatID != chatID {
		t.Errorf("Participant.ChatID = %v, expected %v", p.ChatID, chatID)
	}
	expectedTM := timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC))
	if p.TMJoined == nil || !p.TMJoined.Equal(*expectedTM) {
		t.Errorf("Participant.TMJoined = %v, expected %v", p.TMJoined, expectedTM)
	}
}

func TestParticipant_ConvertWebhookMessage(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	ownerID := uuid.Must(uuid.NewV4())
	chatID := uuid.Must(uuid.NewV4())

	p := Participant{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: "user",
			OwnerID:   ownerID,
		},
		ChatID:   chatID,
		TMJoined: timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	result := p.ConvertWebhookMessage()

	if result.ID != id {
		t.Errorf("WebhookMessage.ID = %v, expected %v", result.ID, id)
	}
	if result.CustomerID != customerID {
		t.Errorf("WebhookMessage.CustomerID = %v, expected %v", result.CustomerID, customerID)
	}
	if result.OwnerType != "user" {
		t.Errorf("WebhookMessage.OwnerType = %v, expected %v", result.OwnerType, "user")
	}
	if result.OwnerID != ownerID {
		t.Errorf("WebhookMessage.OwnerID = %v, expected %v", result.OwnerID, ownerID)
	}
	if result.ChatID != chatID {
		t.Errorf("WebhookMessage.ChatID = %v, expected %v", result.ChatID, chatID)
	}
}

func TestParticipant_CreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	ownerID := uuid.Must(uuid.NewV4())
	chatID := uuid.Must(uuid.NewV4())

	p := Participant{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: "agent",
			OwnerID:   ownerID,
		},
		ChatID:   chatID,
		TMJoined: timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	data, err := p.CreateWebhookEvent()
	if err != nil {
		t.Errorf("CreateWebhookEvent failed: %v", err)
		return
	}

	var wm WebhookMessage
	if err := json.Unmarshal(data, &wm); err != nil {
		t.Errorf("Failed to unmarshal webhook event: %v", err)
		return
	}

	if wm.ID != id {
		t.Errorf("WebhookMessage.ID = %v, expected %v", wm.ID, id)
	}
	if wm.ChatID != chatID {
		t.Errorf("WebhookMessage.ChatID = %v, expected %v", wm.ChatID, chatID)
	}
}

func TestGetDBFields(t *testing.T) {
	fields := GetDBFields()

	expectedFields := []string{
		"id",
		"customer_id",
		"chat_id",
		"owner_type",
		"owner_id",
		"tm_joined",
	}

	if !reflect.DeepEqual(fields, expectedFields) {
		t.Errorf("GetDBFields() = %v, expected %v", fields, expectedFields)
	}
}
