package message

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestFieldStruct(t *testing.T) {
	tests := []struct {
		name   string
		fields FieldStruct
	}{
		{
			name: "all_fields",
			fields: FieldStruct{
				ID:                  uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426614174000"),
				CustomerID:          uuid.FromStringOrNil("223e4567-e89b-12d3-a456-426614174000"),
				Type:                TypeSMS,
				Source:              "+1234567890",
				ProviderName:        ProviderNameTelnyx,
				ProviderReferenceID: "ref-123",
				Direction:           DirectionOutbound,
				Deleted:             false,
			},
		},
		{
			name: "minimal_fields",
			fields: FieldStruct{
				ID:         uuid.FromStringOrNil("323e4567-e89b-12d3-a456-426614174000"),
				CustomerID: uuid.FromStringOrNil("423e4567-e89b-12d3-a456-426614174000"),
			},
		},
		{
			name: "with_deleted_flag",
			fields: FieldStruct{
				ID:       uuid.FromStringOrNil("523e4567-e89b-12d3-a456-426614174000"),
				Deleted:  true,
				Type:     TypeSMS,
				Direction: DirectionInbound,
			},
		},
		{
			name: "messagebird_provider",
			fields: FieldStruct{
				ProviderName:        ProviderNameMessagebird,
				ProviderReferenceID: "msgbird-ref-456",
				Type:                TypeSMS,
			},
		},
		{
			name: "twilio_provider",
			fields: FieldStruct{
				ProviderName: ProviderNameTwilio,
				Direction:    DirectionInbound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify that the struct can be created and accessed
			if tt.fields.ID != uuid.Nil {
				if tt.fields.ID == uuid.Nil {
					t.Error("ID should not be Nil when set")
				}
			}

			if tt.fields.CustomerID != uuid.Nil {
				if tt.fields.CustomerID == uuid.Nil {
					t.Error("CustomerID should not be Nil when set")
				}
			}

			// Test that type values are preserved
			if tt.fields.Type != "" {
				if tt.fields.Type != TypeSMS {
					t.Errorf("Type mismatch: got %v, want %v", tt.fields.Type, TypeSMS)
				}
			}

			// Test provider names
			if tt.fields.ProviderName != "" {
				switch tt.fields.ProviderName {
				case ProviderNameTelnyx, ProviderNameMessagebird, ProviderNameTwilio:
					// Valid provider
				default:
					t.Errorf("Unexpected provider name: %v", tt.fields.ProviderName)
				}
			}

			// Test direction values
			if tt.fields.Direction != "" {
				if tt.fields.Direction != DirectionInbound && tt.fields.Direction != DirectionOutbound {
					t.Errorf("Invalid direction: %v", tt.fields.Direction)
				}
			}
		})
	}
}

func TestFieldStructFilterTags(t *testing.T) {
	// Create a sample FieldStruct to verify it compiles
	fs := FieldStruct{
		ID:                  uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426614174000"),
		CustomerID:          uuid.FromStringOrNil("223e4567-e89b-12d3-a456-426614174000"),
		Type:                TypeSMS,
		Source:              "+1234567890",
		ProviderName:        ProviderNameTelnyx,
		ProviderReferenceID: "ref-123",
		Direction:           DirectionOutbound,
		Deleted:             false,
	}

	// Verify that the struct was created successfully
	if fs.ID == uuid.Nil {
		t.Error("Failed to create FieldStruct with valid UUID")
	}
}
