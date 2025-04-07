package filehandler

import (
	"context"
	"fmt"
	"monorepo/bin-storage-manager/models/file"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CompressCreate creates compressed file with the given src files and returns created compressed file's bucket name and filepath.
func (h *fileHandler) CompressCreate(ctx context.Context, files []*file.File) (string, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "CompressCreate",
		"files": files,
	})

	// create file object
	filenames := []string{}
	for _, f := range files {
		filenames = append(filenames, f.ID.String())
	}

	compressFilepath := createZipFilepathHash(filenames)
	log.Debugf("Created filepath. filepath: %s", compressFilepath)

	// check existence
	if h.IsExist(ctx, h.bucketTmp, compressFilepath) {
		// compressed file already exists
		return h.bucketTmp, compressFilepath, nil
	}

	// genereate
	if errCreate := h.bucketfileCompressFiles(ctx, compressFilepath, files); errCreate != nil {
		return "", "", errors.Wrapf(errCreate, "could not create the compress file. filepath: %s", compressFilepath)
	}

	// check existence
	if !h.IsExist(ctx, h.bucketTmp, compressFilepath) {
		return "", "", fmt.Errorf("could not find compressed file")
	}

	return h.bucketTmp, compressFilepath, nil
}
