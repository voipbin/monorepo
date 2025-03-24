package servicehandler

import (
	"context"
	"fmt"
	"io"
	"os"
)

func (h *serviceHandler) RecordingFileMove(ctx context.Context, filenames []string) error {
	for _, filename := range filenames {
		if err := h.recordingFileMove(filename); err != nil {
			return fmt.Errorf("failed to move recording file: %w", err)
		}
	}

	return nil
}

func (h *serviceHandler) recordingFileMove(filename string) error {

	// Open the source file
	sourceFilepath := fmt.Sprintf("%s/%s", h.recordingAsteriskDirectory, filename)
	sourceFile, err := os.Open(sourceFilepath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create the destination file
	destinationFilepath := fmt.Sprintf("%s/%s", h.recordingBucketDirectory, filename)
	destFile, err := os.Create(destinationFilepath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy data from source to destination
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Flush to ensure data is written
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to flush data: %w", err)
	}

	// Remove the original file
	if err := os.Remove(sourceFilepath); err != nil {
		return fmt.Errorf("failed to delete source file: %w", err)
	}

	return nil
}
