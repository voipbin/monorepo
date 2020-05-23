package rabbitmq

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// ConsumeMessage consumes message
// If the queueName was not defined, then defines with default values.
func (r *rabbit) ConsumeMessage(queueName, consumerName string, messageConsume CbMsgConsume) error {
	q := r.queueGet(queueName)
	if q == nil {
		return fmt.Errorf("no queue found")
	}

	messages, err := r.channel.Consume(
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
		return err
	}

	// process message
	for message := range messages {
		var event Event
		if err := json.Unmarshal(message.Body, &event); err != nil {
			log.Errorf("Could out unmarshal the message. err: %v", err)
			continue
		}

		err := messageConsume(&event)
		if err != nil {
			log.Errorf("Message consumer returns error. err: %v", err)
		}
	}

	return nil
}

// ConsumeRPC consumes RPC message
func (r *rabbit) ConsumeRPC(queueName, consumerName string, cbConsume CbMsgRPC) error {
	messages, err := r.channel.Consume(
		queueName,    // queue
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
	for message := range messages {

		// message parse
		var req Request
		if err := json.Unmarshal(message.Body, &req); err != nil {
			log.Errorf("Could not parse the message. message: %s, err: %v", string(message.Body), err)
			continue
		}

		// execute callback
		res, err := cbConsume(&req)
		if err != nil {
			log.Errorf("Message consumer returns error. err: %v", err)
			continue
		} else if res == nil {
			// nothing to reply
			continue
		}

		// reply response
		if message.ReplyTo != "" {
			channel, err := r.connection.Channel()
			if err != nil {
				log.Errorf("Could not create a channel. err: %v", err)
				continue
			}
			defer channel.Close()

			resMsg, err := json.Marshal(res)
			if err != nil {
				log.Errorf("Could not marshal the response. res: %v, err: %v", res, err)
				continue
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
				log.Errorf("Could not reply the message. message: %v, err: %v", res, err)
				continue
			}
		}
	}

	return nil
}
