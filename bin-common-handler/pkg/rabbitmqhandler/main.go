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

	EventPublish(topic string, key string, evt *sock.Event) error
	EventPublishWithDelay(topic string, key string, evt *sock.Event, delay int) error

	RequestPublish(ctx context.Context, queueName string, req *sock.Request) (*sock.Response, error)
	RequestPublishWithDelay(key string, req *sock.Request, delay int) error

	QueueCreate(name string, queueType string) error
	QueueSubscribe(name string, topic string) error
}

// amqpChannel is an interface for amqp.Channel operations used by queue and exchange.
// This interface enables testing by allowing mock implementations.
// *amqp.Channel implicitly satisfies this interface.
type amqpChannel interface {
	Close() error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Qos(prefetchCount, prefetchSize int, global bool) error
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	QueueDelete(name string, ifUnused, ifEmpty, noWait bool) (int, error)
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
}

// amqpConnection is an interface for amqp.Connection operations.
// This interface enables testing by allowing mock implementations.
// *amqp.Connection implicitly satisfies this interface.
type amqpConnection interface {
	Channel() (*amqp.Channel, error)
	Close() error
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
}

// rabbit struct for rabbitmq
type rabbit struct {
	uri string

	errorChannel chan *amqp.Error
	connection   amqpConnection
	closed       atomic.Bool

	// mu protects concurrent access to queues, exchanges, and queueBinds maps.
	// Use RLock for reads and Lock for writes.
	mu         sync.RWMutex
	queues     map[string]*queue
	exchanges  map[string]*exchange
	queueBinds map[string]*queueBind
}

type queue struct {
	name       string
	durable    bool
	autoDelete bool
	exclusive  bool
	noWait     bool

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
		uri:        uri,
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string]*queueBind),
	}

	return res
}

// Connect connects to rabbitmq
func (r *rabbit) Connect() {
	r.connect()
	go r.reconnector()
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
		r.connection = conn

		// set error channel
		r.errorChannel = make(chan *amqp.Error)
		r.connection.NotifyClose(r.errorChannel)

		log.Debug("Connection established to rabbitmq.")
		return
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
	for _, b := range r.queueBinds {
		bindsCopy = append(bindsCopy, b)
	}
	r.mu.RUnlock()

	// redeclare the queues
	for _, queue := range queuesCopy {
		log.Debugf("Redeclaring the queue. queue: %s", queue.name)
		if err := r.QueueDeclare(queue.name, queue.durable, queue.autoDelete, queue.exclusive, queue.noWait); err != nil {
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
}
