package rabbitmq

import (
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/streadway/amqp"
)

// Queue struct for rabbitmq
type Queue struct {
	url  string
	name string

	errorChannel chan *amqp.Error
	connection   *amqp.Connection
	channel      *amqp.Channel
	closed       bool
	durable      bool
}

// Request struct
type Request struct {
	URI      string `json:"uri"`
	Method   string `json:"method"`
	DataType string `json:"data_type"`
	Data     string `json:"data"`
}

// Response struct
type Response struct {
	StatusCode int    `json:"status_code"`
	DataType   string `json:"data_type"`
	Data       string `json:"data"`
}

// Event struct
type Event struct {
	Type     string `json:"type"`
	DataType string `json:"data_type"`
	Data     string `json:"data"`
}

// CbMsgConsume is func prototype for message read callback.
type CbMsgConsume func(string) error

// CbMsgRPC is func prototype for RPC callback
type CbMsgRPC func(string) (string, error)

// Initiate initiates rabbitmq package
func Initiate() {
	log.Infof("rabbitmq initiated.")
}

// NewQueue creates queue for Rabbitmq
func NewQueue(url string, qName string, durable bool) *Queue {
	q := new(Queue)
	q.url = url
	q.name = qName
	q.durable = durable

	q.connect()
	go q.reconnector()

	return q
}

// PublishMessage sends a message to rabbitmq
func (q *Queue) PublishMessage(m *Event) {

	message, err := json.Marshal(m)
	if err != nil {
		log.Errorf("Could not create a event message. err: %v", err)
	}

	if err := q.channel.Publish(
		"",     // exchange
		q.name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		}); err != nil {

		log.Errorf("Could not publish a event message. err: %v", err)
	}
}

// MessageConsume consumes message
func (q *Queue) MessageConsume(consumerName string, messageConsume CbMsgConsume) {
	deliveries, err := q.channel.Consume(
		q.name,       // queue
		consumerName, // messageConsumer
		true,         // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		log.Errorf("Could not consume the message. err: %v", err)
	}

	// process message
	go func() {
		for delivery := range deliveries {
			err := messageConsume(string(delivery.Body))
			if err != nil {
				log.Errorf("Message consumer returns error. err: %v", err)
			}
		}
	}()
}

// ConsumeRPC consumes RPC message
// it doesn't do any concurrent work at here.
// that guarnatees queued job will be processed in an order.
func (q *Queue) ConsumeRPC(consumerName string, cbHandler CbMsgRPC) {

	messages, err := q.channel.Consume(
		q.name,       // queue
		consumerName, // messageConsumer
		true,         // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		log.Errorf("Could not consume the message. err: %v", err)
		return
	}

	for m := range messages {
		// execute callback
		res, err := cbHandler(string(m.Body))
		if err != nil {
			log.Errorf("Message consumer returns error. err: %v", err)
			continue
		}

		// reply response
		err = q.channel.Publish(
			"",        // exchange
			m.ReplyTo, // routing key
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				ContentType:   "text/plain",
				CorrelationId: m.CorrelationId,
				Body:          []byte(res),
			})
		if err != nil {
			log.Errorf("Could not publish the response. err: %v", err)
		}
	}
}

// Close close the Queue.
func (q *Queue) Close() {
	log.Infof("Close the rabbitmq queue. name: %s", q.name)
	q.closed = true
	q.channel.Close()
	q.connection.Close()
}

// receonnector reconnects the rabbitmq
func (q *Queue) reconnector() {
	for {
		err := <-q.errorChannel
		if q.closed == false {
			log.Errorf("Reconnecting after connection closed. err: %v", err)
			q.connect()
		}
	}
}

// connect connects to rabbitmq.
func (q *Queue) connect() {
	for {
		log := log.WithFields(log.Fields{
			"url":  q.url,
			"namq": q.name,
		})
		log.Debug("Connecting to rabbitmq")

		// connect
		conn, err := amqp.Dial(q.url)
		if err != nil {
			log.Errorf("Could not connect to rabbitmq. Retrying after 1 sec. err: %v", err)
			time.Sleep(time.Second * 1)
			continue
		}

		q.connection = conn
		q.errorChannel = make(chan *amqp.Error)
		q.connection.NotifyClose(q.errorChannel)
		log.Debug("Connection established.")

		q.openChannel()
		q.declareQueue()

		return
	}
}

// declareQueue declares the rabbitmq queue using name.
func (q *Queue) declareQueue() {
	_, err := q.channel.QueueDeclare(
		q.name,    // name
		q.durable, // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Errorf("Could not declare the queue. err: %v", err)
	}
}

// openChannel opens a rabbitmq channel and sets to channel.
func (q *Queue) openChannel() {
	channel, err := q.connection.Channel()
	if err != nil {
		log.Errorf("Could not open the channel. err: %v", err)
	}
	q.channel = channel
}
