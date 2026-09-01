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
//
// Currently unreachable from outside this package (absent from both the Rabbit and SockHandler
// interfaces) -- not fixed as part of VOIP-1431 for that reason, but it uses queue.channel the
// same way QueueBind/QueueUnbind used to, and shares the same hazard class: if this is ever
// wired up to a caller that can run at an arbitrary runtime moment (like scopeRefCount), it
// should get the same dedicated-channel treatment QueueBind/QueueUnbind now have.
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
	// note: prefetch (QoS) is intentionally NOT set here. It used to be fixed
	// at 1 via QueueQoS(name, 1, 0), but that value is superseded by
	// startConsumers's per-registration Qos(reg.numWorkers, ...) call, which
	// also re-applies on every reconnection (see consume.go). Setting a stale
	// prefetch=1 here would be misleading dead configuration.

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

// QueueQoS sets the queue's channel prefetch settings.
//
// Currently unreachable from outside this package (absent from both the Rabbit and SockHandler
// interfaces) -- not fixed as part of VOIP-1431 for that reason, but it uses queue.channel the
// same way QueueBind/QueueUnbind used to, and shares the same hazard class: if this is ever
// wired up to a caller that can run at an arbitrary runtime moment (like scopeRefCount), it
// should get the same dedicated-channel treatment QueueBind/QueueUnbind now have.
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

// QueueBind binds queue and exchange with a key. Appends to the tracked bind set for this
// queue name (does not overwrite), so redeclareAll() can restore ALL active bindings after a
// broker reconnect, not just the most recent one (VOIP-1258 round-1 implementation-plan review
// finding F2).
//
// Issues the bind on a dedicated, short-lived channel (VOIP-1431) rather than the queue's own
// long-lived channel (queue.channel) -- queue.channel is shared with startConsumers's
// Qos/Consume, and amqp091-go's Channel.QueueBind/Consume/Qos all route through an internal
// call() helper with no lock and a single un-correlated reply channel (ch.rpc); calling any two
// of them concurrently on the same *amqp.Channel can cross-deliver replies. See
// docs/plans/2026-09-01-voip-1431-scoperefcount-amqp-safety-analysis.md and
// docs/plans/2026-09-01-voip-1431-amqp-channel-race-design.md for the full analysis.
// queue.bind is scoped to the queue, not the issuing channel, so this is semantically identical
// to binding on queue.channel -- except immediately after a reconnect and before redeclareAll
// has re-declared this queue, where the old code failed fast (ErrClosed on the dead
// queue.channel) and this instead issues queue.bind against a queue that may not be re-declared
// yet on the live connection: benign either way. A durable queue survives the reconnect on the
// broker side, so the bind succeeds and is then harmlessly re-issued (idempotently, via the
// tracked-bind-set dedup below) when redeclareAll's own replay runs. A volatile queue
// (x-expires) may have expired, in which case the broker closes this ephemeral channel with a
// 404 -- harmless precisely because the channel is ephemeral and nothing else is using it; the
// caller sees the bind fail, same outcome as today's ErrClosed case, just a different error.
func (r *rabbit) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	if r.queueGet(name) == nil {
		return fmt.Errorf("no queue found")
	}

	conn := r.connectionGet()
	if conn == nil {
		return amqp.ErrClosed
	}
	channel, err := conn.Channel()
	if err != nil {
		return err
	}
	defer func() { _ = channel.Close() }()

	if err := channel.QueueBind(name, key, exchange, noWait, args); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.queueBinds[name] {
		if b.key == key && b.exchange == exchange {
			return nil // already tracked, idempotent re-bind
		}
	}
	r.queueBinds[name] = append(r.queueBinds[name], &queueBind{
		name:     name,
		key:      key,
		exchange: exchange,
		noWait:   noWait,
		args:     args,
	})
	return nil
}

// QueueUnbind unbinds queue and exchange with a key, removing the matching entry from the
// tracked bind set (not the whole map key, unless the set becomes empty). Issues the unbind on
// a dedicated, short-lived channel for the same reason QueueBind does above -- see that doc
// comment.
func (r *rabbit) QueueUnbind(name, key, exchange string, args amqp.Table) error {
	if r.queueGet(name) == nil {
		return fmt.Errorf("no queue found")
	}

	conn := r.connectionGet()
	if conn == nil {
		return amqp.ErrClosed
	}
	channel, err := conn.Channel()
	if err != nil {
		return err
	}
	defer func() { _ = channel.Close() }()

	if err := channel.QueueUnbind(name, key, exchange, args); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	binds := r.queueBinds[name]
	for i, b := range binds {
		if b.key == key && b.exchange == exchange {
			r.queueBinds[name] = append(binds[:i], binds[i+1:]...)
			break
		}
	}
	if len(r.queueBinds[name]) == 0 {
		delete(r.queueBinds, name)
	}
	return nil
}
