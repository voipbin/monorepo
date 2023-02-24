package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	rmprovider "gitlab.com/voipbin/bin-manager/route-manager.git/models/provider"
)

// providerGet validates the provider's ownership and returns the provider info.
func (h *serviceHandler) providerGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmprovider.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "providerGet",
			"customer_id": u.ID,
			"provider_id": id,
		},
	)

	// send request
	tmp, err := h.reqHandler.RouteV1ProviderGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the provider info. err: %v", err)
		return nil, err
	}
	log.WithField("queue", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Info("The user has no permission for this providers.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ProviderGet sends a request to route-manager
// to getting the provider.
func (h *serviceHandler) ProviderGet(ctx context.Context, u *cscustomer.Customer, providerID uuid.UUID) (*rmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"provider_id": providerID,
	})

	res, err := h.providerGet(ctx, u, providerID)
	if err != nil {
		log.Errorf("Could not validate the provider info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ProviderGets sends a request to route-manager
// to getting a list of providers.
// it returns providers info if it succeed.
func (h *serviceHandler) ProviderGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*rmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderGets",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.GetCurTime()
	}

	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Info("The user has no permission for this provider.")
		return nil, fmt.Errorf("user has no permission")
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
	u *cscustomer.Customer,
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
		"customer_id": u.ID,
	})

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Errorf("The user has no permission for this number. customer_id: %s", u.ID)
		return nil, fmt.Errorf("user has no permission")
	}

	log.Debug("Creating a new provider.")
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
func (h *serviceHandler) ProviderDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmprovider.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ProviderDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"provider_id": id,
	})
	log.Debug("Deleting a outplan.")

	// get provider
	_, err := h.reqHandler.RouteV1ProviderGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get provider info from the route-manager. err: %v", err)
		return nil, fmt.Errorf("could not find provider info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Errorf("The customer has no permission for this provider. customer: %ss", u.ID)
		return nil, fmt.Errorf("customer has no permission")
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
	u *cscustomer.Customer,
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
		"customer_id": u.ID,
		"provider_id": providerID,
	})

	_, err := h.providerGet(ctx, u, providerID)
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
