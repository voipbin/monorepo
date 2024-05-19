package request

import (
	"monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
)

// V1DataFilesPost is
// v1 data type request struct for
// /v1/flows POST
type V1DataFilesPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerID    uuid.UUID `json:"owner_id"`

	ReferenceType file.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID          `json:"reference_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	BucketName string `json:"bucket_name"`
	Filepath   string `json:"filepath"`
}
