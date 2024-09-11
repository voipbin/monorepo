package rabbitmqhandler

//go:generate mockgen -destination ./mock_rabbitmqhandler.go -package rabbitmqhandler -source ./main.go Rabbit

import (
	"context"
	"monorepo/bin-common-handler/models/sock"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// Rabbit defines rabbit queue interfaces
type Rabbit interface {
	Connect()
	Close()

	ConsumeMessage(queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, messageConsume sock.CbMsgConsume) error
	ConsumeRPC(queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cbConsume sock.CbMsgRPC) error

	TopicCreate(name string) error

	EventPublish(topic string, key string, evt *sock.Event) error
	EventPublishWithDelay(topic string, key string, evt *sock.Event, delay int) error

	RequestPublish(ctx context.Context, queueName string, req *sock.Request) (*sock.Response, error)
	RequestPublishWithDelay(key string, req *sock.Request, delay int) error

	QueueCreate(name string, queueType string) error
	QueueSubscribe(name string, topic string) error
}

// rabbit struct for rabbitmq
type rabbit struct {
	uri string

	errorChannel chan *amqp.Error
	connection   *amqp.Connection
	closed       bool

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

	channel *amqp.Channel
	qeueue  *amqp.Queue
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

	channel *amqp.Channel
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

	r.closed = true
	r.connection.Close()
}

// receonnector reconnects the rabbitmq
func (r *rabbit) reconnector() {
	for {
		err := <-r.errorChannel
		if !r.closed {
			logrus.Errorf("Reconnecting after connection closed. err: %v", err)
			r.connect()
			r.redeclareAll()
		}
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

	// redeclare the queues
	for _, queue := range r.queues {
		log.Debugf("Redeclaring the queue. queue: %s", queue.name)
		if err := r.QueueDeclare(queue.name, queue.durable, queue.autoDelete, queue.exclusive, queue.noWait); err != nil {
			log.Errorf("Could not declare the queue. err: %v", err)
		}
	}

	// redeclare the exchanges
	for _, exchange := range r.exchanges {
		log.Debugf("Redeclaring the exchange. exchage: %s", exchange.name)
		if err := r.ExchangeDeclare(exchange.name, exchange.kind, exchange.durable, exchange.autoDelete, exchange.internal, exchange.noWait, exchange.args); err != nil {
			log.Errorf("Could not declare the exchange. err: %v", err)
		}
	}

	// redeclare the binds
	for _, queueBind := range r.queueBinds {
		logrus.Debugf("Redeclaring the bind. bind: %s", queueBind.name)
		if err := r.QueueBind(queueBind.name, queueBind.key, queueBind.exchange, queueBind.noWait, queueBind.args); err != nil {
			log.Errorf("Could not bind the queue. err: %v", err)
		}
	}
}
