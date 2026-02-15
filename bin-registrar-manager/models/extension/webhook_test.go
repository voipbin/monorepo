package extension

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
	now := time.Now()

	ext := &Extension{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Name:       "Test Extension",
		Detail:     "Test Detail",
		Extension:  "1001",
		DomainName: "test.ext.voipbin.net",
		Username:   "testuser",
		Password:   "testpass",
		DirectHash: "abcdef123456",
		TMCreate:   &now,
		TMUpdate:   &now,
		TMDelete:   nil,
	}

	wm := ext.ConvertWebhookMessage()

	if wm.ID != id {
		t.Errorf("Expected ID %v, got %v", id, wm.ID)
	}
	if wm.CustomerID != customerID {
		t.Errorf("Expected CustomerID %v, got %v", customerID, wm.CustomerID)
	}
	if wm.Name != "Test Extension" {
		t.Errorf("Expected Name 'Test Extension', got '%s'", wm.Name)
	}
	if wm.Detail != "Test Detail" {
		t.Errorf("Expected Detail 'Test Detail', got '%s'", wm.Detail)
	}
	if wm.Extension != "1001" {
		t.Errorf("Expected Extension '1001', got '%s'", wm.Extension)
	}
	if wm.DomainName != "test.ext.voipbin.net" {
		t.Errorf("Expected DomainName 'test.ext.voipbin.net', got '%s'", wm.DomainName)
	}
	if wm.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got '%s'", wm.Username)
	}
	if wm.Password != "testpass" {
		t.Errorf("Expected Password 'testpass', got '%s'", wm.Password)
	}
	if wm.DirectHash != "abcdef123456" {
		t.Errorf("Expected DirectHash 'abcdef123456', got '%s'", wm.DirectHash)
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	now := time.Now()

	tests := []struct {
		name    string
		ext     *Extension
		wantErr bool
	}{
		{
			name: "valid_extension",
			ext: &Extension{
				Identity: commonidentity.Identity{
					ID:         id,
					CustomerID: customerID,
				},
				Name:       "Test",
				Extension:  "1001",
				DomainName: "test.ext.voipbin.net",
				Username:   "user",
				Password:   "pass",
				TMCreate:   &now,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.ext.CreateWebhookEvent()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWebhookEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && data == nil {
				t.Error("CreateWebhookEvent() returned nil data")
			}
			if !tt.wantErr {
				// Verify it's valid JSON
				var wm WebhookMessage
				if err := json.Unmarshal(data, &wm); err != nil {
					t.Errorf("Failed to unmarshal webhook event: %v", err)
				}
			}
		})
	}
}
