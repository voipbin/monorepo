package notifyhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
)

// subscriptionIDData is the minimal shape used to extract the top-level "id" of an already
// marshaled event payload. It is the default subscription address of every event whose data type
// does not implement eventtopic.SubscriptionIdentifier (VOIP-1404 design §4.2).
type subscriptionIDData struct {
	ID string `json:"id"`
}

// PublishWebhookEvent publishs the given event type of notification to the webhook and event queue.
// Note: These goroutines are intentionally fire-and-forget. The passed context is used for
// request handlers but cancellation is not propagated since notifications should complete
// independently of the caller's lifecycle.
func (h *notifyHandler) PublishWebhookEvent(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookMessage) {
	go h.PublishEvent(ctx, eventType, data)
	go h.PublishWebhook(ctx, customerID, eventType, data)
}

// PublishWebhook publishes the webhook to the given customer.
func (h *notifyHandler) PublishWebhook(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookMessage) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "PublishWebhook",
		"customer_id": customerID,
		"data":        data,
		"evnet_type":  eventType,
	})

	if customerID == uuid.Nil {
		// no customer id given
		return
	}

	// create webhook event
	m, err := data.CreateWebhookEvent()
	if err != nil {
		log.Errorf("Could not marshal the message. err: %v", err)
		return
	}

	if err := h.reqHandler.WebhookV1WebhookSend(ctx, customerID, wmwebhook.DataTypeJSON, eventType, m); err != nil {
		log.Errorf("Could not publish the webhook. err: %v", err)
		return
	}
}

// PublishEventRaw publishes the raw event to the event queue.
//
// NOTE (VOIP-1404): the subscription id is always passed as "" here -- a []byte payload cannot
// satisfy the eventtopic.SubscriptionIdentifier assertion -- so the global topic publish falls
// back to the top-level "id" of the payload, if any.
func (h *notifyHandler) PublishEventRaw(ctx context.Context, eventType string, dataType string, data []byte) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "PublishEventRaw",
		"evnet_type": eventType,
		"data_type":  dataType,
	})

	if err := h.publishEvent(eventType, dataType, data, requestTimeoutDefault, 0, ""); err != nil {
		log.Errorf("Could not publish the call event. err: %v", err)
		return
	}
}

// PublishEvent publishes event to the event queue.
func (h *notifyHandler) PublishEvent(ctx context.Context, eventType string, data interface{}) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "PublishEvent",
		"evnet_type": eventType,
	})

	// create event
	m, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Could not marshal the message. err: %v", err)
		return
	}

	// VOIP-1404: resolve the subscription address here, while the data is still a typed value.
	// publishEvent only sees the marshaled bytes, so the opt-in override cannot be resolved there.
	// The assertion matches a pointer dynamic type -- implementations use pointer receivers.
	subscriptionID := ""
	if identifier, ok := data.(eventtopic.SubscriptionIdentifier); ok {
		subscriptionID = identifier.EventSubscriptionID()
	}

	if err := h.publishEvent(string(eventType), string(wmwebhook.DataTypeJSON), m, requestTimeoutDefault, 0, subscriptionID); err != nil {
		log.Errorf("Could not publish the call event. err: %v", err)
		return
	}
}

// publishEvent publishes a event to the event queue.
//
// subscriptionID is the resolved subscription address of the event, or "" when the caller could
// not resolve one (VOIP-1404). It is only used by the global topic publish, which falls back to
// the payload's top-level "id" when it is empty.
func (h *notifyHandler) publishEvent(eventType string, dataType string, data json.RawMessage, timeout int, delay int, subscriptionID string) error {

	// create a event
	evt := &sock.Event{
		Type:      eventType,
		Publisher: string(h.publisher),
		DataType:  dataType,
		Data:      data,
	}

	// Note: Using context.Background() intentionally. Events are fire-and-forget notifications
	// that should complete independently of the caller's context lifecycle. The timeout provides
	// its own cancellation mechanism.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	defer cancel()

	switch {
	case delay > 0:
		// send scheduled message.
		// we don't expect the response message here.
		if err := h.publishDelayedEvent(ctx, delay, evt); err != nil {
			return fmt.Errorf("could not publish the delayed event. err: %v", err)
		}
		return nil

	default:
		err := h.publishDirectEvent(ctx, evt)
		if err != nil {
			// NOTE: the fanout publish is the system of record while dual publish lasts, so the
			// global topic publish below is skipped on this path. Delivering an event on the topic
			// exchange that the fanout consumers never saw would create divergent state.
			return fmt.Errorf("could not publish the event. err: %v", err)
		}
	}
	promNotifyTotal.WithLabelValues(evt.Type).Inc()

	// VOIP-1404: dual publish. Reached only when delay == 0 -- the delayed branch above returns
	// early, since delayed-event topic semantics are deferred to the follow-up.
	h.publishTopicEvent(evt, subscriptionID)

	return nil
}

// publishTopicEvent publishes the given event to the global topic exchange `bin-manager.event`
// with a `<publisher>.<resource>.<subscription-id>.<action>` routing key (VOIP-1404).
//
// The payload is the very same sock.Event that was just published to the fanout exchange, so a
// consumer migrating later reuses its existing decode path unchanged.
//
// A failure is logged and counted, never returned: the topic publish must not affect the fanout
// publish nor the caller in any way. It deliberately calls sockHandler.EventPublish directly
// instead of reusing publishDirectEvent/publishDirectEventWithKey, both of which observe
// promNotifyProcessTime -- reusing them would pollute the existing fanout metrics.
func (h *notifyHandler) publishTopicEvent(evt *sock.Event, subscriptionID string) {
	if !h.topicEnabled {
		return
	}

	log := logrus.WithFields(logrus.Fields{
		"func":       "publishTopicEvent",
		"evnet_type": evt.Type,
	})

	if h.topicDisabled {
		// the exchange declare failed at construction time. count every suppressed publish so the
		// degradation stays visible: the error counter grows while the ok counter does not.
		promTopicPublishTotal.WithLabelValues(evt.Type, topicPublishResultError).Inc()
		return
	}

	if subscriptionID == "" {
		subscriptionID = parseSubscriptionID(evt.Data)
	}
	if subscriptionID == "" || subscriptionID == uuid.Nil.String() {
		// no valid subscription address exists. the routing key falls back to the placeholder,
		// which type-level bindings still match. metered so absent-id drift stays visible.
		promTopicPlaceholderTotal.WithLabelValues(evt.Type).Inc()
	}

	key := eventtopic.RoutingKey(string(h.publisher), evt.Type, subscriptionID)
	if err := h.sockHandler.EventPublish(string(commonoutline.QueueNameEvent), key, evt); err != nil {
		log.Errorf("Could not publish the event to the global topic exchange. routing_key: %s, err: %v", key, err)
		promTopicPublishTotal.WithLabelValues(evt.Type, topicPublishResultError).Inc()
		return
	}
	promTopicPublishTotal.WithLabelValues(evt.Type, topicPublishResultOK).Inc()
}

// parseSubscriptionID extracts the top-level "id" of an already marshaled event payload. Returns
// "" when the payload is empty, is not a JSON object, or carries no "id" -- all of which end up
// as the placeholder segment in the routing key.
func parseSubscriptionID(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}

	d := &subscriptionIDData{}
	if err := json.Unmarshal(data, d); err != nil {
		return ""
	}

	return d.ID
}

// PublishEventWithRoutingKey publishes event to the event queue with an explicit AMQP routing
// key, for topic-kind exchanges. Unlike PublishEvent (which always publishes with an empty
// routing key, correct for fanout exchanges), this lets the caller target scope-based topic
// bindings. See VOIP-1258 design doc §6.
func (h *notifyHandler) PublishEventWithRoutingKey(ctx context.Context, eventType string, routingKey string, data interface{}) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "PublishEventWithRoutingKey",
		"evnet_type":  eventType,
		"routing_key": routingKey,
	})

	m, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Could not marshal the message. err: %v", err)
		return
	}

	evt := &sock.Event{
		Type:      eventType,
		Publisher: string(h.publisher),
		DataType:  string(wmwebhook.DataTypeJSON),
		Data:      m,
	}

	// Note: Using context.Background() intentionally, matching publishEvent's existing
	// fire-and-forget rationale -- this event should complete independently of the caller's
	// context lifecycle.
	pubCtx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(requestTimeoutDefault))
	defer cancel()

	if err := h.publishDirectEventWithKey(pubCtx, routingKey, evt); err != nil {
		log.Errorf("Could not publish the event with routing key. err: %v", err)
		return
	}
	promNotifyTotal.WithLabelValues(evt.Type).Inc()
}

// publishDirectEventWithKey publishes the event to the target without delay, using the given
// AMQP routing key instead of publishDirectEvent's hardcoded empty key. Shares the same
// publish+metrics logic as publishDirectEvent (VOIP-1258 code-quality review: avoid duplicating
// this logic in a second near-identical function).
func (h *notifyHandler) publishDirectEventWithKey(ctx context.Context, key string, evt *sock.Event) error {

	start := time.Now()
	err := h.sockHandler.EventPublish(string(h.queueNotify), key, evt)
	elapsed := time.Since(start)
	promNotifyProcessTime.WithLabelValues(string(evt.Type)).Observe(float64(elapsed.Milliseconds()))

	return err
}

// publishDirectEvent publish the event to the target without delay
func (h *notifyHandler) publishDirectEvent(ctx context.Context, evt *sock.Event) error {

	start := time.Now()
	err := h.sockHandler.EventPublish(string(h.queueNotify), "", evt)
	elapsed := time.Since(start)
	promNotifyProcessTime.WithLabelValues(string(evt.Type)).Observe(float64(elapsed.Milliseconds()))

	return err
}

// publishDelayedEvent sends the delayed event
// delay unit is millisecond.
func (h *notifyHandler) publishDelayedEvent(ctx context.Context, delay int, evt *sock.Event) error {

	start := time.Now()
	err := h.sockHandler.EventPublishWithDelay(string(commonoutline.QueueNameDelay), string(h.queueNotify), evt, delay)
	elapsed := time.Since(start)
	promNotifyProcessTime.WithLabelValues(string(evt.Type)).Observe(float64(elapsed.Milliseconds()))

	return err
}

