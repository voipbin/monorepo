package servicehandler

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	amagent "monorepo/bin-agent-manager/models/agent"
	smfile "monorepo/bin-storage-manager/models/file"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ServiceAgentFileCreate sends a request to storage-manager
// to creating a file.
// it returns created file info if it succeed.
func (h *serviceHandler) ServiceAgentFileCreate(ctx context.Context, a *amagent.Agent, f multipart.File, name string, detail string, filename string) (*smfile.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "ServiceAgentFileCreate",
		"agent":    a,
		"name":     name,
		"detail":   detail,
		"filename": filename,
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
	tmp, err := h.storageFileCreate(ctx, a.CustomerID, a.ID, smfile.ReferenceTypeNone, uuid.Nil, name, detail, filename, h.bucketName, filepath)
	if err != nil {
		log.Errorf("Could not create a file. err: %v", err)
		return nil, err
	}
	log.WithField("file", tmp).Debugf("Created file. file_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentFileGets gets the list of file of the given customer id.
// It returns list of files if it succeed.
func (h *serviceHandler) ServiceAgentFileGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*smfile.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "StorageFileGetsByOnwerID",
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
		"owner_id":    a.ID.String(),
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

// ServiceAgentFileGet gets the file of the given id.
// It returns file if it succeed.
func (h *serviceHandler) ServiceAgentFileGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*smfile.WebhookMessage, error) {
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

	if f.OwnerID != a.ID {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	res := f.ConvertWebhookMessage()
	return res, nil
}

// ServiceAgentFileDelete deletes the file of the given id.
func (h *serviceHandler) ServiceAgentFileDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*smfile.WebhookMessage, error) {
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

	if f.OwnerID != a.ID {
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

// convertFileFilters converts map[string]string to map[smfile.Field]any
func (h *serviceHandler) convertFileFilters(filters map[string]string) (map[smfile.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, smfile.File{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[smfile.Field]any, len(typed))
	for k, v := range typed {
		result[smfile.Field(k)] = v
	}

	return result, nil
}
