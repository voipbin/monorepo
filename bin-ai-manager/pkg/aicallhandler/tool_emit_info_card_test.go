package aicallhandler

import (
	"context"
	"reflect"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
)

func Test_toolHandleEmitInfoCard(t *testing.T) {

	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		expectRes *messageContent
	}{
		{
			name: "valid input with title, description, and fields",
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3a1f2c10-c002-11f0-9000-000000000001"),
				},
			},
			tool: &message.ToolCall{
				ID:   "3a1f2c10-c002-11f0-9000-000000000002",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameEmitInfoCard,
					Arguments: `{"title":"Contact Summary","description":"Key details for this contact.","fields":[{"label":"Name","value":"Jane Doe"},{"label":"Company","value":"Acme Corp"}]}`,
				},
			},
			expectRes: &messageContent{
				ToolCallID:   "3a1f2c10-c002-11f0-9000-000000000002",
				Result:       "success",
				Message:      "Displayed an info card titled 'Contact Summary'.",
				ResourceType: "card",
				ResourceID:   "",
				Blocks: []CardBlock{
					{
						Type:        "info",
						Title:       "Contact Summary",
						Description: "Key details for this contact.",
						Fields: []CardField{
							{Label: "Name", Value: "Jane Doe"},
							{Label: "Company", Value: "Acme Corp"},
						},
					},
				},
			},
		},
		{
			name: "empty fields array",
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3a1f2c10-c002-11f0-9000-000000000011"),
				},
			},
			tool: &message.ToolCall{
				ID:   "3a1f2c10-c002-11f0-9000-000000000012",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameEmitInfoCard,
					Arguments: `{"title":"Empty Card","fields":[]}`,
				},
			},
			expectRes: &messageContent{
				ToolCallID:   "3a1f2c10-c002-11f0-9000-000000000012",
				Result:       "success",
				Message:      "Displayed an info card titled 'Empty Card'.",
				ResourceType: "card",
				ResourceID:   "",
				Blocks: []CardBlock{
					{
						Type:   "info",
						Title:  "Empty Card",
						Fields: []CardField{},
					},
				},
			},
		},
		{
			name: "missing optional description",
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3a1f2c10-c002-11f0-9000-000000000021"),
				},
			},
			tool: &message.ToolCall{
				ID:   "3a1f2c10-c002-11f0-9000-000000000022",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameEmitInfoCard,
					Arguments: `{"title":"No Description"}`,
				},
			},
			expectRes: &messageContent{
				ToolCallID:   "3a1f2c10-c002-11f0-9000-000000000022",
				Result:       "success",
				Message:      "Displayed an info card titled 'No Description'.",
				ResourceType: "card",
				ResourceID:   "",
				Blocks: []CardBlock{
					{
						Type:   "info",
						Title:  "No Description",
						Fields: []CardField{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			h := &aicallHandler{}
			ctx := context.Background()

			res := h.toolHandleEmitInfoCard(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %+v, got: %+v", tt.expectRes, res)
			}
		})
	}
}

// Test_toolHandleEmitInfoCard_Truncation verifies the handler-side size
// guards (design doc D1's correction: the JSON Schema's maxItems/maxLength
// are a model-facing hint only, NOT the real enforcement -- the handler's
// own truncation is the primary and only real enforcement).
func Test_toolHandleEmitInfoCard_Truncation(t *testing.T) {
	h := &aicallHandler{}
	ctx := context.Background()

	longTitle := strings.Repeat("T", cardTitleMaxLen+50)
	longDescription := strings.Repeat("D", cardDescriptionMaxLen+50)
	longLabel := strings.Repeat("L", cardFieldLabelMaxLen+50)
	longValue := strings.Repeat("V", cardFieldValueMaxLen+50)

	var fieldsJSON strings.Builder
	fieldsJSON.WriteString("[")
	for i := 0; i < cardFieldsMaxCount+5; i++ {
		if i > 0 {
			fieldsJSON.WriteString(",")
		}
		fieldsJSON.WriteString(`{"label":"` + longLabel + `","value":"` + longValue + `"}`)
	}
	fieldsJSON.WriteString("]")

	args := `{"title":"` + longTitle + `","description":"` + longDescription + `","fields":` + fieldsJSON.String() + `}`

	tool := &message.ToolCall{
		ID:   "3a1f2c10-c002-11f0-9000-000000000031",
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameEmitInfoCard,
			Arguments: args,
		},
	}

	res := h.toolHandleEmitInfoCard(ctx, &aicall.AIcall{}, tool)

	if res.Result != "success" {
		t.Fatalf("expected success result, got: %+v", res)
	}
	if len(res.Blocks) != 1 {
		t.Fatalf("expected exactly one CardBlock, got: %d", len(res.Blocks))
	}
	block := res.Blocks[0]

	if got := len([]rune(block.Title)); got != cardTitleMaxLen {
		t.Errorf("title length = %d, want %d", got, cardTitleMaxLen)
	}
	if !strings.HasSuffix(block.Title, "...") {
		t.Errorf("truncated title should end with '...': %q", block.Title)
	}

	if got := len([]rune(block.Description)); got != cardDescriptionMaxLen {
		t.Errorf("description length = %d, want %d", got, cardDescriptionMaxLen)
	}
	if !strings.HasSuffix(block.Description, "...") {
		t.Errorf("truncated description should end with '...': %q", block.Description)
	}

	if len(block.Fields) != cardFieldsMaxCount {
		t.Errorf("field count = %d, want %d (excess fields must be dropped, not just truncated)", len(block.Fields), cardFieldsMaxCount)
	}

	for i, f := range block.Fields {
		if got := len([]rune(f.Label)); got != cardFieldLabelMaxLen {
			t.Errorf("field[%d] label length = %d, want %d", i, got, cardFieldLabelMaxLen)
		}
		if got := len([]rune(f.Value)); got != cardFieldValueMaxLen {
			t.Errorf("field[%d] value length = %d, want %d", i, got, cardFieldValueMaxLen)
		}
	}

	// Message must be the title-only trace, built from the TRUNCATED title,
	// never the full untruncated title and never any field value.
	expectMessage := "Displayed an info card titled '" + block.Title + "'."
	if res.Message != expectMessage {
		t.Errorf("Message = %q, want %q", res.Message, expectMessage)
	}
	for _, f := range block.Fields {
		if strings.Contains(res.Message, f.Value) {
			t.Errorf("Message must never contain a field value, got: %q", res.Message)
		}
	}
}

// Test_toolHandleEmitInfoCard_MissingTitle verifies an empty/missing title
// (required by D1) is rejected via fillFailed.
func Test_toolHandleEmitInfoCard_MissingTitle(t *testing.T) {
	h := &aicallHandler{}
	ctx := context.Background()

	tool := &message.ToolCall{
		ID:   "3a1f2c10-c002-11f0-9000-000000000041",
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameEmitInfoCard,
			Arguments: `{"description":"no title here"}`,
		},
	}

	res := h.toolHandleEmitInfoCard(ctx, &aicall.AIcall{}, tool)

	if res.Result != "failed" {
		t.Errorf("expected a failed result for missing title, got: %+v", res)
	}
	if len(res.Blocks) != 0 {
		t.Errorf("expected no Blocks on failure, got: %+v", res.Blocks)
	}
}

// Test_toolHandleEmitInfoCard_Message_NeverContainsFieldValues is a focused
// regression guard for design doc D2's resolution of the previously-
// undecided "what does messageContent.Message hold" question: it must be a
// short, title-only trace, identical across storage/LLM-feedback/frontend,
// and must never contain field values (so history replay cannot leak them,
// and the frontend never has a reason to render it as a caption).
func Test_toolHandleEmitInfoCard_Message_NeverContainsFieldValues(t *testing.T) {
	h := &aicallHandler{}
	ctx := context.Background()

	tool := &message.ToolCall{
		ID:   "3a1f2c10-c002-11f0-9000-000000000051",
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameEmitInfoCard,
			Arguments: `{"title":"Card X","description":"secret description text","fields":[{"label":"SSN","value":"123-45-6789"}]}`,
		},
	}

	res := h.toolHandleEmitInfoCard(ctx, &aicall.AIcall{}, tool)

	expectMessage := "Displayed an info card titled 'Card X'."
	if res.Message != expectMessage {
		t.Errorf("Message = %q, want %q", res.Message, expectMessage)
	}
	if strings.Contains(res.Message, "123-45-6789") {
		t.Errorf("Message must not contain field values, got: %q", res.Message)
	}
	if strings.Contains(res.Message, "secret description text") {
		t.Errorf("Message must not contain the description, got: %q", res.Message)
	}
}

// Test_emitInfoCardLLMResult verifies the LLM-facing return value for
// emit_info_card (design doc D2's first-turn bypass of
// unmarshalToolResponse): it must exclude Blocks entirely, carry the same
// title-only Message trace, and use resource_type "card" / resource_id ""
// -- the same five keys unmarshalToolResponse would have produced for any
// other tool.
func Test_emitInfoCardLLMResult(t *testing.T) {
	tmpContent := &messageContent{
		ToolCallID:   "tc-123",
		Result:       "success",
		Message:      "Displayed an info card titled 'Contact Summary'.",
		ResourceType: "card",
		ResourceID:   "",
		Blocks: []CardBlock{
			{
				Type:  "info",
				Title: "Contact Summary",
				Fields: []CardField{
					{Label: "Name", Value: "Jane Doe"},
				},
			},
		},
	}

	res := emitInfoCardLLMResult(tmpContent)

	expect := map[string]any{
		"tool_call_id":  "tc-123",
		"result":        "success",
		"message":       "Displayed an info card titled 'Contact Summary'.",
		"resource_type": "card",
		"resource_id":   "",
	}

	if !reflect.DeepEqual(res, expect) {
		t.Errorf("expected: %+v, got: %+v", expect, res)
	}

	if _, ok := res["blocks"]; ok {
		t.Errorf("LLM-facing result must not contain a 'blocks' key, got: %+v", res)
	}
	if len(res) != 5 {
		t.Errorf("LLM-facing result must have exactly 5 keys (matching unmarshalToolResponse's shape for every other tool), got %d: %+v", len(res), res)
	}
}
