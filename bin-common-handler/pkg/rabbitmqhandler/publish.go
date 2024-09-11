package rabbitmqhandler

import (
	"context"
	"encoding/json"
	"fmt"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	amqp "github.com/rabbitmq/amqp091-go"
)

// PublishMessage sends a message to rabbitmq
func (r *rabbit) publishExchange(exchange, key string, message []byte, headers amqp.Table) error {

	channel, err := r.connection.Channel()
	if err != nil {
		return fmt.Errorf("could not create a channel for PublishMessage. err: %v", err)
	}
	defer channel.Close()

	err = channel.PublishWithContext(
		context.Background(),
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
		return fmt.Errorf("could not send a message. err: %v", err)
	}

	return nil
}

// RequestPublish publishes request message and returns response.
func (r *rabbit) RequestPublish(ctx context.Context, queueName string, req *sock.Request) (*sock.Response, error) {

	reqMsg, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("could not marshal the message. err: %v", err)
	}

	// create a channel
	channel, err := r.connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("could not create a channel for PublishRPC. err: %v", err)
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
		return nil, fmt.Errorf("could not declare the queue. err: %v", err)
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
	if err != nil {
		return nil, fmt.Errorf("could not consume the message. err: %v", err)
	}

	// publish the message
	err = channel.PublishWithContext(
		context.Background(),
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
		return nil, fmt.Errorf("could not send a message. err: %v", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-chanRes:
		var response sock.Response
		if err := json.Unmarshal(res.Body, &response); err != nil {
			return nil, err
		}
		return &response, nil
	}
}

// EventPublish sends a message to rabbitmq
func (r *rabbit) EventPublish(exchange string, key string, evt *sock.Event) error {
	message, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	return r.publishExchange(exchange, key, message, nil)
}

// RequestPublishWithDelay sends a delayed request to the rabbitmq exchange
// delay is ms.
func (r *rabbit) RequestPublishWithDelay(key string, req *sock.Request, delay int) error {
	headers := make(amqp.Table)
	headers["x-delay"] = delay

	message, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return r.publishExchange(string(commonoutline.QueueNameDelay), key, message, headers)
}

// EventPublishWithDelay sends a delayed event to the rabbitmq exchange
// delay is ms.
func (r *rabbit) EventPublishWithDelay(exchange, key string, evt *sock.Event, delay int) error {
	headers := make(amqp.Table)
	headers["x-delay"] = delay

	message, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	return r.publishExchange(exchange, key, message, headers)
}
