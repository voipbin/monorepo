package request

import "github.com/gofrs/uuid"

// V1DataRoutesPost is
// v1 data type request struct for
// /v1/routes POST
type V1DataRoutesPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Name       string    `json:"name"`
	Detail     string    `json:"detail"`
	ProviderID uuid.UUID `json:"provider_id"`
	Priority   int       `json:"priority"`
	Target     string    `json:"target"`
}

// V1DataRoutesIDPut is
// v1 data type request struct for
// /v1/routes/{id} PUT
type V1DataRoutesIDPut struct {
	Name       string    `json:"name"`
	Detail     string    `json:"detail"`
	ProviderID uuid.UUID `json:"provider_id"`
	Priority   int       `json:"priority"`
	Target     string    `json:"target"`
}
