package messagehandler

import (
	"context"
	"strings"
	"testing"
	"time"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/dbhandler"
)

type sendFixture struct {
	util   *utilhandler.MockUtilHandler
	db     *dbhandler.MockDBHandler
	req    *requesthandler.MockRequestHandler
	notify *notifyhandler.MockNotifyHandler

	smsTelnyx *MockSMSProvider
	smsBird   *MockSMSProvider
	mmsTelnyx *MockMMSProvider
	mmsBird   *MockMMSProvider

	handler *messageHandler
}

func newSendFixture(t *testing.T) *sendFixture {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	f := &sendFixture{
		util:      newTestUtil(mc),
		db:        dbhandler.NewMockDBHandler(mc),
		req:       requesthandler.NewMockRequestHandler(mc),
		notify:    notifyhandler.NewMockNotifyHandler(mc),
		smsTelnyx: NewMockSMSProvider(mc),
		smsBird:   NewMockSMSProvider(mc),
		mmsTelnyx: NewMockMMSProvider(mc),
		mmsBird:   NewMockMMSProvider(mc),
	}
	f.handler = &messageHandler{
		utilHandler:   f.util,
		db:            f.db,
		reqHandler:    f.req,
		notifyHandler: f.notify,
		smsProviders: []namedSMSProvider{
			{name: message.ProviderNameTelnyx, provider: f.smsTelnyx},
			{name: message.ProviderNameMessagebird, provider: f.smsBird},
		},
		mmsTelnyx:      f.mmsTelnyx,
		mmsMessagebird: f.mmsBird,
	}

	return f
}

func (f *sendFixture) expectVerified() {
	f.req.EXPECT().CustomerV1CustomerGet(gomock.Any(), testCustomerID).Return(&cucustomer.Customer{
		ID:                         testCustomerID,
		IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
	}, nil)
}

// Test_Send_promotion checks the type, the parts and the balance check type that Send decides from the text, at the
// boundaries of the standard. The expected numbers come from 3GPP TS 23.038: 160 septets or 70 UCS-2 units fit one
// segment, 153 and 67 per segment of a longer message.
func Test_Send_promotion(t *testing.T) {
	tests := []struct {
		name string
		text string

		expectType   message.Type
		expectRef    bmbilling.ReferenceType
		expectParts  int
		expectTooBig bool
	}{
		{"gsm7 160 is sms", strings.Repeat("a", 160), message.TypeSMS, bmbilling.ReferenceTypeSMS, 1, false},
		{"gsm7 161 is promoted", strings.Repeat("a", 161), message.TypeMMS, bmbilling.ReferenceTypeMMS, 2, false},
		{"ucs2 70 is sms", strings.Repeat("가", 70), message.TypeSMS, bmbilling.ReferenceTypeSMS, 1, false},
		{"ucs2 71 is promoted", strings.Repeat("가", 71), message.TypeMMS, bmbilling.ReferenceTypeMMS, 2, false},
		{"extension char making 160 is sms", strings.Repeat("a", 158) + "{", message.TypeSMS, bmbilling.ReferenceTypeSMS, 1, false},
		{"extension char making 161 is promoted", strings.Repeat("a", 159) + "{", message.TypeMMS, bmbilling.ReferenceTypeMMS, 2, false},
		{"emoji making 70 units is sms", strings.Repeat("가", 68) + "\U0001F600", message.TypeSMS, bmbilling.ReferenceTypeSMS, 1, false},
		{"emoji making 71 units is promoted", strings.Repeat("가", 69) + "\U0001F600", message.TypeMMS, bmbilling.ReferenceTypeMMS, 2, false},
		{"10 segments is promoted", strings.Repeat("a", 1530), message.TypeMMS, bmbilling.ReferenceTypeMMS, 10, false},
		{"11 segments is refused", strings.Repeat("a", 1531), "", "", 0, true},
		{"ucs2 671 units is refused", strings.Repeat("가", 671), "", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSendFixture(t)
			ctx := context.Background()

			source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"}
			destinations := []commonaddress.Address{
				{Type: commonaddress.TypeTel, Target: "+15550100002"},
				{Type: commonaddress.TypeTel, Target: "+821000000003"},
			}

			f.expectVerified()

			if tt.expectTooBig {
				res, err := f.handler.Send(ctx, testMessageID, testCustomerID, source, destinations, tt.text)
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				assertVoipbinError(t, err, cerrors.StatusInvalidArgument, "TEXT_TOO_LONG")
				return
			}

			// the balance check asks for the type that is stored, once for each recipient.
			f.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(gomock.Any(), testCustomerID, tt.expectRef, "", 2).Return(true, nil)

			var created *message.Message
			f.db.EXPECT().MessageCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, m *message.Message) error {
				created = m
				return nil
			})
			stored := storedMessage(testTargets("+15550100002", "+821000000003"))
			newMemDB(f.db, stored)
			f.db.EXPECT().MessageGet(ctx, testMessageID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*message.Message, error) {
				return created, nil
			})
			f.notify.EXPECT().PublishWebhookEvent(ctx, testCustomerID, message.EventTypeMessageCreated, gomock.Any())

			// the dispatch accepts everything, which is enough to finish it; the last call is the event of the final
			// write and the test waits for it instead of sleeping.
			done := make(chan struct{})
			f.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testCustomerID, message.EventTypeMessageUpdated, gomock.Any()).Do(func(_ context.Context, _ uuid.UUID, _ string, _ any) { close(done) })
			accept := func(n int) []SendResult {
				res := make([]SendResult, n)
				for i := range res {
					res[i] = SendResult{Outcome: SendOutcomeAccepted}
				}
				return res
			}
			if tt.expectType == message.TypeSMS {
				f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, gomock.Any(), gomock.Any(), tt.text, gomock.Any()).Return(accept(2))
			} else {
				// +1 to +1 goes to the mms stage first; the other target skips it and is sent as sms segments.
				f.mmsTelnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, gomock.Any(), gomock.Any(), tt.text, gomock.Nil(), gomock.Any()).Return(accept(1))
				f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, gomock.Any(), gomock.Any(), tt.text, gomock.Any()).Return(accept(1))
			}

			// the stored message must have the decided type and parts when it is created, because the billing reads
			// the created event.
			_, err := f.handler.Send(ctx, testMessageID, testCustomerID, source, destinations, tt.text)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if created == nil {
				t.Fatalf("The message was not created")
			}
			if created.Type != tt.expectType {
				t.Errorf("Wrong type. expect: %s, got: %s", tt.expectType, created.Type)
			}
			if created.ProviderName != message.ProviderNameTelnyx {
				t.Errorf("Wrong provider. expect: telnyx, got: %s", created.ProviderName)
			}
			if len(created.Targets) != 2 {
				t.Fatalf("Wrong targets. got: %v", created.Targets)
			}
			for _, tg := range created.Targets {
				if tg.Parts != tt.expectParts || tg.Status != target.StatusQueued {
					t.Errorf("Wrong target. expect parts: %d, got: %v", tt.expectParts, tg)
				}
			}

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatalf("The dispatch did not finish")
			}
		})
	}
}

func Test_Send_rejected(t *testing.T) {
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"}
	dests := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550100002"}}

	tests := []struct {
		name string

		source *commonaddress.Address
		dests  []commonaddress.Address

		expectReason string
	}{
		{"no source", nil, dests, "SOURCE_REQUIRED"},
		{"no destination", source, nil, "DESTINATION_REQUIRED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// no provider and no database call is expected: the mocks fail the test on any call.
			f := newSendFixture(t)
			f.expectVerified()

			res, err := f.handler.Send(context.Background(), testMessageID, testCustomerID, tt.source, tt.dests, "hi")
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			assertVoipbinError(t, err, cerrors.StatusInvalidArgument, tt.expectReason)
		})
	}
}
