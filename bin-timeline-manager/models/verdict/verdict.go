package verdict

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// CurrentVersion is the schema version of the persisted verdict JSON.
//
// v2 (VOIP-1200): added Interactions (the Stage 2 content summary, previously
// computed then discarded). v1 records have no interactions key; consumers use
// Version to disambiguate.
//
// v3 (VOIP-1203): added SessionContext (channel-neutral 5W1H header), Outcome
// (who/how the session ended), and Metrics (voice/AI turn+latency). All three
// are Go-authoritative (never LLM-authored) and OPTIONAL pointers: a reference
// that does not resolve serializes them as an omitted key (not null/zero). v2
// records have no session_context/outcome/metrics keys.
const CurrentVersion = 3

// OverallStatus is the holistic verdict status.
type OverallStatus string

const (
	OverallStatusOK      OverallStatus = "ok"
	OverallStatusWarning OverallStatus = "warning"
	OverallStatusError   OverallStatus = "error"
)

// Severity is an issue severity level.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Evidence is a resolved pointer to a concrete event in the canonical event list.
//
// The LLM emits only EvidenceIndex (an integer index into the frozen canonical
// list); Go resolves it to the EventType/Timestamp/ResourceID tuple before
// persisting. EvidenceIndex is kept so a downstream UI can highlight the exact
// frozen-list event without re-deriving from the non-unique tuple.
type Evidence struct {
	EvidenceIndex int    `json:"evidence_index"`
	EventType     string `json:"event_type,omitempty"`
	Timestamp     string `json:"timestamp,omitempty"`
	ResourceID    string `json:"resource_id,omitempty"`
}

// ResourceUsed is a per-type resource count (Go-computed, authoritative).
type ResourceUsed struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// Interaction is one resource's content summary: what was communicated and the
// intent/outcome. Carried forward from the Stage 2 content pass (staged path)
// or emitted directly by the combined call (single-call path). Customer-facing:
// it carries no internal fields (no model id, no token counts).
type Interaction struct {
	ResourceType string `json:"resource_type"`
	Summary      string `json:"summary"`
}

// Issue is a single detected problem.
type Issue struct {
	Severity Severity   `json:"severity"`
	Area     string     `json:"area"`
	Summary  string     `json:"summary"`
	Evidence []Evidence `json:"evidence"`
}

// Participant unifies the per-channel participant fields (call Source/
// Destination, conversation Self/Peer) into one channel-neutral shape (v3).
type Participant struct {
	Role    string `json:"role"`    // "source" | "destination" | "self" | "peer"
	Address string `json:"address"` // the address target (phone/email/handle)
}

// SessionContext is the channel-neutral 5W1H header for a card-bearing
// activeflow reference (v3). It is Go-authoritative and OPTIONAL: nil only when
// nothing resolves (e.g. a deleted reference). The pointer + omitempty means an
// unresolved reference omits the key entirely (never null/zero-value).
//
// For a transcribe/recording activeflow the body is BORROWED from the chased
// origin call/conversation, while ReferenceType stays the real activeflow value
// and OriginKind/OriginType mark "this is a transcription/recording OF that
// origin".
type SessionContext struct {
	ReferenceType string        `json:"reference_type"`          // raw activeflow enum: call|conversation|ai|api|transcribe|recording|campaign|""
	Channel       string        `json:"channel"`                 // normalized: voice|chat|ai|api (derived; NO sms/email — not reference_types)
	Direction     string        `json:"direction,omitempty"`     // normalized "inbound"|"outbound"|"" (derived)
	DirectionRaw  string        `json:"direction_raw,omitempty"` // the source enum verbatim (call incoming/outgoing)
	Participants  []Participant `json:"participants,omitempty"`  // omitted (not []) when none resolvable
	FlowName      string        `json:"flow_name,omitempty"`     // best-effort, customer-scoped
	StartedAt     string        `json:"started_at,omitempty"`    // RFC3339, channel-appropriate start
	OriginKind    string        `json:"origin_kind,omitempty"`   // ""|"transcription"|"recording" (this activeflow IS a X of the origin)
	OriginType    string        `json:"origin_type,omitempty"`   // the chased origin's reference_type (call|conversation|confbridge)
	MultiLeg      bool          `json:"multi_leg"`               // reference expands to >1 leg (groupcall/campaign)
	AIHandled     bool          `json:"ai_handled"`              // a pipecat/ai session was present
	HumanInvolved bool          `json:"human_involved"`          // an agent-manager leg connected
}

// SessionOutcome is the channel-neutral result (v3). Its meaning is
// per-reference_type:
//   - call: ended_by (hangup originator) + reason (hangup_reason) + duration.
//   - conversation: last_activity_by + unanswered + turns + thread span (NO ended_by).
//   - ai: aicall end status.
//   - transcribe/recording: inherits the chased origin's outcome.
//
// EndedBy is populated ONLY for reference_types where "who ended it" is a real
// concept (call). For conversation the dialogue-flow fields live in Detail. The
// UI derives the human label from (reference_type, direction, ended_by); the
// data never bakes in "Customer"/"System".
type SessionOutcome struct {
	Result  string            `json:"result"`             // normalized: completed|failed|no_answer|busy|in_progress|unknown
	EndedBy string            `json:"ended_by,omitempty"` // call only: raw hangup_by (remote|local|""); other types omit
	Reason  string            `json:"reason,omitempty"`   // raw channel reason (call hangup_reason; conversation last-msg status)
	Detail  map[string]string `json:"detail,omitempty"`   // channel-raw extras (call: duration_sec; conversation: last_activity_by, turns_self, turns_peer, unanswered, ...)
}

// SessionMetrics is the deterministic voice/AI interaction-quality aggregate
// (v3). Nil for non-voice references (conversation/api/none) and chased cards in
// P1 — never a misleading zero-value block. Aggregated from the FULL
// pre-reduction event stream (collectedInput.allEvents), not the reduced list.
type SessionMetrics struct {
	TurnsUser       int  `json:"turns_user"`                  // message_user_transcription events
	TurnsBot        int  `json:"turns_bot"`                   // message_bot_transcription events (NOT *_llm_intermediate)
	FirstResponseMS *int `json:"first_response_ms,omitempty"` // pipecatcall_initialized -> first bot event (same clock)
	AvgResponseMS   *int `json:"avg_response_ms,omitempty"`
	MaxResponseMS   *int `json:"max_response_ms,omitempty"`
	MaxGapMS        *int `json:"max_gap_ms,omitempty"` // max gap between adjacent interaction events (NOT silence)
}

// Verdict is the persisted structured analysis result.
type Verdict struct {
	Version        int             `json:"version"`
	OverallStatus  OverallStatus   `json:"overall_status"`
	InputReduced   bool            `json:"input_reduced"`
	SessionContext *SessionContext `json:"session_context,omitempty"` // NEW (v3)
	Outcome        *SessionOutcome `json:"outcome,omitempty"`         // NEW (v3)
	Metrics        *SessionMetrics `json:"metrics,omitempty"`         // NEW (v3)
	ResourcesUsed  []ResourceUsed  `json:"resources_used"`
	Interactions   []Interaction   `json:"interactions"`
	Narrative      string          `json:"narrative"`
	Issues         []Issue         `json:"issues"`
}

// RawVerdict mirrors what the LLM returns: evidence is a bare list of integer
// indices, not the resolved tuples. This is the shape validated and resolved
// in the raw-output phase.
type RawVerdict struct {
	OverallStatus OverallStatus  `json:"overall_status"`
	ResourcesUsed []ResourceUsed `json:"resources_used"`
	Interactions  []Interaction  `json:"interactions"`
	Narrative     string         `json:"narrative"`
	Issues        []RawIssue     `json:"issues"`
}

// RawIssue is an issue as emitted by the LLM (evidence = indices).
type RawIssue struct {
	Severity       Severity `json:"severity"`
	Area           string   `json:"area"`
	Summary        string   `json:"summary"`
	EvidenceIndex  []int    `json:"evidence_index"`
}

func validOverallStatus(s OverallStatus) bool {
	switch s {
	case OverallStatusOK, OverallStatusWarning, OverallStatusError:
		return true
	}
	return false
}

func validSeverity(s Severity) bool {
	switch s {
	case SeverityInfo, SeverityWarning, SeverityError:
		return true
	}
	return false
}

// ValidateRaw performs the raw-output validation phase on the LLM JSON:
//   - enum membership (overall_status, severity)
//   - non-empty evidence on every non-ok issue
//   - every cited evidence_index within [0, eventCount)
//
// It returns the parsed RawVerdict on success. eventCount is the size of the
// frozen canonical event list.
func ValidateRaw(raw json.RawMessage, eventCount int) (*RawVerdict, error) {
	var rv RawVerdict
	if err := json.Unmarshal(raw, &rv); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal the raw verdict")
	}

	if !validOverallStatus(rv.OverallStatus) {
		return nil, errors.Errorf("invalid overall_status: %q", rv.OverallStatus)
	}

	for i, iss := range rv.Issues {
		if !validSeverity(iss.Severity) {
			return nil, errors.Errorf("issue[%d]: invalid severity: %q", i, iss.Severity)
		}
		// non-ok verdict requires evidence on each issue.
		if rv.OverallStatus != OverallStatusOK && len(iss.EvidenceIndex) == 0 {
			return nil, errors.Errorf("issue[%d]: evidence is required for a non-ok verdict", i)
		}
		for _, idx := range iss.EvidenceIndex {
			if idx < 0 || idx >= eventCount {
				return nil, errors.Errorf("issue[%d]: evidence_index %d out of range [0, %d)", i, idx, eventCount)
			}
		}
	}

	return &rv, nil
}
