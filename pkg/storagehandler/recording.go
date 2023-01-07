package storagehandler

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketfile"
)

// RecordingGet returns given recording's bucketfile info.
func (h *storageHandler) RecordingGet(ctx context.Context, id uuid.UUID) (*bucketfile.BucketFile, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "RecordingGet",
			"recoding_id": id,
		},
	)

	// get recording
	r, err := h.reqHandler.CallV1RecordingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get recording info from the call-manager. err: %v", err)
		return nil, err
	}

	// compose files
	targetpaths := []string{}
	for _, filename := range r.Filenames {
		tmp := bucketDirectoryRecording + "/" + filename
		targetpaths = append(targetpaths, tmp)
	}

	// get download uri and filepath
	bucketPath, downloadURI, err := h.bucketHandler.GetDownloadURI(ctx, h.bucketNameMedia, targetpaths, time.Hour*24)
	if err != nil {
		log.Errorf("Could not get download link. err: %v", err)
		return nil, err
	}

	// create recording.Recording
	expire := time.Now().UTC().Add(24 * time.Hour)
	res := &bucketfile.BucketFile{
		ReferenceType:    bucketfile.ReferenceTypeRecording,
		ReferenceID:      id,
		BucketURI:        *bucketPath,
		DownloadURI:      *downloadURI,
		TMDownloadExpire: strings.TrimSuffix(expire.String(), " +0000 UTC"),
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

		if !h.bucketHandler.IsExist(ctx, h.bucketNameMedia, filename) {
			log.Debugf("The file is already deleted. filename: %s", filename)
			continue
		}

		// delete
		if errDelete := h.bucketHandler.Delete(ctx, h.bucketNameMedia, filename); errDelete != nil {
			log.Errorf("Could not delete recording file. filename: %s, err: %v", filename, errDelete)
			return errDelete
		}
	}

	return nil
}
