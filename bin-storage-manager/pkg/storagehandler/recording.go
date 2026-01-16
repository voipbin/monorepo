package storagehandler

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-storage-manager/models/bucketfile"
	"monorepo/bin-storage-manager/models/file"
)

// RecordingGet returns given recording's bucketfile info.
func (h *storageHandler) RecordingGet(ctx context.Context, id uuid.UUID) (*bucketfile.BucketFile, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RecordingGet",
		"recoding_id": id,
	})

	filters := map[file.Field]any{
		file.FieldDeleted:     false,
		file.FieldReferenceID: id,
	}

	files, err := h.FileList(ctx, "", 100, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get files. reference_id: %s", id)
	}

	// create compress file
	bucketName, filepath, err := h.fileHandler.CompressCreate(ctx, files)
	if err != nil {
		return nil, errors.Wrapf(err, "could not compress the files. bucket_name: %s, filepath: %s", bucketName, filepath)
	}
	log.Debugf("Created compress file. bucket_name: %s, filepath: %s", bucketName, filepath)

	// get download uri
	bucketURI, downloadURI, err := h.fileHandler.DownloadURIGet(ctx, bucketName, filepath, time.Hour*24)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get download link. bucket_name: %s, filepath: %s", bucketName, filepath)
	}
	log.Debugf("Created download uri. len: %d", len(downloadURI))

	// create recording.Recording
	tmExpire := h.utilHandler.TimeGetCurTimeAdd(24 * time.Hour)
	res := &bucketfile.BucketFile{
		ReferenceType:    bucketfile.ReferenceTypeRecording,
		ReferenceID:      id,
		BucketURI:        bucketURI,
		DownloadURI:      downloadURI,
		TMDownloadExpire: tmExpire,
	}

	return res, nil
}

// RecordingDelete deletes the given recording file
func (h *storageHandler) RecordingDelete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RecordingDelete",
		"recoding_id": id,
	})

	filters := map[file.Field]any{
		file.FieldDeleted:     false,
		file.FieldReferenceID: id,
	}

	files, err := h.FileList(ctx, "", 100, filters)
	if err != nil {
		return errors.Wrapf(err, "could not get files. reference_id: %s", id)
	}

	for _, f := range files {
		tmp, err := h.fileHandler.Delete(ctx, f.ID)
		if err != nil {
			return errors.Wrapf(err, "could not delete the file. file_id: %s", f.ID)
		}
		log.WithField("file", tmp).Debugf("Deleted file. file_id: %s", f.ID)
	}

	return nil
}
