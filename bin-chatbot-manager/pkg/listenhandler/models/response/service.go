package response

import (
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// V1ResponseServicesChatbotcallPost is
// v1 response type for
// /v1/services/chatbotcall POST
type V1ResponseServicesChatbotcallPost struct {
	ServiceID   uuid.UUID         `json:"service_id"`   // represent started service id "service_id": "e53c3df6-4f85-4714-9980-1cca63caf4f6"
	ServiceType string            `json:"service_type"` // represent started service type. "service_type": "chatbotcall"
	Actions     []fmaction.Action `json:"actions"`      // represent the push actions for the service requested resource.
}
