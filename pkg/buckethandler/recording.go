package buckethandler

import (
	"fmt"
	"time"
)

const (
	bucketDirectoryRecording = "recording"
)

// RecordingGetDownloadURL returns google cloud storage signed url for recording download
func (h *bucketHandler) RecordingGetDownloadURL(recordingID string, expire time.Time) (string, error) {

	target := fmt.Sprintf("%s/%s", bucketDirectoryRecording, recordingID)

	return h.fileGetDownloadURL(target, expire)
}
