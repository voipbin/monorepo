package queuecall

// list of call queuecall types
const (
	EventTypeQueuecallCreated   string = "queuecall_created"  // the queuecall has created
	EventTypeQueuecallEntering  string = "queuecall_entering" // the queuecall is entering to the queue confbridge
	EventTypeQueuecallKicking   string = "queuecall_kicking"  // the queuecall is being kicked
	EventTypeQueuecallServiced  string = "queuecall_serviced" //
	EventTypeQueuecallDone      string = "queuecall_done"
	EventTypeQueuecallAbandoned string = "queuecall_abandoned"
)
