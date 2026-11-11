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

	FunctionCallNameConnectCall       FunctionCallName = "connect_call"
	FunctionCallNameCreateCall        FunctionCallName = "create_call"
	FunctionCallNameGetVariables      FunctionCallName = "get_variables"
	FunctionCallNameGetAIcallMessages FunctionCallName = "get_aicall_messages"
	FunctionCallNameSendEmail         FunctionCallName = "send_email"
	FunctionCallNameSendMessage       FunctionCallName = "send_message"
	FunctionCallNameSetVariables      FunctionCallName = "set_variables"
	FunctionCallNameStopMedia         FunctionCallName = "stop_media"
	FunctionCallNameStopFlow          FunctionCallName = "stop_flow"
	FunctionCallNameStopService       FunctionCallName = "stop_service"
	FunctionCallNameSearchKnowledge   FunctionCallName = "search_knowledge"
	FunctionCallNameGetCorrelation    FunctionCallName = "get_correlation"
	FunctionCallNameGetResource       FunctionCallName = "get_resource"
	FunctionCallNameDescribeAction    FunctionCallName = "describe_action"
	FunctionCallNameCaseCreate        FunctionCallName = "case_create"
	FunctionCallNameListQueues        FunctionCallName = "list_queues"
	FunctionCallNameJoinQueue         FunctionCallName = "join_queue"

	FunctionCallNameGetContactInteractions FunctionCallName = "get_contact_interactions"
	FunctionCallNameGetConversationContent FunctionCallName = "get_conversation_content"
	FunctionCallNameGetRelatedCases        FunctionCallName = "get_related_cases"
	FunctionCallNameGetCaseNotes           FunctionCallName = "get_case_notes"
	FunctionCallNameGetContactProfile      FunctionCallName = "get_contact_profile"
	FunctionCallNameGetCallTranscript      FunctionCallName = "get_call_transcript"
	FunctionCallNameEmitInfoCard           FunctionCallName = "emit_info_card"
	FunctionCallNameNotifyAgent            FunctionCallName = "notify_agent"
	FunctionCallNameDraftMessage           FunctionCallName = "draft_message"
	FunctionCallNameDraftEmail             FunctionCallName = "draft_email"
	FunctionCallNameDraftCall              FunctionCallName = "draft_call"
)

// DraftDecision is the agent's decision on a draft card produced by the
// draft_message tool (VOIP-1704), the draft_email tool (VOIP-1710) or the
// draft_call tool (VOIP-1717).
type DraftDecision string

// list of draft decisions
const (
	DraftDecisionApprove DraftDecision = "approve"
	DraftDecisionReject  DraftDecision = "reject"
)

// BlockTypeSMSDraft is the type of the card block draft_message records.
const BlockTypeSMSDraft = "sms_draft"

// BlockTypeEmailDraft is the type of the card block draft_email records.
const BlockTypeEmailDraft = "email_draft"

// BlockTypeCallDraft is the type of the card block draft_call records.
const BlockTypeCallDraft = "call_draft"
