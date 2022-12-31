package call

// list of call event types
const (
	EventTypeCallCreated string = "call_created" // the call has created
	EventTypeCallUpdated string = "call_updated" // the call's info has updated
	EventTypeCallDeleted string = "call_deleted" // the call's info has deleted

	EventTypeCallRinging  string = "call_ringing"  // the call is ringing
	EventTypeCallAnswered string = "call_answered" // the call has answred
	EventTypeCallHungup   string = "call_hungup"   // the call hungup
)
