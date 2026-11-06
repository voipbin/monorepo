package emailhandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"monorepo/bin-email-manager/models/email"
	mmailgun "monorepo/bin-email-manager/models/mailgun"
	"monorepo/bin-email-manager/models/sendgrid"
	"strconv"
	"strings"
	"time"

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

// Mailgun webhook signature timestamp window. Mailgun retries a rejected
// delivery for about 8 hours, so the window is wider than that.
const (
	mailgunSignatureMaxAge  = 24 * time.Hour
	mailgunSignatureMaxSkew = 5 * time.Minute
)

// verifyMailgunSignature checks the signature of a Mailgun webhook request:
// hex(HMAC-SHA256(key=signing key, msg=timestamp+token)) must equal the
// signature, and the timestamp must be close to the current time.
// The error never contains the key, the signature, the token or the timestamp.
func (h *emailHandler) verifyMailgunSignature(sig *mmailgun.Signature) error {
	if h.mailgunWebhookSigningKey == "" {
		return errors.New("the mailgun webhook signing key is not configured")
	}
	if sig == nil {
		return errors.New("the signature is missing")
	}
	if sig.Token == "" || sig.Signature == "" || len(sig.Timestamp) == 0 {
		return errors.New("the signature is incomplete")
	}

	// the timestamp is a string in the documented payload; a bare number is accepted too.
	// the text is used as is, because it is what was signed.
	timestamp := string(sig.Timestamp)
	if strings.HasPrefix(timestamp, `"`) {
		if errUnmarshal := json.Unmarshal(sig.Timestamp, &timestamp); errUnmarshal != nil {
			return errors.New("the signature timestamp is invalid")
		}
	}
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("the signature timestamp is invalid")
	}

	got, err := hex.DecodeString(sig.Signature)
	if err != nil {
		return errors.New("the signature is not hexadecimal")
	}

	mac := hmac.New(sha256.New, []byte(h.mailgunWebhookSigningKey))
	mac.Write([]byte(timestamp + sig.Token))
	if !hmac.Equal(got, mac.Sum(nil)) {
		return errors.New("the signature does not match")
	}

	// checked after the signature so that only a genuine request tells the reason.
	now := time.Now()
	if h.timeNow != nil {
		now = h.timeNow()
	}
	age := now.Sub(time.Unix(seconds, 0))
	if age > mailgunSignatureMaxAge || age < -mailgunSignatureMaxSkew {
		return errors.New("the signature timestamp is out of the allowed range")
	}

	return nil
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
	if errVerify := h.verifyMailgunSignature(hook.Signature); errVerify != nil {
		// nothing from the request is logged: it carries the recipient address and the signature material.
		log.Warnf("Rejected the mailgun webhook. reason: %v", errVerify)
		return errors.Wrapf(errVerify, "could not verify the mailgun webhook signature.")
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
