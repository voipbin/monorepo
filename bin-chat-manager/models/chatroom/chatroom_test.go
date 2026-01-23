package chatroom

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/chat"
)

func TestChatroomStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	chatID := uuid.Must(uuid.NewV4())
	roomOwnerID := uuid.Must(uuid.NewV4())
	participant1 := uuid.Must(uuid.NewV4())
	participant2 := uuid.Must(uuid.NewV4())

	cr := Chatroom{
		Type:           TypeNormal,
		ChatID:         chatID,
		RoomOwnerID:    roomOwnerID,
		ParticipantIDs: []uuid.UUID{participant1, participant2},
		Name:           "Test Chatroom",
		Detail:         "Test chatroom details",
		TMCreate:       "2024-01-01 00:00:00.000000",
		TMUpdate:       "2024-01-01 00:00:00.000000",
		TMDelete:       "9999-01-01 00:00:00.000000",
	}
	cr.ID = id

	if cr.ID != id {
		t.Errorf("Chatroom.ID = %v, expected %v", cr.ID, id)
	}
	if cr.Type != TypeNormal {
		t.Errorf("Chatroom.Type = %v, expected %v", cr.Type, TypeNormal)
	}
	if cr.ChatID != chatID {
		t.Errorf("Chatroom.ChatID = %v, expected %v", cr.ChatID, chatID)
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_unknown", TypeUnkonwn, "unknown"},
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

func TestConvertType(t *testing.T) {
	tests := []struct {
		name     string
		input    chat.Type
		expected Type
	}{
		{"convert_normal", chat.TypeNormal, TypeNormal},
		{"convert_group", chat.TypeGroup, TypeGroup},
		{"convert_unknown", chat.Type("invalid"), TypeUnkonwn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertType(tt.input)
			if result != tt.expected {
				t.Errorf("ConvertType(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}
