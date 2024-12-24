package servicehandler

import (
	"context"
	"fmt"
	amagent "monorepo/bin-agent-manager/models/agent"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *serviceHandler) storageFileGet(ctx context.Context, fileID uuid.UUID) (*smfile.File, error) {
	res, err := h.reqHandler.StorageV1FileGet(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if res.TMDelete < defaultTimestamp {
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

func (h *serviceHandler) StorageFileGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*smfile.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "StorageFileGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"file_id":     id,
	})
	log.Debug("Getting a file.")

	// get file
	f, err := h.storageFileGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get file info from the storage-manager. err: %v", err)
		return nil, fmt.Errorf("could not find file info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	res := f.ConvertWebhookMessage()
	return res, nil
}

// StorageFileDelete deletes the file of the given id.
func (h *serviceHandler) StorageFileDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*smfile.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "StorageFileDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"file_id":     id,
	})
	log.Debug("Deleting a file.")

	// get file
	f, err := h.storageFileGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get file info from the storage-manager. err: %v", err)
		return nil, fmt.Errorf("could not find file info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.storageFileDelete(ctx, id)
	if err != nil {
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) storageFileDelete(ctx context.Context, id uuid.UUID) (*smfile.File, error) {
	res, err := h.reqHandler.StorageV1FileDelete(ctx, id, 60000)
	if err != nil {
		return nil, err
	}

	return res, nil
}
