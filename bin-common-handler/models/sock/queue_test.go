package sock

import (
	"testing"
)

func TestQueueTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant QueueType
		expected string
	}{
		{"queue_type_normal", QueueTypeNormal, "normal"},
		{"queue_type_volatile", QueueTypeVolatile, "volatile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
