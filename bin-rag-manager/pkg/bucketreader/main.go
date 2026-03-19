package bucketreader

import (
	"context"
	"fmt"
	"io"
	"os"

	"cloud.google.com/go/storage"
)

//go:generate mockgen -package bucketreader -source main.go -destination mock_main.go

// BucketReader reads files from GCS buckets.
type BucketReader interface {
	DownloadToTempFile(ctx context.Context, bucketName, filepath string) (tmpPath string, err error)
}

type bucketReader struct {
	client *storage.Client
}

// NewBucketReader creates a new BucketReader with the given GCS client.
func NewBucketReader(client *storage.Client) BucketReader {
	return &bucketReader{client: client}
}

func (b *bucketReader) DownloadToTempFile(ctx context.Context, bucketName, filepath string) (string, error) {
	reader, err := b.client.Bucket(bucketName).Object(filepath).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("could not open GCS object %s/%s: %w", bucketName, filepath, err)
	}
	defer func() { _ = reader.Close() }()

	tmpFile, err := os.CreateTemp("", "rag_gcs_*")
	if err != nil {
		return "", fmt.Errorf("could not create temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, reader); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("could not download GCS object: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("could not close temp file: %w", err)
	}

	return tmpFile.Name(), nil
}
