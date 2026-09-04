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

	// Insight AI tool set expansion (NOJIRA, docs/plans/
	// 2026-07-30-case-insight-assistant-tool-expansion-design.md §1).
	ToolNameGetRelatedCases ToolName = "get_related_cases"
	ToolNameGetCaseNotes    ToolName = "get_case_notes"

	// Insight AI tool set expansion (NOJIRA, docs/plans/
	// 2026-09-02-insight-assistant-get-contact-profile-design.md).
	ToolNameGetContactProfile ToolName = "get_contact_profile"

	// Insight AI tool set expansion (VOIP-1453, docs/plans/
	// 2026-09-03-insight-assistant-get-call-transcript-design.md).
	ToolNameGetCallTranscript ToolName = "get_call_transcript"

	// Insight AI proactive notification (NOJIRA, docs/plans/
	// 2026-09-03-insight-ai-realtime-listen-design.md §5.5).
	ToolNameNotifyAgent ToolName = "notify_agent"
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
//
// Every entry must be read-only with respect to customer data and external
// systems. The single sanctioned exception is notify_agent, whose only effect is
// to write a message into the AIcall's own conversation thread -- the same
// thread the agent is already reading. It cannot place calls, send email or SMS,
// mutate CRM records, or spend money.
//
// See docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md §2.6
// and docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.5.2.
var AllInsightToolNames = []ToolName{
	ToolNameGetContactInteractions,
	ToolNameGetConversationContent,
	ToolNameGetRelatedCases,
	ToolNameGetCaseNotes,
	ToolNameGetContactProfile,
	ToolNameGetCallTranscript,
	ToolNameNotifyAgent,
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
