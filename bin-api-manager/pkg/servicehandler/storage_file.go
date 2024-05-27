package servicehandler

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	amagent "monorepo/bin-agent-manager/models/agent"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// storageFileGet returns the file info.
func (h *serviceHandler) storageFileGet(ctx context.Context, fileID uuid.UUID) (*smfile.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "storageFileGet",
		"file_id": fileID,
	})

	// send request
	res, err := h.reqHandler.StorageV1FileGet(ctx, fileID)
	if err != nil {
		log.Errorf("Could not get the storage file info. err: %v", err)
		return nil, err
	}
	log.WithField("file", res).Debugf("Received result. file_id: %s", res.ID)

	if res.TMDelete < defaultTimestamp {
		log.Debugf("Deleted storage file. file_id: %s", res.ID)
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

// StorageFileCreate sends a request to storage-manager
// to creating a file.
// it returns created file info if it succeed.
func (h *serviceHandler) StorageFileCreate(ctx context.Context, a *amagent.Agent, f multipart.File, name string, detail string, filename string) (*smfile.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "StorageFileCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"name":        name,
		"detail":      detail,
		"filename":    filename,
	})
	log.Debug("Creating a new file.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
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
	tmp, err := h.reqHandler.StorageV1FileCreate(ctx, a.CustomerID, a.ID, smfile.ReferenceTypeNone, uuid.Nil, name, detail, filename, h.bucketName, filepath, 60000)
	if err != nil {
		log.Errorf("Could not create a file. err: %v", err)
		return nil, err
	}
	log.WithField("file", tmp).Debugf("Created file. file_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// StorageFileGetsByOnwerID gets the list of file of the given customer id.
// It returns list of files if it succeed.
func (h *serviceHandler) StorageFileGetsByOnwerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*smfile.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "StorageFileGetsByOnwerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a file.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// filters
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
		"owner_id":    a.ID.String(),
	}

	// get files
	files, err := h.reqHandler.StorageV1FileGets(ctx, token, size, filters)
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

// StorageFileGet gets the file of the given id.
// It returns file if it succeed.
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

	if f.OwnerID != a.ID && !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
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

	if f.OwnerID != a.ID && !h.hasPermission(ctx, a, f.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.StorageV1FileDelete(ctx, id, 60000)
	if err != nil {
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
