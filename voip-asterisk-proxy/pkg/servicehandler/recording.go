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

	// Open the source file
	sourceFilepath := fmt.Sprintf("%s/%s", h.recordingAsteriskDirectory, filename)
	sourceFile, err := os.Open(sourceFilepath)
	if err != nil {
		return errors.Wrapf(err, "failed to open source file. source_filepath: %s", sourceFilepath)
	}
	defer sourceFile.Close()

	destinationFilepath := fmt.Sprintf("%s/%s", h.recordingBucketDirectory, filename)
	wc := h.client.Bucket(h.recordingBucketName).Object(destinationFilepath).NewWriter(ctx)
	_, err = io.Copy(wc, sourceFile)
	if err != nil {
		return errors.Wrapf(err, "failed to copy data. source_filepath: %s, destination_filepath: %s", sourceFilepath, destinationFilepath)
	}
	log.Debugf("Uploaded the file to bucket. source_filepath: %s, destination_filepath: %s", sourceFilepath, destinationFilepath)
	defer func() {
		_ = wc.Close()
	}()

	// Remove the original file
	if errRemove := os.Remove(sourceFilepath); errRemove != nil {
		return errors.Wrapf(errRemove, "failed to delete source file. source_filepath: %s", sourceFilepath)
	}

	return nil
}
