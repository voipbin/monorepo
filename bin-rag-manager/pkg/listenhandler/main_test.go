package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/document"
	"monorepo/bin-rag-manager/models/query"
	"monorepo/bin-rag-manager/models/rag"

	"monorepo/bin-common-handler/models/sock"
)

// mockSockHandler for testing
type mockSockHandler struct {
	queueCreateFunc func(name string, queueType string) error
	consumeRPCFunc  func(ctx context.Context, queueName, consumerName string, skipVerify, enableDelay, enableDelayHourly bool, delayExpiredMin int, handler sock.CbMsgRPC) error
}

func (m *mockSockHandler) QueueCreate(name string, queueType string) error {
	if m.queueCreateFunc != nil {
		return m.queueCreateFunc(name, queueType)
	}
	return nil
}

func (m *mockSockHandler) ConsumeRPC(ctx context.Context, queueName, consumerName string, skipVerify, enableDelay, enableDelayHourly bool, delayExpiredMin int, handler sock.CbMsgRPC) error {
	if m.consumeRPCFunc != nil {
		return m.consumeRPCFunc(ctx, queueName, consumerName, skipVerify, enableDelay, enableDelayHourly, delayExpiredMin, handler)
	}
	return nil
}

func (m *mockSockHandler) Close() {}

func (m *mockSockHandler) Connect() {}

func (m *mockSockHandler) ConsumeMessage(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, messageConsume sock.CbMsgConsume) error {
	return nil
}

func (m *mockSockHandler) TopicCreate(name string) error {
	return nil
}

func (m *mockSockHandler) EventPublish(topic string, key string, evt *sock.Event) error {
	return nil
}

func (m *mockSockHandler) EventPublishWithDelay(topic string, key string, evt *sock.Event, delay int) error {
	return nil
}

func (m *mockSockHandler) RequestPublish(ctx context.Context, queueName string, req *sock.Request) (*sock.Response, error) {
	return nil, nil
}

func (m *mockSockHandler) RequestPublishWithDelay(queueName string, req *sock.Request, delay int) error {
	return nil
}

func (m *mockSockHandler) QueueSubscribe(name string, topic string) error {
	return nil
}

// QueueBind/QueueUnbind/TopicCreateWithKind added to satisfy sockhandler.SockHandler after
// VOIP-1258 (Tasks 1.3/1.4). This service does not exercise these paths -- no-op stubs.
func (m *mockSockHandler) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return nil
}

func (m *mockSockHandler) QueueUnbind(name, key, exchange string, args amqp.Table) error {
	return nil
}

func (m *mockSockHandler) TopicCreateWithKind(name string, kind string) error {
	return nil
}

// mockRagHandlerForListen for testing
type mockRagHandlerForListen struct{}

func (m *mockRagHandlerForListen) RagCreate(_ context.Context, _ uuid.UUID, _, _ string, _ []uuid.UUID, _ []string) (*rag.Rag, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockRagHandlerForListen) RagGet(_ context.Context, _ uuid.UUID) (*rag.Rag, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockRagHandlerForListen) RagList(_ context.Context, _ uint64, _ string, _ map[rag.Field]any) ([]*rag.Rag, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockRagHandlerForListen) RagUpdate(_ context.Context, _ uuid.UUID, _ map[rag.Field]any) (*rag.Rag, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockRagHandlerForListen) RagDelete(_ context.Context, _ uuid.UUID) error {
	return fmt.Errorf("not implemented")
}
func (m *mockRagHandlerForListen) RagAddSources(_ context.Context, _ uuid.UUID, _ []uuid.UUID, _ []string) (*rag.Rag, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockRagHandlerForListen) RagRemoveSource(_ context.Context, _, _ uuid.UUID) (*rag.Rag, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockRagHandlerForListen) DocumentGet(_ context.Context, _ uuid.UUID) (*document.Document, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockRagHandlerForListen) DocumentList(_ context.Context, _ uint64, _ string, _ map[document.Field]any) ([]*document.Document, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockRagHandlerForListen) QueryRag(_ context.Context, _ uuid.UUID, _ string, _ int) (*query.Response, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockRagHandlerForListen) DocumentIngestPendingAll(_ context.Context) {}
func (m *mockRagHandlerForListen) RunIngestionTicker(_ context.Context, _ time.Duration) {}

func TestListenHandler_Interface(t *testing.T) {
	var _ ListenHandler = &listenHandler{}
}

func TestNewListenHandler(t *testing.T) {
	sockHandler := &mockSockHandler{}
	ragHandler := &mockRagHandlerForListen{}

	h := NewListenHandler(sockHandler, ragHandler)
	if h == nil {
		t.Error("expected non-nil handler")
	}
}

func TestListenHandler_Run(t *testing.T) {
	queueCreated := false
	consumeCalled := false

	sockHandler := &mockSockHandler{
		queueCreateFunc: func(name string, queueType string) error {
			queueCreated = true
			if queueType != "normal" {
				t.Errorf("expected queueType 'normal', got %q", queueType)
			}
			return nil
		},
		consumeRPCFunc: func(ctx context.Context, queueName, consumerName string, skipVerify, enableDelay, enableDelayHourly bool, delayExpiredMin int, handler sock.CbMsgRPC) error {
			consumeCalled = true
			return nil
		},
	}
	ragHandler := &mockRagHandlerForListen{}

	h := NewListenHandler(sockHandler, ragHandler)
	err := h.Run("test-queue", "test-exchange")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !queueCreated {
		t.Error("expected queue to be created")
	}

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)

	if !consumeCalled {
		t.Error("expected ConsumeRPC to be called")
	}
}

func TestListenHandler_Run_QueueCreateError(t *testing.T) {
	expectedErr := errors.New("queue create failed")
	sockHandler := &mockSockHandler{
		queueCreateFunc: func(name string, queueType string) error {
			return expectedErr
		},
	}
	ragHandler := &mockRagHandlerForListen{}

	h := NewListenHandler(sockHandler, ragHandler)
	err := h.Run("test-queue", "test-exchange")
	if err == nil {
		t.Error("expected error from queue creation")
	}
}

func TestSimpleResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"success", 200},
		{"bad request", 400},
		{"not found", 404},
		{"server error", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := simpleResponse(tt.statusCode)
			if resp.StatusCode != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, resp.StatusCode)
			}
		})
	}
}

func TestJsonResponse(t *testing.T) {
	data := map[string]string{"key": "value"}
	resp := jsonResponse(200, data)

	if resp.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", resp.StatusCode)
	}
	if resp.DataType != "application/json" {
		t.Errorf("expected DataType 'application/json', got %q", resp.DataType)
	}

	var decoded map[string]string
	if err := json.Unmarshal(resp.Data, &decoded); err != nil {
		t.Errorf("failed to unmarshal response data: %v", err)
	}
	if decoded["key"] != "value" {
		t.Error("response data not marshalled correctly")
	}
}

func TestProcessRequest_NotFound(t *testing.T) {
	h := &listenHandler{
		sockHandler: &mockSockHandler{},
		ragHandler:  &mockRagHandlerForListen{},
	}

	req := &sock.Request{
		URI:    "/v1/unknown/endpoint",
		Method: sock.RequestMethodGet,
	}

	resp, err := h.processRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected status code 404, got %d", resp.StatusCode)
	}
}
