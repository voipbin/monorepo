package file

import "github.com/gofrs/uuid"

// File defines
type File struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerID    uuid.UUID `json:"owner_id"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	BucketName string `json:"bucket_name"` // bucket name for file storage
	Filepath   string `json:"filepath"`    // file path for file

	URIBucket   string `json:"uri_bucket"`   // uri for bucket
	URIDownload string `json:"uri_download"` // uri for download

	TMDownloadExpire string `json:"tm_download_expire"`
	TMCreate         string `json:"tm_create"`
	TMUpdate         string `json:"tm_update"`
	TMDelete         string `json:"tm_delete"`
}

// ReferenceType define
type ReferenceType string

// list of reference types
const (
	ReferenceTypeNone      ReferenceType = ""
	ReferenceTypeRecording ReferenceType = "recording"
)
