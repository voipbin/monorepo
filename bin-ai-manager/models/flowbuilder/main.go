// Package flowbuilder holds the wire models and shared limits for the Flow
// AI Builder (VOIP-1573). Like the Assistant Builder (bin-ai-manager/models/
// builder), it is stateless: the client sends the full conversation and the
// current draft on every turn and the server stores nothing.
//
// Reason* error codes and the conversation-side input limits are shared with
// package builder (see design doc §5, Open Question 4) rather than
// duplicated; only the draft-shaped wire types and draft-only limits are
// defined here.
package flowbuilder

import (
	"monorepo/bin-ai-manager/models/builder"

	"github.com/gofrs/uuid"
)

// Internal RPC URIs shared by requesthandler and listenhandler.
const (
	URIChat = "/v1/flow_builder/chat"
	// No separate status URI: the Flow Builder reuses builder.URIStatus
	// (design doc 5.1: avoids a second cache instance, RPC and OpenAPI
	// surface for an identical availability check).
)

// Message roles accepted from the client, shared with the Assistant Builder.
const (
	RoleUser      = builder.RoleUser
	RoleAssistant = builder.RoleAssistant
)

// Draft-only input limits, in runes/bytes. The conversation-side limits
// (MaxMessages, MaxMessageRunes, MaxTotalRunes counting message text only)
// are builder.MaxMessages etc; see design doc §5.
const (
	MaxFlowNodes          = 60   // nodes per draft, initial value, not measured
	MaxOptionBytes        = 4096 // per-node option, serialized, initial value
	MaxOptionDepth        = 8    // option nesting depth
	MaxDraftBytes         = 240 * 1024
	MaxLabelRunes         = 64
	MaxSupportedTypes     = 100
	MaxSupportedTypeRunes = 64
)

// Error reasons: shared with the Assistant Builder (design doc §8, Open
// Question 4). Re-exported under this package's name for callers that only
// import flowbuilder.
const (
	ReasonUnavailable     = builder.ReasonUnavailable
	ReasonDailyLimit      = builder.ReasonDailyLimit
	ReasonBusy            = builder.ReasonBusy
	ReasonResponseInvalid = builder.ReasonResponseInvalid
	ReasonTimeout         = builder.ReasonTimeout
	ReasonInputTooLarge   = builder.ReasonInputTooLarge
	ReasonInvalidArgument = builder.ReasonInvalidArgument
)

// Warning keys written to ChatResponse.DraftWarnings. Each is either a bare
// key or "key: detail" (matching the existing Assistant Builder convention);
// see design doc §5.
const (
	WarningDuplicateLabel    = "duplicate_label"
	WarningUnsupportedAction = "unsupported_action"
	WarningInvalidLabelRef   = "invalid_label_ref"
	WarningInvalidOption     = "invalid_option"
	WarningSelectResource    = "select_resource"
	WarningOpenEnd           = "open_end"
	WarningEmptyActionRef    = "empty_action_ref"
	WarningUnreachable       = "unreachable"
	WarningNextIgnored       = "next_ignored"
	WarningMissingRequired   = "missing_required"
	WarningMediaMixed        = "media_mixed"
	WarningEmptyDraft        = "empty_draft"
)

// Message is one chat turn. It is the Assistant Builder's message type so the
// conversation-side validation (builder.ValidateRequest) is shared as is.
type Message = builder.Message

// SymbolicNode is one node of the label-addressed graph the LLM produces
// (design doc §3.1). The builder never asks the model for a UUID or a
// position; label is the only cross-reference the model uses, inside Next
// and inside any ref:"action" option field.
type SymbolicNode struct {
	Label  string         `json:"label"`
	Type   string         `json:"type"`
	Option map[string]any `json:"option,omitempty"`
	Next   *string        `json:"next,omitempty"`
}

// SymbolicGraph is the LLM's draft response shape before the server
// transcodes it into a Draft (design doc §3.1, §3.2).
type SymbolicGraph struct {
	Nodes []SymbolicNode `json:"nodes"`
}

// Draft is the server-confirmed Flow graph: a flat []action.Action (first
// element is the start node, matching flow-manager's IDStart convention),
// plus the label each action id carries so the server can deterministically
// reconstruct the symbolic graph on the next turn without storing anything
// (design doc §3.2 step 9, §5).
//
// Actions is typed as []map[string]any (not []action.Action): the wire shape
// is exactly action.Action's JSON encoding, and the client sends back what it
// received. builderhandler does the typed conversion.
type Draft struct {
	Actions []map[string]any  `json:"actions"`
	Labels  map[string]string `json:"labels"`
}

// ChatRequest is the full client-held conversation plus the latest draft and
// the set of action types the requesting editor can render and round-trip
// faithfully (design doc §2.6). SupportedActionTypes is required: this is a
// new endpoint with no legacy client to default for.
type ChatRequest struct {
	Messages             []Message `json:"messages"`
	CurrentDraft         *Draft    `json:"current_draft,omitempty"`
	SupportedActionTypes []string  `json:"supported_action_types"`
}

// ChatResponse is one turn's result. Every field except Message is decided
// by code, not by model prose (design doc §4, §5).
type ChatResponse struct {
	Message        string      `json:"message"`
	Draft          *Draft      `json:"draft,omitempty"`
	DraftWarnings  []string    `json:"draft_warnings,omitempty"`
	SensitiveNodes []uuid.UUID `json:"sensitive_nodes,omitempty"`
	Assumptions    []string    `json:"assumptions,omitempty"`
}
