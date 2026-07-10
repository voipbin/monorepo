package interaction

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

// Test_Interaction_ConstructAndMarshal verifies the basic Interaction
// struct constructs as expected. The now-removed CaseID field (VOIP-1245)
// used to be asserted here; that field was dead code (Case creation is
// explicit-only since VOIP-1243, and case_id was never written after
// InteractionCreate).
func Test_Interaction_ConstructAndMarshal(t *testing.T) {
	now := time.Now()

	i := Interaction{
		ID:            uuid.FromStringOrNil("f1b2c3d4-8001-8001-8001-000000000002"),
		ReferenceType: "call",
		TMCreate:      &now,
	}
	if i.ReferenceType != "call" {
		t.Errorf("expected ReferenceType: call, got: %v", i.ReferenceType)
	}
	if i.TMCreate == nil || !i.TMCreate.Equal(now) {
		t.Errorf("expected TMCreate: %v, got: %v", now, i.TMCreate)
	}
}
