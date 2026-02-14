package outdialtarget

import (
	"encoding/json"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

func TestConvertWebhookMessage(t *testing.T) {
	tmCreate := time.Now()
	tmUpdate := time.Now().Add(1 * time.Hour)

	tests := []struct {
		name   string
		target *OutdialTarget
	}{
		{
			name: "converts target with all fields",
			target: &OutdialTarget{
				ID:        uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				OutdialID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				Name:      "Test Target",
				Detail:    "Test Detail",
				Data:      `{"key": "value"}`,
				Status:    StatusProgressing,
				Destination0: &commonaddress.Address{
					Type:   "phone",
					Target: "+12345678900",
				},
				Destination1: &commonaddress.Address{
					Type:   "phone",
					Target: "+12345678901",
				},
				Destination2: &commonaddress.Address{
					Type:   "email",
					Target: "test@example.com",
				},
				TryCount0: 1,
				TryCount1: 2,
				TryCount2: 0,
				TryCount3: 0,
				TryCount4: 0,
				TMCreate:  &tmCreate,
				TMUpdate:  &tmUpdate,
				TMDelete:  nil,
			},
		},
		{
			name: "converts target with minimal fields",
			target: &OutdialTarget{
				ID:           uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				OutdialID:    uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				Name:         "",
				Detail:       "",
				Data:         "",
				Status:       StatusIdle,
				Destination0: nil,
				Destination1: nil,
				Destination2: nil,
				Destination3: nil,
				Destination4: nil,
				TryCount0:    0,
				TryCount1:    0,
				TryCount2:    0,
				TryCount3:    0,
				TryCount4:    0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.target.ConvertWebhookMessage()

			if result == nil {
				t.Fatal("Expected webhook message but got nil")
			}

			if result.ID != tt.target.ID {
				t.Errorf("Expected ID %v, got %v", tt.target.ID, result.ID)
			}
			if result.OutdialID != tt.target.OutdialID {
				t.Errorf("Expected OutdialID %v, got %v", tt.target.OutdialID, result.OutdialID)
			}
			if result.Name != tt.target.Name {
				t.Errorf("Expected Name %s, got %s", tt.target.Name, result.Name)
			}
			if result.Detail != tt.target.Detail {
				t.Errorf("Expected Detail %s, got %s", tt.target.Detail, result.Detail)
			}
			if result.Status != tt.target.Status {
				t.Errorf("Expected Status %v, got %v", tt.target.Status, result.Status)
			}
			if result.TryCount0 != tt.target.TryCount0 {
				t.Errorf("Expected TryCount0 %d, got %d", tt.target.TryCount0, result.TryCount0)
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	tmCreate := time.Now()

	tests := []struct {
		name      string
		target    *OutdialTarget
		expectErr bool
	}{
		{
			name: "creates valid webhook event",
			target: &OutdialTarget{
				ID:        uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				OutdialID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				Name:      "Test Target",
				Detail:    "Test Detail",
				Data:      `{"key": "value"}`,
				Status:    StatusDone,
				Destination0: &commonaddress.Address{
					Type:   "phone",
					Target: "+12345678900",
				},
				TryCount0: 3,
				TMCreate:  &tmCreate,
			},
			expectErr: false,
		},
		{
			name: "creates webhook event with empty fields",
			target: &OutdialTarget{
				ID:        uuid.Nil,
				OutdialID: uuid.Nil,
				Name:      "",
				Detail:    "",
				Data:      "",
				Status:    StatusIdle,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.target.CreateWebhookEvent()

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
				if webhookMsg.ID != tt.target.ID {
					t.Errorf("Expected ID %v, got %v", tt.target.ID, webhookMsg.ID)
				}
				if webhookMsg.Name != tt.target.Name {
					t.Errorf("Expected Name %s, got %s", tt.target.Name, webhookMsg.Name)
				}
				if webhookMsg.Status != tt.target.Status {
					t.Errorf("Expected Status %v, got %v", tt.target.Status, webhookMsg.Status)
				}
			}
		})
	}
}

func TestWebhookMessage_JSONMarshaling(t *testing.T) {
	tmCreate := time.Now()
	msg := &WebhookMessage{
		ID:        uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
		OutdialID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
		Name:      "Test",
		Detail:    "Detail",
		Data:      "Data",
		Status:    StatusProgressing,
		Destination0: &commonaddress.Address{
			Type:   "phone",
			Target: "+12345678900",
		},
		TryCount0: 1,
		TryCount1: 0,
		TMCreate:  &tmCreate,
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
	if unmarshaled.Status != msg.Status {
		t.Errorf("Status mismatch after marshal/unmarshal")
	}
}

func TestWebhookMessage_AllDestinations(t *testing.T) {
	target := &OutdialTarget{
		ID:        uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
		OutdialID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
		Destination0: &commonaddress.Address{Type: "phone", Target: "+10000000000"},
		Destination1: &commonaddress.Address{Type: "phone", Target: "+10000000001"},
		Destination2: &commonaddress.Address{Type: "phone", Target: "+10000000002"},
		Destination3: &commonaddress.Address{Type: "phone", Target: "+10000000003"},
		Destination4: &commonaddress.Address{Type: "phone", Target: "+10000000004"},
		TryCount0:    1,
		TryCount1:    2,
		TryCount2:    3,
		TryCount3:    4,
		TryCount4:    5,
		Status:       StatusProgressing,
	}

	msg := target.ConvertWebhookMessage()

	if msg.Destination0 == nil || msg.Destination0.Target != "+10000000000" {
		t.Error("Destination0 not properly converted")
	}
	if msg.Destination1 == nil || msg.Destination1.Target != "+10000000001" {
		t.Error("Destination1 not properly converted")
	}
	if msg.Destination2 == nil || msg.Destination2.Target != "+10000000002" {
		t.Error("Destination2 not properly converted")
	}
	if msg.Destination3 == nil || msg.Destination3.Target != "+10000000003" {
		t.Error("Destination3 not properly converted")
	}
	if msg.Destination4 == nil || msg.Destination4.Target != "+10000000004" {
		t.Error("Destination4 not properly converted")
	}
	if msg.TryCount0 != 1 || msg.TryCount1 != 2 || msg.TryCount2 != 3 || msg.TryCount3 != 4 || msg.TryCount4 != 5 {
		t.Error("Try counts not properly converted")
	}
}
