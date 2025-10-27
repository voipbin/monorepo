package pipecatframe

type CommonFrameMessage struct {
	Label string `json:"label,omitempty"`
	Type  string `json:"type,omitempty"`
	Data  any    `json:"data,omitempty"`
}

// type RTVIFrameType
const (
	RTVIFrameTypeBotTranscription  = "bot-transcription"
	RTVIFrameTypeUserTranscription = "user-transcription"

	RTVIFrameTypeSendText = "send-text"
)
