package aiaudit_test

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"monorepo/bin-ai-manager/models/aiaudit"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestConvertWebhookMessage_stripsInternalFields(t *testing.T) {
	now := time.Now()
	score := 4
	a := &aiaudit.AIAudit{
		Identity:        commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: uuid.Must(uuid.NewV4())},
		AIcallID:        uuid.Must(uuid.NewV4()),
		AIID:            uuid.Must(uuid.NewV4()),
		PromptHistoryID: uuid.Must(uuid.NewV4()),
		Status:          aiaudit.StatusCompleted,
		OverallScore:    &score,
		Language:        "en-US",
		TMCreate:        &now,
	}

	wm := a.ConvertWebhookMessage()

	if wm.ID != a.ID {
		t.Errorf("expected ID %v, got %v", a.ID, wm.ID)
	}
	if wm.Status != a.Status {
		t.Errorf("expected status %v, got %v", a.Status, wm.Status)
	}
	if wm.OverallScore != a.OverallScore {
		t.Errorf("expected overall_score pointer to match")
	}
}
