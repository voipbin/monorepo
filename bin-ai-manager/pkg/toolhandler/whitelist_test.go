package toolhandler

import (
	"reflect"
	"sort"
	"testing"

	"monorepo/bin-ai-manager/models/tool"
)

func sortNames(in []tool.ToolName) []string {
	out := make([]string, 0, len(in))
	for _, n := range in {
		out = append(out, string(n))
	}
	sort.Strings(out)
	return out
}

func uniqueSorted(in []tool.ToolName) []string {
	seen := make(map[string]struct{}, len(in))
	for _, n := range in {
		seen[string(n)] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func Test_FilterToolsForConversation(t *testing.T) {
	tests := []struct {
		name string
		in   []tool.ToolName
		want []tool.ToolName // order-insensitive; deduplicated
	}{
		{
			name: "empty input returns empty",
			in:   nil,
			want: []tool.ToolName{},
		},
		{
			name: "strips voice-only tools",
			in:   []tool.ToolName{tool.ToolNameConnectCall, tool.ToolNameSendEmail, tool.ToolNameStopMedia},
			want: []tool.ToolName{tool.ToolNameSendEmail},
		},
		{
			name: "keeps text-safe tools",
			in:   []tool.ToolName{tool.ToolNameSendMessage, tool.ToolNameSetVariables, tool.ToolNameStopService},
			want: []tool.ToolName{tool.ToolNameSendMessage, tool.ToolNameSetVariables, tool.ToolNameStopService},
		},
		{
			name: "ToolNameAll expands to whitelist",
			in:   []tool.ToolName{tool.ToolNameAll},
			want: []tool.ToolName{
				tool.ToolNameSendEmail,
				tool.ToolNameSendMessage,
				tool.ToolNameSetVariables,
				tool.ToolNameGetVariables,
				tool.ToolNameStopService,
				tool.ToolNameGetAIcallMessages,
				tool.ToolNameSearchKnowledge,
				tool.ToolNameDescribeAction,
				tool.ToolNameCaseCreate,
			},
		},
		{
			name: "strips stop_flow",
			in:   []tool.ToolName{tool.ToolNameStopFlow, tool.ToolNameSendEmail},
			want: []tool.ToolName{tool.ToolNameSendEmail},
		},
		{
			name: "ToolNameAll mixed with explicit names — all wins (no duplicates)",
			in:   []tool.ToolName{tool.ToolNameAll, tool.ToolNameSendEmail},
			want: []tool.ToolName{
				tool.ToolNameSendEmail,
				tool.ToolNameSendMessage,
				tool.ToolNameSetVariables,
				tool.ToolNameGetVariables,
				tool.ToolNameStopService,
				tool.ToolNameGetAIcallMessages,
				tool.ToolNameSearchKnowledge,
				tool.ToolNameDescribeAction,
				tool.ToolNameCaseCreate,
			},
		},
		{
			name: "all voice-only stripped to empty",
			in:   []tool.ToolName{tool.ToolNameConnectCall, tool.ToolNameStopMedia, tool.ToolNameStopFlow},
			want: []tool.ToolName{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterToolsForConversation(tt.in)
			if !reflect.DeepEqual(uniqueSorted(got), uniqueSorted(tt.want)) {
				t.Errorf("got %v, want %v", sortNames(got), sortNames(tt.want))
			}
		})
	}
}
