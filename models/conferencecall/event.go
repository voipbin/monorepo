package conferencecall

// list of event types
const (
	EventTypeConferencecallJoining string = "conferencecall_joining" // the conferencecall is joining the conference
	EventTypeConferencecallJoined  string = "conferencecall_joined"  // the conferencecall has joined
	EventTypeConferencecallLeaving string = "conferencecall_leaving" // the conferencecall is leaving the conference
	EventTypeConferencecallLeaved  string = "conferencecall_leaved"  // the conferencecall has leaved
)
