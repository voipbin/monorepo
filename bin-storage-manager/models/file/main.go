package file

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// File defines
type File struct {
	commonidentity.Identity
	commonidentity.Owner

	AccountID uuid.UUID `json:"account_id" db:"account_id,uuid"`

	ReferenceType ReferenceType `json:"reference_type" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id" db:"reference_id,uuid"`

	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	BucketName string `json:"bucket_name" db:"bucket_name"` // bucket name for file storage
	Filename   string `json:"filename" db:"filename"`       // filename for file. because we are storing the file in bucket with the file's id, this points out the original filename.
	Filepath   string `json:"filepath" db:"filepath"`       // filepath for file.
	Filesize   int64  `json:"filesize" db:"filesize"`       // file size in bytes

	URIBucket   string `json:"uri_bucket" db:"uri_bucket"`     // uri for bucket
	URIDownload string `json:"uri_download" db:"uri_download"` // uri for download

	TMDownloadExpire string `json:"tm_download_expire" db:"tm_download_expire"`
	TMCreate         string `json:"tm_create" db:"tm_create"`
	TMUpdate         string `json:"tm_update" db:"tm_update"`
	TMDelete         string `json:"tm_delete" db:"tm_delete"`
}

// ReferenceType define
type ReferenceType string

// list of reference types
const (
	ReferenceTypeNone      ReferenceType = ""
	ReferenceTypeNormal    ReferenceType = "normal"
	ReferenceTypeRecording ReferenceType = "recording"
)
