package bucketrecording

import "github.com/gofrs/uuid"

// BucketRecording struct
type BucketRecording struct {
	RecordingID    uuid.UUID `json:"reference_id"`
	BucketURI      string    `json:"bucket_uri"`      // bucket uri. gs://...
	DownloadURI    string    `json:"download_uri"`    // download link
	DownloadExpire string    `json:"download_expire"` // download link expire date
}
