package tool

// ToolName defines the name of a tool
type ToolName string

// Tool name constants
const (
	ToolNameAll               ToolName = "all"
	ToolNameConnectCall       ToolName = "connect_call"
	ToolNameCreateCall        ToolName = "create_call"
	ToolNameGetVariables      ToolName = "get_variables"
	ToolNameGetAIcallMessages ToolName = "get_aicall_messages"
	ToolNameSendEmail         ToolName = "send_email"
	ToolNameSendMessage       ToolName = "send_message"
	ToolNameSetVariables      ToolName = "set_variables"
	ToolNameStopFlow          ToolName = "stop_flow"
	ToolNameStopMedia         ToolName = "stop_media"
	ToolNameStopService       ToolName = "stop_service"
	ToolNameSearchKnowledge   ToolName = "search_knowledge"
	ToolNameGetCorrelation    ToolName = "get_correlation"
	ToolNameGetResource       ToolName = "get_resource"
	ToolNameDescribeAction    ToolName = "describe_action"
	ToolNameCaseCreate        ToolName = "case_create"

	// Insight AI tool set (VOIP-1234).
	ToolNameGetContactInteractions ToolName = "get_contact_interactions"
	ToolNameGetConversationContent ToolName = "get_conversation_content"
)

// AllToolNames returns all available tool names (excluding "all")
var AllToolNames = []ToolName{
	ToolNameConnectCall,
	ToolNameCreateCall,
	ToolNameGetVariables,
	ToolNameGetAIcallMessages,
	ToolNameSendEmail,
	ToolNameSendMessage,
	ToolNameSetVariables,
	ToolNameStopFlow,
	ToolNameStopMedia,
	ToolNameStopService,
	ToolNameSearchKnowledge,
	ToolNameGetCorrelation,
	ToolNameGetResource,
	ToolNameDescribeAction,
	ToolNameCaseCreate,
}

// AllInsightToolNames defines the tool set available to ai.TypeInsight AIs.
var AllInsightToolNames = []ToolName{
	ToolNameGetContactInteractions,
	ToolNameGetConversationContent,
}

// Tool defines a tool with its schema for LLM function calling.
// RunLLM is a metadata default that tells the Python Pipecat runner whether
// to feed the tool result back into the LLM for response generation.
// The LLM can still override this per-call via a "run_llm" argument.
type Tool struct {
	Name        ToolName       `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	RunLLM      bool           `json:"run_llm"`
}
