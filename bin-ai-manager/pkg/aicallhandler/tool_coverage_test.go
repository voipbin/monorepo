package aicallhandler

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	rmquery "monorepo/bin-rag-manager/models/query"
)

func Test_toolHandleSearchKnowledge(t *testing.T) {
	aicallID := uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-000000000001")
	assistanceID := uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-000000000002")
	ragID := uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-000000000003")

	newCall := func() *aicall.AIcall {
		return &aicall.AIcall{
			Identity:       commonidentity.Identity{ID: aicallID},
			AssistanceType: aicall.AssistanceTypeAI,
			AssistanceID:   assistanceID,
		}
	}
	newTool := func(args string) *message.ToolCall {
		return &message.ToolCall{
			ID:   "tool-search-1",
			Type: message.ToolTypeFunction,
			Function: message.FunctionCall{
				Name:      message.FunctionCallName("search_knowledge"),
				Arguments: args,
			},
		}
	}

	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		aiResult *ai.AI
		aiErr    error
		ragRes   *rmquery.Response
		ragErr   error

		expectAIGet  bool
		expectRagReq bool
		expectRes    *messageContent
	}{
		{
			name:      "invalid json arguments",
			aicall:    newCall(),
			tool:      newTool(`{"query": `),
			expectRes: &messageContent{Result: "failed", ToolCallID: "tool-search-1", Message: "unexpected end of JSON input"},
		},
		{
			name:        "resolve ai failure is masked",
			aicall:      newCall(),
			tool:        newTool(`{"query":"hello"}`),
			aiErr:       fmt.Errorf("db down"),
			expectAIGet: true,
			expectRes:   &messageContent{Result: "failed", ToolCallID: "tool-search-1", Message: "could not retrieve AI configuration"},
		},
		{
			name:        "ai without knowledge base",
			aicall:      newCall(),
			tool:        newTool(`{"query":"hello"}`),
			aiResult:    &ai.AI{Identity: commonidentity.Identity{ID: assistanceID}},
			expectAIGet: true,
			expectRes:   &messageContent{Result: "failed", ToolCallID: "tool-search-1", Message: "no knowledge base is configured for this assistant"},
		},
		{
			name:         "rag query failure is masked",
			aicall:       newCall(),
			tool:         newTool(`{"query":"hello"}`),
			aiResult:     &ai.AI{Identity: commonidentity.Identity{ID: assistanceID}, RagID: ragID},
			ragErr:       fmt.Errorf("rag down"),
			expectAIGet:  true,
			expectRagReq: true,
			expectRes:    &messageContent{Result: "failed", ToolCallID: "tool-search-1", Message: "knowledge base search failed"},
		},
		{
			name:         "no sources found",
			aicall:       newCall(),
			tool:         newTool(`{"query":"hello"}`),
			aiResult:     &ai.AI{Identity: commonidentity.Identity{ID: assistanceID}, RagID: ragID},
			ragRes:       &rmquery.Response{},
			expectAIGet:  true,
			expectRagReq: true,
			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-search-1",
				ResourceType: "rag",
				ResourceID:   ragID.String(),
				Message:      "No relevant information found in the knowledge base.",
			},
		},
		{
			name:     "sources are rendered in order",
			aicall:   newCall(),
			tool:     newTool(`{"query":"hello"}`),
			aiResult: &ai.AI{Identity: commonidentity.Identity{ID: assistanceID}, RagID: ragID},
			ragRes: &rmquery.Response{Sources: []rmquery.Source{
				{DocumentName: "doc-a", SectionTitle: "sec-a", RelevanceScore: 0.9, Text: "text-a"},
				{DocumentName: "doc-b", SectionTitle: "sec-b", RelevanceScore: 0.456, Text: "text-b"},
			}},
			expectAIGet:  true,
			expectRagReq: true,
			expectRes: &messageContent{
				Result:       "success",
				ToolCallID:   "tool-search-1",
				ResourceType: "rag",
				ResourceID:   ragID.String(),
				Message: "[Source 1: \"doc-a\" > \"sec-a\" (relevance: 0.90)]\ntext-a\n\n" +
					"[Source 2: \"doc-b\" > \"sec-b\" (relevance: 0.46)]\ntext-b\n\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			h := &aicallHandler{reqHandler: mockReq, aiHandler: mockAI}
			ctx := context.Background()

			if tt.expectAIGet {
				mockAI.EXPECT().Get(ctx, assistanceID).Return(tt.aiResult, tt.aiErr)
			}
			if tt.expectRagReq {
				mockReq.EXPECT().RagV1RagQuery(ctx, ragID, "hello", 5).Return(tt.ragRes, tt.ragErr)
			}

			res := h.toolHandleSearchKnowledge(ctx, tt.aicall, tt.tool)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.expectRes, res)
			}
		})
	}
}

func Test_toolHandleSearchKnowledge_unsupportedAssistanceType(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &aicallHandler{
		reqHandler: requesthandler.NewMockRequestHandler(mc),
		aiHandler:  aihandler.NewMockAIHandler(mc),
	}

	c := &aicall.AIcall{
		Identity:       commonidentity.Identity{ID: uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-000000000011")},
		AssistanceType: aicall.AssistanceType("bogus"),
	}
	tool := &message.ToolCall{ID: "tool-search-2", Function: message.FunctionCall{Arguments: `{"query":"x"}`}}

	res := h.toolHandleSearchKnowledge(context.Background(), c, tool)
	expect := &messageContent{Result: "failed", ToolCallID: "tool-search-2", Message: "could not retrieve AI configuration"}
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, res)
	}
}

func Test_toolHandleDescribeAction(t *testing.T) {
	c := &aicall.AIcall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-000000000021")}}

	tests := []struct {
		name string
		args string

		expectResult     string
		expectResType    string
		expectResID      string
		expectMsgContain []string
	}{
		{
			name:             "invalid json arguments",
			args:             `not-json`,
			expectResult:     "failed",
			expectMsgContain: []string{"invalid character"},
		},
		{
			name:             "empty action type",
			args:             `{}`,
			expectResult:     "failed",
			expectMsgContain: []string{"action_type is required", "valid types:"},
		},
		{
			name:             "unknown action type echoes the value",
			args:             `{"action_type":"no_such_action"}`,
			expectResult:     "failed",
			expectMsgContain: []string{"no_such_action", "valid types:"},
		},
		{
			name:             "known action type",
			args:             `{"action_type":"amd"}`,
			expectResult:     "success",
			expectResType:    "action",
			expectResID:      "amd",
			expectMsgContain: []string{"amd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &aicallHandler{}
			tool := &message.ToolCall{ID: "tool-describe-1", Function: message.FunctionCall{Arguments: tt.args}}

			res := h.toolHandleDescribeAction(context.Background(), c, tool)

			if res.Result != tt.expectResult {
				t.Errorf("Wrong result. expect: %s, got: %s (%s)", tt.expectResult, res.Result, res.Message)
			}
			if res.ToolCallID != "tool-describe-1" {
				t.Errorf("Wrong tool call id. got: %s", res.ToolCallID)
			}
			if res.ResourceType != tt.expectResType || res.ResourceID != tt.expectResID {
				t.Errorf("Wrong resource. got: %s/%s", res.ResourceType, res.ResourceID)
			}
			for _, s := range tt.expectMsgContain {
				if !strings.Contains(res.Message, s) {
					t.Errorf("Message %q does not contain %q", res.Message, s)
				}
			}
		})
	}
}

// Test_toolHandle_invalidArgumentsAndDependencyErrors covers the failure
// branches of the simple tool handlers: arguments that are not valid JSON and
// errors returned by the dependency each handler calls.
func Test_toolHandle_invalidArgumentsAndDependencyErrors(t *testing.T) {
	aicallID := uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-000000000031")
	customerID := uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-000000000032")
	activeflowID := uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-000000000033")
	referenceID := uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-000000000034")
	otherAIcallID := uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-000000000035")
	otherCustomerID := uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-000000000036")

	callAIcall := func() *aicall.AIcall {
		return &aicall.AIcall{
			Identity:      commonidentity.Identity{ID: aicallID, CustomerID: customerID},
			ActiveflowID:  activeflowID,
			ReferenceType: aicall.ReferenceTypeCall,
			ReferenceID:   referenceID,
		}
	}
	noFlowAIcall := func() *aicall.AIcall {
		return &aicall.AIcall{Identity: commonidentity.Identity{ID: aicallID, CustomerID: customerID}}
	}

	tests := []struct {
		name string

		aicall  *aicall.AIcall
		handler func(h *aicallHandler, ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent
		args    string
		setup   func(mockReq *requesthandler.MockRequestHandler, mockDB *dbhandler.MockDBHandler, mockMsg *messagehandler.MockMessageHandler, mockUtil *utilhandler.MockUtilHandler)

		expectMessage         string
		expectMessagePrefixes string
	}{
		{
			name:    "connect invalid json",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleConnect,
			args:    `{`,
			setup: func(*requesthandler.MockRequestHandler, *dbhandler.MockDBHandler, *messagehandler.MockMessageHandler, *utilhandler.MockUtilHandler) {
			},

			expectMessage: "unexpected end of JSON input",
		},
		{
			name:    "connect add actions failure",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleConnect,
			args:    `{}`,
			setup: func(r *requesthandler.MockRequestHandler, _ *dbhandler.MockDBHandler, _ *messagehandler.MockMessageHandler, _ *utilhandler.MockUtilHandler) {
				r.EXPECT().FlowV1ActiveflowAddActions(gomock.Any(), activeflowID, gomock.Any()).Return(nil, fmt.Errorf("flow down"))
			},

			expectMessage: "flow down",
		},
		{
			name:    "message_send invalid json",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleMessageSend,
			args:    `[`,
			setup: func(*requesthandler.MockRequestHandler, *dbhandler.MockDBHandler, *messagehandler.MockMessageHandler, *utilhandler.MockUtilHandler) {
			},

			expectMessage: "unexpected end of JSON input",
		},
		{
			name:    "message_send request failure",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleMessageSend,
			args:    `{"text":"hi"}`,
			setup: func(r *requesthandler.MockRequestHandler, _ *dbhandler.MockDBHandler, _ *messagehandler.MockMessageHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-0000000000a1"))
				r.EXPECT().MessageV1MessageSend(gomock.Any(), uuid.FromStringOrNil("3b1f0a10-1a00-4c00-8000-0000000000a1"), customerID, gomock.Any(), gomock.Any(), "hi").Return(nil, fmt.Errorf("message down"))
			},

			expectMessage: "message down",
		},
		{
			name:    "email_send invalid json",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleEmailSend,
			args:    `[`,
			setup: func(*requesthandler.MockRequestHandler, *dbhandler.MockDBHandler, *messagehandler.MockMessageHandler, *utilhandler.MockUtilHandler) {
			},

			expectMessage: "unexpected end of JSON input",
		},
		{
			name:    "email_send request failure",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleEmailSend,
			args:    `{"subject":"s","content":"c"}`,
			setup: func(r *requesthandler.MockRequestHandler, _ *dbhandler.MockDBHandler, _ *messagehandler.MockMessageHandler, _ *utilhandler.MockUtilHandler) {
				r.EXPECT().EmailV1EmailSend(gomock.Any(), customerID, activeflowID, gomock.Any(), "s", "c", gomock.Any()).Return(nil, fmt.Errorf("email down"))
			},

			expectMessage: "email down",
		},
		{
			name:    "stop activeflow failure",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleStop,
			args:    `{}`,
			setup: func(r *requesthandler.MockRequestHandler, _ *dbhandler.MockDBHandler, _ *messagehandler.MockMessageHandler, _ *utilhandler.MockUtilHandler) {
				r.EXPECT().FlowV1ActiveflowStop(gomock.Any(), activeflowID).Return(nil, fmt.Errorf("stop down"))
			},

			expectMessage: "stop down",
		},
		{
			name:    "media_stop request failure",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleMediaStop,
			args:    `{}`,
			setup: func(r *requesthandler.MockRequestHandler, _ *dbhandler.MockDBHandler, _ *messagehandler.MockMessageHandler, _ *utilhandler.MockUtilHandler) {
				r.EXPECT().CallV1CallMediaStop(gomock.Any(), referenceID).Return(fmt.Errorf("media down"))
			},

			expectMessage: "media down",
		},
		{
			name:    "service_stop get aicall failure",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleServiceStop,
			args:    `{}`,
			setup: func(_ *requesthandler.MockRequestHandler, d *dbhandler.MockDBHandler, _ *messagehandler.MockMessageHandler, _ *utilhandler.MockUtilHandler) {
				d.EXPECT().AIcallGet(gomock.Any(), aicallID).Return(nil, fmt.Errorf("db down"))
			},

			expectMessagePrefixes: "could not start terminating the aicall. err: could not get the aicall correctly",
		},
		{
			name:    "set_variables invalid json",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleSetVariables,
			args:    `{"variables":`,
			setup: func(*requesthandler.MockRequestHandler, *dbhandler.MockDBHandler, *messagehandler.MockMessageHandler, *utilhandler.MockUtilHandler) {
			},

			expectMessage: "unexpected end of JSON input",
		},
		{
			name:    "set_variables request failure",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleSetVariables,
			args:    `{"variables":{"k":"v"}}`,
			setup: func(r *requesthandler.MockRequestHandler, _ *dbhandler.MockDBHandler, _ *messagehandler.MockMessageHandler, _ *utilhandler.MockUtilHandler) {
				r.EXPECT().FlowV1VariableSetVariable(gomock.Any(), activeflowID, map[string]string{"k": "v"}).Return(fmt.Errorf("variable down"))
			},

			expectMessage: "variable down",
		},
		{
			name:    "get_variables request failure",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleGetVariables,
			args:    `{}`,
			setup: func(r *requesthandler.MockRequestHandler, _ *dbhandler.MockDBHandler, _ *messagehandler.MockMessageHandler, _ *utilhandler.MockUtilHandler) {
				r.EXPECT().FlowV1VariableGet(gomock.Any(), activeflowID).Return(nil, fmt.Errorf("variable down"))
			},

			expectMessage: "variable down",
		},
		{
			name:    "get_variables without activeflow",
			aicall:  noFlowAIcall(),
			handler: (*aicallHandler).toolHandleGetVariables,
			args:    `{}`,
			setup: func(*requesthandler.MockRequestHandler, *dbhandler.MockDBHandler, *messagehandler.MockMessageHandler, *utilhandler.MockUtilHandler) {
			},

			expectMessage: "no activeflow associated with this aicall",
		},
		{
			name:    "get_aicall_messages invalid json",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleGetAIcallMessages,
			args:    `{"aicall_id":`,
			setup: func(*requesthandler.MockRequestHandler, *dbhandler.MockDBHandler, *messagehandler.MockMessageHandler, *utilhandler.MockUtilHandler) {
			},

			expectMessage: "unexpected end of JSON input",
		},
		{
			name:    "get_aicall_messages lookup failure",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleGetAIcallMessages,
			args:    `{"aicall_id":"3b1f0a10-1a00-4c00-8000-000000000035"}`,
			setup: func(_ *requesthandler.MockRequestHandler, d *dbhandler.MockDBHandler, _ *messagehandler.MockMessageHandler, _ *utilhandler.MockUtilHandler) {
				d.EXPECT().AIcallGet(gomock.Any(), otherAIcallID).Return(nil, fmt.Errorf("db down"))
			},

			expectMessage: "db down",
		},
		{
			name:    "get_aicall_messages customer mismatch",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleGetAIcallMessages,
			args:    `{"aicall_id":"3b1f0a10-1a00-4c00-8000-000000000035"}`,
			setup: func(_ *requesthandler.MockRequestHandler, d *dbhandler.MockDBHandler, _ *messagehandler.MockMessageHandler, _ *utilhandler.MockUtilHandler) {
				d.EXPECT().AIcallGet(gomock.Any(), otherAIcallID).Return(&aicall.AIcall{
					Identity: commonidentity.Identity{ID: otherAIcallID, CustomerID: otherCustomerID},
				}, nil)
			},

			expectMessage: "aicall customer id does not match",
		},
		{
			name:    "get_aicall_messages list failure",
			aicall:  callAIcall(),
			handler: (*aicallHandler).toolHandleGetAIcallMessages,
			args:    `{"aicall_id":"3b1f0a10-1a00-4c00-8000-000000000035"}`,
			setup: func(_ *requesthandler.MockRequestHandler, d *dbhandler.MockDBHandler, m *messagehandler.MockMessageHandler, _ *utilhandler.MockUtilHandler) {
				d.EXPECT().AIcallGet(gomock.Any(), otherAIcallID).Return(&aicall.AIcall{
					Identity: commonidentity.Identity{ID: otherAIcallID, CustomerID: customerID},
				}, nil)
				m.EXPECT().List(gomock.Any(), uint64(1000), "", map[message.Field]any{message.FieldAIcallID: otherAIcallID}).Return(nil, fmt.Errorf("list down"))
			},

			expectMessage: "list down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMsg := messagehandler.NewMockMessageHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &aicallHandler{
				reqHandler:     mockReq,
				db:             mockDB,
				messageHandler: mockMsg,
				utilHandler:    mockUtil,
			}
			tt.setup(mockReq, mockDB, mockMsg, mockUtil)

			tool := &message.ToolCall{ID: "tool-fail-1", Function: message.FunctionCall{Arguments: tt.args}}
			res := tt.handler(h, context.Background(), tt.aicall, tool)

			if res.Result != "failed" {
				t.Fatalf("Wrong result. expect: failed, got: %s (%s)", res.Result, res.Message)
			}
			if res.ToolCallID != "tool-fail-1" {
				t.Errorf("Wrong tool call id. got: %s", res.ToolCallID)
			}
			if tt.expectMessage != "" && res.Message != tt.expectMessage {
				t.Errorf("Wrong message. expect: %s, got: %s", tt.expectMessage, res.Message)
			}
			if tt.expectMessagePrefixes != "" && !strings.HasPrefix(res.Message, tt.expectMessagePrefixes) {
				t.Errorf("Wrong message prefix. expect: %s, got: %s", tt.expectMessagePrefixes, res.Message)
			}
		})
	}
}
