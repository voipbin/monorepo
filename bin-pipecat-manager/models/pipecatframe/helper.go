package pipecatframe

type CommonFrameMessage struct {
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
	Type  string `json:"type,omitempty"`
	Data  any    `json:"data,omitempty"`
}

// type RTVIFrameType
const (
	// pipecat -> pipecat-manager event frame types
	RTVIFrameTypeBotTranscription    = "bot-transcription"
	RTVIFrameTypeUserTranscription   = "user-transcription"
	RTVIFrameTypeBotLLMText          = "bot-llm-text"
	RTVIFrameTypeBotLLMStarted       = "bot-llm-started"
	RTVIFrameTypeBotLLMStopped       = "bot-llm-stopped"
	RTVIFrameTypeBotTTSStarted       = "bot-tts-started"
	RTVIFrameTypeBotTTSStopped       = "bot-tts-stopped"
	RTVIFrameTypeUserStartedSpeaking = "user-started-speaking"
	RTVIFrameTypeUserStoppedSpeaking = "user-stopped-speaking"
	RTVIFrameTypeBotStartedSpeaking  = "bot-started-speaking"
	RTVIFrameTypeBotStoppedSpeaking  = "bot-stopped-speaking"
	RTVIFrameTypeMetrics             = "metrics"

	// pipecat-manager -> pipecat request frame types
	RTVIFrameTypeUserLLMText = "user-llm-text"
	RTVIFrameTypeSendText    = "send-text"
)
