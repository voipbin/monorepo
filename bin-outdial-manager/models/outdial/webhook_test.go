package outdial

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestConvertWebhookMessage(t *testing.T) {
	tmCreate := time.Now()
	tmUpdate := time.Now().Add(1 * time.Hour)

	tests := []struct {
		name    string
		outdial *Outdial
	}{
		{
			name: "converts outdial with all fields",
			outdial: &Outdial{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				CampaignID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
				Name:       "Test Outdial",
				Detail:     "Test Detail",
				Data:       `{"key": "value"}`,
				TMCreate:   &tmCreate,
				TMUpdate:   &tmUpdate,
				TMDelete:   nil,
			},
		},
		{
			name: "converts outdial with minimal fields",
			outdial: &Outdial{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				CampaignID: uuid.Nil,
				Name:       "",
				Detail:     "",
				Data:       "",
				TMCreate:   nil,
				TMUpdate:   nil,
				TMDelete:   nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.outdial.ConvertWebhookMessage()

			if result == nil {
				t.Fatal("Expected webhook message but got nil")
			}

			if result.ID != tt.outdial.ID {
				t.Errorf("Expected ID %v, got %v", tt.outdial.ID, result.ID)
			}
			if result.CustomerID != tt.outdial.CustomerID {
				t.Errorf("Expected CustomerID %v, got %v", tt.outdial.CustomerID, result.CustomerID)
			}
			if result.CampaignID != tt.outdial.CampaignID {
				t.Errorf("Expected CampaignID %v, got %v", tt.outdial.CampaignID, result.CampaignID)
			}
			if result.Name != tt.outdial.Name {
				t.Errorf("Expected Name %s, got %s", tt.outdial.Name, result.Name)
			}
			if result.Detail != tt.outdial.Detail {
				t.Errorf("Expected Detail %s, got %s", tt.outdial.Detail, result.Detail)
			}
			if result.Data != tt.outdial.Data {
				t.Errorf("Expected Data %s, got %s", tt.outdial.Data, result.Data)
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	tmCreate := time.Now()

	tests := []struct {
		name      string
		outdial   *Outdial
		expectErr bool
	}{
		{
			name: "creates valid webhook event",
			outdial: &Outdial{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				CampaignID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
				Name:       "Test Outdial",
				Detail:     "Test Detail",
				Data:       `{"key": "value"}`,
				TMCreate:   &tmCreate,
			},
			expectErr: false,
		},
		{
			name: "creates webhook event with empty fields",
			outdial: &Outdial{
				Identity: commonidentity.Identity{
					ID:         uuid.Nil,
					CustomerID: uuid.Nil,
				},
				CampaignID: uuid.Nil,
				Name:       "",
				Detail:     "",
				Data:       "",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.outdial.CreateWebhookEvent()

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result == nil {
					t.Fatal("Expected result but got nil")
				}

				// Verify it's valid JSON
				var webhookMsg WebhookMessage
				if err := json.Unmarshal(result, &webhookMsg); err != nil {
					t.Errorf("Failed to unmarshal webhook event: %v", err)
				}

				// Verify content matches
				if webhookMsg.ID != tt.outdial.ID {
					t.Errorf("Expected ID %v, got %v", tt.outdial.ID, webhookMsg.ID)
				}
				if webhookMsg.Name != tt.outdial.Name {
					t.Errorf("Expected Name %s, got %s", tt.outdial.Name, webhookMsg.Name)
				}
			}
		})
	}
}

func TestWebhookMessage_JSONMarshaling(t *testing.T) {
	tmCreate := time.Now()
	msg := &WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
		},
		CampaignID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
		Name:       "Test",
		Detail:     "Detail",
		Data:       "Data",
		TMCreate:   &tmCreate,
	}

	// Test marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal webhook message: %v", err)
	}

	// Test unmarshaling
	var unmarshaled WebhookMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal webhook message: %v", err)
	}

	if unmarshaled.ID != msg.ID {
		t.Errorf("ID mismatch after marshal/unmarshal")
	}
	if unmarshaled.Name != msg.Name {
		t.Errorf("Name mismatch after marshal/unmarshal")
	}
}
