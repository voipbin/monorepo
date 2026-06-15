package aicallhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"reflect"
	"testing"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// errTest is a fixed error used to assert masked-vs-unmasked error propagation.
var errTest = stderrors.New("test error")

func Test_toolHandleCreateCall(t *testing.T) {
	customerID := uuid.FromStringOrNil("11110000-0000-4000-8000-000000000001")
	flowID := uuid.FromStringOrNil("11110000-0000-4000-8000-000000000002")
	ephemeralFlowID := uuid.FromStringOrNil("11110000-0000-4000-8000-00000000000e")

	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		mockSetup func(mockReq *requesthandler.MockRequestHandler)

		expectRes *messageContent
	}{
		{
			name: "success single tel destination from call session",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000a1"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-1",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameCreateCall,
					Arguments: `{
						"flow_id": "11110000-0000-4000-8000-000000000002",
						"destinations": [{"type": "tel", "target": "+111****1111"}]
					}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID},
				}, nil)
				mockReq.EXPECT().CallV1CallsCreate(
					gomock.Any(), customerID, flowID, uuid.Nil,
					&commonaddress.Address{},
					[]commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+111****1111"}},
					false, false, "", nil, gomock.Any(),
				).Return(
					[]*cmcall.Call{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000c1")}}},
					[]*cmgroupcall.Groupcall{},
					nil,
				)
			},
			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-1",
				ResourceType: "call",
				ResourceID:   "11110000-0000-4000-8000-0000000000c1",
				Message:      `{"call_ids":["11110000-0000-4000-8000-0000000000c1"],"groupcall_ids":[],"requested":1,"created":1}`,
			},
		},
		{
			name: "success from conversation session (no reference type restriction, no call-only deref)",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000a2"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeConversation,
			},
			tool: &message.ToolCall{
				ID:   "tool-conv",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameCreateCall,
					Arguments: `{
						"flow_id": "11110000-0000-4000-8000-000000000002",
						"destinations": [{"type": "tel", "target": "+111****1111"}]
					}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID},
				}, nil)
				mockReq.EXPECT().CallV1CallsCreate(
					gomock.Any(), customerID, flowID, uuid.Nil,
					&commonaddress.Address{},
					[]commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+111****1111"}},
					false, false, "", nil, gomock.Any(),
				).Return(
					[]*cmcall.Call{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000c2")}}},
					[]*cmgroupcall.Groupcall{},
					nil,
				)
			},
			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-conv",
				ResourceType: "call",
				ResourceID:   "11110000-0000-4000-8000-0000000000c2",
				Message:      `{"call_ids":["11110000-0000-4000-8000-0000000000c2"],"groupcall_ids":[],"requested":1,"created":1}`,
			},
		},
		{
			name: "success with explicit source and anonymous",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000a3"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-src",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameCreateCall,
					Arguments: `{
						"flow_id": "11110000-0000-4000-8000-000000000002",
						"source": {"type": "tel", "target": "+123****6789"},
						"destinations": [{"type": "tel", "target": "+111****1111"}],
						"anonymous": "yes"
					}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID},
				}, nil)
				mockReq.EXPECT().CallV1CallsCreate(
					gomock.Any(), customerID, flowID, uuid.Nil,
					&commonaddress.Address{Type: commonaddress.TypeTel, Target: "+123****6789"},
					[]commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+111****1111"}},
					false, false, "yes", nil, gomock.Any(),
				).Return(
					[]*cmcall.Call{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000c3")}}},
					[]*cmgroupcall.Groupcall{},
					nil,
				)
			},
			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-src",
				ResourceType: "call",
				ResourceID:   "11110000-0000-4000-8000-0000000000c3",
				Message:      `{"call_ids":["11110000-0000-4000-8000-0000000000c3"],"groupcall_ids":[],"requested":1,"created":1}`,
			},
		},
		{
			name: "success groupcall-only resource type is groupcall",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000a4"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-gc",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameCreateCall,
					Arguments: `{
						"flow_id": "11110000-0000-4000-8000-000000000002",
						"destinations": [{"type": "extension", "target": "sales"}]
					}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID},
				}, nil)
				mockReq.EXPECT().CallV1CallsCreate(
					gomock.Any(), customerID, flowID, uuid.Nil,
					&commonaddress.Address{},
					[]commonaddress.Address{{Type: commonaddress.TypeExtension, Target: "sales"}},
					false, false, "", nil, gomock.Any(),
				).Return(
					[]*cmcall.Call{},
					[]*cmgroupcall.Groupcall{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000d1")}}},
					nil,
				)
			},
			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-gc",
				ResourceType: "groupcall",
				ResourceID:   "11110000-0000-4000-8000-0000000000d1",
				Message:      `{"call_ids":[],"groupcall_ids":["11110000-0000-4000-8000-0000000000d1"],"requested":1,"created":1}`,
			},
		},
		{
			name: "success mixed call and groupcall, primary type is call",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000a5"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-mixed",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameCreateCall,
					Arguments: `{
						"flow_id": "11110000-0000-4000-8000-000000000002",
						"destinations": [{"type": "tel", "target": "+111****1111"}, {"type": "extension", "target": "sales"}]
					}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID},
				}, nil)
				mockReq.EXPECT().CallV1CallsCreate(
					gomock.Any(), customerID, flowID, uuid.Nil,
					&commonaddress.Address{},
					[]commonaddress.Address{
						{Type: commonaddress.TypeTel, Target: "+111****1111"},
						{Type: commonaddress.TypeExtension, Target: "sales"},
					},
					false, false, "", nil, gomock.Any(),
				).Return(
					[]*cmcall.Call{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000c5")}}},
					[]*cmgroupcall.Groupcall{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000d5")}}},
					nil,
				)
			},
			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-mixed",
				ResourceType: "call",
				ResourceID:   "11110000-0000-4000-8000-0000000000c5",
				Message:      `{"call_ids":["11110000-0000-4000-8000-0000000000c5"],"groupcall_ids":["11110000-0000-4000-8000-0000000000d5"],"requested":2,"created":2}`,
			},
		},
		{
			name: "partial success flags partial true",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000a6"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-partial",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameCreateCall,
					Arguments: `{
						"flow_id": "11110000-0000-4000-8000-000000000002",
						"destinations": [{"type": "tel", "target": "+111****1111"}, {"type": "tel", "target": "+222****2222"}]
					}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID},
				}, nil)
				mockReq.EXPECT().CallV1CallsCreate(
					gomock.Any(), customerID, flowID, uuid.Nil,
					&commonaddress.Address{},
					[]commonaddress.Address{
						{Type: commonaddress.TypeTel, Target: "+111****1111"},
						{Type: commonaddress.TypeTel, Target: "+222****2222"},
					},
					false, false, "", nil, gomock.Any(),
				).Return(
					[]*cmcall.Call{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000c6")}}},
					[]*cmgroupcall.Groupcall{},
					nil,
				)
			},
			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-partial",
				ResourceType: "call",
				ResourceID:   "11110000-0000-4000-8000-0000000000c6",
				Message:      `{"call_ids":["11110000-0000-4000-8000-0000000000c6"],"groupcall_ids":[],"requested":2,"created":1,"partial":true}`,
			},
		},
		{
			name: "total failure returns fillFailed",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000a7"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-fail",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameCreateCall,
					Arguments: `{
						"flow_id": "11110000-0000-4000-8000-000000000002",
						"destinations": [{"type": "tel", "target": "+111****1111"}]
					}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID},
				}, nil)
				mockReq.EXPECT().CallV1CallsCreate(
					gomock.Any(), customerID, flowID, uuid.Nil,
					&commonaddress.Address{},
					[]commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+111****1111"}},
					false, false, "", nil, gomock.Any(),
				).Return(nil, nil, errTest)
			},
			expectRes: &messageContent{
				Result:     "failed",
				ToolCallID: "tool-fail",
				Message:    "test error",
			},
		},
		{
			name: "neither flow_id nor actions",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000a8"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-noflow",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameCreateCall,
					Arguments: `{"destinations": [{"type": "tel", "target": "+111****1111"}]}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {},
			expectRes: &messageContent{
				Result:     "failed",
				ToolCallID: "tool-noflow",
				Message:    "either flow_id or actions is required",
			},
		},
		{
			name: "missing destinations",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000a9"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-nodest",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameCreateCall,
					Arguments: `{"flow_id": "11110000-0000-4000-8000-000000000002"}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {},
			expectRes: &messageContent{
				Result:     "failed",
				ToolCallID: "tool-nodest",
				Message:    "at least one destination is required",
			},
		},
		{
			name: "empty destination target",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000aa"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-emptytarget",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameCreateCall,
					Arguments: `{"flow_id": "11110000-0000-4000-8000-000000000002", "destinations": [{"type": "tel", "target": ""}]}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {},
			expectRes: &messageContent{
				Result:     "failed",
				ToolCallID: "tool-emptytarget",
				Message:    "destination target must not be empty",
			},
		},
		{
			name: "mixed valid and empty target rejects whole request before ownership RPC",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000af"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-mixedempty",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameCreateCall,
					Arguments: `{"flow_id": "11110000-0000-4000-8000-000000000002", "destinations": [{"type": "tel", "target": "+111****1111"}, {"type": "tel", "target": ""}]}`,
				},
			},
			// no FlowV1FlowGet / CallV1CallsCreate expected: validation short-circuits before any RPC.
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {},
			expectRes: &messageContent{
				Result:     "failed",
				ToolCallID: "tool-mixedempty",
				Message:    "destination target must not be empty",
			},
		},
		{
			name: "cross-customer flow is masked",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000ab"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-crosscust",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameCreateCall,
					Arguments: `{"flow_id": "11110000-0000-4000-8000-000000000002", "destinations": [{"type": "tel", "target": "+111****1111"}]}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: flowID, CustomerID: uuid.FromStringOrNil("99990000-0000-4000-8000-000000000099")},
				}, nil)
			},
			expectRes: &messageContent{
				Result:     "failed",
				ToolCallID: "tool-crosscust",
				Message:    "could not resolve flow",
			},
		},
		{
			name: "flow get error is masked identically to cross-customer",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000ac"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-flowerr",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameCreateCall,
					Arguments: `{"flow_id": "11110000-0000-4000-8000-000000000002", "destinations": [{"type": "tel", "target": "+111****1111"}]}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(nil, errTest)
			},
			expectRes: &messageContent{
				Result:     "failed",
				ToolCallID: "tool-flowerr",
				Message:    "could not resolve flow",
			},
		},
		{
			name: "success with inline actions creates ephemeral flow (ids preserved, branch/goto intact)",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000b1"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-actions",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameCreateCall,
					Arguments: `{
						"actions": [
							{"id": "22220000-0000-4000-8000-000000000001", "type": "talk", "option": {"text": "Hi, the meeting moved to 3pm", "language": "en-US"}},
							{"type": "hangup"}
						],
						"destinations": [{"type": "tel", "target": "+111****1111"}]
					}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				// actions are passed through unchanged (LLM-supplied id preserved so option-level
				// branch/goto targets stay valid). persist=false ephemeral flow.
				mockReq.EXPECT().FlowV1FlowCreate(
					gomock.Any(), customerID, fmflow.TypeFlow, "tmp", "tmp flow for ai create_call action assembly",
					[]fmaction.Action{
						{ID: uuid.FromStringOrNil("22220000-0000-4000-8000-000000000001"), Type: fmaction.TypeTalk, Option: map[string]any{"text": "Hi, the meeting moved to 3pm", "language": "en-US"}},
						{Type: fmaction.TypeHangup},
					},
					uuid.Nil, false,
				).Return(&fmflow.Flow{
					Identity: commonidentity.Identity{ID: ephemeralFlowID, CustomerID: customerID},
				}, nil)
				mockReq.EXPECT().CallV1CallsCreate(
					gomock.Any(), customerID, ephemeralFlowID, uuid.Nil,
					&commonaddress.Address{},
					[]commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+111****1111"}},
					false, false, "", nil, gomock.Any(),
				).Return(
					[]*cmcall.Call{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000e1")}}},
					[]*cmgroupcall.Groupcall{},
					nil,
				)
			},
			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-actions",
				ResourceType: "call",
				ResourceID:   "11110000-0000-4000-8000-0000000000e1",
				Message:      `{"call_ids":["11110000-0000-4000-8000-0000000000e1"],"groupcall_ids":[],"requested":1,"created":1}`,
			},
		},
		{
			name: "both flow_id and actions rejected",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000b2"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-both",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameCreateCall,
					Arguments: `{
						"flow_id": "11110000-0000-4000-8000-000000000002",
						"actions": [{"type": "hangup"}],
						"destinations": [{"type": "tel", "target": "+111****1111"}]
					}`,
				},
			},
			// no RPC expected: XOR validation short-circuits before any RPC.
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {},
			expectRes: &messageContent{
				Result:     "failed",
				ToolCallID: "tool-both",
				Message:    "provide either flow_id or actions, not both",
			},
		},
		{
			name: "empty actions array treated as not provided",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000b3"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-emptyactions",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameCreateCall,
					Arguments: `{"actions": [], "destinations": [{"type": "tel", "target": "+111****1111"}]}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {},
			expectRes: &messageContent{
				Result:     "failed",
				ToolCallID: "tool-emptyactions",
				Message:    "either flow_id or actions is required",
			},
		},
		{
			name: "ephemeral flow creation failure (e.g. invalid action type) is surfaced",
			aicall: &aicall.AIcall{
				Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("11110000-0000-4000-8000-0000000000b4"), CustomerID: customerID},
				ReferenceType: aicall.ReferenceTypeCall,
			},
			tool: &message.ToolCall{
				ID:   "tool-badaction",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameCreateCall,
					Arguments: `{
						"actions": [{"type": "not_a_real_action"}],
						"destinations": [{"type": "tel", "target": "+111****1111"}]
					}`,
				},
			},
			mockSetup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1FlowCreate(
					gomock.Any(), customerID, fmflow.TypeFlow, "tmp", "tmp flow for ai create_call action assembly",
					[]fmaction.Action{{Type: fmaction.Type("not_a_real_action")}},
					uuid.Nil, false,
				).Return(nil, errTest)
			},
			expectRes: &messageContent{
				Result:     "failed",
				ToolCallID: "tool-badaction",
				Message:    "test error",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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

			if tt.mockSetup != nil {
				tt.mockSetup(mockReq)
			}

			res := h.toolHandleCreateCall(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

// Test_toolHandleCreateCall_maskingByteIdentical asserts that the cross-customer and
// not-found (flow-get error) rejection paths return a byte-identical messageContent so the
// tool cannot be used as a flow-existence oracle.
func Test_toolHandleCreateCall_maskingByteIdentical(t *testing.T) {
	customerID := uuid.FromStringOrNil("22220000-0000-4000-8000-000000000001")
	flowID := uuid.FromStringOrNil("22220000-0000-4000-8000-000000000002")

	args := `{"flow_id": "22220000-0000-4000-8000-000000000002", "destinations": [{"type": "tel", "target": "+111****1111"}]}`
	tool := &message.ToolCall{
		ID:       "mask-tool",
		Type:     message.ToolTypeFunction,
		Function: message.FunctionCall{Name: message.FunctionCallNameCreateCall, Arguments: args},
	}
	a := &aicall.AIcall{
		Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("22220000-0000-4000-8000-0000000000a1"), CustomerID: customerID},
		ReferenceType: aicall.ReferenceTypeCall,
	}

	newHandler := func(mc *gomock.Controller, mockReq *requesthandler.MockRequestHandler) *aicallHandler {
		return &aicallHandler{
			utilHandler:    utilhandler.NewMockUtilHandler(mc),
			reqHandler:     mockReq,
			notifyHandler:  notifyhandler.NewMockNotifyHandler(mc),
			db:             dbhandler.NewMockDBHandler(mc),
			aiHandler:      aihandler.NewMockAIHandler(mc),
			messageHandler: messagehandler.NewMockMessageHandler(mc),
		}
	}

	// not-found path
	mc1 := gomock.NewController(t)
	defer mc1.Finish()
	mockReq1 := requesthandler.NewMockRequestHandler(mc1)
	mockReq1.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(nil, errTest)
	resNotFound := newHandler(mc1, mockReq1).toolHandleCreateCall(context.Background(), a, tool)

	// cross-customer path
	mc2 := gomock.NewController(t)
	defer mc2.Finish()
	mockReq2 := requesthandler.NewMockRequestHandler(mc2)
	mockReq2.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(&fmflow.Flow{
		Identity: commonidentity.Identity{ID: flowID, CustomerID: uuid.FromStringOrNil("99990000-0000-4000-8000-000000000099")},
	}, nil)
	resCrossCustomer := newHandler(mc2, mockReq2).toolHandleCreateCall(context.Background(), a, tool)

	if resNotFound.Message != resCrossCustomer.Message {
		t.Errorf("masking not byte-identical: not-found=%q cross-customer=%q", resNotFound.Message, resCrossCustomer.Message)
	}
	if resNotFound.Result != resCrossCustomer.Result {
		t.Errorf("masking result differs: not-found=%q cross-customer=%q", resNotFound.Result, resCrossCustomer.Result)
	}

	// marshalled JSON must also be byte-identical (full oracle protection)
	b1, _ := json.Marshal(resNotFound)
	b2, _ := json.Marshal(resCrossCustomer)
	if string(b1) != string(b2) {
		t.Errorf("masking JSON not byte-identical:\n not-found=%s\n cross-customer=%s", b1, b2)
	}
}

// Test_toolHandleCreateCall_doesNotTerminateAIcall asserts the P1 safety contract: a successful
// create_call must NOT terminate the current AI session (unlike connect_call). The gomock
// controller is strict, so any unexpected AIV1AIcallTerminate call would fail the test; this test
// makes the invariant explicit by exercising a full success path and finishing the controller.
func Test_toolHandleCreateCall_doesNotTerminateAIcall(t *testing.T) {
	customerID := uuid.FromStringOrNil("33330000-0000-4000-8000-000000000001")
	flowID := uuid.FromStringOrNil("33330000-0000-4000-8000-000000000002")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{
		utilHandler:    utilhandler.NewMockUtilHandler(mc),
		reqHandler:     mockReq,
		notifyHandler:  notifyhandler.NewMockNotifyHandler(mc),
		db:             dbhandler.NewMockDBHandler(mc),
		aiHandler:      aihandler.NewMockAIHandler(mc),
		messageHandler: messagehandler.NewMockMessageHandler(mc),
	}

	a := &aicall.AIcall{
		Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("33330000-0000-4000-8000-0000000000a1"), CustomerID: customerID},
		ReferenceType: aicall.ReferenceTypeCall,
	}
	tool := &message.ToolCall{
		ID:   "noterm-tool",
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameCreateCall,
			Arguments: `{"flow_id": "33330000-0000-4000-8000-000000000002", "destinations": [{"type": "tel", "target": "+111****1111"}]}`,
		},
	}

	mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(&fmflow.Flow{
		Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID},
	}, nil)
	mockReq.EXPECT().CallV1CallsCreate(
		gomock.Any(), customerID, flowID, uuid.Nil,
		&commonaddress.Address{},
		[]commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+111****1111"}},
		false, false, "", nil, gomock.Any(),
	).Return(
		[]*cmcall.Call{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("33330000-0000-4000-8000-0000000000c1")}}},
		[]*cmgroupcall.Groupcall{},
		nil,
	)
	// Deliberately NO mockReq.EXPECT().AIV1AIcallTerminate(...): a strict gomock controller fails
	// the test if the handler issues an unexpected terminate, enforcing the non-termination contract.

	res := h.toolHandleCreateCall(context.Background(), a, tool)
	if res.Result != "success" {
		t.Errorf("expected success, got %q (message: %q)", res.Result, res.Message)
	}
}
