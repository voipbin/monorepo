package queue

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestConvertWebhookMessage(t *testing.T) {
	queueID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	waitFlowID := uuid.Must(uuid.NewV4())
	tagID1 := uuid.Must(uuid.NewV4())
	tagID2 := uuid.Must(uuid.NewV4())
	queuecallID1 := uuid.Must(uuid.NewV4())
	queuecallID2 := uuid.Must(uuid.NewV4())
	now := time.Now()

	tests := []struct {
		name     string
		queue    *Queue
		validate func(t *testing.T, msg *WebhookMessage)
	}{
		{
			name: "full_queue_conversion",
			queue: &Queue{
				Identity: commonidentity.Identity{
					ID:         queueID,
					CustomerID: customerID,
				},
				Name:           "Test Queue",
				Detail:         "Test Detail",
				RoutingMethod:  RoutingMethodRandom,
				TagIDs:         []uuid.UUID{tagID1, tagID2},
				WaitFlowID:     waitFlowID,
				WaitTimeout:    60000,
				ServiceTimeout: 300000,
				WaitQueuecallIDs:    []uuid.UUID{queuecallID1},
				ServiceQueuecallIDs: []uuid.UUID{queuecallID2},
				TotalIncomingCount:  100,
				TotalServicedCount:  80,
				TotalAbandonedCount: 20,
				TMCreate:            &now,
				TMUpdate:            &now,
			},
			validate: func(t *testing.T, msg *WebhookMessage) {
				if msg.ID != queueID {
					t.Errorf("ID mismatch: got %v, want %v", msg.ID, queueID)
				}
				if msg.CustomerID != customerID {
					t.Errorf("CustomerID mismatch: got %v, want %v", msg.CustomerID, customerID)
				}
				if msg.Name != "Test Queue" {
					t.Errorf("Name mismatch: got %v, want %v", msg.Name, "Test Queue")
				}
				if msg.RoutingMethod != RoutingMethodRandom {
					t.Errorf("RoutingMethod mismatch: got %v, want %v", msg.RoutingMethod, RoutingMethodRandom)
				}
				if len(msg.TagIDs) != 2 {
					t.Errorf("TagIDs length mismatch: got %d, want 2", len(msg.TagIDs))
				}
				if msg.WaitTimeout != 60000 {
					t.Errorf("WaitTimeout mismatch: got %v, want %v", msg.WaitTimeout, 60000)
				}
				if msg.TotalIncomingCount != 100 {
					t.Errorf("TotalIncomingCount mismatch: got %v, want %v", msg.TotalIncomingCount, 100)
				}
			},
		},
		{
			name: "minimal_queue_conversion",
			queue: &Queue{
				Identity: commonidentity.Identity{
					ID:         queueID,
					CustomerID: customerID,
				},
				Name: "Minimal Queue",
			},
			validate: func(t *testing.T, msg *WebhookMessage) {
				if msg.ID != queueID {
					t.Errorf("ID mismatch: got %v, want %v", msg.ID, queueID)
				}
				if msg.Name != "Minimal Queue" {
					t.Errorf("Name mismatch: got %v, want %v", msg.Name, "Minimal Queue")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.queue.ConvertWebhookMessage()
			if msg == nil {
				t.Fatal("ConvertWebhookMessage returned nil")
			}
			tt.validate(t, msg)
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	queueID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	now := time.Now()

	tests := []struct {
		name      string
		queue     *Queue
		expectErr bool
	}{
		{
			name: "valid_webhook_event",
			queue: &Queue{
				Identity: commonidentity.Identity{
					ID:         queueID,
					CustomerID: customerID,
				},
				Name:           "Test Queue",
				Detail:         "Test Detail",
				RoutingMethod:  RoutingMethodRandom,
				WaitTimeout:    60000,
				ServiceTimeout: 300000,
				TMCreate:       &now,
			},
			expectErr: false,
		},
		{
			name: "minimal_webhook_event",
			queue: &Queue{
				Identity: commonidentity.Identity{
					ID:         queueID,
					CustomerID: customerID,
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.queue.CreateWebhookEvent()

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if data == nil {
				t.Error("CreateWebhookEvent returned nil data")
				return
			}

			// Verify JSON is valid
			var msg WebhookMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				t.Errorf("Failed to unmarshal webhook event: %v", err)
				return
			}

			if msg.ID != tt.queue.ID {
				t.Errorf("Unmarshaled ID mismatch: got %v, want %v", msg.ID, tt.queue.ID)
			}
		})
	}
}
