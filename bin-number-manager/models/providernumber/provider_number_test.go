package providernumber

import (
	"testing"

	"monorepo/bin-number-manager/models/number"
)

func TestProviderNumberStruct(t *testing.T) {
	tests := []struct {
		name   string
		number ProviderNumber
	}{
		{
			name: "basic provider number",
			number: ProviderNumber{
				ID:               "test-id-123",
				Status:           number.StatusActive,
				T38Enabled:       true,
				EmergencyEnabled: false,
			},
		},
		{
			name: "provider number with emergency",
			number: ProviderNumber{
				ID:               "test-id-456",
				Status:           number.StatusActive,
				T38Enabled:       false,
				EmergencyEnabled: true,
			},
		},
		{
			name: "deleted provider number",
			number: ProviderNumber{
				ID:               "test-id-789",
				Status:           number.StatusDeleted,
				T38Enabled:       false,
				EmergencyEnabled: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all fields are set correctly
			if tt.number.ID == "" {
				t.Error("Expected ID to be set")
			}
			if tt.number.Status == "" {
				t.Error("Expected Status to be set")
			}
			// Just verify the struct can be created and accessed
			_ = tt.number.T38Enabled
			_ = tt.number.EmergencyEnabled
		})
	}
}

func TestProviderNumberFields(t *testing.T) {
	pn := ProviderNumber{
		ID:               "unique-id",
		Status:           number.StatusActive,
		T38Enabled:       true,
		EmergencyEnabled: true,
	}

	if pn.ID != "unique-id" {
		t.Errorf("Expected ID to be 'unique-id', got '%s'", pn.ID)
	}
	if pn.Status != number.StatusActive {
		t.Errorf("Expected Status to be StatusActive, got %v", pn.Status)
	}
	if !pn.T38Enabled {
		t.Error("Expected T38Enabled to be true")
	}
	if !pn.EmergencyEnabled {
		t.Error("Expected EmergencyEnabled to be true")
	}
}
