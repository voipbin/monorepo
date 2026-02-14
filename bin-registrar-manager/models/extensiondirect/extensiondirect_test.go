package extensiondirect

import (
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestExtensionDirect(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	extensionID := uuid.Must(uuid.NewV4())
	now := time.Now()

	tests := []struct {
		name string
		ed   *ExtensionDirect
	}{
		{
			name: "complete_extension_direct",
			ed: &ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         id,
					CustomerID: customerID,
				},
				ExtensionID: extensionID,
				Hash:        "abc123def456",
				TMCreate:    &now,
				TMUpdate:    &now,
				TMDelete:    nil,
			},
		},
		{
			name: "minimal_extension_direct",
			ed: &ExtensionDirect{
				Identity: commonidentity.Identity{
					ID:         id,
					CustomerID: customerID,
				},
				ExtensionID: extensionID,
				Hash:        "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ed.ID != id {
				t.Errorf("Expected ID %v, got %v", id, tt.ed.ID)
			}
			if tt.ed.CustomerID != customerID {
				t.Errorf("Expected CustomerID %v, got %v", customerID, tt.ed.CustomerID)
			}
			if tt.ed.ExtensionID != extensionID {
				t.Errorf("Expected ExtensionID %v, got %v", extensionID, tt.ed.ExtensionID)
			}
		})
	}
}
