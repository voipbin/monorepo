package filehandler

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// DownloadURIGet returns a bucket uri and download url for given file
func (h *fileHandler) DownloadURIGet(ctx context.Context, bucketName string, filepath string, expire time.Duration) (string, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "DownloadURIGet",
		"bucket_name": bucketName,
		"filepath":    filepath,
		"expire":      expire,
	})

	// get attrs
	attrs, err := h.bucketfileGetAttrs(ctx, h.bucketTmp, filepath)
	if err != nil {
		log.Errorf("Could not get attrs after created a new compress file. err: %v", err)
		return "", "", err
	}
	log.WithField("attrs", attrs).Debugf("Detailed attrs info. filepath: %s", filepath)

	// get download uri with expiration
	tmExpire := time.Now().UTC().Add(expire)
	resDownloadURL, err := h.bucketfileGenerateDownloadURI(bucketName, filepath, tmExpire)
	if err != nil {
		log.Errorf("Could not get download link. err: %v", err)
		return "", "", err
	}

	return attrs.MediaLink, resDownloadURL, nil
}
