package messagehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

func TestEventPMMessageUserTranscription(t *testing.T) {
	tests := []struct {
		name      string
		event     *pmmessage.Message
		setupMock func(*dbhandler.MockDBHandler)
	}{
		{
			name: "creates_message_for_aicall_reference",
			event: &pmmessage.Message{
				PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
				Text:                     "User transcription text",
			},
			setupMock: func(m *dbhandler.MockDBHandler) {
				testMsg := &message.Message{}
				testMsg.ID = uuid.Must(uuid.NewV4())
				m.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(testMsg, nil).Times(1)
			},
		},
		{
			name: "ignores_non_aicall_reference",
			event: &pmmessage.Message{
				PipecatcallReferenceType: pmpipecatcall.ReferenceTypeCall,
				PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
				Text:                     "Should be ignored",
			},
			setupMock: func(m *dbhandler.MockDBHandler) {
				// Should not create message for non-aicall reference
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := dbhandler.NewMockDBHandler(ctrl)
			mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			tt.setupMock(mockDB)

			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   utilhandler.NewUtilHandler(),
			}

			h.EventPMMessageUserTranscription(context.Background(), tt.event)
			// This function doesn't return error, just logs
		})
	}
}

func TestEventPMMessageBotLLM(t *testing.T) {
	tests := []struct {
		name      string
		event     *pmmessage.Message
		setupMock func(*dbhandler.MockDBHandler)
	}{
		{
			name: "creates_message_for_bot_llm",
			event: &pmmessage.Message{
				PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
				Text:                     "Bot response text",
			},
			setupMock: func(m *dbhandler.MockDBHandler) {
				testMsg := &message.Message{}
				testMsg.ID = uuid.Must(uuid.NewV4())
				m.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(testMsg, nil).Times(1)
			},
		},
		{
			name: "ignores_empty_text",
			event: &pmmessage.Message{
				PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
				Text:                     "",
			},
			setupMock: func(m *dbhandler.MockDBHandler) {
				// Should not create message for empty text
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := dbhandler.NewMockDBHandler(ctrl)
			mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			tt.setupMock(mockDB)

			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   utilhandler.NewUtilHandler(),
			}

			h.EventPMMessageBotLLM(context.Background(), tt.event)
		})
	}
}

func TestEventPMMessageUserLLM(t *testing.T) {
	tests := []struct {
		name      string
		event     *pmmessage.Message
		setupMock func(*dbhandler.MockDBHandler)
	}{
		{
			name: "creates_message_for_user_llm",
			event: &pmmessage.Message{
				PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
				Text:                     "User LLM text",
			},
			setupMock: func(m *dbhandler.MockDBHandler) {
				testMsg := &message.Message{}
				testMsg.ID = uuid.Must(uuid.NewV4())
				m.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(testMsg, nil).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := dbhandler.NewMockDBHandler(ctrl)
			mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			tt.setupMock(mockDB)

			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   utilhandler.NewUtilHandler(),
			}

			h.EventPMMessageUserLLM(context.Background(), tt.event)
		})
	}
}
