package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
)

// domainGet validates the domain's ownership and returns the domain info.
func (h *serviceHandler) domainGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmdomain.Domain, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "domainGet",
		"customer_id": a.CustomerID,
		"domain_id":   id,
	})

	// send request
	res, err := h.reqHandler.RegistrarV1DomainGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the domain info. err: %v", err)
		return nil, err
	}
	log.WithField("domain", res).Debug("Received result.")

	return res, nil
}

// DomainCreate is a service handler for flow creation.
func (h *serviceHandler) DomainCreate(ctx context.Context, a *amagent.Agent, domainName, name, detail string) (*rmdomain.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": a.CustomerID,
		"domain_name": domainName,
		"name":        name,
	})
	log.Debug("Creating a new domain.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.RegistrarV1DomainCreate(ctx, a.CustomerID, domainName, name, detail)
	if err != nil {
		log.Errorf("Could not create a new domain. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// DomainDelete deletes the domain of the given id.
func (h *serviceHandler) DomainDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmdomain.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"domain_id":   id,
	})
	log.Debug("Deleting the domain.")

	d, err := h.domainGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get the domain info. err: %v", err)
		return nil, fmt.Errorf("could not get domain info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, d.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// delete
	tmp, err := h.reqHandler.RegistrarV1DomainDelete(ctx, id)
	if err != nil {
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// DomainGet gets the domain of the given id.
// It returns domain if it succeed.
func (h *serviceHandler) DomainGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmdomain.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"domain_id":   id,
	})
	log.Debug("Getting a domain.")

	// get domain
	tmp, err := h.domainGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get domain info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not get domain info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// DomainGets gets the list of domains of the given customer id.
// It returns list of domains if it succeed.
func (h *serviceHandler) DomainGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmdomain.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"fucn":        "DomainGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a domains.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get tmps
	tmps, err := h.reqHandler.RegistrarV1DomainGets(ctx, a.CustomerID, token, size)
	if err != nil {
		log.Errorf("Could not get domains info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find domains info. err: %v", err)
	}

	// create result
	res := []*rmdomain.WebhookMessage{}
	for _, d := range tmps {
		tmp := d.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// DomainUpdate updates the flow info.
// It returns updated domain if it succeed.
func (h *serviceHandler) DomainUpdate(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail string) (*rmdomain.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"domain_id":   id,
	})
	log.Debug("Updating a domain.")

	// get
	d, err := h.domainGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get domain info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find domain info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, d.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// update
	tmp, err := h.reqHandler.RegistrarV1DomainUpdate(ctx, id, name, detail)
	if err != nil {
		logrus.Errorf("Could not update the domain. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
