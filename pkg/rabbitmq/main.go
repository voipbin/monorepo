package rabbitmq

//go:generate mockgen -destination ./mock_rabbitmq_rabbit.go -package rabbitmq gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq Rabbit

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/streadway/amqp"
)

// Request struct
type Request struct {
	URI      string        `json:"uri"`
	Method   RequestMethod `json:"method"`
	DataType string        `json:"data_type"`
	Data     string        `json:"data"`
}

// Response struct
type Response struct {
	StatusCode int    `json:"status_code"`
	DataType   string `json:"data_type"`
	Data       string `json:"data"`
}

// Event struct
type Event struct {
	Type     EventType `json:"type"`
	DataType string    `json:"data_type"`
	Data     string    `json:"data"`
}

// RequestMethod type
type RequestMethod string

// List of RequestMethod
const (
	RequestMethodPost   RequestMethod = "POST"
	RequestMethodGet    RequestMethod = "GET"
	RequestMethodPut    RequestMethod = "PUT"
	RequestMethodDelete RequestMethod = "DELETE"
)

// EventType type
type EventType string

// List of EventType
const (
	EventTypeCall EventType = "cm_call"
)

// Rabbit defines rabbit queue interfaces
type Rabbit interface {
	Connect()
	Close()
	GetURL() string

	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool) error

	ConsumeMessage(queueName, consumerName string, messageConsume CbMsgConsume) error
	PublishMessage(queueName, message string) error
	ConsumeRPC(queueNqme, consumerName string, cbRPC CbMsgRPC) error
	PublishRPC(ctx context.Context, queueName string, req *Request) (*Response, error)
}

// rabbit struct for rabbitmq
type rabbit struct {
	uri string

	errorChannel chan *amqp.Error
	connection   *amqp.Connection
	closed       bool

	queues map[string]*queue
}

type queue struct {
	name       string
	durable    bool
	autoDelete bool
	exclusive  bool
	noWait     bool

	channel      *amqp.Channel
	channelQueue *amqp.Queue
}

// CbMsgConsume is func prototype for message read callback.
type CbMsgConsume func(*Event) error

// CbMsgRPC is func prototype for RPC callback
type CbMsgRPC func(*Request) (*Response, error)

// NewRabbit creates queue for Rabbitmq
func NewRabbit(uri string) Rabbit {
	res := &rabbit{
		uri:    uri,
		queues: make(map[string]*queue),
	}

	return res
}

// Connect connects to rabbitmq
func (r *rabbit) Connect() {
	r.connect()
	go r.reconnector()
}

// GetURL returns url
func (r *rabbit) GetURL() string {
	return r.uri
}

// Close close the Queue.
func (r *rabbit) Close() {
	log.WithFields(log.Fields{
		"url": r.uri,
	}).Info("Close the rabbitmq connection.")

	r.closed = true

	// close all queues
	for _, q := range r.queues {
		q.channel.Close()
	}
	r.connection.Close()
}

// receonnector reconnects the rabbitmq
func (r *rabbit) reconnector() {
	for {
		err := <-r.errorChannel
		if r.closed == false {
			log.Errorf("Reconnecting after connection closed. err: %v", err)
			r.connect()
		}
	}
}

// connect connects to rabbitmq.
func (r *rabbit) connect() {
	for {
		log := log.WithFields(log.Fields{
			"url": r.uri,
		})
		log.Debug("Connecting to rabbitmq")

		// connect
		conn, err := amqp.Dial(r.uri)
		if err != nil {
			log.Errorf("Could not connect to rabbitmq. Retrying after 1 sec. err: %v", err)
			time.Sleep(time.Second * 1)
			continue
		}

		r.connection = conn
		r.errorChannel = make(chan *amqp.Error)
		r.connection.NotifyClose(r.errorChannel)

		r.queuesRedeclare()

		log.Debug("Connection established to rabbitmq.")
		return
	}
}
