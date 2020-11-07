package buckethandler

//go:generate mockgen -destination ./mock_buckethandler_buckethandler.go -package buckethandler -source ./main.go BucketHandler

import (
	"context"
	"io"
	"os"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

const (
	bucketDirectory = "tts"
)

// BucketHandler intreface for GCP bucket handler
type BucketHandler interface {
	FileUpload(src, dest string) error
	FileExist(target string) bool
}

type bucketHandler struct {
	client *storage.Client

	projectID  string
	bucketName string
}

// NewBucketHandler create bucket handler
func NewBucketHandler(credentialPath string, projectID string, bucketName string) BucketHandler {

	ctx := context.Background()

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
	}

	return h
}

// Init initialize the bucket
func (h *bucketHandler) Init() {
	return
}

// FileUpload uploads the filename to the bucket(target)
func (h *bucketHandler) FileUpload(src, dest string) error {

	// open file
	f, err := os.Open(src)
	if err != nil {
		logrus.Errorf("Could not open the target file. err: %v", err)
		return err
	}
	defer f.Close()

	// create a session
	ctx := context.Background()
	wc := h.client.Bucket(h.bucketName).Object(dest).NewWriter(ctx)
	defer wc.Close()

	// upload the file
	if _, err = io.Copy(wc, f); err != nil {
		logrus.Errorf("Could not upload the file to the bucket. err: %v", err)
		return err
	}

	if err := wc.Close(); err != nil {
		logrus.Errorf("Could not close the write. err: %v", err)
		return err
	}

	return nil
}

// FileExist return the true if the given target is exists in the bucket
func (h *bucketHandler) FileExist(target string) bool {
	ctx := context.Background()

	f := h.client.Bucket(h.bucketName).Object(target)
	attrs, err := f.Attrs(ctx)
	if err != nil {
		return false
	}
	logrus.Debugf("The file is exist. filename: %s, bucket: %s, created: %s", target, attrs.Bucket, attrs.Created)

	return true
}
