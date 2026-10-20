package messagehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/internal/logtest"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

const logPolicySentinel = "SENTINEL-CONVERSATION-TEXT-9f3c"

// The lines of the message events must not carry what the user said or what the bot answered, neither
// in the event nor in the created message, and must keep the ids.
func Test_logPolicy_pipecatMessageEvents(t *testing.T) {
	aicallID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name string
		run  func(h *messageHandler)
	}{
		{
			name: "user transcription",
			run: func(h *messageHandler) {
				h.EventPMMessageUserTranscription(context.Background(), &pmmessage.Message{
					PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
					PipecatcallReferenceID:   aicallID,
					Text:                     logPolicySentinel,
				})
			},
		},
		{
			name: "bot llm",
			run: func(h *messageHandler) {
				h.EventPMMessageBotLLM(context.Background(), &pmmessage.Message{
					PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
					PipecatcallReferenceID:   aicallID,
					Text:                     logPolicySentinel,
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			logs := logtest.New(t)

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			// a voice aicall: the bot message is persisted and not delivered to a conversation
			mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(&aicall.AIcall{ReferenceType: aicall.ReferenceTypeCall}, nil).AnyTimes()

			created := &message.Message{Content: logPolicySentinel}
			created.ID = uuid.Must(uuid.NewV4())
			mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(created, nil).AnyTimes()
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
				utilHandler:   utilhandler.NewUtilHandler(),
			}
			tt.run(h)

			logs.AssertAbsent(t, logPolicySentinel)
			logs.AssertField(t, "message_id")
		})
	}
}
