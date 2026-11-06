package messagehandler

import (
	"context"
	"regexp"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/requestexternal"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

type smsMessagebird struct {
	requestExternal requestexternal.RequestExternal
}

// NewSMSMessagebird returns the messagebird sms provider.
func NewSMSMessagebird(reqExternal requestexternal.RequestExternal) SMSProvider {
	return &smsMessagebird{requestExternal: reqExternal}
}

var nonDigit = regexp.MustCompile(`[^0-9]`)

func digitsOf(s string) string {
	return nonDigit.ReplaceAllString(s, "")
}

// SendSMS sends the sms with one request for all the targets. A request succeeds or fails as a whole, so every
// target gets the same outcome. The recipients of the answer carry the status and the segment count; a target that is
// missing in the answer was accepted with the request and stays queued. The callback is not used: there is no
// per target request, and the results are written by the final write of the dispatch.
func (h *smsMessagebird) SendSMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, _ ResultCallback) []SendResult {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SendSMS",
		"provider":   "messagebird",
		"message_id": messageID,
		"recipients": len(targets),
	})

	res := make([]SendResult, len(targets))

	receivers := make([]string, 0, len(targets))
	for _, t := range targets {
		receivers = append(receivers, t.Destination.Target)
	}

	m, err := h.requestExternal.MessagebirdSendMessage(ctx, source.Target, receivers, text)
	if err != nil {
		outcome := classifyError(err)
		log.Errorf("Could not send the sms to messagebird. outcome: %d, err: %v", outcome, err)
		for i := range res {
			res[i] = SendResult{Outcome: outcome}
		}
		return res
	}

	promMessagebirdSendTotal.WithLabelValues("sms").Add(float64(len(targets)))

	for i := range res {
		res[i] = SendResult{Outcome: SendOutcomeAccepted, ProviderReferenceID: m.ID}

		want := digitsOf(targets[i].Destination.Target)
		for _, item := range m.Recipients.Items {
			rt := item.ConvertTartget()
			if digitsOf(rt.Destination.Target) != want {
				continue
			}

			res[i].ProviderParts = item.MessagePartCount
			res[i].Status = rt.Status
			break
		}
	}

	return res
}
