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
//
// Every field is a pointer: nil means "leave the existing value
// untouched", matching the OpenAPI-layer contract this DTO carries over
// the RabbitMQ hop. See
// docs/plans/2026-09-12-route-put-partial-update-phase3-design.md.
type V1DataRoutesIDPut struct {
	Name       *string    `json:"name,omitempty"`
	Detail     *string    `json:"detail,omitempty"`
	ProviderID *uuid.UUID `json:"provider_id,omitempty"`
	Priority   *int       `json:"priority,omitempty"`
	Target     *string    `json:"target,omitempty"`
}

// V1DataRoutesGet is
// v1 data type request struct for
// /v1/routes GET
type V1DataRoutesGet struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
}
