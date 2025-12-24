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
	return compressCreate(ctx, files, h.bucketTmp, createZipFilepathHash, h.IsExist, h.bucketfileCompressFiles)
}

func compressCreate(
	ctx context.Context,
	files []*file.File,
	bucketTmp string,
	hashFunc func([]string) string,
	isExistFunc func(context.Context, string, string) bool,
	compressFunc func(context.Context, string, []*file.File) error,
) (string, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "CompressCreate",
		"files": files,
	})

	// create file object
	filenames := []string{}
	for _, f := range files {
		filenames = append(filenames, f.ID.String())
	}

	compressFilepath := hashFunc(filenames)
	log.Debugf("Created filepath. filepath: %s", compressFilepath)

	// check existence
	if isExistFunc(ctx, bucketTmp, compressFilepath) {
		// compressed file already exists
		return bucketTmp, compressFilepath, nil
	}

	// genereate
	if errCreate := compressFunc(ctx, compressFilepath, files); errCreate != nil {
		return "", "", errors.Wrapf(errCreate, "could not create the compress file. filepath: %s", compressFilepath)
	}

	// check existence
	if !isExistFunc(ctx, bucketTmp, compressFilepath) {
		return "", "", fmt.Errorf("could not find compressed file")
	}

	return bucketTmp, compressFilepath, nil
}
