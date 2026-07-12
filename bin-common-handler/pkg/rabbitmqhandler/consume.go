package rabbitmqhandler

import (
	"context"
	"encoding/json"
	"fmt"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// headerRetryCount is the amqp delivery header key used to track how many
// times an event message has already been retried. Absent/malformed header
// is treated as retry 0 (first-time failure).
const headerRetryCount = "x-retry-count"

// maxEventRetries is the fixed number of retries applied to a failed event
// message before it is permanently dropped. Total delivery attempts are
// therefore at most maxEventRetries+1 (the original attempt plus retries).
const maxEventRetries = 3

// retryBackoff holds the delay (in ms) applied before the Nth retry.
// retryBackoff[i] is the delay before retry attempt i+1 (i.e. index 0 is the
// delay before the 1st retry, retryCount at that point is 0).
var retryBackoff = []int{5000, 30000, 120000} // 5s, 30s, 120s

// startConsumers starts consuming from the queue and spawns worker goroutines.
// Used by both initial ConsumeMessage/ConsumeRPC and by reconsumerAll during reconnection.
func (r *rabbit) startConsumers(reg *consumerRegistration) error {
	queue := r.queueGet(reg.queueName)
	if queue == nil {
		return fmt.Errorf("queue '%s' not found", reg.queueName)
	}

	// Prefetch must cover the full worker pool, or ack-after-process would
	// serialize workers behind a single in-flight message. Re-applied on
	// every call (both initial registration and reconsumerAll during
	// reconnection) so it survives reconnection -- queueConfig's QueueQoS
	// call at queue-creation time alone does not, since redeclareAll opens a
	// fresh channel with no Qos re-application.
	if err := queue.channel.Qos(reg.numWorkers, 0, false); err != nil {
		return fmt.Errorf("could not set qos for queue '%s': %v", reg.queueName, err)
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
			go r.consumeMessageWorker(messages, reg.queueName, reg.cbMessage)
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

	if err := r.registerConsumer(reg); err != nil {
		return err
	}

	if err := r.startConsumers(reg); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

// registerConsumer adds a consumer registration to r.consumers, enforcing a
// one-queue-one-registration invariant: registering a second consumer for a
// queueName that already has one is rejected. Qos (prefetch) is a per-channel
// setting, not per-consumer, so two registrations sharing a queue would
// silently overwrite each other's intended prefetch value via startConsumers.
// The existence check and the append happen under the same lock acquisition
// to avoid a TOCTOU race between concurrent ConsumeMessage/ConsumeRPC calls
// for the same queue name.
func (r *rabbit) registerConsumer(reg *consumerRegistration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.consumers {
		if existing.queueName == reg.queueName {
			return fmt.Errorf("queue '%s' already has a registered consumer", reg.queueName)
		}
	}

	r.consumers = append(r.consumers, reg)
	return nil
}

func (r *rabbit) consumeMessageWorker(messages <-chan amqp.Delivery, queueName string, messageConsume sock.CbMsgConsume) {
	for message := range messages {
		err := r.executeConsumeMessage(message, messageConsume)
		r.ackOrRetry(message, queueName, err)
	}
}

// ackOrRetry acknowledges a successfully processed event message, or applies
// a bounded-retry policy (see maxEventRetries/retryBackoff) for a failed one.
// See docs/plans/2026-07-12-voip-1233-rabbitmq-ack-after-process-design.md §4.2
// for the full design rationale.
func (r *rabbit) ackOrRetry(message amqp.Delivery, queueName string, processErr error) {
	log := logrus.WithFields(logrus.Fields{"func": "ackOrRetry", "queue": queueName})

	if processErr == nil {
		if err := message.Ack(false); err != nil {
			log.Errorf("Error acknowledging message: %v", err)
		}
		return
	}

	retryCount := getRetryCount(message.Headers) // 0 if header absent/malformed

	if retryCount >= maxEventRetries {
		log.Errorf("Message processing failed after %d retries, dropping. err: %v", maxEventRetries, processErr)
		promEventDropped.WithLabelValues(queueName).Inc()
		if err := message.Ack(false); err != nil {
			log.Errorf("Error acknowledging (dropping) exhausted message: %v", err)
		}
		return
	}

	// Publish a delayed copy BEFORE acking the original: if the republish
	// fails (e.g. broker hiccup), fall back to an immediate Nack(requeue=true)
	// on the original rather than lose it. Worst case is an immediate untimed
	// retry instead of a backed-off one -- never silent loss.
	headers := cloneHeaders(message.Headers)
	headers[headerRetryCount] = retryCount + 1
	delayMs := retryBackoff[retryCount]

	log.Warnf("Message processing failed, scheduling retry %d/%d in %dms. err: %v", retryCount+1, maxEventRetries, delayMs, processErr)
	promEventRetried.WithLabelValues(queueName, strconv.Itoa(retryCount+1)).Inc()

	headers["x-delay"] = delayMs
	// DeliveryMode=Persistent: the retry copy can sit in the delay-exchange's
	// internal wait state for up to 120s (the longest backoff step). Without
	// explicit persistence it would be transient and could be silently lost
	// on a broker restart during that window, defeating the point of this fix.
	if errPub := r.publishExchange(string(commonoutline.QueueNameDelay), queueName, message.Body, headers, amqp.Persistent); errPub != nil {
		log.Errorf("Could not schedule delayed retry, falling back to immediate requeue. err: %v", errPub)
		if err := message.Nack(false, true); err != nil {
			log.Errorf("Error nacking (fallback requeue) message: %v", err)
		}
		return
	}

	if err := message.Ack(false); err != nil {
		// Original could not be acked after a successful republish -- broker
		// may redeliver it too, resulting in a duplicate retry copy in
		// flight. Accepted: duplicate processing, never loss.
		log.Errorf("Error acknowledging original after scheduling retry (possible duplicate): %v", err)
	}
}

// getRetryCount reads x-retry-count from delivery headers, defaulting to 0
// for a missing or malformed header (treats it as a first-time failure).
func getRetryCount(headers amqp.Table) int {
	v, ok := headers[headerRetryCount]
	if !ok {
		return 0
	}
	n, ok := v.(int32) // amqp091-go decodes small ints as int32 on the wire
	if !ok {
		return 0
	}
	return int(n)
}

// cloneHeaders returns a shallow copy of h so the original delivery's
// headers are never mutated in place.
func cloneHeaders(h amqp.Table) amqp.Table {
	out := make(amqp.Table, len(h)+1)
	for k, v := range h {
		out[k] = v
	}
	return out
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

	if err := r.registerConsumer(reg); err != nil {
		return err
	}

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

		// note: The RPC path intentionally keeps ack-before-process. Unlike
		// the event path, RPC failures are already observable to the caller
		// (a 500 response when ReplyTo != "", or a timeout otherwise), so
		// this is out of scope for VOIP-1233. See design doc §2.
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
		// Send a 500 error response so the caller does not time out waiting
		// for a reply that will never arrive. Without this, transport-level
		// timeouts accumulate and can trip the circuit breaker.
		if message.ReplyTo != "" {
			r.publishRPCErrorResponse(message)
		}
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

// publishRPCErrorResponse sends a 500 error response back to the RPC caller.
// This prevents the caller from timing out when the handler returns an error
// without a response, which would otherwise be misinterpreted as a transport
// failure and trip the circuit breaker.
func (r *rabbit) publishRPCErrorResponse(message amqp.Delivery) {
	log := logrus.WithFields(logrus.Fields{
		"func": "publishRPCErrorResponse",
	})

	errRes := &sock.Response{StatusCode: 500}
	resMsg, err := json.Marshal(errRes)
	if err != nil {
		log.Errorf("Could not marshal error response. err: %v", err)
		return
	}

	channel, err := r.connection.Channel()
	if err != nil {
		log.Errorf("Could not create channel for error response. err: %v", err)
		return
	}
	defer func() {
		_ = channel.Close()
	}()

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
		log.Errorf("Could not publish error response. err: %v", err)
	}
}
