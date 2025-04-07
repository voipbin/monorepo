package file

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// File defines
type File struct {
	commonidentity.Identity
	commonidentity.Owner

	AccountID uuid.UUID `json:"account_id"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	BucketName string `json:"bucket_name"` // bucket name for file storage
	Filename   string `json:"filename"`    // filename for file. because we are storing the file in bucket with the file's id, this points out the original filename.
	Filepath   string `json:"filepath"`    // filepath for file.
	Filesize   int64  `json:"filesize"`    // file size in bytes

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
	ReferenceTypeNormal    ReferenceType = "normal"
	ReferenceTypeRecording ReferenceType = "recording"
)
