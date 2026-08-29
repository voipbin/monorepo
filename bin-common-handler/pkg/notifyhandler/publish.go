package notifyhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/eventtopic"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
)

// PublishWebhookEvent publishs the given event type of notification to the webhook and event queue.
// Note: These goroutines are intentionally fire-and-forget. The passed context is used for
// request handlers but cancellation is not propagated since notifications should complete
// independently of the caller's lifecycle.
func (h *notifyHandler) PublishWebhookEvent(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookEventMessage) {
	go h.PublishEvent(ctx, eventType, data)
	go h.PublishWebhook(ctx, customerID, eventType, data)
}

// PublishWebhook publishes the webhook to the given customer.
func (h *notifyHandler) PublishWebhook(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookMessage) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "PublishWebhook",
		"customer_id": customerID,
		"data":        data,
		"event_type":  eventType,
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
// NOTE (VOIP-1419): a []byte payload cannot carry an EventSubscriptionID, so on a topic-enabled
// handler every Raw publish lands on the global topic exchange under the `-` placeholder. The
// sole production caller (voip-asterisk-proxy's ARI passthrough) is not topic-enabled, so this is
// a documented property, not an active behavior change.
func (h *notifyHandler) PublishEventRaw(ctx context.Context, eventType string, dataType string, data []byte) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "PublishEventRaw",
		"event_type": eventType,
		"data_type":  dataType,
	})

	if err := h.publishEvent(eventType, dataType, data, requestTimeoutDefault, 0, ""); err != nil {
		log.Errorf("Could not publish the call event. err: %v", err)
		return
	}
}

// PublishEvent publishes event to the event queue.
//
// The data parameter REQUIRES eventtopic.SubscriptionIdentifier (VOIP-1419): every published
// event data type declares its subscription address explicitly, enforced by the compiler at the
// call site. There is no JSON fallback -- the marshaled payload's top-level "id" plays no role in
// routing-key resolution.
func (h *notifyHandler) PublishEvent(ctx context.Context, eventType string, data eventtopic.SubscriptionIdentifier) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "PublishEvent",
		"event_type": eventType,
	})

	// create event
	m, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Could not marshal the message. err: %v", err)
		return
	}

	// VOIP-1404: resolve the subscription address here, while the data is still a typed value.
	// publishEvent only sees the marshaled bytes. Gated on topicEnabled: with the option off,
	// nothing about the pre-VOIP-1404 fanout path may change -- not even a method call on the
	// caller's data.
	subscriptionID := ""
	if h.topicEnabled {
		subscriptionID = resolveSubscriptionID(data)
	}

	if err := h.publishEvent(string(eventType), string(wmwebhook.DataTypeJSON), m, requestTimeoutDefault, 0, subscriptionID); err != nil {
		log.Errorf("Could not publish the call event. err: %v", err)
		return
	}
}

// resolveSubscriptionID resolves the subscription address of the given event data (VOIP-1419).
// Two guard branches degrade to "" (-> the `-` placeholder) instead of calling the method:
//
//  1. nil interface: an untyped nil argument still compiles against the interface parameter, and
//     reflect reports Kind Invalid for it -- the typed-nil branch below would miss it, and calling
//     the method on a nil interface panics. Pre-VOIP-1419 this input was safe only because the
//     type ASSERTION failed first; with the assertion gone, this explicit check is load-bearing.
//  2. typed nil: a nil pointer whose type implements the interface, e.g. a forwarded nil
//     *transcript.Transcript. Every real implementation dereferences its receiver, so calling the
//     method would panic on the caller's goroutine. Such a payload marshals to `null` -- the `-`
//     placeholder is the correct address for a payload that carries nothing.
func resolveSubscriptionID(data eventtopic.SubscriptionIdentifier) string {
	if data == nil {
		return ""
	}

	if v := reflect.ValueOf(data); v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}

	return data.EventSubscriptionID()
}

// publishEvent publishes a event to the event queue.
//
// subscriptionID is the subscription address resolved from the event data's mandatory
// EventSubscriptionID (VOIP-1419), used only by the global topic publish. An empty value (nil
// payload, typed-nil payload, or a raw []byte publish) degrades to the `-` placeholder.
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

	case h.topicEnabled:
		// Topic-only path (VOIP-1407): no fanout publish. publishTopicEventOrErr's own error
		// already carries full context (including routing_key), so it is returned directly, not
		// re-wrapped, to avoid a doubled "could not publish the event to the global topic
		// exchange" prefix.
		if err := h.publishTopicEventOrErr(ctx, evt, subscriptionID); err != nil {
			return err
		}

	default:
		// Fanout-only path, unchanged: voip-asterisk-proxy and confirmed-dead sites.
		if err := h.publishDirectEvent(ctx, evt); err != nil {
			return fmt.Errorf("could not publish the event. err: %v", err)
		}
	}
	promNotifyTotal.WithLabelValues(evt.Type).Inc()

	return nil
}

// publishTopicEventOrErr publishes the given event to the global topic exchange
// `bin-manager.event` with a `<publisher>.<resource>.<subscription-id>.<action>` routing key
// (VOIP-1404).
//
// VOIP-1407: this is now the primary publish path for topic-enabled instances, not a best-effort
// secondary one, so a failure is returned to the caller instead of being logged and swallowed.
// It observes promNotifyProcessTime under the same metric name/labels the removed fanout leg
// used to -- with only one active publish path per instance, reusing the name cannot double-count
// anything. The routing key is folded into the returned error string (rather than a separate
// log call) so it survives all the way to publishEvent's caller, since this function's caller no
// longer swallows the error.
func (h *notifyHandler) publishTopicEventOrErr(ctx context.Context, evt *sock.Event, subscriptionID string) error {
	start := time.Now()
	defer func() {
		promNotifyProcessTime.WithLabelValues(evt.Type).Observe(float64(time.Since(start).Milliseconds()))
	}()

	if eventtopic.IsPlaceholderSubscriptionID(subscriptionID) {
		// no valid subscription address exists. the routing key falls back to the placeholder,
		// which type-level bindings still match. metered so absent-id drift stays visible. the
		// predicate lives in eventtopic so this counter can never disagree with the key the very
		// next line generates.
		promTopicPlaceholderTotal.WithLabelValues(evt.Type).Inc()
	}

	key := eventtopic.RoutingKey(string(h.publisher), evt.Type, subscriptionID)
	if err := h.sockHandler.EventPublish(string(commonoutline.QueueNameEvent), key, evt); err != nil {
		promTopicPublishTotal.WithLabelValues(evt.Type, topicPublishResultError).Inc()
		return fmt.Errorf("could not publish the event to the global topic exchange. routing_key: %s, err: %v", key, err)
	}
	promTopicPublishTotal.WithLabelValues(evt.Type, topicPublishResultOK).Inc()
	return nil
}

// PublishEventWithRoutingKey publishes event to the event queue with an explicit AMQP routing
// key, for topic-kind exchanges. Unlike PublishEvent (which always publishes with an empty
// routing key, correct for fanout exchanges), this lets the caller target scope-based topic
// bindings. See VOIP-1258 design doc §6.
func (h *notifyHandler) PublishEventWithRoutingKey(ctx context.Context, eventType string, routingKey string, data interface{}) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "PublishEventWithRoutingKey",
		"event_type":  eventType,
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

