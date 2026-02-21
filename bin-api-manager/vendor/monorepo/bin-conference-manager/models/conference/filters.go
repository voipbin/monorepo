package conference

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Conference queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID            uuid.UUID `filter:"id"`
	CustomerID    uuid.UUID `filter:"customer_id"`
	ConfbridgeID  uuid.UUID `filter:"confbridge_id"`
	Type          Type      `filter:"type"`
	Status        Status    `filter:"status"`
	Name          string    `filter:"name"`
	PreFlowID     uuid.UUID `filter:"pre_flow_id"`
	PostFlowID    uuid.UUID `filter:"post_flow_id"`
	RecordingID   uuid.UUID `filter:"recording_id"`
	TranscribeID  uuid.UUID `filter:"transcribe_id"`
	Deleted       bool      `filter:"deleted"`
}
