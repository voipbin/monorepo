package messagechatroom

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/messagechat"
)

func TestMessagechatroomStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	chatroomID := uuid.Must(uuid.NewV4())
	messagechatID := uuid.Must(uuid.NewV4())

	m := Messagechatroom{
		ChatroomID:    chatroomID,
		MessagechatID: messagechatID,
		Type:          TypeNormal,
		Text:          "Hello World",
		TMCreate:      "2024-01-01 00:00:00.000000",
		TMUpdate:      "2024-01-01 00:00:00.000000",
		TMDelete:      "9999-01-01 00:00:00.000000",
	}
	m.ID = id

	if m.ID != id {
		t.Errorf("Messagechatroom.ID = %v, expected %v", m.ID, id)
	}
	if m.ChatroomID != chatroomID {
		t.Errorf("Messagechatroom.ChatroomID = %v, expected %v", m.ChatroomID, chatroomID)
	}
	if m.MessagechatID != messagechatID {
		t.Errorf("Messagechatroom.MessagechatID = %v, expected %v", m.MessagechatID, messagechatID)
	}
	if m.Type != TypeNormal {
		t.Errorf("Messagechatroom.Type = %v, expected %v", m.Type, TypeNormal)
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_unknown", TypeUnknown, ""},
		{"type_system", TypeSystem, "system"},
		{"type_normal", TypeNormal, "normal"},
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
		input    messagechat.Type
		expected Type
	}{
		{"convert_normal", messagechat.TypeNormal, TypeNormal},
		{"convert_system", messagechat.TypeSystem, TypeSystem},
		{"convert_unknown", messagechat.Type("invalid"), TypeUnknown},
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
