package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
)

// processEventMMMessageCreated handles the message-manager's messages_created event
func (h *subscribeHandler) processEventMMMessageCreated(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventMMMessageCreated",
		"event": m,
	})
	log.Debugf("Received call event. event: %s", m.Type)

	var c mmmessage.Message
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return errors.Wrap(err, "could not unmarshal the data")
	}

	if errEvent := h.billingHandler.EventMMMessageCreated(ctx, &c); errEvent != nil {
		log.Errorf("Could not handle the event. err: %v", errEvent)
		return errEvent
	}

	// for _, target := range c.Targets {
	// 	if errBilling := h.billingHandler.BillingStart(ctx, c.CustomerID, billing.ReferenceTypeSMS, c.ID, c.TMCreate, c.Source, &target.Destination); errBilling != nil {
	// 		log.Errorf("Could not create a billing. target: %v, err: %v", target, errBilling)
	// 		return errors.Wrap(errBilling, "could not create a billing")
	// 	}
	// }

	return nil
}
