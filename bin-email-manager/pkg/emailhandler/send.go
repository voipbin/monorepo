package emailhandler

import (
	"context"
	"monorepo/bin-email-manager/models/email"

	"github.com/sirupsen/logrus"
)

func (h *emailHandler) Send(ctx context.Context, e *email.Email) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Send",
		"email_id": e.ID,
	})

	// mailgun is the primary provider, sendgrid is the fallback.
	providers := []struct {
		name         string
		providerType email.ProviderType
		handler      func(context.Context, *email.Email) (string, error)
	}{
		{"mailgun", email.ProviderTypeMailgun, h.engineMailgun.Send},
		{"sendgrid", email.ProviderTypeSendgrid, h.engineSendgrid.Send},
	}

	for _, p := range providers {
		log.WithField("provider", p.name).Debugf("Attempting to send email via provider. provider: %s", p.name)

		providerReferenceID, err := p.handler(ctx, e)
		if err != nil {
			log.Errorf("Could not send email. Trying with next provider. err: %v", err)
			continue
		}
		log.Infof("Email successfully sent via provider. provider_name: %s, provider_reference_id: %s", p.name, providerReferenceID)

		if errUpdate := h.UpdateProvider(ctx, e.ID, p.providerType, providerReferenceID); errUpdate != nil {
			// we could not update the provider info
			// but just log it and return
			log.Errorf("could not update provider info. err: %v", errUpdate)
		}
		return
	}

	log.Errorf("all email providers failed")
}
