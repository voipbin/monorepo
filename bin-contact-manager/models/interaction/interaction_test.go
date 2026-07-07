package interaction

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

// Test_Interaction_CaseID_NullableField verifies Task 3.4's addition of
// the case_id linkage column (contact-case-management design §4 step 3:
// "INSERT INTO contact_interactions (..., case_id=case_id)"). CaseID is
// nullable -- Interactions projected before this feature existed (or
// whose peer/reference_type never resolved to a Case) have it nil.
func Test_Interaction_CaseID_NullableField(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-8001-8001-8001-000000000001")
	now := time.Now()

	withCase := Interaction{
		ID:            uuid.FromStringOrNil("f1b2c3d4-8001-8001-8001-000000000002"),
		ReferenceType: "call",
		CaseID:        &caseID,
		TMCreate:      &now,
	}
	if withCase.CaseID == nil || *withCase.CaseID != caseID {
		t.Errorf("expected CaseID: %v, got: %v", caseID, withCase.CaseID)
	}

	withoutCase := Interaction{
		ID:            uuid.FromStringOrNil("f1b2c3d4-8001-8001-8001-000000000003"),
		ReferenceType: "call",
		TMCreate:      &now,
	}
	if withoutCase.CaseID != nil {
		t.Errorf("expected nil CaseID for an interaction with no resolved case, got: %v", *withoutCase.CaseID)
	}
}
