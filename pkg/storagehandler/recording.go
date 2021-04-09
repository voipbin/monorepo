package storagehandler

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketrecording"
)

func (h *storageHandler) getRecordingFilepath(filename string) string {
	res := bucketDirectoryRecording + "/" + filename
	return res
}

func (h *storageHandler) getBucketpath(filepath string) string {

	res := "gs://" + h.bucketHandler.GetBucketName() + "/" + filepath
	return res
}

// GetRecording returns bucketrecording info of the given recording id.
func (h *storageHandler) GetRecording(id uuid.UUID) (*bucketrecording.BucketRecording, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"recoding_id": id,
		},
	)

	// get recording
	rec, err := h.reqHandler.CMRecordingGet(id)
	if err != nil {
		log.Errorf("Could not get recording info from the call-manager. err: %v", err)
		return nil, err
	}

	// get filepath
	filepath := h.getRecordingFilepath(rec.Filename)

	// get download uri
	expire := time.Now().UTC().Add(24 * time.Hour)
	downloadURI, err := h.bucketHandler.FileGetDownloadURL(filepath, expire)
	if err != nil {
		log.Errorf("Could not get download link. err: %v", err)
		return nil, err
	}

	// get bucket path
	bucketPath := h.getBucketpath(filepath)

	// create recording.Recording
	res := &bucketrecording.BucketRecording{
		RecordingID:    id,
		BucketURI:      bucketPath,
		DownloadURI:    downloadURI,
		TMDownloadExpire: strings.TrimSuffix(expire.String(), " +0000 UTC"),
	}

	return res, nil
}
