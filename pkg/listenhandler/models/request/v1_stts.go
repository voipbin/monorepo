package request

import "github.com/gofrs/uuid"

// V1DataSTTsPost is
// v1 data type request struct for
// /v1/stts POST
type V1DataSTTsPost struct {
	ReferenceID uuid.UUID `json:"reference_id"`
}
