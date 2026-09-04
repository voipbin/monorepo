package aicallhandler

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Test_ToolHandle_unknownToolNameRecordsFailureResult pins the VOIP-1460
// second line of defense: ToolHandle stores the tool-call REQUEST message
// before dispatching, so the unknown-tool-name branch must record a paired
// failure RESULT message before returning the error. Without it the request
// message stays permanently unpaired, and once bin-pipecat-manager's filter
// stops dropping unpaired tool-call messages (the first line of defense in
// the same ticket) every later turn of that aicall would be rejected by an
// OpenAI-shaped provider.
func Test_ToolHandle_unknownToolNameRecordsFailureResult(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockMsg := messagehandler.NewMockMessageHandler(mc)
	h := &aicallHandler{db: mockDB, messageHandler: mockMsg}

	ctx := context.Background()
	aicallID := uuid.FromStringOrNil("9a01c1a0-c900-11f0-8a00-000000000001")
	customerID := uuid.FromStringOrNil("9a01c1a0-c900-11f0-8a00-000000000002")
	activeflowID := uuid.FromStringOrNil("9a01c1a0-c900-11f0-8a00-000000000003")
	toolCallID := "9a01c1a0-c900-11f0-8a00-000000000004"

	responseAIcall := &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         aicallID,
			CustomerID: customerID,
		},
		ActiveflowID: activeflowID,
	}

	function := message.FunctionCall{
		Name:      message.FunctionCallName("no_such_tool"),
		Arguments: "{}",
	}
	expectTool := message.ToolCall{
		ID:       toolCallID,
		Type:     message.ToolTypeFunction,
		Function: function,
	}

	mockDB.EXPECT().AIcallGet(ctx, aicallID).Return(responseAIcall, nil)

	// The tool-call REQUEST message (role=assistant, empty content, tool_calls
	// populated) is created before dispatch -- this is what makes the missing
	// result message an orphan.
	mockMsg.EXPECT().Create(
		ctx, uuid.Nil, customerID, aicallID, activeflowID,
		message.DirectionIncoming, message.RoleAssistant, "",
		[]message.ToolCall{expectTool}, "", gomock.Any(),
	).Return(&message.Message{}, nil)

	// The failure RESULT message must be recorded for the SAME tool_call_id,
	// with the standard Result: "failed" shape (never a novel value).
	var gotContent string
	mockMsg.EXPECT().Create(
		ctx, uuid.Nil, customerID, aicallID, activeflowID,
		message.DirectionOutgoing, message.RoleTool, gomock.Any(),
		nil, toolCallID, gomock.Any(),
	).DoAndReturn(func(
		_ context.Context, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID,
		_ message.Direction, _ message.Role, content string,
		_ []message.ToolCall, _ string, _ ...messagehandler.CreateOption,
	) (*message.Message, error) {
		gotContent = content
		return &message.Message{}, nil
	})

	res, err := h.ToolHandle(ctx, aicallID, toolCallID, message.ToolTypeFunction, function)

	if err == nil {
		t.Fatalf("expected an error for an unknown tool name, got nil")
	}
	if res != nil {
		t.Errorf("expected a nil result for an unknown tool name, got: %v", res)
	}
	if !strings.Contains(err.Error(), "unknown tool call: no_such_tool") {
		t.Errorf("unexpected error message. got: %s", err.Error())
	}

	if !strings.Contains(gotContent, `"result":"failed"`) {
		t.Errorf("failure result message must carry result=failed. got: %s", gotContent)
	}
	if !strings.Contains(gotContent, toolCallID) {
		t.Errorf("failure result message must carry the tool_call_id %s. got: %s", toolCallID, gotContent)
	}
	if !strings.Contains(gotContent, "unknown tool call: no_such_tool") {
		t.Errorf("failure result message must explain the failure. got: %s", gotContent)
	}
}
