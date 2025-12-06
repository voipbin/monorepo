package aicallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	ememail "monorepo/bin-email-manager/models/email"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	mmmessage "monorepo/bin-message-manager/models/message"
	reflect "reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_toolHandleConnect(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		responseActiveflow *fmactiveflow.Activeflow

		expectActions []fmaction.Action
		expectRes     *messageContent
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("092306ea-bba3-11f0-838f-3713971974df"),
					CustomerID: uuid.FromStringOrNil("09678068-bba3-11f0-9e0d-87fbfeb3be46"),
				},
				ActiveflowID: uuid.FromStringOrNil("0990a6f0-bba3-11f0-9abf-a3303095e6e6"),
			},
			tool: &message.ToolCall{
				ID:   "1a6f5f40-f06f-11ef-8f5e-3f3b1d2e8e2f",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameConnect,
					Arguments: `{
						"source": {
							"type":   "tel",
							"target": "+123456789"
						},
						"destinations": [
							{
								"type":   "tel",
								"target": "+11111111"
							},
							{
								"type":   "tel",
								"target": "+22222222"
							}
						],
						"early_media":  true,
						"relay_reason": true
					}`,
				},
			},

			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0990a6f0-bba3-11f0-9abf-a3303095e6e6"),
				},
			},

			expectActions: []fmaction.Action{
				{
					Type: fmaction.TypeConnect,
					Option: map[string]any{
						"source": map[string]any{
							"type":   "tel",
							"target": "+123456789",
						},
						"destinations": []any{
							map[string]any{
								"type":   "tel",
								"target": "+11111111",
							},
							map[string]any{
								"type":   "tel",
								"target": "+22222222",
							},
						},
						"early_media":  true,
						"relay_reason": true,
					},
				},
			},
			expectRes: &messageContent{
				Result:       "success",
				Message:      "Added connect action successfully.",
				ToolCallID:   "1a6f5f40-f06f-11ef-8f5e-3f3b1d2e8e2f",
				ResourceType: "activeflow",
				ResourceID:   "0990a6f0-bba3-11f0-9abf-a3303095e6e6",
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

			mockReq.EXPECT().FlowV1ActiveflowAddActions(ctx, tt.aicall.ActiveflowID, tt.expectActions).Return(tt.responseActiveflow, nil)
			mockReq.EXPECT().AIV1AIcallTerminate(gomock.Any(), tt.aicall.ID).Return(&aicall.AIcall{}, nil)

			res := h.toolHandleConnect(ctx, tt.aicall, tt.tool)

			time.Sleep(100 * time.Millisecond)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_toolHandleMessageSend(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		responseUUIDMessageID uuid.UUID
		responseMMMessage     *mmmessage.Message

		expectSource       *commonaddress.Address
		expectDestinations []commonaddress.Address
		expectText         string
		expectRes          *messageContent
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4bc61c34-bbb3-11f0-9fe7-73c103bcd202"),
					CustomerID: uuid.FromStringOrNil("4c0ab9fc-bbb3-11f0-99a8-8f2a21cdfa73"),
				},
				ActiveflowID: uuid.FromStringOrNil("4c38c02c-bbb3-11f0-a8fb-337d7e8e91f6"),
			},
			tool: &message.ToolCall{
				ID:   "4c620144-bbb3-11f0-823e-77b11be82ade",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameMessageSend,
					Arguments: `{
						"source": {
							"type":   "tel",
							"target": "+123456789"
						},
						"destinations": [
							{
								"type":   "tel",
								"target": "+11111111"
							},
							{
								"type":   "tel",
								"target": "+22222222"
							}
						],
						"text": "test message"
					}`,
				},
			},

			responseUUIDMessageID: uuid.FromStringOrNil("4c8ad7cc-bbb3-11f0-a7ed-4f8dd1e691ca"),
			responseMMMessage: &mmmessage.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4c8ad7cc-bbb3-11f0-a7ed-4f8dd1e691ca"),
				},
			},

			expectSource: &commonaddress.Address{
				Type:   "tel",
				Target: "+123456789",
			},
			expectDestinations: []commonaddress.Address{
				{
					Type:   "tel",
					Target: "+11111111",
				},
				{
					Type:   "tel",
					Target: "+22222222",
				},
			},
			expectText: "test message",
			expectRes: &messageContent{
				ToolCallID:   "4c620144-bbb3-11f0-823e-77b11be82ade",
				Result:       "success",
				Message:      "Message sent successfully.",
				ResourceType: "message",
				ResourceID:   "4c8ad7cc-bbb3-11f0-a7ed-4f8dd1e691ca",
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

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDMessageID)
			mockReq.EXPECT().MessageV1MessageSend(ctx, tt.responseUUIDMessageID, tt.aicall.CustomerID, tt.expectSource, tt.expectDestinations, tt.expectText).Return(tt.responseMMMessage, nil)

			res := h.toolHandleMessageSend(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_toolHandleEmailSend(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		responseEmail   *ememail.Email
		responseMessage *message.Message

		expectDestinations []commonaddress.Address
		expectSubject      string
		expectContent      string
		expectAttachments  []ememail.Attachment
		expectRes          *messageContent
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6e42f08c-c23b-11f0-9a55-6bf59ab4c73a"),
					CustomerID: uuid.FromStringOrNil("6e74dfac-c23b-11f0-965a-53b4e7e7c614"),
				},
				ActiveflowID: uuid.FromStringOrNil("6ea2a126-c23b-11f0-9c83-af0130ca8099"),
			},
			tool: &message.ToolCall{
				ID:   "6ed073c6-c23b-11f0-9ada-035c2e737106",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameEmailSend,
					Arguments: `{
						"destinations": [
							{
								"type":   "email",
								"target": "test1@voipbin.net"
							},
							{
								"type":   "email",
								"target": "test2@voipbin.net"
							}
						],
						"subject": "test subject",
						"content":  "test content",
						"attachments": []
					}`,
				},
			},

			responseEmail: &ememail.Email{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("45e4d190-c23c-11f0-b814-43aec86b987f"),
				},
			},

			expectDestinations: []commonaddress.Address{
				{
					Type:   "email",
					Target: "test1@voipbin.net",
				},
				{
					Type:   "email",
					Target: "test2@voipbin.net",
				},
			},
			expectSubject:     "test subject",
			expectContent:     `test content`,
			expectAttachments: []ememail.Attachment{},
			expectRes: &messageContent{
				ToolCallID:   "6ed073c6-c23b-11f0-9ada-035c2e737106",
				Result:       "success",
				Message:      "Email sent successfully.",
				ResourceType: "email",
				ResourceID:   "45e4d190-c23c-11f0-b814-43aec86b987f",
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

			mockReq.EXPECT().EmailV1EmailSend(
				ctx,
				tt.aicall.CustomerID,
				tt.aicall.ActiveflowID,
				tt.expectDestinations,
				tt.expectSubject,
				tt.expectContent,
				tt.expectAttachments,
			).Return(tt.responseEmail, nil)

			res := h.toolHandleEmailSend(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_toolHandleServiceStop(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		expectRes *messageContent
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9e110dbc-d07d-11f0-9496-ab3dacff7ae1"),
					CustomerID: uuid.FromStringOrNil("6e74dfac-c23b-11f0-965a-53b4e7e7c614"),
				},
				ActiveflowID: uuid.FromStringOrNil("9e4caf8e-d07d-11f0-ba2d-5799bd8fb0b5"),
				ConfbridgeID: uuid.FromStringOrNil("eaf9f682-d0bb-11f0-adb3-33c1048e74d8"),
				ReferenceID:  uuid.FromStringOrNil("eb270d66-d0bb-11f0-87e0-279f3253f2c7"),
			},
			tool: &message.ToolCall{
				ID:   "9e70cf90-d07d-11f0-83f0-fb4840f79cfa",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameServiceStop,
					Arguments: `{}`,
				},
			},

			expectRes: &messageContent{
				ToolCallID:   "9e70cf90-d07d-11f0-83f0-fb4840f79cfa",
				Result:       "success",
				Message:      "Service stopped successfully.",
				ResourceType: "service",
				ResourceID:   "9e110dbc-d07d-11f0-9496-ab3dacff7ae1",
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

			// updateStatus
			mockDB.EXPECT().AIcallUpdateStatus(ctx, tt.aicall.ID, aicall.StatusTerminating).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.aicall.ID).Return(tt.aicall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.aicall.CustomerID, aicall.EventTypeStatusTerminating, tt.aicall)

			mockReq.EXPECT().FlowV1ActiveflowServiceStop(ctx, tt.aicall.ActiveflowID, tt.aicall.ID).Return(nil)
			mockReq.EXPECT().CallV1ConfbridgeCallKick(ctx, tt.aicall.ConfbridgeID, tt.aicall.ReferenceID).Return(nil)

			res := h.toolHandleServiceStop(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_toolHandleStop(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		responseActiveflow *fmactiveflow.Activeflow

		expectRes *messageContent
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c41debb8-d07f-11f0-b990-7b50d26bd157"),
					CustomerID: uuid.FromStringOrNil("6e74dfac-c23b-11f0-965a-53b4e7e7c614"),
				},
				ActiveflowID: uuid.FromStringOrNil("c454ff5e-d07f-11f0-91a8-1350b22b1220"),
			},
			tool: &message.ToolCall{
				ID:   "c482d2c6-d07f-11f0-9feb-c38f67824563",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameStop,
					Arguments: `{}`,
				},
			},

			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c454ff5e-d07f-11f0-91a8-1350b22b1220"),
				},
			},

			expectRes: &messageContent{
				Result:       "success",
				Message:      "Activeflow stopped successfully.",
				ToolCallID:   "c482d2c6-d07f-11f0-9feb-c38f67824563",
				ResourceType: "activeflow",
				ResourceID:   "c454ff5e-d07f-11f0-91a8-1350b22b1220",
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

			mockReq.EXPECT().FlowV1ActiveflowStop(ctx, tt.aicall.ActiveflowID).Return(tt.responseActiveflow, nil)

			res := h.toolHandleStop(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_toolHandleMediaStop(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		expectRes *messageContent
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6d3c498c-d1f3-11f0-bf4d-17a05c30f202"),
					CustomerID: uuid.FromStringOrNil("6e74dfac-c23b-11f0-965a-53b4e7e7c614"),
				},
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("6d9b17fa-d1f3-11f0-bfb9-db362a8ff4da"),
			},
			tool: &message.ToolCall{
				ID:   "6dc4c2f8-d1f3-11f0-8aed-5b0028fe1d06",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameMediaStop,
					Arguments: `{}`,
				},
			},

			expectRes: &messageContent{
				Result:       "success",
				Message:      "Call media stopped successfully.",
				ToolCallID:   "6dc4c2f8-d1f3-11f0-8aed-5b0028fe1d06",
				ResourceType: "call",
				ResourceID:   "6d9b17fa-d1f3-11f0-bfb9-db362a8ff4da",
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

			mockReq.EXPECT().CallV1CallMediaStop(ctx, tt.aicall.ReferenceID).Return(nil)

			res := h.toolHandleMediaStop(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_toolHandleSetVariables(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		expectedVariables map[string]string
		expectRes         *messageContent
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("df982c0a-d2b9-11f0-ae66-3b3a174682e4"),
					CustomerID: uuid.FromStringOrNil("6e74dfac-c23b-11f0-965a-53b4e7e7c614"),
				},
				ActiveflowID: uuid.FromStringOrNil("dfd7b384-d2b9-11f0-a6cf-576bb7892c7c"),
			},
			tool: &message.ToolCall{
				ID:   "e005d49e-d2b9-11f0-bb53-2f0f52e36b88",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name: message.FunctionCallNameSetVariables,
					Arguments: `{
						"key1": "value1",
						"key2": "value2"
					}`,
				},
			},

			expectedVariables: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expectRes: &messageContent{
				Result:       "success",
				Message:      "Variables set successfully.",
				ToolCallID:   "e005d49e-d2b9-11f0-bb53-2f0f52e36b88",
				ResourceType: "activeflow",
				ResourceID:   "dfd7b384-d2b9-11f0-a6cf-576bb7892c7c",
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

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.aicall.ActiveflowID, tt.expectedVariables).Return(nil)

			res := h.toolHandleSetVariables(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_toolHandleGetVariables(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		responseVariable *fmvariable.Variable

		expectRes *messageContent
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d21ceb22-d2c0-11f0-a553-8f1245a47cf1"),
					CustomerID: uuid.FromStringOrNil("6e74dfac-c23b-11f0-965a-53b4e7e7c614"),
				},
				ActiveflowID: uuid.FromStringOrNil("d281281c-d2c0-11f0-b5b7-6744b7d2eeac"),
			},
			tool: &message.ToolCall{
				ID:   "d2581cba-d2c0-11f0-97f8-efbee72f536d",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetVariables,
					Arguments: `{}`,
				},
			},

			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},

			expectRes: &messageContent{
				Result:       "success",
				Message:      `{"key1":"value1","key2":"value2"}`,
				ToolCallID:   "d2581cba-d2c0-11f0-97f8-efbee72f536d",
				ResourceType: "activeflow",
				ResourceID:   "d281281c-d2c0-11f0-b5b7-6744b7d2eeac",
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

			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.aicall.ActiveflowID).Return(tt.responseVariable, nil)

			res := h.toolHandleGetVariables(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
