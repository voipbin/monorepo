package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	commonaddress "monorepo/bin-common-handler/models/address"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *serviceHandler) ServiceAgentExtensionGets(ctx context.Context, a *amagent.Agent) ([]*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentExtensionGets",
		"agent": a,
	})

	res := []*rmextension.WebhookMessage{}
	for _, address := range a.Addresses {
		if address.Type != commonaddress.TypeExtension {
			continue
		}

		extensionID := uuid.FromStringOrNil(address.Target)
		ext, err := h.extensionGet(ctx, extensionID)
		if err != nil {
			log.Errorf("Could not get extension info. err: %v", err)
			continue
		}

		res = append(res, ext.ConvertWebhookMessage())
	}

	return res, nil
}

func (h *serviceHandler) ServiceAgentExtensionGet(ctx context.Context, a *amagent.Agent, extensionID uuid.UUID) (*rmextension.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentExtensionGet",
		"agent": a,
	})

	tmp, err := h.extensionGet(ctx, extensionID)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, err
	}

	for _, address := range a.Addresses {
		if address.Type != commonaddress.TypeExtension {
			continue
		}

		tmpID := uuid.FromStringOrNil(address.Target)
		if tmp.ID == tmpID {
			res := tmp.ConvertWebhookMessage()
			return res, nil
		}
	}

	return nil, fmt.Errorf("could not find the extension")
}
