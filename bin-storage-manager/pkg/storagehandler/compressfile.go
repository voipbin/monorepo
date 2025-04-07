package storagehandler

import (
	"context"
	compressfile "monorepo/bin-storage-manager/models/compressfile"
	"monorepo/bin-storage-manager/models/file"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CompressfileCreate returns given compress file info
func (h *storageHandler) CompressfileCreate(ctx context.Context, referenceIDs []uuid.UUID, fileIDs []uuid.UUID) (*compressfile.CompressFile, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "CompressfileCreate",
		"reference_ids": referenceIDs,
		"file_ids":      fileIDs,
	})

	// get target files
	targetFiles := []*file.File{}
	if len(fileIDs) > 0 {
		tmps, err := h.compressGetFilesByFileIDs(ctx, fileIDs)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get files. file_ids: %v", fileIDs)
		}
		targetFiles = append(targetFiles, tmps...)
	}
	if len(referenceIDs) > 0 {
		tmps, err := h.compressGetFilesByReferenceIDs(ctx, referenceIDs)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get files. reference_ids: %v", referenceIDs)
		}
		targetFiles = append(targetFiles, tmps...)
	}

	targetFileIDs := []uuid.UUID{}
	for _, f := range targetFiles {
		targetFileIDs = append(targetFileIDs, f.ID)
	}

	// create compress file
	bucketName, filepath, err := h.fileHandler.CompressCreate(ctx, targetFiles)
	if err != nil {
		return nil, errors.Wrapf(err, "could not compress the files. reference_ids: %v, file_ids: %v", referenceIDs, fileIDs)
	}
	log.Debugf("Created compress file. bucket_name: %s, filepath: %s", bucketName, filepath)

	// get download uri
	_, downloadURI, err := h.fileHandler.DownloadURIGet(ctx, bucketName, filepath, time.Hour*24)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get download link. bucket_name: %s, filepath: %s", bucketName, filepath)
	}

	// create compress file
	tmExpire := h.utilHandler.TimeGetCurTimeAdd(24 * time.Hour)
	res := &compressfile.CompressFile{
		FileIDs:          targetFileIDs,
		DownloadURI:      downloadURI,
		TMDownloadExpire: tmExpire,
	}

	return res, nil
}

// compressGetFilesByFileIDs returns a list of files of the given file IDs.
func (h *storageHandler) compressGetFilesByFileIDs(ctx context.Context, fileIDs []uuid.UUID) ([]*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "compressGetFilesByFileIDs",
		"file_ids": fileIDs,
	})

	res := []*file.File{}
	for _, id := range fileIDs {
		f, err := h.FileGet(ctx, id)
		if err != nil {
			log.Errorf("Could not get file info. err: %v", err)
			continue
		}
		res = append(res, f)
	}

	return res, nil
}

// compressGetFilesByReferenceIDs returns a list of files of the given reference IDs.
func (h *storageHandler) compressGetFilesByReferenceIDs(ctx context.Context, referenceIDs []uuid.UUID) ([]*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "compressGetFilesByReferenceIDs",
		"reference_ids": referenceIDs,
	})

	res := []*file.File{}
	for _, id := range referenceIDs {
		filters := map[string]string{
			"deleted":      "false",
			"reference_id": id.String(),
		}

		tmps, err := h.FileGets(ctx, "", 1000, filters)
		if err != nil {
			log.Errorf("Could not get file info. err: %v", err)
			continue
		}
		res = append(res, tmps...)
	}

	return res, nil
}
