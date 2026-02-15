package subscribehandler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

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
