package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	emmemail "monorepo/bin-email-manager/models/email"

	"github.com/sirupsen/logrus"
)

// processEventEmailEmailCreated handles the email-manager's email_created event.
//
// Each sent email is recorded as an outgoing conversation message (outbound-only).
func (h *subscribeHandler) processEventEmailEmailCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventEmailEmailCreated",
		"event": m,
	})

	e := &emmemail.Email{}
	if err := json.Unmarshal([]byte(m.Data), e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.conversationHandler.EmailEventSent(ctx, e); errEvent != nil {
		log.Errorf("Could not handle the event correctly. err: %v", errEvent)
		return errEvent
	}

	return nil
}
