package transcribe

// list of transcribe event types
const (
	EventTypeTranscribeCreated string = "transcribe_created"
	EventTypeTranscribeDeleted string = "transcribe_deleted"

	EventTypeTranscribeProgressing string = "transcribe_progressing"
	EventTypeTranscribeDone        string = "transcribe_done"
)
