package request

import "github.com/gofrs/uuid"

// V1DataTagsPost is
// v1 data type request struct for
// /v1/tags POST
type V1DataTagsPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Name       string    `json:"name"`
	Detail     string    `json:"detail"`
}

// V1DataTagsIDPut is
// v1 data type request struct for
// /v1/tags/<tag-id> PUT
type V1DataTagsIDPut struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}
