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

// // GetDownloadURL returns a download url for given target files
// func (h *fileHandler) GetDownloadURI(ctx context.Context, bucketName string, filepaths []string, expire time.Duration) (*string, *string, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":        "GetDownloadURI",
// 		"bucket_name": bucketName,
// 	})

// 	// create file object
// 	filepath := createZipFilepathHash(filepaths)
// 	log.Debugf("Created filepath. filepath: %s", filepath)

// 	// get attrs.
// 	attrs, err := h.bucketfileGetAttrs(ctx, h.bucketTmp, filepath)
// 	if err != nil {
// 		log.Debugf("Could not get attrs. Creating a new compress file. err: %v", err)

// 		// genereate
// 		if errCreate := h.bucketfileCompressFiles(ctx, filepath, bucketName, filepaths); errCreate != nil {
// 			log.Errorf("Could not create the compress file. err: %v", errCreate)
// 			return nil, nil, errCreate
// 		}

// 		// get attrs
// 		attrs, err = h.bucketfileGetAttrs(ctx, h.bucketTmp, filepath)
// 		if err != nil {
// 			log.Errorf("Could not get attrs after created a new compress file. err: %v", err)
// 			return nil, nil, err
// 		}
// 	}
// 	log.WithField("attrs", attrs).Debugf("Detailed attrs info. filepath: %s", filepath)

// 	// get download uri with expiration
// 	tmExpire := time.Now().UTC().Add(expire)
// 	resDownloadURL, err := h.bucketfileGenerateDownloadURI(h.bucketTmp, filepath, tmExpire)
// 	if err != nil {
// 		log.Errorf("Could not get download link. err: %v", err)
// 		return nil, nil, err
// 	}

// 	return &attrs.MediaLink, &resDownloadURL, nil
// }
