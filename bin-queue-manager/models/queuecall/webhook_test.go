package queuecall

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestConvertWebhookMessage(t *testing.T) {
	queuecallID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	agentID := uuid.Must(uuid.NewV4())
	now := time.Now()

	tests := []struct {
		name      string
		queuecall *Queuecall
		validate  func(t *testing.T, msg *WebhookMessage)
	}{
		{
			name: "full_queuecall_conversion",
			queuecall: &Queuecall{
				Identity: commonidentity.Identity{
					ID:         queuecallID,
					CustomerID: customerID,
				},
				ReferenceType:   ReferenceTypeCall,
				ReferenceID:     referenceID,
				Status:          StatusWaiting,
				ServiceAgentID:  agentID,
				DurationWaiting: 5000,
				DurationService: 10000,
				TMCreate:        &now,
				TMService:       &now,
				TMUpdate:        &now,
			},
			validate: func(t *testing.T, msg *WebhookMessage) {
				if msg.ID != queuecallID {
					t.Errorf("ID mismatch: got %v, want %v", msg.ID, queuecallID)
				}
				if msg.CustomerID != customerID {
					t.Errorf("CustomerID mismatch: got %v, want %v", msg.CustomerID, customerID)
				}
				if msg.ReferenceType != string(ReferenceTypeCall) {
					t.Errorf("ReferenceType mismatch: got %v, want %v", msg.ReferenceType, ReferenceTypeCall)
				}
				if msg.Status != string(StatusWaiting) {
					t.Errorf("Status mismatch: got %v, want %v", msg.Status, StatusWaiting)
				}
				if msg.ServiceAgentID != agentID {
					t.Errorf("ServiceAgentID mismatch: got %v, want %v", msg.ServiceAgentID, agentID)
				}
				if msg.DurationWaiting != 5000 {
					t.Errorf("DurationWaiting mismatch: got %v, want %v", msg.DurationWaiting, 5000)
				}
				if msg.DurationService != 10000 {
					t.Errorf("DurationService mismatch: got %v, want %v", msg.DurationService, 10000)
				}
			},
		},
		{
			name: "minimal_queuecall_conversion",
			queuecall: &Queuecall{
				Identity: commonidentity.Identity{
					ID:         queuecallID,
					CustomerID: customerID,
				},
				ReferenceType: ReferenceTypeCall,
				Status:        StatusWaiting,
			},
			validate: func(t *testing.T, msg *WebhookMessage) {
				if msg.ID != queuecallID {
					t.Errorf("ID mismatch: got %v, want %v", msg.ID, queuecallID)
				}
				if msg.ReferenceType != string(ReferenceTypeCall) {
					t.Errorf("ReferenceType mismatch: got %v, want %v", msg.ReferenceType, ReferenceTypeCall)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.queuecall.ConvertWebhookMessage()
			if msg == nil {
				t.Fatal("ConvertWebhookMessage returned nil")
			}
			tt.validate(t, msg)
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	queuecallID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	now := time.Now()

	tests := []struct {
		name      string
		queuecall *Queuecall
		expectErr bool
	}{
		{
			name: "valid_webhook_event",
			queuecall: &Queuecall{
				Identity: commonidentity.Identity{
					ID:         queuecallID,
					CustomerID: customerID,
				},
				ReferenceType:   ReferenceTypeCall,
				ReferenceID:     referenceID,
				Status:          StatusWaiting,
				DurationWaiting: 5000,
				TMCreate:        &now,
			},
			expectErr: false,
		},
		{
			name: "minimal_webhook_event",
			queuecall: &Queuecall{
				Identity: commonidentity.Identity{
					ID:         queuecallID,
					CustomerID: customerID,
				},
				ReferenceType: ReferenceTypeCall,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.queuecall.CreateWebhookEvent()

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

			if msg.ID != tt.queuecall.ID {
				t.Errorf("Unmarshaled ID mismatch: got %v, want %v", msg.ID, tt.queuecall.ID)
			}
		})
	}
}
