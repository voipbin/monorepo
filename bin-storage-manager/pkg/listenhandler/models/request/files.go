package request

import (
	"monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
)

// V1DataFilesPost is
// v1 data type request struct for
// /v1/files POST
type V1DataFilesPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerID    uuid.UUID `json:"owner_id"`

	ReferenceType file.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID          `json:"reference_id"`

	Type file.Type `json:"type"`

	Name     string `json:"name"`
	Detail   string `json:"detail"`
	Filename string `json:"filename"`

	BucketName string `json:"bucket_name"`
	Filepath   string `json:"filepath"`
}

// V1DataFilesContentPost is
// v1 data type request struct for
// /v1/files/content POST
//
// Content travels as a base64 string in the JSON body.
type V1DataFilesContentPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerID    uuid.UUID `json:"owner_id"`

	Type file.Type `json:"type"`

	Name        string `json:"name"`
	Detail      string `json:"detail"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`

	Content []byte `json:"content"`
}
