package messagehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/participanthandler"
	"monorepo/bin-common-handler/models/identity"
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

	// Post-send: mark delivered (succeeds first try).
	mockDB.EXPECT().MessageUpdateDeliveryStatus(gomock.Any(), gomock.Any(), message.DeliveryStatusDelivered).Return(nil).Times(1)

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

	// Post-send: mark delivered (succeeds first try).
	mockDB.EXPECT().MessageUpdateDeliveryStatus(gomock.Any(), gomock.Any(), message.DeliveryStatusDelivered).Return(nil).Times(1)

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

func TestEventPMMessageBotLLM_first_aicall_get_error(t *testing.T) {
	// First AIV1AIcallGet returns an error — handler logs and returns
	// without persisting the message or sending to conversation. Stale
	// counters and reply-send counters MUST NOT increment.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pccA := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.Message{
		PipecatcallID:            pccA,
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   aicallID,
		ActiveflowID:             uuid.Must(uuid.NewV4()),
		Text:                     "should not be persisted",
	}

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)

	// First AIV1AIcallGet errors. NO MessageCreate, NO MessageGet,
	// NO ConversationV1MessageSend, NO PublishWebhookEvent.
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(nil, errors.New("aicall rpc error")).Times(1)

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	beforePrimary := testutil.ToFloat64(promConversationStaleResponseDroppedTotal.WithLabelValues("primary"))
	beforeSecondary := testutil.ToFloat64(promConversationStaleResponseDroppedTotal.WithLabelValues("secondary"))
	beforeSuccess := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("success"))
	beforeFailure := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("failure"))

	h.EventPMMessageBotLLM(context.Background(), evt)

	if got := testutil.ToFloat64(promConversationStaleResponseDroppedTotal.WithLabelValues("primary")); got != beforePrimary {
		t.Errorf("primary stale counter changed unexpectedly: before=%f after=%f", beforePrimary, got)
	}
	if got := testutil.ToFloat64(promConversationStaleResponseDroppedTotal.WithLabelValues("secondary")); got != beforeSecondary {
		t.Errorf("secondary stale counter changed unexpectedly: before=%f after=%f", beforeSecondary, got)
	}
	if got := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("success")); got != beforeSuccess {
		t.Errorf("reply-send success counter changed unexpectedly: before=%f after=%f", beforeSuccess, got)
	}
	if got := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("failure")); got != beforeFailure {
		t.Errorf("reply-send failure counter changed unexpectedly: before=%f after=%f", beforeFailure, got)
	}
}

func TestEventPMMessageBotLLM_second_aicall_get_error(t *testing.T) {
	// First AIV1AIcallGet succeeds (matching PCC, conversation reference);
	// messageHandler.Create persists the assistant message; second
	// AIV1AIcallGet (the secondary guard re-check) returns an error.
	// Expected: message IS persisted; ConversationV1MessageSend MUST NOT
	// be called; no stale or reply-send counters increment.
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
		Text:                     "assistant reply persisted but not delivered",
	}

	freshAIcall := &aicall.AIcall{
		PipecatcallID: pccA,
		ReferenceType: aicall.ReferenceTypeConversation,
		ReferenceID:   convID,
	}

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)

	gomock.InOrder(
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(freshAIcall, nil).Times(1),
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(nil, errors.New("rpc transient")).Times(1),
	)

	// Persistence runs once between the two AIcall fetches.
	testMsg := &message.Message{}
	testMsg.ID = uuid.Must(uuid.NewV4())
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(testMsg, nil).Times(1)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	// ConversationV1MessageSend MUST NOT be called.

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	beforePrimary := testutil.ToFloat64(promConversationStaleResponseDroppedTotal.WithLabelValues("primary"))
	beforeSecondary := testutil.ToFloat64(promConversationStaleResponseDroppedTotal.WithLabelValues("secondary"))
	beforeSuccess := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("success"))
	beforeFailure := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("failure"))

	h.EventPMMessageBotLLM(context.Background(), evt)

	if got := testutil.ToFloat64(promConversationStaleResponseDroppedTotal.WithLabelValues("primary")); got != beforePrimary {
		t.Errorf("primary stale counter changed unexpectedly: before=%f after=%f", beforePrimary, got)
	}
	if got := testutil.ToFloat64(promConversationStaleResponseDroppedTotal.WithLabelValues("secondary")); got != beforeSecondary {
		t.Errorf("secondary stale counter changed unexpectedly: before=%f after=%f", beforeSecondary, got)
	}
	if got := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("success")); got != beforeSuccess {
		t.Errorf("reply-send success counter changed unexpectedly: before=%f after=%f", beforeSuccess, got)
	}
	if got := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("failure")); got != beforeFailure {
		t.Errorf("reply-send failure counter changed unexpectedly: before=%f after=%f", beforeFailure, got)
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
			// h has no reqHandler, so resolveActiveAIID short-circuits to uuid.Nil.
			if wm.ActiveAIID != uuid.Nil {
				t.Errorf("expected ActiveAIID uuid.Nil (no reqHandler), got %v", wm.ActiveAIID)
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

// TestEventPMMessageBotLLM_conversation drives the AIcall+conversation branch
// through the four/five outcomes: happy path, guard #2 fail, send fail,
// update fails-then-retry-OK, update fails both attempts. In every case the row
// MUST be persisted with delivery_status='pending'. The number of
// MessageUpdateDeliveryStatus and ConversationV1MessageSend calls varies.
func TestEventPMMessageBotLLM_conversation(t *testing.T) {
	cases := []struct {
		name              string
		guard2Pass        bool
		sendOK            bool
		updateOK          bool
		updateRetryOK     bool
		wantUpdateCalls   int
		wantConvSendCalls int
		wantSuccessDelta  float64
		wantFailureDelta  float64
		wantUpdateFailed  float64
	}{
		{"happy", true, true, true, false, 1, 1, 1, 0, 0},
		{"guard2_fail", false, false, false, false, 0, 0, 0, 0, 0},
		{"send_fail", true, false, false, false, 0, 1, 0, 1, 0},
		{"update_fail_but_retry_ok", true, true, false, true, 2, 1, 1, 0, 0},
		{"update_fail_both", true, true, false, false, 2, 1, 1, 0, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

			mockDB := dbhandler.NewMockDBHandler(ctrl)
			mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
			mockReq := requesthandler.NewMockRequestHandler(ctrl)

			// Guard #1 always passes for these scenarios; guard #2 is varied via
			// the second AIcallGet response.
			fresh := &aicall.AIcall{
				PipecatcallID: pccA,
				ReferenceType: aicall.ReferenceTypeConversation,
				ReferenceID:   convID,
			}
			second := fresh
			if !tc.guard2Pass {
				second = &aicall.AIcall{
					PipecatcallID: pccB, // mismatch on re-check
					ReferenceType: aicall.ReferenceTypeConversation,
					ReferenceID:   convID,
				}
			}
			gomock.InOrder(
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(fresh, nil).Times(1),
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(second, nil).Times(1),
			)

			// Persistence must always happen (after guard #1) with the pending
			// status and the event's PipecatcallID stamped on the row.
			testMsg := &message.Message{}
			testMsg.ID = uuid.Must(uuid.NewV4())
			mockDB.EXPECT().
				MessageCreate(gomock.Any(), messageWithPCC(pccA, message.DeliveryStatusPending)).
				Return(nil).
				Times(1)
			mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(testMsg, nil).Times(1)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			// ConversationV1MessageSend expectations.
			if tc.wantConvSendCalls > 0 {
				if tc.sendOK {
					mockReq.EXPECT().
						ConversationV1MessageSend(gomock.Any(), convID, "assistant reply", []cvmedia.Media{}).
						Return(&cvmessage.Message{}, nil).
						Times(tc.wantConvSendCalls)
				} else {
					mockReq.EXPECT().
						ConversationV1MessageSend(gomock.Any(), convID, "assistant reply", []cvmedia.Media{}).
						Return(nil, errors.New("delivery failed")).
						Times(tc.wantConvSendCalls)
				}
			}

			// MessageUpdateDeliveryStatus expectations.
			switch tc.wantUpdateCalls {
			case 0:
				// no expectation
			case 1:
				mockDB.EXPECT().
					MessageUpdateDeliveryStatus(gomock.Any(), testMsg.ID, message.DeliveryStatusDelivered).
					Return(nil).
					Times(1)
			case 2:
				// First call fails; second call's outcome depends on tc.updateRetryOK.
				retryErr := error(nil)
				if !tc.updateRetryOK {
					retryErr = errors.New("update retry failed")
				}
				gomock.InOrder(
					mockDB.EXPECT().
						MessageUpdateDeliveryStatus(gomock.Any(), testMsg.ID, message.DeliveryStatusDelivered).
						Return(errors.New("update failed")).
						Times(1),
					mockDB.EXPECT().
						MessageUpdateDeliveryStatus(gomock.Any(), testMsg.ID, message.DeliveryStatusDelivered).
						Return(retryErr).
						Times(1),
				)
			default:
				t.Fatalf("unsupported wantUpdateCalls=%d", tc.wantUpdateCalls)
			}

			// Patch the retry sleep so the test does not pay 100ms wall-clock
			// per retry case.
			origSleep := deliveryStatusUpdateSleep
			sleepCalled := 0
			deliveryStatusUpdateSleep = func(time.Duration) { sleepCalled++ }
			t.Cleanup(func() { deliveryStatusUpdateSleep = origSleep })

			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
				utilHandler:   utilhandler.NewUtilHandler(),
			}

			beforeSuccess := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("success"))
			beforeFailure := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("failure"))
			beforeUpdFailed := testutil.ToFloat64(promConversationDeliveryStatusUpdateFailedTotal)

			h.EventPMMessageBotLLM(context.Background(), evt)

			afterSuccess := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("success"))
			afterFailure := testutil.ToFloat64(promConversationReplySendTotal.WithLabelValues("failure"))
			afterUpdFailed := testutil.ToFloat64(promConversationDeliveryStatusUpdateFailedTotal)

			if got := afterSuccess - beforeSuccess; got != tc.wantSuccessDelta {
				t.Errorf("success counter delta: want %v got %v", tc.wantSuccessDelta, got)
			}
			if got := afterFailure - beforeFailure; got != tc.wantFailureDelta {
				t.Errorf("failure counter delta: want %v got %v", tc.wantFailureDelta, got)
			}
			if got := afterUpdFailed - beforeUpdFailed; got != tc.wantUpdateFailed {
				t.Errorf("delivery-status-update-failed counter delta: want %v got %v", tc.wantUpdateFailed, got)
			}

			// On the retry case, the sleep must be invoked exactly once between
			// the first and second update attempts.
			wantSleep := 0
			if tc.wantUpdateCalls == 2 {
				wantSleep = 1
			}
			if sleepCalled != wantSleep {
				t.Errorf("deliveryStatusUpdateSleep call count: want %d got %d", wantSleep, sleepCalled)
			}
		})
	}
}

// TestEventPMPipecatcallTerminated drives the backstop handler through every
// result-label outcome. Each case asserts the corresponding Prometheus counter
// increments by exactly 1 and that ConversationV1MessageSend is called only on
// the paths that should actually deliver.
func TestEventPMPipecatcallTerminated(t *testing.T) {
	cases := []struct {
		name            string
		evRefType       pmpipecatcall.ReferenceType
		aicallRefType   aicall.ReferenceType
		aicallStatus    aicall.Status
		aicallGetErr    bool
		replyExistsCall bool   // whether MessageAssistantReplyExists is expected
		replyExistsErr  bool   // whether the reply-exists call returns an error
		replyExists     bool   // value returned by MessageAssistantReplyExists
		messageCreateOK bool   // whether MessageCreate returns nil (only when reached)
		sendOK          bool   // whether ConversationV1MessageSend returns nil (only when reached)
		wantResultLabel string // label whose counter must increment by 1
		wantNoLabel     bool   // when true, no counter must change (RPC error short-circuit)
		wantSendCalls   int
		wantErr         bool
	}{
		{
			name:            "skipped_not_aicall",
			evRefType:       pmpipecatcall.ReferenceTypeCall,
			wantResultLabel: "skipped_not_aicall",
		},
		{
			name:            "aicall_get_error_no_label",
			evRefType:       pmpipecatcall.ReferenceTypeAICall,
			aicallGetErr:    true,
			wantNoLabel:     true,
			wantResultLabel: "skipped_voice", // unused (wantNoLabel=true)
		},
		{
			name:            "skipped_voice",
			evRefType:       pmpipecatcall.ReferenceTypeAICall,
			aicallRefType:   aicall.ReferenceTypeCall,
			aicallStatus:    aicall.StatusProgressing,
			wantResultLabel: "skipped_voice",
		},
		{
			name:            "skipped_terminated",
			evRefType:       pmpipecatcall.ReferenceTypeAICall,
			aicallRefType:   aicall.ReferenceTypeConversation,
			aicallStatus:    aicall.StatusTerminated,
			wantResultLabel: "skipped_terminated",
		},
		{
			name:            "skipped_seen",
			evRefType:       pmpipecatcall.ReferenceTypeAICall,
			aicallRefType:   aicall.ReferenceTypeConversation,
			aicallStatus:    aicall.StatusProgressing,
			replyExistsCall: true,
			replyExists:     true,
			wantResultLabel: "skipped_seen",
		},
		{
			name:            "sent",
			evRefType:       pmpipecatcall.ReferenceTypeAICall,
			aicallRefType:   aicall.ReferenceTypeConversation,
			aicallStatus:    aicall.StatusProgressing,
			replyExistsCall: true,
			replyExists:     false,
			messageCreateOK: true,
			sendOK:          true,
			wantResultLabel: "sent",
			wantSendCalls:   1,
		},
		{
			name:            "send_failed",
			evRefType:       pmpipecatcall.ReferenceTypeAICall,
			aicallRefType:   aicall.ReferenceTypeConversation,
			aicallStatus:    aicall.StatusProgressing,
			replyExistsCall: true,
			replyExists:     false,
			messageCreateOK: true,
			sendOK:          false,
			wantResultLabel: "send_failed",
			wantSendCalls:   1,
			wantErr:         true,
		},
		{
			name:            "failed",
			evRefType:       pmpipecatcall.ReferenceTypeAICall,
			aicallRefType:   aicall.ReferenceTypeConversation,
			aicallStatus:    aicall.StatusProgressing,
			replyExistsCall: true,
			replyExists:     false,
			messageCreateOK: false,
			wantResultLabel: "failed",
			wantErr:         true,
		},
		{
			name:            "reply_exists_error_returns",
			evRefType:       pmpipecatcall.ReferenceTypeAICall,
			aicallRefType:   aicall.ReferenceTypeConversation,
			aicallStatus:    aicall.StatusProgressing,
			replyExistsCall: true,
			replyExistsErr:  true,
			wantNoLabel:     true,
			wantResultLabel: "skipped_seen", // unused (wantNoLabel=true)
			wantErr:         true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			pccID := uuid.Must(uuid.NewV4())
			aicallID := uuid.Must(uuid.NewV4())
			convID := uuid.Must(uuid.NewV4())
			customerID := uuid.Must(uuid.NewV4())
			activeflowID := uuid.Must(uuid.NewV4())

			ev := &pmpipecatcall.Pipecatcall{
				ReferenceType: tc.evRefType,
				ReferenceID:   aicallID,
			}
			ev.ID = pccID

			mockDB := dbhandler.NewMockDBHandler(ctrl)
			mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
			mockReq := requesthandler.NewMockRequestHandler(ctrl)

			// AIcallGet expectations: only when ReferenceType is AICall.
			if tc.evRefType == pmpipecatcall.ReferenceTypeAICall {
				if tc.aicallGetErr {
					mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).
						Return(nil, errors.New("rpc transient")).Times(1)
				} else {
					ac := &aicall.AIcall{
						Identity: identity.Identity{
							ID:         aicallID,
							CustomerID: customerID,
						},
						ActiveflowID:  activeflowID,
						ReferenceType: tc.aicallRefType,
						ReferenceID:   convID,
						Status:        tc.aicallStatus,
					}
					mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil).Times(1)
				}
			}

			// MessageAssistantReplyExists expectations.
			if tc.replyExistsCall {
				if tc.replyExistsErr {
					mockDB.EXPECT().MessageAssistantReplyExists(gomock.Any(), pccID).
						Return(false, errors.New("db transient")).Times(1)
				} else {
					mockDB.EXPECT().MessageAssistantReplyExists(gomock.Any(), pccID).
						Return(tc.replyExists, nil).Times(1)
				}
			}

			// MessageCreate / MessageGet only when persistence runs (sent/send_failed/failed).
			if tc.replyExistsCall && !tc.replyExistsErr && !tc.replyExists {
				if tc.messageCreateOK {
					createdMsg := &message.Message{}
					createdMsg.ID = uuid.Must(uuid.NewV4())
					mockDB.EXPECT().
						MessageCreate(gomock.Any(), messageWithPCC(pccID, message.DeliveryStatusDelivered)).
						Return(nil).Times(1)
					mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(createdMsg, nil).Times(1)
					mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
				} else {
					mockDB.EXPECT().
						MessageCreate(gomock.Any(), messageWithPCC(pccID, message.DeliveryStatusDelivered)).
						Return(errors.New("db down")).Times(1)
				}
			}

			// ConversationV1MessageSend only when persist succeeded.
			if tc.wantSendCalls > 0 {
				if tc.sendOK {
					mockReq.EXPECT().
						ConversationV1MessageSend(gomock.Any(), convID, backstopReplyText, []cvmedia.Media{}).
						Return(&cvmessage.Message{}, nil).Times(tc.wantSendCalls)
				} else {
					mockReq.EXPECT().
						ConversationV1MessageSend(gomock.Any(), convID, backstopReplyText, []cvmedia.Media{}).
						Return(nil, errors.New("conversation send failed")).Times(tc.wantSendCalls)
				}
			}

			// Patch the grace sleep so tests do not pay 3s of wall-clock per case.
			origSleep := backstopGraceSleep
			sleepCalled := 0
			backstopGraceSleep = func(time.Duration) { sleepCalled++ }
			t.Cleanup(func() { backstopGraceSleep = origSleep })

			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
				utilHandler:   utilhandler.NewUtilHandler(),
			}

			// Snapshot all 7 result-label counters so we can assert deltas precisely.
			labels := []string{
				"sent", "failed", "send_failed",
				"skipped_seen", "skipped_voice",
				"skipped_terminated", "skipped_not_aicall",
			}
			before := make(map[string]float64, len(labels))
			for _, l := range labels {
				before[l] = testutil.ToFloat64(promBackstopReplyTotal.WithLabelValues(l))
			}

			err := h.EventPMPipecatcallTerminated(context.Background(), ev)

			if tc.wantErr && err == nil {
				t.Errorf("expected non-nil error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected nil error, got %v", err)
			}

			for _, l := range labels {
				after := testutil.ToFloat64(promBackstopReplyTotal.WithLabelValues(l))
				delta := after - before[l]

				switch {
				case tc.wantNoLabel:
					if delta != 0 {
						t.Errorf("label %q: no counter should change (wantNoLabel), got delta=%v", l, delta)
					}
				case l == tc.wantResultLabel:
					if delta != 1 {
						t.Errorf("label %q: expected delta=1, got %v", l, delta)
					}
				default:
					if delta != 0 {
						t.Errorf("label %q: expected delta=0, got %v", l, delta)
					}
				}
			}

			// Sleep must run iff the backstop reached the grace window: only
			// when the AIcall was fetched, was a conversation reference, and
			// was not already terminated.
			wantSleep := 0
			if tc.evRefType == pmpipecatcall.ReferenceTypeAICall &&
				!tc.aicallGetErr &&
				tc.aicallRefType == aicall.ReferenceTypeConversation &&
				tc.aicallStatus != aicall.StatusTerminated {
				wantSleep = 1
			}
			if sleepCalled != wantSleep {
				t.Errorf("backstopGraceSleep call count: want %d got %d", wantSleep, sleepCalled)
			}
		})
	}
}

func TestResolveActiveAIIDFromAIcall_AI(t *testing.T) {
	aiID := uuid.Must(uuid.NewV4())
	ac := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   aiID,
	}
	h := &messageHandler{}
	got := h.resolveActiveAIIDFromAIcall(context.Background(), ac)
	if got != aiID {
		t.Fatalf("expected %v, got %v", aiID, got)
	}
}

func TestResolveActiveAIIDFromAIcall_Team(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	teamID := uuid.Must(uuid.NewV4())
	memberID := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())

	ac := &aicall.AIcall{
		AssistanceType:  aicall.AssistanceTypeTeam,
		AssistanceID:    teamID,
		CurrentMemberID: memberID,
	}

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
		Members: []team.Member{{ID: memberID, AIID: aiID}},
	}, nil)

	h := &messageHandler{db: mockDB}
	got := h.resolveActiveAIIDFromAIcall(context.Background(), ac)
	if got != aiID {
		t.Fatalf("expected %v, got %v", aiID, got)
	}
}

func TestResolveActiveAIID_delegatesToFromAIcall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	aicallID := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())
	ac := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   aiID,
	}

	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil)

	h := &messageHandler{reqHandler: mockReq}
	got := h.resolveActiveAIID(context.Background(), aicallID)
	if got != aiID {
		t.Fatalf("expected %v, got %v", aiID, got)
	}
}

func TestResolveTeamMemberAIID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	aicallID := uuid.Must(uuid.NewV4())
	teamID := uuid.Must(uuid.NewV4())
	memberID := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())

	ac := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeTeam,
		AssistanceID:   teamID,
	}

	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil)

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
		Members: []team.Member{{ID: memberID, AIID: aiID}},
	}, nil)

	h := &messageHandler{db: mockDB, reqHandler: mockReq}
	got := h.resolveTeamMemberAIID(context.Background(), aicallID, memberID)
	if got != aiID {
		t.Fatalf("expected %v, got %v", aiID, got)
	}
}

func TestResolveActiveAIIDFromAIcall_errors(t *testing.T) {
	teamID := uuid.Must(uuid.NewV4())
	memberID := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name      string
		ac        *aicall.AIcall
		mockSetup func(*dbhandler.MockDBHandler)
	}{
		{
			name: "team type — TeamGet error returns uuid.Nil",
			ac: &aicall.AIcall{
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    teamID,
				CurrentMemberID: memberID,
			},
			mockSetup: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().TeamGet(gomock.Any(), teamID).Return(nil, errors.New("db error"))
			},
		},
		{
			name: "team type — member not found returns uuid.Nil",
			ac: &aicall.AIcall{
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    teamID,
				CurrentMemberID: uuid.Must(uuid.NewV4()), // not in team
			},
			mockSetup: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
					Members: []team.Member{{ID: memberID, AIID: aiID}},
				}, nil)
			},
		},
		{
			name: "unknown AssistanceType returns uuid.Nil",
			ac: &aicall.AIcall{
				AssistanceType: "unknown",
				AssistanceID:   teamID,
			},
			mockSetup: func(*dbhandler.MockDBHandler) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := dbhandler.NewMockDBHandler(ctrl)
			tt.mockSetup(mockDB)

			h := &messageHandler{db: mockDB}
			got := h.resolveActiveAIIDFromAIcall(context.Background(), tt.ac)
			if got != uuid.Nil {
				t.Errorf("expected uuid.Nil, got %v", got)
			}
		})
	}
}

func TestResolveActiveAIID_errors(t *testing.T) {
	aicallID := uuid.Must(uuid.NewV4())

	t.Run("nil reqHandler returns uuid.Nil", func(t *testing.T) {
		h := &messageHandler{}
		got := h.resolveActiveAIID(context.Background(), aicallID)
		if got != uuid.Nil {
			t.Errorf("expected uuid.Nil, got %v", got)
		}
	})

	t.Run("AIcallGet error returns uuid.Nil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockReq := requesthandler.NewMockRequestHandler(ctrl)
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(nil, errors.New("rpc error"))

		h := &messageHandler{reqHandler: mockReq}
		got := h.resolveActiveAIID(context.Background(), aicallID)
		if got != uuid.Nil {
			t.Errorf("expected uuid.Nil, got %v", got)
		}
	})
}

func TestResolveTeamMemberAIID_errors(t *testing.T) {
	aicallID := uuid.Must(uuid.NewV4())
	teamID := uuid.Must(uuid.NewV4())
	memberID := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())

	t.Run("nil reqHandler returns uuid.Nil", func(t *testing.T) {
		h := &messageHandler{}
		got := h.resolveTeamMemberAIID(context.Background(), aicallID, memberID)
		if got != uuid.Nil {
			t.Errorf("expected uuid.Nil, got %v", got)
		}
	})

	t.Run("AIcallGet error returns uuid.Nil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockReq := requesthandler.NewMockRequestHandler(ctrl)
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(nil, errors.New("rpc error"))

		h := &messageHandler{reqHandler: mockReq}
		got := h.resolveTeamMemberAIID(context.Background(), aicallID, memberID)
		if got != uuid.Nil {
			t.Errorf("expected uuid.Nil, got %v", got)
		}
	})

	t.Run("wrong AssistanceType returns uuid.Nil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ac := &aicall.AIcall{AssistanceType: aicall.AssistanceTypeAI, AssistanceID: teamID}
		mockReq := requesthandler.NewMockRequestHandler(ctrl)
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil)

		h := &messageHandler{reqHandler: mockReq}
		got := h.resolveTeamMemberAIID(context.Background(), aicallID, memberID)
		if got != uuid.Nil {
			t.Errorf("expected uuid.Nil, got %v", got)
		}
	})

	t.Run("TeamGet error returns uuid.Nil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ac := &aicall.AIcall{AssistanceType: aicall.AssistanceTypeTeam, AssistanceID: teamID}
		mockReq := requesthandler.NewMockRequestHandler(ctrl)
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil)

		mockDB := dbhandler.NewMockDBHandler(ctrl)
		mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(nil, errors.New("db error"))

		h := &messageHandler{db: mockDB, reqHandler: mockReq}
		got := h.resolveTeamMemberAIID(context.Background(), aicallID, memberID)
		if got != uuid.Nil {
			t.Errorf("expected uuid.Nil, got %v", got)
		}
	})

	t.Run("member not found returns uuid.Nil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ac := &aicall.AIcall{AssistanceType: aicall.AssistanceTypeTeam, AssistanceID: teamID}
		mockReq := requesthandler.NewMockRequestHandler(ctrl)
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil)

		mockDB := dbhandler.NewMockDBHandler(ctrl)
		mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
			Members: []team.Member{{ID: uuid.Must(uuid.NewV4()), AIID: aiID}}, // different memberID
		}, nil)

		h := &messageHandler{db: mockDB, reqHandler: mockReq}
		got := h.resolveTeamMemberAIID(context.Background(), aicallID, memberID)
		if got != uuid.Nil {
			t.Errorf("expected uuid.Nil, got %v", got)
		}
	})
}

func TestEventPMTeamMemberSwitched(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customerID := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	teamID := uuid.Must(uuid.NewV4())
	toMemberID := uuid.Must(uuid.NewV4())
	fromMemberID := uuid.Must(uuid.NewV4())
	fromAIID := uuid.Must(uuid.NewV4())
	toAIID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	createdMsgID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.MemberSwitchedEvent{
		CustomerID:             customerID,
		PipecatcallReferenceID: aicallID,
		ActiveflowID:           activeflowID,
		TransitionFunctionName: "escalate",
		FromMember: pmmessage.MemberInfo{
			ID:          fromMemberID,
			Name:        "Alice",
			EngineModel: "openai.gpt-4o",
		},
		ToMember: pmmessage.MemberInfo{
			ID:          toMemberID,
			Name:        "Bob",
			EngineModel: "openai.gpt-4o-mini",
		},
	}

	// resolveTeamMemberAIID is called twice: once for fromMember, once for toMember.
	ac := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeTeam,
		AssistanceID:   teamID,
	}
	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil).Times(2)

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
		Members: []team.Member{
			{ID: fromMemberID, AIID: fromAIID},
			{ID: toMemberID, AIID: toAIID},
		},
	}, nil).Times(2)

	// Notification must be attributed to the FROM AI (fromAIID).
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *message.Message) error {
			if m.ActiveAIID != fromAIID {
				t.Errorf("expected ActiveAIID %v (fromAIID), got %v", fromAIID, m.ActiveAIID)
			}
			if m.AIcallID != aicallID {
				t.Errorf("expected AIcallID %v, got %v", aicallID, m.AIcallID)
			}
			if m.Role != message.RoleNotification {
				t.Errorf("expected Role notification, got %s", m.Role)
			}
			m.ID = createdMsgID
			return nil
		},
	).Times(1)

	createdMsg := &message.Message{}
	createdMsg.ID = createdMsgID
	createdMsg.CustomerID = customerID
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(createdMsg, nil).Times(1)

	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	h := &messageHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	h.EventPMTeamMemberSwitched(context.Background(), evt)
}

func TestEventPMTeamMemberSwitched_participant_written(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customerID := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	teamID := uuid.Must(uuid.NewV4())
	toMemberID := uuid.Must(uuid.NewV4())
	fromMemberID := uuid.Must(uuid.NewV4())
	fromAIID := uuid.Must(uuid.NewV4())
	toAIID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	createdMsgID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.MemberSwitchedEvent{
		CustomerID:             customerID,
		PipecatcallReferenceID: aicallID,
		ActiveflowID:           activeflowID,
		TransitionFunctionName: "escalate",
		FromMember:             pmmessage.MemberInfo{ID: fromMemberID, Name: "Alice", EngineModel: "openai.gpt-4o"},
		ToMember:               pmmessage.MemberInfo{ID: toMemberID, Name: "Bob", EngineModel: "openai.gpt-4o-mini"},
	}

	ac := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeTeam,
		AssistanceID:   teamID,
	}

	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil).Times(2)

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
		Members: []team.Member{
			{ID: fromMemberID, AIID: fromAIID},
			{ID: toMemberID, AIID: toAIID},
		},
	}, nil).Times(2)
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *message.Message) error {
			m.ID = createdMsgID
			return nil
		},
	).Times(1)
	createdMsg := &message.Message{}
	createdMsg.ID = createdMsgID
	createdMsg.CustomerID = customerID
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(createdMsg, nil).Times(1)

	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockParticipant := participanthandler.NewMockParticipantHandler(ctrl)
	// Participant record must use the TO AI's ID.
	mockParticipant.EXPECT().Create(gomock.Any(), aicallID, toAIID).Return(nil).Times(1)

	h := &messageHandler{
		db:                 mockDB,
		notifyHandler:      mockNotify,
		reqHandler:         mockReq,
		utilHandler:        utilhandler.NewUtilHandler(),
		participantHandler: mockParticipant,
	}

	h.EventPMTeamMemberSwitched(context.Background(), evt)
}

func TestEventPMTeamMemberSwitched_participant_skipped_when_nil_ai(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customerID := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	toMemberID := uuid.Must(uuid.NewV4())
	fromMemberID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	createdMsgID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.MemberSwitchedEvent{
		CustomerID:             customerID,
		PipecatcallReferenceID: aicallID,
		ActiveflowID:           activeflowID,
		TransitionFunctionName: "escalate",
		FromMember:             pmmessage.MemberInfo{ID: fromMemberID, Name: "Alice"},
		ToMember:               pmmessage.MemberInfo{ID: toMemberID, Name: "Bob"},
	}

	// Both resolveTeamMemberAIID calls (from and to) fail because AIcall get fails.
	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(nil, errors.New("not found")).Times(2)

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *message.Message) error {
			m.ID = createdMsgID
			return nil
		},
	).Times(1)
	createdMsg := &message.Message{}
	createdMsg.ID = createdMsgID
	createdMsg.CustomerID = customerID
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(createdMsg, nil).Times(1)

	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockParticipant := participanthandler.NewMockParticipantHandler(ctrl)
	// Create must NOT be called when toAIID == uuid.Nil
	mockParticipant.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	h := &messageHandler{
		db:                 mockDB,
		notifyHandler:      mockNotify,
		reqHandler:         mockReq,
		utilHandler:        utilhandler.NewUtilHandler(),
		participantHandler: mockParticipant,
	}

	h.EventPMTeamMemberSwitched(context.Background(), evt)
}

func TestEventPMTeamMemberSwitched_participant_create_error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customerID := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	teamID := uuid.Must(uuid.NewV4())
	toMemberID := uuid.Must(uuid.NewV4())
	fromMemberID := uuid.Must(uuid.NewV4())
	fromAIID := uuid.Must(uuid.NewV4())
	toAIID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	createdMsgID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.MemberSwitchedEvent{
		CustomerID:             customerID,
		PipecatcallReferenceID: aicallID,
		ActiveflowID:           activeflowID,
		TransitionFunctionName: "escalate",
		FromMember:             pmmessage.MemberInfo{ID: fromMemberID, Name: "Alice", EngineModel: "openai.gpt-4o"},
		ToMember:               pmmessage.MemberInfo{ID: toMemberID, Name: "Bob", EngineModel: "openai.gpt-4o-mini"},
	}

	ac := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeTeam,
		AssistanceID:   teamID,
	}

	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil).Times(2)

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
		Members: []team.Member{
			{ID: fromMemberID, AIID: fromAIID},
			{ID: toMemberID, AIID: toAIID},
		},
	}, nil).Times(2)
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *message.Message) error {
			m.ID = createdMsgID
			return nil
		},
	).Times(1)
	createdMsg := &message.Message{}
	createdMsg.ID = createdMsgID
	createdMsg.CustomerID = customerID
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(createdMsg, nil).Times(1)

	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockParticipant := participanthandler.NewMockParticipantHandler(ctrl)
	// Create is called with toAIID; returns an error — handler must log a warning and continue.
	mockParticipant.EXPECT().Create(gomock.Any(), aicallID, toAIID).Return(errors.New("db error")).Times(1)

	h := &messageHandler{
		db:                 mockDB,
		notifyHandler:      mockNotify,
		reqHandler:         mockReq,
		utilHandler:        utilhandler.NewUtilHandler(),
		participantHandler: mockParticipant,
	}

	h.EventPMTeamMemberSwitched(context.Background(), evt)
}

func TestEventPMTeamMemberSwitched_from_ai_nil_participant_still_written(t *testing.T) {
	// fromMember is absent from the team → fromAIID resolves to uuid.Nil.
	// toMember is present → toAIID resolves to a real ID.
	// Expected: notification has ActiveAIID = uuid.Nil; participant Create is still called with toAIID.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	customerID := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	teamID := uuid.Must(uuid.NewV4())
	toMemberID := uuid.Must(uuid.NewV4())
	fromMemberID := uuid.Must(uuid.NewV4()) // will NOT appear in team members
	toAIID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	createdMsgID := uuid.Must(uuid.NewV4())

	evt := &pmmessage.MemberSwitchedEvent{
		CustomerID:             customerID,
		PipecatcallReferenceID: aicallID,
		ActiveflowID:           activeflowID,
		TransitionFunctionName: "escalate",
		FromMember:             pmmessage.MemberInfo{ID: fromMemberID, Name: "Alice", EngineModel: "openai.gpt-4o"},
		ToMember:               pmmessage.MemberInfo{ID: toMemberID, Name: "Bob", EngineModel: "openai.gpt-4o-mini"},
	}

	ac := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeTeam,
		AssistanceID:   teamID,
	}

	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil).Times(2)

	// Team only contains toMember; fromMember lookup will log a warning and return uuid.Nil.
	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().TeamGet(gomock.Any(), teamID).Return(&team.Team{
		Members: []team.Member{
			{ID: toMemberID, AIID: toAIID},
		},
	}, nil).Times(2)

	// Notification is created with ActiveAIID = uuid.Nil (fromAIID not found).
	mockDB.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, m *message.Message) error {
			if m.ActiveAIID != uuid.Nil {
				t.Errorf("expected ActiveAIID uuid.Nil (fromMember absent), got %v", m.ActiveAIID)
			}
			m.ID = createdMsgID
			return nil
		},
	).Times(1)
	createdMsg := &message.Message{}
	createdMsg.ID = createdMsgID
	createdMsg.CustomerID = customerID
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(createdMsg, nil).Times(1)

	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	mockParticipant := participanthandler.NewMockParticipantHandler(ctrl)
	// Participant Create must still be called with toAIID despite fromAIID being uuid.Nil.
	mockParticipant.EXPECT().Create(gomock.Any(), aicallID, toAIID).Return(nil).Times(1)

	h := &messageHandler{
		db:                 mockDB,
		notifyHandler:      mockNotify,
		reqHandler:         mockReq,
		utilHandler:        utilhandler.NewUtilHandler(),
		participantHandler: mockParticipant,
	}

	h.EventPMTeamMemberSwitched(context.Background(), evt)
}
