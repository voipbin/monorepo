package buckethandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package buckethandler -destination ./mock_buckethandler_buckethandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"io"
	"io/ioutil"
	"os"
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
	FileGet(target string) ([]byte, error)
	FileGetDownloadURL(target string, expire time.Time) (string, error)
	FileExist(target string) bool
	FileUpload(src, dest string) error
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

// FileGetDownloadURL returns google cloud storage signed url for file download
func (h *bucketHandler) FileGetDownloadURL(target string, expire time.Time) (string, error) {

	// create opt
	opts := &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "GET",
		GoogleAccessID: h.accessID,
		PrivateKey:     h.privateKey,
		Expires:        expire,
	}

	// get downloadable url
	u, err := storage.SignedURL(h.bucketName, target, opts)
	if err != nil {
		logrus.Errorf("Could not get signed url. err: %v", err)
		return "", err
	}

	return u, nil
}

// FileGetDownloadURL returns google cloud storage signed url for file download
// The caller must close the returned reader.
func (h *bucketHandler) FileGet(target string) ([]byte, error) {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"target": target,
		},
	)

	rc, err := h.client.Bucket(h.bucketName).Object(target).NewReader(ctx)
	if err != nil {
		log.Errorf("Could not get object info. err: %v", err)
		return nil, err
	}
	defer rc.Close()

	// read the data
	data, err := ioutil.ReadAll(rc)
	if err != nil {
		log.Errorf("Could not read the file. err: %v", err)
		return nil, err
	}

	return data, nil
}
