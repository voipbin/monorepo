package subscribehandler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/internal/logtest"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	"monorepo/bin-common-handler/models/sock"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

const logPolicySentinel = "SENTINEL-CONVERSATION-TEXT-9f3c"

// The event line must not carry the event data. The pipecat message events carry what the user said
// and what the bot answered.
func Test_logPolicy_pipecatMessageEvents(t *testing.T) {
	msg := &pmmessage.Message{
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
		Text:                     logPolicySentinel,
	}
	data, _ := json.Marshal(msg)

	tests := []struct {
		name string
		typ  string
		run  func(h *subscribeHandler, m *sock.Event) error
		mock func(m *messagehandler.MockMessageHandler)
	}{
		{
			name: "user transcription",
			typ:  pmmessage.EventTypeUserTranscription,
			run: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventPMMessageUserTranscription(context.Background(), m)
			},
			mock: func(m *messagehandler.MockMessageHandler) {
				m.EXPECT().EventPMMessageUserTranscription(gomock.Any(), gomock.Any())
			},
		},
		{
			name: "bot llm",
			typ:  pmmessage.EventTypeBotLLM,
			run: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventPMMessageBotLLM(context.Background(), m)
			},
			mock: func(m *messagehandler.MockMessageHandler) {
				m.EXPECT().EventPMMessageBotLLM(gomock.Any(), gomock.Any())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			logs := logtest.New(t)

			mockMessage := messagehandler.NewMockMessageHandler(mc)
			tt.mock(mockMessage)
			h := &subscribeHandler{messageHandler: mockMessage}

			evt := &sock.Event{Publisher: "pipecat-manager", Type: tt.typ, Data: json.RawMessage(data)}
			if err := tt.run(h, evt); err != nil {
				t.Fatalf("process: %v", err)
			}

			logs.AssertAbsent(t, logPolicySentinel)
			logs.AssertField(t, "data_len")
		})
	}
}
