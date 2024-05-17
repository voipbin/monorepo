package storagehandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-storage-manager/models/bucketfile"
)

// RecordingGet returns given recording's bucketfile info.
func (h *storageHandler) RecordingGet(ctx context.Context, id uuid.UUID) (*bucketfile.BucketFile, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RecordingGet",
		"recoding_id": id,
	})

	// get recording
	r, err := h.reqHandler.CallV1RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get recording info from the call-manager. err: %v", err)
		return nil, err
	}

	// compose files
	targetpaths := []string{}
	for _, filename := range r.Filenames {
		tmp := directoryRecording + "/" + filename
		targetpaths = append(targetpaths, tmp)
	}

	// create compress file
	bucketName, filepath, err := h.fileHandler.CompressCreate(ctx, h.bucketNameMedia, targetpaths)
	if err != nil {
		log.Errorf("Could not compress the files. err: %v", err)
		return nil, err
	}

	// get download uri
	bucketURI, downloadURI, err := h.fileHandler.DownloadURIGet(ctx, bucketName, filepath, time.Hour*24)
	if err != nil {
		log.Errorf("Could not get download link. err: %v", err)
		return nil, err
	}

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
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "RecordingDelete",
			"recoding_id": id,
		},
	)

	// get recording
	r, err := h.reqHandler.CallV1RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get recording info from the call-manager. err: %v", err)
		return err
	}

	for _, filename := range r.Filenames {
		filepath := fmt.Sprintf("recording/%s", filename)
		log.Debugf("Deleting recording file. bucket_name: %s, filepath: %s", h.bucketNameMedia, filepath)
		if !h.fileHandler.IsExist(ctx, h.bucketNameMedia, filepath) {
			log.Debugf("The file is already deleted. bucket: %s, filepath: %s", h.bucketNameMedia, filepath)
			continue
		}

		// delete
		if errDelete := h.fileHandler.DeleteForce(ctx, h.bucketNameMedia, filepath); errDelete != nil {
			log.Errorf("Could not delete recording file. filepath: %s, err: %v", filepath, errDelete)
			return errDelete
		}
	}

	return nil
}
