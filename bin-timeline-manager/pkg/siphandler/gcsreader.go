package siphandler

//go:generate mockgen -package siphandler -destination ./mock_gcsreader.go -source gcsreader.go -build_flags=-mod=mod

import (
	"context"
	"io"
)

// GCSReader provides read access to GCS objects for fetching RTP pcap files.
type GCSReader interface {
	// ListObjects lists object names in the bucket matching the given prefix.
	ListObjects(ctx context.Context, prefix string) ([]string, error)

	// DownloadObject downloads a GCS object and writes its content to dest.
	DownloadObject(ctx context.Context, objectPath string, dest io.Writer) error
}
