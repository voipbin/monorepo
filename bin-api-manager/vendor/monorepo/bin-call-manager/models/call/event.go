package call

// list of call event types
const (
	EventTypeCallCreated string = "call_created" // the call has created
	EventTypeCallUpdated string = "call_updated" // the call's info has updated
	EventTypeCallDeleted string = "call_deleted" // the call's info has deleted

	EventTypeCallDialing     string = "call_dialing"
	EventTypeCallRinging     string = "call_ringing"
	EventTypeCallProgressing string = "call_progressing"
	EventTypeCallTerminating string = "call_terminating"
	EventTypeCallCanceling   string = "call_canceling"
	EventTypeCallHangup      string = "call_hangup"
)
