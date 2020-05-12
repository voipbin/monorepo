package rabbitmq

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// PublishMessage sends a message to rabbitmq
func (r *rabbit) PublishMessage(queueName, message string) error {
	channel, err := r.connection.Channel()
	if err != nil {
		log.Errorf("Could not create a channel for PublishMessage. err: %v", err)
		return err
	}

	err = channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
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
func (r *rabbit) PublishRPC(ctx context.Context, queueName string, req *Request) (*Response, error) {
	log.WithFields(log.Fields{
		"name":    queueName,
		"request": req,
	}).Info("Publish message to RPC.")

	reqMsg, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// create a channel
	channel, err := r.connection.Channel()
	if err != nil {
		log.Errorf("Could not create a channel for PublishRPC. err: %v", err)
		return nil, err
	}
	defer channel.Close()

	// declare the response queue
	resQueue, err := channel.QueueDeclare(
		"",    // name
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, err
	}

	// consuming the message from the tmpQueue
	chanRes, err := channel.Consume(
		resQueue.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)

	// publish the message
	err = channel.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			ReplyTo:     resQueue.Name,
			Body:        reqMsg,
		},
	)
	if err != nil {
		log.Errorf("Could not send a request. err: %v", err)
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-chanRes:
		var response Response
		if err := json.Unmarshal(res.Body, &response); err != nil {
			return nil, err
		}
		return &response, nil
	}
}
