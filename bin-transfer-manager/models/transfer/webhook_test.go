package transfer

import (
	"encoding/json"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestConvertWebhookMessage(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		transfer *Transfer
		expected *WebhookMessage
	}{
		{
			name: "converts_attended_transfer",
			transfer: &Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             TypeAttended,
				TransfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
				TransfereeAddresses: []commonaddress.Address{
					{Type: commonaddress.TypeTel, Target: "+821100000001"},
				},
				TransfereeCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
				GroupcallID:      uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
				ConfbridgeID:     uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
				TMCreate:         &now,
				TMUpdate:         nil,
				TMDelete:         nil,
			},
			expected: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             TypeAttended,
				TransfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
				TransfereeAddresses: []commonaddress.Address{
					{Type: commonaddress.TypeTel, Target: "+821100000001"},
				},
				TransfereeCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
				GroupcallID:      uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
				ConfbridgeID:     uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
				TMCreate:         &now,
				TMUpdate:         nil,
				TMDelete:         nil,
			},
		},
		{
			name: "converts_blind_transfer",
			transfer: &Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             TypeBlind,
				TransfererCallID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440002"),
				TransfereeAddresses: []commonaddress.Address{
					{Type: commonaddress.TypeSIP, Target: "sip:user@domain.com"},
				},
				TransfereeCallID: uuid.Nil,
				GroupcallID:      uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440004"),
				ConfbridgeID:     uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440005"),
				TMCreate:         &now,
				TMUpdate:         &now,
				TMDelete:         nil,
			},
			expected: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             TypeBlind,
				TransfererCallID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440002"),
				TransfereeAddresses: []commonaddress.Address{
					{Type: commonaddress.TypeSIP, Target: "sip:user@domain.com"},
				},
				TransfereeCallID: uuid.Nil,
				GroupcallID:      uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440004"),
				ConfbridgeID:     uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440005"),
				TMCreate:         &now,
				TMUpdate:         &now,
				TMDelete:         nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.transfer.ConvertWebhookMessage()

			if result.ID != tt.expected.ID {
				t.Errorf("Wrong ID. expect: %s, got: %s", tt.expected.ID, result.ID)
			}
			if result.CustomerID != tt.expected.CustomerID {
				t.Errorf("Wrong CustomerID. expect: %s, got: %s", tt.expected.CustomerID, result.CustomerID)
			}
			if result.Type != tt.expected.Type {
				t.Errorf("Wrong Type. expect: %s, got: %s", tt.expected.Type, result.Type)
			}
			if result.TransfererCallID != tt.expected.TransfererCallID {
				t.Errorf("Wrong TransfererCallID. expect: %s, got: %s", tt.expected.TransfererCallID, result.TransfererCallID)
			}
			if result.TransfereeCallID != tt.expected.TransfereeCallID {
				t.Errorf("Wrong TransfereeCallID. expect: %s, got: %s", tt.expected.TransfereeCallID, result.TransfereeCallID)
			}
			if result.GroupcallID != tt.expected.GroupcallID {
				t.Errorf("Wrong GroupcallID. expect: %s, got: %s", tt.expected.GroupcallID, result.GroupcallID)
			}
			if result.ConfbridgeID != tt.expected.ConfbridgeID {
				t.Errorf("Wrong ConfbridgeID. expect: %s, got: %s", tt.expected.ConfbridgeID, result.ConfbridgeID)
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		transfer    *Transfer
		shouldError bool
	}{
		{
			name: "creates_webhook_event_successfully",
			transfer: &Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				Type:             TypeAttended,
				TransfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
				TransfereeAddresses: []commonaddress.Address{
					{Type: commonaddress.TypeTel, Target: "+821100000001"},
				},
				TransfereeCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
				GroupcallID:      uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
				ConfbridgeID:     uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
				TMCreate:         &now,
			},
			shouldError: false,
		},
		{
			name: "creates_webhook_event_with_nil_values",
			transfer: &Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440001"),
				},
				Type:                TypeBlind,
				TransfererCallID:    uuid.Nil,
				TransfereeAddresses: []commonaddress.Address{},
				TransfereeCallID:    uuid.Nil,
				GroupcallID:         uuid.Nil,
				ConfbridgeID:        uuid.Nil,
				TMCreate:            nil,
				TMUpdate:            nil,
				TMDelete:            nil,
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.transfer.CreateWebhookEvent()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if data == nil {
					t.Error("Expected data but got nil")
				}

				// Verify the data can be unmarshaled back to WebhookMessage
				var webhook WebhookMessage
				if err := json.Unmarshal(data, &webhook); err != nil {
					t.Errorf("Failed to unmarshal webhook data: %v", err)
				}

				// Verify the unmarshaled data matches
				if webhook.ID != tt.transfer.ID {
					t.Errorf("Wrong ID in webhook. expect: %s, got: %s", tt.transfer.ID, webhook.ID)
				}
				if webhook.Type != tt.transfer.Type {
					t.Errorf("Wrong Type in webhook. expect: %s, got: %s", tt.transfer.Type, webhook.Type)
				}
			}
		})
	}
}
