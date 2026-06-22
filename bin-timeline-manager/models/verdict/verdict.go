package verdict

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// CurrentVersion is the schema version of the persisted verdict JSON.
const CurrentVersion = 1

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

// Issue is a single detected problem.
type Issue struct {
	Severity Severity   `json:"severity"`
	Area     string     `json:"area"`
	Summary  string     `json:"summary"`
	Evidence []Evidence `json:"evidence"`
}

// Verdict is the persisted structured analysis result.
type Verdict struct {
	Version       int            `json:"version"`
	OverallStatus OverallStatus  `json:"overall_status"`
	InputReduced  bool           `json:"input_reduced"`
	ResourcesUsed []ResourceUsed `json:"resources_used"`
	Narrative     string         `json:"narrative"`
	Issues        []Issue        `json:"issues"`
}

// RawVerdict mirrors what the LLM returns: evidence is a bare list of integer
// indices, not the resolved tuples. This is the shape validated and resolved
// in the raw-output phase.
type RawVerdict struct {
	OverallStatus OverallStatus  `json:"overall_status"`
	ResourcesUsed []ResourceUsed `json:"resources_used"`
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
