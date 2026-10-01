// Package builder holds the wire models, shared limits and request validation
// for the conversational Assistant Builder (VOIP-1558).
//
// The Builder is stateless: the client sends the full conversation on every
// turn and the server stores nothing. ValidateRequest lives here (not in the
// ai-manager handler package) so api-manager and ai-manager run the same
// checks while importing only models, matching the existing convention.
package builder

import "monorepo/bin-ai-manager/models/tool"

// Internal RPC URIs shared by requesthandler and listenhandler.
const (
	URIChat   = "/v1/ai_builder/chat"
	URIStatus = "/v1/ai_builder/status"
)

// Message roles accepted from the client.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Input limits, in runes (Korean is 3 bytes per rune in UTF-8, so a byte cap
// would penalise Korean users). Initial values, not measured.
const (
	MaxMessages        = 40
	MaxMessageRunes    = 2000
	MaxTotalRunes      = 40000
	MaxInitPromptRunes = 8000
	MaxNameRunes       = 100
	MaxDetailRunes     = 500
)

// Error reasons carried in VoipbinError.Reason.
const (
	ReasonDisabled        = "BUILDER_DISABLED"
	ReasonUnavailable     = "BUILDER_UNAVAILABLE"
	ReasonDailyLimit      = "BUILDER_DAILY_LIMIT"
	ReasonBusy            = "BUILDER_BUSY"
	ReasonResponseInvalid = "BUILDER_RESPONSE_INVALID"
	ReasonTimeout         = "BUILDER_TIMEOUT"
	ReasonInputTooLarge   = "BUILDER_INPUT_TOO_LARGE" // body bytes exceeded (api-manager MaxBytesReader only)
	ReasonInvalidArgument = "INVALID_ARGUMENT"
)

// AllowedTools is the fail-closed allow-list of tools a drafted assistant may
// use. A tool added to tool.AllToolNames later is NOT picked up automatically.
var AllowedTools = []tool.ToolName{
	tool.ToolNameConnectCall,
	tool.ToolNameStopService,
	tool.ToolNameSendEmail,
	tool.ToolNameSendMessage,
	tool.ToolNameSetVariables,
	tool.ToolNameCaseCreate,
}

// IsAllowedTool reports whether name is in AllowedTools.
func IsAllowedTool(name string) bool {
	for _, t := range AllowedTools {
		if string(t) == name {
			return true
		}
	}
	return false
}

// Message is one chat turn. Assistant entries are the plain-text "message" of
// an earlier response.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Draft is the assistant configuration the Builder proposes. The user reviews
// and saves it through the existing AI create form; nothing is saved here.
type Draft struct {
	Name       string   `json:"name"`
	Detail     string   `json:"detail"`
	InitPrompt string   `json:"init_prompt"`
	ToolNames  []string `json:"tool_names"`
}

// ChatRequest is the full client-held conversation plus the latest draft.
type ChatRequest struct {
	Messages     []Message `json:"messages"`
	CurrentDraft *Draft    `json:"current_draft,omitempty"`
}

// ChatResponse is one turn's result. Every field except Message is decided by
// code, not by model prose: Draft is validated and DraftWarnings are facts the
// parser recorded.
type ChatResponse struct {
	Message       string   `json:"message"`
	Draft         *Draft   `json:"draft,omitempty"`
	Assumptions   []string `json:"assumptions,omitempty"`
	DraftWarnings []string `json:"draft_warnings,omitempty"`
}

// StatusResponse tells the client whether the Builder is usable and its limits.
type StatusResponse struct {
	Available       bool `json:"available"`
	MaxMessages     int  `json:"max_messages"`
	MaxMessageChars int  `json:"max_message_chars"`
}
