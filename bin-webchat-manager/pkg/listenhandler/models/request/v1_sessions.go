package request

import (
	"github.com/gofrs/uuid"
)

// V1DataSessionsPost is
// v1 data type request struct for
// /v1/sessions POST
type V1DataSessionsPost struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	WidgetID   uuid.UUID `json:"widget_id,omitempty"`
}
