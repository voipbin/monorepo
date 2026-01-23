package hook

import (
	"testing"
)

func TestHookStruct(t *testing.T) {
	h := Hook{
		Type:   TypeSubscribe,
		Topics: []string{"topic1", "topic2"},
	}

	if h.Type != TypeSubscribe {
		t.Errorf("Hook.Type = %v, expected %v", h.Type, TypeSubscribe)
	}
	if len(h.Topics) != 2 {
		t.Errorf("Hook.Topics length = %v, expected %v", len(h.Topics), 2)
	}
	if h.Topics[0] != "topic1" {
		t.Errorf("Hook.Topics[0] = %v, expected %v", h.Topics[0], "topic1")
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_subscribe", TypeSubscribe, "subscribe"},
		{"type_unsubscribe", TypeUnsubscribe, "unsubscribe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
