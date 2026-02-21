package rabbitmqhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/models/sock"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// startConsumers starts consuming from the queue and spawns worker goroutines.
// Used by both initial ConsumeMessage/ConsumeRPC and by reconsumerAll during reconnection.
func (r *rabbit) startConsumers(reg *consumerRegistration) error {
	queue := r.queueGet(reg.queueName)
	if queue == nil {
		return fmt.Errorf("queue '%s' not found", reg.queueName)
	}

	messages, err := queue.channel.Consume(
		reg.queueName,
		reg.consumerName,
		false,
		reg.exclusive,
		reg.noLocal,
		reg.noWait,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not start consuming from queue '%s': %v", reg.queueName, err)
	}

	for i := 0; i < reg.numWorkers; i++ {
		switch reg.cType {
		case consumerTypeMessage:
			go r.consumeMessageWorker(messages, reg.cbMessage)
		case consumerTypeRPC:
			go r.consumeRPCWorker(messages, reg.cbRPC)
		}
	}

	return nil
}

// ConsumeMessage consumes messages from the given queue with provided options.
// If the queueName is not provided, it defaults to a pre-configured queue.
// It uses goroutines with a worker pool to process messages concurrently.
func (r *rabbit) ConsumeMessage(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, messageConsume sock.CbMsgConsume) error {
	reg := &consumerRegistration{
		queueName:    queueName,
		consumerName: consumerName,
		exclusive:    exclusive,
		noLocal:      noLocal,
		noWait:       noWait,
		numWorkers:   numWorkers,
		cType:        consumerTypeMessage,
		cbMessage:    messageConsume,
	}

	r.mu.Lock()
	r.consumers = append(r.consumers, reg)
	r.mu.Unlock()

	if err := r.startConsumers(reg); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func (r *rabbit) consumeMessageWorker(messages <-chan amqp.Delivery, messageConsume sock.CbMsgConsume) {
	log := logrus.WithFields(logrus.Fields{
		"func": "consumeMessageWorker",
	})

	for message := range messages {
		// note: Acknowledgement should be done before processing the message.
		// otherwise, it will block the channel and the message will not be consumed.
		if err := message.Ack(false); err != nil {
			log.Errorf("Error acknowledging message: %v", err)
		}

		if err := r.executeConsumeMessage(message, messageConsume); err != nil {
			log.Errorf("Error while processing message: %v", err)
		}
	}
}

// executeConsumeMessage runs the callback with the given amqp message
func (r *rabbit) executeConsumeMessage(message amqp.Delivery, messageConsume sock.CbMsgConsume) error {
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
func (r *rabbit) ConsumeRPC(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, cbConsume sock.CbMsgRPC) error {
	reg := &consumerRegistration{
		queueName:    queueName,
		consumerName: consumerName,
		exclusive:    exclusive,
		noLocal:      noLocal,
		noWait:       noWait,
		numWorkers:   numWorkers,
		cType:        consumerTypeRPC,
		cbRPC:        cbConsume,
	}

	r.mu.Lock()
	r.consumers = append(r.consumers, reg)
	r.mu.Unlock()

	if err := r.startConsumers(reg); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func (r *rabbit) consumeRPCWorker(messages <-chan amqp.Delivery, cbConsume sock.CbMsgRPC) {
	log := logrus.WithFields(logrus.Fields{
		"func": "consumeMessageWorker",
	})

	for message := range messages {

		// note: Acknowledgement should be done before processing the message.
		// otherwise, it will block the channel and the message will not be consumed.
		if err := message.Ack(false); err != nil {
			log.Errorf("Could not ack the message. err: %v", err)
		}

		if err := r.executeConsumeRPC(message, cbConsume); err != nil {
			log.Errorf("Could not execute the consumer. err: %v", err)
		}
	}
}

// executeConsumeRPC runs the callback with the given amqp message
func (r *rabbit) executeConsumeRPC(message amqp.Delivery, cbConsume sock.CbMsgRPC) error {

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
	defer func() {
		_ = channel.Close()
	}()

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
