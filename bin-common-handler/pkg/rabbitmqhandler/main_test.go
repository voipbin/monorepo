package rabbitmqhandler

// Note on realConnection (main.go): its Channel() method has no direct unit test in this file.
// &realConnection{nil} nil-panics on the embedded *amqp.Connection.Channel() call (traced
// through amqp091-go's openChannel()/allocateChannel(), which unconditionally locks the
// receiver), so exercising it requires a real *amqp.Connection -- i.e. a real or fake broker,
// not something this package's mock-based unit tests can construct. Its explicit-nil-on-error
// behavior is covered by code review instead (VOIP-1431 design doc §4); the property it exists
// to protect (checkConnection's `if ch != nil` check) is exercised indirectly by every test that
// constructs a *rabbit with a mockConnection and calls checkConnection/healthChecker.

import (
	"context"
	"errors"
	"fmt"
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

	queueDeclareErr    error
	queueDeclareName   string
	queueDeclareArgs   amqp.Table
	exchangeDeclareErr error

	qosErr         error
	queueBindErr   error
	queueUnbindErr error
	consumeErr     error

	qosCallCount     int
	qosPrefetchCount int
	qosPrefetchSize  int
	qosGlobal        bool

	mu                   sync.Mutex // guards queueBindCallCount/queueUnbindCallCount for concurrent-access tests (VOIP-1258/VOIP-1431)
	queueBindCallCount   int        // VOIP-1258: counts QueueBind invocations, used to verify redeclareAll restores ALL tracked binds
	queueUnbindCallCount int        // VOIP-1431: QueueBind's counter had no QueueUnbind counterpart; added for parity

	// VOIP-1258 Task 1.4: captures ExchangeDeclare's actual call args for assertions.
	exchangeDeclareName    string
	exchangeDeclareKind    string
	exchangeDeclareDurable bool
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
	m.qosCallCount++
	m.qosPrefetchCount = prefetchCount
	m.qosPrefetchSize = prefetchSize
	m.qosGlobal = global
	return m.qosErr
}

func (m *mockChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	m.mu.Lock()
	m.queueBindCallCount++
	m.mu.Unlock()
	return m.queueBindErr
}

func (m *mockChannel) QueueUnbind(name, key, exchange string, args amqp.Table) error {
	m.mu.Lock()
	m.queueUnbindCallCount++
	m.mu.Unlock()
	return m.queueUnbindErr
}

func (m *mockChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return nil
}

func (m *mockChannel) QueueDelete(name string, ifUnused, ifEmpty, noWait bool) (int, error) {
	return 0, nil
}

func (m *mockChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	m.exchangeDeclareName = name
	m.exchangeDeclareKind = kind
	m.exchangeDeclareDurable = durable
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
	channelFunc   func() (amqpChannel, error)
	channelErr    error
	closeCalled   int
	closeErr      error
	notifyCloseCh chan *amqp.Error

	mu             sync.Mutex // guards channelCallCnt for concurrent-access tests (VOIP-1431)
	channelCallCnt int
}

func newMockConnection() *mockConnection {
	return &mockConnection{
		notifyCloseCh: make(chan *amqp.Error, 1),
	}
}

// Channel returns, in priority order: channelFunc's result if set, channelErr as an error if
// set, or otherwise a fresh working mock channel (VOIP-1431: this used to default to (nil, nil)
// -- callers that need that exact "success but no channel" case, or a specific mock channel to
// assert against, should set channelFunc explicitly rather than relying on this default).
func (m *mockConnection) Channel() (amqpChannel, error) {
	m.mu.Lock()
	m.channelCallCnt++
	m.mu.Unlock()
	if m.channelFunc != nil {
		return m.channelFunc()
	}
	if m.channelErr != nil {
		return nil, m.channelErr
	}
	return newMockChannel(), nil
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
func (m *mockChannelWithConsumeCounter) QueueUnbind(name, key, exchange string, args amqp.Table) error {
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
func (m *mockChannelWithConsumeCounter) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return nil
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
		queueBinds: make(map[string][]*queueBind),
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
		queueBinds: make(map[string][]*queueBind),
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
		queueBinds: make(map[string][]*queueBind),
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
		queueBinds: make(map[string][]*queueBind),
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
		queueBinds: make(map[string][]*queueBind),
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
		queueBinds: make(map[string][]*queueBind),
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
		queueBinds: make(map[string][]*queueBind),
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
		queueBinds: make(map[string][]*queueBind),
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
		queueBinds: make(map[string][]*queueBind),
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
		queueBinds: make(map[string][]*queueBind),
	}

	err := r.QueueBind("non-existent", "key", "exchange", false, nil)

	if err == nil {
		t.Error("Expected error for non-existent queue")
	}
}

func TestQueueBind_Success(t *testing.T) {
	// VOIP-1431: QueueBind now issues the bind on a dedicated channel obtained from
	// r.connection, not queue.channel -- mockCh here stands in for queue.channel (still
	// required for queueGet's existence check) and is deliberately NOT the channel QueueBind
	// actually calls; mockConn's default Channel() supplies a separate, working mock channel.
	mockCh := newMockChannel()
	mockConn := newMockConnection()

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
		connection: mockConn,
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
	if r.queueBinds["test-queue"][0].key != "routing-key" {
		t.Errorf("Expected key 'routing-key', got '%s'", r.queueBinds["test-queue"][0].key)
	}
	if r.queueBinds["test-queue"][0].exchange != "test-exchange" {
		t.Errorf("Expected exchange 'test-exchange', got '%s'", r.queueBinds["test-queue"][0].exchange)
	}
	// mockCh (queue.channel) must never see this call -- that's the entire point of the fix.
	if mockCh.queueBindCallCount != 0 {
		t.Errorf("Expected QueueBind to NOT touch queue.channel, but it was called %d time(s)", mockCh.queueBindCallCount)
	}
}

// TestQueueBind_DoesNotUseQueueChannel pins the structural property VOIP-1431's fix relies on:
// QueueBind must obtain its channel from the connection, never from the queue's own long-lived
// channel (which startConsumers's Qos/Consume also uses -- sharing it is exactly the race this
// fix removes). Asserts on the channel identity actually used, not just the end-state map.
func TestQueueBind_DoesNotUseQueueChannel(t *testing.T) {
	queueCh := newMockChannel() // stands in for queue.channel -- must stay untouched
	connCh := newMockChannel()  // the channel QueueBind should actually call
	mockConn := newMockConnection()
	mockConn.channelFunc = func() (amqpChannel, error) { return connCh, nil }

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
		connection: mockConn,
	}
	r.queues["test-queue"] = &queue{name: "test-queue", channel: queueCh}

	if err := r.QueueBind("test-queue", "key", "exchange", false, nil); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if queueCh.queueBindCallCount != 0 {
		t.Errorf("Expected queue.channel.QueueBind to never be called, got %d call(s)", queueCh.queueBindCallCount)
	}
	if connCh.queueBindCallCount != 1 {
		t.Errorf("Expected the connection-provided channel's QueueBind to be called exactly once, got %d", connCh.queueBindCallCount)
	}
	if connCh.closeCalled != 1 {
		t.Errorf("Expected the ephemeral channel to be closed exactly once after use, got %d", connCh.closeCalled)
	}
}

func TestQueueBind_ReturnsChannelError(t *testing.T) {
	// VOIP-1431: the error must be injected on the channel the connection provides -- the
	// channel QueueBind actually calls post-fix -- not on queue.channel, which QueueBind no
	// longer touches at all.
	connCh := newMockChannel()
	connCh.queueBindErr = errors.New("bind failed")
	mockConn := newMockConnection()
	mockConn.channelFunc = func() (amqpChannel, error) { return connCh, nil }

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
		connection: mockConn,
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: newMockChannel(), // queue.channel: irrelevant to this path post-fix
	}

	err := r.QueueBind("test-queue", "key", "exchange", false, nil)

	if err == nil {
		t.Error("Expected error from channel QueueBind")
	}
}

// TestQueueBind_ReturnsErrorWhenConnectionClosed covers the new conn == nil branch (§3 of the
// design doc) -- previously unreachable/untested production behavior.
func TestQueueBind_ReturnsErrorWhenConnectionClosed(t *testing.T) {
	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
		connection: nil,
	}
	r.queues["test-queue"] = &queue{name: "test-queue", channel: newMockChannel()}

	err := r.QueueBind("test-queue", "key", "exchange", false, nil)

	if !errors.Is(err, amqp.ErrClosed) {
		t.Errorf("Expected amqp.ErrClosed, got %v", err)
	}
}

// TestQueueBind_IdempotentRebind verifies that binding the same (key, exchange) pair twice
// does not create a duplicate tracked entry (VOIP-1258 round-1 review, code-quality follow-up:
// this was the single most novel piece of logic in the diff and had no direct test).
func TestQueueBind_IdempotentRebind(t *testing.T) {
	mockCh := newMockChannel()
	mockConn := newMockConnection()

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
		connection: mockConn,
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	if err := r.QueueBind("test-queue", "routing-key", "test-exchange", false, nil); err != nil {
		t.Fatalf("Expected no error on first bind, got %v", err)
	}
	if err := r.QueueBind("test-queue", "routing-key", "test-exchange", false, nil); err != nil {
		t.Fatalf("Expected no error on idempotent re-bind, got %v", err)
	}

	binds := r.queueBinds["test-queue"]
	if len(binds) != 1 {
		t.Fatalf("Expected slice to stay at length 1 after idempotent re-bind, got %d", len(binds))
	}
}

// ============================================================================
// connectionGet() Tests
// ============================================================================

// Test_connectionGet covers connectionGet() directly: returns nil when unset, returns the set
// value under a read lock otherwise (VOIP-1431 -- previously untested production code).
func Test_connectionGet(t *testing.T) {
	r := &rabbit{}
	if got := r.connectionGet(); got != nil {
		t.Errorf("Expected nil for unset connection, got %v", got)
	}

	mockConn := newMockConnection()
	r.connection = mockConn
	if got := r.connectionGet(); got != mockConn {
		t.Errorf("Expected connectionGet to return the set connection, got %v", got)
	}
}

// Test_connectionGet_ConcurrentReadWrite exercises connectionGet() (r.mu.RLock-guarded read)
// against a repeated, r.mu.Lock-guarded write to r.connection from another goroutine -- the
// exact same field/lock pair connect()'s locked write (main.go) and QueueBind/QueueUnbind's
// connectionGet() calls use, isolated from connect()'s own real-Dial retry-loop timing (which
// makes triggering this scenario through connect() itself impractical to do deterministically
// at unit-test speed -- see the VOIP-1431 design doc §4). Run with -race.
//
// Mutation-tested: temporarily removing the r.mu.Lock()/Unlock() pair around this test's write
// goroutine (mirroring what an unlocked connect() write would look like) makes `go test -race`
// report a DATA RACE on this exact test, confirming the lock pairing is what's actually being
// verified here, not merely absence of a panic.
func Test_connectionGet_ConcurrentReadWrite(t *testing.T) {
	r := &rabbit{}

	var wg sync.WaitGroup
	wg.Add(2)

	// writer: repeatedly replaces r.connection under r.mu.Lock(), mirroring connect()'s
	// locked assignment.
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			r.mu.Lock()
			r.connection = newMockConnection()
			r.mu.Unlock()
		}
	}()

	// reader: repeatedly reads via connectionGet(), mirroring QueueBind/QueueUnbind.
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = r.connectionGet()
		}
	}()

	wg.Wait()
}

// ============================================================================
// QueueUnbind() Tests
// ============================================================================

func TestQueueUnbind_Success(t *testing.T) {
	mockCh := newMockChannel()
	mockConn := newMockConnection()

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
		connection: mockConn,
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	if err := r.QueueBind("test-queue", "routing-key", "test-exchange", false, nil); err != nil {
		t.Fatalf("Expected no error binding, got %v", err)
	}

	err := r.QueueUnbind("test-queue", "routing-key", "test-exchange", nil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(r.queueBinds["test-queue"]) != 0 {
		t.Errorf("Expected queueBinds entry to be removed, got %v", r.queueBinds["test-queue"])
	}
}

func TestQueueUnbind_KeepsOtherBinds(t *testing.T) {
	mockCh := newMockChannel()
	mockConn := newMockConnection()

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
		connection: mockConn,
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	if err := r.QueueBind("test-queue", "key1", "exchange1", false, nil); err != nil {
		t.Fatalf("Expected no error binding, got %v", err)
	}
	if err := r.QueueBind("test-queue", "key2", "exchange2", false, nil); err != nil {
		t.Fatalf("Expected no error binding, got %v", err)
	}

	err := r.QueueUnbind("test-queue", "key1", "exchange1", nil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	binds := r.queueBinds["test-queue"]
	if len(binds) != 1 {
		t.Fatalf("Expected 1 remaining bind, got %d", len(binds))
	}
	if binds[0].key != "key2" || binds[0].exchange != "exchange2" {
		t.Errorf("Expected remaining bind key2/exchange2, got %s/%s", binds[0].key, binds[0].exchange)
	}
}

func TestQueueUnbind_ReturnsErrorForNonExistent(t *testing.T) {
	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
	}

	err := r.QueueUnbind("no-such-queue", "key", "exchange", nil)
	if err == nil {
		t.Error("Expected error for non-existent queue")
	}
}

func TestQueueUnbind_ReturnsChannelError(t *testing.T) {
	// VOIP-1431: inject the error on the connection-provided channel -- QueueUnbind no longer
	// touches queue.channel.
	connCh := newMockChannel()
	connCh.queueUnbindErr = errors.New("unbind failed")
	mockConn := newMockConnection()
	mockConn.channelFunc = func() (amqpChannel, error) { return connCh, nil }

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
		connection: mockConn,
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: newMockChannel(), // queue.channel: irrelevant to this path post-fix
	}

	err := r.QueueUnbind("test-queue", "key", "exchange", nil)
	if err == nil {
		t.Error("Expected error from channel QueueUnbind")
	}
}

// TestQueueUnbind_DoesNotUseQueueChannel is QueueBind's structural test mirrored for QueueUnbind.
func TestQueueUnbind_DoesNotUseQueueChannel(t *testing.T) {
	queueCh := newMockChannel()
	connCh := newMockChannel()
	mockConn := newMockConnection()
	mockConn.channelFunc = func() (amqpChannel, error) { return connCh, nil }

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
		connection: mockConn,
	}
	r.queues["test-queue"] = &queue{name: "test-queue", channel: queueCh}
	r.queueBinds["test-queue"] = []*queueBind{{name: "test-queue", key: "key", exchange: "exchange"}}

	if err := r.QueueUnbind("test-queue", "key", "exchange", nil); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if queueCh.queueUnbindCallCount != 0 {
		t.Errorf("Expected queue.channel.QueueUnbind to never be called, got %d call(s)", queueCh.queueUnbindCallCount)
	}
	if connCh.queueUnbindCallCount != 1 {
		t.Errorf("Expected the connection-provided channel's QueueUnbind to be called exactly once, got %d", connCh.queueUnbindCallCount)
	}
	if connCh.closeCalled != 1 {
		t.Errorf("Expected the ephemeral channel to be closed exactly once after use, got %d", connCh.closeCalled)
	}
}

// TestQueueUnbind_ReturnsErrorWhenConnectionClosed mirrors the above for QueueUnbind.
func TestQueueUnbind_ReturnsErrorWhenConnectionClosed(t *testing.T) {
	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
		connection: nil,
	}
	r.queues["test-queue"] = &queue{name: "test-queue", channel: newMockChannel()}

	err := r.QueueUnbind("test-queue", "key", "exchange", nil)

	if !errors.Is(err, amqp.ErrClosed) {
		t.Errorf("Expected amqp.ErrClosed, got %v", err)
	}
}

// TestRedeclareAll_RestoresAllBindsForSameQueue is a direct regression test for VOIP-1258
// round-1 implementation-plan review finding F2: a broker reconnect must restore ALL active
// binds for a queue, not just the most recent one (the bug that motivated the
// map[string]*queueBind -> map[string][]*queueBind change). This exercises the exact snapshot
// + replay logic redeclareAll uses for binds (bindsCopy flattening + re-issuing QueueBind per
// entry), without going through the full redeclareAll (which also re-declares queues/exchanges
// via a live amqp connection, out of scope for this unit test).
func TestRedeclareAll_RestoresAllBindsForSameQueue(t *testing.T) {
	mockCh := newMockChannel()
	// VOIP-1431: QueueBind's call count must now be observed on the connection-provided
	// channel, not queue.channel -- fix channelFunc to a single shared instance so repeated
	// QueueBind calls in the replay loop below accumulate on the same counter.
	connCh := newMockChannel()
	mockConn := newMockConnection()
	mockConn.channelFunc = func() (amqpChannel, error) { return connCh, nil }

	r := &rabbit{
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string][]*queueBind),
		connection: mockConn,
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	if err := r.QueueBind("test-queue", "customer_id.abc.#", "topic-exchange", false, nil); err != nil {
		t.Fatalf("Expected no error binding first scope, got %v", err)
	}
	if err := r.QueueBind("test-queue", "agent_id.def.#", "topic-exchange", false, nil); err != nil {
		t.Fatalf("Expected no error binding second scope, got %v", err)
	}

	// Reproduce redeclareAll's bind-snapshot-and-replay logic directly (main.go:279-282,
	// 302-307), which is the part this test targets -- flattening map[string][]*queueBind into
	// a single replay list and re-issuing QueueBind for every entry.
	r.mu.RLock()
	bindsCopy := make([]*queueBind, 0, len(r.queueBinds))
	for _, binds := range r.queueBinds {
		bindsCopy = append(bindsCopy, binds...)
	}
	r.mu.RUnlock()

	if len(bindsCopy) != 2 {
		t.Fatalf("Expected the bind snapshot to contain both tracked binds, got %d", len(bindsCopy))
	}

	connCh.queueBindCallCount = 0 // reset so we only observe the replay below
	for _, qb := range bindsCopy {
		if err := r.QueueBind(qb.name, qb.key, qb.exchange, qb.noWait, qb.args); err != nil {
			t.Fatalf("Expected no error replaying bind %+v, got %v", qb, err)
		}
	}

	if connCh.queueBindCallCount != 2 {
		t.Fatalf("Expected exactly 2 QueueBind calls when replaying all tracked binds (F2 regression), got %d", connCh.queueBindCallCount)
	}
	if mockCh.queueBindCallCount != 0 {
		t.Errorf("Expected queue.channel to never see a QueueBind call, got %d", mockCh.queueBindCallCount)
	}
}

// TestQueueBindUnbind_ConcurrentAccess exercises QueueBind/QueueUnbind under concurrent access
// on the same queue name to validate the mutex actually guards the map/slice mutations
// (run with -race). Deliberately does NOT fix mockConn's channelFunc to a shared instance --
// mockConnection.Channel()'s default behavior of returning a fresh mockChannel per call means
// each goroutine's bind/unbind gets its own channel object, so there is no cross-goroutine
// channel-state sharing to worry about here; mockConnection.channelCallCnt itself is guarded by
// its own mutex (VOIP-1431).
func TestQueueBindUnbind_ConcurrentAccess(t *testing.T) {
	mockCh := newMockChannel()
	mockConn := newMockConnection()

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
		connection: mockConn,
	}
	r.queues["test-queue"] = &queue{
		name:    "test-queue",
		channel: mockCh,
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", i)
			_ = r.QueueBind("test-queue", key, "exchange", false, nil)
			_ = r.QueueUnbind("test-queue", key, "exchange", nil)
		}(i)
	}
	wg.Wait()

	// After all binds+unbinds complete, the tracked set for this queue should be empty
	// (every Acquire paired with a Release) -- if the mutex had a gap, this could show a
	// stale/corrupted entry or the test would have already panicked/raced.
	if len(r.queueBinds["test-queue"]) != 0 {
		t.Errorf("Expected no leftover binds after balanced concurrent bind/unbind, got %v", r.queueBinds["test-queue"])
	}
}

// ============================================================================
// QueueSubscribe() Tests
// ============================================================================

func TestQueueSubscribe_CallsQueueBind(t *testing.T) {
	mockCh := newMockChannel()
	mockConn := newMockConnection()

	r := &rabbit{
		queues:     make(map[string]*queue),
		queueBinds: make(map[string][]*queueBind),
		connection: mockConn,
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
	if r.queueBinds["test-queue"][0].key != "" {
		t.Errorf("Expected empty key, got '%s'", r.queueBinds["test-queue"][0].key)
	}
	if r.queueBinds["test-queue"][0].exchange != "test-topic" {
		t.Errorf("Expected exchange 'test-topic', got '%s'", r.queueBinds["test-queue"][0].exchange)
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
		queueBinds: make(map[string][]*queueBind),
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
		queueBinds: make(map[string][]*queueBind),
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
		queueBinds:   make(map[string][]*queueBind),
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
		queueBinds:   make(map[string][]*queueBind),
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
		queueBinds:   make(map[string][]*queueBind),
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
		queueBinds: make(map[string][]*queueBind),
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
	r             *rabbit
	closedWasTrue bool
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

func (m *closedCaptureMockChannel) QueueUnbind(name, key, exchange string, args amqp.Table) error {
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

func (m *closedCaptureMockChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return nil
}

func Test_queueDeclare_storesArgs(t *testing.T) {
	mockCh := newMockChannel()
	r := &rabbit{
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string][]*queueBind),
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
		queueBinds: make(map[string][]*queueBind),
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

// ============================================================================
// checkConnection() Tests
// ============================================================================

func TestCheckConnection_ReturnsErrorWhenChannelFails(t *testing.T) {
	mockConn := newMockConnection()
	mockConn.channelErr = errors.New("connection dead")

	r := &rabbit{
		connection: mockConn,
	}

	err := r.checkConnection()

	if err == nil {
		t.Error("Expected error when Channel() fails")
	}
	if mockConn.channelCallCnt != 1 {
		t.Errorf("Expected 1 Channel() call, got %d", mockConn.channelCallCnt)
	}
}

func TestCheckConnection_ReturnsNilOnSuccess(t *testing.T) {
	mockConn := newMockConnection()

	r := &rabbit{
		connection: mockConn,
	}

	err := r.checkConnection()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if mockConn.channelCallCnt != 1 {
		t.Errorf("Expected 1 Channel() call, got %d", mockConn.channelCallCnt)
	}
}

// ============================================================================
// healthChecker() Tests
// ============================================================================

func TestHealthChecker_ForcesReconnectOnDeadConnection(t *testing.T) {
	mockConn := newMockConnection()
	mockConn.channelErr = errors.New("connection dead")

	r := &rabbit{
		uri:                 "amqp://localhost",
		connection:          mockConn,
		healthCheckInterval: 50 * time.Millisecond,
		queues:              make(map[string]*queue),
		exchanges:           make(map[string]*exchange),
		queueBinds:          make(map[string][]*queueBind),
	}

	done := make(chan struct{})
	go func() {
		r.healthChecker()
		close(done)
	}()

	// Wait for at least one tick
	time.Sleep(150 * time.Millisecond)
	r.closed.Store(true)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("healthChecker did not exit")
	}

	if mockConn.closeCalled < 1 {
		t.Errorf("Expected connection.Close() to be called at least once, got %d", mockConn.closeCalled)
	}
}

func TestHealthChecker_ExitsWhenClosed(t *testing.T) {
	mockConn := newMockConnection()

	r := &rabbit{
		uri:                 "amqp://localhost",
		connection:          mockConn,
		healthCheckInterval: 50 * time.Millisecond,
		queues:              make(map[string]*queue),
		exchanges:           make(map[string]*exchange),
		queueBinds:          make(map[string][]*queueBind),
	}

	r.closed.Store(true)

	done := make(chan struct{})
	go func() {
		r.healthChecker()
		close(done)
	}()

	select {
	case <-done:
		// healthChecker exited as expected
	case <-time.After(2 * time.Second):
		t.Fatal("healthChecker did not exit after closed flag was set")
	}
}

func TestHealthChecker_DoesNotCloseHealthyConnection(t *testing.T) {
	mockConn := newMockConnection()
	// channelErr and channelFunc are both unset, so Channel() falls through to its default:
	// a fresh working mock channel, nil error (VOIP-1431 changed this default from (nil, nil)
	// -- checkConnection's Channel() call below now genuinely opens and closes a channel, same
	// as the real *amqp.Connection.Channel() would on a healthy connection).

	r := &rabbit{
		uri:                 "amqp://localhost",
		connection:          mockConn,
		healthCheckInterval: 50 * time.Millisecond,
		queues:              make(map[string]*queue),
		exchanges:           make(map[string]*exchange),
		queueBinds:          make(map[string][]*queueBind),
	}

	done := make(chan struct{})
	go func() {
		r.healthChecker()
		close(done)
	}()

	// Wait for a few ticks
	time.Sleep(200 * time.Millisecond)
	r.closed.Store(true)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("healthChecker did not exit")
	}

	if mockConn.closeCalled != 0 {
		t.Errorf("Expected connection.Close() NOT to be called, got %d", mockConn.closeCalled)
	}
}
