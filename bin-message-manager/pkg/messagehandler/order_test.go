package messagehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	commonaddress "monorepo/bin-common-handler/models/address"
	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

var (
	orderMB      = []message.ProviderName{message.ProviderNameMessagebird, message.ProviderNameTelnyx}
	orderTwilio  = []message.ProviderName{message.ProviderNameTwilio, message.ProviderNameTelnyx, message.ProviderNameMessagebird}
	orderDefault = []message.ProviderName{message.ProviderNameTelnyx, message.ProviderNameMessagebird}
)

func Test_providerOrder(t *testing.T) {
	yes := func(context.Context) bool { return true }
	no := func(context.Context) bool { return false }

	tests := []struct {
		name string

		owner     nmnumber.ProviderName
		available func(context.Context) bool

		expect []message.ProviderName
	}{
		{"telnyx owner", nmnumber.ProviderNameTelnyx, no, orderDefault},
		{"messagebird owner", nmnumber.ProviderNameMessagebird, no, orderMB},
		{"twilio owner with credentials", nmnumber.ProviderNameTwilio, yes, orderTwilio},
		{"twilio owner without credentials", nmnumber.ProviderNameTwilio, no, orderDefault},
		{"twilio owner and no way to ask", nmnumber.ProviderNameTwilio, nil, orderDefault},
		{"no owner", nmnumber.ProviderNameNone, yes, orderDefault},
		{"an owner that cannot send", nmnumber.ProviderName("other"), yes, orderDefault},
		{"telnyx owner with twilio credentials never gets twilio", nmnumber.ProviderNameTelnyx, yes, orderDefault},
		{"messagebird owner with twilio credentials never gets twilio", nmnumber.ProviderNameMessagebird, yes, orderMB},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &messageHandler{twilioAvailable: tt.available}

			got := h.providerOrder(context.Background(), tt.owner)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}

			// the order belongs to the message: changing it never changes the default order or another message.
			got[0] = "changed"
			if !reflect.DeepEqual(defaultProviderOrder, orderDefault) {
				t.Errorf("The default order was changed: %v", defaultProviderOrder)
			}
			if again := h.providerOrder(context.Background(), tt.owner); !reflect.DeepEqual(again, tt.expect) {
				t.Errorf("A later order is not independent. got: %v", again)
			}
		})
	}
}

func Test_mmsEligible(t *testing.T) {
	us := addr(usSource)
	tgUS := testTargets(usDest1)[0]
	tgKR := testTargets(krDest)[0]
	kr := addr("+8210****0001")

	tests := []struct {
		name string

		provider message.ProviderName
		source   *commonaddress.Address
		target   target.Target

		expect bool
	}{
		{"telnyx sends everywhere", message.ProviderNameTelnyx, us, tgKR, true},
		{"telnyx from outside", message.ProviderNameTelnyx, kr, tgUS, true},
		{"messagebird us to us", message.ProviderNameMessagebird, us, tgUS, true},
		{"messagebird to outside", message.ProviderNameMessagebird, us, tgKR, false},
		{"messagebird from outside", message.ProviderNameMessagebird, kr, tgUS, false},
		{"twilio us to us", message.ProviderNameTwilio, us, tgUS, true},
		{"twilio to outside", message.ProviderNameTwilio, us, tgKR, false},
		{"twilio from outside", message.ProviderNameTwilio, kr, tgUS, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mmsEligible(tt.provider, tt.source, tt.target); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

func Test_dispatchSessionProviderName(t *testing.T) {
	tests := []struct {
		name string

		order    []message.ProviderName
		accepted map[message.ProviderName]int

		expect message.ProviderName
	}{
		{"nothing accepted goes to the first of a telnyx order", orderDefault, nil, message.ProviderNameTelnyx},
		{"nothing accepted goes to the first of a messagebird order", orderMB, nil, message.ProviderNameMessagebird},
		{"nothing accepted goes to the first of a twilio order", orderTwilio, nil, message.ProviderNameTwilio},
		{"a tie goes to the earlier provider", orderMB, map[message.ProviderName]int{message.ProviderNameMessagebird: 1, message.ProviderNameTelnyx: 1}, message.ProviderNameMessagebird},
		{"the most accepted wins", orderMB, map[message.ProviderName]int{message.ProviderNameMessagebird: 1, message.ProviderNameTelnyx: 2}, message.ProviderNameTelnyx},
		{"the most accepted of three", orderTwilio, map[message.ProviderName]int{message.ProviderNameTwilio: 1, message.ProviderNameMessagebird: 3}, message.ProviderNameMessagebird},
		{"telnyx order tie is telnyx like before", orderDefault, map[message.ProviderName]int{message.ProviderNameTelnyx: 2, message.ProviderNameMessagebird: 2}, message.ProviderNameTelnyx},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &dispatchSession{order: tt.order, accepted: map[message.ProviderName]int{}}
			for k, v := range tt.accepted {
				s.accepted[k] = v
			}

			if got := s.providerName(); got != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, got)
			}
		})
	}
}

// T1, T6, T7, T9: the sms chain follows the order of the message.
func Test_dispatchSMS_order(t *testing.T) {
	source := addr(usSource)

	acc := func(ref string) SendResult { return result(SendOutcomeAccepted, ref, 1) }
	rej := result(SendOutcomeRejected, "", 0)
	unk := result(SendOutcomeFailed, "", 0)

	tests := []struct {
		name string

		order   []message.ProviderName
		targets []target.Target

		// the results of each provider for the targets that it is expected to get
		twilio, telnyx, bird       []SendResult
		expectTwilio, expectTelnyx []target.Target
		expectBird                 []target.Target
		expectProvider             message.ProviderName
		expectStatus               []target.Status
		expectRefs                 []string
	}{
		{
			name:    "messagebird owner: messagebird first, only the refused target goes to telnyx",
			order:   orderMB,
			targets: withParts(testTargets(usDest1, usDest2), 1),
			bird:    []SendResult{acc("b-1"), rej}, expectBird: withParts(testTargets(usDest1, usDest2), 1),
			telnyx: []SendResult{acc("t-2")}, expectTelnyx: withParts(testTargets(usDest2), 1),
			expectProvider: message.ProviderNameMessagebird, // one each: the tie goes to the earlier provider of the order
			expectStatus:   []target.Status{target.StatusQueued, target.StatusQueued},
			expectRefs:     []string{"b-1", "t-2"},
		},
		{
			name:    "messagebird owner: an unknown result is never sent again",
			order:   orderMB,
			targets: withParts(testTargets(usDest1, usDest2), 1),
			bird:    []SendResult{unk, acc("b-2")}, expectBird: withParts(testTargets(usDest1, usDest2), 1),
			expectProvider: message.ProviderNameMessagebird,
			expectStatus:   []target.Status{target.StatusFailed, target.StatusQueued},
			expectRefs:     []string{"", "b-2"},
		},
		{
			name:    "messagebird owner: nothing accepted records messagebird",
			order:   orderMB,
			targets: withParts(testTargets(usDest1), 1),
			bird:    []SendResult{rej}, expectBird: withParts(testTargets(usDest1), 1),
			telnyx: []SendResult{rej}, expectTelnyx: withParts(testTargets(usDest1), 1),
			expectProvider: message.ProviderNameMessagebird,
			expectStatus:   []target.Status{target.StatusFailed},
			expectRefs:     []string{""},
		},
		{
			name:    "twilio owner: twilio, telnyx, messagebird, only a refusal moves on",
			order:   orderTwilio,
			targets: withParts(testTargets(usDest1, usDest2, krDest), 1),
			twilio:  []SendResult{acc("SM1"), rej, rej}, expectTwilio: withParts(testTargets(usDest1, usDest2, krDest), 1),
			telnyx: []SendResult{rej, unk}, expectTelnyx: withParts(testTargets(usDest2, krDest), 1),
			bird: []SendResult{acc("b-2")}, expectBird: withParts(testTargets(usDest2), 1),
			expectProvider: message.ProviderNameTwilio, // one each for twilio and messagebird: the earlier provider wins
			expectStatus:   []target.Status{target.StatusQueued, target.StatusQueued, target.StatusFailed},
			expectRefs:     []string{"SM1", "b-2", ""},
		},
		{
			name:    "twilio owner: nothing accepted records twilio",
			order:   orderTwilio,
			targets: withParts(testTargets(usDest1), 1),
			twilio:  []SendResult{rej}, expectTwilio: withParts(testTargets(usDest1), 1),
			telnyx: []SendResult{rej}, expectTelnyx: withParts(testTargets(usDest1), 1),
			bird: []SendResult{rej}, expectBird: withParts(testTargets(usDest1), 1),
			expectProvider: message.ProviderNameTwilio,
			expectStatus:   []target.Status{target.StatusFailed},
			expectRefs:     []string{""},
		},
		{
			name:    "twilio owner: an unknown twilio result stops the target",
			order:   orderTwilio,
			targets: withParts(testTargets(usDest1), 1),
			twilio:  []SendResult{unk}, expectTwilio: withParts(testTargets(usDest1), 1),
			expectProvider: message.ProviderNameTwilio,
			expectStatus:   []target.Status{target.StatusFailed},
			expectRefs:     []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newChainFixture(t, tt.targets)
			smsTwilio := NewMockSMSProvider(gomock.NewController(t))
			f.handler.smsProviders[message.ProviderNameTwilio] = smsTwilio

			var calls []string
			record := func(name string, res []SendResult) func(context.Context, uuid.UUID, *commonaddress.Address, []target.Target, string, ResultCallback) []SendResult {
				return func(context.Context, uuid.UUID, *commonaddress.Address, []target.Target, string, ResultCallback) []SendResult {
					calls = append(calls, name)
					return res
				}
			}

			if tt.expectTwilio != nil {
				smsTwilio.EXPECT().SendSMS(gomock.Any(), testMessageID, source, tt.expectTwilio, "hello", gomock.Any()).DoAndReturn(record("twilio", tt.twilio)).Times(1)
			}
			if tt.expectTelnyx != nil {
				f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, tt.expectTelnyx, "hello", gomock.Any()).DoAndReturn(record("telnyx", tt.telnyx)).Times(1)
			}
			if tt.expectBird != nil {
				f.smsBird.EXPECT().SendSMS(gomock.Any(), testMessageID, source, tt.expectBird, "hello", gomock.Any()).DoAndReturn(record("messagebird", tt.bird)).Times(1)
			}

			f.handler.dispatchSMS(context.Background(), testMessageID, source, tt.targets, "hello", tt.order)

			// the providers are called in the order of the message
			var expectCalls []string
			for _, name := range tt.order {
				switch {
				case name == message.ProviderNameTwilio && tt.expectTwilio != nil:
					expectCalls = append(expectCalls, "twilio")
				case name == message.ProviderNameTelnyx && tt.expectTelnyx != nil:
					expectCalls = append(expectCalls, "telnyx")
				case name == message.ProviderNameMessagebird && tt.expectBird != nil:
					expectCalls = append(expectCalls, "messagebird")
				}
			}
			if !reflect.DeepEqual(calls, expectCalls) {
				t.Errorf("Wrong call order. expect: %v, got: %v", expectCalls, calls)
			}

			got := f.mem.targets()
			if !reflect.DeepEqual(statusesOf(got), tt.expectStatus) {
				t.Errorf("Wrong statuses. expect: %v, got: %v", tt.expectStatus, statusesOf(got))
			}
			if !reflect.DeepEqual(refsOf(got), tt.expectRefs) {
				t.Errorf("Wrong ids. expect: %v, got: %v", tt.expectRefs, refsOf(got))
			}
			if f.mem.providerName() != tt.expectProvider {
				t.Errorf("Wrong provider. expect: %s, got: %s", tt.expectProvider, f.mem.providerName())
			}
		})
	}
}

// failedMetric reads the mms_target_failed_total counter.
func failedMetric(t *testing.T, provider string, reason string) float64 {
	t.Helper()

	return promValue(t, "message_manager_mms_target_failed_total", map[string]string{"provider": provider, "reason": reason})
}

// A8 (T2): the metric pairs of a telnyx owned number are the four pairs that the code had before the order was a value
// of the message, and twilio is never called.
func Test_dispatchMMS_telnyxOwnerMetricPairs(t *testing.T) {
	m := newDispatchMocks(t)
	mmsTwilio := NewMockMMSProvider(gomock.NewController(t)) // any call fails the test
	m.handler.mmsProviders[message.ProviderNameTwilio] = mmsTwilio

	source := addr(usSource)
	us3 := "+1555****0007"
	targets := testTargets(usDest1, usDest2, krDest, us3)
	urls := []string{"https://example.invalid/a.png"}
	newMemDB(m.db, storedMessage(append([]target.Target(nil), targets...)))
	expectEvents(m.notify)

	before := map[[2]string]float64{}
	pairs := [][2]string{{"telnyx", "ineligible"}, {"telnyx", "error"}, {"messagebird", "rejected"}, {"messagebird", "error"}, {"telnyx", "rejected"}, {"messagebird", "ineligible"}, {"twilio", "rejected"}, {"twilio", "ineligible"}}
	for _, p := range pairs {
		before[p] = failedMetric(t, p[0], p[1])
	}

	// telnyx: us1 refused, us2 unknown, kr refused, us3 refused. messagebird: us1 refused, us3 unknown.
	m.telnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, source, targets, "hello", urls, gomock.Any()).Return(outs(SendOutcomeRejected, SendOutcomeFailed, SendOutcomeRejected, SendOutcomeRejected)).Times(1)
	m.bird.EXPECT().SendMMS(gomock.Any(), testMessageID, source, []target.Target{targets[0], targets[3]}, "hello", urls, gomock.Any()).Return(outs(SendOutcomeRejected, SendOutcomeFailed)).Times(1)

	m.handler.dispatchMMS(context.Background(), testMessageID, source, targets, "hello", urls, false, defaultProviderOrder)

	expect := map[[2]string]float64{
		{"telnyx", "ineligible"}:      1, // kr: refused and not a +1 target, so messagebird was passed over
		{"telnyx", "error"}:           1, // us2
		{"messagebird", "rejected"}:   1, // us1
		{"messagebird", "error"}:      1, // us3
		{"telnyx", "rejected"}:        0,
		{"messagebird", "ineligible"}: 0,
		{"twilio", "rejected"}:        0,
		{"twilio", "ineligible"}:      0,
	}
	for _, p := range pairs {
		if got := failedMetric(t, p[0], p[1]) - before[p]; got != expect[p] {
			t.Errorf("Wrong metric %v. expect: %v, got: %v", p, expect[p], got)
		}
	}
}

// T4: a messagebird owned number, media mms, a +1 target and a target outside. The +1 target goes messagebird, then
// telnyx. The other target skips messagebird without a call. What telnyx refuses is failed and no sms provider is called.
func Test_dispatchMMS_messagebirdOwnerMedia(t *testing.T) {
	m := newDispatchMocks(t)
	smsTelnyx := NewMockSMSProvider(gomock.NewController(t)) // any sms call fails the test
	smsBird := NewMockSMSProvider(gomock.NewController(t))
	m.handler.smsProviders = map[message.ProviderName]SMSProvider{message.ProviderNameTelnyx: smsTelnyx, message.ProviderNameMessagebird: smsBird}

	source := addr(usSource)
	targets := testTargets(usDest1, krDest)
	urls := []string{"https://example.invalid/a.png"}
	mem := newMemDB(m.db, storedMessage(append([]target.Target(nil), targets...)))
	expectEvents(m.notify)

	beforeTelnyxRej := failedMetric(t, "telnyx", "rejected")
	beforeTelnyxInel := failedMetric(t, "telnyx", "ineligible")
	beforeBirdAny := failedMetric(t, "messagebird", "rejected") + failedMetric(t, "messagebird", "ineligible") + failedMetric(t, "messagebird", "error")

	gomock.InOrder(
		m.bird.EXPECT().SendMMS(gomock.Any(), testMessageID, source, []target.Target{targets[0]}, "hello", urls, gomock.Any()).Return(outs(SendOutcomeRejected)).Times(1),
		m.telnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, source, targets, "hello", urls, gomock.Any()).Return(outs(SendOutcomeRejected, SendOutcomeRejected)).Times(1),
	)

	m.handler.dispatchMMS(context.Background(), testMessageID, source, targets, "hello", urls, false, orderMB)

	got := mem.targets()
	if !reflect.DeepEqual(statusesOf(got), []target.Status{target.StatusFailed, target.StatusFailed}) {
		t.Errorf("Wrong statuses. got: %v", statusesOf(got))
	}
	if mem.providerName() != message.ProviderNameMessagebird {
		t.Errorf("Wrong provider. expect: messagebird, got: %s", mem.providerName())
	}
	if d := failedMetric(t, "telnyx", "rejected") - beforeTelnyxRej; d != 1 { // the +1 target, nobody was passed over
		t.Errorf("Wrong (telnyx, rejected). expect: 1, got: %v", d)
	}
	if d := failedMetric(t, "telnyx", "ineligible") - beforeTelnyxInel; d != 1 { // the target outside, messagebird was passed over
		t.Errorf("Wrong (telnyx, ineligible). expect: 1, got: %v", d)
	}
	if d := failedMetric(t, "messagebird", "rejected") + failedMetric(t, "messagebird", "ineligible") + failedMetric(t, "messagebird", "error") - beforeBirdAny; d != 0 {
		t.Errorf("Wrong messagebird failed metrics. expect: 0, got: %v", d)
	}
}

// T5: a messagebird owned number, a promoted text: every mms stage refuses, then the sms chain runs in the same order
// (messagebird, telnyx). A target that is not +1 to +1 goes to the sms chain without an mms call.
func Test_dispatchMMS_messagebirdOwnerPromoted(t *testing.T) {
	m := newDispatchMocks(t)
	mc := gomock.NewController(t)
	smsTelnyx := NewMockSMSProvider(mc)
	smsBird := NewMockSMSProvider(mc)
	m.handler.smsProviders = map[message.ProviderName]SMSProvider{message.ProviderNameTelnyx: smsTelnyx, message.ProviderNameMessagebird: smsBird}

	source := addr(usSource)
	targets := withParts(testTargets(usDest1, krDest), 2)
	mem := newMemDB(m.db, storedMessage(append([]target.Target(nil), targets...)))
	expectEvents(m.notify)

	// only the +1 to +1 target is offered to the mms providers, and both refuse it.
	gomock.InOrder(
		m.bird.EXPECT().SendMMS(gomock.Any(), testMessageID, source, []target.Target{targets[0]}, "long text", gomock.Nil(), gomock.Any()).Return(outs(SendOutcomeRejected)).Times(1),
		m.telnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, source, []target.Target{targets[0]}, "long text", gomock.Nil(), gomock.Any()).Return(outs(SendOutcomeRejected)).Times(1),
		// the sms chain: the target outside first (it skipped the mms stages), then the refused one; messagebird first.
		smsBird.EXPECT().SendSMS(gomock.Any(), testMessageID, source, []target.Target{targets[1], targets[0]}, "long text", gomock.Any()).Return([]SendResult{result(SendOutcomeRejected, "", 0), result(SendOutcomeAccepted, "b-1", 2)}).Times(1),
		smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, []target.Target{targets[1]}, "long text", gomock.Any()).Return([]SendResult{result(SendOutcomeAccepted, "t-1", 2)}).Times(1),
	)

	m.handler.dispatchMMS(context.Background(), testMessageID, source, targets, "long text", nil, true, orderMB)

	got := mem.targets()
	if !reflect.DeepEqual(refsOf(got), []string{"b-1", "t-1"}) {
		t.Errorf("Wrong ids. got: %v", refsOf(got))
	}
}

// T6 for the mms stages: an unknown result is never sent to the next provider, in any order, and for a promoted
// message it does not go to the sms chain either.
func Test_dispatchMMS_unknownNeverMovesOn(t *testing.T) {
	for _, order := range [][]message.ProviderName{orderDefault, orderMB, orderTwilio} {
		for _, promoted := range []bool{false, true} {
			m := newDispatchMocks(t)
			mc := gomock.NewController(t)
			mmsTwilio := NewMockMMSProvider(mc)
			m.handler.mmsProviders[message.ProviderNameTwilio] = mmsTwilio
			// no sms provider is registered: a call would be an error, and a missing one is only logged.
			m.handler.smsProviders = map[message.ProviderName]SMSProvider{
				message.ProviderNameTelnyx: NewMockSMSProvider(mc), message.ProviderNameMessagebird: NewMockSMSProvider(mc), message.ProviderNameTwilio: NewMockSMSProvider(mc),
			}

			source := addr(usSource)
			targets := withParts(testTargets(usDest1), 2)
			mem := newMemDB(m.db, storedMessage(append([]target.Target(nil), targets...)))
			expectEvents(m.notify)

			first := map[message.ProviderName]*MockMMSProvider{message.ProviderNameTelnyx: m.telnyx, message.ProviderNameMessagebird: m.bird, message.ProviderNameTwilio: mmsTwilio}[order[0]]
			first.EXPECT().SendMMS(gomock.Any(), testMessageID, source, targets, "x", gomock.Any(), gomock.Any()).Return(outs(SendOutcomeFailed)).Times(1)

			m.handler.dispatchMMS(context.Background(), testMessageID, source, targets, "x", []string{"https://example.invalid/a.png"}, promoted, order)

			if got := mem.targets(); got[0].Status != target.StatusFailed {
				t.Errorf("Wrong status. order: %v, promoted: %v, got: %v", order, promoted, got[0].Status)
			}
		}
	}
}

// T9: a twilio owned number, media mms: twilio, telnyx, messagebird. A target that does not fit the twilio rule skips
// twilio without a call.
func Test_dispatchMMS_twilioOwnerMedia(t *testing.T) {
	m := newDispatchMocks(t)
	mmsTwilio := NewMockMMSProvider(gomock.NewController(t))
	m.handler.mmsProviders[message.ProviderNameTwilio] = mmsTwilio

	source := addr(usSource)
	targets := testTargets(usDest1, krDest, usDest2)
	urls := []string{"https://example.invalid/a.png"}
	mem := newMemDB(m.db, storedMessage(append([]target.Target(nil), targets...)))
	expectEvents(m.notify)

	gomock.InOrder(
		// twilio gets the two +1 targets; it accepts the first and refuses the second
		mmsTwilio.EXPECT().SendMMS(gomock.Any(), testMessageID, source, []target.Target{targets[0], targets[2]}, "hello", urls, gomock.Any()).Return([]SendResult{result(SendOutcomeAccepted, "MM1", 1), result(SendOutcomeRejected, "", 0)}).Times(1),
		// telnyx gets what twilio refused and what twilio passed over, in the order of the message
		m.telnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, source, []target.Target{targets[1], targets[2]}, "hello", urls, gomock.Any()).Return([]SendResult{result(SendOutcomeAccepted, "t-2", 1), result(SendOutcomeRejected, "", 0)}).Times(1),
		// messagebird gets the one that telnyx refused
		m.bird.EXPECT().SendMMS(gomock.Any(), testMessageID, source, []target.Target{targets[2]}, "hello", urls, gomock.Any()).Return(outs(SendOutcomeAccepted)).Times(1),
	)

	m.handler.dispatchMMS(context.Background(), testMessageID, source, targets, "hello", urls, false, orderTwilio)

	got := mem.targets()
	for i, tg := range got {
		if tg.Status == target.StatusFailed {
			t.Errorf("Wrong status of target %d: %v", i, tg.Status)
		}
	}
	if got[0].ProviderReferenceID != "MM1" || got[1].ProviderReferenceID != "t-2" {
		t.Errorf("Wrong ids. got: %v", refsOf(got))
	}
	// twilio 1, telnyx 1, messagebird 1: the tie goes to twilio, the first of the order.
	if mem.providerName() != message.ProviderNameTwilio {
		t.Errorf("Wrong provider. expect: twilio, got: %s", mem.providerName())
	}
}

// T9: a twilio owned number, promoted text: the mms stages use the twilio order and a refused target goes through the
// sms chain in the same order.
func Test_dispatchMMS_twilioOwnerPromoted(t *testing.T) {
	m := newDispatchMocks(t)
	mc := gomock.NewController(t)
	mmsTwilio := NewMockMMSProvider(mc)
	smsTwilio := NewMockSMSProvider(mc)
	smsTelnyx := NewMockSMSProvider(mc)
	m.handler.mmsProviders[message.ProviderNameTwilio] = mmsTwilio
	m.handler.smsProviders = map[message.ProviderName]SMSProvider{message.ProviderNameTwilio: smsTwilio, message.ProviderNameTelnyx: smsTelnyx}

	source := addr(usSource)
	targets := withParts(testTargets(usDest1), 2)
	mem := newMemDB(m.db, storedMessage(append([]target.Target(nil), targets...)))
	expectEvents(m.notify)

	gomock.InOrder(
		mmsTwilio.EXPECT().SendMMS(gomock.Any(), testMessageID, source, targets, "long", gomock.Nil(), gomock.Any()).Return(outs(SendOutcomeRejected)).Times(1),
		m.telnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, source, targets, "long", gomock.Nil(), gomock.Any()).Return(outs(SendOutcomeRejected)).Times(1),
		m.bird.EXPECT().SendMMS(gomock.Any(), testMessageID, source, targets, "long", gomock.Nil(), gomock.Any()).Return(outs(SendOutcomeRejected)).Times(1),
		smsTwilio.EXPECT().SendSMS(gomock.Any(), testMessageID, source, targets, "long", gomock.Any()).Return([]SendResult{result(SendOutcomeRejected, "", 0)}).Times(1),
		smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, targets, "long", gomock.Any()).Return([]SendResult{result(SendOutcomeAccepted, "t-1", 2)}).Times(1),
	)

	m.handler.dispatchMMS(context.Background(), testMessageID, source, targets, "long", nil, true, orderTwilio)

	if got := mem.targets(); got[0].ProviderReferenceID != "t-1" {
		t.Errorf("Wrong ids. got: %v", refsOf(got))
	}
	if mem.providerName() != message.ProviderNameTelnyx {
		t.Errorf("Wrong provider. expect: telnyx, got: %s", mem.providerName())
	}
}

// expectOwner expects the number lookups of a send, whose result holds the owner. The count is the assertion of A6.
func (f *sendFixture) expectOwner(owner nmnumber.ProviderName, times int) {
	f.req.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), gomock.Any()).Return([]nmnumber.Number{{ProviderName: owner}}, nil).Times(times)
}

// T1, T3, T8, T10 on the whole of Send: the owner of the number decides the order, the provider recorded at creation is
// the first of that order, and the number-manager is asked once.
func Test_Send_ownerOrder(t *testing.T) {
	tests := []struct {
		name string

		owner         nmnumber.ProviderName
		twilioEnabled bool
		defaultSource bool

		expectFirst message.ProviderName // the provider that is called first
		expectOrder []message.ProviderName
	}{
		{"telnyx owner", nmnumber.ProviderNameTelnyx, false, false, message.ProviderNameTelnyx, orderDefault},
		{"messagebird owner", nmnumber.ProviderNameMessagebird, false, false, message.ProviderNameMessagebird, orderMB},
		{"no owner", nmnumber.ProviderNameNone, false, false, message.ProviderNameTelnyx, orderDefault},
		{"twilio owner without credentials", nmnumber.ProviderNameTwilio, false, false, message.ProviderNameTelnyx, orderDefault},
		{"twilio owner with credentials", nmnumber.ProviderNameTwilio, true, false, message.ProviderNameTwilio, orderTwilio},
		{"messagebird owner of the default source", nmnumber.ProviderNameMessagebird, false, true, message.ProviderNameMessagebird, orderMB},
		{"telnyx owner of the default source", nmnumber.ProviderNameTelnyx, false, true, message.ProviderNameTelnyx, orderDefault},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSendFixture(t)
			ctx := context.Background()
			mc := gomock.NewController(t)

			smsTwilio := NewMockSMSProvider(mc)
			f.handler.smsProviders[message.ProviderNameTwilio] = smsTwilio
			f.handler.twilioAvailable = func(context.Context) bool { return tt.twilioEnabled }

			byName := map[message.ProviderName]*MockSMSProvider{message.ProviderNameTelnyx: f.smsTelnyx, message.ProviderNameMessagebird: f.smsBird, message.ProviderNameTwilio: smsTwilio}

			f.expectVerified()
			f.expectOwner(tt.owner, 1)

			var source *commonaddress.Address
			if tt.defaultSource {
				f.expectOutboundConfig([]cmoutboundconfig.OutboundConfig{{DefaultOutgoingSourceNumberID: testDefaultNumberID}}, nil)
			} else {
				source = &commonaddress.Address{Type: commonaddress.TypeTel, Target: usSource}
			}
			destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: usDest1}}

			f.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(gomock.Any(), testCustomerID, bmbilling.ReferenceTypeSMS, "", 1).Return(true, nil)

			var created *message.Message
			f.db.EXPECT().MessageCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, m *message.Message) error {
				created = m
				return nil
			})
			newMemDB(f.db, storedMessage(testTargets(usDest1)))
			f.db.EXPECT().MessageGet(ctx, testMessageID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*message.Message, error) {
				return created, nil
			})
			f.notify.EXPECT().PublishWebhookEvent(ctx, testCustomerID, message.EventTypeMessageCreated, gomock.Any())
			done := make(chan struct{})
			f.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testCustomerID, message.EventTypeMessageUpdated, gomock.Any()).Do(func(_ context.Context, _ uuid.UUID, _ string, _ any) { close(done) })

			// the first provider of the order accepts the target, so no other provider is called.
			byName[tt.expectFirst].EXPECT().SendSMS(gomock.Any(), testMessageID, gomock.Any(), gomock.Any(), "hi", gomock.Any()).Return([]SendResult{{Outcome: SendOutcomeAccepted}}).Times(1)

			if _, err := f.handler.Send(ctx, testMessageID, testCustomerID, source, destinations, "hi"); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if created == nil || created.ProviderName != tt.expectOrder[0] {
				t.Errorf("Wrong provider at creation. expect: %s, got: %v", tt.expectOrder[0], created)
			}

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatalf("The dispatch did not finish")
			}
		})
	}
}

// T8, T9, T10 for SendMMS: the owner of the number decides the order, the message is created with the first provider of
// it, and the number-manager is asked once.
func Test_SendMMS_ownerOrder(t *testing.T) {
	tests := []struct {
		name string

		owner         nmnumber.ProviderName
		twilioEnabled bool

		expectFirst message.ProviderName
	}{
		{"telnyx owner", nmnumber.ProviderNameTelnyx, false, message.ProviderNameTelnyx},
		{"messagebird owner", nmnumber.ProviderNameMessagebird, false, message.ProviderNameMessagebird},
		{"no owner", nmnumber.ProviderNameNone, false, message.ProviderNameTelnyx},
		{"twilio owner without credentials", nmnumber.ProviderNameTwilio, false, message.ProviderNameTelnyx},
		{"twilio owner with credentials", nmnumber.ProviderNameTwilio, true, message.ProviderNameTwilio},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newSendMMSFixture(t)
			ctx := context.Background()

			mmsTwilio := NewMockMMSProvider(f.mc)
			f.handler.mmsProviders[message.ProviderNameTwilio] = mmsTwilio
			f.handler.twilioAvailable = func(context.Context) bool { return tt.twilioEnabled }
			byName := map[message.ProviderName]*MockMMSProvider{message.ProviderNameTelnyx: f.telnyx, message.ProviderNameMessagebird: f.bird, message.ProviderNameTwilio: mmsTwilio}

			source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: usSource}
			destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: usDest1}}
			urls := []string{"https://example.invalid/a.png"}

			f.expectVerifiedCustomer()
			f.req.EXPECT().NumberV1NumberList(gomock.Any(), "", uint64(1), gomock.Any()).Return([]nmnumber.Number{{ProviderName: tt.owner}}, nil).Times(1)
			f.expectValidMedia()
			f.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(gomock.Any(), testCustomerID, bmbilling.ReferenceTypeMMS, "", 1).Return(true, nil)
			f.req.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testMediaID1).Return(urls[0], nil)

			var created *message.Message
			f.db.EXPECT().MessageCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, m *message.Message) error {
				created = m
				return nil
			})
			f.db.EXPECT().MessageGet(ctx, testMessageID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*message.Message, error) { return created, nil })
			f.notify.EXPECT().PublishWebhookEvent(ctx, testCustomerID, message.EventTypeMessageCreated, gomock.Any())
			newMemDB(f.db, storedMessage(testTargets(usDest1)))
			done := make(chan struct{})
			f.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testCustomerID, message.EventTypeMessageUpdated, gomock.Any()).Do(func(_ context.Context, _ uuid.UUID, _ string, _ any) { close(done) })

			byName[tt.expectFirst].EXPECT().SendMMS(gomock.Any(), testMessageID, gomock.Any(), gomock.Any(), "hello", urls, gomock.Any()).Return(outs(SendOutcomeAccepted)).Times(1)

			if _, err := f.handler.SendMMS(ctx, testMessageID, testCustomerID, source, destinations, "hello", []uuid.UUID{testMediaID1}); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if created == nil || created.ProviderName != tt.expectFirst {
				t.Errorf("Wrong provider at creation. expect: %s, got: %v", tt.expectFirst, created)
			}

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatalf("The dispatch did not finish")
			}
		})
	}
}
