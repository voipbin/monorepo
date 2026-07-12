package rabbitmqhandler

import (
	"errors"
	"monorepo/bin-common-handler/models/sock"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ============================================================================
// getRetryCount() Tests
// ============================================================================

func Test_getRetryCount_missingHeader(t *testing.T) {
	got := getRetryCount(amqp.Table{})
	if got != 0 {
		t.Errorf("Expected 0 for missing header, got %d", got)
	}
}

func Test_getRetryCount_nilHeaders(t *testing.T) {
	got := getRetryCount(nil)
	if got != 0 {
		t.Errorf("Expected 0 for nil headers, got %d", got)
	}
}

func Test_getRetryCount_malformedType(t *testing.T) {
	// A string value where an int32 is expected should fail open to 0
	// (treated as first attempt), never silently exceeding the retry cap.
	got := getRetryCount(amqp.Table{headerRetryCount: "not-an-int"})
	if got != 0 {
		t.Errorf("Expected 0 for malformed header type, got %d", got)
	}
}

func Test_getRetryCount_presentInt32(t *testing.T) {
	got := getRetryCount(amqp.Table{headerRetryCount: int32(2)})
	if got != 2 {
		t.Errorf("Expected 2, got %d", got)
	}
}

// ============================================================================
// cloneHeaders() Tests
// ============================================================================

func Test_cloneHeaders_doesNotMutateOriginal(t *testing.T) {
	original := amqp.Table{"foo": "bar"}
	clone := cloneHeaders(original)
	clone["foo"] = "changed"
	clone[headerRetryCount] = 1

	if original["foo"] != "bar" {
		t.Errorf("Expected original to be unmodified, got %v", original["foo"])
	}
	if _, ok := original[headerRetryCount]; ok {
		t.Error("Expected original to not have headerRetryCount added")
	}
}

func Test_cloneHeaders_nilInput(t *testing.T) {
	clone := cloneHeaders(nil)
	if clone == nil {
		t.Fatal("Expected non-nil clone for nil input")
	}
	clone["x"] = "y" // must not panic
}

// ============================================================================
// ackOrRetry() Tests -- success path only (the retry-publish path requires a
// real *amqp.Channel via r.connection.Channel(), which publishExchange uses
// as a concrete type rather than the amqpChannel interface, so it cannot be
// unit-tested with the existing mock seams. Per the design doc §7, that path
// is covered by a recommended (not required-to-merge) integration test
// against a real RabbitMQ broker with the delayed-message-exchange plugin.)
// ============================================================================

func Test_ackOrRetry_success_acksMessage(t *testing.T) {
	r := &rabbit{}

	// A real amqp.Delivery cannot be constructed with a working Ack() outside
	// the amqp091-go library (Acknowledger is unexported/internal), so this
	// test verifies ackOrRetry does not panic and takes the success branch by
	// asserting no panic occurs; deeper Ack-call verification is covered by
	// the retry-publish path's documented integration test.
	defer func() {
		if rec := recover(); rec != nil {
			t.Errorf("ackOrRetry panicked on success path: %v", rec)
		}
	}()

	r.ackOrRetry(amqp.Delivery{}, "test-queue", nil)
}

// ============================================================================
// registerConsumer() Tests -- one-queue-one-registration invariant
// ============================================================================

func Test_registerConsumer_firstRegistrationSucceeds(t *testing.T) {
	r := &rabbit{consumers: make([]*consumerRegistration, 0)}

	reg := &consumerRegistration{queueName: "test-queue", cType: consumerTypeMessage}
	if err := r.registerConsumer(reg); err != nil {
		t.Errorf("Expected no error for first registration, got %v", err)
	}
	if len(r.consumers) != 1 {
		t.Errorf("Expected 1 registered consumer, got %d", len(r.consumers))
	}
}

func Test_registerConsumer_duplicateQueueRejected(t *testing.T) {
	r := &rabbit{consumers: make([]*consumerRegistration, 0)}

	first := &consumerRegistration{queueName: "test-queue", cType: consumerTypeMessage}
	if err := r.registerConsumer(first); err != nil {
		t.Fatalf("Expected first registration to succeed, got %v", err)
	}

	second := &consumerRegistration{queueName: "test-queue", cType: consumerTypeRPC}
	err := r.registerConsumer(second)
	if err == nil {
		t.Error("Expected an error for duplicate queue registration, got nil")
	}
	if len(r.consumers) != 1 {
		t.Errorf("Expected registration count to stay at 1 after rejected duplicate, got %d", len(r.consumers))
	}
}

func Test_registerConsumer_differentQueuesBothSucceed(t *testing.T) {
	r := &rabbit{consumers: make([]*consumerRegistration, 0)}

	if err := r.registerConsumer(&consumerRegistration{queueName: "queue-a"}); err != nil {
		t.Fatalf("Expected queue-a registration to succeed, got %v", err)
	}
	if err := r.registerConsumer(&consumerRegistration{queueName: "queue-b"}); err != nil {
		t.Fatalf("Expected queue-b registration to succeed, got %v", err)
	}
	if len(r.consumers) != 2 {
		t.Errorf("Expected 2 registered consumers, got %d", len(r.consumers))
	}
}

// ============================================================================
// startConsumers() Qos application Tests (VOIP-1233 mandatory companion fix)
// ============================================================================

func Test_startConsumers_appliesQosWithNumWorkers(t *testing.T) {
	mockCh := newMockChannel()
	r := &rabbit{queues: make(map[string]*queue)}
	r.queues["test-queue"] = &queue{name: "test-queue", channel: mockCh}

	reg := &consumerRegistration{
		queueName:    "test-queue",
		consumerName: "test-consumer",
		numWorkers:   7,
		cType:        consumerTypeMessage,
		cbMessage:    func(evt *sock.Event) error { return nil },
	}

	if err := r.startConsumers(reg); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if mockCh.qosCallCount != 1 {
		t.Errorf("Expected Qos to be called exactly once, got %d", mockCh.qosCallCount)
	}
	if mockCh.qosPrefetchCount != 7 {
		t.Errorf("Expected Qos prefetchCount=numWorkers(7), got %d", mockCh.qosPrefetchCount)
	}
	if mockCh.qosPrefetchSize != 0 {
		t.Errorf("Expected Qos prefetchSize=0, got %d", mockCh.qosPrefetchSize)
	}
	if mockCh.qosGlobal != false {
		t.Errorf("Expected Qos global=false, got %v", mockCh.qosGlobal)
	}
}

func Test_startConsumers_qosErrorSurfacesBeforeConsume(t *testing.T) {
	mockCh := newMockChannel()
	mockCh.qosErr = errors.New("qos failed")

	r := &rabbit{queues: make(map[string]*queue)}
	r.queues["test-queue"] = &queue{name: "test-queue", channel: mockCh}

	reg := &consumerRegistration{
		queueName:  "test-queue",
		numWorkers: 1,
		cType:      consumerTypeMessage,
	}

	err := r.startConsumers(reg)
	if err == nil {
		t.Fatal("Expected error when Qos fails")
	}
	if mockCh.qosCallCount != 1 {
		t.Errorf("Expected Qos to have been attempted once, got %d", mockCh.qosCallCount)
	}
}

func Test_startConsumers_reappliesQosOnReRegistration(t *testing.T) {
	// Simulates reconsumerAll calling startConsumers again on reconnection --
	// Qos must be re-applied each time, not just once at queue creation.
	mockCh := newMockChannel()
	r := &rabbit{queues: make(map[string]*queue)}
	r.queues["test-queue"] = &queue{name: "test-queue", channel: mockCh}

	reg := &consumerRegistration{
		queueName:  "test-queue",
		numWorkers: 3,
		cType:      consumerTypeMessage,
		cbMessage:  func(evt *sock.Event) error { return nil },
	}

	if err := r.startConsumers(reg); err != nil {
		t.Fatalf("first startConsumers call failed: %v", err)
	}
	if err := r.startConsumers(reg); err != nil {
		t.Fatalf("second startConsumers call (simulated reconnection) failed: %v", err)
	}

	if mockCh.qosCallCount != 2 {
		t.Errorf("Expected Qos to be called once per startConsumers invocation (2 total), got %d", mockCh.qosCallCount)
	}
}
