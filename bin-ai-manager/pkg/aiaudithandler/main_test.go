package aiaudithandler_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/aiaudit"
	message "monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aiaudithandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/geminiaudithandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestCreate_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockGemini := geminiaudithandler.NewMockGeminiAuditHandler(ctrl)

	customerID := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	aicallID := uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	aiID := uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc")
	auditID := uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd")
	promptHistoryID := uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")

	ac := &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         aicallID,
			CustomerID: customerID,
		},
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   aiID,
		Status:         aicall.StatusTerminated,
		Metadata: map[string]any{
			aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
				{
					AIID:            aiID,
					PromptHistoryID: promptHistoryID,
					Prompt:          "You are a helpful assistant.",
				},
			},
		},
	}

	expectedAudit := &aiaudit.AIAudit{
		Identity: commonidentity.Identity{
			ID:         auditID,
			CustomerID: customerID,
		},
		AIcallID: aicallID,
		AIID:     aiID,
		Status:   aiaudit.StatusProgressing,
	}

	mockDB.EXPECT().AIcallGet(gomock.Any(), aicallID).Return(ac, nil)
	mockDB.EXPECT().AIAuditCountProgressing(gomock.Any(), customerID).Return(int64(0), nil)
	mockDB.EXPECT().AIAuditUpsert(gomock.Any(), gomock.Any()).Return(int64(1), nil)
	mockDB.EXPECT().AIAuditList(gomock.Any(), uint64(1), "", gomock.Any()).Return([]*aiaudit.AIAudit{expectedAudit}, nil)

	// Background goroutine may call these; AnyTimes to prevent test flakiness.
	mockDB.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	mockGemini.EXPECT().Evaluate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil, nil).AnyTimes()
	mockDB.EXPECT().AIAuditUpdateFinal(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1), nil).AnyTimes()

	h := aiaudithandler.NewAIAuditHandler(mockDB, mockGemini)
	got, err := h.Create(context.Background(), customerID, aicallID, "en-US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(got))
	}
	if got[0].ID != auditID {
		t.Errorf("expected audit ID %s, got %s", auditID, got[0].ID)
	}
	if got[0].Status != aiaudit.StatusProgressing {
		t.Errorf("expected status %s, got %s", aiaudit.StatusProgressing, got[0].Status)
	}
}

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

func TestCreate_CompletedPath_StoresMessageIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockGemini := geminiaudithandler.NewMockGeminiAuditHandler(ctrl)

	auditID := uuid.FromStringOrNil("11110001-0001-0001-0001-000000000001")
	customerID := uuid.FromStringOrNil("22220001-0001-0001-0001-000000000001")
	aicallID := uuid.FromStringOrNil("33330001-0001-0001-0001-000000000001")
	aiID := uuid.FromStringOrNil("44440001-0001-0001-0001-000000000001")
	promptHistoryID := uuid.FromStringOrNil("55550001-0001-0001-0001-000000000001")
	msgID1 := uuid.FromStringOrNil("66660001-0001-0001-0001-000000000001")
	msgID2 := uuid.FromStringOrNil("66660001-0001-0001-0001-000000000002")

	ac := &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         aicallID,
			CustomerID: customerID,
		},
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   aiID,
		Status:         aicall.StatusTerminated,
		Metadata: map[string]any{
			aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
				{
					AIID:            aiID,
					PromptHistoryID: promptHistoryID,
					Prompt:          "You are a helpful assistant.",
				},
			},
		},
	}

	expectedAudit := &aiaudit.AIAudit{
		Identity: commonidentity.Identity{
			ID:         auditID,
			CustomerID: customerID,
		},
		AIcallID: aicallID,
		AIID:     aiID,
		Status:   aiaudit.StatusProgressing,
	}

	msgs := []*message.Message{
		{Identity: commonidentity.Identity{ID: msgID1}},
		{Identity: commonidentity.Identity{ID: msgID2}},
	}

	score := 88
	evalResult := &geminiaudithandler.EvaluationResponse{OverallScore: score}
	rawJSON := json.RawMessage(`{"overall_score":88}`)

	resultCh := make(chan []uuid.UUID, 1)

	mockDB.EXPECT().AIcallGet(gomock.Any(), aicallID).Return(ac, nil)
	mockDB.EXPECT().AIAuditCountProgressing(gomock.Any(), customerID).Return(int64(0), nil)
	mockDB.EXPECT().AIAuditUpsert(gomock.Any(), gomock.Any()).Return(int64(1), nil)
	mockDB.EXPECT().AIAuditList(gomock.Any(), uint64(1), "", gomock.Any()).Return([]*aiaudit.AIAudit{expectedAudit}, nil)
	mockDB.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(msgs, nil)
	mockGemini.EXPECT().Evaluate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(evalResult, rawJSON, nil)
	mockDB.EXPECT().AIAuditUpdateFinal(
		gomock.Any(),
		auditID,
		aiaudit.StatusCompleted,
		gomock.Any(),
		gomock.Any(),
		"",
		gomock.Any(),
	).DoAndReturn(func(_ context.Context, _ uuid.UUID, _ aiaudit.Status, _ *int, _ json.RawMessage, _ string, messageIDs []uuid.UUID) (int64, error) {
		resultCh <- messageIDs
		return int64(1), nil
	})

	h := aiaudithandler.NewAIAuditHandler(mockDB, mockGemini)
	_, err := h.Create(context.Background(), customerID, aicallID, "en-US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var receivedIDs []uuid.UUID
	select {
	case receivedIDs = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for background audit goroutine to complete")
	}
	if len(receivedIDs) != 2 {
		t.Fatalf("expected 2 message IDs, got %d", len(receivedIDs))
	}
	if receivedIDs[0] != msgID1 {
		t.Errorf("expected receivedIDs[0] = %s, got %s", msgID1, receivedIDs[0])
	}
	if receivedIDs[1] != msgID2 {
		t.Errorf("expected receivedIDs[1] = %s, got %s", msgID2, receivedIDs[1])
	}
}

// TestCreate_FailedPath_MessageIDsNil asserts that messageIDs is nil when Gemini
// returns an error — finalMsgIDs must never be populated on a non-completed path.
func TestCreate_FailedPath_MessageIDsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockGemini := geminiaudithandler.NewMockGeminiAuditHandler(ctrl)

	auditID := uuid.FromStringOrNil("88880001-0001-0001-0001-000000000001")
	customerID := uuid.FromStringOrNil("99990001-0001-0001-0001-000000000001")
	aicallID := uuid.FromStringOrNil("aaaa0001-0001-0001-0001-000000000001")
	aiID := uuid.FromStringOrNil("bbbb0001-0001-0001-0001-000000000001")
	promptHistoryID := uuid.FromStringOrNil("cccc0001-0001-0001-0001-000000000001")
	msgID1 := uuid.FromStringOrNil("dddd0001-0001-0001-0001-000000000001")

	ac := &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         aicallID,
			CustomerID: customerID,
		},
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   aiID,
		Status:         aicall.StatusTerminated,
		Metadata: map[string]any{
			aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
				{
					AIID:            aiID,
					PromptHistoryID: promptHistoryID,
					Prompt:          "You are a helpful assistant.",
				},
			},
		},
	}

	expectedAudit := &aiaudit.AIAudit{
		Identity: commonidentity.Identity{
			ID:         auditID,
			CustomerID: customerID,
		},
		AIcallID: aicallID,
		AIID:     aiID,
		Status:   aiaudit.StatusProgressing,
	}

	// MessageList returns one message, but Evaluate will fail — IDs must NOT be stored.
	msgs := []*message.Message{
		{Identity: commonidentity.Identity{ID: msgID1}},
	}

	resultCh := make(chan []uuid.UUID, 1)

	mockDB.EXPECT().AIcallGet(gomock.Any(), aicallID).Return(ac, nil)
	mockDB.EXPECT().AIAuditCountProgressing(gomock.Any(), customerID).Return(int64(0), nil)
	mockDB.EXPECT().AIAuditUpsert(gomock.Any(), gomock.Any()).Return(int64(1), nil)
	mockDB.EXPECT().AIAuditList(gomock.Any(), uint64(1), "", gomock.Any()).Return([]*aiaudit.AIAudit{expectedAudit}, nil)
	mockDB.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(msgs, nil)
	// Gemini fails — Step-7 success block is never reached.
	mockGemini.EXPECT().Evaluate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil, context.DeadlineExceeded)
	mockDB.EXPECT().AIAuditUpdateFinal(
		gomock.Any(),
		auditID,
		aiaudit.StatusFailed,
		gomock.Nil(),
		gomock.Nil(),
		gomock.Any(),
		gomock.Any(),
	).DoAndReturn(func(_ context.Context, _ uuid.UUID, _ aiaudit.Status, _ *int, _ json.RawMessage, _ string, messageIDs []uuid.UUID) (int64, error) {
		resultCh <- messageIDs
		return int64(1), nil
	})

	h := aiaudithandler.NewAIAuditHandler(mockDB, mockGemini)
	_, err := h.Create(context.Background(), customerID, aicallID, "en-US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var receivedIDs []uuid.UUID
	select {
	case receivedIDs = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for background audit goroutine to complete")
	}
	if receivedIDs != nil {
		t.Errorf("expected nil messageIDs on failed path, got %v", receivedIDs)
	}
}
