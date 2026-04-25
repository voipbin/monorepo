package servicehandler

import (
	"context"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	smaccount "monorepo/bin-storage-manager/models/account"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// storageAccountGet validates the storage account info.
func (h *serviceHandler) storageAccountGet(ctx context.Context, accountID uuid.UUID) (*smaccount.Account, error) {
	res, err := h.reqHandler.StorageV1AccountGet(ctx, accountID)
	if err != nil {
		return nil, err
	}

	if res.TMDelete != nil {
		return nil, serviceerrors.ErrNotFound
	}

	return res, nil
}

// StorageAccountGet sends a request to storage-manager
// to getting a storage account.
// it returns storage account if it succeed.
func (h *serviceHandler) StorageAccountGet(ctx context.Context, a *auth.AuthIdentity, storageAccountID uuid.UUID) (*smaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "StorageAccountGet",
		"customer_id":        a.CustomerID,
		"username":           a.DisplayName(),
		"storage_account_id": storageAccountID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get storage account
	sa, err := h.storageAccountGet(ctx, storageAccountID)
	if err != nil {
		log.Infof("Could not get storage account info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, sa.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	// convert
	res := sa.ConvertWebhookMessage()
	return res, nil
}

// StorageAccountGetByCustomerID sends a request to storage-manager
// to getting a storage account.
// it returns storage account if it succeed.
func (h *serviceHandler) StorageAccountGetByCustomerID(ctx context.Context, a *auth.AuthIdentity) (*smaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "StorageAccountGetByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false",
	}

	// get storage accounts
	// Convert string filters to typed filters
	typedFilters, err := h.convertAccountFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.StorageV1AccountList(ctx, "", 1, typedFilters)
	if err != nil || len(tmps) == 0 {
		log.Infof("Could not get storage account info. err: %v", err)
		return nil, err
	}

	res := tmps[0].ConvertWebhookMessage()

	if !h.hasPermission(ctx, a, res.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	return res, nil
}

// StorageAccountDelete sends a request to storage-manager
// to deleting a storage account.
// it returns storage account if it succeed.
func (h *serviceHandler) StorageAccountDelete(ctx context.Context, a *auth.AuthIdentity, storageAccountID uuid.UUID) (*smaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "StorageAccountDelete",
		"customer_id":        a.CustomerID,
		"username":           a.DisplayName(),
		"storage_account_id": storageAccountID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get storage account
	ba, err := h.storageAccountGet(ctx, storageAccountID)
	if err != nil {
		log.Infof("Could not get storage account info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ba.CustomerID, amagent.PermissionProjectSuperAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.StorageV1AccountDelete(ctx, storageAccountID, 60000)
	if err != nil {
		log.Errorf("Could not delete storage account. err: %v", err)
	}

	return res, nil
}

// StorageAccountGets sends a request to storage-manager
// to getting a list of storage accounts.
// it returns list of storage accounts if it succeed.
func (h *serviceHandler) StorageAccountList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*smaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "StorageAccountGets",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"size":        size,
		"token":       token,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionProjectSuperAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	// filters
	filters := map[string]string{
		"deleted": "false", // we don't need deleted items
	}

	// Convert string filters to typed filters
	typedFilters, err := h.convertAccountFilters(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, err
	}

	// get storage accounts
	tmps, err := h.reqHandler.StorageV1AccountList(ctx, token, size, typedFilters)
	if err != nil {
		log.Infof("Could not get storage account info. err: %v", err)
		return nil, err
	}

	res := make([]*smaccount.Account, len(tmps))
	for i := range tmps {
		res[i] = &tmps[i]
	}

	return res, nil
}

// StorageAccountCreate sends a request to storage-manager
// to create a new storage accounts.
// it returns created storage account if it succeed.
func (h *serviceHandler) StorageAccountCreate(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID) (*smaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "StorageAccountCreate",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionProjectSuperAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	// create storage accounts
	res, err := h.reqHandler.StorageV1AccountCreate(ctx, a.CustomerID)
	if err != nil {
		log.Infof("Could not get storage account info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// convertAccountFilters converts map[string]string to map[smaccount.Field]any
func (h *serviceHandler) convertAccountFilters(filters map[string]string) (map[smaccount.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, smaccount.Account{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[smaccount.Field]any, len(typed))
	for k, v := range typed {
		result[smaccount.Field(k)] = v
	}

	return result, nil
}
