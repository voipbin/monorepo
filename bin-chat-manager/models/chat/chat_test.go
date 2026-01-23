package chat

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestChatStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	roomOwnerID := uuid.Must(uuid.NewV4())
	participant1 := uuid.Must(uuid.NewV4())
	participant2 := uuid.Must(uuid.NewV4())

	c := Chat{
		Type:           TypeNormal,
		RoomOwnerID:    roomOwnerID,
		ParticipantIDs: []uuid.UUID{participant1, participant2},
		Name:           "Test Chat",
		Detail:         "Test chat details",
		TMCreate:       "2024-01-01 00:00:00.000000",
		TMUpdate:       "2024-01-01 00:00:00.000000",
		TMDelete:       "9999-01-01 00:00:00.000000",
	}
	c.ID = id

	if c.ID != id {
		t.Errorf("Chat.ID = %v, expected %v", c.ID, id)
	}
	if c.Type != TypeNormal {
		t.Errorf("Chat.Type = %v, expected %v", c.Type, TypeNormal)
	}
	if c.RoomOwnerID != roomOwnerID {
		t.Errorf("Chat.RoomOwnerID = %v, expected %v", c.RoomOwnerID, roomOwnerID)
	}
	if len(c.ParticipantIDs) != 2 {
		t.Errorf("Chat.ParticipantIDs length = %v, expected %v", len(c.ParticipantIDs), 2)
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_normal", TypeNormal, "normal"},
		{"type_group", TypeGroup, "group"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
