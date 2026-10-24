package listenhandler

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/voip-kamailio-proxy/pkg/siphandler"
)

const (
	testQueuePermanent = "test.permanent.request"
	testQueueVolatile  = "test.volatile.request"
)

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	sentinel := &siphandler.HealthCheckResult{Healthy: true, ResponseCode: "sentinel"}
	checker := func(_ context.Context, _ string, _ time.Duration) (*siphandler.HealthCheckResult, error) {
		return sentinel, nil
	}

	res := NewListenHandler(mockSock, testQueuePermanent, testQueueVolatile, 3*time.Second, checker)

	h, ok := res.(*listenHandler)
	if !ok {
		t.Fatalf("Expected *listenHandler, got %T", res)
	}
	if h.sockHandler != mockSock {
		t.Errorf("sockHandler was not preserved")
	}
	if h.rabbitQueueListenPermanent != testQueuePermanent {
		t.Errorf("Expected permanent queue %q, got %q", testQueuePermanent, h.rabbitQueueListenPermanent)
	}
	if h.rabbitQueueListenVolatile != testQueueVolatile {
		t.Errorf("Expected volatile queue %q, got %q", testQueueVolatile, h.rabbitQueueListenVolatile)
	}
	if h.sipTimeout != 3*time.Second {
		t.Errorf("Expected timeout 3s, got %v", h.sipTimeout)
	}

	// func values are not comparable, so call the stored checker.
	got, err := h.sipChecker(context.Background(), "host.invalid", time.Second)
	if err != nil || got != sentinel {
		t.Errorf("sipChecker was not preserved. got: %v, err: %v", got, err)
	}
}

// waitConsumers waits for the ConsumeRPC goroutines with an upper bound so a
// broken expectation fails the test instead of hanging it.
func waitConsumers(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("Timed out waiting for the ConsumeRPC calls")
	}
}

// consumeCapture collects the consume callbacks handed to ConsumeRPC by the
// background goroutines. The callbacks are exposed only after wg.Wait.
type consumeCapture struct {
	mu        sync.Mutex
	callbacks map[string]sock.CbMsgRPC
}

func (c *consumeCapture) set(queue string, cb sock.CbMsgRPC) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.callbacks[queue] = cb
}

func newTestListenHandler(mockSock sockhandler.SockHandler) *listenHandler {
	return &listenHandler{
		sockHandler:                mockSock,
		rabbitQueueListenPermanent: testQueuePermanent,
		rabbitQueueListenVolatile:  testQueueVolatile,
		sipTimeout:                 3 * time.Second,
		sipChecker:                 stubHealthyChecker,
	}
}

// expectConsume registers the ConsumeRPC expectation for one queue. The
// returned error is what the mocked ConsumeRPC returns.
func expectConsume(mockSock *sockhandler.MockSockHandler, wg *sync.WaitGroup, capture *consumeCapture, queue string, retErr error) {
	wg.Add(1)
	mockSock.EXPECT().ConsumeRPC(
		gomock.Any(), queue, "kamailio-proxy", false, false, false, 10, gomock.Any(),
	).DoAndReturn(func(_ context.Context, q string, _ string, _ bool, _ bool, _ bool, _ int, cb sock.CbMsgRPC) error {
		capture.set(q, cb)
		wg.Done()
		return retErr
	})
}

func Test_listenRun(t *testing.T) {
	tests := []struct {
		name       string
		consumeErr error
	}{
		{
			name:       "both queues created and consumed",
			consumeErr: nil,
		},
		{
			name:       "consume error does not stop the other queue",
			consumeErr: fmt.Errorf("consume failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := newTestListenHandler(mockSock)

			var wg sync.WaitGroup
			capture := &consumeCapture{callbacks: map[string]sock.CbMsgRPC{}}

			mockSock.EXPECT().QueueCreate(testQueuePermanent, "normal").Return(nil)
			mockSock.EXPECT().QueueCreate(testQueueVolatile, "volatile").Return(nil)
			expectConsume(mockSock, &wg, capture, testQueuePermanent, tt.consumeErr)
			expectConsume(mockSock, &wg, capture, testQueueVolatile, tt.consumeErr)

			if err := h.listenRun(); err != nil {
				t.Fatalf("Expected nil error, got: %v", err)
			}
			waitConsumers(t, &wg)

			// Each queue must route to processRequest.
			for _, q := range []string{testQueuePermanent, testQueueVolatile} {
				capture.mu.Lock()
				cb := capture.callbacks[q]
				capture.mu.Unlock()
				if cb == nil {
					t.Fatalf("No consume callback captured for queue %q", q)
				}

				res, err := cb(&sock.Request{
					URI:    "/v1/providers/health",
					Method: sock.RequestMethodPost,
					Data:   []byte(`{"hostname": "192.0.2.10"}`),
				})
				if err != nil {
					t.Errorf("queue %q: expected nil error, got: %v", q, err)
				}
				if res == nil || res.StatusCode != 200 {
					t.Errorf("queue %q: expected status 200 from the health route, got: %v", q, res)
				}

				res, err = cb(&sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet})
				if err != nil || res == nil || res.StatusCode != 404 {
					t.Errorf("queue %q: expected 404 for unknown route, got: %v, err: %v", q, res, err)
				}
			}
		})
	}
}

func Test_listenRun_queueCreateError(t *testing.T) {
	tests := []struct {
		name  string
		setup func(mockSock *sockhandler.MockSockHandler)
	}{
		{
			name: "permanent queue creation fails",
			setup: func(mockSock *sockhandler.MockSockHandler) {
				// no volatile creation and no consumers are expected after this failure.
				mockSock.EXPECT().QueueCreate(testQueuePermanent, "normal").Return(fmt.Errorf("declare failed"))
			},
		},
		{
			name: "volatile queue creation fails",
			setup: func(mockSock *sockhandler.MockSockHandler) {
				mockSock.EXPECT().QueueCreate(testQueuePermanent, "normal").Return(nil)
				mockSock.EXPECT().QueueCreate(testQueueVolatile, "volatile").Return(fmt.Errorf("declare failed"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := newTestListenHandler(mockSock)
			tt.setup(mockSock)

			if err := h.listenRun(); err == nil {
				t.Errorf("Expected an error, got nil")
			}
		})
	}
}

func Test_Run(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := newTestListenHandler(mockSock)

	var wg sync.WaitGroup
	capture := &consumeCapture{callbacks: map[string]sock.CbMsgRPC{}}

	// QueueCreate runs before ConsumeRPC in the background goroutine, so
	// waiting for both ConsumeRPC calls also covers both QueueCreate calls.
	mockSock.EXPECT().QueueCreate(testQueuePermanent, "normal").Return(nil)
	mockSock.EXPECT().QueueCreate(testQueueVolatile, "volatile").Return(nil)
	expectConsume(mockSock, &wg, capture, testQueuePermanent, nil)
	expectConsume(mockSock, &wg, capture, testQueueVolatile, nil)

	if err := h.Run(); err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}
	waitConsumers(t, &wg)

	capture.mu.Lock()
	defer capture.mu.Unlock()
	if len(capture.callbacks) != 2 {
		t.Errorf("Expected consumers on 2 queues, got %d", len(capture.callbacks))
	}
}
