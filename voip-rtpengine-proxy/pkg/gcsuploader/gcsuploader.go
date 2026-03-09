package gcsuploader

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
)

type gcsUploader struct {
	client     *storage.Client
	bucketName string
}

// New creates a GCS uploader using Application Default Credentials.
func New(bucketName string) (Uploader, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create GCS client: %w", err)
	}

	return &gcsUploader{
		client:     client,
		bucketName: bucketName,
	}, nil
}

func (u *gcsUploader) Upload(localPath string, objectPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("could not open file %s: %w", localPath, err)
	}
	defer f.Close()

	obj := u.client.Bucket(u.bucketName).Object(objectPath)
	w := obj.NewWriter(ctx)
	w.ContentType = "application/vnd.tcpdump.pcap"

	if _, err := io.Copy(w, f); err != nil {
		w.Close()
		return "", fmt.Errorf("could not upload to GCS: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("could not finalize GCS upload: %w", err)
	}

	uri := fmt.Sprintf("gs://%s/%s", u.bucketName, objectPath)
	log.WithField("uri", uri).Info("uploaded pcap to GCS")
	return uri, nil
}

func (u *gcsUploader) Close() error {
	return u.client.Close()
}
