package tool

// ToolName defines the name of a tool
type ToolName string

// Tool name constants
const (
	ToolNameAll               ToolName = "all"
	ToolNameConnectCall       ToolName = "connect_call"
	ToolNameGetVariables      ToolName = "get_variables"
	ToolNameGetAIcallMessages ToolName = "get_aicall_messages"
	ToolNameSendEmail         ToolName = "send_email"
	ToolNameSendMessage       ToolName = "send_message"
	ToolNameSetVariables      ToolName = "set_variables"
	ToolNameStopFlow          ToolName = "stop_flow"
	ToolNameStopMedia         ToolName = "stop_media"
	ToolNameStopService       ToolName = "stop_service"
)

// AllToolNames returns all available tool names (excluding "all")
var AllToolNames = []ToolName{
	ToolNameConnectCall,
	ToolNameGetVariables,
	ToolNameGetAIcallMessages,
	ToolNameSendEmail,
	ToolNameSendMessage,
	ToolNameSetVariables,
	ToolNameStopFlow,
	ToolNameStopMedia,
	ToolNameStopService,
}

// Tool defines a tool with its schema for LLM function calling
type Tool struct {
	Name        ToolName       `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}
