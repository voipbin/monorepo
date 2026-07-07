package conversationhandler

import (
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/conversation"
)

// Test_CaseIDHint_NeverReadByExecuteMode is the explicit negative test
// required by contact-case-management design §4.3: getExecuteMode's
// agent/flow dispatch decision must be governed exclusively by
// Conversation.Owner, never by Metadata.ContactCaseID. This proves it by
// constructing two Conversations that are identical except for
// Metadata.ContactCaseID, and asserting getExecuteMode returns the same
// mode for both -- i.e. the case_id hint cannot possibly be influencing
// the result, whatever its value.
func Test_CaseIDHint_NeverReadByExecuteMode(t *testing.T) {
	h := &conversationHandler{}
	caseID := uuid.FromStringOrNil("f1b2c3d4-000f-000f-000f-000000000001")

	base := conversation.Conversation{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("f1b2c3d4-000f-000f-000f-000000000002"),
		},
	}

	tests := []struct {
		name     string
		metadata *conversation.Metadata
	}{
		{name: "nil metadata", metadata: nil},
		{name: "metadata with ContactCaseID set", metadata: &conversation.Metadata{ContactCaseID: &caseID}},
		{name: "metadata with ContactCaseID unset", metadata: &conversation.Metadata{}},
	}

	var results []ExecuteMode
	for _, tt := range tests {
		cv := base
		cv.Metadata = tt.metadata
		results = append(results, h.getExecuteMode(&cv))
	}

	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Fatalf("getExecuteMode must be invariant to Metadata.ContactCaseID; got %v for %q vs %v for %q",
				results[i], tests[i].name, results[0], tests[0].name)
		}
	}

	// Also verify caseIDHint itself correctly reads only what's on Metadata,
	// independent from getExecuteMode's dispatch decision above.
	cvWithHint := base
	cvWithHint.Metadata = &conversation.Metadata{ContactCaseID: &caseID}
	if got := caseIDHint(&cvWithHint); got == nil || *got != caseID {
		t.Errorf("expected caseIDHint to return %v, got: %v", caseID, got)
	}

	cvNoMetadata := base
	cvNoMetadata.Metadata = nil
	if got := caseIDHint(&cvNoMetadata); got != nil {
		t.Errorf("expected caseIDHint to return nil for nil Metadata, got: %v", *got)
	}
}
