package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	ememail "monorepo/bin-email-manager/models/email"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventEMEmailCreated handles the email-manager's email_created event
func (h *subscribeHandler) processEventEMEmailCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventEMEmailCreated",
		"event": m,
	})
	log.Debugf("Received email event. event: %s", m.Type)

	var e ememail.Email
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return errors.Wrap(err, "could not unmarshal the data")
	}

	if errEvent := h.billingHandler.EventEMEmailCreated(ctx, &e); errEvent != nil {
		log.Errorf("Could not handle the event. err: %v", errEvent)
		return errEvent
	}

	return nil
}
