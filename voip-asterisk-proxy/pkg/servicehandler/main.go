package servicehandler

//go:generate mockgen -package servicehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

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

// NewServiceHandler creates a ServiceHandler.
//
// credentialJSON is the GCP service-account key JSON (not a file path). When it is set the
// GCS client is built with option.WithAuthCredentialsJSON(ServiceAccount); when it is empty the client falls back
// to Application Default Credentials (local dev / transition). In either case, if the client
// cannot be created the handler is still returned with a nil client rather than a nil handler.
// A nil-interface return would panic the RPC dispatcher (proxy_handler dispatches on this
// interface); instead RecordingFileMove guards against a nil client and returns a clear error,
// so a missing credential degrades recording upload only, without taking the proxy down. This
// mirrors bin-storage-manager / bin-transcribe-manager's "constructor must not return nil"
// convention.
func NewServiceHandler(credentialJSON string, recordingBucketName string, recordingAsteriskDirectory string, recordingBucketDirectory string) ServiceHandler {
	log := logrus.WithFields(logrus.Fields{
		"func": "NewServiceHandler",
	})

	var client *storage.Client
	var err error
	if credentialJSON != "" {
		// WithAuthCredentialsJSON with an explicit ServiceAccount type (not the deprecated
		// WithCredentialsJSON): the credential is always a VoIPBin-owned service-account key,
		// so pinning the type avoids loading an unexpected credential kind.
		client, err = storage.NewClient(context.Background(), option.WithAuthCredentialsJSON(option.ServiceAccount, []byte(credentialJSON)))
	} else {
		client, err = storage.NewClient(context.Background())
	}
	if err != nil {
		// Do NOT return nil: the RPC dispatcher calls this interface's method directly, and a
		// nil interface would panic on dispatch. Keep the handler alive with a nil client;
		// RecordingFileMove guards on it and returns a clear error so only recording upload
		// degrades.
		log.Errorf("Could not create Google Cloud Storage client. Recording upload will be unavailable until a valid credential is configured. err: %v", err)
		client = nil
	}

	return &serviceHandler{
		client: client,

		recordingBucketName:        recordingBucketName,
		recordingAsteriskDirectory: recordingAsteriskDirectory,
		recordingBucketDirectory:   recordingBucketDirectory,
	}
}
