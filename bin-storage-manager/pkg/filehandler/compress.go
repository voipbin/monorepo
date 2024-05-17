package filehandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// CompressCreate creates compressed file with the given src files and returns created compressed file's bucket name and filepath.
func (h *fileHandler) CompressCreate(ctx context.Context, srcBucketName string, srcFilepaths []string) (string, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "CompressCreate",
		"src_bucket_name": srcBucketName,
		"src_filepaths":   srcFilepaths,
	})

	// create file object
	compressFilepath := createZipFilepathHash(srcFilepaths)
	log.Debugf("Created filepath. filepath: %s", compressFilepath)

	// check existence
	if h.IsExist(ctx, h.bucketTmp, compressFilepath) {
		// compressed file already exists
		return h.bucketTmp, compressFilepath, nil
	}

	// genereate
	if errCreate := h.bucketfileCompressFiles(ctx, compressFilepath, srcBucketName, srcFilepaths); errCreate != nil {
		log.Errorf("Could not create the compress file. err: %v", errCreate)
		return "", "", errCreate
	}

	// check existence
	if !h.IsExist(ctx, h.bucketTmp, compressFilepath) {
		log.Errorf("Could not find created compressed file info")
		return "", "", fmt.Errorf("could not find compressed file")
	}

	return h.bucketTmp, compressFilepath, nil
}
