package rabbitmqhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/models/sock"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// ConsumeMessage consumes message with given options
// If the queueName was not defined, then uses with default queue name values.
func (r *rabbit) ConsumeMessage(queueName, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, messageConsume CbMsgConsume) error {
	log := logrus.WithField("func", "ConsumeMessageOpt")

	queue := r.queueGet(queueName)
	if queue == nil {
		return fmt.Errorf("queue not found")
	}

	workers := make(chan int, numWorkers)
	for {
		// fetch the messages
		messages, err := queue.channel.Consume(
			queueName,    // queue
			consumerName, // messageConsumer
			false,        // auto-ack
			exclusive,    // exclusive
			noLocal,      // no-local
			noWait,       // no-wait
			nil,          // args
		)
		if err != nil {
			return fmt.Errorf("could not consume the message. err: %v", err)
		}

		// process message
		for message := range messages {

			workers <- 1 // will block if there is MAX ints in workers
			go func(m amqp.Delivery) {
				// execute callback
				if errConsume := r.executeConsumeMessage(m, messageConsume); errConsume != nil {
					logrus.Errorf("Could not execute the message consume callback. err: %v", errConsume)
				}

				// ack
				if errAck := m.Ack(false); errAck != nil {
					log.Errorf("Could not ack the message. err: %v", errAck)
				}
				<-workers // removes an int from workers, allowing another to proceed
			}(message)
		}
	}
}

// executeConsumeMessage runs the callback with the given amqp message
func (r *rabbit) executeConsumeMessage(message amqp.Delivery, messageConsume CbMsgConsume) error {
	var event sock.Event

	if err := json.Unmarshal(message.Body, &event); err != nil {
		return fmt.Errorf("could out unmarshal the message. err: %v", err)
	}

	if err := messageConsume(&event); err != nil {
		return fmt.Errorf("message consumer returns error. err: %v", err)
	}

	return nil
}

// ConsumeRPC consumes RPC message with given options
func (r *rabbit) ConsumeRPC(queueName, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, cbConsume CbMsgRPC) error {
	log := logrus.WithField("func", "ConsumeRPCOpt")

	queue := r.queueGet(queueName)
	if queue == nil {
		return fmt.Errorf("queue not found")
	}

	workers := make(chan int, numWorkers)
	for {
		// fetch teh messages
		messages, err := queue.channel.Consume(
			queueName,    // queue
			consumerName, // messageConsumer
			false,        // auto-ack
			exclusive,    // exclusive
			noLocal,      // no-local
			noWait,       // no-wait
			nil,          // args
		)
		if err != nil {
			return fmt.Errorf("could not consume the RPC message. err: %v", err)
		}

		// process message
		for message := range messages {

			workers <- 1 // will block if there is MAX ints in workers
			go func(m amqp.Delivery) {
				if errConsume := r.executeConsumeRPC(m, cbConsume); errConsume != nil {
					log.Errorf("Could not consume the RPC message correctly. err: %v", errConsume)
				}

				// ack
				if err := m.Ack(false); err != nil {
					log.Errorf("Could not ack the message. err: %v", err)
				}
				<-workers // removes an int from workers, allowing another to proceed
			}(message)
		}
	}
}

// executeConsumeRPC runs the callback with the given amqp message
func (r *rabbit) executeConsumeRPC(message amqp.Delivery, cbConsume CbMsgRPC) error {

	// message parse
	var req sock.Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		return fmt.Errorf("could not parse the message. message: %s, err: %v", string(message.Body), err)
	}

	// execute callback
	res, err := cbConsume(&req)
	if err != nil {
		return fmt.Errorf("message consumer returns error. err: %v", err)
	} else if res == nil {
		// nothing to return
		return nil
	}

	// check reply destination
	if message.ReplyTo == "" {
		// no place to reply send
		return nil
	}

	channel, err := r.connection.Channel()
	if err != nil {
		return fmt.Errorf("could not create a channel. err: %v", err)
	}
	defer channel.Close()

	resMsg, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("could not marshal the response. res: %v, err: %v", res, err)
	}

	if err := channel.PublishWithContext(
		context.Background(),
		"",              // exchange
		message.ReplyTo, // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: message.CorrelationId,
			Body:          resMsg,
		}); err != nil {
		return fmt.Errorf("could not reply the message. message: %v, err: %v", res, err)
	}

	return nil
}
