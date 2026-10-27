package mcpserver

import "testing"

func Test_AuthType_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		input  AuthType
		expect bool
	}{
		{name: "none (empty string)", input: AuthTypeNone, expect: true},
		{name: "bearer", input: AuthTypeBearer, expect: true},
		{name: "api_key", input: AuthTypeAPIKey, expect: true},
		{name: "oauth", input: AuthTypeOAuth, expect: true},
		{name: "unknown", input: AuthType("basic"), expect: false},
		{name: "case differs", input: AuthType("Bearer"), expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.IsValid(); got != tt.expect {
				t.Errorf("IsValid(%q) = %v, expect: %v", tt.input, got, tt.expect)
			}
		})
	}
}

func Test_Status_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		input  Status
		expect bool
	}{
		{name: "active", input: StatusActive, expect: true},
		{name: "disabled", input: StatusDisabled, expect: true},
		{name: "empty", input: Status(""), expect: false},
		{name: "unknown", input: Status("deleted"), expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.IsValid(); got != tt.expect {
				t.Errorf("IsValid(%q) = %v, expect: %v", tt.input, got, tt.expect)
			}
		})
	}
}
