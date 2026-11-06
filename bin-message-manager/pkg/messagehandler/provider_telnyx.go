package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_provider_sms.go -source provider_telnyx.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/requestexternal"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// SMSProvider sends an sms through one provider. Like MMSProvider it reports the result of every target by
// itself, in the order of the given targets.
type SMSProvider interface {
	SendSMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, onResult ResultCallback) []SendResult
}

type smsTelnyx struct {
	requestExternal requestexternal.RequestExternal
}

// NewSMSTelnyx returns the telnyx sms provider.
func NewSMSTelnyx(reqExternal requestexternal.RequestExternal) SMSProvider {
	return &smsTelnyx{requestExternal: reqExternal}
}

// SendSMS sends the sms to every target, one request per target.
func (h *smsTelnyx) SendSMS(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string, onResult ResultCallback) []SendResult {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SendSMS",
		"provider":   "telnyx",
		"message_id": messageID,
	})

	return sendTargets(ctx, len(targets), func(ctx context.Context, i int) SendResult {
		m, err := h.requestExternal.TelnyxSendMessage(ctx, source.Target, targets[i].Destination.Target, text)
		if err != nil {
			outcome := classifyError(err)
			log.Errorf("Could not send the sms to telnyx. target_index: %d, outcome: %d, err: %v", i, outcome, err)
			return SendResult{Outcome: outcome}
		}

		promTelnyxSendTotal.WithLabelValues("sms").Inc()
		res := SendResult{Outcome: SendOutcomeAccepted}
		if m != nil {
			res.ProviderReferenceID = m.Data.ID
			res.ProviderParts = m.Data.Parts
		}
		return res
	}, onResult)
}
