package identity

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestOwnerStruct(t *testing.T) {
	ownerID := uuid.Must(uuid.NewV4())

	o := Owner{
		OwnerType: OwnerTypeAgent,
		OwnerID:   ownerID,
	}

	if o.OwnerType != OwnerTypeAgent {
		t.Errorf("Owner.OwnerType = %v, expected %v", o.OwnerType, OwnerTypeAgent)
	}
	if o.OwnerID != ownerID {
		t.Errorf("Owner.OwnerID = %v, expected %v", o.OwnerID, ownerID)
	}
}

func TestOwnerTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant OwnerType
		expected string
	}{
		{"owner_type_none", OwnerTypeNone, ""},
		{"owner_type_agent", OwnerTypeAgent, "agent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestOwnerWithDifferentTypes(t *testing.T) {
	ownerID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name      string
		ownerType OwnerType
		ownerID   uuid.UUID
	}{
		{"no_owner", OwnerTypeNone, uuid.Nil},
		{"agent_owner", OwnerTypeAgent, ownerID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := Owner{
				OwnerType: tt.ownerType,
				OwnerID:   tt.ownerID,
			}
			if o.OwnerType != tt.ownerType {
				t.Errorf("Owner.OwnerType = %v, expected %v", o.OwnerType, tt.ownerType)
			}
			if o.OwnerID != tt.ownerID {
				t.Errorf("Owner.OwnerID = %v, expected %v", o.OwnerID, tt.ownerID)
			}
		})
	}
}
