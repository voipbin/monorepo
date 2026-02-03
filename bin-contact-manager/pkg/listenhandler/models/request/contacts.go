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

	PhoneNumbers []PhoneNumberCreate `json:"phone_numbers,omitempty"`
	Emails       []EmailCreate       `json:"emails,omitempty"`
	TagIDs       []uuid.UUID         `json:"tag_ids,omitempty"`
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

// PhoneNumberCreate is the request body for POST /v1/contacts/{id}/phone-numbers
type PhoneNumberCreate struct {
	Number     string `json:"number"`
	NumberE164 string `json:"number_e164"`
	Type       string `json:"type"`
	IsPrimary  bool   `json:"is_primary"`
}

// EmailCreate is the request body for POST /v1/contacts/{id}/emails
type EmailCreate struct {
	Address   string `json:"address"`
	Type      string `json:"type"`
	IsPrimary bool   `json:"is_primary"`
}

// TagAssignment is the request body for POST /v1/contacts/{id}/tags
type TagAssignment struct {
	TagID uuid.UUID `json:"tag_id"`
}
