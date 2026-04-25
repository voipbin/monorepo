package servicehandler

import (
	"context"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// accesskeyGet returns accesskey
func (h *serviceHandler) accesskeyGet(ctx context.Context, a *auth.AuthIdentity, accesskeyID uuid.UUID) (*csaccesskey.Accesskey, error) {
	res, err := h.reqHandler.CustomerV1AccesskeyGet(ctx, accesskeyID)
	if err != nil {
		return nil, err
	}

	if res.CustomerID != a.CustomerID {
		return nil, serviceerrors.ErrNotFound
	}

	if res.TMDelete != nil {
		return nil, fmt.Errorf("deleted item")
	}

	return res, nil
}

// AccesskeyCreate sends a request to customer-manager
// to create a accesskey.
// it returns created accesskey info if it succeed.
func (h *serviceHandler) AccesskeyCreate(ctx context.Context, a *auth.AuthIdentity, name string, detail string, expire int32) (*csaccesskey.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":   "AccesskeyCreate",
		"auth":   a.DisplayName(),
		"name":   name,
		"detail": detail,
		"expire": expire,
	})
	log.Debug("Creating a new accesskey.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
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
func (h *serviceHandler) AccesskeyGet(ctx context.Context, a *auth.AuthIdentity, accesskeyID uuid.UUID) (*csaccesskey.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

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
		return nil, serviceerrors.ErrPermissionDenied
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

	// Hash the token before lookup
	tokenHash := h.utilHandler.HashSHA256Hex(token)

	// filters
	filters := map[csaccesskey.Field]any{
		csaccesskey.FieldTokenHash: tokenHash,
		csaccesskey.FieldDeleted:   false,
	}

	tmps, err := h.reqHandler.CustomerV1AccesskeyList(ctx, "", 10, filters)
	if err != nil {
		log.Infof("Could not get accesskeys info. err: %v", err)
		return nil, err
	}

	if len(tmps) == 0 {
		return nil, serviceerrors.ErrNotFound
	}
	if len(tmps) > 1 {
		log.Errorf("Multiple accesskeys found for token hash, expected exactly one")
		return nil, fmt.Errorf("ambiguous token")
	}

	res := tmps[0]
	return &res, nil
}

// AccesskeyGets sends a request to customer-manager
// to getting a list of accesskeys.
// it returns list of accesskeys if it succeed.
func (h *serviceHandler) AccesskeyList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*csaccesskey.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "AccesskeyGets",
		"customer_id": a.CustomerID,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
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
func (h *serviceHandler) AccesskeyDelete(ctx context.Context, a *auth.AuthIdentity, accesskeyID uuid.UUID) (*csaccesskey.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":         "AccesskeyDelete",
		"customer_id":  a.CustomerID,
		"auth":         a.DisplayName(),
		"accesskey_id": accesskeyID,
	})

	ak, err := h.accesskeyGet(ctx, a, accesskeyID)
	if err != nil {
		log.Infof("Could not get accesskey info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ak.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
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
func (h *serviceHandler) AccesskeyUpdate(ctx context.Context, a *auth.AuthIdentity, accesskeyID uuid.UUID, name string, detail string) (*csaccesskey.WebhookMessage, error) {
	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "AccesskeyUpdate",
		"customer_id": a.CustomerID,
		"auth":        a.DisplayName(),
	})

	ak, err := h.accesskeyGet(ctx, a, accesskeyID)
	if err != nil {
		log.Errorf("Could not validate the agent info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ak.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.CustomerV1AccesskeyUpdate(ctx, accesskeyID, name, detail)
	if err != nil {
		log.Infof("Could not delete the agent info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
