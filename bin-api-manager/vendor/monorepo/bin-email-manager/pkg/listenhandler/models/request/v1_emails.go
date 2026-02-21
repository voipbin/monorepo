package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-email-manager/models/email"

	"github.com/gofrs/uuid"
)

// V1DataEmailsPost is
// data type request struct for
// /v1/emails POST
type V1DataEmailsPost struct {
	CustomerID   uuid.UUID               `json:"customer_id"`
	ActiveflowID uuid.UUID               `json:"activeflow_id"`
	Destinations []commonaddress.Address `json:"destinations"`
	Subject      string                  `json:"subject"`
	Content      string                  `json:"content"`
	Attachments  []email.Attachment      `json:"attachments"`
}
