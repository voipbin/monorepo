package rabbitmqhandler

import (
	"fmt"

	commonoutline "monorepo/bin-common-handler/models/outline"

	amqp "github.com/rabbitmq/amqp091-go"
)

// queueGet returns amqp.Queue.
// If it was not defined, defined new queue with default options.
func (r *rabbit) queueGet(name string) *queue {
	r.mu.RLock()
	q := r.queues[name]
	r.mu.RUnlock()
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
		return h.queueCreateVolatile(name)

	case "normal":
		return h.queueCreateNormal(name)

	default:
		return fmt.Errorf("invalid queue type. type: %s", queueType)
	}
}

func (h *rabbit) queueCreateNormal(name string) error {

	// declare the queue
	if errDeclare := h.QueueDeclare(name, true, false, false, false, nil); errDeclare != nil {
		return fmt.Errorf("could not declare the queue for normal. err: %v", errDeclare)
	}

	if errConfig := h.queueConfig(name); errConfig != nil {
		return fmt.Errorf("could not config the queue. err: %v", errConfig)
	}

	return nil
}

func (h *rabbit) queueCreateVolatile(name string) error {

	// declare the queue with x-expires for automatic cleanup of stale queues.
	// x-expires deletes the queue after 30 minutes with no consumers.
	if errDeclare := h.QueueDeclare(name, false, true, false, false, amqp.Table{
		"x-expires": int32(1800000),
	}); errDeclare != nil {
		return fmt.Errorf("could not declare the queue for volatile. err: %v", errDeclare)
	}

	if errConfig := h.queueConfig(name); errConfig != nil {
		return fmt.Errorf("could not config the queue. err: %v", errConfig)
	}

	return nil
}

func (h *rabbit) queueConfig(name string) error {
	// set qos
	if errQos := h.QueueQoS(name, 1, 0); errQos != nil {
		return fmt.Errorf("could not set the queue's qos. err: %v", errQos)
	}

	// declare the exchange for deplayed message
	if errDeclare := h.ExchangeDeclareForDelay(string(commonoutline.QueueNameDelay), true, false, false, false); errDeclare != nil {
		return fmt.Errorf("could not declare the exchange for dealyed message. err: %v", errDeclare)
	}

	// bind the delay exchange to the queue
	if errSubscribe := h.QueueBind(name, name, string(commonoutline.QueueNameDelay), false, nil); errSubscribe != nil {
		return fmt.Errorf("could not bind the queue and exchange. err: %v", errSubscribe)
	}

	return nil
}

// QueueDeclare declares the rabbitmq queue using name and add it to the queues.
func (r *rabbit) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) error {
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
		args,       // arguments
	)
	if err != nil {
		_ = channel.Close() // close channel on error to prevent leak
		return err
	}

	r.mu.Lock()
	// close existing channel if re-declaring (e.g., during reconnection)
	if existing := r.queues[name]; existing != nil && existing.channel != nil {
		_ = existing.channel.Close()
	}

	r.queues[name] = &queue{
		name:       name,
		durable:    durable,
		autoDelete: autoDelete,
		exclusive:  exclusive,
		noWait:     noWait,
		args:       args,

		channel: channel,
		queue:   &q,
	}
	r.mu.Unlock()

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
func (h *rabbit) QueueSubscribe(name string, topic string) error {
	return h.QueueBind(name, "", topic, false, nil)
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

	r.mu.Lock()
	r.queueBinds[name] = &queueBind{
		name:     name,
		key:      key,
		exchange: exchange,
		noWait:   noWait,
		args:     args,
	}
	r.mu.Unlock()
	return nil
}
