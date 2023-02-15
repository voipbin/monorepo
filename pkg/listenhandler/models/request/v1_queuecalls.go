package request

import (
	"github.com/gofrs/uuid"
)

// V1DataQueuecallsIDExecutePost is
// v1 data type request struct for
// /v1/queuecalls/<queuecall-id>/execute POST
type V1DataQueuecallsIDExecutePost struct {
	AgentID uuid.UUID `json:"agent_id"`
}
