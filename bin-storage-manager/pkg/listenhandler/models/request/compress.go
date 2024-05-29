package request

import (
	"github.com/gofrs/uuid"
)

// V1DataCompressPost is
// v1 data type request struct for
// /v1/compress POST
type V1DataCompressPost struct {
	ReferenceIDs []uuid.UUID `json:"reference_ids"`
	FileIDs      []uuid.UUID `json:"file_ids"`
}
