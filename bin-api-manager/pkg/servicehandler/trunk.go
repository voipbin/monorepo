package servicehandler

import (
	"context"
	"fmt"

	rmsipauth "monorepo/bin-registrar-manager/models/sipauth"
	rmtrunk "monorepo/bin-registrar-manager/models/trunk"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// trunkGet validates the trunk's ownership and returns the trunk info.
func (h *serviceHandler) trunkGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmtrunk.Trunk, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "trunkGet",
		"customer_id": a.CustomerID,
		"domain_id":   id,
	})

	// send request
	res, err := h.reqHandler.RegistrarV1TrunkGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the trunk info. err: %v", err)
		return nil, err
	}
	log.WithField("trunk", res).Debug("Received result.")

	return res, nil
}

// TrunkCreate is a service handler for trunk creation.
func (h *serviceHandler) TrunkCreate(ctx context.Context, a *amagent.Agent, name string, detail string, domainName string, authTypes []rmsipauth.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TrunkCreate",
		"customer_id": a.CustomerID,
		"domain_name": domainName,
		"name":        name,
	})
	log.Debug("Creating a new trunk.")

	// permission check
	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.RegistrarV1TrunkCreate(ctx, a.CustomerID, name, detail, domainName, authTypes, username, password, allowedIPs)
	if err != nil {
		log.Errorf("Could not create a new trunk. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TrunkDelete deletes the trunk of the given id.
func (h *serviceHandler) TrunkDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmtrunk.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TrunkDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"trunk_id":    id,
	})
	log.Debug("Deleting the domain.")

	t, err := h.trunkGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get the domain info. err: %v", err)
		return nil, fmt.Errorf("could not get domain info. err: %v", err)
	}

	// permission check
	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// delete
	tmp, err := h.reqHandler.RegistrarV1TrunkDelete(ctx, id)
	if err != nil {
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TrunkGet gets the trunk of the given id.
// It returns trunk if it succeed.
func (h *serviceHandler) TrunkGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmtrunk.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TrunkGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"domain_id":   id,
	})
	log.Debug("Getting a trunk.")

	// get trunk
	tmp, err := h.trunkGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get trunk info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not get trunk info. err: %v", err)
	}

	// permission check
	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TrunkGets gets the list of trunks of the given customer id.
// It returns list of trunks if it succeed.
func (h *serviceHandler) TrunkGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmtrunk.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"fucn":        "TrunkGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a trunks.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// permission check
	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	filters := map[string]string{
		"deleted":     "false",
		"customer_id": a.CustomerID.String(),
	}

	// get tmps
	tmps, err := h.reqHandler.RegistrarV1TrunkGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get trunks info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find trunks info. err: %v", err)
	}

	// create result
	res := []*rmtrunk.WebhookMessage{}
	for _, d := range tmps {
		tmp := d.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// TrunkUpdateBasicInfo updates the trunk info.
// It returns updated trunk if it succeed.
func (h *serviceHandler) TrunkUpdateBasicInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, name string, detail string, authTypes []rmsipauth.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TrunkUpdateBasicInfo",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"trunk_id":    id,
	})
	log.Debug("Updating a trunk.")

	// get
	t, err := h.trunkGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get trunk info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find domain info. err: %v", err)
	}

	// permission check
	if !h.hasPermission(ctx, a, t.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// update
	tmp, err := h.reqHandler.RegistrarV1TrunkUpdateBasicInfo(ctx, id, name, detail, authTypes, username, password, allowedIPs)
	if err != nil {
		logrus.Errorf("Could not update the trunk. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
