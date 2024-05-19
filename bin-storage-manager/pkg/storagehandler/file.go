package storagehandler

import (
	"context"
	"monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// FileCreate creates a new file and returns created file info.
func (h *storageHandler) FileCreate(
	ctx context.Context,
	customerID uuid.UUID,
	ownerID uuid.UUID,
	referenceType file.ReferenceType,
	referenceID uuid.UUID,
	name string,
	detail string,
	bucketName string,
	filepath string,
) (*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "FileCreate",
		"customer_id":    customerID,
		"owner_id":       ownerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"name":           name,
		"detail":         detail,
		"bucket_name":    bucketName,
		"filepath":       filepath,
	})

	res, err := h.fileHandler.Create(ctx, customerID, ownerID, referenceType, referenceID, name, detail, bucketName, filepath)
	if err != nil {
		log.Errorf("Could not create file. err: %v", err)
		return nil, err
	}

	return res, nil
}

// FileGet returns given file info.
func (h *storageHandler) FileGet(ctx context.Context, id uuid.UUID) (*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "FileGet",
		"id":   id,
	})

	res, err := h.fileHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get file. err: %v", err)
		return nil, err
	}

	return res, nil
}

// FileGets returns list of files info.
func (h *storageHandler) FileGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "FileGets",
		"token":   token,
		"size":    size,
		"filters": filters,
	})

	res, err := h.fileHandler.Gets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get file. err: %v", err)
		return nil, err
	}

	return res, nil
}

// FileDelete deletes file and returns the deleted file info
func (h *storageHandler) FileDelete(ctx context.Context, id uuid.UUID) (*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "FileDelete",
		"id":   id,
	})

	res, err := h.fileHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete file. err: %v", err)
		return nil, err
	}

	return res, nil
}
