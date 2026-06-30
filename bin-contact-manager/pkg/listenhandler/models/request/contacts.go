package request

import "github.com/gofrs/uuid"

// ContactCreate is the request body for POST /v1/contacts
type ContactCreate struct {
	CustomerID uuid.UUID `json:"customer_id"`

	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`
	Company     string `json:"company"`
	JobTitle    string `json:"job_title"`

	Source     string `json:"source"`
	ExternalID string `json:"external_id"`
	Notes      string `json:"notes"`

	Addresses []AddressCreate `json:"addresses,omitempty"`
	TagIDs    []uuid.UUID     `json:"tag_ids,omitempty"`
}

// ContactUpdate is the request body for PUT /v1/contacts/{id}
type ContactUpdate struct {
	FirstName   *string `json:"first_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Company     *string `json:"company,omitempty"`
	JobTitle    *string `json:"job_title,omitempty"`
	ExternalID  *string `json:"external_id,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

// AddressCreate is the body for POST /v1/contacts/{id}/addresses
type AddressCreate struct {
	Type      string `json:"type"`       // "tel" | "email" — required
	Target    string `json:"target"`     // E.164 or email   — required
	IsPrimary bool   `json:"is_primary"`
}

// AddressUpdate is the body for PUT /v1/contacts/{id}/addresses/{address_id}
type AddressUpdate struct {
	Target    *string `json:"target,omitempty"`
	IsPrimary *bool   `json:"is_primary,omitempty"`
}

// TagAssignment is the request body for POST /v1/contacts/{id}/tags
type TagAssignment struct {
	TagID uuid.UUID `json:"tag_id"`
}
