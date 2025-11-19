package emailhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/models/sendgrid"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *emailHandler) Hook(ctx context.Context, uri string, data []byte) error {
	var err error

	suffix := ""
	if idx := strings.LastIndex(uri, "/"); idx != -1 {
		suffix = uri[idx+1:]
	}

	switch suffix {
	case hookSendgrid:
		err = h.hookSendgrid(ctx, data)

	case hookMailgun:
		err = h.hookMailgun(ctx, data)

	default:
		err = errors.Errorf("unknown hook uri: %s", uri)
	}
	if err != nil {
		return errors.Wrapf(err, "could not handle the hook message correctly.")
	}

	return nil
}

func (h *emailHandler) hookSendgrid(ctx context.Context, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "hookSendgrid",
	})

	events := []sendgrid.SendGridEvent{}
	if errUnmarshal := json.Unmarshal(data, &events); errUnmarshal != nil {
		return errors.Wrapf(errUnmarshal, "could not unmarshal the sendgrid events. data: %s", string(data))
	}

	for i := range events {
		// note: because of the sendtrid sends the events in reverse order, we need to process the events in reverse order
		e := events[len(events)-1-i]
		id := uuid.FromStringOrNil(e.VoipbinMessageID)
		if id == uuid.Nil {
			log.WithField("event", e).Errorf("could not get the email id.")
			continue
		}

		_, err := h.UpdateStatus(ctx, id, email.Status(e.Event))
		if err != nil {
			log.WithField("event", e).Errorf("could not update the status. id: %s, event: %s, err: %v", id, e.Event, err)
			continue
		}
	}

	return nil
}

func (h *emailHandler) hookMailgun(ctx context.Context, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "hookMailgun",
	})

	log.WithField("data", string(data)).Debug("Received mailgun webhook data.")

	// TODO: implement mailgun webhook handling

	return nil
}
