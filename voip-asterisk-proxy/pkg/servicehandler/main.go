package servicehandler

//go:generate mockgen -package servicehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/base64"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
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

func NewServiceHandler(gcpCredentialBase64 string, recordingBucketName string, recordingAsteriskDirectory string, recordingBucketDirectory string) ServiceHandler {
	log := logrus.WithFields(logrus.Fields{
		"func": "NewServiceHandler",
	})

	decodedCredential, err := base64.StdEncoding.DecodeString(gcpCredentialBase64)
	if err != nil {
		log.Printf("Error decoding base64 credential: %v", err)
		return nil
	}

	// Create storage client using the decoded credentials
	client, err := storage.NewClient(context.Background(), option.WithCredentialsJSON(decodedCredential))
	if err != nil {
		log.Printf("Could not create a new storage client. Error: %v", err)
		return nil
	}

	return &serviceHandler{
		client: client,

		recordingBucketName:        recordingBucketName,
		recordingAsteriskDirectory: recordingAsteriskDirectory,
		recordingBucketDirectory:   recordingBucketDirectory,
	}
}
