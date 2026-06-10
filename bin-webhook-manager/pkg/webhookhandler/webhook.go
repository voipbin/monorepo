package webhookhandler

import (
	"context"
	"encoding/json"
	"fmt"

	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webhook-manager/models/webhook"
)

// webhookEnvelope is the nested wire envelope delivered to SendWebhookToCustomer:
// {"type":"call_updated","data":{...resource WebhookMessage with activeflow_id...}}.
// The activeflow_id lives at data.data.activeflow_id (NESTED), NOT top level (design 5.2).
type webhookEnvelope struct {
	Data struct {
		ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
	} `json:"data"`
}

// SendWebhookToCustomer sends the webhook to the given customerID with the given method and data.
func (h *webhookHandler) SendWebhookToCustomer(ctx context.Context, customerID uuid.UUID, dataType webhook.DataType, data json.RawMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SendWebhookToCustomer",
		"customer_id": customerID,
	})
	log.WithFields(logrus.Fields{
		"data_type": dataType,
		"data":      data,
	}).Debugf("Sending an webhook. customer_id: %s", customerID)

	// system customers have no webhook configuration
	if customerID == cscustomer.IDSystem {
		log.Debugf("Skipping webhook for system customer. customer_id: %s", customerID)
		return nil
	}

	m, err := h.accoutHandler.Get(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get account. err: %v", err)
		return fmt.Errorf("could not get account. err: %v", err)
	}

	if m.WebhookURI != "" {
		// send webhook message
		go func() {
			if err := h.sendMessage(m.WebhookURI, string(m.WebhookMethod), string(dataType), data); err != nil {
				promDeliveryTotal.WithLabelValues("customer", "error").Inc()
				log.Errorf("Could not send a request. err: %v", err)
				return
			}
			promDeliveryTotal.WithLabelValues("customer", "success").Inc()
		}()
	}

	wh := &webhook.Webhook{
		CustomerID: customerID,
		DataType:   dataType,
		Data:       data,
	}
	h.notifyHandler.PublishEvent(ctx, webhook.EventTypeWebhookPublished, wh)

	// additionally deliver to the per-activeflow webhook destination, if any.
	h.sendWebhookToActiveflow(ctx, dataType, data)

	return nil
}

// sendWebhookToActiveflow extracts the nested activeflow_id from the data
// envelope and, if a positive per-activeflow destination is resolved, delivers
// the same payload there additionally. It never affects the customer delivery.
func (h *webhookHandler) sendWebhookToActiveflow(ctx context.Context, dataType webhook.DataType, data json.RawMessage) {
	if h.activeflowHandler == nil {
		return
	}

	var env webhookEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		// best-effort: malformed payload simply skips the extra delivery.
		return
	}

	activeflowID := env.Data.ActiveflowID
	if activeflowID == uuid.Nil {
		// no activeflow_id present: customer-only delivery (unchanged).
		return
	}

	log := logrus.WithFields(logrus.Fields{
		"func":          "sendWebhookToActiveflow",
		"activeflow_id": activeflowID,
	})

	dest, err := h.activeflowHandler.Get(ctx, activeflowID)
	if err != nil {
		log.Errorf("Could not resolve the activeflow webhook destination. err: %v", err)
		return
	}
	if dest == nil {
		// negative / no destination: skip the extra delivery.
		return
	}

	log.Debugf("Delivering webhook to the per-activeflow destination. uri: %s", dest.URI)
	go func() {
		if err := h.sendMessage(dest.URI, string(dest.Method), string(dataType), data); err != nil {
			promDeliveryTotal.WithLabelValues("activeflow", "error").Inc()
			log.Errorf("Could not send a request to the activeflow destination. err: %v", err)
			return
		}
		promDeliveryTotal.WithLabelValues("activeflow", "success").Inc()
	}()
}

// SendWebhookToURI sends the webhook to the given uri with the given method and data.
func (h *webhookHandler) SendWebhookToURI(ctx context.Context, customerID uuid.UUID, uri string, method webhook.MethodType, dataType webhook.DataType, data json.RawMessage) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SendWebhookToURI",
		"customer_id": customerID,
		"uri":         uri,
	})
	log.WithFields(logrus.Fields{
		"data_type": dataType,
		"data":      data,
	}).Debugf("Sending an webhook. customer_id: %s", customerID)

	// send message
	go func() {
		if err := h.sendMessage(uri, string(method), string(dataType), data); err != nil {
			log.Errorf("Could not send a request. err: %v", err)
		}
	}()

	wh := &webhook.Webhook{
		CustomerID: customerID,
		DataType:   dataType,
		Data:       data,
	}
	h.notifyHandler.PublishEvent(ctx, webhook.EventTypeWebhookPublished, wh)

	return nil
}
