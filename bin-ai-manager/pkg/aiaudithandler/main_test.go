package aiaudithandler_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/pkg/aiaudithandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/geminiaudithandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestAIAuditHandlerInterfaceExists(t *testing.T) {
	var _ aiaudithandler.AIAuditHandler = nil
	t.Log("AIAuditHandler interface exists")
}

func TestGet_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockGemini := geminiaudithandler.NewMockGeminiAuditHandler(ctrl)

	auditID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	customerID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	expected := &aiaudit.AIAudit{
		Identity: commonidentity.Identity{
			ID:         auditID,
			CustomerID: customerID,
		},
		Status: aiaudit.StatusCompleted,
	}

	mockDB.EXPECT().
		AIAuditGet(gomock.Any(), auditID).
		Return(expected, nil)

	h := aiaudithandler.NewAIAuditHandler(mockDB, mockGemini)
	got, err := h.Get(context.Background(), auditID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != auditID {
		t.Errorf("expected ID %s, got %s", auditID, got.ID)
	}
	if got.Status != aiaudit.StatusCompleted {
		t.Errorf("expected status %s, got %s", aiaudit.StatusCompleted, got.Status)
	}
}

func TestDelete_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockGemini := geminiaudithandler.NewMockGeminiAuditHandler(ctrl)

	auditID := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	customerID := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")

	existing := &aiaudit.AIAudit{
		Identity: commonidentity.Identity{
			ID:         auditID,
			CustomerID: customerID,
		},
		Status: aiaudit.StatusProgressing,
	}

	mockDB.EXPECT().
		AIAuditGet(gomock.Any(), auditID).
		Return(existing, nil)
	mockDB.EXPECT().
		AIAuditDelete(gomock.Any(), auditID).
		Return(nil)

	h := aiaudithandler.NewAIAuditHandler(mockDB, mockGemini)
	got, err := h.Delete(context.Background(), auditID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != auditID {
		t.Errorf("expected ID %s, got %s", auditID, got.ID)
	}
}

func TestList_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockGemini := geminiaudithandler.NewMockGeminiAuditHandler(ctrl)

	customerID := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")
	records := []*aiaudit.AIAudit{
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
				CustomerID: customerID,
			},
			Status: aiaudit.StatusCompleted,
		},
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777"),
				CustomerID: customerID,
			},
			Status: aiaudit.StatusFailed,
		},
	}

	filters := map[aiaudit.Field]any{
		aiaudit.FieldCustomerID: customerID,
	}

	mockDB.EXPECT().
		AIAuditList(gomock.Any(), uint64(10), "", filters).
		Return(records, nil)

	h := aiaudithandler.NewAIAuditHandler(mockDB, mockGemini)
	got, err := h.List(context.Background(), 10, "", filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 records, got %d", len(got))
	}
	if got[0].Status != aiaudit.StatusCompleted {
		t.Errorf("expected first record status %s, got %s", aiaudit.StatusCompleted, got[0].Status)
	}
	if got[1].Status != aiaudit.StatusFailed {
		t.Errorf("expected second record status %s, got %s", aiaudit.StatusFailed, got[1].Status)
	}
}
