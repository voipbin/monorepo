package aicallhandler

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/aicall"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cmcontact "monorepo/bin-contact-manager/models/contact"
)

func Test_resolveInsightListLimit_bounds(t *testing.T) {
	tests := []struct {
		name   string
		in     int
		expect uint64
	}{
		{name: "negative uses default", in: -5, expect: insightDefaultListLimit},
		{name: "zero uses default", in: 0, expect: insightDefaultListLimit},
		{name: "one is kept", in: 1, expect: 1},
		{name: "max is kept", in: insightMaxListLimit, expect: insightMaxListLimit},
		{name: "above max is clamped", in: insightMaxListLimit + 1, expect: insightMaxListLimit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveInsightListLimit(tt.in); got != tt.expect {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expect, got)
			}
		})
	}
}

func Test_contactDisplayName_fallbacks(t *testing.T) {
	tests := []struct {
		name    string
		contact *cmcontact.Contact
		expect  string
	}{
		{name: "display name wins", contact: &cmcontact.Contact{DisplayName: "  Display A ", FirstName: "F", LastName: "L"}, expect: "Display A"},
		{name: "first and last name", contact: &cmcontact.Contact{FirstName: "First", LastName: "Last"}, expect: "First Last"},
		{name: "first name only", contact: &cmcontact.Contact{FirstName: "First"}, expect: "First"},
		{name: "nothing known", contact: &cmcontact.Contact{DisplayName: "  "}, expect: "(unknown)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contactDisplayName(tt.contact); got != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, got)
			}
		})
	}
}

func Test_fillCaseDefaultTerminal_outcomes(t *testing.T) {
	caseID := uuid.FromStringOrNil("7e4f0d10-4d00-4c00-8000-000000000001")
	log := logrus.NewEntry(logrus.New())

	tests := []struct {
		name string

		outcome caseResolveOutcome
		err     error

		expectResult  string
		expectResType string
		expectMessage string
	}{
		{
			name:          "not a contact case",
			outcome:       caseResolveNotContactCase,
			expectResult:  "failed",
			expectMessage: "get_conversation_content without conversation_id is only supported for contact_case reference type",
		},
		{
			name:          "case not found",
			outcome:       caseResolveCaseNotFound,
			expectResult:  "success",
			expectResType: "conversation_content",
			expectMessage: msgResourceNotFound,
		},
		{
			name:          "case not from a conversation",
			outcome:       caseResolveNotFromConversation,
			expectResult:  "success",
			expectResType: "conversation_content",
			expectMessage: msgCaseNotFromConversation,
		},
		{
			name:          "rpc error is masked",
			outcome:       caseResolveRPCError,
			err:           fmt.Errorf("case service down"),
			expectResult:  "failed",
			expectMessage: "resource lookup failed",
		},
		{
			name:          "unknown outcome fails closed",
			outcome:       caseResolveOutcome(99),
			expectResult:  "failed",
			expectMessage: "resource lookup failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &aicall.AIcall{Identity: commonidentity.Identity{ID: caseID}, ReferenceID: caseID}
			res := newToolResult("tool-case-1")

			fillCaseDefaultTerminal(log, res, c, tt.outcome, tt.err)

			if res.Result != tt.expectResult || res.Message != tt.expectMessage || res.ResourceType != tt.expectResType {
				t.Errorf("Wrong match. got: %+v", res)
			}
			if tt.expectResult == "success" && res.ResourceID != caseID.String() {
				t.Errorf("Wrong resource id. expect: %s, got: %s", caseID, res.ResourceID)
			}
		})
	}
}

func Test_truncateListenLineText_bounds(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		maxChars int
		expect   string
	}{
		{name: "zero disables the cap", in: "abcdef", maxChars: 0, expect: "abcdef"},
		{name: "negative disables the cap", in: "abcdef", maxChars: -1, expect: "abcdef"},
		{name: "exact length is kept", in: "abcdef", maxChars: 6, expect: "abcdef"},
		{name: "longer is cut and marked", in: "abcdef", maxChars: 3, expect: "abc" + listenConversationTruncatedSuffix},
		{name: "cut counts runes", in: "가나다라", maxChars: 2, expect: "가나" + listenConversationTruncatedSuffix},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateListenLineText(tt.in, tt.maxChars); got != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, got)
			}
		})
	}
}

func Test_truncateCardText_bounds(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		maxLen int
		expect string
	}{
		{name: "short text is kept", in: "abc", maxLen: 5, expect: "abc"},
		{name: "tiny cap has no ellipsis", in: "abcdef", maxLen: 3, expect: "abc"},
		{name: "cap with room gets ellipsis", in: "abcdefgh", maxLen: 6, expect: "abc..."},
		{name: "multibyte cut is by rune", in: "가나다라마바", maxLen: 5, expect: "가나..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateCardText(tt.in, tt.maxLen); got != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expect, got)
			}
		})
	}
}

func Test_mcpServerIDShort_value(t *testing.T) {
	id := uuid.FromStringOrNil("7e4f0d10-4d00-4c00-8000-000000000002")
	if got := mcpServerIDShort(id); got != "7e4f0d10" {
		t.Errorf("Wrong match. expect: 7e4f0d10, got: %s", got)
	}
}

func Test_decodeMcpToolRef_shapes(t *testing.T) {
	serverID := uuid.FromStringOrNil("7e4f0d10-4d00-4c00-8000-000000000003")
	want := aicall.McpToolRef{ServerID: serverID, ToolName: "tool_a"}

	tests := []struct {
		name     string
		in       any
		expectOK bool
		expect   aicall.McpToolRef
	}{
		{name: "typed value", in: want, expectOK: true, expect: want},
		{name: "decoded map", in: map[string]any{"server_id": serverID.String(), "tool_name": "tool_a"}, expectOK: true, expect: want},
		{name: "wrong type", in: "text"},
		{name: "missing server id", in: map[string]any{"tool_name": "tool_a"}},
		{name: "malformed server id", in: map[string]any{"server_id": "not-a-uuid", "tool_name": "tool_a"}},
		{name: "missing tool name", in: map[string]any{"server_id": serverID.String()}},
		{name: "non-string tool name", in: map[string]any{"server_id": serverID.String(), "tool_name": 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := decodeMcpToolRef(tt.in)
			if ok != tt.expectOK {
				t.Fatalf("Wrong ok. expect: %v, got: %v", tt.expectOK, ok)
			}
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match. expect: %+v, got: %+v", tt.expect, got)
			}
		})
	}
}

func Test_skipEmptyJSONObject_values(t *testing.T) {
	if got := skipEmptyJSONObject("{}"); got != "" {
		t.Errorf("Wrong match. expect empty, got: %q", got)
	}
	if got := skipEmptyJSONObject(`{"a":1}`); !strings.Contains(got, `"a"`) {
		t.Errorf("Wrong match. non-empty object must be kept, got: %q", got)
	}
}
