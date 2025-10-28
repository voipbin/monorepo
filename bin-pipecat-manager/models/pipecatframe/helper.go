package pipecatframe

type CommonFrameMessage struct {
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
	Type  string `json:"type,omitempty"`
	Data  any    `json:"data,omitempty"`
}

// type RTVIFrameType
const (
	RTVIFrameTypeBotTranscription  = "bot-transcription"  // pipecat -> pipecat-manager event frame type.
	RTVIFrameTypeUserTranscription = "user-transcription" // pipecat -> pipecat-manager event frame type.

	RTVIFrameTypeSendText = "send-text" // pipecat-manager -> pipecat request frame type.
)
