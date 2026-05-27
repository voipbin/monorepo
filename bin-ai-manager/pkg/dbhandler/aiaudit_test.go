package dbhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// insertTestAudit seeds a row directly via SQL to avoid MySQL-specific ON DUPLICATE KEY syntax.
func insertTestAudit(t *testing.T, a *aiaudit.AIAudit, status aiaudit.Status, tmDelete *time.Time, tmCreate *time.Time) {
	t.Helper()

	var tmDeleteVal any
	if tmDelete != nil {
		tmDeleteVal = tmDelete.Format("2006-01-02 15:04:05.000000")
	}
	var tmCreateVal any
	if tmCreate != nil {
		tmCreateVal = tmCreate.Format("2006-01-02 15:04:05.000000")
	}

	_, err := dbTest.Exec(
		`INSERT INTO ai_ai_audits
		 (id, customer_id, aicall_id, ai_id, prompt_history_id, status, language, error, tm_create, tm_update, tm_delete)
		 VALUES (?, ?, ?, ?, ?, ?, ?, '', ?, NULL, ?)`,
		a.ID.Bytes(),
		a.CustomerID.Bytes(),
		a.AIcallID.Bytes(),
		a.AIID.Bytes(),
		a.PromptHistoryID.Bytes(),
		string(status),
		a.Language,
		tmCreateVal,
		tmDeleteVal,
	)
	if err != nil {
		t.Fatalf("insertTestAudit: %v", err)
	}
}

func Test_AIAuditGet_ReturnsTMDeleteNull(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx := context.Background()
	tm := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)

	a := &aiaudit.AIAudit{
		Identity:        identity.Identity{ID: uuid.FromStringOrNil("aaaa0001-0001-0001-0001-000000000001"), CustomerID: uuid.FromStringOrNil("bbbb0001-0001-0001-0001-000000000001")},
		AIcallID:        uuid.FromStringOrNil("cccc0001-0001-0001-0001-000000000001"),
		AIID:            uuid.FromStringOrNil("dddd0001-0001-0001-0001-000000000001"),
		PromptHistoryID: uuid.FromStringOrNil("eeee0001-0001-0001-0001-000000000001"),
		Language:        "en-US",
	}
	insertTestAudit(t, a, aiaudit.StatusProgressing, nil, &tm)

	got, err := h.AIAuditGet(ctx, a.ID)
	if err != nil {
		t.Fatalf("AIAuditGet error = %v", err)
	}
	if got.TMDelete != nil {
		t.Errorf("expected tm_delete = NULL, got %v", got.TMDelete)
	}
	if got.Status != aiaudit.StatusProgressing {
		t.Errorf("expected status progressing, got %s", got.Status)
	}
}

func Test_AIAuditUpdateFinal_OnlyUpdatesWhenNotDeleted(t *testing.T) {
	curTime := func() *time.Time {
		t := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
		return &t
	}()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx := context.Background()
	tm := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)

	liveID := uuid.FromStringOrNil("aaaa0002-0002-0002-0002-000000000001")
	deletedID := uuid.FromStringOrNil("aaaa0002-0002-0002-0002-000000000002")
	customerID := uuid.FromStringOrNil("bbbb0002-0002-0002-0002-000000000001")

	liveAudit := &aiaudit.AIAudit{
		Identity:        identity.Identity{ID: liveID, CustomerID: customerID},
		AIcallID:        uuid.FromStringOrNil("cccc0002-0002-0002-0002-000000000001"),
		AIID:            uuid.FromStringOrNil("dddd0002-0002-0002-0002-000000000001"),
		PromptHistoryID: uuid.FromStringOrNil("eeee0002-0002-0002-0002-000000000001"),
		Language:        "en-US",
	}
	deletedAudit := &aiaudit.AIAudit{
		Identity:        identity.Identity{ID: deletedID, CustomerID: customerID},
		AIcallID:        uuid.FromStringOrNil("cccc0002-0002-0002-0002-000000000002"),
		AIID:            uuid.FromStringOrNil("dddd0002-0002-0002-0002-000000000002"),
		PromptHistoryID: uuid.FromStringOrNil("eeee0002-0002-0002-0002-000000000002"),
		Language:        "en-US",
	}

	insertTestAudit(t, liveAudit, aiaudit.StatusProgressing, nil, &tm)
	insertTestAudit(t, deletedAudit, aiaudit.StatusProgressing, &tm, &tm)

	score := 85

	// Live record: should update (1 row affected).
	mockUtil.EXPECT().TimeNow().Return(curTime)
	n, err := h.AIAuditUpdateFinal(ctx, liveID, aiaudit.StatusCompleted, &score, nil, "")
	if err != nil {
		t.Fatalf("AIAuditUpdateFinal (live) error = %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row updated for live record, got %d", n)
	}

	// Soft-deleted record: must return 0 rows.
	mockUtil.EXPECT().TimeNow().Return(curTime)
	n, err = h.AIAuditUpdateFinal(ctx, deletedID, aiaudit.StatusCompleted, &score, nil, "")
	if err != nil {
		t.Fatalf("AIAuditUpdateFinal (deleted) error = %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 rows updated for soft-deleted record, got %d", n)
	}
}

func Test_AIAuditCountProgressing_ExcludesSoftDeleted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx := context.Background()
	tm := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)
	customerID := uuid.FromStringOrNil("bbbb0003-0003-0003-0003-000000000001")

	liveAudit := &aiaudit.AIAudit{
		Identity:        identity.Identity{ID: uuid.FromStringOrNil("aaaa0003-0003-0003-0003-000000000001"), CustomerID: customerID},
		AIcallID:        uuid.FromStringOrNil("cccc0003-0003-0003-0003-000000000001"),
		AIID:            uuid.FromStringOrNil("dddd0003-0003-0003-0003-000000000001"),
		PromptHistoryID: uuid.FromStringOrNil("eeee0003-0003-0003-0003-000000000001"),
		Language:        "en-US",
	}
	deletedAudit := &aiaudit.AIAudit{
		Identity:        identity.Identity{ID: uuid.FromStringOrNil("aaaa0003-0003-0003-0003-000000000002"), CustomerID: customerID},
		AIcallID:        uuid.FromStringOrNil("cccc0003-0003-0003-0003-000000000002"),
		AIID:            uuid.FromStringOrNil("dddd0003-0003-0003-0003-000000000002"),
		PromptHistoryID: uuid.FromStringOrNil("eeee0003-0003-0003-0003-000000000002"),
		Language:        "en-US",
	}

	insertTestAudit(t, liveAudit, aiaudit.StatusProgressing, nil, &tm)
	insertTestAudit(t, deletedAudit, aiaudit.StatusProgressing, &tm, &tm) // soft-deleted

	count, err := h.AIAuditCountProgressing(ctx, customerID)
	if err != nil {
		t.Fatalf("AIAuditCountProgressing error = %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 progressing audit, got %d", count)
	}
}
