package servicehandler

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *serviceHandler) RecordingFileMove(ctx context.Context, filenames []string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "RecordingFileMove",
		"filenames": filenames,
	})

	if h.client == nil {
		return errors.New("GCS client is not configured; cannot upload recording files. Check the service-account credential (GOOGLE_APPLICATION_CREDENTIALS_JSON)")
	}

	log.Debugf("Moving the recording files.")

	for _, filename := range filenames {
		if errUpload := h.recordingFileUpload(ctx, filename); errUpload != nil {
			return errors.Wrapf(errUpload, "Could not upload the recording file. filename: %s", filename)
		}
	}
	log.Debugf("Uploaded the recording files.")

	return nil
}

func (h *serviceHandler) recordingFileUpload(ctx context.Context, filename string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "recordingFileUpload",
		"filename": filename,
	})

	sourceFilepath := fmt.Sprintf("%s/%s", h.recordingAsteriskDirectory, filename)
	destinationFilepath := fmt.Sprintf("%s/%s", h.recordingBucketDirectory, filename)

	sourceFile, err := os.Open(sourceFilepath)
	if err != nil {
		return errors.Wrapf(err, "failed to open source file. source_filepath: %s", sourceFilepath)
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	// The writer context is cancelled on a copy failure so that Close aborts the upload
	// instead of committing a partial object. Cancelling after a successful Close is a no-op.
	writerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	wc := h.client.Bucket(h.recordingBucketName).Object(destinationFilepath).NewWriter(writerCtx)
	if _, errCopy := io.Copy(wc, sourceFile); errCopy != nil {
		cancel()
		_ = wc.Close() // abort; the copy error is the one returned
		return errors.Wrapf(errCopy, "failed to copy data. source_filepath: %s, destination_filepath: %s", sourceFilepath, destinationFilepath)
	}

	// GCS commits the upload in Close; its error is the upload result.
	if errClose := wc.Close(); errClose != nil {
		return errors.Wrapf(errClose, "failed to upload the file to the bucket. source_filepath: %s, destination_filepath: %s", sourceFilepath, destinationFilepath)
	}
	log.Debugf("Uploaded the file to bucket. source_filepath: %s, destination_filepath: %s", sourceFilepath, destinationFilepath)

	// Delete the local file only after the upload is committed. A delete failure does not
	// fail the request: the recording is safely stored and must still be registered.
	if errRemove := os.Remove(sourceFilepath); errRemove != nil {
		log.WithFields(logrus.Fields{
			"source_filepath":      sourceFilepath,
			"destination_filepath": destinationFilepath,
		}).Errorf("Uploaded the recording file but could not delete the local file. Remove it manually. err: %v", errRemove)
	}

	return nil
}
