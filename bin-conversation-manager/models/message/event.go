package message

// list of message event types
const (
	EventTypeMessageCreated string = "conversation_message_created" // the conversation message created.
	EventTypeMessageUpdated string = "conversation_message_updated" // the conversation message updated.
	EventTypeMessageDeleted string = "conversation_message_deleted" // the conversation message deleted.
)
