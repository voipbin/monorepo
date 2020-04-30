package rabbitmq

//go:generate mockgen -destination ./mock_rabbitmq_rabbit.go -package rabbitmq gitlab.com/voipbin/bin-manager/flow-manager/pkg/rabbitmq Rabbit

import (
	"context"
	"encoding/json"
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

// RequestMethod type
type RequestMethod string

// List of RequestMethod
const (
	RequestMethodPost   RequestMethod = "POST"
	RequestMethodGet    RequestMethod = "GET"
	RequestMethodPut    RequestMethod = "PUT"
	RequestMethodDelete RequestMethod = "DELETE"
)

// Rabbit defines rabbit queue interfaces
type Rabbit interface {
	Connect()
	Close()
	GetURL() string

	DeclareQueue(name string, durable, autoDelete, exclusive, noWait bool) error

	ConsumeMessage(queueName, consumerName string, messageConsume CbMsgConsume) error
	PublishMessage(queueName, message string) error
	ConsumeRPC(queueNqme, consumerName string, cbRPC CbMsgRPC) error
	PublishRPC(ctx context.Context, queueName, message string) ([]byte, error)
}

// rabbit struct for rabbitmq
type rabbit struct {
	url string

	errorChannel chan *amqp.Error
	connection   *amqp.Connection
	channel      *amqp.Channel
	closed       bool

	queues map[string]amqp.Queue
}

// CbMsgConsume is func prototype for message read callback.
type CbMsgConsume func(*Request) (*Response, error)

// CbMsgRPC is func prototype for RPC callback
type CbMsgRPC func(string) (string, error)

// NewRabbit creates queue for Rabbitmq
func NewRabbit(url string) Rabbit {
	q := new(rabbit)
	q.url = url
	q.queues = make(map[string]amqp.Queue)

	return q
}

// Connect connects to rabbitmq
func (q *rabbit) Connect() {
	q.connect()
	go q.reconnector()
}

// GetURL returns url
func (q *rabbit) GetURL() string {
	return q.url
}

// PublishMessage sends a message to rabbitmq
func (q *rabbit) PublishMessage(queueName, message string) error {
	queue, err := q.getQueue(queueName)
	if err != nil {
		return err
	}

	err = q.channel.Publish(
		"",         // exchange
		queue.Name, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		})
	if err != nil {
		log.Errorf("Could not send a message. err: %v", err)
		return err
	}

	return nil
}

// PublishRPC publishes RPC message and returns response.
func (q *rabbit) PublishRPC(ctx context.Context, queueName, message string) ([]byte, error) {
	log.WithFields(log.Fields{
		"name":    queueName,
		"message": message,
	}).Info("Publish message to RPC.")

	queue, err := q.getQueue(queueName)
	if err != nil {
		return nil, err
	}

	// declare the tmp queue for rpc response
	tmpQueue, err := q.declareQueue("", false, true, true, false)
	if err != nil {
		return nil, err
	}

	// consuming the message from the tmpQueue
	chanRes, err := q.channel.Consume(
		tmpQueue.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		q.DeleteQueue(tmpQueue.Name, false, false, false)
		return nil, err
	}

	// publish the message
	err = q.channel.Publish(
		"",
		queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			ReplyTo:     tmpQueue.Name,
			Body:        []byte(message),
		},
	)
	if err != nil {
		q.DeleteQueue(tmpQueue.Name, false, false, false)
		return nil, err
	}

	select {
	case <-ctx.Done():
		q.DeleteQueue(tmpQueue.Name, false, false, false)
		return nil, ctx.Err()
	case res := <-chanRes:
		q.DeleteQueue(tmpQueue.Name, false, false, false)
		return res.Body, nil
	}
}

// getQueue returns amqp.Queue.
// If it was not defined, defined new queue with default options.
func (q *rabbit) getQueue(name string) (amqp.Queue, error) {
	queue, ok := q.queues[name]
	if ok == true {
		return queue, nil
	}

	// does not exist, create a new one.
	err := q.DeclareQueue(name, false, false, false, false)
	if err != nil {
		return amqp.Queue{}, err
	}
	queue = q.queues[name]

	return queue, nil
}

// closeQueue delete the queue with given args.
// returns deleted messages in the queue.
func (q *rabbit) DeleteQueue(name string, ifUnused, ifEmpty, noWait bool) (int, error) {
	return q.channel.QueueDelete(name, ifUnused, ifEmpty, noWait)
}

// ConsumeMessage consumes message
// If the queueName was not defined, then defines with default values.
func (q *rabbit) ConsumeMessage(queueName, consumerName string, cbMsgConsume CbMsgConsume) error {
	queue, err := q.getQueue(queueName)
	if err != nil {
		return err
	}

	messages, err := q.channel.Consume(
		queue.Name,   // queue
		consumerName, // messageConsumer
		true,         // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		log.Errorf("Could not consume the message. err: %v", err)
		return err
	}

	// process message
	go func() {
		for message := range messages {
			message.Ack(false)

			// message parse
			var req Request
			if err := json.Unmarshal(message.Body, &req); err != nil {
				log.Errorf("Could not parse the message. message: %s, err: %v", string(message.Body), err)
				continue
			}

			// process the message
			res, err := cbMsgConsume(&req)
			if err != nil {
				log.Errorf("Message consumer returns error. err: %v", err)
				continue
			}

			// reply response
			if message.ReplyTo != "" && res != nil {
				resMsg, err := json.Marshal(res)
				if err != nil {
					log.Errorf("Could not marshal the response. res: %v, err: %v", res, err)
					continue
				}

				if err := q.channel.Publish(
					"",              // exchange
					message.ReplyTo, // routing key
					false,           // mandatory
					false,           // immediate
					amqp.Publishing{
						ContentType:   "text/plain",
						CorrelationId: message.CorrelationId,
						Body:          resMsg,
					}); err != nil {
					log.Errorf("Could not reply the message. message: %v, err: %v", res, err)
					continue
				}
			}
		}
	}()

	return nil
}

// ConsumeRPC consumes RPC message
func (q *rabbit) ConsumeRPC(queueName, consumerName string, cbRPC CbMsgRPC) error {
	queue, err := q.getQueue(queueName)
	if err != nil {
		return err
	}

	deliveries, err := q.channel.Consume(
		queue.Name,   // queue
		consumerName, // messageConsumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		log.Errorf("Could not consume the message. err: %v", err)
		return err
	}

	// process message
	go func() {
		for d := range deliveries {

			// execute callback
			res, err := cbRPC(string(d.Body))
			if err != nil {
				log.Errorf("Message consumer returns error. err: %v", err)
				continue
			}

			// reply response
			err = q.channel.Publish(
				"",        // exchange
				d.ReplyTo, // routing key
				false,     // mandatory
				false,     // immediate
				amqp.Publishing{
					ContentType:   "text/plain",
					CorrelationId: d.CorrelationId,
					Body:          []byte(res),
				})
			d.Ack(false)
		}
	}()

	return nil
}

// Close close the Queue.
func (q *rabbit) Close() {
	log.WithFields(log.Fields{
		"url": q.url,
	}).Info("Close the rabbitmq connection.")
	q.closed = true
	q.channel.Close()
	q.connection.Close()
}

// receonnector reconnects the rabbitmq
func (q *rabbit) reconnector() {
	for {
		err := <-q.errorChannel
		if q.closed == false {
			log.Errorf("Reconnecting after connection closed. err: %v", err)
			q.connect()
		}
	}
}

// connect connects to rabbitmq.
func (q *rabbit) connect() {
	for {
		log := log.WithFields(log.Fields{
			"url": q.url,
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

		return
	}
}

// DeclareQueue declares the rabbitmq queue using name and add it to the queues.
func (q *rabbit) DeclareQueue(name string, durable, autoDelete, exclusive, noWait bool) error {
	// declare
	queue, err := q.declareQueue(name, durable, autoDelete, exclusive, noWait)
	if err != nil {
		return err
	}

	q.queues[name] = queue
	return nil
}

// declareQueue declares the rabbitmq queue using name.
func (q *rabbit) declareQueue(name string, durable, autoDelete, exclusive, noWait bool) (amqp.Queue, error) {
	// declare
	queue, err := q.channel.QueueDeclare(
		name,       // name
		durable,    // durable
		autoDelete, // delete when unused
		exclusive,  // exclusive
		noWait,     // no-wait
		nil,        // arguments
	)
	if err != nil {
		log.Errorf("Could not declare the queue. err: %v", err)
		return amqp.Queue{}, err
	}

	return queue, nil
}

// openChannel opens a rabbitmq channel and sets to channel.
func (q *rabbit) openChannel() {
	channel, err := q.connection.Channel()
	if err != nil {
		log.Errorf("Could not open the channel. err: %v", err)
	}
	q.channel = channel
}
