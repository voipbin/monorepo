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
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
)

func Test_defaultMediaConstants(t *testing.T) {
	// These constants must match Asterisk's chan_websocket slin16 format.
	// Pipecat's PipelineParams.audio_out_sample_rate must also be set to 16000
	// to avoid Go-side per-chunk resampling which causes robotic audio.
	if defaultMediaSampleRate != 16000 {
		t.Errorf("defaultMediaSampleRate = %d, want 16000 (Asterisk slin16)", defaultMediaSampleRate)
	}
	if defaultMediaNumChannel != 1 {
		t.Errorf("defaultMediaNumChannel = %d, want 1 (mono)", defaultMediaNumChannel)
	}
}

func Test_runAsteriskReceivedMediaHandle(t *testing.T) {
	tests := []struct {
		name string

		readMessages []struct {
			msgType int
			data    []byte
			err     error
		}

		expectAudioFrames int
	}{
		{
			name: "receives binary audio frames",
			readMessages: []struct {
				msgType int
				data    []byte
				err     error
			}{
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: 0, data: nil, err: fmt.Errorf("connection closed")},
			},
			expectAudioFrames: 2,
		},
		{
			name: "skips non-binary messages",
			readMessages: []struct {
				msgType int
				data    []byte
				err     error
			}{
				{msgType: websocket.TextMessage, data: []byte("text"), err: nil},
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: 0, data: nil, err: fmt.Errorf("connection closed")},
			},
			expectAudioFrames: 1,
		},
		{
			name: "skips empty binary messages",
			readMessages: []struct {
				msgType int
				data    []byte
				err     error
			}{
				{msgType: websocket.BinaryMessage, data: []byte{}, err: nil},
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: 0, data: nil, err: fmt.Errorf("connection closed")},
			},
			expectAudioFrames: 1,
		},
		{
			name:              "nil ConnAst returns immediately",
			readMessages:      nil,
			expectAudioFrames: 0,
		},
		{
			name: "websocket close normal closure",
			readMessages: []struct {
				msgType int
				data    []byte
				err     error
			}{
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: 0, data: nil, err: &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "normal"}},
			},
			expectAudioFrames: 1,
		},
		{
			name: "websocket close going away",
			readMessages: []struct {
				msgType int
				data    []byte
				err     error
			}{
				{msgType: 0, data: nil, err: &websocket.CloseError{Code: websocket.CloseGoingAway, Text: "going away"}},
			},
			expectAudioFrames: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockWS := NewMockWebsocketHandler(mc)
			mockPF := NewMockPipecatframeHandler(mc)

			var conn *websocket.Conn
			if tt.readMessages != nil {
				conn = &websocket.Conn{}
				for _, msg := range tt.readMessages {
					mockWS.EXPECT().ReadMessage(conn).Return(msg.msgType, msg.data, msg.err)
				}
			}

			if tt.expectAudioFrames > 0 {
				mockPF.EXPECT().SendAudio(gomock.Any(), gomock.Any(), gomock.Any()).Times(tt.expectAudioFrames).Return(nil)
			}

			se := &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID: uuid.Must(uuid.NewV4()),
				},
				Ctx:     context.Background(),
				ConnAst: conn,
			}

			h := &pipecatcallHandler{
				websocketHandler:    mockWS,
				pipecatframeHandler: mockPF,
			}

			h.runAsteriskReceivedMediaHandle(se)
		})
	}
}

func Test_runAsteriskReceivedMediaHandle_contextCancelled(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockWS := NewMockWebsocketHandler(mc)
	mockPF := NewMockPipecatframeHandler(mc)
	// No ReadMessage expectations — context is cancelled before any read

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID: uuid.Must(uuid.NewV4()),
		},
		Ctx:     ctx,
		ConnAst: &websocket.Conn{},
	}

	h := &pipecatcallHandler{
		websocketHandler:    mockWS,
		pipecatframeHandler: mockPF,
	}

	h.runAsteriskReceivedMediaHandle(se)
	// Should return without panic or hanging
}

func Test_resolveAIFromAIcall(t *testing.T) {
	aiID := uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111")
	teamID := uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222")
	memberID := uuid.FromStringOrNil("c3c3c3c3-3333-3333-3333-333333333333")
	memberAIID := uuid.FromStringOrNil("d4d4d4d4-4444-4444-4444-444444444444")

	tests := []struct {
		name string

		aicall *amaicall.AIcall

		prepareMockFn func(mockReq *requesthandler.MockRequestHandler)

		expectedAIID uuid.UUID
		expectedErr  bool
	}{
		{
			name: "assistance type ai resolves directly",

			aicall: &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeAI,
				AssistanceID:   aiID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{
					Identity: commonidentity.Identity{
						ID: aiID,
					},
					EngineKey: "test-key",
				}, nil)
			},

			expectedAIID: aiID,
		},
		{
			name: "assistance type team resolves via start member",

			aicall: &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeTeam,
				AssistanceID:   teamID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(&amateam.Team{
					Identity: commonidentity.Identity{
						ID: teamID,
					},
					StartMemberID: memberID,
					Members: []amateam.Member{
						{
							ID:   memberID,
							AIID: memberAIID,
						},
					},
				}, nil)
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), memberAIID).Return(&amai.AI{
					Identity: commonidentity.Identity{
						ID: memberAIID,
					},
					EngineKey: "member-key",
				}, nil)
			},

			expectedAIID: memberAIID,
		},
		{
			name: "assistance type team with start member not found",

			aicall: &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeTeam,
				AssistanceID:   teamID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(&amateam.Team{
					Identity: commonidentity.Identity{
						ID: teamID,
					},
					StartMemberID: uuid.FromStringOrNil("eeee0000-0000-0000-0000-000000000000"),
					Members: []amateam.Member{
						{
							ID:   memberID,
							AIID: memberAIID,
						},
					},
				}, nil)
			},

			expectedErr: true,
		},
		{
			name: "assistance type team with team fetch error",

			aicall: &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeTeam,
				AssistanceID:   teamID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(nil, fmt.Errorf("team not found"))
			},

			expectedErr: true,
		},
		{
			name: "default assistance type resolves as ai",

			aicall: &amaicall.AIcall{
				AssistanceType: "",
				AssistanceID:   aiID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{
					Identity: commonidentity.Identity{
						ID: aiID,
					},
					EngineKey: "default-key",
				}, nil)
			},

			expectedAIID: aiID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			tt.prepareMockFn(mockReq)

			h := &pipecatcallHandler{
				requestHandler: mockReq,
			}

			result, err := h.resolveAIFromAIcall(context.Background(), tt.aicall)
			if tt.expectedErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.ID != tt.expectedAIID {
				t.Errorf("Wrong AI ID. expect: %v, got: %v", tt.expectedAIID, result.ID)
			}
		})
	}
}

func Test_resolveTeamForPython(t *testing.T) {
	teamID := uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222")
	member1ID := uuid.FromStringOrNil("c3c3c3c3-3333-3333-3333-333333333333")
	member2ID := uuid.FromStringOrNil("c4c4c4c4-4444-4444-4444-444444444444")
	ai1ID := uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111")
	ai2ID := uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222")

	tests := []struct {
		name string

		aicall *amaicall.AIcall

		prepareMockFn func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler)

		expectedNil bool
		expectedErr bool
		validate    func(t *testing.T, result *resolvedTeamData)
	}{
		{
			name: "non-team aicall returns nil",

			aicall: &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeAI,
				AssistanceID:   ai1ID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				// No calls expected for non-team type
			},

			expectedNil: true,
		},
		{
			name: "team with two members resolves all AIs and tools",

			aicall: &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeTeam,
				AssistanceID:   teamID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(&amateam.Team{
					Identity: commonidentity.Identity{
						ID: teamID,
					},
					StartMemberID: member1ID,
					Members: []amateam.Member{
						{
							ID:   member1ID,
							Name: "greeter",
							AIID: ai1ID,
							Transitions: []amateam.Transition{
								{
									FunctionName: "transfer_to_support",
									Description:  "Transfer to support agent",
									NextMemberID: member2ID,
								},
							},
						},
						{
							ID:   member2ID,
							Name: "support",
							AIID: ai2ID,
							Transitions: []amateam.Transition{
								{
									FunctionName: "transfer_to_greeter",
									Description:  "Transfer back to greeter",
									NextMemberID: member1ID,
								},
							},
						},
					},
				}, nil)

				mockReq.EXPECT().AIV1AIGet(gomock.Any(), ai1ID).Return(&amai.AI{
					Identity: commonidentity.Identity{
						ID: ai1ID,
					},
					EngineModel: amai.EngineModelOpenaiGPT4O,
					EngineKey:   "key-1",
					InitPrompt:  "You are a greeter",
					Parameter:   map[string]any{"temperature": 0.7},
					TTSType:     amai.TTSTypeElevenLabs,
					TTSVoiceID:  "voice-1",
					STTType:     amai.STTTypeDeepgram,
					ToolNames:   []aitool.ToolName{aitool.ToolNameConnectCall},
				}, nil)

				mockReq.EXPECT().AIV1AIGet(gomock.Any(), ai2ID).Return(&amai.AI{
					Identity: commonidentity.Identity{
						ID: ai2ID,
					},
					EngineModel: amai.EngineModelOpenaiGPT4OMini,
					EngineKey:   "key-2",
					InitPrompt:  "You are support",
					TTSType:     amai.TTSTypeCartesia,
					TTSVoiceID:  "voice-2",
					STTType:     amai.STTTypeDeepgram,
					ToolNames:   []aitool.ToolName{aitool.ToolNameSendEmail},
				}, nil)

				mockTool.EXPECT().GetByNames([]aitool.ToolName{aitool.ToolNameConnectCall}).Return([]aitool.Tool{
					{
						Name:        aitool.ToolNameConnectCall,
						Description: "Connect a call",
						Parameters:  map[string]any{"destination": "string"},
					},
				})

				mockTool.EXPECT().GetByNames([]aitool.ToolName{aitool.ToolNameSendEmail}).Return([]aitool.Tool{
					{
						Name:        aitool.ToolNameSendEmail,
						Description: "Send an email",
						Parameters:  map[string]any{"to": "string"},
					},
				})
			},

			validate: func(t *testing.T, result *resolvedTeamData) {
				if result.ID != teamID {
					t.Errorf("Wrong team ID. expect: %v, got: %v", teamID, result.ID)
				}
				if result.StartMemberID != member1ID {
					t.Errorf("Wrong start member ID. expect: %v, got: %v", member1ID, result.StartMemberID)
				}
				if len(result.Members) != 2 {
					t.Fatalf("Expected 2 members, got %d", len(result.Members))
				}

				// Verify first member
				m1 := result.Members[0]
				if m1.ID != member1ID {
					t.Errorf("Wrong member 1 ID. expect: %v, got: %v", member1ID, m1.ID)
				}
				if m1.Name != "greeter" {
					t.Errorf("Wrong member 1 name. expect: greeter, got: %s", m1.Name)
				}
				if m1.AI.EngineModel != string(amai.EngineModelOpenaiGPT4O) {
					t.Errorf("Wrong member 1 engine model. expect: %s, got: %s", amai.EngineModelOpenaiGPT4O, m1.AI.EngineModel)
				}
				if m1.AI.EngineKey != "key-1" {
					t.Errorf("Wrong member 1 engine key. expect: key-1, got: %s", m1.AI.EngineKey)
				}
				if m1.AI.InitPrompt != "You are a greeter" {
					t.Errorf("Wrong member 1 init prompt. expect: You are a greeter, got: %s", m1.AI.InitPrompt)
				}
				if m1.AI.TTSType != string(amai.TTSTypeElevenLabs) {
					t.Errorf("Wrong member 1 TTS type. expect: %s, got: %s", amai.TTSTypeElevenLabs, m1.AI.TTSType)
				}
				if m1.AI.TTSVoiceID != "voice-1" {
					t.Errorf("Wrong member 1 TTS voice ID. expect: voice-1, got: %s", m1.AI.TTSVoiceID)
				}
				if m1.AI.STTType != string(amai.STTTypeDeepgram) {
					t.Errorf("Wrong member 1 STT type. expect: %s, got: %s", amai.STTTypeDeepgram, m1.AI.STTType)
				}
				if len(m1.Tools) != 1 || m1.Tools[0].Name != aitool.ToolNameConnectCall {
					t.Errorf("Wrong member 1 tools. got: %v", m1.Tools)
				}
				if len(m1.Transitions) != 1 || m1.Transitions[0].FunctionName != "transfer_to_support" {
					t.Errorf("Wrong member 1 transitions. got: %v", m1.Transitions)
				}

				// Verify second member
				m2 := result.Members[1]
				if m2.ID != member2ID {
					t.Errorf("Wrong member 2 ID. expect: %v, got: %v", member2ID, m2.ID)
				}
				if m2.Name != "support" {
					t.Errorf("Wrong member 2 name. expect: support, got: %s", m2.Name)
				}
				if m2.AI.EngineKey != "key-2" {
					t.Errorf("Wrong member 2 engine key. expect: key-2, got: %s", m2.AI.EngineKey)
				}
			},
		},
		{
			name: "team fetch error returns error",

			aicall: &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeTeam,
				AssistanceID:   teamID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(nil, fmt.Errorf("team not found"))
			},

			expectedErr: true,
		},
		{
			name: "member AI fetch error returns error",

			aicall: &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeTeam,
				AssistanceID:   teamID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(&amateam.Team{
					Identity: commonidentity.Identity{
						ID: teamID,
					},
					StartMemberID: member1ID,
					Members: []amateam.Member{
						{
							ID:   member1ID,
							Name: "greeter",
							AIID: ai1ID,
						},
					},
				}, nil)
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), ai1ID).Return(nil, fmt.Errorf("ai not found"))
			},

			expectedErr: true,
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

			result, err := h.resolveTeamForPython(context.Background(), tt.aicall)
			if tt.expectedErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectedNil {
				if result != nil {
					t.Errorf("Expected nil result but got: %v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected non-nil result but got nil")
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func Test_runGetLLMKey(t *testing.T) {
	aiID := uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111")
	teamID := uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222")
	memberID := uuid.FromStringOrNil("c3c3c3c3-3333-3333-3333-333333333333")
	memberAIID := uuid.FromStringOrNil("d4d4d4d4-4444-4444-4444-444444444444")
	pipecatcallID := uuid.FromStringOrNil("e5e5e5e5-5555-5555-5555-555555555555")
	referenceID := uuid.FromStringOrNil("f6f6f6f6-6666-6666-6666-666666666666")

	tests := []struct {
		name string

		pc *pipecatcall.Pipecatcall

		prepareMockFn func(mockReq *requesthandler.MockRequestHandler)

		expectedKey string
	}{
		{
			name: "aicall reference with ai assistance returns key",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{
					AssistanceType: amaicall.AssistanceTypeAI,
					AssistanceID:   aiID,
				}, nil)
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{
					Identity: commonidentity.Identity{
						ID: aiID,
					},
					EngineKey: "ai-direct-key",
				}, nil)
			},

			expectedKey: "ai-direct-key",
		},
		{
			name: "aicall reference with team assistance returns start member key",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
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
					EngineKey: "team-member-key",
				}, nil)
			},

			expectedKey: "team-member-key",
		},
		{
			name: "aicall reference with aicall fetch error returns empty",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(nil, fmt.Errorf("aicall not found"))
			},

			expectedKey: "",
		},
		{
			name: "aicall reference with ai resolve error returns empty",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{
					AssistanceType: amaicall.AssistanceTypeAI,
					AssistanceID:   aiID,
				}, nil)
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(nil, fmt.Errorf("ai not found"))
			},

			expectedKey: "",
		},
		{
			name: "non-aicall reference type returns empty",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeCall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {},

			expectedKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			tt.prepareMockFn(mockReq)

			h := &pipecatcallHandler{
				requestHandler: mockReq,
			}

			result := h.runGetLLMKey(context.Background(), tt.pc)
			if result != tt.expectedKey {
				t.Errorf("Wrong LLM key. expect: %q, got: %q", tt.expectedKey, result)
			}
		})
	}
}
