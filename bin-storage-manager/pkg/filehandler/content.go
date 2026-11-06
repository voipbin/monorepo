package filehandler

import (
	"context"
	"fmt"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// fileContentMaxSize is the largest content CreateFromContent accepts.
//
// fileContentChunkSize is the writer chunk size: one chunk above the largest content, so every allowed
// content goes up in a single request and the writer does not allocate its 16 MiB default buffer.
// It must stay a multiple of 256 KiB (a requirement of the storage client library).
const (
	fileContentMaxSize   = 2 * 1024 * 1024
	fileContentChunkSize = fileContentMaxSize + 256*1024
)

// CreateFromContent creates a file from the given bytes and returns the created file.
// The bytes are written to the tmp bucket first and registered through Create, so the quota check,
// the move to the media bucket, the signed URL and the events are the same as for every other file.
func (h *fileHandler) CreateFromContent(
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
		"func":         "CreateFromContent",
		"customer_id":  customerID,
		"file_type":    fileType,
		"content_size": len(content),
	})

	if len(content) == 0 || len(content) > fileContentMaxSize {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameStorageManager,
			"FILE_CONTENT_INVALID",
			fmt.Sprintf("The file content must be between 1 and %d bytes.", fileContentMaxSize),
		)
	}

	tmpPath := fmt.Sprintf("%s/%s", bucketDirectoryTmp, h.utilHandler.UUIDCreate())
	if errWrite := h.bucketfileWrite(ctx, h.bucketTmp, tmpPath, contentType, content); errWrite != nil {
		return nil, errors.Wrapf(errWrite, "could not write the content")
	}

	res, err := h.Create(ctx, customerID, ownerID, file.ReferenceTypeNone, uuid.Nil, fileType, name, detail, filename, h.bucketTmp, tmpPath)
	if err != nil {
		// Create moves the object only after the checks, so this removes the leftover of a failure before the move.
		if errDelete := h.bucketfileDelete(ctx, h.bucketTmp, tmpPath); errDelete != nil {
			log.Debugf("Could not delete the temporary object. err: %v", errDelete)
		}
		return nil, err
	}

	return res, nil
}
