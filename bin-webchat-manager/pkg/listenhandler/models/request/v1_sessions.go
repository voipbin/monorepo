package request

import (
	"github.com/gofrs/uuid"
)

// V1DataSessionsPost is
// v1 data type request struct for
// /v1/sessions POST
type V1DataSessionsPost struct {
	// ID pins the session's primary key. Zero means "generate one". Only the
	// direct-token path pins it. omitempty is inert on a uuid.UUID, so this
	// always serializes as the zero UUID; that is fine and intentional.
	ID uuid.UUID `json:"id,omitempty"`

	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	WidgetID   uuid.UUID `json:"widget_id,omitempty"`
	PageURL    string    `json:"page_url,omitempty"`
	Referrer   string    `json:"referrer,omitempty"`
}
