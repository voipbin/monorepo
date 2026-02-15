package message

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func TestMessage(t *testing.T) {
	tests := []struct {
		name      string
		aicallID  uuid.UUID
		direction Direction
		role      Role
		content   string
	}{
		{
			name:      "creates_message_with_all_fields",
			aicallID:  uuid.Must(uuid.NewV4()),
			direction: DirectionIncoming,
			role:      RoleUser,
			content:   "Hello, how can I help you?",
		},
		{
			name:      "creates_message_with_empty_fields",
			aicallID:  uuid.Nil,
			direction: DirectionNone,
			role:      RoleNone,
			content:   "",
		},
		{
			name:      "creates_message_with_assistant_role",
			aicallID:  uuid.Must(uuid.NewV4()),
			direction: DirectionOutgoing,
			role:      RoleAssistant,
			content:   "I can help you with that.",
		},
		{
			name:      "creates_message_with_system_role",
			aicallID:  uuid.Must(uuid.NewV4()),
			direction: DirectionNone,
			role:      RoleSystem,
			content:   "System initialization complete",
		},
		{
			name:      "creates_message_with_tool_role",
			aicallID:  uuid.Must(uuid.NewV4()),
			direction: DirectionIncoming,
			role:      RoleTool,
			content:   "Tool execution result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			m := &Message{
				AIcallID:  tt.aicallID,
				Direction: tt.direction,
				Role:      tt.role,
				Content:   tt.content,
				TMCreate:  &now,
			}

			if m.AIcallID != tt.aicallID {
				t.Errorf("Wrong AIcallID. expect: %s, got: %s", tt.aicallID, m.AIcallID)
			}
			if m.Direction != tt.direction {
				t.Errorf("Wrong Direction. expect: %s, got: %s", tt.direction, m.Direction)
			}
			if m.Role != tt.role {
				t.Errorf("Wrong Role. expect: %s, got: %s", tt.role, m.Role)
			}
			if m.Content != tt.content {
				t.Errorf("Wrong Content. expect: %s, got: %s", tt.content, m.Content)
			}
		})
	}
}

func TestRoleConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Role
		expected string
	}{
		{
			name:     "role_none",
			constant: RoleNone,
			expected: "",
		},
		{
			name:     "role_system",
			constant: RoleSystem,
			expected: "system",
		},
		{
			name:     "role_user",
			constant: RoleUser,
			expected: "user",
		},
		{
			name:     "role_assistant",
			constant: RoleAssistant,
			expected: "assistant",
		},
		{
			name:     "role_function",
			constant: RoleFunction,
			expected: "function",
		},
		{
			name:     "role_tool",
			constant: RoleTool,
			expected: "tool",
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

func TestDirectionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Direction
		expected string
	}{
		{
			name:     "direction_incoming",
			constant: DirectionIncoming,
			expected: "incoming",
		},
		{
			name:     "direction_outgoing",
			constant: DirectionOutgoing,
			expected: "outgoing",
		},
		{
			name:     "direction_none",
			constant: DirectionNone,
			expected: "",
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
