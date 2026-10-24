package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/voip-rtpengine-proxy/pkg/ngclient"
	"monorepo/voip-rtpengine-proxy/pkg/processmanager"
)

const testWait = 5 * time.Second

type consumed struct {
	queue string
	cb    sock.CbMsgRPC
}

// expectConsume registers a ConsumeRPC expectation that checks the fixed
// arguments and reports the queue and callback it was called with.
func expectConsume(t *testing.T, mockSock *sockhandler.MockSockHandler, queue string, out chan<- consumed) {
	t.Helper()
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, "rtpengine-proxy", false, false, false, 10, gomock.Any()).DoAndReturn(
		func(ctx context.Context, q, _ string, _, _, _ bool, _ int, cb sock.CbMsgRPC) error {
			if ctx == nil {
				t.Errorf("Wrong match. ConsumeRPC ctx is nil")
			}
			out <- consumed{queue: q, cb: cb}
			return nil
		})
}

func receiveConsumed(t *testing.T, ch <-chan consumed) consumed {
	t.Helper()
	select {
	case c := <-ch:
		return c
	case <-time.After(testWait):
		t.Fatalf("timed out waiting for ConsumeRPC")
		return consumed{}
	}
}

func Test_listenRun(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockNG := ngclient.NewMockNGClient(mc)
	mockProc := processmanager.NewMockProcessManager(mc)

	lh := NewListenHandler(mockSock, "perm-q", "vol-q", mockNG, mockProc)
	h, ok := lh.(*listenHandler)
	if !ok {
		t.Fatalf("Wrong match. NewListenHandler returned %T", lh)
	}

	gomock.InOrder(
		mockSock.EXPECT().QueueCreate("perm-q", "normal").Return(nil),
		mockSock.EXPECT().QueueCreate("vol-q", "volatile").Return(nil),
	)
	ch := make(chan consumed, 2)
	expectConsume(t, mockSock, "perm-q", ch)
	expectConsume(t, mockSock, "vol-q", ch)

	if err := h.listenRun(); err != nil {
		t.Fatalf("Wrong match. expected: ok, got: %v", err)
	}

	got := map[string]sock.CbMsgRPC{}
	for i := 0; i < 2; i++ {
		c := receiveConsumed(t, ch)
		got[c.queue] = c.cb
	}
	for _, q := range []string{"perm-q", "vol-q"} {
		cb, found := got[q]
		if !found || cb == nil {
			t.Fatalf("Wrong match. queue %q has no consumer callback", q)
		}

		// the callback must be the request processor: an empty command body is rejected with 400
		resp, err := cb(&sock.Request{URI: "/v1/commands", Method: sock.RequestMethodPost})
		if err != nil {
			t.Fatalf("Wrong match. queue %q callback error: %v", q, err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("Wrong match. queue %q callback expected: 400, got: %d", q, resp.StatusCode)
		}
	}
}

func Test_listenRun_queueCreateError(t *testing.T) {
	tests := []struct {
		name string

		failSecond bool
	}{
		{
			name:       "first queue create fails, no consumer is started",
			failSecond: false,
		},
		{
			name:       "second queue create fails after the first consumer started",
			failSecond: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := &listenHandler{
				sockHandler:                       mockSock,
				rabbitQueueListenRequestPermanent: "perm-q",
				rabbitQueueListenRequestVolatile:  "vol-q",
			}

			queueErr := errors.New("broker down")
			if !tt.failSecond {
				// no ConsumeRPC and no second QueueCreate expectation: any further call fails the test
				mockSock.EXPECT().QueueCreate("perm-q", "normal").Return(queueErr)
			} else {
				ch := make(chan consumed, 1)
				gomock.InOrder(
					mockSock.EXPECT().QueueCreate("perm-q", "normal").Return(nil),
					mockSock.EXPECT().QueueCreate("vol-q", "volatile").Return(queueErr),
				)
				expectConsume(t, mockSock, "perm-q", ch)
				defer func() {
					if c := receiveConsumed(t, ch); c.queue != "perm-q" {
						t.Errorf("Wrong match. expected: perm-q, got: %s", c.queue)
					}
				}()
			}

			err := h.listenRun()
			if err == nil {
				t.Fatalf("Wrong match. expected: error, got: nil")
			}
			if !errors.Is(err, queueErr) {
				t.Errorf("Wrong match. expected wrapped: %v, got: %v", queueErr, err)
			}
		})
	}
}

func Test_Run(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	called := make(chan struct{}, 1)

	// the first queue creation fails, so the background goroutine stops without any further mock call
	mockSock.EXPECT().QueueCreate("perm-q", "normal").DoAndReturn(func(string, string) error {
		called <- struct{}{}
		return errors.New("broker down")
	})

	h := NewListenHandler(mockSock, "perm-q", "vol-q", ngclient.NewMockNGClient(mc), processmanager.NewMockProcessManager(mc))
	if err := h.Run(); err != nil {
		t.Fatalf("Wrong match. expected: ok, got: %v", err)
	}

	select {
	case <-called:
	case <-time.After(testWait):
		t.Fatalf("timed out waiting for QueueCreate")
	}
}

func Test_processRequest_commandsPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &listenHandler{
		ngClient: ngclient.NewMockNGClient(mc),
		procMgr:  processmanager.NewMockProcessManager(mc),
	}

	// POST /v1/commands is routed to the command processor; a body without a type is a 400
	resp, err := h.processRequest(&sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"command":"query"}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. expected: ok, got: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("Wrong match. expected: 400, got: %d", resp.StatusCode)
	}
}

func Test_processNG_marshalFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNG := ngclient.NewMockNGClient(mc)
	// a func value cannot be marshaled to JSON
	mockNG.EXPECT().Send(gomock.Any()).Return(map[string]interface{}{"result": func() {}}, nil)

	h := &listenHandler{
		ngClient: mockNG,
		procMgr:  processmanager.NewMockProcessManager(mc),
	}

	resp, err := h.processCommandPost(&sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"ng","command":"query","call-id":"abc123"}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. expected: ok, got: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("Wrong match. expected: 500, got: %d", resp.StatusCode)
	}
	if resp.Data != nil {
		t.Errorf("Wrong match. expected: nil data, got: %s", resp.Data)
	}
}
