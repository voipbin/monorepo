package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// V1DataCasesIDClose is the request body for POST /v1/cases/{id}/close.
type V1DataCasesIDClose struct {
	CustomerID   uuid.UUID `json:"customer_id"`
	ClosedByType string    `json:"closed_by_type"`
	ClosedByID   uuid.UUID `json:"closed_by_id"`
}

// V1DataCasesIDAssign is the request body for POST /v1/cases/{id}/assign.
type V1DataCasesIDAssign struct {
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerType  string    `json:"owner_type"`
	OwnerID    uuid.UUID `json:"owner_id"`
}

// V1DataCasesIDContinue is the request body for
// POST /v1/cases/{id}/continue.
type V1DataCasesIDContinue struct {
	CustomerID    uuid.UUID `json:"customer_id"`
	CallerType    string    `json:"caller_type"`
	CallerID      uuid.UUID `json:"caller_id"`
	CallerIsAdmin bool      `json:"caller_is_admin"`
}

// V1DataCasesIDGet is the request body for GET /v1/cases/{id}.
// customer_id is read from the JSON request body (consistent with other
// GET handlers).
type V1DataCasesIDGet struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

// V1DataCasesIDNotesGet is the request body for
// GET /v1/cases/{id}/notes.
type V1DataCasesIDNotesGet struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

// V1DataCasesIDNotesPost is the request body for
// POST /v1/cases/{id}/notes.
type V1DataCasesIDNotesPost struct {
	CustomerID uuid.UUID  `json:"customer_id"`
	AuthorType string     `json:"author_type"`
	AuthorID   *uuid.UUID `json:"author_id,omitempty"`
	Text       string     `json:"text"`
}

// V1DataCasesIDNotesIDDelete is the request body for
// DELETE /v1/cases/{id}/notes/{note_id}.
type V1DataCasesIDNotesIDDelete struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

// V1DataCasesIDTagsGet is the request body for GET /v1/cases/{id}/tags.
type V1DataCasesIDTagsGet struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

// V1DataCasesIDTagsPost is the request body for
// POST /v1/cases/{id}/tags.
type V1DataCasesIDTagsPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	TagID      uuid.UUID `json:"tag_id"`
}

// V1DataCasesIDTagsIDDelete is the request body for
// DELETE /v1/cases/{id}/tags/{tag_id}.
type V1DataCasesIDTagsIDDelete struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

// V1DataCasesUnresolvedGet is the request body for
// GET /v1/cases/unresolved.
type V1DataCasesUnresolvedGet struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

// V1DataCasesGet is the request body for GET /v1/cases?...
type V1DataCasesGet struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

// V1DataCasesPost is the request body for POST /v1/cases.
type V1DataCasesPost struct {
	CustomerID    uuid.UUID             `json:"customer_id"`
	Self          commonaddress.Address `json:"self"`
	Peer          commonaddress.Address `json:"peer"`
	ReferenceType string                `json:"reference_type"`
	Name          string                `json:"name,omitempty"`
	Detail        string                `json:"detail,omitempty"`
	ReferenceID   string                `json:"reference_id,omitempty"`
}
