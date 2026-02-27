package pipecatcallhandler

import (
	"context"
	"fmt"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	amateam "monorepo/bin-ai-manager/models/team"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

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
