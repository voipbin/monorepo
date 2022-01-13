package confbridge

// list of confbridge event types
const (
	EventTypeConfbridgeCreated string = "confbridge_created" // confbridge created
	EventTypeConfbridgeDeleted string = "confbridge_deleted" // confbridge deleted
	EventTypeConfbridgeJoined  string = "confbridge_joined"  // the call has joined to the confbridge
	EventTypeConfbridgeLeaved  string = "confbridge_leaved"  // the call has left from the confbridge
)
