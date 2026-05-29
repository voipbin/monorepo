package aipromptproposal_test

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"monorepo/bin-ai-manager/models/aipromptproposal"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestConvertWebhookMessage_preservesFields(t *testing.T) {
	now := time.Now()
	auditID := uuid.Must(uuid.NewV4())
	p := &aipromptproposal.AIPromptProposal{
		Identity:             commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: uuid.Must(uuid.NewV4())},
		AIID:                 uuid.Must(uuid.NewV4()),
		AuditIDs:             []uuid.UUID{auditID},
		BasisPromptHistoryID: uuid.Must(uuid.NewV4()),
		OriginalPrompt:       "original",
		ProposedPrompt:       "proposed",
		Rationale:            "because",
		Status:               aipromptproposal.StatusCompleted,
		TMCreate:             &now,
	}

	wm := p.ConvertWebhookMessage()

	if wm.ID != p.ID {
		t.Errorf("expected ID %v, got %v", p.ID, wm.ID)
	}
	if wm.Status != p.Status {
		t.Errorf("expected status %v, got %v", p.Status, wm.Status)
	}
	if len(wm.AuditIDs) != 1 || wm.AuditIDs[0] != auditID {
		t.Errorf("expected audit_ids to contain %v, got %v", auditID, wm.AuditIDs)
	}
	if wm.OriginalPrompt != p.OriginalPrompt {
		t.Errorf("expected original_prompt %q, got %q", p.OriginalPrompt, wm.OriginalPrompt)
	}
}
