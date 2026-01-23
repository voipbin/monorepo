package transfer

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestTransfer(t *testing.T) {
	tests := []struct {
		name string

		transferType     Type
		transfererCallID uuid.UUID
		transfereeCallID uuid.UUID
		groupcallID      uuid.UUID
		confbridgeID     uuid.UUID
	}{
		{
			name: "creates_attended_transfer",

			transferType:     TypeAttended,
			transfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			transfereeCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
			groupcallID:      uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			confbridgeID:     uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
		},
		{
			name: "creates_blind_transfer",

			transferType:     TypeBlind,
			transfererCallID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
			transfereeCallID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440001"),
			groupcallID:      uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440002"),
			confbridgeID:     uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440003"),
		},
		{
			name: "creates_transfer_with_nil_uuids",

			transferType:     TypeAttended,
			transfererCallID: uuid.Nil,
			transfereeCallID: uuid.Nil,
			groupcallID:      uuid.Nil,
			confbridgeID:     uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Transfer{
				Type:             tt.transferType,
				TransfererCallID: tt.transfererCallID,
				TransfereeCallID: tt.transfereeCallID,
				GroupcallID:      tt.groupcallID,
				ConfbridgeID:     tt.confbridgeID,
			}

			if tr.Type != tt.transferType {
				t.Errorf("Wrong Type. expect: %s, got: %s", tt.transferType, tr.Type)
			}
			if tr.TransfererCallID != tt.transfererCallID {
				t.Errorf("Wrong TransfererCallID. expect: %s, got: %s", tt.transfererCallID, tr.TransfererCallID)
			}
			if tr.TransfereeCallID != tt.transfereeCallID {
				t.Errorf("Wrong TransfereeCallID. expect: %s, got: %s", tt.transfereeCallID, tr.TransfereeCallID)
			}
			if tr.GroupcallID != tt.groupcallID {
				t.Errorf("Wrong GroupcallID. expect: %s, got: %s", tt.groupcallID, tr.GroupcallID)
			}
			if tr.ConfbridgeID != tt.confbridgeID {
				t.Errorf("Wrong ConfbridgeID. expect: %s, got: %s", tt.confbridgeID, tr.ConfbridgeID)
			}
		})
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{
			name:     "type_attended",
			constant: TypeAttended,
			expected: "attended",
		},
		{
			name:     "type_blind",
			constant: TypeBlind,
			expected: "blind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
