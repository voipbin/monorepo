package rabbitmq

import (
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// queueGet returns amqp.Queue.
// If it was not defined, defined new queue with default options.
func (r *rabbit) queueGet(name string) *queue {
	q := r.queues[name]
	return q

	// if q != nil {
	// 	return q.queue, nil
	// }

	// return r.queueDeclare(q.channel, q.name, false, false, false, false)
}

// closeQueue delete the queue with given args.
// returns deleted messages in the queue.
func (r *rabbit) DeleteQueue(name string, ifUnused, ifEmpty, noWait bool) (int, error) {
	queue := r.queueGet(name)
	if queue == nil {
		return 0, nil
	}

	return queue.channel.QueueDelete(name, ifUnused, ifEmpty, noWait)
}

// QueueDeclare declares the rabbitmq queue using name and add it to the queues.
func (r *rabbit) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool) error {
	// declare the channel
	channel, err := r.connection.Channel()
	if err != nil {
		return err
	}

	// declare the queue
	q, err := r.channelQueueDeclare(channel, name, durable, autoDelete, exclusive, noWait)
	if err != nil {
		return err
	}

	tmpQueue := &queue{
		name:         name,
		durable:      durable,
		autoDelete:   autoDelete,
		exclusive:    exclusive,
		noWait:       noWait,
		channel:      channel,
		channelQueue: q,
	}

	r.queues[name] = tmpQueue
	return nil
}

// queueDeclare declares the rabbitmq queue using name.
func (r *rabbit) channelQueueDeclare(channel *amqp.Channel, name string, durable, autoDelete, exclusive, noWait bool) (*amqp.Queue, error) {
	queue, err := channel.QueueDeclare(
		name,       // name
		durable,    // durable
		autoDelete, // delete when unused
		exclusive,  // exclusive
		noWait,     // no-wait
		nil,        // arguments
	)
	if err != nil {
		log.Errorf("Could not declare the queue. err: %v", err)
		return nil, err
	}

	return &queue, nil
}

func (r *rabbit) queuesRedeclare() {
	for _, queue := range r.queues {
		log.Debugf("Redeclaring the queue. queue: %s", queue.name)
		r.QueueDeclare(queue.name, queue.durable, queue.autoDelete, queue.exclusive, queue.noWait)
	}
}
