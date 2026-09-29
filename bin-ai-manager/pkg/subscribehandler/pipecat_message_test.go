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

func Test_processEventPMPipelineError(t *testing.T) {
	tests := []struct {
		name      string
		event     *sock.Event
		expect    *pmmessage.PipelineErrorEvent
		wantError bool
	}{
		{
			name: "normal",
			event: &sock.Event{
				Publisher: "pipecat-manager",
				Type:      pmmessage.EventTypePipelineError,
				Data:      json.RawMessage(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","pipecatcall_id":"9c5c6e64-6289-4ac9-ad88-b20796a6bc96","pipecatcall_reference_type":"ai_call","pipecatcall_reference_id":"443bfb46-fa3a-4dd5-912d-5482d31fed22","category":"authentication","fatal":false}`),
			},
			expect: &pmmessage.PipelineErrorEvent{
				CustomerID:               uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				PipecatcallID:            uuid.FromStringOrNil("9c5c6e64-6289-4ac9-ad88-b20796a6bc96"),
				PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.FromStringOrNil("443bfb46-fa3a-4dd5-912d-5482d31fed22"),
				Category:                 pmmessage.ErrorCategoryAuthentication,
			},
			wantError: false,
		},
		{
			name: "invalid json",
			event: &sock.Event{
				Publisher: "pipecat-manager",
				Type:      pmmessage.EventTypePipelineError,
				Data:      json.RawMessage([]byte("invalid json")),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockMessageHandler := messagehandler.NewMockMessageHandler(mc)
			if tt.expect != nil {
				mockMessageHandler.EXPECT().EventPMPipelineError(gomock.Any(), tt.expect).Times(1)
			}

			h := &subscribeHandler{
				messageHandler: mockMessageHandler,
			}

			err := h.processEventPMPipelineError(context.Background(), tt.event)
			if (err != nil) != tt.wantError {
				t.Errorf("processEventPMPipelineError() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// The pipeline_error event must be routed by processEvent's dispatch switch, not only handled
// when processEventPMPipelineError is called directly (VOIP-1542).
func Test_processEvent_PMPipelineError_dispatch(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockMessageHandler := messagehandler.NewMockMessageHandler(mc)
	h := &subscribeHandler{
		messageHandler: mockMessageHandler,
	}

	expect := &pmmessage.PipelineErrorEvent{
		PipecatcallID:            uuid.FromStringOrNil("9c5c6e64-6289-4ac9-ad88-b20796a6bc96"),
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   uuid.FromStringOrNil("443bfb46-fa3a-4dd5-912d-5482d31fed22"),
		Category:                 pmmessage.ErrorCategoryTimeout,
	}
	mockMessageHandler.EXPECT().EventPMPipelineError(gomock.Any(), expect).Times(1)

	h.processEvent(&sock.Event{
		Publisher: "pipecat-manager",
		Type:      pmmessage.EventTypePipelineError,
		DataType:  "application/json",
		Data:      []byte(`{"pipecatcall_id":"9c5c6e64-6289-4ac9-ad88-b20796a6bc96","pipecatcall_reference_type":"ai_call","pipecatcall_reference_id":"443bfb46-fa3a-4dd5-912d-5482d31fed22","category":"timeout","fatal":false}`),
	})
}
