package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/circuitbreakerhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_sendRequest_CircuitBreakerAllowRejects(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockCB := circuitbreakerhandler.NewMockCircuitBreakerHandler(mc)

	h := requestHandler{
		sock: mockSock,
		cb:   mockCB,
	}

	queue := commonoutline.QueueName("test.queue")
	mockCB.EXPECT().Allow(string(queue)).Return(fmt.Errorf("%w for target: %s", circuitbreakerhandler.ErrCircuitOpen, string(queue)))

	// sock should NOT be called when circuit is open
	// (no mockSock.EXPECT calls)

	_, err := h.sendRequest(context.Background(), queue, "/test", sock.RequestMethodGet, "", 3000, 0, "", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error when circuit is open, got nil")
	}
}

func Test_sendRequest_CircuitBreakerRecordsSuccess(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockCB := circuitbreakerhandler.NewMockCircuitBreakerHandler(mc)

	h := requestHandler{
		sock: mockSock,
		cb:   mockCB,
	}

	queue := commonoutline.QueueName("test.queue")
	resp := &sock.Response{StatusCode: 200}

	mockCB.EXPECT().Allow(string(queue)).Return(nil)
	mockSock.EXPECT().RequestPublish(gomock.Any(), string(queue), gomock.Any()).Return(resp, nil)
	mockCB.EXPECT().RecordSuccess(string(queue))

	res, err := h.sendRequest(context.Background(), queue, "/test", sock.RequestMethodGet, "", 3000, 0, "", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res != resp {
		t.Errorf("expected response %v, got %v", resp, res)
	}
}

func Test_sendRequest_CircuitBreakerRecordsFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockCB := circuitbreakerhandler.NewMockCircuitBreakerHandler(mc)

	h := requestHandler{
		sock: mockSock,
		cb:   mockCB,
	}

	queue := commonoutline.QueueName("test.queue")

	mockCB.EXPECT().Allow(string(queue)).Return(nil)
	mockSock.EXPECT().RequestPublish(gomock.Any(), string(queue), gomock.Any()).Return(nil, fmt.Errorf("connection refused"))
	mockCB.EXPECT().RecordFailure(string(queue))

	_, err := h.sendRequest(context.Background(), queue, "/test", sock.RequestMethodGet, "", 3000, 0, "", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error on send failure, got nil")
	}
}

func Test_sendRequest_DelayedRequestBypassesCircuitBreaker(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockCB := circuitbreakerhandler.NewMockCircuitBreakerHandler(mc)

	h := requestHandler{
		sock: mockSock,
		cb:   mockCB,
	}

	queue := commonoutline.QueueName("test.queue")

	// No cb.Allow, cb.RecordSuccess, or cb.RecordFailure should be called for delayed requests
	mockSock.EXPECT().RequestPublishWithDelay(string(queue), gomock.Any(), 5000).Return(nil)

	_, err := h.sendRequest(context.Background(), queue, "/test", sock.RequestMethodGet, "", 3000, 5000, "", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("expected no error for delayed request, got %v", err)
	}
}

func Test_sendRequest_NilCircuitBreakerIsSafe(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := requestHandler{
		sock: mockSock,
		// cb is nil — simulates legacy test setup without circuit breaker
	}

	queue := commonoutline.QueueName("test.queue")
	resp := &sock.Response{StatusCode: 200}

	mockSock.EXPECT().RequestPublish(gomock.Any(), string(queue), gomock.Any()).Return(resp, nil)

	res, err := h.sendRequest(context.Background(), queue, "/test", sock.RequestMethodGet, "", 3000, 0, "", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("expected no error with nil cb, got %v", err)
	}
	if res != resp {
		t.Errorf("expected response %v, got %v", resp, res)
	}
}

func Test_sendRequest_NilCircuitBreakerOnFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := requestHandler{
		sock: mockSock,
		// cb is nil
	}

	queue := commonoutline.QueueName("test.queue")

	mockSock.EXPECT().RequestPublish(gomock.Any(), string(queue), gomock.Any()).Return(nil, fmt.Errorf("timeout"))

	_, err := h.sendRequest(context.Background(), queue, "/test", sock.RequestMethodGet, "", 3000, 0, "", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
