package filehandler

import (
	"context"
	"fmt"
	"monorepo/bin-storage-manager/models/file"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Create Creates the file and returns the created file
func (h *fileHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	ownerID uuid.UUID,
	referenceType file.ReferenceType,
	referenceID uuid.UUID,
	name string,
	detail string,
	bucketName string,
	filepath string,
) (*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"customer_id":    customerID,
		"owner_id":       ownerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"name":           name,
		"detail":         detail,
		"bucket_name":    bucketName,
		"filepath":       filepath,
	})

	// check file does exist
	// we are expecting all original files are located in the tmp bucket
	attrs, err := h.bucketfileGetAttrs(ctx, bucketName, filepath)
	if err != nil {
		log.Errorf("Could not get attrs. Considering the file does not exist. err: %v", err)
		return nil, errors.Wrapf(err, "file does not exist")
	}
	log.WithField("attrs", attrs).Debugf("Found file. name: %s", attrs.Name)

	// generate destination filepath
	id := h.utilHandler.UUIDCreate()
	dstFilepath := fmt.Sprintf("%s/%s", bucketDirectoryBin, id)

	// move the file from the tmp bucket to the new location
	dstAttrs, err := h.bucketfileMove(ctx, bucketName, filepath, h.bucketMedia, dstFilepath)
	if err != nil {
		log.Errorf("Could not move the file. err: %v", err)
		return nil, err
	}
	log.WithField("dst_attrs", dstAttrs).Debugf("Moved file. bucket_link: %s", dstAttrs.MediaLink)

	// get dowload uri
	expireDuration := 3650 * 24 * time.Hour // valid for 10 years
	tmExpire := time.Now().UTC().Add(expireDuration)
	tmDownloadExpire := h.utilHandler.TimeGetCurTimeAdd(expireDuration)
	downloadURI, err := h.bucketfileGenerateDownloadURI(h.bucketMedia, dstFilepath, tmExpire)
	if err != nil {
		log.Errorf("Could not generate download URI. err: %v", err)
		return nil, err
	}

	// create db row
	f := &file.File{
		ID:               id,
		CustomerID:       customerID,
		OwnerID:          ownerID,
		ReferenceType:    referenceType,
		ReferenceID:      referenceID,
		Name:             name,
		Detail:           detail,
		BucketName:       h.bucketMedia,
		Filepath:         dstFilepath,
		URIBucket:        dstAttrs.MediaLink,
		URIDownload:      downloadURI,
		TMDownloadExpire: tmDownloadExpire,
	}

	if errCreate := h.db.FileCreate(ctx, f); errCreate != nil {
		log.Errorf("Could not create file. err: %v", errCreate)
		return nil, errCreate
	}

	// get created file
	res, err := h.db.FileGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created file info. err: %v", err)
		return nil, err
	}
	log.WithField("file", res).Debugf("Created file info. id: %s", res.ID)

	h.notifyHandler.PublishEvent(ctx, file.EventTypeFileCreated, res)

	return res, nil
}

func (h *fileHandler) Get(ctx context.Context, id uuid.UUID) (*file.File, error) {
	res, err := h.db.FileGet(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "could not get file info")
	}

	return res, nil
}

// Gets returns list of files
func (h *fileHandler) Gets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "Gets",
		"token": token,
		"size":  size,
		"limit": size,
	})

	res, err := h.db.FileGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get files. err: %v", err)
		return nil, err
	}

	return res, nil
}

// IsExist returns true if the given file exist
func (h *fileHandler) IsExist(ctx context.Context, bucketName string, filepath string) bool {
	_, err := h.bucketfileGetAttrs(ctx, bucketName, filepath)
	return err == nil
}

// DeleteForce deletes the given file from the bucket
func (h *fileHandler) Delete(ctx context.Context, id uuid.UUID) (*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Delete",
		"id":   id,
	})

	if errDelete := h.db.FileDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete the file. err: %v", errDelete)
		return nil, errDelete
	}

	// get deleted file
	res, err := h.db.FileGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted file info. err: %v", err)
		return nil, err
	}

	// delete file
	log.WithField("file", res).Debugf("Deleting bucketfile. bucket_name: %s, filepath: %s", res.BucketName, res.Filepath)
	if errDelete := h.bucketfileDelete(ctx, res.BucketName, res.Filepath); errDelete != nil {
		log.Errorf("Could not delete the bucketfile. err: %v", errDelete)
		// we could not delete the bucketfile. but we don't return the error here.
	}

	h.notifyHandler.PublishEvent(ctx, file.EventTypeFileDeleted, res)

	return res, nil
}

// DeleteBucketfile deletes the given file from the bucket
func (h *fileHandler) DeleteBucketfile(ctx context.Context, bucketName string, filepath string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "DeleteForce",
		"bucket_name": bucketName,
		"filepath":    filepath,
	})

	if errDelete := h.bucketfileDelete(ctx, bucketName, filepath); errDelete != nil {
		log.Errorf("Could not delete the bucket file. err: %v", errDelete)
		return errDelete
	}

	return nil
}
