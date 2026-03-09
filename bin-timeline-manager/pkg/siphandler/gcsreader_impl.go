package siphandler

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
)

type gcsReaderImpl struct {
	client     *storage.Client
	bucketName string
}

// NewGCSReader creates a GCSReader backed by a real GCS storage client.
func NewGCSReader(client *storage.Client, bucketName string) GCSReader {
	return &gcsReaderImpl{
		client:     client,
		bucketName: bucketName,
	}
}

func (g *gcsReaderImpl) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "ListObjects",
		"bucket": g.bucketName,
		"prefix": prefix,
	})

	it := g.client.Bucket(g.bucketName).Objects(ctx, &storage.Query{Prefix: prefix})

	names := []string{}
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not list GCS objects with prefix %s: %w", prefix, err)
		}
		names = append(names, attrs.Name)
	}

	log.WithField("count", len(names)).Debugf("Listed GCS objects. prefix: %s", prefix)
	return names, nil
}

func (g *gcsReaderImpl) DownloadObject(ctx context.Context, objectPath string, dest io.Writer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "DownloadObject",
		"bucket":      g.bucketName,
		"object_path": objectPath,
	})

	reader, err := g.client.Bucket(g.bucketName).Object(objectPath).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("could not open GCS object %s: %w", objectPath, err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			log.WithError(closeErr).Warn("could not close GCS reader")
		}
	}()

	n, err := io.Copy(dest, reader)
	if err != nil {
		return fmt.Errorf("could not download GCS object %s: %w", objectPath, err)
	}

	log.WithField("bytes", n).Debugf("Downloaded GCS object. object_path: %s", objectPath)
	return nil
}
