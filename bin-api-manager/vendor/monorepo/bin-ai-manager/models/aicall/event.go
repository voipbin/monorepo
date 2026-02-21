package aicall

// list of event types
const (
	EventTypeStatusInitializing string = "aicall_status_initializing" // the aicall status is initializing
	EventTypeStatusProgressing  string = "aicall_status_progressing"  // the aicall status is progressing
	EventTypeStatusPausing      string = "aicall_status_pausing"      // the aicall status is pausing
	EventTypeStatusResuming     string = "aicall_status_resuming"     // the aicall status is resuming
	EventTypeStatusTerminating  string = "aicall_status_terminating"  // the aicall status is terminating
	EventTypeStatusTerminated   string = "aicall_status_terminated"   // the aicall status is terminated
)
