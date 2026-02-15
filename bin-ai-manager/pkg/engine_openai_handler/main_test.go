package engine_openai_handler

import (
	"testing"
)

func TestNewEngineOpenaiHandler(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
	}{
		{
			name:   "creates_handler_with_valid_api_key",
			apiKey: "sk-test-api-key-12345",
		},
		{
			name:   "creates_handler_with_empty_api_key",
			apiKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewEngineOpenaiHandler(tt.apiKey)
			if handler == nil {
				t.Error("Expected non-nil handler, got nil")
			}
		})
	}
}

func TestDefaultConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "default_model",
			constant: defaultModel,
			expected: "gpt-4-turbo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestDefaultSystemPromptExists(t *testing.T) {
	if defaultSystemPrompt == "" {
		t.Error("defaultSystemPrompt should not be empty")
	}
	if len(defaultSystemPrompt) < 50 {
		t.Error("defaultSystemPrompt seems too short to be valid")
	}
}
