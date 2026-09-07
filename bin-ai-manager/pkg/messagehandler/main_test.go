package messagehandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// TestMessageHandler_hasEventPMPipecatcallTerminated asserts the MessageHandler
// interface declares the EventPMPipecatcallTerminated method with the expected
// signature. This guards against accidental signature drift between the
// interface, the mock, and the implementation.
func TestMessageHandler_hasEventPMPipecatcallTerminated(t *testing.T) {
	var _ interface {
		EventPMPipecatcallTerminated(context.Context, *pmpipecatcall.Pipecatcall) error
	} = (MessageHandler)(nil)
}

func TestCreateOptions_apply(t *testing.T) {
	pcID := uuid.Must(uuid.NewV4())
	var p createParams
	WithPipecatcallID(pcID)(&p)
	WithDeliveryStatus(message.DeliveryStatusPending)(&p)
	if p.pipecatcallID != pcID || p.deliveryStatus != message.DeliveryStatusPending {
		t.Fatalf("options not applied: %+v", p)
	}
}

func TestCreateOptions_WithActiveAIID(t *testing.T) {
	aiID := uuid.Must(uuid.NewV4())
	var p createParams
	WithActiveAIID(aiID)(&p)
	if p.activeAIID != aiID {
		t.Fatalf("WithActiveAIID not applied: got %v", p.activeAIID)
	}
}

// messageWithPCC returns a gomock matcher that asserts the *message.Message
// argument has the expected PipecatcallID and DeliveryStatus fields.
func messageWithPCC(pcID uuid.UUID, status message.DeliveryStatus) gomock.Matcher {
	return messageFieldMatcher{pcID: pcID, status: status, checkPCC: true}
}

// messageWithDelivery returns a gomock matcher that asserts only the
// DeliveryStatus field of the *message.Message argument.
func messageWithDelivery(status message.DeliveryStatus) gomock.Matcher {
	return messageFieldMatcher{status: status}
}

type messageFieldMatcher struct {
	pcID     uuid.UUID
	status   message.DeliveryStatus
	checkPCC bool
}

func (m messageFieldMatcher) Matches(x any) bool {
	msg, ok := x.(*message.Message)
	if !ok || msg == nil {
		return false
	}
	if m.checkPCC && msg.PipecatcallID != m.pcID {
		return false
	}
	return msg.DeliveryStatus == m.status
}

func (m messageFieldMatcher) String() string {
	if m.checkPCC {
		return fmt.Sprintf("message with PipecatcallID=%s DeliveryStatus=%s", m.pcID, m.status)
	}
	return fmt.Sprintf("message with DeliveryStatus=%s", m.status)
}

func newTestMessageHandler(t *testing.T) (*messageHandler, *dbhandler.MockDBHandler) {
	t.Helper()
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	// Allow generated UUID + follow-up Get + notify calls without requiring
	// per-test setup; these tests focus on the MessageCreate argument.
	mockUtil.EXPECT().UUIDCreate().AnyTimes().Return(uuid.Must(uuid.NewV4()))
	mockDB.EXPECT().MessageGet(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(_ context.Context, _ uuid.UUID) (*message.Message, error) {
			return &message.Message{}, nil
		},
	)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	h := &messageHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	return h, mockDB
}

func TestCreate_withPipecatcallID_andDeliveryStatus(t *testing.T) {
	h, mockDB := newTestMessageHandler(t)
	ctx := context.Background()
	pcID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().MessageCreate(gomock.Any(), messageWithPCC(pcID, message.DeliveryStatusPending)).Return(nil)

	msg, err := h.Create(ctx, uuid.Nil, customerID, aicallID, activeflowID,
		message.DirectionIncoming, message.RoleAssistant, "hi", nil, "",
		WithPipecatcallID(pcID), WithDeliveryStatus(message.DeliveryStatusPending))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatalf("expected non-nil message")
	}
}

func TestCreate_withoutOpts_defaultsDelivered(t *testing.T) {
	h, mockDB := newTestMessageHandler(t)
	ctx := context.Background()
	customerID := uuid.Must(uuid.NewV4())
	aicallID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().MessageCreate(gomock.Any(), messageWithDelivery(message.DeliveryStatusDelivered)).Return(nil)

	if _, err := h.Create(ctx, uuid.Nil, customerID, aicallID, activeflowID,
		message.DirectionIncoming, message.RoleAssistant, "hi", nil, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test_ApplyCreateOptions pins the option probe other packages' tests match
// Create's variadic argument with. Its defaults must stay identical to Create's
// own, or a matcher built on it would assert against a row shape Create never
// produces.
func Test_ApplyCreateOptions(t *testing.T) {
	aiID := uuid.FromStringOrNil("c1000000-0001-11f0-4444-000000000001")
	pcID := uuid.FromStringOrNil("c1000000-0002-11f0-4444-000000000001")
	replyID := uuid.FromStringOrNil("c1000000-0003-11f0-4444-000000000001")

	tests := []struct {
		name string

		opts []CreateOption

		expectRes CreateOptionView
	}{
		{
			name: "no options resolves to create's own defaults",

			opts: nil,

			expectRes: CreateOptionView{
				PipecatcallID:  uuid.Nil,
				DeliveryStatus: message.DeliveryStatusDelivered,
			},
		},
		{
			name: "with active ai id",

			opts: []CreateOption{WithActiveAIID(aiID)},

			expectRes: CreateOptionView{
				PipecatcallID:  uuid.Nil,
				DeliveryStatus: message.DeliveryStatusDelivered,
				ActiveAIID:     aiID,
			},
		},
		{
			name: "every option, applied in order",

			opts: []CreateOption{
				WithPipecatcallID(pcID),
				WithDeliveryStatus(message.DeliveryStatusPending),
				WithActiveAIID(aiID),
				WithInReplyToMessageID(replyID),
				WithOrigin(message.OriginListenInternal),
			},

			expectRes: CreateOptionView{
				PipecatcallID:      pcID,
				DeliveryStatus:     message.DeliveryStatusPending,
				ActiveAIID:         aiID,
				InReplyToMessageID: replyID,
				Origin:             message.OriginListenInternal,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ApplyCreateOptions(tt.opts...)
			if res != tt.expectRes {
				t.Errorf("wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
