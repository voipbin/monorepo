package servicehandler

//go:generate mockgen -package servicehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
)

type ServiceHandler interface {
	RecordingFileMove(ctx context.Context, filenames []string) error
}

type serviceHandler struct {
	client *storage.Client

	recordingBucketName        string
	recordingAsteriskDirectory string
	recordingBucketDirectory   string
}

func NewServiceHandler(recordingBucketName string, recordingAsteriskDirectory string, recordingBucketDirectory string) ServiceHandler {
	log := logrus.WithFields(logrus.Fields{
		"func": "NewServiceHandler",
	})

	// Create storage client using the decoded credentials
	client, err := storage.NewClient(context.Background())
	if err != nil {
		log.Errorf("Could not create a new storage client. Please ensure the environment is configured for Application Default Credentials (ADC), for example by setting the GOOGLE_APPLICATION_CREDENTIALS environment variable, using workload identity, or relying on the metadata server when running on GCP infrastructure. error: %v", err)
		return nil
	}

	return &serviceHandler{
		client: client,

		recordingBucketName:        recordingBucketName,
		recordingAsteriskDirectory: recordingAsteriskDirectory,
		recordingBucketDirectory:   recordingBucketDirectory,
	}
}
