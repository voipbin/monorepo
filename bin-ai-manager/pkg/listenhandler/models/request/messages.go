package request

import (
	"monorepo/bin-ai-manager/models/message"

	"github.com/gofrs/uuid"
)

// V1DataMessagesPost is
// v1 data type request struct for
// /v1/messages POST
type V1DataMessagesPost struct {
	AIcallID uuid.UUID    `json:"aicall_id,omitempty"`
	Role     message.Role `json:"role,omitempty"`
	Content  string       `json:"content,omitempty"`

	ReturnRespnse bool `json:"return_response,omitempty"` // if true, it will return the response message
}
