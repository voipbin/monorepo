package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// accesskeyGet returns accesskey
func (h *serviceHandler) accesskeyGet(ctx context.Context, a *amagent.Agent, accesskeyID uuid.UUID) (*csaccesskey.Accesskey, error) {
	res, err := h.reqHandler.CustomerV1AccesskeyGet(ctx, accesskeyID)
	if err != nil {
		return nil, err
	}

	if res.CustomerID != a.CustomerID {
		return nil, fmt.Errorf("not found")
	}

	if res.TMDelete < defaultTimestamp {
		return nil, fmt.Errorf("deleted item")
	}

	return res, nil
}

// AccesskeyCreate sends a request to customer-manager
// to create a accesskey.
// it returns created accesskey info if it succeed.
func (h *serviceHandler) AccesskeyCreate(ctx context.Context, a *amagent.Agent, name string, detail string, expire int32) (*csaccesskey.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "AccesskeyCreate",
		"agent":  a,
		"name":   name,
		"detail": detail,
		"expire": expire,
	})
	log.Debug("Creating a new accesskey.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	if expire < 86400 {
		return nil, fmt.Errorf("wrong expiration")
	}

	tmp, err := h.reqHandler.CustomerV1AccesskeyCreate(ctx, a.CustomerID, name, detail, expire)
	if err != nil {
		log.Errorf("Could not create activeflow. erR: %v", err)
		return nil, err
	}
	log.WithField("accesskey", tmp).Debugf("Created accesskey. accesskey_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AccesskeyGet sends a request to customer-manager
// to getting a accesskey.
// it returns accesskey if it succeed.
func (h *serviceHandler) AccesskeyGet(ctx context.Context, a *amagent.Agent, accesskeyID uuid.UUID) (*csaccesskey.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "AccesskeyGet",
		"customer_id":  a.CustomerID,
		"accesskey_id": accesskeyID,
	})

	tmp, err := h.accesskeyGet(ctx, a, accesskeyID)
	if err != nil {
		log.Infof("Could not get accesskey info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AccesskeyRawGetByToken sends a request to customer-manager
// to getting a accesskey.
// it returns accesskey if it succeed.
func (h *serviceHandler) AccesskeyRawGetByToken(ctx context.Context, token string) (*csaccesskey.Accesskey, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "AccesskeyGetByToken",
	})

	// filters
	filters := map[csaccesskey.Field]any{
		csaccesskey.FieldToken:   token,
		csaccesskey.FieldDeleted: false,
	}

	tmps, err := h.reqHandler.CustomerV1AccesskeyList(ctx, "", 10, filters)
	if err != nil {
		log.Infof("Could not get accesskeys info. err: %v", err)
		return nil, err
	}

	if len(tmps) == 0 {
		return nil, fmt.Errorf("not found")
	}

	res := tmps[0]
	return &res, nil
}

// AccesskeyGets sends a request to customer-manager
// to getting a list of accesskeys.
// it returns list of accesskeys if it succeed.
func (h *serviceHandler) AccesskeyList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*csaccesskey.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AccesskeyGets",
		"customer_id": a.CustomerID,
		"agent_id":    a.CustomerID,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	// filters
	filters := map[csaccesskey.Field]any{
		csaccesskey.FieldCustomerID: a.CustomerID,
		csaccesskey.FieldDeleted:    false, // we don't need deleted items
	}

	tmps, err := h.reqHandler.CustomerV1AccesskeyList(ctx, token, size, filters)
	if err != nil {
		log.Infof("Could not get accesskeys info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*csaccesskey.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// AccesskeyDelete sends a request to customer-manager
// to delete the accesskey.
// it returns accesskey if it succeed.
func (h *serviceHandler) AccesskeyDelete(ctx context.Context, a *amagent.Agent, accesskeyID uuid.UUID) (*csaccesskey.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "AccesskeyDelete",
		"customer_id":  a.CustomerID,
		"agent_id":     a.ID,
		"accesskey_id": accesskeyID,
	})

	ak, err := h.accesskeyGet(ctx, a, accesskeyID)
	if err != nil {
		log.Infof("Could not get accesskey info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ak.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	tmp, err := h.reqHandler.CustomerV1AccesskeyDelete(ctx, accesskeyID)
	if err != nil {
		log.Infof("Could not delete accesskey info. err: %v", err)
		return nil, err
	}

	// convert
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// AccesskeyUpdate sends a request to customer-manager
// to update the accesskey info.
func (h *serviceHandler) AccesskeyUpdate(ctx context.Context, a *amagent.Agent, accesskeyID uuid.UUID, name string, detail string) (*csaccesskey.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "AccesskeyUpdate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	ak, err := h.accesskeyGet(ctx, a, accesskeyID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ak.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.CustomerV1AccesskeyUpdate(ctx, accesskeyID, name, detail)
	if err != nil {
		log.Infof("Could not delete the agent info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
