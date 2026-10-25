package servicehandler

import (
	"context"
	"regexp"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	nmnumber "monorepo/bin-number-manager/models/number"
	nmprovider "monorepo/bin-number-manager/models/provider"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// numberProviderNamePattern is the shape of a number provider name. It matches the route of
// number-manager, and keeps a request value from altering the rpc uri.
const numberProviderNamePattern = `^[a-z0-9_]{1,32}$`

var regNumberProviderName = regexp.MustCompile(numberProviderNamePattern)

// numberProviderAuthorize checks that the identity may manage the number providers:
// direct access is refused, and only the project super admin is allowed.
func (h *serviceHandler) numberProviderAuthorize(ctx context.Context, a *auth.AuthIdentity) error {
	if a.IsDirect() {
		return serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return serviceerrors.ErrPermissionDenied
	}

	return nil
}

// numberProviderAuthorizeName does the numberProviderAuthorize and validates the provider name.
func (h *serviceHandler) numberProviderAuthorizeName(ctx context.Context, a *auth.AuthIdentity, name string) error {
	if err := h.numberProviderAuthorize(ctx, a); err != nil {
		return err
	}

	if !regNumberProviderName.MatchString(name) {
		return serviceerrors.ErrInvalidArgument
	}

	return nil
}

// NumberProviderList returns the number providers.
// Only the project super admin is allowed.
func (h *serviceHandler) NumberProviderList(ctx context.Context, a *auth.AuthIdentity) ([]*nmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "NumberProviderList",
		"username": a.DisplayName(),
	})

	if err := h.numberProviderAuthorize(ctx, a); err != nil {
		log.Info("The request is not allowed.")
		return nil, err
	}

	tmps, err := h.reqHandler.NumberV1NumberProviderList(ctx)
	if err != nil {
		log.Errorf("Could not get the number providers. err: %v", err)
		return nil, err
	}

	res := make([]*nmprovider.WebhookMessage, len(tmps))
	for i := range tmps {
		res[i] = tmps[i].ConvertWebhookMessage()
	}

	return res, nil
}

// NumberProviderGet returns the given number provider.
// Only the project super admin is allowed.
func (h *serviceHandler) NumberProviderGet(ctx context.Context, a *auth.AuthIdentity, name string) (*nmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "NumberProviderGet",
		"username":      a.DisplayName(),
		"provider_name": name,
	})

	if err := h.numberProviderAuthorizeName(ctx, a, name); err != nil {
		log.Info("The request is not allowed.")
		return nil, err
	}

	tmp, err := h.reqHandler.NumberV1NumberProviderGet(ctx, nmnumber.ProviderName(name))
	if err != nil {
		log.Errorf("Could not get the number provider. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// NumberProviderUpdate updates the enabled, priority and config of the given number provider.
// A nil enabled or priority, and a nil config, are left unchanged.
// Only the project super admin is allowed.
func (h *serviceHandler) NumberProviderUpdate(ctx context.Context, a *auth.AuthIdentity, name string, enabled *bool, priority *int, config map[string]string) (*nmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "NumberProviderUpdate",
		"username":      a.DisplayName(),
		"provider_name": name,
	})

	if err := h.numberProviderAuthorizeName(ctx, a, name); err != nil {
		log.Info("The request is not allowed.")
		return nil, err
	}

	tmp, err := h.reqHandler.NumberV1NumberProviderUpdate(ctx, nmnumber.ProviderName(name), a.AgentID(), enabled, priority, config)
	if err != nil {
		log.Errorf("Could not update the number provider. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// NumberProviderUpdateSecret stores the secret of the given number provider.
// The secret is passed on in the request only. Nothing of it is logged.
// Only the project super admin is allowed.
func (h *serviceHandler) NumberProviderUpdateSecret(ctx context.Context, a *auth.AuthIdentity, name string, secret string) (*nmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "NumberProviderUpdateSecret",
		"username":      a.DisplayName(),
		"provider_name": name,
	})

	if err := h.numberProviderAuthorizeName(ctx, a, name); err != nil {
		log.Info("The request is not allowed.")
		return nil, err
	}

	tmp, err := h.reqHandler.NumberV1NumberProviderUpdateSecret(ctx, nmnumber.ProviderName(name), a.AgentID(), secret)
	if err != nil {
		log.Errorf("Could not store the number provider secret. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// NumberProviderDeleteSecret deletes the stored secret of the given number provider.
// Only the project super admin is allowed.
func (h *serviceHandler) NumberProviderDeleteSecret(ctx context.Context, a *auth.AuthIdentity, name string) (*nmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "NumberProviderDeleteSecret",
		"username":      a.DisplayName(),
		"provider_name": name,
	})

	if err := h.numberProviderAuthorizeName(ctx, a, name); err != nil {
		log.Info("The request is not allowed.")
		return nil, err
	}

	tmp, err := h.reqHandler.NumberV1NumberProviderDeleteSecret(ctx, nmnumber.ProviderName(name), a.AgentID())
	if err != nil {
		log.Errorf("Could not delete the number provider secret. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}
