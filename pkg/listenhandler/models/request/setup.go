package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
)

// V1DataSetupPost is
// v1 data type request struct for
// /v1/setup POST
type V1DataSetupPost struct {
	CustomerID    uuid.UUID                  `json:"customer_id"`
	ReferenceType conversation.ReferenceType `json:"reference_type"`
}
