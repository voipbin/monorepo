package messagechat

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestMessagechatStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	chatID := uuid.Must(uuid.NewV4())

	m := Messagechat{
		ChatID:   chatID,
		Type:     TypeNormal,
		Text:     "Hello World",
		TMCreate: "2024-01-01 00:00:00.000000",
		TMUpdate: "2024-01-01 00:00:00.000000",
		TMDelete: "9999-01-01 00:00:00.000000",
	}
	m.ID = id

	if m.ID != id {
		t.Errorf("Messagechat.ID = %v, expected %v", m.ID, id)
	}
	if m.ChatID != chatID {
		t.Errorf("Messagechat.ChatID = %v, expected %v", m.ChatID, chatID)
	}
	if m.Type != TypeNormal {
		t.Errorf("Messagechat.Type = %v, expected %v", m.Type, TypeNormal)
	}
	if m.Text != "Hello World" {
		t.Errorf("Messagechat.Text = %v, expected %v", m.Text, "Hello World")
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
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
