package messagehandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"
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

func TestEventPMMessageBotLLM_voice_path(t *testing.T) {
	// Voice / non-conversation path: persists the message and returns; no AIcall
	// fetch beyond the single one used to determine reference type, no delivery.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pipecatcallID := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.Message{
		PipecatcallID:            pipecatcallID,
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   aicallID,
		ActiveflowID:             activeflowID,
		Text:                     "Bot voice response",
	}
	evt.CustomerID = customerID

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)

	// AIcall is a voice (call) reference type — no conversation delivery.
	voiceAIcall := &aicall.AIcall{
		PipecatcallID: pipecatcallID,
		ReferenceType: aicall.ReferenceTypeCall,
	}
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(voiceAIcall, nil).Times(1)

	testMsg := &message.Message{}
	testMsg.ID = uuid.Must(uuid.NewV4())
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(testMsg, nil).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	// No ConversationV1MessageSend expected.

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	h.EventPMMessageBotLLM(context.Background(), evt)
}

func TestEventPMMessageBotLLM_non_aicall_reference(t *testing.T) {
	// Pipecat reference type that isn't AICall (e.g., direct call): legacy path,
	// persist and return — no AIV1AIcallGet, no delivery.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	evt := &pmmessage.Message{
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeCall,
		PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
		ActiveflowID:             uuid.Must(uuid.NewV4()),
		Text:                     "Bot response",
	}

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)

	// No AIV1AIcallGet expected.

	testMsg := &message.Message{}
	testMsg.ID = uuid.Must(uuid.NewV4())
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(testMsg, nil).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	h.EventPMMessageBotLLM(context.Background(), evt)
}

func TestEventPMMessageBotLLM_empty_text(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	evt := &pmmessage.Message{
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   uuid.Must(uuid.NewV4()),
		ActiveflowID:             uuid.Must(uuid.NewV4()),
		Text:                     "",
	}

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)

	// No expectations: empty text returns immediately.

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	h.EventPMMessageBotLLM(context.Background(), evt)
}

func TestEventPMMessageBotLLM_conversation_guard_primary_drop(t *testing.T) {
	// Guard #1 miss: AIcall.PipecatcallID != evt.PipecatcallID.
	// Expected: NO persistence, NO send, primary stale counter += 1.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pccA := uuid.Must(uuid.NewV4())
	pccB := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	convID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.Message{
		PipecatcallID:            pccA,
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   aicallID,
		ActiveflowID:             uuid.Must(uuid.NewV4()),
		Text:                     "stale assistant reply",
	}

	staleAIcall := &aicall.AIcall{
		PipecatcallID: pccB, // mismatch
		ReferenceType: aicall.ReferenceTypeConversation,
		ReferenceID:   convID,
	}

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)

	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(staleAIcall, nil).Times(1)
	// No MessageCreate, no second AIV1AIcallGet, no ConversationV1MessageSend.

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	before := testutil.ToFloat64(promConversationStaleResponseDroppedTotal.WithLabelValues("primary"))
	h.EventPMMessageBotLLM(context.Background(), evt)
	after := testutil.ToFloat64(promConversationStaleResponseDroppedTotal.WithLabelValues("primary"))

	if after-before != 1 {
		t.Errorf("expected primary stale counter to increment by 1, got delta=%f", after-before)
	}
}

func TestEventPMMessageBotLLM_conversation_guard_secondary_drop(t *testing.T) {
	// Guard #1 passes, guard #2 miss after persistence (race).
	// Expected: persistence happens, NO send, secondary stale counter += 1.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pccA := uuid.Must(uuid.NewV4())
	pccB := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	convID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.Message{
		PipecatcallID:            pccA,
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   aicallID,
		ActiveflowID:             uuid.Must(uuid.NewV4()),
		Text:                     "assistant reply",
	}

	freshAIcall := &aicall.AIcall{
		PipecatcallID: pccA,
		ReferenceType: aicall.ReferenceTypeConversation,
		ReferenceID:   convID,
	}
	rotatedAIcall := &aicall.AIcall{
		PipecatcallID: pccB, // mismatch on re-check
		ReferenceType: aicall.ReferenceTypeConversation,
		ReferenceID:   convID,
	}

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)

	gomock.InOrder(
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(freshAIcall, nil).Times(1),
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(rotatedAIcall, nil).Times(1),
	)

	testMsg := &message.Message{}
	testMsg.ID = uuid.Must(uuid.NewV4())
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(testMsg, nil).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	// No ConversationV1MessageSend.

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	before := testutil.ToFloat64(promConversationStaleResponseDroppedTotal.WithLabelValues("secondary"))
	h.EventPMMessageBotLLM(context.Background(), evt)
	after := testutil.ToFloat64(promConversationStaleResponseDroppedTotal.WithLabelValues("secondary"))

	if after-before != 1 {
		t.Errorf("expected secondary stale counter to increment by 1, got delta=%f", after-before)
	}
}

func TestEventPMMessageBotLLM_conversation_send_success(t *testing.T) {
	// Both guards pass; ConversationV1MessageSend called with conv ID and text.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pccA := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	convID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.Message{
		PipecatcallID:            pccA,
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   aicallID,
		ActiveflowID:             uuid.Must(uuid.NewV4()),
		Text:                     "hello human",
	}

	freshAIcall := &aicall.AIcall{
		PipecatcallID: pccA,
		ReferenceType: aicall.ReferenceTypeConversation,
		ReferenceID:   convID,
	}

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)

	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(freshAIcall, nil).Times(2)

	testMsg := &message.Message{}
	testMsg.ID = uuid.Must(uuid.NewV4())
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(testMsg, nil).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockReq.EXPECT().
		ConversationV1MessageSend(gomock.Any(), convID, "hello human", []cvmedia.Media{}).
		Return(&cvmessage.Message{}, nil).
		Times(1)

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	before := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("success"))
	h.EventPMMessageBotLLM(context.Background(), evt)
	after := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("success"))

	if after-before != 1 {
		t.Errorf("expected success counter to increment by 1, got delta=%f", after-before)
	}
}

func TestEventPMMessageBotLLM_conversation_send_failure_silent(t *testing.T) {
	// Both guards pass; ConversationV1MessageSend errors. No panic, no retry.
	// failure counter += 1.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pccA := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	convID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.Message{
		PipecatcallID:            pccA,
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   aicallID,
		ActiveflowID:             uuid.Must(uuid.NewV4()),
		Text:                     "hello human",
	}

	freshAIcall := &aicall.AIcall{
		PipecatcallID: pccA,
		ReferenceType: aicall.ReferenceTypeConversation,
		ReferenceID:   convID,
	}

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)

	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(freshAIcall, nil).Times(2)

	testMsg := &message.Message{}
	testMsg.ID = uuid.Must(uuid.NewV4())
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(testMsg, nil).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockReq.EXPECT().
		ConversationV1MessageSend(gomock.Any(), convID, "hello human", []cvmedia.Media{}).
		Return(nil, errors.New("delivery failed")).
		Times(1)

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	before := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("failure"))
	h.EventPMMessageBotLLM(context.Background(), evt)
	after := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("failure"))

	if after-before != 1 {
		t.Errorf("expected failure counter to increment by 1, got delta=%f", after-before)
	}
}

// TestEventPMMessageBotLLM_customer_id_mismatch_proceeds documents that the
// handler does NOT cross-check evt.CustomerID against ac.CustomerID at
// runtime — see design doc §11 (Accepted v1 limits): "Customer cross-check
// between AIcall and conversation skipped at the RPC level. Already enforced
// upstream; one-time integration test asserts isolation."
//
// This test is a safety net: if a future PR adds a cross-check, this test
// fails and forces conscious review of the design constraint.
func TestEventPMMessageBotLLM_customer_id_mismatch_proceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customerA := uuid.Must(uuid.NewV4())
	customerB := uuid.Must(uuid.NewV4())
	pccA := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	convID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.Message{
		PipecatcallID:            pccA,
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   aicallID,
		ActiveflowID:             uuid.Must(uuid.NewV4()),
		Text:                     "hello human",
	}
	evt.CustomerID = customerA

	// AIcall belongs to a DIFFERENT customer than the inbound event.
	freshAIcall := &aicall.AIcall{
		PipecatcallID: pccA,
		ReferenceType: aicall.ReferenceTypeConversation,
		ReferenceID:   convID,
	}
	freshAIcall.CustomerID = customerB

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)

	// Both guard fetches return the customer-mismatched AIcall — handler
	// MUST still proceed (no cross-check at this layer).
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(freshAIcall, nil).Times(2)

	// MessageCreate is invoked with evt.CustomerID (customerA), per current code.
	testMsg := &message.Message{}
	testMsg.ID = uuid.Must(uuid.NewV4())
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *message.Message) error {
			if m.CustomerID != customerA {
				t.Errorf("expected MessageCreate to use evt.CustomerID=%s, got %s", customerA, m.CustomerID)
			}
			return nil
		},
	).Times(1)
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(testMsg, nil).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	// Conversation send proceeds — uses acFinal.ReferenceID (the AIcall's
	// reference, not the event's). No customer-isolation guard at runtime.
	mockReq.EXPECT().
		ConversationV1MessageSend(gomock.Any(), convID, "hello human", []cvmedia.Media{}).
		Return(&cvmessage.Message{}, nil).
		Times(1)

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	before := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("success"))
	h.EventPMMessageBotLLM(context.Background(), evt)
	after := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("success"))

	if after-before != 1 {
		t.Errorf("expected success counter to increment by 1 (handler proceeds despite customer mismatch), got delta=%f", after-before)
	}
}

func TestEventPMMessageBotLLM_forwards_pre_generated_id(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	preGeneratedID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	pipecatcallID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.Message{
		PipecatcallID:            pipecatcallID,
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   referenceID,
		ActiveflowID:             activeflowID,
		Text:                     "Bot response",
	}
	evt.ID = preGeneratedID
	evt.CustomerID = customerID

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)

	// Voice path — single AIcall fetch returning ReferenceTypeCall.
	voiceAIcall := &aicall.AIcall{
		PipecatcallID: pipecatcallID,
		ReferenceType: aicall.ReferenceTypeCall,
	}
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(voiceAIcall, nil).Times(1)

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
		reqHandler:    mockReq,
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
