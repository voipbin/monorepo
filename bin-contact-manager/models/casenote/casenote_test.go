package casenote

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

// Test_CaseNote_Fields verifies the CaseNote struct shape matches design
// §3.5: ID/CustomerID/CaseID identity, AuthorType/AuthorID attribution,
// Text content, and soft-delete via TMDelete (design §3.5's explicit
// choice: mirrors contact_resolutions' retraction pattern, not
// contact_interactions' append-only pattern, since a note can be removed
// by the authoring agent).
func Test_CaseNote_Fields(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	caseID := uuid.Must(uuid.NewV4())
	authorID := uuid.Must(uuid.NewV4())
	now := time.Now()

	n := CaseNote{
		ID:         id,
		CustomerID: customerID,
		CaseID:     caseID,
		AuthorType: AuthorTypeAgent,
		AuthorID:   &authorID,
		Text:       "customer confirmed the outage is resolved",
		TMCreate:   &now,
	}

	if n.ID != id || n.CustomerID != customerID || n.CaseID != caseID {
		t.Errorf("identity fields not set as expected: %+v", n)
	}
	if n.AuthorType != AuthorTypeAgent || n.AuthorID == nil || *n.AuthorID != authorID {
		t.Errorf("author fields not set as expected: %+v", n)
	}
	if n.TMDelete != nil {
		t.Errorf("expected TMDelete nil (active) by default, got: %v", n.TMDelete)
	}
}
