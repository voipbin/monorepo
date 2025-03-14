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

	handlers := []func(context.Context, *email.Email) (string, error){
		h.engineSendgrid.Send,
	}

	for _, handler := range handlers {
		providerReferenceID, err := handler(ctx, e)
		if err != nil {
			log.Errorf("could not send email. Trying with next provider. err: %v", err)
			continue
		}

		if errUpdate := h.UpdateProviderReferenceID(ctx, e.ID, providerReferenceID); errUpdate != nil {
			// we could not update the provider reference id
			// but just log it and return
			log.Errorf("could not update provider reference id. err: %v", errUpdate)
			return
		}
	}

	log.Errorf("all email providers failed")
}
