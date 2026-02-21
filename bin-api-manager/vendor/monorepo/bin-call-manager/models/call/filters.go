package call

import (
	"github.com/gofrs/uuid"
)

// FieldStruct defines allowed filters for Call queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	CustomerID   uuid.UUID `filter:"customer_id"`
	OwnerID      uuid.UUID `filter:"owner_id"`
	ChannelID    string    `filter:"channel_id"`
	BridgeID     string    `filter:"bridge_id"`
	FlowID       uuid.UUID `filter:"flow_id"`
	ActiveflowID uuid.UUID `filter:"activeflow_id"`
	ConfbridgeID uuid.UUID `filter:"confbridge_id"`
	Type         Type      `filter:"type"`
	RecordingID  uuid.UUID `filter:"recording_id"`
	GroupcallID  uuid.UUID `filter:"groupcall_id"`
	Status       Status    `filter:"status"`
	Direction    Direction `filter:"direction"`
	Deleted      bool      `filter:"deleted"`
}
