package filehandler

import (
	"archive/zip"
	"context"
	"io"
	"monorepo/bin-storage-manager/models/file"
	"time"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// bucketfileMove moves the file from one bucket to another
func (h *fileHandler) bucketfileMove(ctx context.Context, sourceBucketName string, sourceFilepath string, destBucketName string, destFilepath string) (*storage.ObjectAttrs, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "bucketfileMove",
		"source_bucket_name": sourceBucketName,
		"source_filepath":    sourceFilepath,
		"dest_bucket_name":   destBucketName,
		"dest_filepath":      destFilepath,
	})

	// check source
	src := h.client.Bucket(sourceBucketName).Object(sourceFilepath)
	if _, err := src.Attrs(ctx); err != nil {
		return nil, errors.Wrap(err, "source does not exist")
	}

	// check destination
	dst := h.client.Bucket(destBucketName).Object(destFilepath)
	_, err := dst.Attrs(ctx)
	if err == nil {
		return nil, errors.Wrap(err, "destination already exists")
	}

	// copy to the destination
	res, err := dst.CopierFrom(src).Run(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not copy the file to the destination. source_bucket_name: %s, source_filepath: %s, dest_bucket_name: %s, dest_filepath: %s")
	}

	// delete source
	if err := src.Delete(ctx); err != nil {
		// we could not delete the source, but we don't want to fail
		log.Errorf("Could not delete the source. err: %v", err)
	}

	return res, nil
}

// bucketfileDelete deletes the given bucketfile from the bucket
func (h *fileHandler) bucketfileDelete(ctx context.Context, bucketName string, filepath string) error {

	fo := h.client.Bucket(bucketName).Object(filepath)
	if errDelete := fo.Delete(ctx); errDelete != nil {
		return errors.Wrapf(errDelete, "could not delete the file. bucket_name: %s, filepath: %s", bucketName, filepath)
	}

	return nil
}

// bucketfileCompressFiles create a new compress file into the tmp bucket.
func (h *fileHandler) bucketfileCompressFiles(ctx context.Context, dstFilepath string, srcBucketName string, srcFilepaths []string) (resErr error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "bucketfileCompressFiles",
		"dst_filepath":    dstFilepath,
		"src_bucket_name": srcBucketName,
		"src_filepaths":   srcFilepaths,
	})

	// create zip filepath writer
	fo := h.client.Bucket(h.bucketTmp).Object(dstFilepath)
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

	for _, target := range srcFilepaths {
		f := h.client.Bucket(srcBucketName).Object(target)

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
			return errors.Wrapf(err, "could not add the file to the res file. err: %v", err)
		}

		// copy
		_, err = io.Copy(fp, reader)
		if err != nil {
			return errors.Wrapf(err, "could not copy the file. err: %v", err)
		}
	}

	return nil
}

// BucketfileCompressFiles create a new compress file into the tmp bucket.
func (h *fileHandler) BucketfileCompressFiles(ctx context.Context, dstFilepath string, files []*file.File) (resErr error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "BucketfileCompressFiles",
		"dst_filepath": dstFilepath,
		"files":        files,
	})

	// create zip filepath writer
	fo := h.client.Bucket(h.bucketTmp).Object(dstFilepath)
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

	for _, f := range files {
		bucketFile := h.client.Bucket(f.BucketName).Object(f.Filepath)

		// read open
		reader, err := bucketFile.NewReader(ctx)
		if err != nil {
			log.Errorf("Could not create a reader. err: %v", err)
			continue
		}
		defer reader.Close()

		// add the filename to the result file
		fp, err := zw.Create(f.Filename)
		if err != nil {
			return errors.Wrapf(err, "could not add the file to the res file. err: %v", err)
		}

		// copy
		_, err = io.Copy(fp, reader)
		if err != nil {
			return errors.Wrapf(err, "could not copy the file. err: %v", err)
		}
	}

	return nil
}

// bucketfileGetAttrs returns the given bucket file's attrs
func (h *fileHandler) bucketfileGetAttrs(ctx context.Context, bucketName string, filepath string) (*storage.ObjectAttrs, error) {

	res, err := h.client.Bucket(bucketName).Object(filepath).Attrs(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the file attrs. bucket_name: %s, filepath: %s", bucketName, filepath)
	}

	return res, nil
}

// bucketfileGenerateDownloadURI returns google cloud storage signed url for file download
func (h *fileHandler) bucketfileGenerateDownloadURI(bucketName string, filepath string, expire time.Time) (string, error) {

	// create opt
	opts := &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV2,
		Method:         "GET",
		GoogleAccessID: h.accessID,
		PrivateKey:     h.privateKey,
		Expires:        expire,
	}

	// get downloadable url
	u, err := storage.SignedURL(bucketName, filepath, opts)
	if err != nil {
		return "", errors.Wrapf(err, "could not get the signed url. bucket_name: %s, filepath: %s", bucketName, filepath)
	}

	return u, nil
}
