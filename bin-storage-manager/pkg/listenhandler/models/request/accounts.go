package request

import "github.com/gofrs/uuid"

// V1DataAccountsPost is
// v1 data type request struct for
// /v1/accounts POST
type V1DataAccountsPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
}
