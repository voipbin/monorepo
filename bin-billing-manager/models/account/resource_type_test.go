package account

import (
	"testing"
)

func TestResourceType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		resource ResourceType
		expected string
	}{
		{"extension", ResourceTypeExtension, "extension"},
		{"agent", ResourceTypeAgent, "agent"},
		{"queue", ResourceTypeQueue, "queue"},
		{"flow", ResourceTypeFlow, "flow"},
		{"conference", ResourceTypeConference, "conference"},
		{"trunk", ResourceTypeTrunk, "trunk"},
		{"virtual_number", ResourceTypeVirtualNumber, "virtual_number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.resource) != tt.expected {
				t.Errorf("ResourceType = %s, expected %s", tt.resource, tt.expected)
			}
		})
	}
}
