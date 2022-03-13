package message

// list of message event types
const (
	EventTypeMessageCreated string = "message_created" // the message has created
	EventTypeMessageUpdated string = "message_updated" // the message's info has updated
	EventTypeMessageDeleted string = "message_ringing" // the message is ringing
)
