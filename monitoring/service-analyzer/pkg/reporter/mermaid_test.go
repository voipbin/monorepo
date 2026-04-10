package reporter

import (
	"strings"
	"testing"

	"monorepo/monitoring/service-analyzer/pkg/analyzer"
)

func TestGenerateMermaid(t *testing.T) {
	g := &analyzer.Graph{
		Services: []analyzer.Service{
			{Name: "call-manager"},
			{Name: "flow-manager"},
			{Name: "api-manager"},
		},
		Dependencies: []analyzer.Dependency{
			{From: "api-manager", To: "call-manager", Type: analyzer.DepRPC, Methods: []string{"CallV1CallGet"}},
			{From: "flow-manager", To: "call-manager", Type: analyzer.DepEvent, Methods: []string{"call_hungup"}},
		},
	}

	output := GenerateMermaid(g)

	if !strings.Contains(output, "graph TD") {
		t.Error("output should start with 'graph TD'")
	}
	if !strings.Contains(output, "api_manager -->|RPC| call_manager") {
		t.Error("output should contain RPC edge")
	}
	if !strings.Contains(output, "flow_manager -.->|event| call_manager") {
		t.Error("output should contain event edge")
	}
	if !strings.Contains(output, "Core Layer") {
		t.Error("output should contain Core Layer subgraph")
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"call-manager", "call_manager"},
		{"api-manager", "api_manager"},
		{"no-dashes-here", "no_dashes_here"},
	}
	for _, tt := range tests {
		got := sanitizeID(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
