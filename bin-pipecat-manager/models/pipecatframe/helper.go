package pipecatframe

type CommonFrameMessage struct {
	Label string `json:"label,omitempty"`
	Type  string `json:"type,omitempty"`
}

// type RTVIFrameType
const (
	RTVIFrameTypeBotTranscription  = "bot-transcription"
	RTVIFrameTypeUserTranscription = "user-transcription"
)
