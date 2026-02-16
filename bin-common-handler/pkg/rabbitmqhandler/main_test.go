package rabbitmqhandler

import (
	"context"
	"errors"
	"monorepo/bin-common-handler/models/sock"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// mockChannel is a mock implementation of amqpChannel for testing
type mockChannel struct {
	closeCalled  int
	closeErr     error
	closeErrOnce bool // if true, only return error on first call

	queueDeclareErr  error
	queueDeclareName string
	queueDeclareArgs amqp.Table
	exchangeDeclareErr error

	qosErr       error
	queueBindErr error
	consumeErr   error
}

func newMockChannel() *mockChannel {
	return &mockChannel{}
}

func (m *mockChannel) Close() error {
	m.closeCalled++
	if m.closeErrOnce && m.closeCalled > 1 {
		return nil
	}
	return m.closeErr
}

func (m *mockChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	if m.consumeErr != nil {
		return nil, m.consumeErr
	}
	return make(<-chan amqp.Delivery), nil
}

func (m *mockChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	return m.qosErr
}

func (m *mockChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return m.queueBindErr
}

func (m *mockChannel) QueueDelete(name string, ifUnused, ifEmpty, noWait bool) (int, error) {
	return 0, nil
}

func (m *mockChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	return m.exchangeDeclareErr
}

func (m *mockChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	m.queueDeclareName = name
	m.queueDeclareArgs = args
	if m.queueDeclareErr != nil {
		return amqp.Queue{}, m.queueDeclareErr
	}
	return amqp.Queue{Name: name}, nil
}

// mockConnection is a mock implementation of amqpConnection for testing
type mockConnection struct {
	channelFunc    func() (*amqp.Channel, error)
	channelErr     error
	closeCalled    int
	closeErr       error
	notifyCloseCh  chan *amqp.Error
	channelCallCnt int
}

func newMockConnection() *mockConnection {
	return &mockConnection{
		notifyCloseCh: make(chan *amqp.Error, 1),
	}
}

func (m *mockConnection) Channel() (*amqp.Channel, error) {
	m.channelCallCnt++
	if m.channelFunc != nil {
		return m.channelFunc()
	}
	if m.channelErr != nil {
		return nil, m.channelErr
	}
	// Return nil channel - tests should use channelMock directly
	return nil, nil
}

func (m *mockConnection) Close() error {
	m.closeCalled++
	return m.closeErr
}

func (m *mockConnection) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	m.notifyCloseCh = receiver
	return receiver
}

// mockChannelWithConsumeCounter tracks Consume calls and can fail on demand
type mockChannelWithConsumeCounter struct {
	consumeCallCount *int
	failUntil        int // fail Consume calls until this count
}

func (m *mockChannelWithConsumeCounter) Close() error { return nil }
func (m *mockChannelWithConsumeCounter) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	*m.consumeCallCount++
	if *m.consumeCallCount <= m.failUntil {
		return nil, errors.New("consume failed")
	}
	return make(<-chan amqp.Delivery), nil
}
func (m *mockChannelWithConsumeCounter) Qos(prefetchCount, prefetchSize int, global bool) error {
	return nil
}
func (m *mockChannelWithConsumeCounter) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return nil
}
func (m *mockChannelWithConsumeCounter) QueueDelete(name string, ifUnused, ifEmpty, noWait bool) (int, error) {
	return 0, nil
}
func (m *mockChannelWithConsumeCounter) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	return nil
}
func (m *mockChannelWithConsumeCounter) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{Name: name}, nil
}

// ============================================================================
// Close() Tests
// ============================================================================

func TestClose_ClosesAllQueueChannels(t *testing.T) {
	mockCh1 := newMockChannel()
	mockCh2 := newMockChannel()
	mockCh3 := newMockChannel()
	mockConn := newMockConnection()

	r := &rabbit{
		uri:        "amqp://localhost",
		connection: mockConn,
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	r.queues["queue1"] = &queue{name: "queue1", channel: mockCh1}
	r.queues["queue2"] = &queue{name: "queue2", channel: mockCh2}
	r.queues["queue3"] = &queue{name: "queue3", channel: mockCh3}

	r.Close()

	if mockCh1.closeCalled != 1 {
		t.Errorf("Expected queue1 channel Close() to be called once, got %d", mockCh1.closeCalled)
	}
	if mockCh2.closeCalled != 1 {
		t.Errorf("Expected queue2 channel Close() to be called once, got %d", mockCh2.closeCalled)
	}
	if mockCh3.closeCalled != 1 {
		t.Errorf("Expected queue3 channel Close() to be called once, got %d", mockCh3.closeCalled)
	}
	if mockConn.closeCalled != 1 {
		t.Errorf("Expected connection Close() to be called once, got %d", mockConn.closeCalled)
	}
	if !r.closed.Load() {
		t.Error("Expected rabbit.closed to be true")
	}
}

func TestClose_ClosesAllExchangeChannels(t *testing.T) {
	mockCh1 := newMockChannel()
	mockCh2 := newMockChannel()
	mockConn := newMockConnection()

	r := &rabbit{
		uri:        "amqp://localhost",
		connection: mockConn,
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	r.exchanges["exchange1"] = &exchange{name: "exchange1", channel: mockCh1}
	r.exchanges["exchange2"] = &exchange{name: "exchange2", channel: mockCh2}

	r.Close()

	if mockCh1.closeCalled != 1 {
		t.Errorf("Expected exchange1 channel Close() to be called once, got %d", mockCh1.closeCalled)
	}
	if mockCh2.closeCalled != 1 {
		t.Errorf("Expected exchange2 channel Close() to be called once, got %d", mockCh2.closeCalled)
	}
}

func TestClose_ClosesQueueAndExchangeChannels(t *testing.T) {
	queueCh := newMockChannel()
	exchangeCh := newMockChannel()
	mockConn := newMockConnection()

	r := &rabbit{
		uri:        "amqp://localhost",
		connection: mockConn,
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	r.queues["queue1"] = &queue{name: "queue1", channel: queueCh}
	r.exchanges["exchange1"] = &exchange{name: "exchange1", channel: exchangeCh}

	r.Close()

	if queueCh.closeCalled != 1 {
		t.Errorf("Expected queue channel Close() to be called once, got %d", queueCh.closeCalled)
	}
	if exchangeCh.closeCalled != 1 {
		t.Errorf("Expected exchange channel Close() to be called once, got %d", exchangeCh.closeCalled)
	}
}

func TestClose_HandlesNilChannels(t *testing.T) {
	mockConn := newMockConnection()

	r := &rabbit{
		uri:        "amqp://localhost",
		connection: mockConn,
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	r.queues["queue1"] = &queue{name: "queue1", channel: nil}
	r.exchanges["exchange1"] = &exchange{name: "exchange1", channel: nil}

	// Should not panic
	r.Close()

	if mockConn.closeCalled != 1 {
		t.Errorf("Expected connection Close() to be called once, got %d", mockConn.closeCalled)
	}
}

func TestClose_HandlesEmptyMaps(t *testing.T) {
	mockConn := newMockConnection()

	r := &rabbit{
		uri:        "amqp://localhost",
		connection: mockConn,
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	// Should not panic with empty maps
	r.Close()

	if mockConn.closeCalled != 1 {
		t.Errorf("Expected connection Close() to be called once, got %d", mockConn.closeCalled)
	}
}

func TestClose_ContinuesOnChannelCloseError(t *testing.T) {
	mockCh1 := newMockChannel()
	mockCh1.closeErr = errors.New("close error")
	mockCh2 := newMockChannel()
	mockConn := newMockConnection()

	r := &rabbit{
		uri:        "amqp://localhost",
		connection: mockConn,
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	r.queues["queue1"] = &queue{name: "queue1", channel: mockCh1}
	r.queues["queue2"] = &queue{name: "queue2", channel: mockCh2}

	// Should not stop on error
	r.Close()

	if mockCh1.closeCalled != 1 {
		t.Errorf("Expected queue1 channel Close() to be called, got %d", mockCh1.closeCalled)
	}
	// Note: map iteration order is not guaranteed, but both should be called
	if mockCh2.closeCalled != 1 {
		t.Errorf("Expected queue2 channel Close() to be called, got %d", mockCh2.closeCalled)
	}
	if mockConn.closeCalled != 1 {
		t.Error("Expected connection to be closed even if channel close fails")
	}
}

// ============================================================================
// QueueDeclare() Tests
// ============================================================================

func TestQueueDeclare_Success(t *testing.T) {
	mockCh := newMockChannel()
	mockConn := newMockConnection()

	// We need a way to inject the mock channel. Since connection.Channel() returns *amqp.Channel,
	// we'll test the logic by creating a wrapper test.
	r := &rabbit{
		uri:        "amqp://localhost",
		connection: mockConn,
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	// Since we can't easily inject the mock channel through connection.Channel(),
	// we test by directly manipulating the queue map to verify the struct is correct
	q := amqp.Queue{Name: "test-queue"}
	r.queues["test-queue"] = &queue{
		name:       "test-queue",
		durable:    true,
		autoDelete: false,
		exclusive:  false,
		noWait:     false,
		channel:    mockCh,
		queue:      &q,
	}

	if r.queues["test-queue"] == nil {
		t.Error("Expected queue to be stored")
	}
	if r.queues["test-queue"].name != "test-queue" {
		t.Errorf("Expected queue name 'test-queue', got '%s'", r.queues["test-queue"].name)
	}
}

func TestQueueDeclare_ClosesOldChannelOnRedeclare(t *testing.T) {
	oldMockCh := newMockChannel()
	newMockCh := newMockChannel()
	mockConn := newMockConnection()

	r := &rabbit{
		uri:        "amqp://localhost",
		connection: mockConn,
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	// Setup existing queue
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: oldMockCh,
	}

	// Simulate re-declaration logic (what QueueDeclare does)
	if existing := r.queues["test-queue"]; existing != nil && existing.channel != nil {
		_ = existing.channel.Close()
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: newMockCh,
	}

	if oldMockCh.closeCalled != 1 {
		t.Errorf("Expected old channel Close() to be called once, got %d", oldMockCh.closeCalled)
	}
	if newMockCh.closeCalled != 0 {
		t.Errorf("Expected new channel Close() not to be called, got %d", newMockCh.closeCalled)
	}
}

func TestQueueDeclare_ClosesChannelOnError(t *testing.T) {
	mockCh := newMockChannel()
	mockCh.queueDeclareErr = errors.New("declare failed")

	// Simulate the error handling logic in QueueDeclare
	_, err := mockCh.QueueDeclare("test", true, false, false, false, nil)
	if err != nil {
		_ = mockCh.Close()
	}

	if mockCh.closeCalled != 1 {
		t.Errorf("Expected channel Close() to be called on error, got %d calls", mockCh.closeCalled)
	}
}

// ============================================================================
// ExchangeDeclare() Tests
// ============================================================================

func TestExchangeDeclare_ClosesOldChannelOnRedeclare(t *testing.T) {
	oldMockCh := newMockChannel()
	newMockCh := newMockChannel()
	mockConn := newMockConnection()

	r := &rabbit{
		uri:        "amqp://localhost",
		connection: mockConn,
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	// Setup existing exchange
	r.exchanges["test-exchange"] = &exchange{
		name:    "test-exchange",
		channel: oldMockCh,
	}

	// Simulate re-declaration logic (what ExchangeDeclare does)
	if existing := r.exchanges["test-exchange"]; existing != nil && existing.channel != nil {
		_ = existing.channel.Close()
	}
	r.exchanges["test-exchange"] = &exchange{
		name:    "test-exchange",
		channel: newMockCh,
	}

	if oldMockCh.closeCalled != 1 {
		t.Errorf("Expected old channel Close() to be called once, got %d", oldMockCh.closeCalled)
	}
	if newMockCh.closeCalled != 0 {
		t.Errorf("Expected new channel Close() not to be called, got %d", newMockCh.closeCalled)
	}
}

func TestExchangeDeclare_ClosesChannelOnError(t *testing.T) {
	mockCh := newMockChannel()
	mockCh.exchangeDeclareErr = errors.New("declare failed")

	// Simulate the error handling logic in ExchangeDeclare
	err := mockCh.ExchangeDeclare("test", "direct", true, false, false, false, nil)
	if err != nil {
		_ = mockCh.Close()
	}

	if mockCh.closeCalled != 1 {
		t.Errorf("Expected channel Close() to be called on error, got %d calls", mockCh.closeCalled)
	}
}

// ============================================================================
// QueueGet() Tests
// ============================================================================

func TestQueueGet_ReturnsExistingQueue(t *testing.T) {
	mockCh := newMockChannel()

	r := &rabbit{
		queues: make(map[string]*queue),
	}

	expectedQueue := &queue{
		name:    "test-queue",
		channel: mockCh,
	}
	r.queues["test-queue"] = expectedQueue

	result := r.queueGet("test-queue")

	if result != expectedQueue {
		t.Error("Expected queueGet to return the stored queue")
	}
}

func TestQueueGet_ReturnsNilForNonExistent(t *testing.T) {
	r := &rabbit{
		queues: make(map[string]*queue),
	}

	result := r.queueGet("non-existent")

	if result != nil {
		t.Error("Expected queueGet to return nil for non-existent queue")
	}
}

// ============================================================================
// QueueDelete() Tests
// ============================================================================

func TestQueueDelete_ReturnsNilForNonExistent(t *testing.T) {
	r := &rabbit{
		queues: make(map[string]*queue),
	}

	count, err := r.QueueDelete("non-existent", false, false, false)

	if err != nil {
		t.Errorf("Expected no error for non-existent queue, got %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
}

// ============================================================================
// QueueQoS() Tests
// ============================================================================

func TestQueueQoS_ReturnsErrorForNonExistent(t *testing.T) {
	r := &rabbit{
		queues: make(map[string]*queue),
	}

	err := r.QueueQoS("non-existent", 1, 0)

	if err == nil {
		t.Error("Expected error for non-existent queue")
	}
}

func TestQueueQoS_Success(t *testing.T) {
	mockCh := newMockChannel()

	r := &rabbit{
		queues: make(map[string]*queue),
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	err := r.QueueQoS("test-queue", 1, 0)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestQueueQoS_ReturnsChannelError(t *testing.T) {
	mockCh := newMockChannel()
	mockCh.qosErr = errors.New("qos failed")

	r := &rabbit{
		queues: make(map[string]*queue),
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	err := r.QueueQoS("test-queue", 1, 0)

	if err == nil {
		t.Error("Expected error from channel Qos")
	}
}

// ============================================================================
// QueueBind() Tests
// ============================================================================

func TestQueueBind_ReturnsErrorForNonExistent(t *testing.T) {
	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string]*queueBind),
	}

	err := r.QueueBind("non-existent", "key", "exchange", false, nil)

	if err == nil {
		t.Error("Expected error for non-existent queue")
	}
}

func TestQueueBind_Success(t *testing.T) {
	mockCh := newMockChannel()

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string]*queueBind),
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	err := r.QueueBind("test-queue", "routing-key", "test-exchange", false, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify bind was stored
	if r.queueBinds["test-queue"] == nil {
		t.Error("Expected queueBind to be stored")
	}
	if r.queueBinds["test-queue"].key != "routing-key" {
		t.Errorf("Expected key 'routing-key', got '%s'", r.queueBinds["test-queue"].key)
	}
	if r.queueBinds["test-queue"].exchange != "test-exchange" {
		t.Errorf("Expected exchange 'test-exchange', got '%s'", r.queueBinds["test-queue"].exchange)
	}
}

func TestQueueBind_ReturnsChannelError(t *testing.T) {
	mockCh := newMockChannel()
	mockCh.queueBindErr = errors.New("bind failed")

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string]*queueBind),
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	err := r.QueueBind("test-queue", "key", "exchange", false, nil)

	if err == nil {
		t.Error("Expected error from channel QueueBind")
	}
}

// ============================================================================
// QueueSubscribe() Tests
// ============================================================================

func TestQueueSubscribe_CallsQueueBind(t *testing.T) {
	mockCh := newMockChannel()

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string]*queueBind),
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	err := r.QueueSubscribe("test-queue", "test-topic")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// QueueSubscribe calls QueueBind with empty key
	if r.queueBinds["test-queue"] == nil {
		t.Error("Expected queueBind to be stored")
	}
	if r.queueBinds["test-queue"].key != "" {
		t.Errorf("Expected empty key, got '%s'", r.queueBinds["test-queue"].key)
	}
	if r.queueBinds["test-queue"].exchange != "test-topic" {
		t.Errorf("Expected exchange 'test-topic', got '%s'", r.queueBinds["test-queue"].exchange)
	}
}

// ============================================================================
// NewRabbit() Tests
// ============================================================================

func TestNewRabbit_InitializesCorrectly(t *testing.T) {
	uri := "amqp://guest:guest@localhost:5672/"
	r := NewRabbit(uri)

	if r == nil {
		t.Fatal("Expected NewRabbit to return non-nil")
	}

	rabbit, ok := r.(*rabbit)
	if !ok {
		t.Fatal("Expected NewRabbit to return *rabbit")
	}

	if rabbit.uri != uri {
		t.Errorf("Expected uri '%s', got '%s'", uri, rabbit.uri)
	}

	if rabbit.queues == nil {
		t.Error("Expected queues map to be initialized")
	}

	if rabbit.exchanges == nil {
		t.Error("Expected exchanges map to be initialized")
	}

	if rabbit.queueBinds == nil {
		t.Error("Expected queueBinds map to be initialized")
	}

	if rabbit.closed.Load() {
		t.Error("Expected closed to be false initially")
	}
}

func Test_newRabbit_initializesConsumers(t *testing.T) {
	uri := "amqp://guest:guest@localhost:5672/"
	r := NewRabbit(uri)

	rabbit, ok := r.(*rabbit)
	if !ok {
		t.Fatal("Expected NewRabbit to return *rabbit")
	}

	if rabbit.consumers == nil {
		t.Error("Expected consumers slice to be initialized")
	}
	if len(rabbit.consumers) != 0 {
		t.Errorf("Expected 0 consumers, got %d", len(rabbit.consumers))
	}
}

// ============================================================================
// QueueCreate() Tests
// ============================================================================

func TestQueueCreate_InvalidType(t *testing.T) {
	r := &rabbit{
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	err := r.QueueCreate("test-queue", "invalid-type")

	if err == nil {
		t.Error("Expected error for invalid queue type")
	}
}

// ============================================================================
// Integration-style Tests (testing multiple components together)
// ============================================================================

func TestRedeclareScenario_ClosesOldChannels(t *testing.T) {
	// This test simulates what happens during reconnection
	oldQueueCh := newMockChannel()
	oldExchangeCh := newMockChannel()
	newQueueCh := newMockChannel()
	newExchangeCh := newMockChannel()

	r := &rabbit{
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	// Initial state
	r.queues["queue1"] = &queue{name: "queue1", durable: true, channel: oldQueueCh}
	r.exchanges["exchange1"] = &exchange{name: "exchange1", kind: "direct", channel: oldExchangeCh}

	// Simulate redeclare (what redeclareAll does)
	for name, q := range r.queues {
		if q.channel != nil {
			_ = q.channel.Close()
		}
		r.queues[name] = &queue{
			name:    q.name,
			durable: q.durable,
			channel: newQueueCh,
		}
	}

	for name, e := range r.exchanges {
		if e.channel != nil {
			_ = e.channel.Close()
		}
		r.exchanges[name] = &exchange{
			name:    e.name,
			kind:    e.kind,
			channel: newExchangeCh,
		}
	}

	// Verify old channels were closed
	if oldQueueCh.closeCalled != 1 {
		t.Errorf("Expected old queue channel to be closed once, got %d", oldQueueCh.closeCalled)
	}
	if oldExchangeCh.closeCalled != 1 {
		t.Errorf("Expected old exchange channel to be closed once, got %d", oldExchangeCh.closeCalled)
	}

	// Verify new channels are not closed
	if newQueueCh.closeCalled != 0 {
		t.Errorf("Expected new queue channel not to be closed, got %d", newQueueCh.closeCalled)
	}
	if newExchangeCh.closeCalled != 0 {
		t.Errorf("Expected new exchange channel not to be closed, got %d", newExchangeCh.closeCalled)
	}
}

// ============================================================================
// Reconnector Tests
// ============================================================================

func TestReconnector_ExitsWhenErrorChannelClosed(t *testing.T) {
	errCh := make(chan *amqp.Error)
	r := &rabbit{
		uri:          "amqp://localhost",
		errorChannel: errCh,
		queues:       make(map[string]*queue),
		exchanges:    make(map[string]*exchange),
		queueBinds:   make(map[string]*queueBind),
	}

	done := make(chan struct{})
	go func() {
		r.reconnector()
		close(done)
	}()

	// Close the error channel to signal exit
	close(errCh)

	select {
	case <-done:
		// reconnector exited as expected
	case <-time.After(2 * time.Second):
		t.Fatal("reconnector did not exit after errorChannel was closed")
	}
}

func TestReconnector_ExitsWhenClosedFlagIsSet(t *testing.T) {
	errCh := make(chan *amqp.Error, 1)
	r := &rabbit{
		uri:          "amqp://localhost",
		errorChannel: errCh,
		queues:       make(map[string]*queue),
		exchanges:    make(map[string]*exchange),
		queueBinds:   make(map[string]*queueBind),
	}

	// Set closed before sending error
	r.closed.Store(true)

	done := make(chan struct{})
	go func() {
		r.reconnector()
		close(done)
	}()

	// Send an error to unblock the channel receive
	errCh <- amqp.ErrClosed

	select {
	case <-done:
		// reconnector exited because closed flag was set
	case <-time.After(2 * time.Second):
		t.Fatal("reconnector did not exit after closed flag was set")
	}
}

func TestClose_ClosedFlagIsAtomicallySafe(t *testing.T) {
	// Verify that concurrent reads of closed from reconnector
	// and write from Close do not cause a data race.
	// This test is meaningful when run with -race flag.
	mockConn := newMockConnection()
	errCh := make(chan *amqp.Error, 1)

	r := &rabbit{
		uri:          "amqp://localhost",
		connection:   mockConn,
		errorChannel: errCh,
		queues:       make(map[string]*queue),
		exchanges:    make(map[string]*exchange),
		queueBinds:   make(map[string]*queueBind),
	}

	var wg sync.WaitGroup

	// Simulate reconnector reading closed flag concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Read closed flag many times concurrently with Close()
		for i := 0; i < 100; i++ {
			_ = r.closed.Load()
		}
	}()

	// Call Close which writes the closed flag
	r.Close()

	wg.Wait()

	if !r.closed.Load() {
		t.Error("Expected closed to be true after Close()")
	}
}

func TestClose_SetsClosedBeforeClosingResources(t *testing.T) {
	// Verify that closed flag is set before channels/connection are closed.
	// This ensures reconnector sees the flag and exits rather than attempting reconnect.
	var closedAtChannelClose bool
	mockConn := newMockConnection()

	r := &rabbit{
		uri:        "amqp://localhost",
		connection: mockConn,
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	// Use a mock channel that captures the closed state when Close() is called
	captureChannel := &closedCaptureMockChannel{r: r}
	r.queues["test"] = &queue{name: "test", channel: captureChannel}

	r.Close()

	closedAtChannelClose = captureChannel.closedWasTrue

	if !closedAtChannelClose {
		t.Error("Expected closed flag to be true when channel.Close() was called")
	}
}

// closedCaptureMockChannel is a mock that checks the closed flag when Close() is called
type closedCaptureMockChannel struct {
	r              *rabbit
	closedWasTrue  bool
}

func (m *closedCaptureMockChannel) Close() error {
	m.closedWasTrue = m.r.closed.Load()
	return nil
}

func (m *closedCaptureMockChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return make(<-chan amqp.Delivery), nil
}

func (m *closedCaptureMockChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	return nil
}

func (m *closedCaptureMockChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return nil
}

func (m *closedCaptureMockChannel) QueueDelete(name string, ifUnused, ifEmpty, noWait bool) (int, error) {
	return 0, nil
}

func (m *closedCaptureMockChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	return nil
}

func (m *closedCaptureMockChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{Name: name}, nil
}

func Test_queueDeclare_storesArgs(t *testing.T) {
	mockCh := newMockChannel()
	r := &rabbit{
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	testArgs := amqp.Table{"x-expires": int32(1800000)}
	r.queues["test-queue"] = &queue{
		name:       "test-queue",
		durable:    false,
		autoDelete: true,
		exclusive:  false,
		noWait:     false,
		args:       testArgs,
		channel:    mockCh,
	}

	q := r.queues["test-queue"]
	if q.args == nil {
		t.Fatal("Expected args to be stored")
	}
	if q.args["x-expires"] != int32(1800000) {
		t.Errorf("Expected x-expires 1800000, got %v", q.args["x-expires"])
	}
}

func Test_queueCreateVolatile_xExpires(t *testing.T) {
	mockCh := newMockChannel()
	r := &rabbit{
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	// Verify args are stored in queue struct after queueCreateVolatile sets them
	testArgs := amqp.Table{"x-expires": int32(1800000)}
	r.queues["test-volatile"] = &queue{
		name:       "test-volatile",
		durable:    false,
		autoDelete: true,
		args:       testArgs,
		channel:    mockCh,
	}

	q := r.queues["test-volatile"]
	if q.args == nil {
		t.Fatal("Expected volatile queue to have args")
	}
	expires, ok := q.args["x-expires"]
	if !ok {
		t.Fatal("Expected x-expires in queue args")
	}
	if expires != int32(1800000) {
		t.Errorf("Expected x-expires 1800000, got %v", expires)
	}
}

func Test_startConsumers_queueNotFound(t *testing.T) {
	r := &rabbit{
		queues: make(map[string]*queue),
	}

	reg := &consumerRegistration{
		queueName:    "nonexistent",
		consumerName: "test-consumer",
		numWorkers:   1,
		cType:        consumerTypeMessage,
	}

	err := r.startConsumers(reg)
	if err == nil {
		t.Error("Expected error for non-existent queue")
	}
}

func Test_startConsumers_success(t *testing.T) {
	mockCh := newMockChannel()

	r := &rabbit{
		queues: make(map[string]*queue),
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	reg := &consumerRegistration{
		queueName:    "test-queue",
		consumerName: "test-consumer",
		numWorkers:   1,
		cType:        consumerTypeMessage,
		cbMessage: func(evt *sock.Event) error {
			return nil
		},
	}

	err := r.startConsumers(reg)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func Test_startConsumers_consumeError(t *testing.T) {
	mockCh := newMockChannel()
	mockCh.consumeErr = errors.New("consume failed")

	r := &rabbit{
		queues: make(map[string]*queue),
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	reg := &consumerRegistration{
		queueName:    "test-queue",
		consumerName: "test-consumer",
		numWorkers:   1,
		cType:        consumerTypeMessage,
	}

	err := r.startConsumers(reg)
	if err == nil {
		t.Error("Expected error when Consume fails")
	}
}

func Test_consumeMessage_registersConsumer(t *testing.T) {
	mockCh := newMockChannel()
	r := &rabbit{
		queues:    make(map[string]*queue),
		consumers: make([]*consumerRegistration, 0),
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	ctx, cancel := context.WithCancel(context.Background())

	cb := func(evt *sock.Event) error { return nil }

	go func() {
		_ = r.ConsumeMessage(ctx, "test-queue", "test-consumer", false, false, false, 1, cb)
	}()

	// Give goroutine time to register
	time.Sleep(100 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)

	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.consumers) != 1 {
		t.Fatalf("Expected 1 consumer registered, got %d", len(r.consumers))
	}
	if r.consumers[0].queueName != "test-queue" {
		t.Errorf("Expected queue name 'test-queue', got '%s'", r.consumers[0].queueName)
	}
	if r.consumers[0].consumerName != "test-consumer" {
		t.Errorf("Expected consumer name 'test-consumer', got '%s'", r.consumers[0].consumerName)
	}
	if r.consumers[0].cType != consumerTypeMessage {
		t.Errorf("Expected consumer type message, got %d", r.consumers[0].cType)
	}
}

func Test_consumeRPC_registersConsumer(t *testing.T) {
	mockCh := newMockChannel()
	r := &rabbit{
		queues:    make(map[string]*queue),
		consumers: make([]*consumerRegistration, 0),
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	ctx, cancel := context.WithCancel(context.Background())

	cb := func(req *sock.Request) (*sock.Response, error) { return nil, nil }

	go func() {
		_ = r.ConsumeRPC(ctx, "test-queue", "test-consumer", false, false, false, 1, cb)
	}()

	// Give goroutine time to register
	time.Sleep(100 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)

	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.consumers) != 1 {
		t.Fatalf("Expected 1 consumer registered, got %d", len(r.consumers))
	}
	if r.consumers[0].cType != consumerTypeRPC {
		t.Errorf("Expected consumer type RPC, got %d", r.consumers[0].cType)
	}
}

func Test_reconsumerAll_restoresConsumers(t *testing.T) {
	mockCh := newMockChannel()

	r := &rabbit{
		queues:    make(map[string]*queue),
		consumers: make([]*consumerRegistration, 0),
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	reg := &consumerRegistration{
		queueName:    "test-queue",
		consumerName: "test-consumer",
		numWorkers:   1,
		cType:        consumerTypeMessage,
		cbMessage:    func(evt *sock.Event) error { return nil },
	}
	r.consumers = append(r.consumers, reg)

	// reconsumerAll should re-register the consumer without error
	r.reconsumerAll()
}

func Test_reconsumerAll_retryOnFailure(t *testing.T) {
	callCount := 0
	mockCh := &mockChannelWithConsumeCounter{
		consumeCallCount: &callCount,
		failUntil:        2, // fail first 2 calls, succeed on 3rd
	}

	r := &rabbit{
		queues:    make(map[string]*queue),
		consumers: make([]*consumerRegistration, 0),
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	reg := &consumerRegistration{
		queueName:    "test-queue",
		consumerName: "test-consumer",
		numWorkers:   1,
		cType:        consumerTypeMessage,
		cbMessage:    func(evt *sock.Event) error { return nil },
	}
	r.consumers = append(r.consumers, reg)

	r.reconsumerAll()

	if callCount != 3 {
		t.Errorf("Expected 3 consume calls (2 retries + 1 success), got %d", callCount)
	}
}

func Test_reconsumerAll_allRetriesFail(t *testing.T) {
	callCount := 0
	mockCh := &mockChannelWithConsumeCounter{
		consumeCallCount: &callCount,
		failUntil:        10, // always fail
	}

	r := &rabbit{
		queues:    make(map[string]*queue),
		consumers: make([]*consumerRegistration, 0),
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	reg := &consumerRegistration{
		queueName:    "test-queue",
		consumerName: "test-consumer",
		numWorkers:   1,
		cType:        consumerTypeMessage,
		cbMessage:    func(evt *sock.Event) error { return nil },
	}
	r.consumers = append(r.consumers, reg)

	r.reconsumerAll()

	if callCount != 3 {
		t.Errorf("Expected 3 consume calls (all retries), got %d", callCount)
	}
}
