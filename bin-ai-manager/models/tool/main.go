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
	ToolNameListQueues        ToolName = "list_queues"
	ToolNameJoinQueue         ToolName = "join_queue"

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

	// Insight AI tool set expansion (VOIP-1455, docs/plans/
	// 2026-09-04-insight-assistant-emit-info-card-design.md).
	ToolNameEmitInfoCard ToolName = "emit_info_card"

	// Insight AI proactive notification (NOJIRA, docs/plans/
	// 2026-09-03-insight-ai-realtime-listen-design.md §5.5).
	ToolNameNotifyAgent ToolName = "notify_agent"

	// Insight AI SMS draft (VOIP-1704, docs/plans/
	// 2026-11-10-insight-assistant-sms-design.md).
	ToolNameDraftMessage ToolName = "draft_message"

	// Insight AI email draft (VOIP-1710, docs/plans/
	// 2026-11-11-insight-assistant-email-design.md).
	ToolNameDraftEmail ToolName = "draft_email"

	// Insight AI call draft (VOIP-1717, docs/plans/
	// 2026-11-11-insight-assistant-call-design.md).
	ToolNameDraftCall ToolName = "draft_call"
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
	ToolNameListQueues,
	ToolNameJoinQueue,
}

// AllInsightToolNames defines the tool set available to ai.TypeInsight AIs.
//
// Every entry MUST have no side effects outside the session's own
// message/expression surface -- see
// docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md §2.6
// (reworded from "must be read-only" by
// docs/plans/2026-09-04-insight-assistant-emit-info-card-design.md §1.4:
// emit_info_card writes a message into its own session's stream, the same
// surface a plain assistant-text turn already writes to, so a literal
// "read-only" reading would incorrectly exclude it).
//
// notify_agent is the same shape and is sanctioned on the same grounds
// (docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.5.2): its
// only effect is a message in the AIcall's own conversation thread -- the
// thread the agent is already reading. Neither tool can place calls, send
// email or SMS, mutate CRM records, or spend money.
//
// Sending an SMS or an email is not part of this list, so that this list
// stays free of outside side effects. The SMS tools live in
// InsightMessageToolNames and the email tools in InsightEmailToolNames below;
// the "all" selector of an Insight AI covers all lists. The call tools live in
// InsightCallToolNames. CRM writes remain unavailable to Insight AIs.
var AllInsightToolNames = []ToolName{
	ToolNameGetContactInteractions,
	ToolNameGetConversationContent,
	ToolNameGetRelatedCases,
	ToolNameGetCaseNotes,
	ToolNameGetContactProfile,
	ToolNameGetCallTranscript,
	ToolNameEmitInfoCard,
	ToolNameNotifyAgent,
}

// InsightMessageToolNames lists the tools with which an ai.TypeInsight AI
// sends or proposes an SMS (VOIP-1704). They are accepted by name by
// ValidateToolNames and, since VOIP-1708, are also produced by the "all"
// selector together with AllInsightToolNames (ai.ExpandAllToolNames).
//
// This list is deliberately separate from AllInsightToolNames: send_message
// has outside side effects (an SMS or MMS is sent at once and the cost is
// charged to the customer account), so it must not pass through
// TestAllInsightToolNamesAreReadOnly. Test_InsightMessageToolNames pins the
// exact names, so adding to this list is a deliberate change.
//
// send_message is the existing SMS/MMS tool, reused as is. draft_message only
// shows an SMS draft card to the agent and sends nothing.
var InsightMessageToolNames = []ToolName{
	ToolNameSendMessage,
	ToolNameDraftMessage,
}

// InsightEmailToolNames lists the tools with which an ai.TypeInsight AI sends
// or proposes an email (VOIP-1710). Like InsightMessageToolNames they are
// accepted by name by ValidateToolNames and are produced by the "all" selector
// together with the other Insight lists (ai.ExpandAllToolNames).
//
// This list is deliberately separate from AllInsightToolNames: send_email has
// outside side effects (an email is sent at once from the platform's default
// sender address and the cost is charged to the customer account), so it must
// not pass through TestAllInsightToolNamesAreReadOnly.
// Test_InsightEmailToolNames pins the exact names, so adding to this list is a
// deliberate change.
//
// send_email is the existing email tool, reused as is. draft_email only shows
// an email draft card to the agent and sends nothing. draft_email is not in
// AllToolNames, so a Normal AI cannot use it.
var InsightEmailToolNames = []ToolName{
	ToolNameSendEmail,
	ToolNameDraftEmail,
}

// InsightCallToolNames lists the tools with which an ai.TypeInsight AI places
// or proposes an outbound call (VOIP-1717). Like the other Insight lists they
// are accepted by name by ValidateToolNames and are produced by the "all"
// selector (ai.ExpandAllToolNames).
//
// This list is deliberately separate from AllInsightToolNames: create_call has
// outside side effects (a call is placed at once and the minutes are charged to
// the customer account), so it must not pass through
// TestAllInsightToolNamesAreReadOnly. Test_InsightCallToolNames pins the exact
// names, so adding to this list is a deliberate change.
//
// create_call is the existing tool, reused as is. draft_call only shows a call
// draft card to the agent and places nothing; it is not in AllToolNames, so a
// Normal AI cannot use it. describe_action is read-only and is the tool that
// create_call's "actions" argument tells the model to use; it is also in
// AllToolNames.
var InsightCallToolNames = []ToolName{
	ToolNameCreateCall,
	ToolNameDraftCall,
	ToolNameDescribeAction,
}

// Tool defines a tool with its schema for LLM function calling.
// RunLLM is a metadata default that tells the Python Pipecat runner whether
// to feed the tool result back into the LLM for response generation.
// The LLM can still override this per-call via a "run_llm" argument.
type Tool struct {
	Name        ToolName       `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	RunLLM      bool           `json:"run_llm"`
}
