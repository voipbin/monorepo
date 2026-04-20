package servicehandler

import (
	"context"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	rmprovider "monorepo/bin-route-manager/models/provider"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// providerGet validates the provider's ownership and returns the provider info.
func (h *serviceHandler) providerGet(ctx context.Context, id uuid.UUID) (*rmprovider.Provider, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "providerGet",
		"provider_id": id,
	})

	// send request
	res, err := h.reqHandler.RouteV1ProviderGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the provider info. err: %v", err)
		return nil, err
	}
	log.WithField("provider", res).Debug("Received result.")

	// create result
	return res, nil
}

// ProviderGet sends a request to route-manager
// to getting the provider.
func (h *serviceHandler) ProviderGet(ctx context.Context, a *auth.AuthIdentity, providerID uuid.UUID) (*rmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderGet",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"provider_id": providerID,
	})

	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	tmp, err := h.providerGet(ctx, providerID)
	if err != nil {
		log.Errorf("Could not validate the provider info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	return tmp.ConvertWebhookMessage(), nil
}

// ProviderList sends a request to route-manager
// to getting a list of providers.
// it returns providers info if it succeed.
func (h *serviceHandler) ProviderList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*rmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderList",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"size":        size,
		"token":       token,
	})

	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// permission check
	// only project admin allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmps, err := h.reqHandler.RouteV1ProviderList(ctx, token, size)
	if err != nil {
		log.Errorf("Could not get providers from the route-manager. err: %v", err)
		return nil, err
	}

	res := make([]*rmprovider.WebhookMessage, len(tmps))
	for i := range tmps {
		res[i] = tmps[i].ConvertWebhookMessage()
	}

	return res, nil
}

// ProviderCreate is a service handler for provider creation.
func (h *serviceHandler) ProviderCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	providerType rmprovider.Type,
	hostname string,
	techPrefix string,
	techPostfix string,
	techHeaders map[string]string,
	name string,
	detail string,
) (*rmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderCreate",
		"customer_id": a.CustomerID,
	})
	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	log.Debug("Creating a new provider.")

	// permission check
	// only project admin allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.RouteV1ProviderCreate(
		ctx,
		providerType,
		hostname,
		techPrefix,
		techPostfix,
		techHeaders,
		name,
		detail,
	)
	if err != nil {
		log.Errorf("Could not create a new provider. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// ProviderDelete deletes the provider.
func (h *serviceHandler) ProviderDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderDelete",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"provider_id": id,
	})
	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	log.Debug("Deleting a provider.")

	// permission check
	// only project admin allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get provider
	_, err := h.reqHandler.RouteV1ProviderGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get provider info from the route-manager. err: %v", err)
		return nil, fmt.Errorf("could not find provider info. err: %v", err)
	}

	tmp, err := h.reqHandler.RouteV1ProviderDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the provider. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// ProviderUpdate sends a request to route-manager
// to updating the provider.
// it returns error if it failed.
func (h *serviceHandler) ProviderUpdate(
	ctx context.Context,
	a *auth.AuthIdentity,
	providerID uuid.UUID,
	providerType rmprovider.Type,
	hostname string,
	techPrefix string,
	techPostfix string,
	techHeaders map[string]string,
	name string,
	detail string,
) (*rmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderUpdate",
		"customer_id": a.CustomerID,
		"provider_id": providerID,
	})

	if a.IsDirect() {
		return nil, fmt.Errorf("direct access not supported")
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	_, err := h.providerGet(ctx, providerID)
	if err != nil {
		log.Errorf("Could not get provider. err: %v", err)
		return nil, err
	}

	tmp, err := h.reqHandler.RouteV1ProviderUpdate(
		ctx,
		providerID,
		providerType,
		hostname,
		techPrefix,
		techPostfix,
		techHeaders,
		name,
		detail,
	)
	if err != nil {
		log.Errorf("Could not update the provider. err: %v", err)
		return nil, err
	}
	log.WithField("provider", tmp).Debugf("Updated provider. provider_id: %s", tmp.ID)

	return tmp.ConvertWebhookMessage(), nil
}
