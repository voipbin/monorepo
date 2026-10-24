package verdict

import (
	"encoding/json"
	"strings"
	"testing"
)

func Test_ValidateRaw(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		eventCount int

		expectErr string // substring of the error; empty means success
	}{
		{"invalid json", `{"overall_status":`, 3, "could not unmarshal the raw verdict"},
		{"null has no status", `null`, 3, "invalid overall_status"},
		{"empty object has no status", `{}`, 3, "invalid overall_status"},
		{"unknown status", `{"overall_status":"bad"}`, 3, `invalid overall_status: "bad"`},
		{"ok without issues", `{"overall_status":"ok"}`, 0, ""},

		{"unknown severity", `{"overall_status":"ok","issues":[{"severity":"fatal"}]}`, 3, `issue[0]: invalid severity: "fatal"`},
		{"severity is checked before evidence", `{"overall_status":"error","issues":[{"severity":"fatal"}]}`, 3, "invalid severity"},
		{"warning issue without evidence", `{"overall_status":"warning","issues":[{"severity":"info"}]}`, 3, "issue[0]: evidence is required for a non-ok verdict"},
		{"error issue without evidence", `{"overall_status":"error","issues":[{"severity":"error"}]}`, 3, "evidence is required for a non-ok verdict"},
		{"ok issue without evidence is allowed", `{"overall_status":"ok","issues":[{"severity":"info"}]}`, 3, ""},

		{"index below zero", `{"overall_status":"error","issues":[{"severity":"error","evidence_index":[-1]}]}`, 3, "evidence_index -1 out of range [0, 3)"},
		{"index at the lower bound", `{"overall_status":"error","issues":[{"severity":"error","evidence_index":[0]}]}`, 3, ""},
		{"index at the upper bound", `{"overall_status":"error","issues":[{"severity":"error","evidence_index":[2]}]}`, 3, ""},
		{"index equal to the count", `{"overall_status":"error","issues":[{"severity":"error","evidence_index":[3]}]}`, 3, "evidence_index 3 out of range [0, 3)"},
		{"no events and an index", `{"overall_status":"error","issues":[{"severity":"error","evidence_index":[0]}]}`, 0, "evidence_index 0 out of range [0, 0)"},
		{"out of range index is rejected under ok too", `{"overall_status":"ok","issues":[{"severity":"info","evidence_index":[5]}]}`, 3, "out of range"},
		{"second issue is named in the error", `{"overall_status":"error","issues":[{"severity":"error","evidence_index":[0]},{"severity":"warning","evidence_index":[9]}]}`, 3, "issue[1]: evidence_index 9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateRaw(json.RawMessage(tt.raw), tt.eventCount)
			if tt.expectErr == "" {
				if err != nil || got == nil {
					t.Errorf("Wrong match. expect: ok, got: %v err=%v", got, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
			}
			if got != nil {
				t.Errorf("Wrong match. expect: nil result on error, got: %v", got)
			}
		})
	}
}

func Test_ValidateRaw_parsed_fields(t *testing.T) {
	raw := `{"overall_status":"warning","resources_used":[{"type":"call","count":2}],"interactions":[{"resource_type":"call","summary":"s"}],"narrative":"n","issues":[{"severity":"warning","area":"a","summary":"x","evidence_index":[0,2]}]}`

	got, err := ValidateRaw(json.RawMessage(raw), 3)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got.OverallStatus != OverallStatusWarning || got.Narrative != "n" {
		t.Errorf("Wrong match. unexpected status or narrative: %+v", got)
	}
	if len(got.ResourcesUsed) != 1 || got.ResourcesUsed[0].Type != "call" || got.ResourcesUsed[0].Count != 2 {
		t.Errorf("Wrong match. unexpected resources: %+v", got.ResourcesUsed)
	}
	if len(got.Interactions) != 1 || got.Interactions[0].Summary != "s" {
		t.Errorf("Wrong match. unexpected interactions: %+v", got.Interactions)
	}
	if len(got.Issues) != 1 || got.Issues[0].Severity != SeverityWarning || len(got.Issues[0].EvidenceIndex) != 2 || got.Issues[0].EvidenceIndex[1] != 2 {
		t.Errorf("Wrong match. unexpected issues: %+v", got.Issues)
	}
}
