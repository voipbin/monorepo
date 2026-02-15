package message

import (
	"encoding/json"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-message-manager/models/target"
)

func TestConvertWebhookMessage(t *testing.T) {
	tmCreate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		message  *Message
		expected *WebhookMessage
	}{
		{
			name: "complete_message",
			message: &Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426614174000"),
					CustomerID: uuid.FromStringOrNil("223e4567-e89b-12d3-a456-426614174000"),
				},
				Type: TypeSMS,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+1234567890",
				},
				Targets: []target.Target{
					{
						Destination: commonaddress.Address{
							Type:   commonaddress.TypeTel,
							Target: "+0987654321",
						},
						Status: target.StatusSent,
					},
				},
				ProviderName:        ProviderNameTelnyx,
				ProviderReferenceID: "ref-12345",
				Text:                "Test message",
				Direction:           DirectionOutbound,
				TMCreate:            &tmCreate,
				TMUpdate:            &tmUpdate,
			},
			expected: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426614174000"),
					CustomerID: uuid.FromStringOrNil("223e4567-e89b-12d3-a456-426614174000"),
				},
				Type: TypeSMS,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+1234567890",
				},
				Targets: []target.Target{
					{
						Destination: commonaddress.Address{
							Type:   commonaddress.TypeTel,
							Target: "+0987654321",
						},
						Status: target.StatusSent,
					},
				},
				Text:      "Test message",
				Direction: DirectionOutbound,
				TMCreate:  &tmCreate,
				TMUpdate:  &tmUpdate,
			},
		},
		{
			name: "minimal_message",
			message: &Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("323e4567-e89b-12d3-a456-426614174000"),
					CustomerID: uuid.FromStringOrNil("423e4567-e89b-12d3-a456-426614174000"),
				},
				Type:      TypeSMS,
				Direction: DirectionInbound,
			},
			expected: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("323e4567-e89b-12d3-a456-426614174000"),
					CustomerID: uuid.FromStringOrNil("423e4567-e89b-12d3-a456-426614174000"),
				},
				Type:      TypeSMS,
				Direction: DirectionInbound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.message.ConvertWebhookMessage()

			// Check all fields
			if result.ID != tt.expected.ID {
				t.Errorf("ID mismatch: got %v, want %v", result.ID, tt.expected.ID)
			}
			if result.CustomerID != tt.expected.CustomerID {
				t.Errorf("CustomerID mismatch: got %v, want %v", result.CustomerID, tt.expected.CustomerID)
			}
			if result.Type != tt.expected.Type {
				t.Errorf("Type mismatch: got %v, want %v", result.Type, tt.expected.Type)
			}
			if result.Direction != tt.expected.Direction {
				t.Errorf("Direction mismatch: got %v, want %v", result.Direction, tt.expected.Direction)
			}
			if result.Text != tt.expected.Text {
				t.Errorf("Text mismatch: got %v, want %v", result.Text, tt.expected.Text)
			}

			// Check that provider info is NOT included
			if result.TMCreate != tt.expected.TMCreate {
				if (result.TMCreate == nil && tt.expected.TMCreate != nil) ||
					(result.TMCreate != nil && tt.expected.TMCreate == nil) ||
					!result.TMCreate.Equal(*tt.expected.TMCreate) {
					t.Errorf("TMCreate mismatch: got %v, want %v", result.TMCreate, tt.expected.TMCreate)
				}
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	tmCreate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		message     *Message
		expectError bool
		checkJSON   bool
	}{
		{
			name: "valid_message",
			message: &Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426614174000"),
					CustomerID: uuid.FromStringOrNil("223e4567-e89b-12d3-a456-426614174000"),
				},
				Type: TypeSMS,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+1234567890",
				},
				Targets: []target.Target{
					{
						Destination: commonaddress.Address{
							Type:   commonaddress.TypeTel,
							Target: "+0987654321",
						},
						Status: target.StatusSent,
					},
				},
				Text:      "Test message",
				Direction: DirectionOutbound,
				TMCreate:  &tmCreate,
			},
			expectError: false,
			checkJSON:   true,
		},
		{
			name: "minimal_message",
			message: &Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("323e4567-e89b-12d3-a456-426614174000"),
					CustomerID: uuid.FromStringOrNil("423e4567-e89b-12d3-a456-426614174000"),
				},
				Type:      TypeSMS,
				Direction: DirectionInbound,
			},
			expectError: false,
			checkJSON:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.message.CreateWebhookEvent()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Expected non-nil result")
				return
			}

			if tt.checkJSON {
				// Verify it's valid JSON
				var webhook WebhookMessage
				if err := json.Unmarshal(result, &webhook); err != nil {
					t.Errorf("Result is not valid JSON: %v", err)
				}

				// Verify essential fields are present
				if webhook.ID != tt.message.ID {
					t.Errorf("ID mismatch in JSON: got %v, want %v", webhook.ID, tt.message.ID)
				}
				if webhook.CustomerID != tt.message.CustomerID {
					t.Errorf("CustomerID mismatch in JSON: got %v, want %v", webhook.CustomerID, tt.message.CustomerID)
				}
				if webhook.Type != tt.message.Type {
					t.Errorf("Type mismatch in JSON: got %v, want %v", webhook.Type, tt.message.Type)
				}
			}
		})
	}
}

func TestWebhookMessageStruct(t *testing.T) {
	tmCreate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	id := uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426614174000")
	customerID := uuid.FromStringOrNil("223e4567-e89b-12d3-a456-426614174000")

	wm := WebhookMessage{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Type: TypeSMS,
		Source: &commonaddress.Address{
			Type:   commonaddress.TypeTel,
			Target: "+1234567890",
		},
		Targets: []target.Target{
			{
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+0987654321",
				},
				Status: target.StatusSent,
			},
		},
		Text:      "Test message",
		Direction: DirectionOutbound,
		TMCreate:  &tmCreate,
	}

	if wm.ID != id {
		t.Errorf("ID mismatch: got %v, want %v", wm.ID, id)
	}
	if wm.CustomerID != customerID {
		t.Errorf("CustomerID mismatch: got %v, want %v", wm.CustomerID, customerID)
	}
	if wm.Type != TypeSMS {
		t.Errorf("Type mismatch: got %v, want %v", wm.Type, TypeSMS)
	}
	if wm.Direction != DirectionOutbound {
		t.Errorf("Direction mismatch: got %v, want %v", wm.Direction, DirectionOutbound)
	}
	if wm.Text != "Test message" {
		t.Errorf("Text mismatch: got %v, want %v", wm.Text, "Test message")
	}
	if wm.TMCreate == nil || !wm.TMCreate.Equal(tmCreate) {
		t.Errorf("TMCreate mismatch: got %v, want %v", wm.TMCreate, tmCreate)
	}
}
