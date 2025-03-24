package servicehandler

//go:generate mockgen -package servicehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
)

type ServiceHandler interface {
	RecordingFileMove(ctx context.Context, filenames []string) error
}

type serviceHandler struct {
	recordingAsteriskDirectory string
	recordingBucketDirectory   string
}

func NewServiceHandler(recordingAsteriskDirectory string, recordingBucketDirectory string) ServiceHandler {
	return &serviceHandler{
		recordingAsteriskDirectory: recordingAsteriskDirectory,
		recordingBucketDirectory:   recordingBucketDirectory,
	}
}
