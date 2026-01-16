package servicehandler

import (
	"context"
	"fmt"

	rmsipauth "monorepo/bin-registrar-manager/models/sipauth"
	rmtrunk "monorepo/bin-registrar-manager/models/trunk"

	amagent "monorepo/bin-agent-manager/models/agent"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// trunkGet validates the trunk's ownership and returns the trunk info.
func (h *serviceHandler) trunkGet(ctx context.Context, id uuid.UUID) (*rmtrunk.Trunk, error) {
	res, err := h.reqHandler.RegistrarV1TrunkGet(ctx, id)
	if err != nil {
		return nil, err
	}

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

	t, err := h.trunkGet(ctx, id)
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
	tmp, err := h.trunkGet(ctx, id)
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
func (h *serviceHandler) TrunkList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmtrunk.WebhookMessage, error) {
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
	// Convert string filters to typed filters
	typedFilters, err := h.convertTrunkFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.RegistrarV1TrunkList(ctx, token, size, typedFilters)
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
	t, err := h.trunkGet(ctx, id)
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

// convertTrunkFilters converts map[string]string to map[rmtrunk.Field]any
func (h *serviceHandler) convertTrunkFilters(filters map[string]string) (map[rmtrunk.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, rmtrunk.Trunk{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[rmtrunk.Field]any, len(typed))
	for k, v := range typed {
		result[rmtrunk.Field(k)] = v
	}

	return result, nil
}
