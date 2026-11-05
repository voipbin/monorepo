package emailhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-email-manager/models/email"
	mmailgun "monorepo/bin-email-manager/models/mailgun"
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

// statusLevels orders the statuses by how far the delivery has progressed.
// Webhook events are not delivered in order and may be retried, so a status
// is applied only if it is in a strictly later level than the current one.
// Statuses in the same level never replace each other.
var statusLevels = [][]email.Status{
	{email.StatusNone, email.StatusInitiated},
	{email.StatusProcessed},
	{email.StatusDeffered},
	{email.StatusDelivered},
	{email.StatusBounce, email.StatusDropped},
	{email.StatusOpen},
	{email.StatusClick},
	{email.StatusUnsubscribe},
	{email.StatusSpamreport},
}

// statusesBefore returns the statuses that the given status may replace.
func statusesBefore(status email.Status) []email.Status {
	res := []email.Status{}
	for _, level := range statusLevels {
		for _, s := range level {
			if s == status {
				return res
			}
		}
		res = append(res, level...)
	}

	return nil
}

// mailgunEventToStatus maps the mailgun webhook event to the email status.
// It returns false for the events that have no matching status.
func mailgunEventToStatus(e *mmailgun.Event) (email.Status, bool) {
	switch e.Event {
	case mmailgun.EventAccepted:
		return email.StatusProcessed, true

	case mmailgun.EventDelivered:
		return email.StatusDelivered, true

	case mmailgun.EventFailed:
		switch e.Severity {
		case mmailgun.SeverityTemporary:
			return email.StatusDeffered, true
		case mmailgun.SeverityPermanent:
			return email.StatusBounce, true
		}
		return email.StatusNone, false

	case mmailgun.EventOpened:
		return email.StatusOpen, true

	case mmailgun.EventClicked:
		return email.StatusClick, true

	case mmailgun.EventUnsubscribed:
		return email.StatusUnsubscribe, true

	case mmailgun.EventComplained:
		return email.StatusSpamreport, true
	}

	return email.StatusNone, false
}

// mailgunEmailID returns the voipbin email id carried by the event's user variables.
func mailgunEmailID(e *mmailgun.Event) uuid.UUID {
	// "user-variables" is [] when empty and an object otherwise.
	vars := map[string]json.RawMessage{}
	if errUnmarshal := json.Unmarshal(e.UserVariables, &vars); errUnmarshal != nil {
		return uuid.Nil
	}

	var value string
	if errUnmarshal := json.Unmarshal(vars[mmailgun.UserVariableMessageID], &value); errUnmarshal != nil {
		return uuid.Nil
	}

	return uuid.FromStringOrNil(value)
}

func (h *emailHandler) hookMailgun(ctx context.Context, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "hookMailgun",
	})

	// the event is under "event-data" or is the body itself.
	hook := mmailgun.Webhook{}
	if errUnmarshal := json.Unmarshal(data, &hook); errUnmarshal != nil {
		// the data is not logged because it contains the recipient address.
		return errors.Wrapf(errUnmarshal, "could not unmarshal the mailgun webhook.")
	}
	e := &hook.Event
	if hook.EventData != nil {
		e = hook.EventData
	}
	if e.Event == "" {
		return errors.New("the mailgun webhook has no event.")
	}

	status, ok := mailgunEventToStatus(e)
	if !ok {
		log.Debugf("Ignoring the mailgun event. event: %s, severity: %s", e.Event, e.Severity)
		return nil
	}

	id := mailgunEmailID(e)
	if id == uuid.Nil {
		log.Debugf("The mailgun event has no email id. Ignoring. event: %s", e.Event)
		return nil
	}

	log = log.WithFields(logrus.Fields{
		"email_id": id,
		"event":    e.Event,
		"severity": e.Severity,
		"reason":   e.Reason,
	})

	changed, err := h.db.EmailUpdateStatusFrom(ctx, id, status, statusesBefore(status))
	if err != nil {
		return errors.Wrapf(err, "could not update the status. email_id: %s, status: %s", id, status)
	}
	if !changed {
		log.Debugf("The status is not updated. The event is stale or duplicated, or the email does not exist. status: %s", status)
		return nil
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "could not get the updated email. email_id: %s", id)
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, email.EventTypeUpdated, res)

	return nil
}
