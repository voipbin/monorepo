package messagehandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

type smsTwilio struct {
	requestExternal requestexternal.RequestExternal
}

// NewSMSTwilio returns the twilio sms provider.
func NewSMSTwilio(reqExternal requestexternal.RequestExternal) SMSProvider {
	return &smsTwilio{requestExternal: reqExternal}
}

// SendSMS sends the sms to every target, one request per target.
func (h *smsTwilio) SendSMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, onResult ResultCallback) []SendResult {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SendSMS",
		"provider":   "twilio",
		"message_id": messageID,
	})

	return sendTargets(ctx, len(targets), func(ctx context.Context, i int) SendResult {
		m, err := h.requestExternal.TwilioSendSMS(ctx, source.Target, targets[i].Destination.Target, text)
		if err != nil {
			outcome := classifyError(err)
			log.Errorf("Could not send the sms to twilio. target_index: %d, outcome: %d, err: %v", i, outcome, err)
			return SendResult{Outcome: outcome}
		}

		promTwilioSendTotal.WithLabelValues("sms").Inc()
		res := SendResult{Outcome: SendOutcomeAccepted}
		if m != nil {
			res.ProviderReferenceID = m.SID
			res.ProviderParts = m.Segments()
		}
		return res
	}, onResult)
}

type mmsTwilio struct {
	requestExternal requestexternal.RequestExternal
}

// NewMMSTwilio returns the twilio mms provider.
func NewMMSTwilio(reqExternal requestexternal.RequestExternal) MMSProvider {
	return &mmsTwilio{requestExternal: reqExternal}
}

// SendMMS sends the mms to every target, one request per target.
func (h *mmsTwilio) SendMMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, mediaURLs []string, onResult ResultCallback) []SendResult {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SendMMS",
		"provider":   "twilio",
		"message_id": messageID,
	})

	return sendTargets(ctx, len(targets), func(ctx context.Context, i int) SendResult {
		m, err := h.requestExternal.TwilioSendMMS(ctx, source.Target, targets[i].Destination.Target, text, mediaURLs)
		if err == nil {
			promTwilioSendTotal.WithLabelValues("mms").Inc()
			res := SendResult{Outcome: SendOutcomeAccepted}
			if m != nil {
				res.ProviderReferenceID = m.SID
				res.ProviderParts = m.Segments()
			}
			return res
		}

		outcome := classifyError(err)
		switch {
		case outcome == SendOutcomeRejected:
			log.Errorf("The provider refused the mms. target_index: %d, err: %v", i, err)
		case requestexternal.IsTokenUnavailable(err):
			log.Errorf("The mms was not sent: the provider credentials could not be read. target_index: %d, err: %v", i, err)
		default:
			log.Errorf("Could not send the mms. The result is not known. target_index: %d, err: %v", i, err)
		}
		return SendResult{Outcome: outcome}
	}, onResult)
}
