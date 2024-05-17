package filehandler

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"monorepo/bin-storage-manager/models/file"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *fileHandler) Create(ctx context.Context, customerID uuid.UUID, ownerID uuid.UUID, name string, detail string, filepath string) (*file.File, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
		"owner_id":    ownerID,
		"filepath":    filepath,
	})

	// check file does exist
	attrs, err := h.getAttrs(ctx, h.bucketTmp, filepath)
	if err != nil {
		log.Errorf("Could not get attrs. Considering the file does not exist. err: %v", err)
		return nil, errors.Wrapf(err, "file does not exist")
	}
	log.WithField("attrs", attrs).Debugf("Found file. name: %s", attrs.Name)

	// generate destination filepath
	tmpFilename := getFilename(filepath)
	destFilepath := fmt.Sprintf("%s/%s", "bin", tmpFilename)

	// move the file to the new location
	dstAttrs, err := h.moveFile(ctx, h.bucketTmp, filepath, h.bucketMedia, destFilepath)
	if err != nil {
		log.Errorf("Could not move the file. err: %v", err)
		return nil, err
	}

	// get permernat dowload uri
	expireDuration := 365 * 24 * time.Hour           // 1 year
	tmExpire := time.Now().UTC().Add(expireDuration) // 1 year4
	tmDownloadExpire := h.utilHandler.TimeGetCurTimeAdd(expireDuration)
	downloadURI, err := h.generateDownloadURI(h.bucketTmp, filepath, tmExpire)

	// create db row
	f := &file.File{
		ID:               h.utilHandler.UUIDCreate(),
		CustomerID:       customerID,
		OwnerID:          ownerID,
		Name:             name,
		Detail:           detail,
		URIBucket:        dstAttrs.MediaLink,
		URIDownload:      downloadURI,
		TMDownloadExpire: tmDownloadExpire,
	}

	if errCreate := h.db.FileCreate(ctx, f); errCreate != nil {
		log.Errorf("Could not create file")
	}

	// res, err :=

}

// GetDownloadURL returns a download url for given target files
func (h *fileHandler) GetDownloadURI(ctx context.Context, bucketName string, filepaths []string, expire time.Duration) (*string, *string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetDownloadURI",
		"bucket_name": bucketName,
	})

	// create file object
	filepath := createZipFilepathHash(filepaths)
	log.Debugf("Created filepath. filepath: %s", filepath)

	// get attrs.
	attrs, err := h.getAttrs(ctx, h.bucketTmp, filepath)
	if err != nil {
		log.Debugf("Could not get attrs. Creating a new compress file. err: %v", err)

		// genereate
		if errCreate := h.createCompressFile(ctx, filepath, bucketName, filepaths); errCreate != nil {
			log.Errorf("Could not create the compress file. err: %v", errCreate)
			return nil, nil, errCreate
		}

		// get attrs
		attrs, err = h.getAttrs(ctx, h.bucketTmp, filepath)
		if err != nil {
			log.Errorf("Could not get attrs after created a new compress file. err: %v", err)
			return nil, nil, err
		}
	}
	log.WithField("attrs", attrs).Debugf("Detailed attrs info. filepath: %s", filepath)

	// get download uri with expiration
	tmExpire := time.Now().UTC().Add(expire)
	resDownloadURL, err := h.generateDownloadURI(h.bucketTmp, filepath, tmExpire)
	if err != nil {
		log.Errorf("Could not get download link. err: %v", err)
		return nil, nil, err
	}

	return &attrs.MediaLink, &resDownloadURL, nil
}

// createCompressFile create a new compress file
func (h *fileHandler) createCompressFile(ctx context.Context, zipFilepath string, targetBucketName string, targetFilepaths []string) (resErr error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "createCompressFile",
	})

	// create zip filepath writer
	fo := h.client.Bucket(h.bucketTmp).Object(zipFilepath)
	defer func() {
		if resErr != nil {
			log.Errorf("Could not finish the create compress file correctly. Deleting the file. err: %v", resErr)
			_ = fo.Delete(ctx)
		}
	}()

	fw := fo.NewWriter(ctx)
	defer fw.Close()

	// create a zip
	zw := zip.NewWriter(fw)
	defer func() {
		// close zip
		if errClose := zw.Close(); errClose != nil {
			log.Errorf("Could not close the zip writer. err: %v", errClose)
			return
		}
	}()

	for _, target := range targetFilepaths {
		f := h.client.Bucket(targetBucketName).Object(target)

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
			return err
		}

		// copy
		_, err = io.Copy(fp, reader)
		if err != nil {
			log.Errorf("Could not copy the file. err: %v", err)
			return err
		}
	}

	return nil
}

// moveFile moves the file from one bucket to another
func (h *fileHandler) moveFile(ctx context.Context, sourceBucketName string, sourceFilepath string, destBucketName string, destFilepath string) (*storage.ObjectAttrs, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "moveFile",
		"source_bucket_name": sourceBucketName,
		"source_filepath":    sourceFilepath,
		"dest_bucket_name":   destBucketName,
		"dest_filepath":      destFilepath,
	})

	// check source
	src := h.client.Bucket(sourceBucketName).Object(sourceFilepath)
	if _, err := src.Attrs(ctx); err != nil {
		log.Errorf("The source does not exist. err: %v", err)
		return nil, errors.Wrap(err, "source does not exist")
	}

	// check destination
	dst := h.client.Bucket(destBucketName).Object(destFilepath)
	res, err := dst.Attrs(ctx)
	if err == nil {
		log.Errorf("The destination already exists. err: %v", err)
		return nil, errors.Wrap(err, "destination already exists")
	}

	// copy to the destination
	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		return nil, errors.Wrap(err, "could not copy the file to the destination")
	}

	// delete source
	if err := src.Delete(ctx); err != nil {
		// we could not delete the source, but we don't want to fail
		log.Errorf("Could not delete the source. err: %v", err)
	}

	return res, nil
}

// generateDownloadURI returns google cloud storage signed url for file download
func (h *fileHandler) generateDownloadURI(bucketName string, target string, expire time.Time) (string, error) {
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

// IsExist returns true if the given file exist
func (h *fileHandler) getAttrs(ctx context.Context, bucketName string, filepath string) (*storage.ObjectAttrs, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "getAttrs",
	})

	res, err := h.client.Bucket(bucketName).Object(filepath).Attrs(ctx)
	if err != nil {
		log.Errorf("Could not get attrs. err: %v", err)
		return nil, err
	}

	return res, nil
}

// IsExist returns true if the given file exist
func (h *fileHandler) IsExist(ctx context.Context, bucketName string, filepath string) bool {
	_, err := h.getAttrs(ctx, bucketName, filepath)
	return err == nil
}

// Delete deletes the given file
func (h *fileHandler) Delete(ctx context.Context, bucketName string, filepath string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Delete",
	})

	fo := h.client.Bucket(bucketName).Object(filepath)
	if errDelete := fo.Delete(ctx); errDelete != nil {
		log.Errorf("Could not delete the file. err: %v", errDelete)
		return errDelete
	}

	return nil
}
