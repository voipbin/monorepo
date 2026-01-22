package address

import (
	"testing"
)

func TestAddressStruct(t *testing.T) {
	a := Address{
		Type:       TypeTel,
		Target:     "+15551234567",
		TargetName: "John Doe",
		Name:       "Primary Phone",
		Detail:     "Work phone number",
	}

	if a.Type != TypeTel {
		t.Errorf("Address.Type = %v, expected %v", a.Type, TypeTel)
	}
	if a.Target != "+15551234567" {
		t.Errorf("Address.Target = %v, expected %v", a.Target, "+15551234567")
	}
	if a.TargetName != "John Doe" {
		t.Errorf("Address.TargetName = %v, expected %v", a.TargetName, "John Doe")
	}
	if a.Name != "Primary Phone" {
		t.Errorf("Address.Name = %v, expected %v", a.Name, "Primary Phone")
	}
	if a.Detail != "Work phone number" {
		t.Errorf("Address.Detail = %v, expected %v", a.Detail, "Work phone number")
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_none", TypeNone, ""},
		{"type_agent", TypeAgent, "agent"},
		{"type_conference", TypeConference, "conference"},
		{"type_email", TypeEmail, "email"},
		{"type_extension", TypeExtension, "extension"},
		{"type_line", TypeLine, "line"},
		{"type_sip", TypeSIP, "sip"},
		{"type_tel", TypeTel, "tel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestAddressWithDifferentTypes(t *testing.T) {
	tests := []struct {
		name       string
		addrType   Type
		target     string
		targetName string
	}{
		{"agent_address", TypeAgent, "agent-uuid-123", "Agent Smith"},
		{"conference_address", TypeConference, "conf-uuid-456", "Team Meeting"},
		{"email_address", TypeEmail, "test@example.com", "Test User"},
		{"extension_address", TypeExtension, "1001", "Reception"},
		{"line_address", TypeLine, "line-id-789", "Line User"},
		{"sip_address", TypeSIP, "sip:user@domain.com", "SIP User"},
		{"tel_address", TypeTel, "+15551234567", "Phone User"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Address{
				Type:       tt.addrType,
				Target:     tt.target,
				TargetName: tt.targetName,
			}
			if a.Type != tt.addrType {
				t.Errorf("Address.Type = %v, expected %v", a.Type, tt.addrType)
			}
			if a.Target != tt.target {
				t.Errorf("Address.Target = %v, expected %v", a.Target, tt.target)
			}
			if a.TargetName != tt.targetName {
				t.Errorf("Address.TargetName = %v, expected %v", a.TargetName, tt.targetName)
			}
		})
	}
}
