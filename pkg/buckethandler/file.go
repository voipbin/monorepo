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

// FileUpload uploads the filename to the bucket(target)
func (h *bucketHandler) FileUpload(ctx context.Context, src, dest string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "FileUpload",
	})

	// open file
	f, err := os.Open(src)
	if err != nil {
		log.Errorf("Could not open the target file. err: %v", err)
		return err
	}
	defer f.Close()

	// create a session
	wc := h.client.Bucket(h.bucketName).Object(dest).NewWriter(ctx)
	defer wc.Close()

	// upload the file
	if _, err = io.Copy(wc, f); err != nil {
		log.Errorf("Could not upload the file to the bucket. err: %v", err)
		return err
	}

	if err := wc.Close(); err != nil {
		log.Errorf("Could not close the write. err: %v", err)
		return err
	}

	return nil
}

// FileExist return the true if the given target is exists in the bucket
func (h *bucketHandler) FileExist(ctx context.Context, target string) bool {
	log := logrus.WithFields(logrus.Fields{
		"func": "FileExist",
	})

	f := h.client.Bucket(h.bucketName).Object(target)
	attrs, err := f.Attrs(ctx)
	if err != nil {
		return false
	}
	log.Debugf("The file is exist. filename: %s, bucket: %s, created: %s", target, attrs.Bucket, attrs.Created)

	return true
}

// FileGetDownloadURL returns google cloud storage signed url for file download
func (h *bucketHandler) FileGetDownloadURL(target string, expire time.Time) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "FileGetDownloadURL",
	})

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
		log.Errorf("Could not get signed url. err: %v", err)
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
