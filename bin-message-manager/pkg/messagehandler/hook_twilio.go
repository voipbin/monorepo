package messagehandler

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

// twilioCallbackMaxBody is the largest body of a twilio callback that is read. A status callback is a short form.
const twilioCallbackMaxBody = 64 * 1024

// twilioSIDPattern is the shape of the id of a twilio message: SM or MM and 32 hex digits.
var twilioSIDPattern = regexp.MustCompile(`^(SM|MM)[0-9a-fA-F]{32}$`)

// reasons of a refused twilio callback (labels of twilio_callback_rejected_total)
const (
	twilioRejectNotConfigured = "not_configured"
	twilioRejectMissing       = "missing_signature"
	twilioRejectInvalid       = "invalid_signature"
	twilioRejectMalformed     = "malformed"
	twilioRejectTooLarge      = "too_large"
)

// mapTwilioStatus converts the MessageStatus of a twilio callback to the target status. An empty result keeps the
// current status: queued, accepted, scheduled, sending and every status that is not a delivery state of an outbound
// message change nothing.
func mapTwilioStatus(s string) target.Status {
	switch s {
	case "sent":
		return target.StatusSent
	case "delivered":
		return target.StatusDelivered
	case "failed", "undelivered", "canceled":
		return target.StatusFailed
	default:
		return ""
	}
}

// refuseTwilio counts and logs a refused callback and returns the error for it. Nothing of the request is logged.
func refuseTwilio(reason string) error {
	promTwilioCallbackRejectedTotal.WithLabelValues(reason).Inc()
	logrus.WithFields(logrus.Fields{
		"func":   "hookTwilio",
		"reason": reason,
	}).Warnf("A twilio callback was refused. reason: %s", reason)

	return fmt.Errorf("%w: %s", ErrHookUnauthorized, reason)
}

// verifyTwilioCallback checks the size and the signature of a callback and returns the parsed form. The form is only
// parsed to compute the signature (fail-closed): any refusal is returned as refuseTwilio does.
func (h *messageHandler) verifyTwilioCallback(ctx context.Context, data []byte, signature string) (url.Values, error) {
	if len(data) > twilioCallbackMaxBody {
		return nil, refuseTwilio(twilioRejectTooLarge)
	}

	form, err := url.ParseQuery(string(data))
	if err != nil {
		return nil, refuseTwilio(twilioRejectMalformed)
	}

	if h.reqExternal == nil {
		return nil, refuseTwilio(twilioRejectNotConfigured)
	}
	switch errValidate := h.reqExternal.TwilioValidateCallback(ctx, signature, form); {
	case errValidate == nil:
		return form, nil
	case errors.Is(errValidate, requestexternal.ErrTwilioSignatureMissing):
		return nil, refuseTwilio(twilioRejectMissing)
	case errors.Is(errValidate, requestexternal.ErrTwilioSignatureInvalid):
		return nil, refuseTwilio(twilioRejectInvalid)
	default:
		return nil, refuseTwilio(twilioRejectNotConfigured)
	}
}

// hookTwilio handles a twilio status callback.
//
// The request is verified first (fail-closed): the form is only parsed to compute the signature, and without a valid
// signature no value of the body is used and nothing is changed. A verified body that is not a status callback is ignored; a message is never created from a twilio
// callback. The message is found by the twilio message id that the dispatch stored in the target, and a target
// only moves forward, like the telnyx receipt.
func (h *messageHandler) hookTwilio(ctx context.Context, data []byte, signature string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "hookTwilio",
	})

	// the body carries the numbers, so it is never part of an error or a log line.
	form, err := h.verifyTwilioCallback(ctx, data, signature)
	if err != nil {
		return err
	}

	// only a status callback changes a target; an incoming message (no MessageStatus) is out of scope.
	status := form.Get("MessageStatus")
	if status == "" {
		log.Debugf("Ignoring the callback: it is not a status callback.")
		return nil
	}

	sid := form.Get("MessageSid")
	if !twilioSIDPattern.MatchString(sid) {
		log.Debugf("Ignoring the callback: the message id has not the shape of a twilio id.")
		return nil
	}

	newStatus := mapTwilioStatus(status)
	if newStatus == "" {
		log.Debugf("The status of the callback does not change the target. status: %s", status)
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, receiptBudget)
	defer cancel()

	since := h.utilHandler.TimeNow().Add(-receiptWindow)
	msgs, err := h.db.MessageListByProviderReference(ctx, sid, since)
	if err != nil {
		return fmt.Errorf("%w: could not look up the message: %w", ErrReceiptRetry, err)
	}

	var m *message.Message
	for _, c := range msgs {
		for _, ct := range c.Targets {
			if ct.ProviderReferenceID == sid {
				m = c
				break
			}
		}
		if m != nil {
			break
		}
	}
	if m == nil {
		promReceiptUnmatchedTotal.Inc()
		log.Debugf("No message matches the callback.")
		return nil
	}

	changed, err := h.updateTargets(ctx, m.ID, updateOpts{maxTries: callbackWriteTries, budget: receiptBudget}, func(targets []target.Target) error {
		for j := range targets {
			if targets[j].ProviderReferenceID != sid {
				continue
			}

			if statusRank(newStatus) <= statusRank(targets[j].Status) {
				return errNoChange
			}

			targets[j].Status = newStatus
			targets[j].TMUpdate = h.utilHandler.TimeNow()
			return nil
		}

		return errNoChange
	})
	if err != nil {
		return fmt.Errorf("%w: could not update the target: %w", ErrReceiptRetry, err)
	}
	if !changed {
		log.Debugf("The callback changes nothing. message_id: %s", m.ID)
		return nil
	}

	h.publishUpdated(ctx, m.ID)

	return nil
}
