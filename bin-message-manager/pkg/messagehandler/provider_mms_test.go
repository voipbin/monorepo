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
	mockExternal.EXPECT().TelnyxSendMMS(gomock.Any(), source.Target, targets[0].Destination.Target, "hello", urls).Return(&telnyx.MessageResponse{Data: telnyx.MessageData{ID: "ref-1", Parts: 1}}, nil)
	mockExternal.EXPECT().TelnyxSendMMS(gomock.Any(), source.Target, targets[1].Destination.Target, "hello", urls).Return(nil, &requestexternal.ProviderError{Provider: "telnyx", StatusCode: 422})
	mockExternal.EXPECT().TelnyxSendMMS(gomock.Any(), source.Target, targets[2].Destination.Target, "hello", urls).Return(nil, &requestexternal.ProviderError{Provider: "telnyx", StatusCode: 503})
	mockExternal.EXPECT().TelnyxSendMMS(gomock.Any(), source.Target, targets[3].Destination.Target, "hello", urls).Return(nil, errors.New("transport"))

	res := h.SendMMS(context.Background(), uuid.Nil, source, targets, "hello", urls, nil)

	expect := []SendOutcome{SendOutcomeAccepted, SendOutcomeRejected, SendOutcomeFailed, SendOutcomeFailed}
	if fmt.Sprint(outcomesOf(res)) != fmt.Sprint(expect) {
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

	mockExternal.EXPECT().MessagebirdSendMMS(gomock.Any(), source.Target, targets[0].Destination.Target, "", urls).Return(&messagebird.Message{ID: "mb-1"}, nil)
	mockExternal.EXPECT().MessagebirdSendMMS(gomock.Any(), source.Target, targets[1].Destination.Target, "", urls).Return(nil, &requestexternal.ProviderError{Provider: "messagebird", StatusCode: 403})
	mockExternal.EXPECT().MessagebirdSendMMS(gomock.Any(), source.Target, targets[2].Destination.Target, "", urls).Return(nil, errors.New("transport"))

	res := h.SendMMS(context.Background(), uuid.Nil, source, targets, "", urls, nil)

	expect := []SendOutcome{SendOutcomeAccepted, SendOutcomeRejected, SendOutcomeFailed}
	if fmt.Sprint(outcomesOf(res)) != fmt.Sprint(expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, res)
	}
}

// Test_sendTargets_concurrencyAndOrder checks that no more than mmsProviderConcurrency requests are in
// flight, that every target is attempted once, and that the outcomes keep the order of the targets.
func Test_sendTargets_concurrencyAndOrder(t *testing.T) {
	const count = 45

	var inFlight, peak int32
	var calls int32
	var mu sync.Mutex
	seen := map[int]int{}

	res := sendTargets(context.Background(), count, func(_ context.Context, i int) SendResult {
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
			return SendResult{Outcome: SendOutcomeAccepted}
		}
		return SendResult{Outcome: SendOutcomeRejected}
	}, nil)

	if peak > 10 || peak < 2 {
		t.Errorf("Wrong concurrency. expect between 2 and 10 in flight, got: %d", peak)
	}
	if int(calls) != count || len(seen) != count {
		t.Errorf("Wrong calls. expect every target once, calls: %d, distinct: %d", calls, len(seen))
	}
	for i, o := range res {
		expect := SendOutcomeAccepted
		if i%2 != 0 {
			expect = SendOutcomeRejected
		}
		if o.Outcome != expect {
			t.Errorf("Wrong outcome order at %d. expect: %v, got: %v", i, expect, o)
		}
	}
}

// Test_sendTargets_notAttempted checks that a target whose turn does not come before the context ends
// is not attempted and stays failed, so it can never become a fallback candidate.
func Test_sendTargets_notAttempted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	release := make(chan struct{})

	var attempted int32
	done := make(chan []SendResult)
	go func() {
		done <- sendTargets(ctx, 15, func(_ context.Context, i int) SendResult {
			atomic.AddInt32(&attempted, 1)
			<-release
			return SendResult{Outcome: SendOutcomeAccepted}
		}, nil)
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
		switch o.Outcome {
		case SendOutcomeAccepted:
			accepted++
		case SendOutcomeFailed:
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

func outcomesOf(results []SendResult) []SendOutcome {
	res := make([]SendOutcome, 0, len(results))
	for _, r := range results {
		res = append(res, r.Outcome)
	}

	return res
}

// Test_mmsProviders_idAndCallback checks that the provider id and parts are returned and that the callback is called
// once for each attempted target with the index of the slice it was given.
func Test_mmsProviders_idAndCallback(t *testing.T) {
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555****0001"}
	targets := testTargets("+1555****0002", "+1555****0003")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExternal := requestexternal.NewMockRequestExternal(mc)
	mockExternal.EXPECT().TelnyxSendMMS(gomock.Any(), source.Target, targets[0].Destination.Target, "hi", gomock.Nil()).Return(&telnyx.MessageResponse{Data: telnyx.MessageData{ID: "t-0", Parts: 1}}, nil)
	mockExternal.EXPECT().TelnyxSendMMS(gomock.Any(), source.Target, targets[1].Destination.Target, "hi", gomock.Nil()).Return(nil, &requestexternal.ProviderError{Provider: "telnyx", StatusCode: 422})

	var mu sync.Mutex
	got := map[int]SendResult{}
	res := NewMMSTelnyx(mockExternal).SendMMS(context.Background(), uuid.Nil, source, targets, "hi", nil, func(i int, r SendResult) {
		mu.Lock()
		defer mu.Unlock()
		got[i] = r
	})

	if res[0].ProviderReferenceID != "t-0" || res[0].ProviderParts != 1 || res[1].Outcome != SendOutcomeRejected {
		t.Errorf("Wrong results. got: %v", res)
	}
	if len(got) != 2 || got[0].ProviderReferenceID != "t-0" || got[1].Outcome != SendOutcomeRejected {
		t.Errorf("Wrong callback calls. got: %v", got)
	}
}

func Test_smsTelnyx_SendSMS(t *testing.T) {
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555****0001"}
	targets := testTargets("+1555****0002", "+1555****0003", "+1555****0004", "+1555****0005")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExternal := requestexternal.NewMockRequestExternal(mc)
	mockExternal.EXPECT().TelnyxSendMessage(gomock.Any(), source.Target, targets[0].Destination.Target, "hello").Return(&telnyx.MessageResponse{Data: telnyx.MessageData{ID: "t-0", Parts: 2}}, nil)
	mockExternal.EXPECT().TelnyxSendMessage(gomock.Any(), source.Target, targets[1].Destination.Target, "hello").Return(nil, &requestexternal.ProviderError{Provider: "telnyx", StatusCode: 400, Detail: "code 40301"})
	mockExternal.EXPECT().TelnyxSendMessage(gomock.Any(), source.Target, targets[2].Destination.Target, "hello").Return(nil, &requestexternal.ProviderError{Provider: "telnyx", StatusCode: 408})
	mockExternal.EXPECT().TelnyxSendMessage(gomock.Any(), source.Target, targets[3].Destination.Target, "hello").Return(nil, errors.New("transport"))

	res := NewSMSTelnyx(mockExternal).SendSMS(context.Background(), uuid.Nil, source, targets, "hello", nil)

	expect := []SendOutcome{SendOutcomeAccepted, SendOutcomeRejected, SendOutcomeFailed, SendOutcomeFailed}
	if fmt.Sprint(outcomesOf(res)) != fmt.Sprint(expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, res)
	}
	if res[0].ProviderReferenceID != "t-0" || res[0].ProviderParts != 2 {
		t.Errorf("Wrong id or parts. got: %v", res[0])
	}
}

func Test_smsMessagebird_SendSMS(t *testing.T) {
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555****0001"}
	targets := testTargets("+15550000002", "+15550000003", "+15550000004")

	t.Run("recipients are matched by digits and a missing one is accepted as queued", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockExternal := requestexternal.NewMockRequestExternal(mc)
		mockExternal.EXPECT().MessagebirdSendMessage(gomock.Any(), source.Target, []string{"+15550000002", "+15550000003", "+15550000004"}, "hello").Return(&messagebird.Message{
			ID: "mb-1",
			Recipients: messagebird.RecipientStruct{Items: []messagebird.Recipient{
				{Recipient: 15550000003, Status: "sent", MessagePartCount: 3},
				{Recipient: 15550000002, Status: "scheduled", MessagePartCount: 1},
			}},
		}, nil)

		res := NewSMSMessagebird(mockExternal).SendSMS(context.Background(), uuid.Nil, source, targets, "hello", nil)
		if len(res) != 3 {
			t.Fatalf("Wrong count. got: %d", len(res))
		}
		if res[0].Status != "scheduled" || res[0].ProviderParts != 1 || res[1].Status != "sent" || res[1].ProviderParts != 3 {
			t.Errorf("Wrong recipient match. got: %v", res)
		}
		if res[2].Outcome != SendOutcomeAccepted || res[2].Status != "" || res[2].ProviderParts != 0 || res[2].ProviderReferenceID != "mb-1" {
			t.Errorf("Wrong missing recipient. got: %v", res[2])
		}
	})

	t.Run("the whole request has one outcome", func(t *testing.T) {
		tests := []struct {
			name      string
			err       error
			expectOut SendOutcome
		}{
			{"422 is rejected", &requestexternal.ProviderError{Provider: "messagebird", StatusCode: 422, Detail: "code 9"}, SendOutcomeRejected},
			{"408 is unknown", &requestexternal.ProviderError{Provider: "messagebird", StatusCode: 408}, SendOutcomeFailed},
			{"500 is unknown", &requestexternal.ProviderError{Provider: "messagebird", StatusCode: 500}, SendOutcomeFailed},
			{"transport is unknown", errors.New("timeout"), SendOutcomeFailed},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				mockExternal := requestexternal.NewMockRequestExternal(mc)
				mockExternal.EXPECT().MessagebirdSendMessage(gomock.Any(), source.Target, gomock.Any(), "hello").Return(nil, tt.err)

				res := NewSMSMessagebird(mockExternal).SendSMS(context.Background(), uuid.Nil, source, targets, "hello", nil)
				for i, r := range res {
					if r.Outcome != tt.expectOut {
						t.Errorf("Wrong outcome at %d. expect: %v, got: %v", i, tt.expectOut, r.Outcome)
					}
				}
			})
		}
	})
}
