package messagehandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

// sendLogHook forwards the messages accepted by the filter to a channel, without blocking the logger.
type sendLogHook struct {
	filter func(msg string) bool
	ch     chan string
}

func (h *sendLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *sendLogHook) Fire(e *logrus.Entry) error {
	if !h.filter(e.Message) {
		return nil
	}

	select {
	case h.ch <- e.Message:
	default:
	}
	return nil
}

// Test_Send_updateTargetsError covers the asynchronous provider dispatch when saving the provider result fails.
// The provider map is iterated in random order, so both providers succeed and only the first one that runs is used.
func Test_Send_updateTargetsError(t *testing.T) {
	const waitLimit = 5 * time.Second

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newErrorPathHandler(mc)
	mockBird := NewMockMessageHandlerMessagebird(mc)
	mockTelnyx := NewMockMessageHandlerTelnyx(mc)
	h.messageHandlerMessagebird = mockBird
	h.messageHandlerTelnyx = mockTelnyx
	ctx := context.Background()

	id := uuid.FromStringOrNil("1b0c3d7e-8f21-4a56-9c34-5d6e7f8091a2")
	customerID := uuid.FromStringOrNil("2c1d4e8f-9032-4b67-8d45-6e7f8091a2b3")
	sentinel := errors.New("sentinel update targets failure")

	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550000002"}}
	stored := &message.Message{
		Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
		Source:   source,
		Targets:  []target.Target{{Destination: destinations[0], Status: target.StatusQueued}},
	}
	sent := []target.Target{{Destination: destinations[0], Status: target.StatusSent}}

	hook := &sendLogHook{
		filter: func(msg string) bool {
			return strings.Contains(msg, "handler:") && strings.Contains(msg, sentinel.Error())
		},
		ch: make(chan string, 1),
	}
	origLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	origHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.AddHook(hook)
	t.Cleanup(func() {
		logrus.StandardLogger().ReplaceHooks(origHooks)
		logrus.SetLevel(origLevel)
	})

	m.req.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(&cucustomer.Customer{
		ID:                         customerID,
		IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
	}, nil)
	m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeSMS, "", 1).Return(true, nil)

	// the synchronous part: the message is stored and the created event is published.
	m.db.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
	m.db.EXPECT().MessageGet(ctx, id).Return(stored, nil).Times(1)
	m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, message.EventTypeMessageCreated, stored).Times(1)

	// the asynchronous part: whichever provider runs first succeeds and its result cannot be saved.
	mockTelnyx.EXPECT().SendMessage(ctx, id, gomock.Any(), gomock.Any(), "hello").Return(sent, nil).AnyTimes()
	mockBird.EXPECT().SendMessage(ctx, id, gomock.Any(), gomock.Any(), "hello").Return(sent, nil).AnyTimes()
	m.db.EXPECT().MessageUpdateTargets(ctx, id, gomock.Any(), sent).Return(sentinel).Times(1)

	res, err := h.Send(ctx, id, customerID, source, destinations, "hello")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != stored {
		t.Errorf("Wrong match. expect: %v, got: %v", stored, res)
	}

	select {
	case <-hook.ch:
	case <-time.After(waitLimit):
		t.Fatalf("Wrong match. expect: the update targets failure log, got: timeout")
	}
}
