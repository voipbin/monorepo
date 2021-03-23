package buckethandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package buckethandler -destination ./mock_buckethandler_buckethandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"io/ioutil"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const (
	bucketDirectory = "tts"
)

// BucketHandler intreface for GCP bucket handler
type BucketHandler interface {
	RecordingGetDownloadURL(recordingID string, expire time.Time) (string, error)
}

type bucketHandler struct {
	client *storage.Client

	projectID  string
	bucketName string
	accessID   string
	privateKey []byte
}

// NewBucketHandler create bucket handler
func NewBucketHandler(credentialPath string, projectID string, bucketName string) BucketHandler {

	ctx := context.Background()

	jsonKey, err := ioutil.ReadFile(credentialPath)
	if err != nil {
		logrus.Errorf("Could not read the credential file. err: %v", err)
		return nil
	}

	// parse service account
	conf, err := google.JWTConfigFromJSON(jsonKey)
	if err != nil {
		logrus.Errorf("Could not parse the credential file. err: %v", err)
		return nil
	}

	// create client
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialPath))
	if err != nil {
		logrus.Errorf("Could not create a new client. err: %v", err)
		return nil
	}

	h := &bucketHandler{
		client: client,

		projectID:  projectID,
		bucketName: bucketName,
		accessID:   conf.Email,
		privateKey: conf.PrivateKey,
	}

	return h
}

// Init initialize the bucket
func (h *bucketHandler) Init() {
	return
}
