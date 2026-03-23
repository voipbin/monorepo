package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/aicallhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	"monorepo/bin-common-handler/models/sock"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

func TestProcessEventPMMessageUserTranscription(t *testing.T) {
	tests := []struct {
		name      string
		event     *sock.Event
		setupMock func(*messagehandler.MockMessageHandler)
		wantError bool
	}{
		{
			name: "processes_user_transcription_event_successfully",
			event: func() *sock.Event {
				msg := &pmmessage.Message{
					PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
					PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
					Text:                     "Hello, how are you?",
				}
				msg.CustomerID = uuid.Must(uuid.NewV4())

				data, _ := json.Marshal(msg)
				return &sock.Event{
					Publisher: "pipecat-manager",
					Type:      string(pmmessage.EventTypeUserTranscription),
					Data:      json.RawMessage(data),
				}
			}(),
			setupMock: func(m *messagehandler.MockMessageHandler) {
				m.EXPECT().EventPMMessageUserTranscription(gomock.Any(), gomock.Any()).Times(1)
			},
			wantError: false,
		},
		{
			name: "handles_invalid_json_data",
			event: &sock.Event{
				Publisher: "pipecat-manager",
				Type:      string(pmmessage.EventTypeUserTranscription),
				Data:      json.RawMessage([]byte("invalid json")),
			},
			setupMock: func(m *messagehandler.MockMessageHandler) {
				// Should not be called on error
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMessageHandler := messagehandler.NewMockMessageHandler(ctrl)
			tt.setupMock(mockMessageHandler)

			h := &subscribeHandler{
				messageHandler: mockMessageHandler,
			}

			err := h.processEventPMMessageUserTranscription(context.Background(), tt.event)
			if (err != nil) != tt.wantError {
				t.Errorf("processEventPMMessageUserTranscription() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestProcessEventPMMessageBotLLM(t *testing.T) {
	tests := []struct {
		name      string
		event     *sock.Event
		setupMock func(*messagehandler.MockMessageHandler)
		wantError bool
	}{
		{
			name: "processes_bot_llm_event_successfully",
			event: func() *sock.Event {
				msg := &pmmessage.Message{
					PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
					PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
					Text:                     "I'm doing well, thank you!",
				}
				msg.CustomerID = uuid.Must(uuid.NewV4())

				data, _ := json.Marshal(msg)
				return &sock.Event{
					Publisher: "pipecat-manager",
					Type:      string(pmmessage.EventTypeBotLLM),
					Data:      json.RawMessage(data),
				}
			}(),
			setupMock: func(m *messagehandler.MockMessageHandler) {
				m.EXPECT().EventPMMessageBotLLM(gomock.Any(), gomock.Any()).Times(1)
			},
			wantError: false,
		},
		{
			name: "handles_invalid_json_data",
			event: &sock.Event{
				Publisher: "pipecat-manager",
				Type:      string(pmmessage.EventTypeBotLLM),
				Data:      json.RawMessage([]byte("invalid json")),
			},
			setupMock: func(m *messagehandler.MockMessageHandler) {
				// Should not be called on error
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMessageHandler := messagehandler.NewMockMessageHandler(ctrl)
			tt.setupMock(mockMessageHandler)

			h := &subscribeHandler{
				messageHandler: mockMessageHandler,
			}

			err := h.processEventPMMessageBotLLM(context.Background(), tt.event)
			if (err != nil) != tt.wantError {
				t.Errorf("processEventPMMessageBotLLM() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestProcessEventPMTeamMemberSwitched(t *testing.T) {
	tests := []struct {
		name      string
		event     *sock.Event
		setupMock func(*messagehandler.MockMessageHandler, *aicallhandler.MockAIcallHandler)
		wantError bool
	}{
		{
			name: "processes_team_member_switched_successfully",
			event: func() *sock.Event {
				evt := &pmmessage.MemberSwitchedEvent{
					CustomerID:               uuid.Must(uuid.NewV4()),
					PipecatcallID:            uuid.Must(uuid.NewV4()),
					PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
					PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
					TransitionFunctionName:   "switch_to_sales",
					FromMember: pmmessage.MemberInfo{
						ID:          uuid.Must(uuid.NewV4()),
						Name:        "support-agent",
						EngineModel: "openai.gpt-4o",
					},
					ToMember: pmmessage.MemberInfo{
						ID:          uuid.Must(uuid.NewV4()),
						Name:        "sales-agent",
						EngineModel: "openai.gpt-4o",
					},
				}

				data, _ := json.Marshal(evt)
				return &sock.Event{
					Publisher: "pipecat-manager",
					Type:      string(pmmessage.EventTypeTeamMemberSwitched),
					Data:      json.RawMessage(data),
				}
			}(),
			setupMock: func(m *messagehandler.MockMessageHandler, a *aicallhandler.MockAIcallHandler) {
				m.EXPECT().EventPMTeamMemberSwitched(gomock.Any(), gomock.Any()).Times(1)
				a.EXPECT().UpdateCurrentMemberID(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&aicall.AIcall{}, nil).Times(1)
			},
			wantError: false,
		},
		{
			name: "handles_invalid_json_data",
			event: &sock.Event{
				Publisher: "pipecat-manager",
				Type:      string(pmmessage.EventTypeTeamMemberSwitched),
				Data:      json.RawMessage([]byte("invalid json")),
			},
			setupMock: func(m *messagehandler.MockMessageHandler, a *aicallhandler.MockAIcallHandler) {
				// Should not be called on error
			},
			wantError: true,
		},
		{
			name: "continues_when_update_current_member_fails",
			event: func() *sock.Event {
				evt := &pmmessage.MemberSwitchedEvent{
					CustomerID:               uuid.Must(uuid.NewV4()),
					PipecatcallID:            uuid.Must(uuid.NewV4()),
					PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
					PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
					TransitionFunctionName:   "switch_to_billing",
					FromMember: pmmessage.MemberInfo{
						ID:          uuid.Must(uuid.NewV4()),
						Name:        "support-agent",
						EngineModel: "openai.gpt-4o",
					},
					ToMember: pmmessage.MemberInfo{
						ID:          uuid.Must(uuid.NewV4()),
						Name:        "billing-agent",
						EngineModel: "openai.gpt-4o",
					},
				}

				data, _ := json.Marshal(evt)
				return &sock.Event{
					Publisher: "pipecat-manager",
					Type:      string(pmmessage.EventTypeTeamMemberSwitched),
					Data:      json.RawMessage(data),
				}
			}(),
			setupMock: func(m *messagehandler.MockMessageHandler, a *aicallhandler.MockAIcallHandler) {
				m.EXPECT().EventPMTeamMemberSwitched(gomock.Any(), gomock.Any()).Times(1)
				a.EXPECT().UpdateCurrentMemberID(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil, fmt.Errorf("db error")).Times(1)
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMessageHandler := messagehandler.NewMockMessageHandler(ctrl)
			mockAIcallHandler := aicallhandler.NewMockAIcallHandler(ctrl)
			tt.setupMock(mockMessageHandler, mockAIcallHandler)

			h := &subscribeHandler{
				messageHandler: mockMessageHandler,
				aicallHandler:  mockAIcallHandler,
			}

			err := h.processEventPMTeamMemberSwitched(context.Background(), tt.event)
			if (err != nil) != tt.wantError {
				t.Errorf("processEventPMTeamMemberSwitched() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
