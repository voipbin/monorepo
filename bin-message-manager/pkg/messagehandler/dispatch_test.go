package messagehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

const (
	usSource = "+15550100001"
	usDest1  = "+15550100002"
	usDest2  = "+15550100003"
	krDest   = "+821000000004"
)

func withParts(targets []target.Target, parts int) []target.Target {
	for i := range targets {
		targets[i].Parts = parts
	}
	return targets
}

func result(o SendOutcome, ref string, parts int) SendResult {
	return SendResult{Outcome: o, ProviderReferenceID: ref, ProviderParts: parts}
}

func refsOf(targets []target.Target) []string {
	res := make([]string, 0, len(targets))
	for _, t := range targets {
		res = append(res, t.ProviderReferenceID)
	}
	return res
}

func statusesOf(targets []target.Target) []target.Status {
	res := make([]target.Status, 0, len(targets))
	for _, t := range targets {
		res = append(res, t.Status)
	}
	return res
}

type chainFixture struct {
	*sendFixture
	mem *memDB
	ev  *events
}

func newChainFixture(t *testing.T, targets []target.Target) *chainFixture {
	t.Helper()

	f := newSendFixture(t)
	mem := newMemDB(f.db, storedMessage(append([]target.Target(nil), targets...)))

	return &chainFixture{sendFixture: f, mem: mem, ev: expectEvents(f.notify)}
}

// Test_dispatchSMS covers the ordered provider chain of an sms: the first provider gets every target, only a
// definitive refusal goes on to the next one, an unknown result is never sent again, and the result is written once.
func Test_dispatchSMS(t *testing.T) {
	source := addr(usSource)

	tests := []struct {
		name string

		targets []target.Target

		telnyx []SendResult

		expectBird []target.Target // the targets that reach messagebird; nil means messagebird is not called
		bird       []SendResult

		expectProvider message.ProviderName
		expectStatus   []target.Status
		expectRefs     []string
	}{
		{
			name:           "telnyx accepts everything",
			targets:        withParts(testTargets(usDest1, usDest2), 1),
			telnyx:         []SendResult{result(SendOutcomeAccepted, "t-1", 1), result(SendOutcomeAccepted, "t-2", 1)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusQueued, target.StatusQueued},
			expectRefs:     []string{"t-1", "t-2"},
		},
		{
			name:           "a refused target goes to messagebird and the provider with the most targets is recorded",
			targets:        withParts(testTargets(usDest1, usDest2, krDest), 1),
			telnyx:         []SendResult{result(SendOutcomeAccepted, "t-1", 1), result(SendOutcomeRejected, "", 0), result(SendOutcomeRejected, "", 0)},
			expectBird:     withParts(testTargets(usDest2, krDest), 1),
			bird:           []SendResult{{Outcome: SendOutcomeAccepted, ProviderReferenceID: "b-1", Status: "sent"}, {Outcome: SendOutcomeAccepted, ProviderReferenceID: "b-1"}},
			expectProvider: message.ProviderNameMessagebird,
			expectStatus:   []target.Status{target.StatusQueued, target.StatusSent, target.StatusQueued},
			expectRefs:     []string{"t-1", "b-1", "b-1"},
		},
		{
			name:           "a tie goes to the first provider of the chain",
			targets:        withParts(testTargets(usDest1, usDest2), 1),
			telnyx:         []SendResult{result(SendOutcomeAccepted, "t-1", 1), result(SendOutcomeRejected, "", 0)},
			expectBird:     withParts(testTargets(usDest2), 1),
			bird:           []SendResult{result(SendOutcomeAccepted, "b-1", 0)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusQueued, target.StatusQueued},
			expectRefs:     []string{"t-1", "b-1"},
		},
		{
			name:           "an unknown telnyx result is not sent again and is failed",
			targets:        withParts(testTargets(usDest1, usDest2), 1),
			telnyx:         []SendResult{result(SendOutcomeFailed, "", 0), result(SendOutcomeAccepted, "t-2", 1)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusFailed, target.StatusQueued},
			expectRefs:     []string{"", "t-2"},
		},
		{
			name:           "refused by both is failed",
			targets:        withParts(testTargets(usDest1), 1),
			telnyx:         []SendResult{result(SendOutcomeRejected, "", 0)},
			expectBird:     withParts(testTargets(usDest1), 1),
			bird:           []SendResult{result(SendOutcomeRejected, "", 0)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusFailed},
			expectRefs:     []string{""},
		},
		{
			name:           "unknown at messagebird is failed",
			targets:        withParts(testTargets(usDest1), 1),
			telnyx:         []SendResult{result(SendOutcomeRejected, "", 0)},
			expectBird:     withParts(testTargets(usDest1), 1),
			bird:           []SendResult{result(SendOutcomeFailed, "", 0)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusFailed},
			expectRefs:     []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newChainFixture(t, tt.targets)

			f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, tt.targets, "hello", gomock.Any()).Return(tt.telnyx).Times(1)
			if tt.expectBird != nil {
				f.smsBird.EXPECT().SendSMS(gomock.Any(), testMessageID, source, tt.expectBird, "hello", gomock.Any()).Return(tt.bird).Times(1)
			}

			f.handler.dispatchSMS(context.Background(), testMessageID, source, tt.targets, "hello")

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
			if f.ev.count() != 1 {
				t.Errorf("Wrong event count. expect: 1, got: %d", f.ev.count())
			}
		})
	}
}

// Test_dispatchSMS_segmentMismatch checks that a provider count that differs from the platform count is counted, and
// that the provider count is stored beside the platform count and does not replace it.
func Test_dispatchSMS_segmentMismatch(t *testing.T) {
	source := addr(usSource)
	targets := withParts(testTargets(usDest1, usDest2), 1)

	f := newChainFixture(t, targets)
	before := promValue(t, "message_manager_segment_mismatch_total", map[string]string{"provider": "telnyx"})

	f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, targets, "hello", gomock.Any()).Return([]SendResult{
		result(SendOutcomeAccepted, "t-1", 1), // same as the platform count
		result(SendOutcomeAccepted, "t-2", 2), // different
	})

	f.handler.dispatchSMS(context.Background(), testMessageID, source, targets, "hello")

	if got := promValue(t, "message_manager_segment_mismatch_total", map[string]string{"provider": "telnyx"}) - before; got != 1 {
		t.Errorf("Wrong mismatch count. expect: 1, got: %v", got)
	}
	stored := f.mem.targets()
	if stored[1].Parts != 1 || stored[1].ProviderParts != 2 || stored[0].ProviderParts != 1 {
		t.Errorf("Wrong parts. got: %v", stored)
	}
}

// Test_dispatchMMS_promoted covers a promoted text: a target that is not +1 to +1 skips the mms stages, a refused mms
// target is sent as sms segments, an unknown mms result is never sent as sms, and the indexes of the stored targets are
// kept through the partial slices.
func Test_dispatchMMS_promoted(t *testing.T) {
	source := addr(usSource)

	tests := []struct {
		name string

		source  *commonaddress.Address
		targets []target.Target

		mmsTelnyx    []SendResult // for the targets expectMMS
		expectMMS    []target.Target
		mmsBird      []SendResult
		expectBirdMM []target.Target
		sms          []SendResult
		expectSMS    []target.Target
		smsBirdTo    []target.Target
		smsBird      []SendResult

		expectProvider message.ProviderName
		expectStatus   []target.Status
		expectRefs     []string
	}{
		{
			name:           "a +1 to +1 target accepted as mms never reaches sms",
			source:         source,
			targets:        withParts(testTargets(usDest1), 3),
			expectMMS:      withParts(testTargets(usDest1), 3),
			mmsTelnyx:      []SendResult{result(SendOutcomeAccepted, "m-1", 1)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusQueued},
			expectRefs:     []string{"m-1"},
		},
		{
			name:           "a target outside +1 skips the mms stages and goes to sms",
			source:         source,
			targets:        withParts(testTargets(krDest), 2),
			expectSMS:      withParts(testTargets(krDest), 2),
			sms:            []SendResult{result(SendOutcomeAccepted, "s-1", 2)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusQueued},
			expectRefs:     []string{"s-1"},
		},
		{
			name:           "a source outside +1 skips the mms stages",
			source:         addr("+821000000009"),
			targets:        withParts(testTargets(usDest1), 2),
			expectSMS:      withParts(testTargets(usDest1), 2),
			sms:            []SendResult{result(SendOutcomeAccepted, "s-1", 2)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusQueued},
			expectRefs:     []string{"s-1"},
		},
		{
			name:           "an alphanumeric sender skips the mms stages",
			source:         addr("VOIPBIN"),
			targets:        withParts(testTargets(usDest1), 2),
			expectSMS:      withParts(testTargets(usDest1), 2),
			sms:            []SendResult{result(SendOutcomeAccepted, "s-1", 2)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusQueued},
			expectRefs:     []string{"s-1"},
		},
		{
			name:           "refused by telnyx and accepted by messagebird as mms",
			source:         source,
			targets:        withParts(testTargets(usDest1), 2),
			expectMMS:      withParts(testTargets(usDest1), 2),
			mmsTelnyx:      []SendResult{result(SendOutcomeRejected, "", 0)},
			expectBirdMM:   withParts(testTargets(usDest1), 2),
			mmsBird:        []SendResult{result(SendOutcomeAccepted, "b-1", 0)},
			expectProvider: message.ProviderNameMessagebird,
			expectStatus:   []target.Status{target.StatusQueued},
			expectRefs:     []string{"b-1"},
		},
		{
			name:           "refused by both mms stages falls back to sms segments",
			source:         source,
			targets:        withParts(testTargets(usDest1), 2),
			expectMMS:      withParts(testTargets(usDest1), 2),
			mmsTelnyx:      []SendResult{result(SendOutcomeRejected, "", 0)},
			expectBirdMM:   withParts(testTargets(usDest1), 2),
			mmsBird:        []SendResult{result(SendOutcomeRejected, "", 0)},
			expectSMS:      withParts(testTargets(usDest1), 2),
			sms:            []SendResult{result(SendOutcomeAccepted, "s-1", 2)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusQueued},
			expectRefs:     []string{"s-1"},
		},
		{
			name:           "an unknown mms result at telnyx is not sent as sms",
			source:         source,
			targets:        withParts(testTargets(usDest1), 2),
			expectMMS:      withParts(testTargets(usDest1), 2),
			mmsTelnyx:      []SendResult{result(SendOutcomeFailed, "", 0)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusFailed},
			expectRefs:     []string{""},
		},
		{
			name:           "an unknown mms result at messagebird is not sent as sms",
			source:         source,
			targets:        withParts(testTargets(usDest1), 2),
			expectMMS:      withParts(testTargets(usDest1), 2),
			mmsTelnyx:      []SendResult{result(SendOutcomeRejected, "", 0)},
			expectBirdMM:   withParts(testTargets(usDest1), 2),
			mmsBird:        []SendResult{result(SendOutcomeFailed, "", 0)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusFailed},
			expectRefs:     []string{""},
		},
		{
			name:           "the sms chain refuses too and the target is failed",
			source:         source,
			targets:        withParts(testTargets(krDest), 2),
			expectSMS:      withParts(testTargets(krDest), 2),
			sms:            []SendResult{result(SendOutcomeRejected, "", 0)},
			smsBirdTo:      withParts(testTargets(krDest), 2),
			smsBird:        []SendResult{result(SendOutcomeRejected, "", 0)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusFailed},
			expectRefs:     []string{""},
		},
		{
			name:    "mixed destinations keep the stored indexes through the partial slices",
			source:  source,
			targets: withParts(testTargets(krDest, usDest1, usDest2), 4),
			// +1 to +1 targets (index 1 and 2) go to the mms stage; index 0 is sent as sms from the start.
			expectMMS:      withParts(testTargets(usDest1, usDest2), 4),
			mmsTelnyx:      []SendResult{result(SendOutcomeAccepted, "m-1", 1), result(SendOutcomeRejected, "", 0)},
			expectBirdMM:   withParts(testTargets(usDest2), 4),
			mmsBird:        []SendResult{result(SendOutcomeRejected, "", 0)},
			expectSMS:      withParts(testTargets(krDest, usDest2), 4),
			sms:            []SendResult{result(SendOutcomeAccepted, "s-0", 4), result(SendOutcomeAccepted, "s-2", 4)},
			expectProvider: message.ProviderNameTelnyx,
			expectStatus:   []target.Status{target.StatusQueued, target.StatusQueued, target.StatusQueued},
			expectRefs:     []string{"s-0", "m-1", "s-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newChainFixture(t, tt.targets)

			// the mms providers get the text and no urls.
			if tt.expectMMS != nil {
				f.mmsTelnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, tt.source, tt.expectMMS, "long text", gomock.Nil(), gomock.Any()).Return(tt.mmsTelnyx).Times(1)
			}
			if tt.expectBirdMM != nil {
				f.mmsBird.EXPECT().SendMMS(gomock.Any(), testMessageID, tt.source, tt.expectBirdMM, "long text", gomock.Nil(), gomock.Any()).Return(tt.mmsBird).Times(1)
			}
			if tt.expectSMS != nil {
				f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, tt.source, tt.expectSMS, "long text", gomock.Any()).Return(tt.sms).Times(1)
			}
			if tt.smsBirdTo != nil {
				f.smsBird.EXPECT().SendSMS(gomock.Any(), testMessageID, tt.source, tt.smsBirdTo, "long text", gomock.Any()).Return(tt.smsBird).Times(1)
			}

			f.handler.dispatchMMS(context.Background(), testMessageID, tt.source, tt.targets, "long text", nil, true)

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
			if f.ev.count() != 1 {
				t.Errorf("Wrong event count. expect: 1, got: %d", f.ev.count())
			}
		})
	}
}

// Test_dispatchMMS_attachedIsUnchanged pins that an mms with media (promoted == false) is never handed to the sms chain,
// even when the source and the destination are outside +1: the target is failed.
func Test_dispatchMMS_attachedIsUnchanged(t *testing.T) {
	source := addr(usSource)
	targets := withParts(testTargets(krDest), 1)
	f := newChainFixture(t, targets)

	f.mmsTelnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, source, targets, "hi", []string{"https://example.invalid/a.png"}, gomock.Any()).Return(outs(SendOutcomeRejected))

	f.handler.dispatchMMS(context.Background(), testMessageID, source, targets, "hi", []string{"https://example.invalid/a.png"}, false)

	if got := f.mem.targets(); got[0].Status != target.StatusFailed {
		t.Errorf("Wrong status. expect: failed, got: %v", got[0].Status)
	}
}

// Test_dispatchMMS_promotedMetrics checks the observation counters: promoted_mms_total counts the mms requests only,
// a target that skipped the mms stages does not touch it, and promoted_fallback_total is split by reason and parts.
func Test_dispatchMMS_promotedMetrics(t *testing.T) {
	source := addr(usSource)
	targets := withParts(testTargets(usDest1, krDest), 7)
	f := newChainFixture(t, targets)

	mmsRejected := func() float64 {
		return promValue(t, "message_manager_promoted_mms_total", map[string]string{"provider": "telnyx", "outcome": "rejected"})
	}
	mmsBirdRejected := func() float64 {
		return promValue(t, "message_manager_promoted_mms_total", map[string]string{"provider": "messagebird", "outcome": "rejected"})
	}
	fbIneligible := func() float64 {
		return promValue(t, "message_manager_promoted_fallback_total", map[string]string{"reason": "ineligible", "segments": "7"})
	}
	fbRejected := func() float64 {
		return promValue(t, "message_manager_promoted_fallback_total", map[string]string{"reason": "rejected", "segments": "7"})
	}
	b1, b2, b3, b4 := mmsRejected(), mmsBirdRejected(), fbIneligible(), fbRejected()

	f.mmsTelnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, source, withParts(testTargets(usDest1), 7), "x", gomock.Nil(), gomock.Any()).Return(outs(SendOutcomeRejected))
	f.mmsBird.EXPECT().SendMMS(gomock.Any(), testMessageID, source, withParts(testTargets(usDest1), 7), "x", gomock.Nil(), gomock.Any()).Return(outs(SendOutcomeRejected))
	f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, gomock.Any(), "x", gomock.Any()).Return(outs(SendOutcomeAccepted, SendOutcomeAccepted))

	f.handler.dispatchMMS(context.Background(), testMessageID, source, targets, "x", nil, true)

	if mmsRejected()-b1 != 1 || mmsBirdRejected()-b2 != 1 {
		t.Errorf("Wrong promoted_mms_total. telnyx delta: %v, messagebird delta: %v", mmsRejected()-b1, mmsBirdRejected()-b2)
	}
	if fbIneligible()-b3 != 1 || fbRejected()-b4 != 1 {
		t.Errorf("Wrong promoted_fallback_total. ineligible delta: %v, rejected delta: %v", fbIneligible()-b3, fbRejected()-b4)
	}
}

// Test_dispatch_callbackStoresTheIdAtOnce checks that the callback writes the id of an accepted target while the batch
// is still running, to the right stored index even when the stage slice is a subset, and that a receipt that arrives
// between the callback and the final write is kept by the final write.
func Test_dispatch_callbackStoresTheIdAtOnce(t *testing.T) {
	source := addr(usSource)
	targets := withParts(testTargets(krDest, usDest1, usDest2), 2)
	f := newChainFixture(t, targets)

	// index 1 and 2 are +1 to +1: the mms stage gets the slice [1, 2]; its position 1 is the stored index 2.
	sub := withParts(testTargets(usDest1, usDest2), 2)
	f.mmsTelnyx.EXPECT().SendMMS(gomock.Any(), testMessageID, source, sub, "x", gomock.Nil(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, _ *commonaddress.Address, _ []target.Target, _ string, _ []string, cb ResultCallback) []SendResult {
			r := result(SendOutcomeAccepted, "m-2", 1)
			cb(1, r) // position 1 of the stage slice is the stored index 2

			stored := f.mem.targets()
			if stored[2].ProviderReferenceID != "m-2" || stored[1].ProviderReferenceID != "" || stored[0].ProviderReferenceID != "" {
				t.Errorf("The id was not stored at the right index before the batch ended. got: %v", refsOf(stored))
			}

			// a receipt arrives for the target before the batch ends
			f.mem.setTargets(func(ts []target.Target) { ts[2].Status = target.StatusDelivered })

			return []SendResult{result(SendOutcomeRejected, "", 0), r}
		})
	f.mmsBird.EXPECT().SendMMS(gomock.Any(), testMessageID, source, withParts(testTargets(usDest1), 2), "x", gomock.Nil(), gomock.Any()).Return(outs(SendOutcomeAccepted))
	f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, withParts(testTargets(krDest), 2), "x", gomock.Any()).Return([]SendResult{result(SendOutcomeAccepted, "s-0", 2)})

	f.handler.dispatchMMS(context.Background(), testMessageID, source, targets, "x", nil, true)

	got := f.mem.targets()
	if got[2].Status != target.StatusDelivered {
		t.Errorf("The receipt was lowered by the final write. got: %v", statusesOf(got))
	}
	if got[0].ProviderReferenceID != "s-0" || got[2].ProviderReferenceID != "m-2" {
		t.Errorf("Wrong ids. got: %v", refsOf(got))
	}
	if f.ev.count() != 1 {
		t.Errorf("Wrong event count. expect one event for the batch, got: %d", f.ev.count())
	}
}

// Test_dispatch_callbackCircuitBreaker checks that three consecutive failed callback writes stop the callbacks and that
// the final write still stores the ids.
func Test_dispatch_callbackCircuitBreaker(t *testing.T) {
	source := addr(usSource)
	targets := withParts(testTargets(usDest1, usDest2, krDest, "+15550100005", "+15550100006"), 1)
	f := newChainFixture(t, targets)
	f.mem.failIfErr = errors.New("db down")

	f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, targets, "x", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, _ *commonaddress.Address, ts []target.Target, _ string, cb ResultCallback) []SendResult {
			out := make([]SendResult, len(ts))
			for i := range ts {
				out[i] = result(SendOutcomeAccepted, "t-"+string(rune('0'+i)), 1)
				cb(i, out[i])
			}
			return out
		})

	f.handler.dispatchSMS(context.Background(), testMessageID, source, targets, "x")

	// 3 failed callbacks, the others are skipped, then the final write tries once before the merge.
	f.mem.mu.Lock()
	ifCalls := f.mem.ifCalls
	f.mem.mu.Unlock()
	if ifCalls > 4 {
		t.Errorf("The callbacks were not stopped. conditional writes: %d", ifCalls)
	}
	if f.mem.mergeCalls != 1 {
		t.Errorf("Wrong merge calls. expect: 1, got: %d", f.mem.mergeCalls)
	}
	if !reflect.DeepEqual(refsOf(f.mem.targets()), []string{"t-0", "t-1", "t-2", "t-3", "t-4"}) {
		t.Errorf("The final write did not store the ids. got: %v", refsOf(f.mem.targets()))
	}
}

// Test_dispatch_finalWrite covers the conflict handling of the final write.
func Test_dispatch_finalWrite(t *testing.T) {
	source := addr(usSource)

	t.Run("a receipt that wins the race is kept", func(t *testing.T) {
		targets := withParts(testTargets(usDest1), 1)
		f := newChainFixture(t, targets)
		f.mem.beforeIf = func(call int) {
			if call == 1 {
				f.mem.setTargets(func(ts []target.Target) {
					ts[0].ProviderReferenceID = "t-1"
					ts[0].Status = target.StatusDelivered
				})
			}
		}
		f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, targets, "x", gomock.Any()).Return([]SendResult{result(SendOutcomeAccepted, "t-1", 1)})

		f.handler.dispatchSMS(context.Background(), testMessageID, source, targets, "x")

		got := f.mem.targets()
		if got[0].Status != target.StatusDelivered || got[0].ProviderReferenceID != "t-1" {
			t.Errorf("Wrong target. got: %v", got)
		}
		if f.mem.ifCalls != 2 || f.mem.mergeCalls != 0 {
			t.Errorf("Wrong calls. expect 2 conditional writes and no merge, got: %d and %d", f.mem.ifCalls, f.mem.mergeCalls)
		}
	})

	t.Run("a failed mark never lowers a target that already moved on", func(t *testing.T) {
		targets := withParts(testTargets(usDest1), 1)
		f := newChainFixture(t, targets)
		f.mem.setTargets(func(ts []target.Target) { ts[0].Status = target.StatusSent })
		f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, targets, "x", gomock.Any()).Return(outs(SendOutcomeFailed))

		f.handler.dispatchSMS(context.Background(), testMessageID, source, targets, "x")

		if got := f.mem.targets(); got[0].Status != target.StatusSent {
			t.Errorf("Wrong status. expect: sent, got: %v", got[0].Status)
		}
	})

	t.Run("exhausted compare and set is merged in a fresh context", func(t *testing.T) {
		oldCAS, oldMerge := finalWriteCASBudget, finalWriteMergeBudget
		finalWriteCASBudget, finalWriteMergeBudget = 60*time.Millisecond, 5*time.Second
		t.Cleanup(func() { finalWriteCASBudget, finalWriteMergeBudget = oldCAS, oldMerge })

		targets := withParts(testTargets(usDest1), 1)
		f := newChainFixture(t, targets)
		f.mem.forceConflicts = 1 << 30
		f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, targets, "x", gomock.Any()).Return([]SendResult{result(SendOutcomeAccepted, "t-1", 1)})

		f.handler.dispatchSMS(context.Background(), testMessageID, source, targets, "x")

		if f.mem.mergeCalls != 1 {
			t.Errorf("Wrong merge calls. expect: 1, got: %d", f.mem.mergeCalls)
		}
		if got := f.mem.targets(); got[0].ProviderReferenceID != "t-1" {
			t.Errorf("The merge did not store the result. got: %v", got)
		}
		if f.ev.count() != 1 {
			t.Errorf("Wrong event count. expect: 1, got: %d", f.ev.count())
		}
	})

	t.Run("stored targets of another shape are refused and counted", func(t *testing.T) {
		targets := withParts(testTargets(usDest1), 1)
		f := newChainFixture(t, withParts(testTargets(usDest1, usDest2), 1)) // the stored row has two targets
		before := promValue(t, "message_manager_dispatch_final_write_failed_total", nil)
		f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, targets, "x", gomock.Any()).Return([]SendResult{result(SendOutcomeAccepted, "t-1", 1)})

		f.handler.dispatchSMS(context.Background(), testMessageID, source, targets, "x")

		if f.mem.writes() != 0 {
			t.Errorf("Nothing may be written. writes: %d", f.mem.writes())
		}
		if got := promValue(t, "message_manager_dispatch_final_write_failed_total", nil) - before; got != 1 {
			t.Errorf("Wrong failed final write count. expect: 1, got: %v", got)
		}
		if f.ev.count() != 0 {
			t.Errorf("Wrong event count. expect: 0, got: %d", f.ev.count())
		}
	})

	t.Run("the event context is alive after the dispatch context ended", func(t *testing.T) {
		targets := withParts(testTargets(usDest1), 1)
		f := newChainFixture(t, targets)
		ctx, cancel := context.WithCancel(context.Background())
		f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, targets, "x", gomock.Any()).DoAndReturn(
			func(_ context.Context, _ uuid.UUID, _ *commonaddress.Address, _ []target.Target, _ string, _ ResultCallback) []SendResult {
				cancel()
				return []SendResult{result(SendOutcomeAccepted, "t-1", 1)}
			})

		f.handler.dispatchSMS(ctx, testMessageID, source, targets, "x")

		f.ev.mu.Lock()
		defer f.ev.mu.Unlock()
		if len(f.ev.ctxErr) != 1 || f.ev.ctxErr[0] != nil {
			t.Errorf("The context of the event must not be canceled. got: %v", f.ev.ctxErr)
		}
	})

	t.Run("the event body has no provider keys while the row keeps them", func(t *testing.T) {
		targets := withParts(testTargets(usDest1), 1)
		f := newChainFixture(t, targets)
		f.smsTelnyx.EXPECT().SendSMS(gomock.Any(), testMessageID, source, targets, "x", gomock.Any()).Return([]SendResult{result(SendOutcomeAccepted, "t-1", 3)})

		f.handler.dispatchSMS(context.Background(), testMessageID, source, targets, "x")

		if got := f.mem.targets(); got[0].ProviderReferenceID != "t-1" || got[0].ProviderParts != 3 {
			t.Errorf("The row lost the provider keys. got: %v", got)
		}
		f.ev.mu.Lock()
		defer f.ev.mu.Unlock()
		if len(f.ev.bodies) != 1 || f.ev.bodies[0].Targets[0].ProviderReferenceID != "" || f.ev.bodies[0].Targets[0].ProviderParts != 0 {
			t.Errorf("The event leaks the provider keys. got: %v", f.ev.bodies)
		}
	})
}
