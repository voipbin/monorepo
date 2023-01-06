package storagehandler

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketfile"
)

func (h *storageHandler) getRecordingFilepath(filename string) string {
	res := bucketDirectoryRecording + "/" + filename
	return res
}

// GetRecording returns bucketrecording info of the given recording id.
func (h *storageHandler) GetRecording(ctx context.Context, id uuid.UUID) (*bucketfile.BucketFile, error) {
	log := logrus.WithFields(
		logrus.Fields{
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
	bucketPath, downloadURI, err := h.bucketHandler.GetDownloadURI(ctx, targetpaths, time.Hour*24)
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
