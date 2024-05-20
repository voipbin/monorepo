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

// FileCreate sends a request to storage-manager
// to creating a file.
// it returns created file info if it succeed.
func (h *serviceHandler) FileCreate(ctx context.Context, a *amagent.Agent, f multipart.File, name string, detail string) (*smfile.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "FileCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"name":        name,
		"detail":      detail,
	})
	log.Debug("Creating a new file.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		return nil, fmt.Errorf("user has no permission")
	}

	// open file writer
	filepath := fmt.Sprintf("tmp/%s", h.utilHandler.UUIDCreate())
	wc := h.storageClient.Bucket(h.bucketName).Object(filepath).NewWriter(ctx)

	// upload the file
	if _, err := io.Copy(wc, f); err != nil {
		log.Errorf("Could not upload the file. err: %v", err)
		return nil, err
	}
	if err := wc.Close(); err != nil {
		log.Errorf("Could not close the file. err: %v", err)
		return nil, err
	}

	// create file
	tmp, err := h.reqHandler.StorageV1FileCreate(ctx, a.CustomerID, a.ID, smfile.ReferenceTypeNone, uuid.Nil, name, detail, h.bucketName, filepath)
	if err != nil {
		log.Errorf("Could not create a file. err: %v", err)
		return nil, err
	}
	log.WithField("file", tmp).Debugf("Created file. file_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
