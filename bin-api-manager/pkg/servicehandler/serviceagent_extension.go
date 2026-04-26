package servicehandler

import (
	"context"
	"fmt"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonaddress "monorepo/bin-common-handler/models/address"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *serviceHandler) ServiceAgentExtensionList(ctx context.Context, a *auth.AuthIdentity) ([]*rmextension.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentExtensionGets",
		"agent": a,
	})

	res := []*rmextension.WebhookMessage{}
	for _, address := range a.Agent.Addresses {
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

func (h *serviceHandler) ServiceAgentExtensionGet(ctx context.Context, a *auth.AuthIdentity, extensionID uuid.UUID) (*rmextension.WebhookMessage, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":  "ServiceAgentExtensionGet",
		"agent": a,
	})

	tmp, err := h.extensionGet(ctx, extensionID)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, err
	}

	for _, address := range a.Agent.Addresses {
		if address.Type != commonaddress.TypeExtension {
			continue
		}

		tmpID := uuid.FromStringOrNil(address.Target)
		if tmp.ID == tmpID {
			res := tmp.ConvertWebhookMessage()
			return res, nil
		}
	}

	return nil, fmt.Errorf("%w: extension not found", serviceerrors.ErrNotFound)
}
