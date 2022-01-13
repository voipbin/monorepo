package notifyhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// PublishWebhookEvent publishs the given event type of notification to the webhook and event queue.
func (h *notifyHandler) PublishWebhookEvent(ctx context.Context, eventType string, webhookURI string, message WebhookMessage) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "PublishWebhookEvent",
			"evnet_type":  eventType,
			"event":       message,
			"webhook_uri": webhookURI,
		},
	)
	log.Debugf("publishing the event to the webhook and event queue.. event_type: %s", eventType)

	go h.PublishEvent(ctx, eventType, message)
	go h.PublishWebhook(ctx, eventType, webhookURI, message)
}

// PublishWebhook publishes the webhook to the given destination.
func (h *notifyHandler) PublishWebhook(ctx context.Context, t string, webhookURI string, c WebhookMessage) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "PublishWebhook",
			"call":       c,
			"evnet_type": t,
		},
	)
	log.Debugf("Sending webhook event. event_type: %s, message: %s", t, c)

	if webhookURI == "" {
		// no webhook uri
		return
	}

	// create webhook event
	m, err := c.CreateWebhookEvent()
	if err != nil {
		log.Errorf("Could not marshal the message. err: %v", err)
		return
	}

	if err := h.reqHandler.WMV1WebhookSend(ctx, "POST", webhookURI, dataTypeJSON, string(t), m); err != nil {
		log.Errorf("Could not publish the webhook. err: %v", err)
		return
	}
}

// PublishEvent publishes event to the event queue.
func (h *notifyHandler) PublishEvent(ctx context.Context, t string, c interface{}) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "PublishEvent",
			"evnet_type": t,
		},
	)

	// create event
	m, err := json.Marshal(c)
	if err != nil {
		log.Errorf("Could not marshal the message. err: %v", err)
		return
	}

	if err := h.publishEvent(string(t), dataTypeJSON, m, requestTimeoutDefault, 0); err != nil {
		log.Errorf("Could not publish the call event. err: %v", err)
		return
	}
}

// publishEvent publishes a event to the event queue.
func (h *notifyHandler) publishEvent(eventType string, dataType string, data json.RawMessage, timeout int, delay int) error {

	log := logrus.WithFields(
		logrus.Fields{
			"func":      "publishEvent",
			"type":      eventType,
			"data_type": dataType,
			"data":      data,
		},
	)
	log.Debugf("Publishing the event. type: %s", eventType)

	// create a event
	evt := &rabbitmqhandler.Event{
		Type:      eventType,
		Publisher: h.publisher,
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
	}).Debugf("Published event. type: %s", evt.Type)

	return nil

}

// publishDirectEvent publish the event to the target without delay
func (h *notifyHandler) publishDirectEvent(ctx context.Context, evt *rabbitmqhandler.Event) error {

	start := time.Now()
	err := h.sock.PublishExchangeEvent(h.exchangeNotify, "", evt)
	elapsed := time.Since(start)
	promNotifyProcessTime.WithLabelValues(string(evt.Type)).Observe(float64(elapsed.Milliseconds()))

	return err
}

// publishDelayedEvent sends the delayed event
// delay unit is millisecond.
func (h *notifyHandler) publishDelayedEvent(ctx context.Context, delay int, evt *rabbitmqhandler.Event) error {

	start := time.Now()
	err := h.sock.PublishExchangeDelayedEvent(h.exchangeDelay, h.exchangeNotify, evt, delay)
	elapsed := time.Since(start)
	promNotifyProcessTime.WithLabelValues(string(evt.Type)).Observe(float64(elapsed.Milliseconds()))

	return err
}
