package messagehandler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/messagebird"
	"monorepo/bin-message-manager/models/telnyx"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

func Test_mmsTelnyx_SendMMS(t *testing.T) {
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555****0001"}
	targets := testTargets("+1555****0002", "+1555****0003", "+1555****0004", "+1555****0005")
	urls := []string{"https://example.invalid/a.png", "https://example.invalid/b.pdf"}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExternal := requestexternal.NewMockRequestExternal(mc)
	h := NewMMSTelnyx(mockExternal)

	// each target gets its own request with all the urls.
	mockExternal.EXPECT().TelnyxSendMMS(gomock.Any(), source.Target, targets[0].Destination.Target, "hello", urls).Return(&telnyx.MessageResponse{}, nil)
	mockExternal.EXPECT().TelnyxSendMMS(gomock.Any(), source.Target, targets[1].Destination.Target, "hello", urls).Return(nil, &requestexternal.ProviderError{Provider: "telnyx", StatusCode: 422})
	mockExternal.EXPECT().TelnyxSendMMS(gomock.Any(), source.Target, targets[2].Destination.Target, "hello", urls).Return(nil, &requestexternal.ProviderError{Provider: "telnyx", StatusCode: 503})
	mockExternal.EXPECT().TelnyxSendMMS(gomock.Any(), source.Target, targets[3].Destination.Target, "hello", urls).Return(nil, errors.New("transport"))

	res := h.SendMMS(context.Background(), uuid.Nil, source, targets, "hello", urls)

	expect := []MMSOutcome{MMSOutcomeAccepted, MMSOutcomeRejected, MMSOutcomeFailed, MMSOutcomeFailed}
	if fmt.Sprint(res) != fmt.Sprint(expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, res)
	}
}

func Test_mmsMessagebird_SendMMS(t *testing.T) {
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555****0001"}
	targets := testTargets("+1555****0002", "+1555****0003", "+1555****0004")
	urls := []string{"https://example.invalid/a.png"}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExternal := requestexternal.NewMockRequestExternal(mc)
	h := NewMMSMessagebird(mockExternal)

	mockExternal.EXPECT().MessagebirdSendMMS(gomock.Any(), source.Target, targets[0].Destination.Target, "", urls).Return(&messagebird.Message{}, nil)
	mockExternal.EXPECT().MessagebirdSendMMS(gomock.Any(), source.Target, targets[1].Destination.Target, "", urls).Return(nil, &requestexternal.ProviderError{Provider: "messagebird", StatusCode: 403})
	mockExternal.EXPECT().MessagebirdSendMMS(gomock.Any(), source.Target, targets[2].Destination.Target, "", urls).Return(nil, errors.New("transport"))

	res := h.SendMMS(context.Background(), uuid.Nil, source, targets, "", urls)

	expect := []MMSOutcome{MMSOutcomeAccepted, MMSOutcomeRejected, MMSOutcomeFailed}
	if fmt.Sprint(res) != fmt.Sprint(expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, res)
	}
}

// Test_sendMMSTargets_concurrencyAndOrder checks that no more than mmsProviderConcurrency requests are in
// flight, that every target is attempted once, and that the outcomes keep the order of the targets.
func Test_sendMMSTargets_concurrencyAndOrder(t *testing.T) {
	const count = 45

	var inFlight, peak int32
	var calls int32
	var mu sync.Mutex
	seen := map[int]int{}

	res := sendMMSTargets(context.Background(), count, func(_ context.Context, i int) MMSOutcome {
		cur := atomic.AddInt32(&inFlight, 1)
		for {
			p := atomic.LoadInt32(&peak)
			if cur <= p || atomic.CompareAndSwapInt32(&peak, p, cur) {
				break
			}
		}
		time.Sleep(2 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		atomic.AddInt32(&calls, 1)

		mu.Lock()
		seen[i]++
		mu.Unlock()

		if i%2 == 0 {
			return MMSOutcomeAccepted
		}
		return MMSOutcomeRejected
	})

	if peak > 10 || peak < 2 {
		t.Errorf("Wrong concurrency. expect between 2 and 10 in flight, got: %d", peak)
	}
	if int(calls) != count || len(seen) != count {
		t.Errorf("Wrong calls. expect every target once, calls: %d, distinct: %d", calls, len(seen))
	}
	for i, o := range res {
		expect := MMSOutcomeAccepted
		if i%2 != 0 {
			expect = MMSOutcomeRejected
		}
		if o != expect {
			t.Errorf("Wrong outcome order at %d. expect: %v, got: %v", i, expect, o)
		}
	}
}

// Test_sendMMSTargets_notAttempted checks that a target whose turn does not come before the context ends
// is not attempted and stays failed, so it can never become a fallback candidate.
func Test_sendMMSTargets_notAttempted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	release := make(chan struct{})

	var attempted int32
	done := make(chan []MMSOutcome)
	go func() {
		done <- sendMMSTargets(ctx, 15, func(_ context.Context, i int) MMSOutcome {
			atomic.AddInt32(&attempted, 1)
			<-release
			return MMSOutcomeAccepted
		})
	}()

	// wait until all the slots are taken, then end the context and let the running requests finish.
	for atomic.LoadInt32(&attempted) < 10 {
		time.Sleep(time.Millisecond)
	}
	cancel()
	time.Sleep(10 * time.Millisecond)
	close(release)

	res := <-done
	accepted, failed := 0, 0
	for _, o := range res {
		switch o {
		case MMSOutcomeAccepted:
			accepted++
		case MMSOutcomeFailed:
			failed++
		}
	}
	if accepted != 10 || failed != 5 {
		t.Errorf("Wrong match. expect accepted: 10, failed: 5, got accepted: %d, failed: %d", accepted, failed)
	}
	if int(attempted) != 10 {
		t.Errorf("Wrong match. targets that were not attempted must not reach the provider, attempted: %d", attempted)
	}
}
