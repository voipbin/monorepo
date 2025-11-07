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
	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	mmmessage "monorepo/bin-message-manager/models/message"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_toolHandleConnect(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		expectActions []fmaction.Action
		expectContent string
		expectRes     map[string]any
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
					Arguments: map[string]any{
						"source": map[string]any{
							"type":   "tel",
							"target": "+123456789",
						},
						"destinations": []map[string]any{
							{
								"type":   "tel",
								"target": "+11111111",
							},
							{
								"type":   "tel",
								"target": "+22222222",
							},
						},
						"early_media":  true,
						"relay_reason": true,
					},
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
			expectContent: `{"result": "success", "message": ""}`,
			expectRes: map[string]any{
				"result":  "success",
				"message": "",
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

			mockReq.EXPECT().FlowV1ActiveflowAddActions(ctx, tt.aicall.ActiveflowID, tt.expectActions).Return(&fmactiveflow.Activeflow{}, nil)
			mockMessage.EXPECT().Create(ctx, tt.aicall.CustomerID, tt.aicall.ID, message.DirectionOutgoing, message.RoleTool, tt.expectContent, nil, tt.tool.ID).Return(&message.Message{}, nil)

			res, err := h.toolHandleConnect(ctx, tt.aicall, tt.tool)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

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

		expectSource       *commonaddress.Address
		expectDestinations []commonaddress.Address
		expectText         string
		expectContent      string
		expectRes          map[string]any
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
					Arguments: map[string]any{
						"source": map[string]any{
							"type":   "tel",
							"target": "+123456789",
						},
						"destinations": []map[string]any{
							{
								"type":   "tel",
								"target": "+11111111",
							},
							{
								"type":   "tel",
								"target": "+22222222",
							},
						},
						"text": "test message",
					},
				},
			},

			responseUUIDMessageID: uuid.FromStringOrNil("4c8ad7cc-bbb3-11f0-a7ed-4f8dd1e691ca"),

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
			expectText:    "test message",
			expectContent: `{"result": "success", "message": ""}`,
			expectRes: map[string]any{
				"result":  "success",
				"message": "",
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

			mockReq.EXPECT().MessageV1MessageSend(ctx, tt.responseUUIDMessageID, tt.aicall.CustomerID, tt.expectSource, tt.expectDestinations, tt.expectText).Return(&mmmessage.Message{}, nil)
			mockMessage.EXPECT().Create(ctx, tt.aicall.CustomerID, tt.aicall.ID, message.DirectionOutgoing, message.RoleTool, tt.expectContent, nil, tt.tool.ID).Return(&message.Message{}, nil)

			res, err := h.toolHandleMessageSend(ctx, tt.aicall, tt.tool)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
