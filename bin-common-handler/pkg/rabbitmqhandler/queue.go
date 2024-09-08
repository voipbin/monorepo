package rabbitmqhandler

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// queueGet returns amqp.Queue.
// If it was not defined, defined new queue with default options.
func (r *rabbit) queueGet(name string) *queue {
	q := r.queues[name]
	return q
}

// QueueDelete deletes the queue with given args.
// returns deleted messages in the queue.
func (r *rabbit) QueueDelete(name string, ifUnused, ifEmpty, noWait bool) (int, error) {
	queue := r.queueGet(name)
	if queue == nil {
		return 0, nil
	}

	_, err := queue.channel.QueueDelete(name, ifUnused, ifEmpty, noWait)
	if err != nil {
		return 0, err
	}

	return 0, nil
}

func (h *rabbit) QueueCreate(name string, queueType string) error {
	switch queueType {
	case "volatile":
	}
}

// QueueDeclare declares the rabbitmq queue using name and add it to the queues.
func (r *rabbit) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool) error {
	channel, err := r.connection.Channel()
	if err != nil {
		return err
	}

	// declare the queue
	q, err := channel.QueueDeclare(
		name,       // name
		durable,    // durable
		autoDelete, // delete when unused
		exclusive,  // exclusive
		noWait,     // no-wait
		nil,        // arguments
	)
	if err != nil {
		return err
	}

	r.queues[name] = &queue{
		name:       name,
		durable:    durable,
		autoDelete: autoDelete,
		exclusive:  exclusive,
		noWait:     noWait,

		channel: channel,
		qeueue:  &q,
	}

	return nil
}

func (r *rabbit) QueueQoS(name string, prefetchCount, prefetchSize int) error {
	q := r.queueGet(name)
	if q == nil {
		return fmt.Errorf("no queue found")
	}

	if err := q.channel.Qos(prefetchCount, prefetchSize, false); err != nil {
		return fmt.Errorf("could not set channel qos. queue: %s, cnt: %d, size: %d, err: %v", name, prefetchCount, prefetchSize, err)
	}

	return nil
}

// QueueBind binds queue and exchange with a key
func (r *rabbit) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	queue := r.queueGet(name)
	if queue == nil {
		return fmt.Errorf("no queue found")
	}

	if err := queue.channel.QueueBind(name, key, exchange, noWait, args); err != nil {
		return err
	}

	r.queueBinds[name] = &queueBind{
		name:     name,
		key:      key,
		exchange: exchange,
		noWait:   noWait,
		args:     args,
	}
	return nil
}
