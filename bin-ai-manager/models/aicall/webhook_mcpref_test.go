package aicall

import (
	"testing"

	"github.com/gofrs/uuid"
)

func Test_decodeMcpToolRefForStatus(t *testing.T) {
	serverID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name string
		in   any

		expectRef McpToolRef
		expectOK  bool
	}{
		{
			name:      "typed ref passes through",
			in:        McpToolRef{ServerID: serverID, ToolName: "search"},
			expectRef: McpToolRef{ServerID: serverID, ToolName: "search"},
			expectOK:  true,
		},
		{
			name:      "round-tripped map",
			in:        map[string]any{"server_id": "11111111-1111-1111-1111-111111111111", "tool_name": "lookup"},
			expectRef: McpToolRef{ServerID: serverID, ToolName: "lookup"},
			expectOK:  true,
		},
		{
			name:      "empty tool name is still accepted",
			in:        map[string]any{"server_id": "11111111-1111-1111-1111-111111111111", "tool_name": ""},
			expectRef: McpToolRef{ServerID: serverID, ToolName: ""},
			expectOK:  true,
		},
		{name: "not a map", in: "text", expectOK: false},
		{name: "nil", in: nil, expectOK: false},
		{name: "missing server_id", in: map[string]any{"tool_name": "x"}, expectOK: false},
		{name: "server_id not a string", in: map[string]any{"server_id": 5, "tool_name": "x"}, expectOK: false},
		{name: "server_id not a uuid", in: map[string]any{"server_id": "nope", "tool_name": "x"}, expectOK: false},
		{name: "missing tool_name", in: map[string]any{"server_id": "11111111-1111-1111-1111-111111111111"}, expectOK: false},
		{name: "tool_name not a string", in: map[string]any{"server_id": "11111111-1111-1111-1111-111111111111", "tool_name": 3}, expectOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, ok := decodeMcpToolRefForStatus(tt.in)
			if ok != tt.expectOK {
				t.Fatalf("ok = %v, expect: %v", ok, tt.expectOK)
			}
			if ref != tt.expectRef {
				t.Errorf("ref = %v, expect: %v", ref, tt.expectRef)
			}
		})
	}
}

func Test_mcpToolStatusFromRaw_roundTrippedShape(t *testing.T) {
	tests := []struct {
		name string
		in   any

		expect   McpToolStatus
		expectOK bool
	}{
		{
			name: "counts servers deduped and skips malformed entries",
			in: map[string]any{
				"a": map[string]any{"server_id": "11111111-1111-1111-1111-111111111111", "tool_name": "t1"},
				"b": map[string]any{"server_id": "11111111-1111-1111-1111-111111111111", "tool_name": "t2"},
				"c": map[string]any{"server_id": "22222222-2222-2222-2222-222222222222", "tool_name": "t3"},
				"d": map[string]any{"server_id": "bad", "tool_name": "t4"},
				"e": "junk",
			},
			expect:   McpToolStatus{Servers: 2, Tools: 3},
			expectOK: true,
		},
		{
			name:     "all entries malformed",
			in:       map[string]any{"a": "junk", "b": 1},
			expectOK: false,
		},
		{name: "empty map", in: map[string]any{}, expectOK: false},
		{name: "unsupported type", in: []string{"x"}, expectOK: false},
		{name: "nil", in: nil, expectOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := mcpToolStatusFromRaw(tt.in)
			if ok != tt.expectOK {
				t.Fatalf("ok = %v, expect: %v", ok, tt.expectOK)
			}
			if got != tt.expect {
				t.Errorf("status = %v, expect: %v", got, tt.expect)
			}
		})
	}
}
