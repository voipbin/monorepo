package message

type ToolCall struct {
	ID       string       `json:"id,omitempty"`
	Type     ToolType     `json:"type"`
	Function FunctionCall `json:"function"`
}

type ToolType string

const (
	ToolTypeFunction ToolType = "function"
)

type FunctionCall struct {
	Name      FunctionCallName `json:"name,omitempty"`
	Arguments string           `json:"arguments,omitempty"`
}

type ToolResponse struct {
	ID      string
	Content string
}

type FunctionCallName string

const (
	FunctionCallNameNone FunctionCallName = ""

	FunctionCallNameConnect      FunctionCallName = "connect"
	FunctionCallNameEmailSend    FunctionCallName = "email_send"
	FunctionCallNameGetVariables FunctionCallName = "get_variables"
	FunctionCallNameMediaStop    FunctionCallName = "media_stop"
	FunctionCallNameMessageSend  FunctionCallName = "message_send"
	FunctionCallNameServiceStop  FunctionCallName = "service_stop"
	FunctionCallNameSetVariables FunctionCallName = "set_variables"
	FunctionCallNameStop         FunctionCallName = "stop"
)
