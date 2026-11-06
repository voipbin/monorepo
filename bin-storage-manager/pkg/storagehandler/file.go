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
	fileType file.Type,
	name string,
	detail string,
	filename string,
	bucketName string,
	filepath string,
) (*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "FileCreate",
		"customer_id":    customerID,
		"owner_id":       ownerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"file_type":      fileType,
		"name":           name,
		"detail":         detail,
		"filename":       filename,
		"bucket_name":    bucketName,
		"filepath":       filepath,
	})

	res, err := h.fileHandler.Create(ctx, customerID, ownerID, referenceType, referenceID, fileType, name, detail, filename, bucketName, filepath)
	if err != nil {
		log.Errorf("Could not create file. err: %v", err)
		return nil, err
	}

	return res, nil
}

// FileCreateContent creates a new file from the given bytes and returns created file info.
// The content is never logged.
func (h *storageHandler) FileCreateContent(
	ctx context.Context,
	customerID uuid.UUID,
	ownerID uuid.UUID,
	fileType file.Type,
	name string,
	detail string,
	filename string,
	contentType string,
	content []byte,
) (*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "FileCreateContent",
		"customer_id":  customerID,
		"file_type":    fileType,
		"content_size": len(content),
	})

	res, err := h.fileHandler.CreateFromContent(ctx, customerID, ownerID, fileType, name, detail, filename, contentType, content)
	if err != nil {
		log.Errorf("Could not create file from content. err: %v", err)
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

// FileList returns list of files info.
func (h *storageHandler) FileList(ctx context.Context, token string, size uint64, filters map[file.Field]any) ([]*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "FileGets",
		"token":   token,
		"size":    size,
		"filters": filters,
	})

	res, err := h.fileHandler.List(ctx, token, size, filters)
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
