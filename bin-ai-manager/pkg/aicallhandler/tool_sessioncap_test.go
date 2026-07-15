package aicallhandler

import (
	"context"
	"testing"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmvariable "monorepo/bin-flow-manager/models/variable"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	gomock "go.uber.org/mock/gomock"
)

// Test_ToolHandle_SessionCapExceeded verifies the session (AIcall) tool-call cap
// (VOIP-1259, design doc §7): under-cap calls dispatch the underlying tool as normal,
// while over-cap calls are failed via fillFailed WITHOUT invoking the underlying tool
// function (asserted here by NOT registering any mock expectation for the RPC the
// underlying function would otherwise call — an unexpected call fails the gomock
// controller).
func Test_ToolHandle_SessionCapExceeded(t *testing.T) {
	config.SetAIcallSessionToolCallLimitForTest(100)
	defer config.SetAIcallSessionToolCallLimitForTest(0)

	customerID := uuid.FromStringOrNil("aaaa0000-0000-4000-8000-000000000001")
	aicallID := uuid.FromStringOrNil("aaaa0000-0000-4000-8000-000000000002")
	activeflowID := uuid.FromStringOrNil("aaaa0000-0000-4000-8000-000000000003")

	tests := []struct {
		name string

		initialMetadata map[string]any

		// expectDispatch: whether the underlying tool function (toolHandleGetVariables)
		// is expected to run. When false, no mockReq expectation is registered for
		// FlowV1VariableGet, so a call would fail the test via unexpected-call.
		expectDispatch bool

		expectResult  string
		expectMessage string
	}{
		{
			name:            "under cap dispatches underlying tool",
			initialMetadata: map[string]any{},
			expectDispatch:  true,
			expectResult:    "success",
			expectMessage:   `{"key":"value"}`,
		},
		{
			name:            "boundary: 100th call (count 99->100) still dispatches",
			initialMetadata: map[string]any{aicall.MetaKeyToolCallCount: 99},
			expectDispatch:  true,
			expectResult:    "success",
			expectMessage:   `{"key":"value"}`,
		},
		{
			name:            "cap exceeded (count already at 100) blocks dispatch",
			initialMetadata: map[string]any{aicall.MetaKeyToolCallCount: 100},
			expectDispatch:  false,
			expectResult:    "failed",
			expectMessage:   errToolCallSessionCapExceeded.Error(),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// no t.Parallel(): asserts a package-level Prometheus counter delta.
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			ac := &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         aicallID,
					CustomerID: customerID,
				},
				ActiveflowID: activeflowID,
				Metadata:     tt.initialMetadata,
			}

			tool := &message.ToolCall{
				ID:   "tool-cap-1",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetVariables,
					Arguments: `{}`,
				},
			}

			// h.Get(ctx, id) inside ToolHandle
			mockDB.EXPECT().AIcallGet(gomock.Any(), aicallID).Return(ac, nil)

			// tool-call request message create
			mockMessage.EXPECT().Create(gomock.Any(), uuid.Nil, customerID, aicallID, activeflowID,
				message.DirectionIncoming, message.RoleAssistant, "", []message.ToolCall{*tool}, "", gomock.Any(),
			).Return(&message.Message{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("aaaa0000-0000-4000-8000-000000000004")}}, nil)

			if tt.expectDispatch {
				// validateSessionToolCallRate -> UpdateMetadata read-merge-write path
				mockDB.EXPECT().AIcallGet(gomock.Any(), aicallID).Return(ac, nil)
				mockDB.EXPECT().AIcallUpdate(gomock.Any(), aicallID, gomock.Any()).Return(nil)
				mockDB.EXPECT().AIcallGet(gomock.Any(), aicallID).Return(ac, nil)

				// underlying toolHandleGetVariables dispatch
				mockReq.EXPECT().FlowV1VariableGet(gomock.Any(), activeflowID).Return(&fmvariable.Variable{Variables: map[string]string{"key": "value"}}, nil)
			}
			// else: no FlowV1VariableGet expectation registered — an unexpected call fails the test.

			before := testutil.ToFloat64(promAIcallToolCallSessionCapExceededTotal)

			// tool-call response message create
			mockMessage.EXPECT().Create(gomock.Any(), uuid.Nil, customerID, aicallID, activeflowID,
				message.DirectionOutgoing, message.RoleTool, gomock.Any(), nil, tool.ID, gomock.Any(),
			).DoAndReturn(func(_ context.Context, _, _, _, _ uuid.UUID, _ message.Direction, _ message.Role, content string, _ []message.ToolCall, _ string, _ ...messagehandler.CreateOption) (*message.Message, error) {
				return &message.Message{Content: content}, nil
			})

			res, err := h.ToolHandle(ctx, aicallID, tool.ID, tool.Type, tool.Function)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res["result"] != tt.expectResult {
				t.Errorf("expected result %q, got %q (full: %v)", tt.expectResult, res["result"], res)
			}
			if tt.expectResult == "failed" && res["message"] != tt.expectMessage {
				t.Errorf("expected message %q, got %q", tt.expectMessage, res["message"])
			}

			after := testutil.ToFloat64(promAIcallToolCallSessionCapExceededTotal)
			expectDelta := 0.0
			if !tt.expectDispatch {
				expectDelta = 1.0
			}
			if after-before != expectDelta {
				t.Errorf("expected promAIcallToolCallSessionCapExceededTotal delta %v, got %v", expectDelta, after-before)
			}
		})
	}
}
