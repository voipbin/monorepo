package buckethandler

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
)

// fileUpload uploads the filename to the bucket(target)
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

// fileExist return the true if the given target is exists in the bucket
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

// fileExist return the true if the given target is exists in the bucket
func (h *bucketHandler) FileGetAttrs(target string) (*storage.ObjectAttrs, error) {
	ctx := context.Background()

	f := h.client.Bucket(h.bucketName).Object(target)
	attrs, err := f.Attrs(ctx)
	if err != nil {
		return nil, err
	}

	return attrs, nil
}

// fileGetDownloadURL returns google cloud storage signed url for file download
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

// fileGet returns google cloud storage signed url for file download
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
