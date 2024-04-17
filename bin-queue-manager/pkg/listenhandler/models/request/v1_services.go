package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// V1DataServicesTypeQueuecallPost is
// v1 data type request struct for
// /v1/services/queuecall" POST
type V1DataServicesTypeQueuecallPost struct {
	QueueID       uuid.UUID               `json:"queue_id"`
	ActiveflowID  uuid.UUID               `json:"activeflow_id"`
	ReferenceType queuecall.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID               `json:"reference_id"`
	ExitActionID  uuid.UUID               `json:"exit_action_id"`
}
