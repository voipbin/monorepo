package request

import "github.com/gofrs/uuid"

// V1DataContactsIDCaseNotesGet is the request body for
// GET /v1/contacts/{id}/case_notes. customer_id is read from the JSON
// request body (consistent with the other GET handlers); page_size and
// page_token are query parameters.
type V1DataContactsIDCaseNotesGet struct {
	CustomerID uuid.UUID `json:"customer_id"`
}
