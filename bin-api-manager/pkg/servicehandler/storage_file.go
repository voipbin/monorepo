package servicehandler

import (
	"context"
	"fmt"
	"io"
	multipart "mime/multipart"
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
		return nil, fmt.Errorf("could not find file info. err: %v", err)
	}
	log.WithField("file", f).Debugf("Found file info. file_id: %s", f.ID)

	if !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAll) {
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

// StorageFileCreate sends a request to storage-manager
// to creating a file.
// it returns created file info if it succeed.
func (h *serviceHandler) StorageFileCreate(ctx context.Context, a *amagent.Agent, f multipart.File, name string, detail string, filename string) (*smfile.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "StorageFileCreate",
		"agent":    a,
		"name":     name,
		"detail":   detail,
		"filename": filename,
	})
	log.Debug("Creating a new file.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// open file writer
	filepath := fmt.Sprintf("tmp/%s", h.utilHandler.UUIDCreate())
	log.Debugf("Filename: %s", filepath)
	wc := h.storageClient.Bucket(h.bucketName).Object(filepath).NewWriter(ctx)

	// upload the file
	w, err := io.Copy(wc, f)
	if err != nil {
		log.Errorf("Could not upload the file. err: %v", err)
		return nil, err
	}
	log.Debugf("Wrote file. count: %d", w)

	if err := wc.Close(); err != nil {
		log.Errorf("Could not close the file. err: %v", err)
		return nil, err
	}

	// create file
	// set timeout for 60 secs
	tmp, err := h.storageFileCreate(ctx, a.CustomerID, a.ID, smfile.ReferenceTypeNone, uuid.Nil, name, detail, filename, h.bucketName, filepath)
	if err != nil {
		log.Errorf("Could not create a file. err: %v", err)
		return nil, err
	}
	log.WithField("file", tmp).Debugf("Created file. file_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) storageFileCreate(
	ctx context.Context,
	customerID uuid.UUID,
	ownerID uuid.UUID,
	referenceType smfile.ReferenceType,
	referenceID uuid.UUID,
	name string,
	detail string,
	filename string,
	bucketName string,
	filepath string,
) (*smfile.File, error) {
	res, err := h.reqHandler.StorageV1FileCreate(ctx, customerID, ownerID, referenceType, referenceID, name, detail, filename, bucketName, filepath, 60000)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// StorageFileGets gets the list of file of the given customer id.
// It returns list of files if it succeed.
func (h *serviceHandler) StorageFileList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*smfile.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "StorageFileGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a file.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// filters
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	// get files
	// Convert string filters to typed filters
	typedFilters, err := h.convertFileFilters(filters)
	if err != nil {
		return nil, err
	}

	files, err := h.reqHandler.StorageV1FileList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get files info from the storage-manager. err: %v", err)
		return nil, fmt.Errorf("could not find files info. err: %v", err)
	}

	// create result
	res := []*smfile.WebhookMessage{}
	for _, f := range files {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

