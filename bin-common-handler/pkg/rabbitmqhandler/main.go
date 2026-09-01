package rabbitmqhandler

//go:generate mockgen -destination ./mock_rabbitmqhandler.go -package rabbitmqhandler -source ./main.go Rabbit

import (
	"context"
	"monorepo/bin-common-handler/models/sock"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// Rabbit defines rabbit queue interfaces
type Rabbit interface {
	Connect()
	Close()

	ConsumeMessage(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, messageConsume sock.CbMsgConsume) error
	ConsumeRPC(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cbConsume sock.CbMsgRPC) error

	TopicCreate(name string) error
	TopicCreateWithKind(name string, kind string) error

	EventPublish(topic string, key string, evt *sock.Event) error
	EventPublishWithDelay(topic string, key string, evt *sock.Event, delay int) error

	RequestPublish(ctx context.Context, queueName string, req *sock.Request) (*sock.Response, error)
	RequestPublishWithDelay(key string, req *sock.Request, delay int) error

	QueueCreate(name string, queueType string) error
	QueueSubscribe(name string, topic string) error
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	QueueUnbind(name, key, exchange string, args amqp.Table) error
}

// amqpChannel is an interface for amqp.Channel operations used by queue and exchange.
// This interface enables testing by allowing mock implementations.
// *amqp.Channel implicitly satisfies this interface.
type amqpChannel interface {
	Close() error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Qos(prefetchCount, prefetchSize int, global bool) error
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	QueueUnbind(name, key, exchange string, args amqp.Table) error
	QueueDelete(name string, ifUnused, ifEmpty, noWait bool) (int, error)
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	// PublishWithContext is required by publishExchange/RequestPublish/publishRPCErrorResponse
	// (consume.go, publish.go), which all open a channel via amqpConnection.Channel() and
	// publish on it. Added alongside the VOIP-1431 widening of amqpConnection.Channel()'s
	// return type from *amqp.Channel to amqpChannel -- see realConnection below.
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

// amqpConnection is an interface for amqp.Connection operations.
// This interface enables testing by allowing mock implementations.
// *amqp.Connection does NOT implicitly satisfy this interface as of VOIP-1431 -- Channel()
// returns amqpChannel (not the concrete *amqp.Channel) so that QueueBind/QueueUnbind's
// dedicated-channel fix (queue.go) can be exercised against a mock connection/channel pair in
// tests. Go has no return-type covariance, so *amqp.Connection's concrete
// `Channel() (*amqp.Channel, error)` no longer matches this signature -- realConnection below is
// the adapter that bridges the two. See docs/plans/2026-09-01-voip-1431-amqp-channel-race-design.md
// for the full rationale, including why this widening is a deliberate choice (to make the
// "QueueBind/QueueUnbind don't touch queue.channel" structural property unit-testable) rather
// than something the underlying race fix strictly requires.
type amqpConnection interface {
	Channel() (amqpChannel, error)
	Close() error
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
}

// realConnection adapts *amqp.Connection to amqpConnection's Channel() signature. Close() and
// NotifyClose() are inherited via embedding since *amqp.Connection's own signatures for those
// already match amqpConnection unchanged.
type realConnection struct {
	*amqp.Connection
}

// Channel opens a new channel and returns it as the amqpChannel interface. On error, returns an
// explicit nil interface -- not a typed-nil *amqp.Channel boxed into amqpChannel -- as
// defensive practice for any future caller that doesn't check err before consulting the channel
// (amqp091-go's own Connection.Channel() never actually returns a non-nil channel alongside a
// non-nil error, so this isn't fixing a currently-reachable bug in this package's own callers).
func (rc *realConnection) Channel() (amqpChannel, error) {
	ch, err := rc.Connection.Channel()
	if err != nil {
		return nil, err
	}
	return ch, nil
}

// rabbit struct for rabbitmq
type rabbit struct {
	uri string

	errorChannel        chan *amqp.Error
	connection          amqpConnection
	closed              atomic.Bool
	healthCheckInterval time.Duration

	// mu protects concurrent access to queues, exchanges, queueBinds, and consumers.
	// Use RLock for reads and Lock for writes.
	mu         sync.RWMutex
	queues     map[string]*queue
	exchanges  map[string]*exchange
	queueBinds map[string][]*queueBind
	consumers  []*consumerRegistration
}

type queue struct {
	name       string
	durable    bool
	autoDelete bool
	exclusive  bool
	noWait     bool
	args       amqp.Table

	channel amqpChannel
	queue   *amqp.Queue
}

type queueBind struct {
	name     string
	key      string
	exchange string
	noWait   bool
	args     amqp.Table
}

type consumerType int

const (
	consumerTypeMessage consumerType = iota
	consumerTypeRPC
)

type consumerRegistration struct {
	queueName    string
	consumerName string
	exclusive    bool
	noLocal      bool
	noWait       bool
	numWorkers   int
	cType        consumerType
	cbMessage    sock.CbMsgConsume
	cbRPC        sock.CbMsgRPC
}

type exchange struct {
	name string

	kind       string
	durable    bool
	autoDelete bool
	internal   bool
	noWait     bool
	args       amqp.Table

	channel amqpChannel
}

// NewRabbit creates queue for Rabbitmq
func NewRabbit(uri string) Rabbit {
	res := &rabbit{
		uri:                 uri,
		healthCheckInterval: 30 * time.Second,
		queues:              make(map[string]*queue),
		exchanges:           make(map[string]*exchange),
		queueBinds:          make(map[string][]*queueBind),
		consumers:           make([]*consumerRegistration, 0),
	}

	return res
}

// Connect connects to rabbitmq
func (r *rabbit) Connect() {
	r.connect()
	go r.reconnector()
	go r.healthChecker()
}

// Close close the Queue.
func (r *rabbit) Close() {
	logrus.WithFields(logrus.Fields{
		"url": r.uri,
	}).Info("Close the rabbitmq connection.")

	r.closed.Store(true)

	r.mu.RLock()
	// close all queue channels
	for _, q := range r.queues {
		if q.channel != nil {
			_ = q.channel.Close()
		}
	}

	// close all exchange channels
	for _, e := range r.exchanges {
		if e.channel != nil {
			_ = e.channel.Close()
		}
	}
	r.mu.RUnlock()

	_ = r.connection.Close()

	// Close error channel to signal reconnector goroutine to exit.
	// This must be done after connection.Close() to avoid race conditions.
	if r.errorChannel != nil {
		close(r.errorChannel)
	}
}

// reconnector monitors the connection and reconnects when the connection is lost.
// It exits when the rabbit is closed via Close().
func (r *rabbit) reconnector() {
	for {
		err, ok := <-r.errorChannel
		if !ok {
			// Channel closed, exit the goroutine
			return
		}
		if r.closed.Load() {
			// Rabbit is being closed, exit the goroutine
			return
		}
		logrus.Errorf("Reconnecting after connection closed. err: %v", err)
		r.connect()
		r.redeclareAll()
	}
}

// connect connects to rabbitmq.
func (r *rabbit) connect() {
	log := logrus.WithFields(logrus.Fields{
		"url": r.uri,
	})

	for {
		log.Debug("Connecting to rabbitmq")

		// connect
		conn, err := amqp.Dial(r.uri)
		if err != nil {
			log.Errorf("Could not connect to rabbitmq. Will retry again after 1 sec. err: %v", err)
			time.Sleep(time.Second * 1)
			continue
		}

		// r.connection is read from arbitrary caller goroutines via connectionGet()
		// (VOIP-1431's QueueBind/QueueUnbind, in particular, called from scopeRefCount at
		// arbitrary runtime moments). connect() itself is only ever invoked sequentially
		// (Connect() calls it once synchronously before spawning reconnector(); reconnector()
		// calls it one error at a time from its own single goroutine), so this lock protects
		// the write against those other readers, not against connect() re-entering itself.
		r.mu.Lock()
		r.connection = &realConnection{conn}
		r.mu.Unlock()

		// set error channel
		r.errorChannel = make(chan *amqp.Error)
		r.connection.NotifyClose(r.errorChannel)

		log.Debug("Connection established to rabbitmq.")
		return
	}
}

// connectionGet returns the current AMQP connection under a read lock. Pairs with connect()'s
// locked write above -- a read-side-only lock against an unlocked writer provides no
// synchronization at all. Used by QueueBind/QueueUnbind (queue.go), which read r.connection from
// arbitrary caller goroutines; this keeps that read from becoming a race in the same goroutine
// pairing (scopeRefCount vs. the reconnector() goroutine) VOIP-1431 exists to fix. The package's
// other nine callers of r.connection -- publishExchange, RequestPublish, executeConsumeRPC,
// publishRPCErrorResponse, ExchangeDeclare, QueueDeclare, checkConnection, Close(), and
// healthChecker() -- still read the field directly, not through this accessor; that remains a
// separate, pre-existing, out-of-scope data race (see the design doc).
func (r *rabbit) connectionGet() amqpConnection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.connection
}

// checkConnection probes the RabbitMQ connection by opening and closing a channel.
// Returns an error if the connection is dead.
func (r *rabbit) checkConnection() error {
	ch, err := r.connection.Channel()
	if err != nil {
		return err
	}
	if ch != nil {
		_ = ch.Close()
	}
	return nil
}

// healthChecker periodically probes the RabbitMQ connection and forces a
// reconnect if the connection is dead. This detects half-open TCP connections
// that NotifyClose cannot detect on its own.
func (r *rabbit) healthChecker() {
	ticker := time.NewTicker(r.healthCheckInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		if r.closed.Load() {
			return
		}
		if err := r.checkConnection(); err != nil {
			logrus.Errorf("Health check failed, forcing reconnect. err: %v", err)
			_ = r.connection.Close()
		}
	}
}

// redeclareAll recovers the all pre-defined queue/exchange/bind in the channel.
func (r *rabbit) redeclareAll() {
	log := logrus.WithField("func", "redeclareAll")

	// Take a snapshot of declarations to avoid holding lock during network operations.
	// QueueDeclare/ExchangeDeclare will acquire the lock when updating maps.
	r.mu.RLock()
	queuesCopy := make([]*queue, 0, len(r.queues))
	for _, q := range r.queues {
		queuesCopy = append(queuesCopy, q)
	}
	exchangesCopy := make([]*exchange, 0, len(r.exchanges))
	for _, e := range r.exchanges {
		exchangesCopy = append(exchangesCopy, e)
	}
	bindsCopy := make([]*queueBind, 0, len(r.queueBinds))
	for _, binds := range r.queueBinds { // now a [][]*queueBind value per key
		bindsCopy = append(bindsCopy, binds...)
	}
	r.mu.RUnlock()

	// redeclare the queues
	for _, queue := range queuesCopy {
		log.Debugf("Redeclaring the queue. queue: %s", queue.name)
		if err := r.QueueDeclare(queue.name, queue.durable, queue.autoDelete, queue.exclusive, queue.noWait, queue.args); err != nil {
			log.Errorf("Could not declare the queue. err: %v", err)
		}
	}

	// redeclare the exchanges
	for _, exchange := range exchangesCopy {
		log.Debugf("Redeclaring the exchange. exchage: %s", exchange.name)
		if err := r.ExchangeDeclare(exchange.name, exchange.kind, exchange.durable, exchange.autoDelete, exchange.internal, exchange.noWait, exchange.args); err != nil {
			log.Errorf("Could not declare the exchange. err: %v", err)
		}
	}

	// redeclare the binds
	for _, queueBind := range bindsCopy {
		logrus.Debugf("Redeclaring the bind. bind: %s", queueBind.name)
		if err := r.QueueBind(queueBind.name, queueBind.key, queueBind.exchange, queueBind.noWait, queueBind.args); err != nil {
			log.Errorf("Could not bind the queue. err: %v", err)
		}
	}

	// re-register consumers on the new channels
	r.reconsumerAll()
}

// reconsumerAll restores all registered consumers after reconnection.
// Called at the end of redeclareAll to re-register consumers on new channels.
func (r *rabbit) reconsumerAll() {
	log := logrus.WithField("func", "reconsumerAll")

	r.mu.RLock()
	consumersCopy := make([]*consumerRegistration, len(r.consumers))
	copy(consumersCopy, r.consumers)
	r.mu.RUnlock()

	for _, reg := range consumersCopy {
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			if err := r.startConsumers(reg); err != nil {
				lastErr = err
				log.Warnf("Could not re-register consumer (attempt %d/3). queue: %s, err: %v", attempt+1, reg.queueName, err)
				time.Sleep(time.Second)
				continue
			}
			log.Infof("Re-registered consumer. queue: %s, consumer: %s", reg.queueName, reg.consumerName)
			lastErr = nil
			break
		}
		if lastErr != nil {
			log.Errorf("Failed to re-register consumer after 3 attempts. queue: %s, err: %v", reg.queueName, lastErr)
		}
	}
}
