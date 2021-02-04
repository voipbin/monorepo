package rabbitmqhandler

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// ConsumeMessage consumes message
// If the queueName was not defined, then defines with default values.
func (r *rabbit) ConsumeMessage(queueName, consumerName string, messageConsume CbMsgConsume) error {

	return r.ConsumeMessageOpt(queueName, consumerName, false, false, false, messageConsume)
}

// ConsumeMessageOpt consumes message with given options
// If the queueName was not defined, then uses with default queue name values.
func (r *rabbit) ConsumeMessageOpt(queueName, consumerName string, exclusive bool, noLocal bool, noWait bool, messageConsume CbMsgConsume) error {
	queue := r.queueGet(queueName)
	if queue == nil {
		return fmt.Errorf("queue not found")
	}

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
		logrus.Errorf("Could not consume the message. err: %v", err)
		return err
	}

	// process message
	for message := range messages {
		// execute callback
		err := r.executeConsumeMessage(message, messageConsume)
		if err != nil {
			logrus.Errorf("Could not execute the message consume callback. err: %v", err)
		}
		message.Ack(false)
	}

	return nil
}

// executeConsumeMessage runs the callback with the given amqp message
func (r *rabbit) executeConsumeMessage(message amqp.Delivery, messageConsume CbMsgConsume) error {
	var event Event

	if err := json.Unmarshal(message.Body, &event); err != nil {
		return fmt.Errorf("Could out unmarshal the message. err: %v", err)
	}

	if err := messageConsume(&event); err != nil {
		return fmt.Errorf("Message consumer returns error. err: %v", err)
	}

	return nil
}

// ConsumeRPC consumes RPC message
func (r *rabbit) ConsumeRPC(queueName, consumerName string, cbConsume CbMsgRPC) error {

	return r.ConsumeRPCOpt(queueName, consumerName, false, false, false, cbConsume)
}

// ConsumeRPCOpt consumes RPC message with given options
func (r *rabbit) ConsumeRPCOpt(queueName, consumerName string, exclusive bool, noLocal bool, noWait bool, cbConsume CbMsgRPC) error {
	queue := r.queueGet(queueName)
	if queue == nil {
		return fmt.Errorf("queue not found")
	}

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
		logrus.Errorf("Could not consume the message. err: %v", err)
		return err
	}

	// process message
	for message := range messages {

		if err := r.executeConsumeRPC(message, cbConsume); err != nil {
			logrus.Errorf("Could not consume the RPC correctly. err: %v", err)
		}
		message.Ack(false)
	}

	return nil
}

// executeConsumeRPC runs the callback with the given amqp message
func (r *rabbit) executeConsumeRPC(message amqp.Delivery, cbConsume CbMsgRPC) error {

	// message parse
	var req Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		return fmt.Errorf("Could not parse the message. message: %s, err: %v", string(message.Body), err)
	}

	// execute callback
	res, err := cbConsume(&req)
	if err != nil {
		return fmt.Errorf("Message consumer returns error. err: %v", err)
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
		return fmt.Errorf("Could not create a channel. err: %v", err)
	}
	defer channel.Close()

	resMsg, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("Could not marshal the response. res: %v, err: %v", res, err)
	}

	if err := channel.Publish(
		"",              // exchange
		message.ReplyTo, // routing key
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: message.CorrelationId,
			Body:          resMsg,
		}); err != nil {
		return fmt.Errorf("Could not reply the message. message: %v, err: %v", res, err)
	}

	return nil
}
