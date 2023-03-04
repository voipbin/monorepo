package queuecall

// list of call queuecall types
const (
	EventTypeQueuecallCreated    string = "queuecall_created"    // the queuecall has created
	EventTypeQueuecallWaiting    string = "queuecall_waiting"    // the queuecall is waiting for agent
	EventTypeQueuecallConnecting string = "queuecall_connecting" // the queuecall is entering to the queue conference
	EventTypeQueuecallServiced   string = "queuecall_serviced"   //
	EventTypeQueuecallDone       string = "queuecall_done"
	EventTypeQueuecallAbandoned  string = "queuecall_abandoned"
	EventTypeQueuecallDeleted    string = "queuecall_deleted"
)
