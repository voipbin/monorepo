package resolution

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
)

// Test_Resolution_CaseLevel_NilInteractionID verifies Task 2.1 of the
// contact-case-management design (§3.3): InteractionID is now nullable
// (*uuid.UUID, was uuid.UUID) so a case-level Resolution (attributing a
// whole Case to a contact, independent of any single Interaction) can be
// constructed with InteractionID: nil and CaseID: &someID, and
// JSON-marshals without error.
func Test_Resolution_CaseLevel_NilInteractionID(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-2003-2003-2003-000000000001")

	r := &Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-2003-2003-2003-000000000002"),
		CustomerID:     uuid.FromStringOrNil("f1b2c3d4-2003-2003-2003-000000000003"),
		ContactID:      uuid.FromStringOrNil("f1b2c3d4-2003-2003-2003-000000000004"),
		InteractionID:  nil,
		CaseID:         &caseID,
		ResolutionType: ResolutionTypePositive,
		ResolvedByType: ResolvedByTypeAgent,
		ResolvedByID:   uuid.FromStringOrNil("f1b2c3d4-2003-2003-2003-000000000005"),
	}

	if r.InteractionID != nil {
		t.Fatalf("expected InteractionID to be nil for a case-level resolution, got: %v", r.InteractionID)
	}
	if r.CaseID == nil || *r.CaseID != caseID {
		t.Fatalf("expected CaseID: %v, got: %v", caseID, r.CaseID)
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if raw["interaction_id"] != nil {
		t.Errorf("expected interaction_id to be null in JSON, got: %v", raw["interaction_id"])
	}
	if raw["case_id"] != caseID.String() {
		t.Errorf("expected case_id: %v, got: %v", caseID.String(), raw["case_id"])
	}
}

// Test_Resolution_InteractionLevel_NilCaseID verifies the existing
// interaction-level shape still constructs correctly: InteractionID set,
// CaseID nil. This is the far more common case and must remain
// unaffected by the nullable widen.
func Test_Resolution_InteractionLevel_NilCaseID(t *testing.T) {
	interactionID := uuid.FromStringOrNil("f1b2c3d4-2003-2003-2003-000000000006")

	r := &Resolution{
		ID:             uuid.FromStringOrNil("f1b2c3d4-2003-2003-2003-000000000007"),
		CustomerID:     uuid.FromStringOrNil("f1b2c3d4-2003-2003-2003-000000000008"),
		ContactID:      uuid.FromStringOrNil("f1b2c3d4-2003-2003-2003-000000000009"),
		InteractionID:  &interactionID,
		CaseID:         nil,
		ResolutionType: ResolutionTypePositive,
		ResolvedByType: ResolvedByTypeAgent,
	}

	if r.InteractionID == nil || *r.InteractionID != interactionID {
		t.Fatalf("expected InteractionID: %v, got: %v", interactionID, r.InteractionID)
	}
	if r.CaseID != nil {
		t.Fatalf("expected CaseID to be nil, got: %v", *r.CaseID)
	}
}
