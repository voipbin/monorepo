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
				ActiveflowID:             uuid.Must(uuid.NewV4()),
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
				ActiveflowID:             uuid.Must(uuid.NewV4()),
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
				ActiveflowID:             uuid.Must(uuid.NewV4()),
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

func TestEventPMMessageBotLLM_forwards_pre_generated_id(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	preGeneratedID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.Message{
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   referenceID,
		ActiveflowID:             activeflowID,
		Text:                     "Bot response",
	}
	evt.ID = preGeneratedID
	evt.CustomerID = customerID

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)

	// Verify MessageCreate receives the pre-generated ID (not a new one).
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *message.Message) error {
			if m.ID != preGeneratedID {
				t.Errorf("expected pre-generated ID %s passed to MessageCreate, got %s", preGeneratedID, m.ID)
			}
			return nil
		},
	).Times(1)

	returnMsg := &message.Message{}
	returnMsg.ID = preGeneratedID
	returnMsg.CustomerID = customerID
	mockDB.EXPECT().MessageGet(gomock.Any(), preGeneratedID).Return(returnMsg, nil).Times(1)

	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	h.EventPMMessageBotLLM(context.Background(), evt)
}

func TestEventPMMessageBotLLMIntermediate(t *testing.T) {
	tests := []struct {
		name          string
		event         *pmmessage.Message
		expectPublish bool
	}{
		{
			name: "publishes_intermediate_webhook_for_aicall",
			event: &pmmessage.Message{
				PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
				ActiveflowID:             uuid.Must(uuid.NewV4()),
				Text:                     "Hello world",
				Sequence:                 1,
			},
			expectPublish: true,
		},
		{
			name: "ignores_empty_text",
			event: &pmmessage.Message{
				PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
				Text:                     "",
			},
			expectPublish: false,
		},
		{
			name: "ignores_non_aicall_reference",
			event: &pmmessage.Message{
				PipecatcallReferenceType: pmpipecatcall.ReferenceTypeCall,
				PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
				Text:                     "Should be ignored",
			},
			expectPublish: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := dbhandler.NewMockDBHandler(ctrl)
			mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)

			if tt.expectPublish {
				mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.event.CustomerID, message.EventTypeMessageIntermediate, gomock.Any()).Times(1)
			}

			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   utilhandler.NewUtilHandler(),
			}

			h.EventPMMessageBotLLMIntermediate(context.Background(), tt.event)
		})
	}
}

func TestEventPMMessageBotLLMIntermediate_field_mapping(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	msgID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.Message{
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   referenceID,
		ActiveflowID:             activeflowID,
		Text:                     "Hello world",
		Sequence:                 3,
	}
	evt.ID = msgID
	evt.CustomerID = customerID

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)

	// Capture the webhook message and assert all fields.
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, message.EventTypeMessageIntermediate, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, _ string, wm *message.IntermediateWebhookMessage) {
			if wm.ID != msgID {
				t.Errorf("expected ID %s, got %s", msgID, wm.ID)
			}
			if wm.CustomerID != customerID {
				t.Errorf("expected CustomerID %s, got %s", customerID, wm.CustomerID)
			}
			if wm.AIcallID != referenceID {
				t.Errorf("expected AIcallID %s, got %s", referenceID, wm.AIcallID)
			}
			if wm.ActiveflowID != activeflowID {
				t.Errorf("expected ActiveflowID %s, got %s", activeflowID, wm.ActiveflowID)
			}
			if wm.Role != message.RoleAssistant {
				t.Errorf("expected Role %s, got %s", message.RoleAssistant, wm.Role)
			}
			if wm.Content != "Hello world" {
				t.Errorf("expected Content 'Hello world', got %q", wm.Content)
			}
			if wm.Direction != message.DirectionIncoming {
				t.Errorf("expected Direction %s, got %s", message.DirectionIncoming, wm.Direction)
			}
			if wm.Sequence != 3 {
				t.Errorf("expected Sequence 3, got %d", wm.Sequence)
			}
		},
	).Times(1)

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	h.EventPMMessageBotLLMIntermediate(context.Background(), evt)
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
				ActiveflowID:             uuid.Must(uuid.NewV4()),
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
