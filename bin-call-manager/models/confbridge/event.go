package confbridge

import "github.com/gofrs/uuid"

// list of confbridge event types
const (
	EventTypeConfbridgeCreated     string = "confbridge_created" // confbridge created
	EventTypeConfbridgeDeleted     string = "confbridge_deleted" // confbridge deleted
	EventTypeConfbridgeTerminating string = "confbridge_terminating"
	EventTypeConfbridgeTerminated  string = "confbridge_terminated" // confbridge terminated
	EventTypeConfbridgeJoined      string = "confbridge_joined"     // EventConfbridgeJoined, the call has joined to the confbridge
	EventTypeConfbridgeLeaved      string = "confbridge_leaved"     // EventConfbridgeLeaved, the call has left from the confbridge
)

// EventConfbridgeLeaved event struct for confbridge leaved
type EventConfbridgeLeaved struct {
	Confbridge
	LeavedCallID uuid.UUID `json:"leaved_call_id"`
}

// EventConfbridgeJoined event struct for confbridge joined
type EventConfbridgeJoined struct {
	Confbridge
	JoinedCallID uuid.UUID `json:"joined_call_id"`
}
