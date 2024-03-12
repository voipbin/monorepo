package notifyhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	wmwebhook "gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"

	commonoutline "gitlab.com/voipbin/bin-manager/common-handler.git/models/outline"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// PublishWebhookEvent publishs the given event type of notification to the webhook and event queue.
func (h *notifyHandler) PublishWebhookEvent(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookMessage) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "PublishWebhookEvent",
		"evnet_type":  eventType,
		"event":       data,
		"customer_id": customerID,
	})
	log.Debugf("publishing the event to the webhook and event queue.. event_type: %s", eventType)

	go h.PublishEvent(ctx, eventType, data)
	go h.PublishWebhook(ctx, customerID, eventType, data)
}

// PublishWebhook publishes the webhook to the given customer.
func (h *notifyHandler) PublishWebhook(ctx context.Context, customerID uuid.UUID, eventType string, data WebhookMessage) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "PublishWebhook",
		"call":       data,
		"evnet_type": eventType,
	})
	log.Debugf("Sending webhook event. event_type: %s, message: %s", eventType, data)

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
func (h *notifyHandler) PublishEventRaw(ctx context.Context, eventType string, dataType string, data []byte) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "PublishEventRaw",
		"evnet_type": eventType,
		"data_type":  dataType,
	})

	if err := h.publishEvent(eventType, dataType, data, requestTimeoutDefault, 0); err != nil {
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

	if err := h.publishEvent(string(eventType), string(wmwebhook.DataTypeJSON), m, requestTimeoutDefault, 0); err != nil {
		log.Errorf("Could not publish the call event. err: %v", err)
		return
	}
}

// publishEvent publishes a event to the event queue.
func (h *notifyHandler) publishEvent(eventType string, dataType string, data json.RawMessage, timeout int, delay int) error {

	log := logrus.WithFields(logrus.Fields{
		"func":      "publishEvent",
		"type":      eventType,
		"data_type": dataType,
		"data":      data,
		"timeout":   timeout,
		"delay":     delay,
	})
	log.Debugf("Publishing the event. type: %s", eventType)

	// create a event
	evt := &rabbitmqhandler.Event{
		Type:      eventType,
		Publisher: string(h.publisher),
		DataType:  dataType,
		Data:      data,
	}

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
			return fmt.Errorf("could not publish the event. err: %v", err)
		}
	}

	promNotifyTotal.WithLabelValues(evt.Type).Inc()
	log.WithFields(logrus.Fields{
		"event": evt,
	}).Debugf("Published event. queue_name: %s, event_publisher: %s, event_type: %s, data_type: %s", h.queueNotify, evt.Publisher, evt.Type, evt.DataType)

	return nil

}

// publishDirectEvent publish the event to the target without delay
func (h *notifyHandler) publishDirectEvent(ctx context.Context, evt *rabbitmqhandler.Event) error {

	start := time.Now()
	err := h.sock.PublishExchangeEvent(string(h.queueNotify), "", evt)
	elapsed := time.Since(start)
	promNotifyProcessTime.WithLabelValues(string(evt.Type)).Observe(float64(elapsed.Milliseconds()))

	return err
}

// publishDelayedEvent sends the delayed event
// delay unit is millisecond.
func (h *notifyHandler) publishDelayedEvent(ctx context.Context, delay int, evt *rabbitmqhandler.Event) error {

	start := time.Now()
	err := h.sock.PublishExchangeDelayedEvent(string(commonoutline.QueueNameDelay), string(h.queueNotify), evt, delay)
	elapsed := time.Since(start)
	promNotifyProcessTime.WithLabelValues(string(evt.Type)).Observe(float64(elapsed.Milliseconds()))

	return err
}
