package pipecatframe

// RTVIProtocolVersion defines the current RTVI protocol version.
const RTVIProtocolVersion = "1.0.0"

// RTVIMessageLabel defines the label for RTVI messages.
const RTVIMessageLabel = "rtvi-ai"

// ActionResult represents the result of an RTVI action.
// In Go, this would typically be an interface{} to allow for various types.
type ActionResult interface{}

// Deprecated: Pipeline Configuration has been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIServiceOption struct {
	Name string `json:"name"`
	Type string `json:"type"` // "bool", "number", "string", "array", "object"
	// Handler field is a Python callable and has no direct Go equivalent in struct definition.
	// It would be implemented as a method on a Go struct or a separate function.
}

// Deprecated: Pipeline Configuration has been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIService struct {
	Name    string              `json:"name"`
	Options []RTVIServiceOption `json:"options"`
	// _options_dict is a private Python attribute and is not directly translated.
	// In Go, you might build a map from the Options slice if needed.
}

// Deprecated: Actions have been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIActionArgumentData struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// Deprecated: Actions have been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIActionArgument struct {
	Name string `json:"name"`
	Type string `json:"type"` // "bool", "number", "string", "array", "object"
}

// Deprecated: Actions have been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIAction struct {
	Service   string               `json:"service"`
	Action    string               `json:"action"`
	Arguments []RTVIActionArgument `json:"arguments"`
	Result    string               `json:"result"` // "bool", "number", "string", "array", "object"
	// Handler field is a Python callable and has no direct Go equivalent in struct definition.
}

// Deprecated: Pipeline Configuration has been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIServiceOptionConfig struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// Deprecated: Pipeline Configuration has been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIServiceConfig struct {
	Service string                    `json:"service"`
	Options []RTVIServiceOptionConfig `json:"options"`
}

// Deprecated: Pipeline Configuration has been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIConfig struct {
	Config []RTVIServiceConfig `json:"config"`
}

// Deprecated: Pipeline Configuration has been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIUpdateConfig struct {
	Config    []RTVIServiceConfig `json:"config"`
	Interrupt bool                `json:"interrupt"`
}

// Deprecated: Actions have been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIActionRunArgument struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// Deprecated: Actions have been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIActionRun struct {
	Service   string                  `json:"service"`
	Action    string                  `json:"action"`
	Arguments []RTVIActionRunArgument `json:"arguments,omitempty"`
}

// RTVIActionFrame is a logical frame, not directly for JSON serialization.
// It would be represented by a Go struct with its fields.
type RTVIActionFrame struct {
	RTVIActionRun RTVIActionRun
	MessageID     *string
}

// RTVIRawClientMessageData is the raw data structure for client messages.
type RTVIRawClientMessageData struct {
	Type string      `json:"t"`
	Data interface{} `json:"d,omitempty"`
}

// RTVIClientMessage is the cleansed data structure for client messages.
type RTVIClientMessage struct {
	MsgID string      `json:"msg_id"`
	Type  string      `json:"type"`
	Data  interface{} `json:"data,omitempty"`
}

// RTVIClientMessageFrame is a logical frame, not directly for JSON serialization.
type RTVIClientMessageFrame struct {
	MsgID string
	Type  string
	Data  interface{}
}

// RTVIServerResponseFrame is a logical frame, not directly for JSON serialization.
type RTVIServerResponseFrame struct {
	ClientMsg RTVIClientMessageFrame
	Data      interface{}
	Error     *string
}

// RTVIRawServerResponseData is the raw data structure for server responses.
type RTVIRawServerResponseData struct {
	Type string      `json:"t"`
	Data interface{} `json:"d,omitempty"`
}

// RTVIServerResponse is the RTVI-formatted message response from server to client.
type RTVIServerResponse struct {
	Label string                    `json:"label"` // RTVIMessageLabel
	Type  string                    `json:"type"`  // "server-response"
	ID    string                    `json:"id"`
	Data  RTVIRawServerResponseData `json:"data"`
}

// RTVIMessage is the base RTVI message structure.
type RTVIMessage struct {
	Label string                 `json:"label"` // RTVIMessageLabel
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

// RTVIErrorResponseData provides data for an RTVI error response.
type RTVIErrorResponseData struct {
	Error string `json:"error"`
}

// RTVIErrorResponse is an RTVI formatted error response message.
type RTVIErrorResponse struct {
	Label string                `json:"label"` // RTVIMessageLabel
	Type  string                `json:"type"`  // "error-response"
	ID    string                `json:"id"`
	Data  RTVIErrorResponseData `json:"data"`
}

// RTVIErrorData provides data for an RTVI error event.
type RTVIErrorData struct {
	Error string `json:"error"`
	Fatal bool   `json:"fatal"`
}

// RTVIError is an RTVI formatted error event message.
type RTVIError struct {
	Label string        `json:"label"` // RTVIMessageLabel
	Type  string        `json:"type"`  // "error"
	Data  RTVIErrorData `json:"data"`
}

// Deprecated: Pipeline Configuration has been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIDescribeConfigData struct {
	Config []RTVIService `json:"config"`
}

// Deprecated: Pipeline Configuration has been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIDescribeConfig struct {
	Label string                 `json:"label"` // RTVIMessageLabel
	Type  string                 `json:"type"`  // "config-available"
	ID    string                 `json:"id"`
	Data  RTVIDescribeConfigData `json:"data"`
}

// Deprecated: Actions have been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIDescribeActionsData struct {
	Actions []RTVIAction `json:"actions"`
}

// Deprecated: Actions have been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIDescribeActions struct {
	Label string                  `json:"label"` // RTVIMessageLabel
	Type  string                  `json:"type"`  // "actions-available"
	ID    string                  `json:"id"`
	Data  RTVIDescribeActionsData `json:"data"`
}

// Deprecated: Pipeline Configuration has been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIConfigResponse struct {
	Label string     `json:"label"` // RTVIMessageLabel
	Type  string     `json:"type"`  // "config"
	ID    string     `json:"id"`
	Data  RTVIConfig `json:"data"`
}

// Deprecated: Actions have been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIActionResponseData struct {
	Result ActionResult `json:"result"`
}

// Deprecated: Actions have been removed as part of the RTVI protocol 1.0.0.
// Use custom client and server messages instead.
type RTVIActionResponse struct {
	Label string                 `json:"label"` // RTVIMessageLabel
	Type  string                 `json:"type"`  // "action-response"
	ID    string                 `json:"id"`
	Data  RTVIActionResponseData `json:"data"`
}

// AboutClientData provides data about the RTVI client.
type AboutClientData struct {
	Library         string      `json:"library"`
	LibraryVersion  *string     `json:"library_version,omitempty"`
	Platform        *string     `json:"platform,omitempty"`
	PlatformVersion *string     `json:"platform_version,omitempty"`
	PlatformDetails interface{} `json:"platform_details,omitempty"`
}

// RTVIClientReadyData is the data format for client ready messages.
type RTVIClientReadyData struct {
	Version string          `json:"version"`
	About   AboutClientData `json:"about"`
}

// RTVIBotReadyData provides data for bot ready notification.
type RTVIBotReadyData struct {
	Version string                 `json:"version"`
	Config  []RTVIServiceConfig    `json:"config,omitempty"` // Deprecated field
	About   map[string]interface{} `json:"about,omitempty"`
}

// RTVIBotReady is a message indicating bot is ready for interaction.
type RTVIBotReady struct {
	Label string           `json:"label"` // RTVIMessageLabel
	Type  string           `json:"type"`  // "bot-ready"
	ID    string           `json:"id"`
	Data  RTVIBotReadyData `json:"data"`
}

// RTVILLMFunctionCallMessageData provides data for LLM function call notification.
type RTVILLMFunctionCallMessageData struct {
	FunctionName string                 `json:"function_name"`
	ToolCallID   string                 `json:"tool_call_id"`
	Args         map[string]interface{} `json:"args"`
}

// RTVILLMFunctionCallMessage is a message notifying of an LLM function call.
type RTVILLMFunctionCallMessage struct {
	Label string                         `json:"label"` // RTVIMessageLabel
	Type  string                         `json:"type"`  // "llm-function-call"
	Data  RTVILLMFunctionCallMessageData `json:"data"`
}

// RTVISendTextOptions provides options for sending text input to the LLM.
type RTVISendTextOptions struct {
	RunImmediately bool `json:"run_immediately"`
	AudioResponse  bool `json:"audio_response"`
}

// RTVISendTextData is the data format for sending text input to the LLM.
type RTVISendTextData struct {
	Content string               `json:"content"`
	Options *RTVISendTextOptions `json:"options,omitempty"`
}

// Deprecated: The RTVI message, append-to-context, has been deprecated. Use send-text
// or custom client and server messages instead.
type RTVIAppendToContextData struct {
	Role           string      `json:"role"` // "user", "assistant", or custom string
	Content        interface{} `json:"content"`
	RunImmediately bool        `json:"run_immediately"`
}

// RTVIAppendToContext is an RTVI message format to append content to the LLM context.
// Deprecated.
type RTVIAppendToContext struct {
	Label string                  `json:"label"` // RTVIMessageLabel
	Type  string                  `json:"type"`  // "append-to-context"
	Data  RTVIAppendToContextData `json:"data"`
}

// RTVILLMFunctionCallStartMessageData provides data for LLM function call start notification.
type RTVILLMFunctionCallStartMessageData struct {
	FunctionName string `json:"function_name"`
}

// RTVILLMFunctionCallStartMessage is a message notifying that an LLM function call has started.
type RTVILLMFunctionCallStartMessage struct {
	Label string                              `json:"label"` // RTVIMessageLabel
	Type  string                              `json:"type"`  // "llm-function-call-start"
	Data  RTVILLMFunctionCallStartMessageData `json:"data"`
}

// RTVILLMFunctionCallResultData provides data for LLM function call result.
type RTVILLMFunctionCallResultData struct {
	FunctionName string                 `json:"function_name"`
	ToolCallID   string                 `json:"tool_call_id"`
	Arguments    map[string]interface{} `json:"arguments"`
	Result       interface{}            `json:"result"` // Can be dict or string
}

// RTVIBotLLMStartedMessage indicates bot LLM processing has started.
type RTVIBotLLMStartedMessage struct {
	Label string `json:"label"` // RTVIMessageLabel
	Type  string `json:"type"`  // "bot-llm-started"
}

// RTVIBotLLMStoppedMessage indicates bot LLM processing has stopped.
type RTVIBotLLMStoppedMessage struct {
	Label string `json:"label"` // RTVIMessageLabel
	Type  string `json:"type"`  // "bot-llm-stopped"
}

// RTVIBotTTSStartedMessage indicates bot TTS processing has started.
type RTVIBotTTSStartedMessage struct {
	Label string `json:"label"` // RTVIMessageLabel
	Type  string `json:"type"`  // "bot-tts-started"
}

// RTVIBotTTSStoppedMessage indicates bot TTS processing has stopped.
type RTVIBotTTSStoppedMessage struct {
	Label string `json:"label"` // RTVIMessageLabel
	Type  string `json:"type"`  // "bot-tts-stopped"
}

// RTVITextMessageData provides data for text-based RTVI messages.
type RTVITextMessageData struct {
	Text string `json:"text"`
}

// RTVIBotTranscriptionMessage contains bot transcription text.
type RTVIBotTranscriptionMessage struct {
	Label string              `json:"label"` // RTVIMessageLabel
	Type  string              `json:"type"`  // "bot-transcription"
	Data  RTVITextMessageData `json:"data"`
}

// RTVIBotLLMTextMessage contains bot LLM text output.
type RTVIBotLLMTextMessage struct {
	Label string              `json:"label"` // RTVIMessageLabel
	Type  string              `json:"type"`  // "bot-llm-text"
	Data  RTVITextMessageData `json:"data"`
}

// RTVIBotTTSTextMessage contains bot TTS text output.
type RTVIBotTTSTextMessage struct {
	Label string              `json:"label"` // RTVIMessageLabel
	Type  string              `json:"type"`  // "bot-tts-text"
	Data  RTVITextMessageData `json:"data"`
}

// RTVIAudioMessageData provides data for audio-based RTVI messages.
type RTVIAudioMessageData struct {
	Audio       string `json:"audio"` // base64 encoded audio
	SampleRate  int    `json:"sample_rate"`
	NumChannels int    `json:"num_channels"`
}

// RTVIBotTTSAudioMessage contains bot TTS audio output.
type RTVIBotTTSAudioMessage struct {
	Label string               `json:"label"` // RTVIMessageLabel
	Type  string               `json:"type"`  // "bot-tts-audio"
	Data  RTVIAudioMessageData `json:"data"`
}

// RTVIUserTranscriptionMessageData provides data for user transcription messages.
type RTVIUserTranscriptionMessageData struct {
	Text      string `json:"text"`
	UserID    string `json:"user_id"`
	Timestamp string `json:"timestamp"` // ISO 8601 formatted string
	Final     bool   `json:"final"`
}

// RTVIUserTranscriptionMessage contains user transcription.
type RTVIUserTranscriptionMessage struct {
	Label string                           `json:"label"` // RTVIMessageLabel
	Type  string                           `json:"type"`  // "user-transcription"
	Data  RTVIUserTranscriptionMessageData `json:"data"`
}

// RTVIUserLLMTextMessage contains user text input for LLM.
type RTVIUserLLMTextMessage struct {
	Label string              `json:"label"` // RTVIMessageLabel
	Type  string              `json:"type"`  // "user-llm-text"
	Data  RTVITextMessageData `json:"data"`
}

// RTVIUserStartedSpeakingMessage indicates user has started speaking.
type RTVIUserStartedSpeakingMessage struct {
	Label string `json:"label"` // RTVIMessageLabel
	Type  string `json:"type"`  // "user-started-speaking"
}

// RTVIUserStoppedSpeakingMessage indicates user has stopped speaking.
type RTVIUserStoppedSpeakingMessage struct {
	Label string `json:"label"` // RTVIMessageLabel
	Type  string `json:"type"`  // "user-stopped-speaking"
}

// RTVIBotStartedSpeakingMessage indicates bot has started speaking.
type RTVIBotStartedSpeakingMessage struct {
	Label string `json:"label"` // RTVIMessageLabel
	Type  string `json:"type"`  // "bot-started-speaking"
}

// RTVIBotStoppedSpeakingMessage indicates bot has stopped speaking.
type RTVIBotStoppedSpeakingMessage struct {
	Label string `json:"label"` // RTVIMessageLabel
	Type  string `json:"type"`  // "bot-stopped-speaking"
}

// RTVIMetricsMessage contains performance metrics.
type RTVIMetricsMessage struct {
	Label string                 `json:"label"` // RTVIMessageLabel
	Type  string                 `json:"type"`  // "metrics"
	Data  map[string]interface{} `json:"data"`
}

// RTVIServerMessage is a generic server message for custom server-to-client messages.
type RTVIServerMessage struct {
	Label string      `json:"label"` // RTVIMessageLabel
	Type  string      `json:"type"`  // "server-message"
	Data  interface{} `json:"data"`
}

// RTVIAudioLevelMessageData is the data format for sending audio levels.
type RTVIAudioLevelMessageData struct {
	Value float64 `json:"value"`
}

// RTVIUserAudioLevelMessage indicates user audio level.
type RTVIUserAudioLevelMessage struct {
	Label string                    `json:"label"` // RTVIMessageLabel
	Type  string                    `json:"type"`  // "user-audio-level"
	Data  RTVIAudioLevelMessageData `json:"data"`
}

// RTVIBotAudioLevelMessage indicates bot audio level.
type RTVIBotAudioLevelMessage struct {
	Label string                    `json:"label"` // RTVIMessageLabel
	Type  string                    `json:"type"`  // "bot-audio-level"
	Data  RTVIAudioLevelMessageData `json:"data"`
}

// RTVISystemLogMessage includes a system log.
type RTVISystemLogMessage struct {
	Label string              `json:"label"` // RTVIMessageLabel
	Type  string              `json:"type"`  // "system-log"
	Data  RTVITextMessageData `json:"data"`
}

// RTVIServerMessageFrame is a logical frame, not directly for JSON serialization.
type RTVIServerMessageFrame struct {
	Data interface{}
}

// RTVIObserverParams defines parameters for configuring RTVI Observer behavior.
// Deprecated: Parameter `ErrorsEnabled` is deprecated. Error messages are always enabled.
type RTVIObserverParams struct {
	BotLLMEnabled            bool    `json:"bot_llm_enabled"`
	BotTTSEnabled            bool    `json:"bot_tts_enabled"`
	BotSpeakingEnabled       bool    `json:"bot_speaking_enabled"`
	BotAudioLevelEnabled     bool    `json:"bot_audio_level_enabled"`
	UserLLMEnabled           bool    `json:"user_llm_enabled"`
	UserSpeakingEnabled      bool    `json:"user_speaking_enabled"`
	UserTranscriptionEnabled bool    `json:"user_transcription_enabled"`
	UserAudioLevelEnabled    bool    `json:"user_audio_level_enabled"`
	MetricsEnabled           bool    `json:"metrics_enabled"`
	SystemLogsEnabled        bool    `json:"system_logs_enabled"`
	ErrorsEnabled            *bool   `json:"errors_enabled,omitempty"` // Deprecated
	AudioLevelPeriodSecs     float64 `json:"audio_level_period_secs"`
}

// RTVIObserver is a logical component for handling RTVI server messages.
// Its internal fields and methods are not directly translated as Go structs for JSON.
