package rabbitmq

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/rabbitmq/models"
)

// PublishMessage sends a message to rabbitmq
func (r *rabbit) publishExchange(exchange, key string, message []byte, headers amqp.Table) error {
	channel, err := r.connection.Channel()
	if err != nil {
		log.Errorf("Could not create a channel for PublishMessage. err: %v", err)
		return err
	}
	defer channel.Close()

	err = channel.Publish(
		exchange, // exchange
		key,      // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
			Headers:     headers,
		})
	if err != nil {
		log.Errorf("Could not send a message. err: %v", err)
		return err
	}

	return nil
}

// PublishMessage sends a request to rabbitmq
func (r *rabbit) PublishRequest(queueName string, req *models.Request) error {
	message, err := json.Marshal(req)
	if err != nil {
		return err
	}

	if err := r.publishExchange("", queueName, message, nil); err != nil {
		log.Errorf("Could not send a message. err: %v", err)
		return err
	}

	return nil
}

// PublishEvent sends a event to rabbitmq
func (r *rabbit) PublishEvent(queueName string, evt *models.Event) error {
	message, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	if err := r.publishExchange("", queueName, message, nil); err != nil {
		log.Errorf("Could not send a message. err: %v", err)
		return err
	}

	return nil
}

// PublishRPC publishes RPC message and returns response.
func (r *rabbit) PublishRPC(ctx context.Context, queueName string, req *models.Request) (*models.Response, error) {
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
		var response models.Response
		if err := json.Unmarshal(res.Body, &response); err != nil {
			return nil, err
		}
		return &response, nil
	}
}

// PublishMessage sends a message to rabbitmq
func (r *rabbit) PublishExchangeRequest(exchange, key string, req *models.Request) error {
	message, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return r.publishExchange(exchange, key, message, nil)
}

// PublishExchangeDelayedRequest sends a delayed request to rabbitmq
// delay is ms.
func (r *rabbit) PublishExchangeDelayedRequest(exchange, key string, req *models.Request, delay int) error {
	headers := make(amqp.Table)
	headers["x-delay"] = delay

	message, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return r.publishExchange(exchange, key, message, headers)
}
