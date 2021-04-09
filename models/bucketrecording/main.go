package bucketrecording

import "github.com/gofrs/uuid"

// BucketRecording struct
type BucketRecording struct {
	RecordingID      uuid.UUID `json:"reference_id"`
	BucketURI        string    `json:"bucket_uri"`         // bucket uri. gs://...
	DownloadURI      string    `json:"download_uri"`       // download link
	TMDownloadExpire string    `json:"tm_download_expire"` // timestamp for download link expire
}
