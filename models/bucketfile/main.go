package bucketfile

import "github.com/gofrs/uuid"

// BucketFile struct
type BucketFile struct {
	ReferenceType    ReferenceType `json:"reference_type"`
	ReferenceID      uuid.UUID     `json:"reference_id"`
	BucketURI        string        `json:"bucket_uri"`         // bucket uri.
	DownloadURI      string        `json:"download_uri"`       // download link
	TMDownloadExpire string        `json:"tm_download_expire"` // timestamp for download link expire
}

// ReferenceType define
type ReferenceType string

// list of reference types
const (
	ReferenceTypeRecording ReferenceType = "recording"
)
