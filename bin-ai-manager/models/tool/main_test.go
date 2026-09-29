package tool

import (
	"encoding/json"
	"strings"
	"testing"
)

// Test_Tool_Parameters_OmitEmpty pins B2b: a Tool with a nil Parameters map
// must marshal WITHOUT a "parameters" key at all. Design docs/plans/
// 2026-09-29-mcp-tool-exposure-pr-b2-design.md §1 (B2b): a null "parameters"
// key kills pipecat's init_llm the moment one MCP tool (built-in tools always
// set Parameters, but an MCP tool with no input schema decodes to nil) is
// advertised without a schema.
func Test_Tool_Parameters_OmitEmpty(t *testing.T) {
	tool := Tool{
		Name:        ToolNameConnectCall,
		Description: "test",
		Parameters:  nil,
		RunLLM:      true,
	}

	b, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(string(b), `"parameters"`) {
		t.Errorf("expected no \"parameters\" key when Parameters is nil, got: %s", b)
	}
}

// Test_Tool_Parameters_PresentWhenSet pins that a non-empty Parameters map
// is still marshaled as normal -- omitempty must not silently drop real data.
func Test_Tool_Parameters_PresentWhenSet(t *testing.T) {
	tool := Tool{
		Name:       ToolNameConnectCall,
		Parameters: map[string]any{"type": "object"},
	}

	b, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(b), `"parameters"`) {
		t.Errorf("expected \"parameters\" key to be present when Parameters is set, got: %s", b)
	}
}
