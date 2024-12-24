package servicehandler

import (
	"context"
	"fmt"

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
	log.WithField("queue", res).Debug("Received result.")

	// create result
	return res, nil
}

// ProviderGet sends a request to route-manager
// to getting the provider.
func (h *serviceHandler) ProviderGet(ctx context.Context, a *amagent.Agent, providerID uuid.UUID) (*rmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"provider_id": providerID,
	})

	tmp, err := h.providerGet(ctx, providerID)
	if err != nil {
		log.Errorf("Could not validate the provider info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ProviderGets sends a request to route-manager
// to getting a list of providers.
// it returns providers info if it succeed.
func (h *serviceHandler) ProviderGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// permission check
	// only project admin allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmps, err := h.reqHandler.RouteV1ProviderGets(ctx, token, size)
	if err != nil {
		log.Errorf("Could not get queues from the route-manager. err: %v", err)
		return nil, err
	}

	res := []*rmprovider.WebhookMessage{}
	for _, tmp := range tmps {
		e := tmp.ConvertWebhookMessage()
		res = append(res, e)
	}

	return res, nil
}

// ProviderCreate is a service handler for provider creation.
func (h *serviceHandler) ProviderCreate(
	ctx context.Context,
	a *amagent.Agent,
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
		log.Errorf("Could not create a new flow. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ProviderDelete deletes the provider.
func (h *serviceHandler) ProviderDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"provider_id": id,
	})
	log.Debug("Deleting a outplan.")

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

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ProviderUpdate sends a request to route-manager
// to updating the route.
// it returns error if it failed.
func (h *serviceHandler) ProviderUpdate(
	ctx context.Context,
	a *amagent.Agent,
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
		log.Errorf("Could not update the queue. err: %v", err)
		return nil, err
	}
	log.WithField("queue", tmp).Debugf("Updated queue. queue_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
