package streaming

// list of streaming event types
const (
	EventTypeStreamingStarted string = "streaming_started"
	EventTypeStreamingStopped string = "streaming_stopped"

	EventTypeSpeechStarted string = "transcribe_speech_started"
	EventTypeSpeechInterim string = "transcribe_speech_interim"
	EventTypeSpeechEnded   string = "transcribe_speech_ended"
)
