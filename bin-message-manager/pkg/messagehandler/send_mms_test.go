package messagehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
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

var testMessageID = uuid.FromStringOrNil("804d4eb1-00ef-424b-9e14-e8d4c7a060e7")

func testTargets(numbers ...string) []target.Target {
	res := []target.Target{}
	for _, n := range numbers {
		res = append(res, target.Target{
			Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: n},
			Status:      target.StatusQueued,
		})
	}
	return res
}

type dispatchMocks struct {
	util    *utilhandler.MockUtilHandler
	db      *dbhandler.MockDBHandler
	notify  *notifyhandler.MockNotifyHandler
	telnyx  *MockMMSProvider
	bird    *MockMMSProvider
	handler *messageHandler
}

func newDispatchMocks(t *testing.T) *dispatchMocks {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &dispatchMocks{
		util:   newTestUtil(mc),
		db:     dbhandler.NewMockDBHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		telnyx: NewMockMMSProvider(mc),
		bird:   NewMockMMSProvider(mc),
	}
	m.handler = &messageHandler{
		utilHandler:    m.util,
		db:             m.db,
		notifyHandler:  m.notify,
		mmsTelnyx:      m.telnyx,
		mmsMessagebird: m.bird,
	}

	return m
}

// Test_dispatchMMS covers the provider selection of an mms: every target goes to telnyx once, only a
// definitive refusal of an eligible target goes to messagebird once, and no target is sent as text only.
func Test_dispatchMMS(t *testing.T) {
	const (
		usSource = "+1555****0001"
		usDest1  = "+1555****0002"
		usDest2  = "+1555****0003"
		krDest   = "+8210****0004"
	)
	now := time.Date(2026, 11, 6, 1, 2, 3, 0, time.UTC)

	failedTarget := func(number string) target.Target {
		return target.Target{
			Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: number},
			Status:      target.StatusFailed,
			TMUpdate:    &now,
		}
	}

	tests := []struct {
		name string

		source  string
		targets []target.Target

		telnyxOutcomes []SendResult

		// the targets that are expected to reach messagebird, and its outcomes
		expectBirdTargets []target.Target
		birdOutcomes      []SendResult

		expectProvider message.ProviderName
		expectFinal    []target.Target
	}{
		{
			name:           "all accepted by telnyx",
			source:         usSource,
			targets:        testTargets(usDest1, usDest2),
			telnyxOutcomes: outs(SendOutcomeAccepted, SendOutcomeAccepted),
			expectProvider: message.ProviderNameTelnyx,
			expectFinal:    testTargets(usDest1, usDest2),
		},
		{
			name:              "one refused by telnyx is accepted by messagebird",
			source:            usSource,
			targets:           testTargets(usDest1, usDest2),
			telnyxOutcomes:    outs(SendOutcomeAccepted, SendOutcomeRejected),
			expectBirdTargets: testTargets(usDest2),
			birdOutcomes:      outs(SendOutcomeAccepted),
			expectProvider:    message.ProviderNameTelnyx,
			expectFinal:       testTargets(usDest1, usDest2),
		},
		{
			name:              "all refused by telnyx and accepted by messagebird records messagebird",
			source:            usSource,
			targets:           testTargets(usDest1, usDest2),
			telnyxOutcomes:    outs(SendOutcomeRejected, SendOutcomeRejected),
			expectBirdTargets: testTargets(usDest1, usDest2),
			birdOutcomes:      outs(SendOutcomeAccepted, SendOutcomeAccepted),
			expectProvider:    message.ProviderNameMessagebird,
			expectFinal:       testTargets(usDest1, usDest2),
		},
		{
			name:              "refused by both is failed and never text only",
			source:            usSource,
			targets:           testTargets(usDest1),
			telnyxOutcomes:    outs(SendOutcomeRejected),
			expectBirdTargets: testTargets(usDest1),
			birdOutcomes:      outs(SendOutcomeRejected),
			expectProvider:    message.ProviderNameTelnyx,
			expectFinal:       []target.Target{failedTarget(usDest1)},
		},
		{
			name:              "messagebird result is not known",
			source:            usSource,
			targets:           testTargets(usDest1),
			telnyxOutcomes:    outs(SendOutcomeRejected),
			expectBirdTargets: testTargets(usDest1),
			birdOutcomes:      outs(SendOutcomeFailed),
			expectProvider:    message.ProviderNameTelnyx,
			expectFinal:       []target.Target{failedTarget(usDest1)},
		},
		{
			name:           "refused target outside the us and canada has no fallback",
			source:         usSource,
			targets:        testTargets(usDest1, krDest),
			telnyxOutcomes: outs(SendOutcomeAccepted, SendOutcomeRejected),
			expectProvider: message.ProviderNameTelnyx,
			expectFinal:    []target.Target{testTargets(usDest1)[0], failedTarget(krDest)},
		},
		{
			name:           "a source outside the us and canada has no fallback",
			source:         "+8210****0001",
			targets:        testTargets(usDest1),
			telnyxOutcomes: outs(SendOutcomeRejected),
			expectProvider: message.ProviderNameTelnyx,
			expectFinal:    []target.Target{failedTarget(usDest1)},
		},
		{
			name:           "an unknown telnyx result never falls back, so it cannot be sent twice",
			source:         usSource,
			targets:        testTargets(usDest1, usDest2),
			telnyxOutcomes: outs(SendOutcomeFailed, SendOutcomeAccepted),
			expectProvider: message.ProviderNameTelnyx,
			expectFinal:    []target.Target{failedTarget(usDest1), testTargets(usDest2)[0]},
		},
		{
			name:           "all unknown",
			source:         usSource,
			targets:        testTargets(usDest1, usDest2),
			telnyxOutcomes: outs(SendOutcomeFailed, SendOutcomeFailed),
			expectProvider: message.ProviderNameTelnyx,
			expectFinal:    []target.Target{failedTarget(usDest1), failedTarget(usDest2)},
		},
		{
			name:              "mixed: accepted, refused and fallback accepted, unknown, refused and fallback refused",
			source:            usSource,
			targets:           testTargets(usDest1, usDest2, krDest, "+1555****0005", "+1555****0006"),
			telnyxOutcomes:    outs(SendOutcomeAccepted, SendOutcomeRejected, SendOutcomeFailed, SendOutcomeRejected, SendOutcomeRejected),
			expectBirdTargets: testTargets(usDest2, "+1555****0005", "+1555****0006"),
			birdOutcomes:      outs(SendOutcomeAccepted, SendOutcomeRejected, SendOutcomeFailed),
			expectProvider:    message.ProviderNameTelnyx,
			expectFinal: []target.Target{
				testTargets(usDest1)[0],
				testTargets(usDest2)[0],
				failedTarget(krDest),
				failedTarget("+1555****0005"),
				failedTarget("+1555****0006"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newDispatchMocks(t)
			source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: tt.source}
			urls := []string{"https://example.invalid/a.png"}
			mem := newMemDB(m.db, storedMessage(append([]target.Target(nil), tt.targets...)))
			ev := expectEvents(m.notify)

			// each target is offered to telnyx exactly once, with all the urls and the text.
			m.telnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, source, tt.targets, "hello", urls, gomock.Any()).Return(tt.telnyxOutcomes).Times(1)
			if len(tt.expectBirdTargets) > 0 {
				m.bird.EXPECT().SendMMS(gomock.Any(), testMessageID, source, tt.expectBirdTargets, "hello", urls, gomock.Any()).Return(tt.birdOutcomes).Times(1)
			}

			m.handler.dispatchMMS(context.Background(), testMessageID, source, tt.targets, "hello", urls, false)

			if !reflect.DeepEqual(mem.targets(), tt.expectFinal) {
				t.Errorf("Wrong final targets.\nexpect: %v\ngot: %v", tt.expectFinal, mem.targets())
			}
			if mem.providerName() != tt.expectProvider {
				t.Errorf("Wrong provider. expect: %s, got: %s", tt.expectProvider, mem.providerName())
			}
			if ev.count() != 1 {
				t.Errorf("Wrong event count. expect: 1, got: %d", ev.count())
			}
		})
	}
}

// Test_dispatchMMS_finalWriteSurvivesTheDispatchDeadline checks that the result is written with its own
// context even when the context of the attempts has already ended.
func Test_dispatchMMS_finalWriteSurvivesTheDispatchDeadline(t *testing.T) {
	m := newDispatchMocks(t)
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555****0001"}
	targets := testTargets("+1555****0002")
	mem := newMemDB(m.db, storedMessage(append([]target.Target(nil), targets...)))
	ev := expectEvents(m.notify)

	ctx, cancel := context.WithCancel(context.Background())
	m.telnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, source, targets, "", gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, _ *commonaddress.Address, _ []target.Target, _ string, _ []string, _ ResultCallback) []SendResult {
			cancel() // the attempts ran out of time; the target was not attempted
			return outs(SendOutcomeFailed)
		})

	m.handler.dispatchMMS(ctx, testMessageID, source, targets, "", []string{"https://example.invalid/a.png"}, false)

	got := mem.targets()
	if len(got) != 1 || got[0].Status != target.StatusFailed {
		t.Errorf("Wrong final targets: %v", got)
	}
	if ev.count() != 1 {
		t.Errorf("Wrong event count. expect: 1, got: %d", ev.count())
	}
}

func Test_dispatchMMS_finalWriteFailureIsOnlyLogged(t *testing.T) {
	m := newDispatchMocks(t)
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555****0001"}
	targets := testTargets("+1555****0002")
	mem := newMemDB(m.db, storedMessage(append([]target.Target(nil), targets...)))
	mem.failIfErr = errors.New("db down")
	mem.failMergeErr = errors.New("db down")
	ev := expectEvents(m.notify)

	m.telnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, source, targets, "hello", gomock.Any(), gomock.Any()).Return(outs(SendOutcomeAccepted))

	m.handler.dispatchMMS(context.Background(), testMessageID, source, targets, "hello", []string{"https://example.invalid/a.png"}, false)

	if mem.mergeCalls != 1 {
		t.Errorf("Wrong merge calls. expect: 1, got: %d", mem.mergeCalls)
	}
	if ev.count() != 0 {
		t.Errorf("Wrong event count. nothing was written, so nothing is published. got: %d", ev.count())
	}
}

type sendMMSFixture struct {
	mc     *gomock.Controller
	util   *utilhandler.MockUtilHandler
	db     *dbhandler.MockDBHandler
	req    *requesthandler.MockRequestHandler
	notify *notifyhandler.MockNotifyHandler
	telnyx *MockMMSProvider
	bird   *MockMMSProvider

	handler *messageHandler
}

func newSendMMSFixture(t *testing.T) *sendMMSFixture {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	f := &sendMMSFixture{
		mc:     mc,
		util:   utilhandler.NewMockUtilHandler(mc),
		db:     dbhandler.NewMockDBHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		telnyx: NewMockMMSProvider(mc),
		bird:   NewMockMMSProvider(mc),
	}
	f.handler = &messageHandler{
		utilHandler:    f.util,
		db:             f.db,
		reqHandler:     f.req,
		notifyHandler:  f.notify,
		mmsTelnyx:      f.telnyx,
		mmsMessagebird: f.bird,
	}

	return f
}

func (f *sendMMSFixture) expectVerifiedCustomer() {
	f.req.EXPECT().CustomerV1CustomerGet(gomock.Any(), testCustomerID).Return(&cucustomer.Customer{
		ID:                         testCustomerID,
		IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
	}, nil)
}

func (f *sendMMSFixture) expectValidMedia() {
	f.req.EXPECT().StorageV1FileGet(gomock.Any(), testMediaID1).Return(testMediaFile(testMediaID1, testCustomerID, "a.png", 1000), nil)
}

func Test_SendMMS(t *testing.T) {
	f := newSendMMSFixture(t)
	ctx := context.Background()

	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1 555 010 0001"}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+1 555 010 0002"}}

	expectSource := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100001"}
	expectTargets := testTargets("+15550100002")
	expectTargets[0].Parts = 1
	expectMessage := &message.Message{
		Identity:     commonidentity.Identity{ID: testMessageID, CustomerID: testCustomerID},
		Type:         message.TypeMMS,
		Source:       expectSource,
		Targets:      expectTargets,
		ProviderName: message.ProviderNameTelnyx,
		Text:         "hello",
		Medias:       []string{testMediaID1.String()},
		Direction:    message.DirectionOutbound,
	}
	urls := []string{"https://example.invalid/a.png"}

	f.expectVerifiedCustomer()
	f.expectValidMedia()
	f.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(gomock.Any(), testCustomerID, bmbilling.ReferenceTypeMMS, "", 1).Return(true, nil)
	f.req.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testMediaID1).Return(urls[0], nil)

	f.db.EXPECT().MessageCreate(ctx, expectMessage).Return(nil)
	f.db.EXPECT().MessageGet(ctx, testMessageID).Return(expectMessage, nil)
	f.notify.EXPECT().PublishWebhookEvent(ctx, testCustomerID, message.EventTypeMessageCreated, expectMessage.EventCopy())

	f.telnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, expectSource, expectTargets, "hello", urls, gomock.Any()).Return(outs(SendOutcomeAccepted))

	// the last call of the goroutine is the event of the final update, and the test waits for it
	// instead of sleeping.
	done := make(chan struct{})
	mem := newMemDB(f.db, storedMessage(append([]target.Target(nil), expectTargets...)))
	f.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testCustomerID, message.EventTypeMessageUpdated, gomock.Any()).Do(func(_ context.Context, _ uuid.UUID, _ string, _ any) { close(done) })

	res, err := f.handler.SendMMS(ctx, testMessageID, testCustomerID, source, destinations, "hello", []uuid.UUID{testMediaID1})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != expectMessage {
		t.Errorf("Wrong match. expect: %v, got: %v", expectMessage, res)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("The dispatch did not finish")
	}
	if mem.writes() != 1 {
		t.Errorf("Wrong writes. expect: 1, got: %d", mem.writes())
	}
}

// Test_SendMMS_rejected checks that every rejection happens before the message is created and before any
// provider is called, and that the checks that cost something run last.
func Test_SendMMS_rejected(t *testing.T) {
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555****0001"}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+1555****0002"}}

	tests := []struct {
		name string

		source *commonaddress.Address
		text   string
		medias []uuid.UUID

		setup func(f *sendMMSFixture)

		expectStatus cerrors.Status
		expectReason string
		expectPlain  string
	}{
		{
			name:   "identity not verified",
			source: source,
			medias: []uuid.UUID{testMediaID1},
			setup: func(f *sendMMSFixture) {
				f.req.EXPECT().CustomerV1CustomerGet(gomock.Any(), testCustomerID).Return(&cucustomer.Customer{
					ID:                         testCustomerID,
					IdentityVerificationStatus: cucustomer.IdentityVerificationStatusNone,
				}, nil)
			},
			expectPlain: "customer identity verification required to send message",
		},
		{
			name:   "no source",
			source: nil,
			medias: []uuid.UUID{testMediaID1},
			setup: func(f *sendMMSFixture) {
				f.expectVerifiedCustomer()
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "SOURCE_REQUIRED",
		},
		{
			name:   "no destination",
			source: source,
			medias: []uuid.UUID{testMediaID1},
			setup: func(f *sendMMSFixture) {
				f.expectVerifiedCustomer()
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "DESTINATION_REQUIRED",
		},
		{
			name:   "text too long",
			source: source,
			text:   string(make([]rune, mmsTextMaxLength+1)),
			medias: []uuid.UUID{testMediaID1},
			setup: func(f *sendMMSFixture) {
				f.expectVerifiedCustomer()
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "TEXT_TOO_LONG",
		},
		{
			name:   "no media",
			source: source,
			medias: nil,
			setup: func(f *sendMMSFixture) {
				f.expectVerifiedCustomer()
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_REQUIRED",
		},
		{
			name:   "media of another customer: the balance is not even asked",
			source: source,
			medias: []uuid.UUID{testMediaID1},
			setup: func(f *sendMMSFixture) {
				f.expectVerifiedCustomer()
				f.req.EXPECT().StorageV1FileGet(gomock.Any(), testMediaID1).Return(testMediaFile(testMediaID1, testOtherID, "a.png", 10), nil)
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_NOT_FOUND",
		},
		{
			name:   "insufficient balance: no url is refreshed",
			source: source,
			medias: []uuid.UUID{testMediaID1},
			setup: func(f *sendMMSFixture) {
				f.expectVerifiedCustomer()
				f.expectValidMedia()
				f.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(gomock.Any(), testCustomerID, bmbilling.ReferenceTypeMMS, "", 1).Return(false, nil)
			},
			expectStatus: cerrors.StatusPaymentRequired,
			expectReason: "INSUFFICIENT_BALANCE",
		},
		{
			name:   "balance check error",
			source: source,
			medias: []uuid.UUID{testMediaID1},
			setup: func(f *sendMMSFixture) {
				f.expectVerifiedCustomer()
				f.expectValidMedia()
				f.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(gomock.Any(), testCustomerID, bmbilling.ReferenceTypeMMS, "", 1).Return(false, errors.New("billing down"))
			},
			expectPlain: "could not validate the customer's balance: billing down",
		},
		{
			name:   "url refresh error",
			source: source,
			medias: []uuid.UUID{testMediaID1},
			setup: func(f *sendMMSFixture) {
				f.expectVerifiedCustomer()
				f.expectValidMedia()
				f.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(gomock.Any(), testCustomerID, bmbilling.ReferenceTypeMMS, "", 1).Return(true, nil)
				f.req.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testMediaID1).Return("", cerrors.Unavailable("storage-manager", "SIGNING_NOT_CONFIGURED", "no key"))
			},
			expectStatus: cerrors.StatusUnavailable,
			expectReason: "SIGNING_NOT_CONFIGURED",
		},
		{
			name:   "create error",
			source: source,
			medias: []uuid.UUID{testMediaID1},
			setup: func(f *sendMMSFixture) {
				f.expectVerifiedCustomer()
				f.expectValidMedia()
				f.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(gomock.Any(), testCustomerID, bmbilling.ReferenceTypeMMS, "", 1).Return(true, nil)
				f.req.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testMediaID1).Return("https://example.invalid/a.png", nil)
				f.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(errors.New("db down"))
			},
			expectPlain: "db down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// the providers are mocks without expectations: any call of them fails the test.
			f := newSendMMSFixture(t)
			tt.setup(f)

			dests := append([]commonaddress.Address(nil), destinations...)
			if tt.name == "no destination" {
				dests = nil
			}
			src := tt.source
			if src != nil {
				c := *src
				src = &c
			}

			res, err := f.handler.SendMMS(context.Background(), testMessageID, testCustomerID, src, dests, tt.text, tt.medias)
			if err == nil || res != nil {
				t.Fatalf("Wrong match. expect: error and no message, got: res: %v, err: %v", res, err)
			}

			if tt.expectReason != "" {
				assertVoipbinError(t, err, tt.expectStatus, tt.expectReason)
			}
			if tt.expectPlain != "" && err.Error() != tt.expectPlain {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectPlain, err)
			}
		})
	}
}

// Test_SendMMS_doesNotChangeSMS pins that the sms path still asks the sms balance type, so the shared
// address helper did not change it.
func Test_SendMMS_usesTheMMSBalanceType(t *testing.T) {
	f := newSendMMSFixture(t)

	f.expectVerifiedCustomer()
	f.expectValidMedia()
	// two destinations are two billed targets
	f.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(gomock.Any(), testCustomerID, bmbilling.ReferenceTypeMMS, "", 2).Return(false, nil)

	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555****0001"}
	destinations := []commonaddress.Address{
		{Type: commonaddress.TypeTel, Target: "+1555****0002"},
		{Type: commonaddress.TypeTel, Target: "+1555****0003"},
	}

	_, err := f.handler.SendMMS(context.Background(), testMessageID, testCustomerID, source, destinations, "hi", []uuid.UUID{testMediaID1})
	assertVoipbinError(t, err, cerrors.StatusPaymentRequired, "INSUFFICIENT_BALANCE")
}

// Test_dispatchMMS_providerContractViolation checks that a provider answer with the wrong number of
// outcomes fails every target and sends nothing through the fallback.
func Test_dispatchMMS_providerContractViolation(t *testing.T) {
	m := newDispatchMocks(t)
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555****0001"}
	targets := testTargets("+1555****0002", "+1555****0003")
	mem := newMemDB(m.db, storedMessage(append([]target.Target(nil), targets...)))
	expectEvents(m.notify)

	m.telnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, source, targets, "", gomock.Any(), gomock.Any()).Return(outs(SendOutcomeRejected))

	m.handler.dispatchMMS(context.Background(), testMessageID, source, targets, "", []string{"https://example.invalid/a.png"}, false)

	for _, f := range mem.targets() {
		if f.Status != target.StatusFailed {
			t.Errorf("Wrong final target. expect: failed, got: %v", f.Status)
		}
	}
}
