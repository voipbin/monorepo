package buckethandler

import (
	"archive/zip"
	"context"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
)

// GetDownloadURL returns a download url for given target files
func (h *bucketHandler) GetDownloadURI(ctx context.Context, filepaths []string, expire time.Duration) (*string, *string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "GetDownloadURI",
	})

	// create file object
	filepath := bucketDirectoryTmp + "/" + h.utilHandler.CreateUUID().String() + ".zip"
	fo := h.client.Bucket(h.bucketTmp).Object(filepath)
	resBucketpath := "gs://" + h.bucketTmp + "/" + filepath
	log.Debugf("Created a file object. filepath: %s, bucketpath: %s", filepath, resBucketpath)

	fw := fo.NewWriter(ctx)
	defer fw.Close()

	// create a zip
	zw := zip.NewWriter(fw)
	for _, target := range filepaths {
		f := h.client.Bucket(h.bucketMedia).Object(target)

		// read open
		reader, err := f.NewReader(ctx)
		if err != nil {
			log.Errorf("Could not create a reader. err: %v", err)
			continue
		}
		defer reader.Close()

		// add the filename to the result file
		filename := getFilename(target)
		fp, err := zw.Create(filename)
		if err != nil {
			log.Errorf("Could not add the file to the res file. err: %v", err)
			continue
		}

		// copy
		_, err = io.Copy(fp, reader)
		if err != nil {
			log.Errorf("Could not copy the file. err: %v", err)
			continue
		}
	}

	// close zip
	if errClose := zw.Close(); errClose != nil {
		log.Errorf("Could not close the zip writer. err: %v", errClose)
		return nil, nil, errClose
	}

	// get download uri with expiration
	tmExpire := time.Now().UTC().Add(expire)
	resDownloadURL, err := h.generateDownloadURI(h.bucketTmp, filepath, tmExpire)
	if err != nil {
		log.Errorf("Could not get download link. err: %v", err)
		return nil, nil, err
	}

	return &resBucketpath, &resDownloadURL, nil
}

// generateDownloadURI returns google cloud storage signed url for file download
func (h *bucketHandler) generateDownloadURI(bucketName string, target string, expire time.Time) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "generateDownloadURI",
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
	u, err := storage.SignedURL(bucketName, target, opts)
	if err != nil {
		log.Errorf("Could not get signed url. err: %v", err)
		return "", err
	}

	return u, nil
}
