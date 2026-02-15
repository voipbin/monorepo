package extensiondirect

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
	extensionID := uuid.Must(uuid.NewV4())
	now := time.Now()

	ed := &ExtensionDirect{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		ExtensionID: extensionID,
		Hash:        "abc123def456",
		TMCreate:    &now,
		TMUpdate:    &now,
		TMDelete:    nil,
	}

	wm := ed.ConvertWebhookMessage()

	if wm.ID != id {
		t.Errorf("Expected ID %v, got %v", id, wm.ID)
	}
	if wm.CustomerID != customerID {
		t.Errorf("Expected CustomerID %v, got %v", customerID, wm.CustomerID)
	}
	if wm.ExtensionID != extensionID {
		t.Errorf("Expected ExtensionID %v, got %v", extensionID, wm.ExtensionID)
	}
	if wm.Hash != "abc123def456" {
		t.Errorf("Expected Hash 'abc123def456', got '%s'", wm.Hash)
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	extensionID := uuid.Must(uuid.NewV4())
	now := time.Now()

	tests := []struct {
		name    string
		ed      *ExtensionDirect
		wantErr bool
	}{
		{
			name: "valid_extension_direct",
			ed: &ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         id,
					CustomerID: customerID,
				},
				ExtensionID: extensionID,
				Hash:        "test123",
				TMCreate:    &now,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.ed.CreateWebhookEvent()
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
