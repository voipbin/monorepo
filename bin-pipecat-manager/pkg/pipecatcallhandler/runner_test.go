package pipecatcallhandler

import (
	"context"
	"fmt"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	amateam "monorepo/bin-ai-manager/models/team"
	aitool "monorepo/bin-ai-manager/models/tool"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
)

func Test_receiveMessageFrameTypeMessage(t *testing.T) {

	tests := []struct {
		name string

		se *pipecatcall.Session
		m  []byte

		responseUUID  uuid.UUID
		expectEvent   string
		expectMessage message.Message
	}{
		{
			name: "bot-transcription",

			se: &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("54416ae0-af23-11f0-8991-07dd3ffd4def"),
					CustomerID: uuid.FromStringOrNil("546f7606-af23-11f0-a7ca-c32fd2659ee7"),
				},
				Ctx: context.Background(),
			},
			m: []byte(`{
				"label": "rtvi-ai",
				"type": "bot-transcription",
				"data": {"text": " How can I assist you today?"}
			}`),

			responseUUID: uuid.FromStringOrNil("c15f98f8-af1f-11f0-b009-535ac8cbc876"),
			expectEvent:  message.EventTypeBotTranscription,
			expectMessage: message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c15f98f8-af1f-11f0-b009-535ac8cbc876"),
					CustomerID: uuid.FromStringOrNil("546f7606-af23-11f0-a7ca-c32fd2659ee7"),
				},
				PipecatcallID: uuid.FromStringOrNil("54416ae0-af23-11f0-8991-07dd3ffd4def"),
				Text:          " How can I assist you today?",
			},
		},
		{
			name: "user-transcription",
			se: &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("54986764-af23-11f0-9793-db91dfe17f29"),
					CustomerID: uuid.FromStringOrNil("54c1efee-af23-11f0-af7c-a7f393ea7de5"),
				},
				Ctx: context.Background(),
			},
			m: []byte(`{
				"label": "rtvi-ai",
				"type": "user-transcription",
				"data": {"text": "to by the way, who are you?", "user_id": "", "timestamp": "2025-10-22T02:38:39.119+00:00", "final": true}
			}`),

			responseUUID: uuid.FromStringOrNil("54eb0456-af23-11f0-986c-4bb2d9cd75de"),
			expectEvent:  message.EventTypeUserTranscription,

			expectMessage: message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("54eb0456-af23-11f0-986c-4bb2d9cd75de"),
					CustomerID: uuid.FromStringOrNil("54c1efee-af23-11f0-af7c-a7f393ea7de5"),
				},
				PipecatcallID: uuid.FromStringOrNil("54986764-af23-11f0-9793-db91dfe17f29"),
				Text:          "to by the way, who are you?",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := pipecatcallHandler{
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockNotify.EXPECT().PublishEvent(tt.se.Ctx, tt.expectEvent, tt.expectMessage)

			if err := h.receiveMessageFrameTypeMessage(tt.se, tt.m); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test_runnerWebsocketHandleAudio(t *testing.T) {

	t.Run("16kHz mono audio passes through without resampling", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatcallHandler{
			websocketHandler: mockWS,
		}

		// ConnAst is non-nil so WriteMessage will be called to forward
		// the audio data directly to Asterisk.
		conn := &websocket.Conn{}
		se := &pipecatcall.Session{
			Ctx:     context.Background(),
			ConnAst: conn,
		}

		data := []byte{0x01, 0x02, 0x03, 0x04}
		mockWS.EXPECT().WriteMessage(conn, websocket.BinaryMessage, data).Return(nil)

		if err := h.runnerWebsocketHandleAudio(se, 16000, 1, data); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("non-16kHz audio is resampled before writing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockAudio := NewMockAudiosocketHandler(mc)
		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatcallHandler{
			audiosocketHandler: mockAudio,
			websocketHandler:   mockWS,
		}

		conn := &websocket.Conn{}
		se := &pipecatcall.Session{
			Ctx:     context.Background(),
			ConnAst: conn,
		}

		inputData := []byte{0x01, 0x02, 0x03, 0x04}
		resampledData := []byte{0x10, 0x20}

		mockAudio.EXPECT().GetDataSamples(8000, inputData).Return(resampledData, nil)
		mockWS.EXPECT().WriteMessage(conn, websocket.BinaryMessage, resampledData).Return(nil)

		if err := h.runnerWebsocketHandleAudio(se, 8000, 1, inputData); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("24kHz audio is resampled (pipecat default rate safety net)", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockAudio := NewMockAudiosocketHandler(mc)
		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatcallHandler{
			audiosocketHandler: mockAudio,
			websocketHandler:   mockWS,
		}

		conn := &websocket.Conn{}
		se := &pipecatcall.Session{
			Ctx:     context.Background(),
			ConnAst: conn,
		}

		// 24kHz is Pipecat's default audio_out_sample_rate. If PipelineParams
		// doesn't set audio_out_sample_rate=16000, TTS outputs 24kHz and this
		// resampling path runs per chunk — creating boundary artifacts.
		inputData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
		resampledData := []byte{0x10, 0x20, 0x30, 0x40}

		mockAudio.EXPECT().GetDataSamples(24000, inputData).Return(resampledData, nil)
		mockWS.EXPECT().WriteMessage(conn, websocket.BinaryMessage, resampledData).Return(nil)

		if err := h.runnerWebsocketHandleAudio(se, 24000, 1, inputData); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("stereo audio is rejected", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &pipecatcallHandler{}
		se := &pipecatcall.Session{
			Ctx: context.Background(),
		}

		err := h.runnerWebsocketHandleAudio(se, 16000, 2, []byte{0x01, 0x02})
		if err == nil {
			t.Errorf("expected error for stereo audio, got nil")
		}
	})

	t.Run("nil ConnAst returns nil without writing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &pipecatcallHandler{}
		se := &pipecatcall.Session{
			Ctx:     context.Background(),
			ConnAst: nil,
		}

		if err := h.runnerWebsocketHandleAudio(se, 16000, 1, []byte{0x01, 0x02}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty data returns nil", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &pipecatcallHandler{}
		conn := &websocket.Conn{}
		se := &pipecatcall.Session{
			Ctx:     context.Background(),
			ConnAst: conn,
		}

		// Empty data returns nil without calling WriteMessage
		if err := h.runnerWebsocketHandleAudio(se, 16000, 1, []byte{}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func Test_getToolsForPipecatcall(t *testing.T) {
	aiID := uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111")
	teamID := uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222")
	memberID := uuid.FromStringOrNil("c3c3c3c3-3333-3333-3333-333333333333")
	memberAIID := uuid.FromStringOrNil("d4d4d4d4-4444-4444-4444-444444444444")
	pipecatcallID := uuid.FromStringOrNil("e5e5e5e5-5555-5555-5555-555555555555")
	referenceID := uuid.FromStringOrNil("f6f6f6f6-6666-6666-6666-666666666666")

	allTools := []aitool.Tool{
		{Name: aitool.ToolNameConnectCall, Description: "connect call"},
		{Name: aitool.ToolNameSendEmail, Description: "send email"},
		{Name: aitool.ToolNameStopFlow, Description: "stop flow"},
	}

	filteredTools := []aitool.Tool{
		{Name: aitool.ToolNameConnectCall, Description: "connect call"},
		{Name: aitool.ToolNameStopFlow, Description: "stop flow"},
	}

	tests := []struct {
		name string

		pc *pipecatcall.Pipecatcall

		prepareMockFn func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler)

		expectedTools []aitool.Tool
	}{
		{
			name: "aicall with ai assistance filters tools by ai tool names",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{
					AssistanceType: amaicall.AssistanceTypeAI,
					AssistanceID:   aiID,
				}, nil)
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{
					Identity: commonidentity.Identity{
						ID: aiID,
					},
					ToolNames: []aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameStopFlow},
				}, nil)
				mockTool.EXPECT().GetByNames([]aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameStopFlow}).Return(filteredTools)
			},

			expectedTools: filteredTools,
		},
		{
			name: "aicall with team assistance filters tools by start member ai tool names",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{
					AssistanceType: amaicall.AssistanceTypeTeam,
					AssistanceID:   teamID,
				}, nil)
				mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(&amateam.Team{
					Identity: commonidentity.Identity{
						ID: teamID,
					},
					StartMemberID: memberID,
					Members: []amateam.Member{
						{ID: memberID, AIID: memberAIID},
					},
				}, nil)
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), memberAIID).Return(&amai.AI{
					Identity: commonidentity.Identity{
						ID: memberAIID,
					},
					ToolNames: []aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameStopFlow},
				}, nil)
				mockTool.EXPECT().GetByNames([]aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameStopFlow}).Return(filteredTools)
			},

			expectedTools: filteredTools,
		},
		{
			name: "non-aicall reference returns all tools",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeCall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockTool.EXPECT().GetAll().Return(allTools)
			},

			expectedTools: allTools,
		},
		{
			name: "aicall fetch error falls back to all tools",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(nil, fmt.Errorf("aicall not found"))
				mockTool.EXPECT().GetAll().Return(allTools)
			},

			expectedTools: allTools,
		},
		{
			name: "ai resolve error falls back to all tools",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{
					AssistanceType: amaicall.AssistanceTypeTeam,
					AssistanceID:   teamID,
				}, nil)
				mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(nil, fmt.Errorf("team not found"))
				mockTool.EXPECT().GetAll().Return(allTools)
			},

			expectedTools: allTools,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTool := toolhandler.NewMockToolHandler(mc)
			tt.prepareMockFn(mockReq, mockTool)

			h := &pipecatcallHandler{
				requestHandler: mockReq,
				toolHandler:    mockTool,
			}

			result := h.getToolsForPipecatcall(context.Background(), tt.pc)
			if len(result) != len(tt.expectedTools) {
				t.Errorf("Wrong number of tools. expect: %d, got: %d", len(tt.expectedTools), len(result))
				return
			}

			for i, tool := range result {
				if tool.Name != tt.expectedTools[i].Name {
					t.Errorf("Wrong tool at index %d. expect: %s, got: %s", i, tt.expectedTools[i].Name, tool.Name)
				}
			}
		})
	}
}
