package aipromptproposalhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/aiprompthistory"
	"monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/geminiproposalhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

var (
	g4Cust     = uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	g4OtherCus = uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	g4AIID     = uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	g4PID      = uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
	g4AuditID  = uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")
	g4HistID   = uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666")
)

// g4WaitDone waits for ch to be closed or fails after a bounded timeout.
func g4WaitDone(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatalf("timed out waiting for background job")
	}
}

func Test_Get_Success(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	want := &aipromptproposal.AIPromptProposal{Identity: commonidentity.Identity{ID: g4PID, CustomerID: g4Cust}}
	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(want, nil)

	res, err := h.Get(context.Background(), g4PID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res != want {
		t.Errorf("result mismatch: %v", res)
	}
}

func Test_Get_DBError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	dbErr := errors.New("db down")
	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(nil, dbErr)

	res, err := h.Get(context.Background(), g4PID)
	if res != nil || !errors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not get proposal") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_List_Success(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	filters := map[aipromptproposal.Field]any{aipromptproposal.FieldDeleted: false}
	want := []*aipromptproposal.AIPromptProposal{{Identity: commonidentity.Identity{ID: g4PID}}}
	mdb.EXPECT().AIPromptProposalList(gomock.Any(), uint64(10), "token-a", filters).Return(want, nil)

	res, err := h.List(context.Background(), 10, "token-a", filters)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(res) != 1 || res[0].ID != g4PID {
		t.Errorf("result mismatch: %v", res)
	}
}

func Test_List_DBError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	dbErr := errors.New("db down")
	mdb.EXPECT().AIPromptProposalList(gomock.Any(), uint64(5), "", gomock.Any()).Return(nil, dbErr)

	res, err := h.List(context.Background(), 5, "", nil)
	if res != nil || !errors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not list proposals") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Delete_Success_ReturnsPreDeleteState(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	pre := &aipromptproposal.AIPromptProposal{
		Identity: commonidentity.Identity{ID: g4PID, CustomerID: g4Cust},
		Status:   aipromptproposal.StatusCompleted,
	}
	gomock.InOrder(
		mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(pre, nil),
		mdb.EXPECT().AIPromptProposalDelete(gomock.Any(), g4PID).Return(nil),
	)

	res, err := h.Delete(context.Background(), g4PID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res != pre || res.Status != aipromptproposal.StatusCompleted {
		t.Errorf("result mismatch: %v", res)
	}
}

func Test_Delete_GetError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Delete(context.Background(), g4PID)
	if res != nil || !errors.Is(err, dbhandler.ErrNotFound) || !strings.Contains(err.Error(), "before delete") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Delete_DeleteError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	dbErr := errors.New("delete failed")
	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(&aipromptproposal.AIPromptProposal{}, nil)
	mdb.EXPECT().AIPromptProposalDelete(gomock.Any(), g4PID).Return(dbErr)

	res, err := h.Delete(context.Background(), g4PID)
	if res != nil || !errors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not delete proposal") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Reject_Success(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	rejected := &aipromptproposal.AIPromptProposal{
		Identity: commonidentity.Identity{ID: g4PID, CustomerID: g4Cust},
		Status:   aipromptproposal.StatusRejected,
	}
	gomock.InOrder(
		mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(&aipromptproposal.AIPromptProposal{
			Identity: commonidentity.Identity{ID: g4PID, CustomerID: g4Cust},
			Status:   aipromptproposal.StatusCompleted,
		}, nil),
		mdb.EXPECT().AIPromptProposalUpdateRejected(gomock.Any(), g4PID).Return(int64(1), nil),
		mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(rejected, nil),
	)

	res, err := h.Reject(context.Background(), g4Cust, g4PID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Status != aipromptproposal.StatusRejected {
		t.Errorf("status mismatch: %s", res.Status)
	}
}

func Test_Reject_AlreadyRejected_Idempotent(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(&aipromptproposal.AIPromptProposal{
		Identity: commonidentity.Identity{ID: g4PID, CustomerID: g4Cust},
		Status:   aipromptproposal.StatusRejected,
	}, nil)

	res, err := h.Reject(context.Background(), g4Cust, g4PID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Status != aipromptproposal.StatusRejected {
		t.Errorf("status mismatch: %s", res.Status)
	}
}

func Test_Reject_GetError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Reject(context.Background(), g4Cust, g4PID)
	if res != nil || !errors.Is(err, dbhandler.ErrNotFound) || !strings.Contains(err.Error(), "proposal not found") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Reject_OtherCustomer_NotFound(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(&aipromptproposal.AIPromptProposal{
		Identity: commonidentity.Identity{ID: g4PID, CustomerID: g4OtherCus},
		Status:   aipromptproposal.StatusCompleted,
	}, nil)

	res, err := h.Reject(context.Background(), g4Cust, g4PID)
	if res != nil || err == nil || err.Error() != "proposal not found" {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Reject_NotCompleted(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(&aipromptproposal.AIPromptProposal{
		Identity: commonidentity.Identity{ID: g4PID, CustomerID: g4Cust},
		Status:   aipromptproposal.StatusProgressing,
	}, nil)

	res, err := h.Reject(context.Background(), g4Cust, g4PID)
	if res != nil || err == nil || !strings.Contains(err.Error(), "status=progressing") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Reject_UpdateError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	dbErr := errors.New("update failed")
	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(&aipromptproposal.AIPromptProposal{
		Identity: commonidentity.Identity{ID: g4PID, CustomerID: g4Cust},
		Status:   aipromptproposal.StatusCompleted,
	}, nil)
	mdb.EXPECT().AIPromptProposalUpdateRejected(gomock.Any(), g4PID).Return(int64(0), dbErr)

	res, err := h.Reject(context.Background(), g4Cust, g4PID)
	if res != nil || !errors.Is(err, dbErr) {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Reject_RaceNoRowUpdated(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(&aipromptproposal.AIPromptProposal{
		Identity: commonidentity.Identity{ID: g4PID, CustomerID: g4Cust},
		Status:   aipromptproposal.StatusCompleted,
	}, nil)
	mdb.EXPECT().AIPromptProposalUpdateRejected(gomock.Any(), g4PID).Return(int64(0), nil)

	res, err := h.Reject(context.Background(), g4Cust, g4PID)
	if res != nil || err == nil || !strings.Contains(err.Error(), "race") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func g4CompletedProposal() *aipromptproposal.AIPromptProposal {
	return &aipromptproposal.AIPromptProposal{
		Identity:       commonidentity.Identity{ID: g4PID, CustomerID: g4Cust},
		Status:         aipromptproposal.StatusCompleted,
		AuditIDs:       []uuid.UUID{g4AuditID},
		ProposedPrompt: "new prompt",
	}
}

func Test_Accept_Success(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	h.utilHandler = utilhandler.NewMockUtilHandler(mc)
	h.utilHandler.(*utilhandler.MockUtilHandler).EXPECT().UUIDCreate().Return(g4HistID)

	accepted := &aipromptproposal.AIPromptProposal{
		Identity:               commonidentity.Identity{ID: g4PID, CustomerID: g4Cust},
		Status:                 aipromptproposal.StatusAccepted,
		AppliedPromptHistoryID: g4HistID,
	}
	gomock.InOrder(
		mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(g4CompletedProposal(), nil),
		mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil),
		mdb.EXPECT().AIAcceptProposal(gomock.Any(), g4PID, g4HistID, "new prompt").Return(nil),
		mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(accepted, nil),
	)

	res, err := h.Accept(context.Background(), g4Cust, g4PID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.AppliedPromptHistoryID != g4HistID || res.Status != aipromptproposal.StatusAccepted {
		t.Errorf("result mismatch: %v", res)
	}
}

func Test_Accept_GetError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Accept(context.Background(), g4Cust, g4PID)
	if res != nil || !errors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Accept_OtherCustomer_NotFound(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	p := g4CompletedProposal()
	p.CustomerID = g4OtherCus
	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(p, nil)

	res, err := h.Accept(context.Background(), g4Cust, g4PID)
	if res != nil || err == nil || err.Error() != "proposal not found" {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Accept_AuditNotCompleted_MarksExpiredEvenIfUpdateFails(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(g4CompletedProposal(), nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(nil, dbhandler.ErrNotFound)
	mdb.EXPECT().AIPromptProposalUpdateExpired(gomock.Any(), g4PID, string(aipromptproposal.ErrorInvalidAuditSet)).
		Return(int64(0), errors.New("update failed"))

	res, err := h.Accept(context.Background(), g4Cust, g4PID)
	if res != nil || err == nil || err.Error() != "audit set invalidated" {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Accept_DriftedAndUpdateExpiredFails(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	h.utilHandler = utilhandler.NewMockUtilHandler(mc)
	h.utilHandler.(*utilhandler.MockUtilHandler).EXPECT().UUIDCreate().Return(g4HistID)

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(g4CompletedProposal(), nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil)
	mdb.EXPECT().AIAcceptProposal(gomock.Any(), g4PID, g4HistID, "new prompt").Return(dbhandler.ErrPromptVersionDrifted)
	mdb.EXPECT().AIPromptProposalUpdateExpired(gomock.Any(), g4PID, string(aipromptproposal.ErrorPromptVersionDrifted)).
		Return(int64(0), errors.New("update failed"))

	res, err := h.Accept(context.Background(), g4Cust, g4PID)
	if res != nil || err == nil || err.Error() != "prompt version drifted" {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Accept_AcceptProposalErrors(t *testing.T) {
	genericErr := errors.New("tx failed")
	tests := []struct {
		name      string
		acceptErr error
		wantMsg   string
		wantIs    error
	}{
		{"not found", dbhandler.ErrNotFound, "proposal not found", nil},
		{"not acceptable", dbhandler.ErrProposalNotAcceptable, "proposal not completed", nil},
		{"generic", genericErr, "could not apply proposal", genericErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mdb, _, mc := newHandlerWithMocks(t)
			defer mc.Finish()
			h.utilHandler = utilhandler.NewMockUtilHandler(mc)
			h.utilHandler.(*utilhandler.MockUtilHandler).EXPECT().UUIDCreate().Return(g4HistID)

			mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(g4CompletedProposal(), nil)
			mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil)
			mdb.EXPECT().AIAcceptProposal(gomock.Any(), g4PID, g4HistID, "new prompt").Return(tt.acceptErr)

			res, err := h.Accept(context.Background(), g4Cust, g4PID)
			if res != nil || err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("unexpected result: %v, err: %v", res, err)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("error chain does not contain cause: %v", err)
			}
		})
	}
}

func Test_Accept_AlreadyAcceptedReloadError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	h.utilHandler = utilhandler.NewMockUtilHandler(mc)
	h.utilHandler.(*utilhandler.MockUtilHandler).EXPECT().UUIDCreate().Return(g4HistID)

	reloadErr := errors.New("reload failed")
	gomock.InOrder(
		mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(g4CompletedProposal(), nil),
		mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil),
		mdb.EXPECT().AIAcceptProposal(gomock.Any(), g4PID, g4HistID, "new prompt").Return(dbhandler.ErrProposalAlreadyAccepted),
		mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(nil, reloadErr),
	)

	res, err := h.Accept(context.Background(), g4Cust, g4PID)
	if res != nil || !errors.Is(err, reloadErr) || !strings.Contains(err.Error(), "already-accepted") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Accept_FinalReloadError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	h.utilHandler = utilhandler.NewMockUtilHandler(mc)
	h.utilHandler.(*utilhandler.MockUtilHandler).EXPECT().UUIDCreate().Return(g4HistID)

	reloadErr := errors.New("reload failed")
	gomock.InOrder(
		mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(g4CompletedProposal(), nil),
		mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil),
		mdb.EXPECT().AIAcceptProposal(gomock.Any(), g4PID, g4HistID, "new prompt").Return(nil),
		mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(nil, reloadErr),
	)

	res, err := h.Accept(context.Background(), g4Cust, g4PID)
	if res != nil || !errors.Is(err, reloadErr) || !strings.Contains(err.Error(), "reload accepted proposal") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func g4AIFor(cust uuid.UUID) *ai.AI {
	return &ai.AI{
		Identity:               commonidentity.Identity{ID: g4AIID, CustomerID: cust},
		CurrentPromptHistoryID: g4HistID,
	}
}

func Test_Create_AIGetError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	dbErr := errors.New("db down")
	mdb.EXPECT().AIGet(gomock.Any(), g4AIID).Return(nil, dbErr)

	res, err := h.Create(context.Background(), g4Cust, g4AIID, []uuid.UUID{g4AuditID}, "en-US")
	if res != nil || !errors.Is(err, dbErr) || !strings.Contains(err.Error(), "ai not found") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Create_AIDeleted(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	a := g4AIFor(g4Cust)
	del := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	a.TMDelete = &del
	mdb.EXPECT().AIGet(gomock.Any(), g4AIID).Return(a, nil)

	res, err := h.Create(context.Background(), g4Cust, g4AIID, []uuid.UUID{g4AuditID}, "en-US")
	if res != nil || err == nil || err.Error() != "ai not found" {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Create_CountError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	dbErr := errors.New("count failed")
	mdb.EXPECT().AIGet(gomock.Any(), g4AIID).Return(g4AIFor(g4Cust), nil)
	mdb.EXPECT().AIPromptProposalCountProgressing(gomock.Any(), g4Cust).Return(int64(0), dbErr)

	res, err := h.Create(context.Background(), g4Cust, g4AIID, []uuid.UUID{g4AuditID}, "en-US")
	if res != nil || !errors.Is(err, dbErr) {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Create_CountBelowLimit_Proceeds(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	// count one below the limit must not be rate limited; the flow continues to the audit lookup.
	mdb.EXPECT().AIGet(gomock.Any(), g4AIID).Return(g4AIFor(g4Cust), nil)
	mdb.EXPECT().AIPromptProposalCountProgressing(gomock.Any(), g4Cust).Return(int64(maxConcurrentCustomer-1), nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(nil, dbhandler.ErrNotFound)

	_, err := h.Create(context.Background(), g4Cust, g4AIID, []uuid.UUID{g4AuditID}, "en-US")
	if err == nil || !strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "rate limit") {
		t.Errorf("unexpected err: %v", err)
	}
}

func Test_Create_AuditGetError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIGet(gomock.Any(), g4AIID).Return(g4AIFor(g4Cust), nil)
	mdb.EXPECT().AIPromptProposalCountProgressing(gomock.Any(), g4Cust).Return(int64(0), nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Create(context.Background(), g4Cust, g4AIID, []uuid.UUID{g4AuditID}, "en-US")
	if res != nil || err == nil || !strings.Contains(err.Error(), g4AuditID.String()+" not found") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

// g4OKAudit returns a completed audit matching the default AI and prompt history.
func g4OKAudit() *aiaudit.AIAudit {
	return &aiaudit.AIAudit{
		Identity:        commonidentity.Identity{ID: g4AuditID, CustomerID: g4Cust},
		AIID:            g4AIID,
		Status:          aiaudit.StatusCompleted,
		PromptHistoryID: g4HistID,
	}
}

func Test_Create_BasisHistoryError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	dbErr := errors.New("history failed")
	mdb.EXPECT().AIGet(gomock.Any(), g4AIID).Return(g4AIFor(g4Cust), nil)
	mdb.EXPECT().AIPromptProposalCountProgressing(gomock.Any(), g4Cust).Return(int64(0), nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(g4OKAudit(), nil)
	mdb.EXPECT().AIPromptHistoryGet(gomock.Any(), g4HistID).Return(nil, dbErr)

	res, err := h.Create(context.Background(), g4Cust, g4AIID, []uuid.UUID{g4AuditID}, "en-US")
	if res != nil || !errors.Is(err, dbErr) || !strings.Contains(err.Error(), "basis prompt history") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func g4SetupCreateUntilInsert(mdb *dbhandler.MockDBHandler, mu *utilhandler.MockUtilHandler) {
	mdb.EXPECT().AIGet(gomock.Any(), g4AIID).Return(g4AIFor(g4Cust), nil)
	mdb.EXPECT().AIPromptProposalCountProgressing(gomock.Any(), g4Cust).Return(int64(0), nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(g4OKAudit(), nil)
	mdb.EXPECT().AIPromptHistoryGet(gomock.Any(), g4HistID).Return(&aiprompthistory.AIPromptHistory{Prompt: "basis prompt"}, nil)
	mu.EXPECT().UUIDCreate().Return(g4PID)
}

func Test_Create_InsertError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	mu := utilhandler.NewMockUtilHandler(mc)
	h.utilHandler = mu
	g4SetupCreateUntilInsert(mdb, mu)

	dbErr := errors.New("insert failed")
	mdb.EXPECT().AIPromptProposalCreate(gomock.Any(), gomock.Any()).Return(dbErr)

	res, err := h.Create(context.Background(), g4Cust, g4AIID, []uuid.UUID{g4AuditID}, "en-US")
	if res != nil || !errors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not create proposal") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Create_ReloadError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	mu := utilhandler.NewMockUtilHandler(mc)
	h.utilHandler = mu
	g4SetupCreateUntilInsert(mdb, mu)

	dbErr := errors.New("reload failed")
	mdb.EXPECT().AIPromptProposalCreate(gomock.Any(), gomock.Any()).Return(nil)
	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(nil, dbErr)

	res, err := h.Create(context.Background(), g4Cust, g4AIID, []uuid.UUID{g4AuditID}, "en-US")
	if res != nil || !errors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not reload proposal") {
		t.Errorf("unexpected result: %v, err: %v", res, err)
	}
}

func Test_Create_Success_StartsJob(t *testing.T) {
	// The controller finishes through its own t.Cleanup, which runs after the wait registered below.
	h, mdb, mg, mc := newHandlerWithMocks(t)
	mu := utilhandler.NewMockUtilHandler(mc)
	h.utilHandler = mu
	g4SetupCreateUntilInsert(mdb, mu)

	done := make(chan struct{})
	// Registered after the controller so it runs before the controller's own cleanup.
	t.Cleanup(func() { g4WaitDone(t, done) })

	mdb.EXPECT().AIPromptProposalCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p *aipromptproposal.AIPromptProposal) error {
			if p.ID != g4PID || p.CustomerID != g4Cust || p.AIID != g4AIID ||
				p.BasisPromptHistoryID != g4HistID || p.OriginalPrompt != "basis prompt" ||
				len(p.AuditIDs) != 1 || p.AuditIDs[0] != g4AuditID {
				t.Errorf("unexpected proposal row: %+v", p)
			}
			return nil
		})
	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), g4PID).Return(&aipromptproposal.AIPromptProposal{
		Identity: commonidentity.Identity{ID: g4PID, CustomerID: g4Cust},
		Status:   aipromptproposal.StatusProgressing,
	}, nil)

	// Detached job.
	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil)
	mdb.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
	mg.EXPECT().Evaluate(gomock.Any(), "basis prompt", gomock.Any(), "ko-KR").
		Return(&geminiproposalhandler.ProposalResponse{ProposedPrompt: "p2", Rationale: "r2"}, nil)
	mdb.EXPECT().AIPromptProposalUpdateFinal(gomock.Any(), g4PID, aipromptproposal.StatusCompleted, "p2", "r2", "").
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ aipromptproposal.Status, _, _, _ string) (int64, error) {
			close(done)
			return 1, nil
		})

	res, err := h.Create(context.Background(), g4Cust, g4AIID, []uuid.UUID{g4AuditID}, "ko-KR")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.ID != g4PID || res.Status != aipromptproposal.StatusProgressing {
		t.Errorf("result mismatch: %+v", res)
	}
}

func Test_runProposalJob_ParentCancelled_WritesCancelled(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mdb.EXPECT().AIPromptProposalUpdateFinal(gomock.Any(), g4PID, aipromptproposal.StatusFailed, "", "", string(aipromptproposal.ErrorCancelled)).
		Return(int64(1), nil)

	h.runProposalJob(ctx, g4PID, "orig", []uuid.UUID{g4AuditID}, "en-US")

	if len(h.semaphore) != 0 {
		t.Errorf("semaphore slot not released: %d", len(h.semaphore))
	}
}

func Test_runProposalJob_LoadBlocksError_WritesEvaluatorUnavailable(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(nil, errors.New("db down"))
	mdb.EXPECT().AIPromptProposalUpdateFinal(gomock.Any(), g4PID, aipromptproposal.StatusFailed, "", "", string(aipromptproposal.ErrorEvaluatorUnavailable)).
		Return(int64(1), nil)

	h.runProposalJob(context.Background(), g4PID, "orig", []uuid.UUID{g4AuditID}, "en-US")
}

func Test_runProposalJob_FinalWriteError_StillReleasesSemaphore(t *testing.T) {
	h, mdb, mg, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil)
	mdb.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
	mg.EXPECT().Evaluate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&geminiproposalhandler.ProposalResponse{ProposedPrompt: "p", Rationale: "r"}, nil)
	mdb.EXPECT().AIPromptProposalUpdateFinal(gomock.Any(), g4PID, aipromptproposal.StatusCompleted, "p", "r", "").
		Return(int64(0), errors.New("write failed"))

	h.runProposalJob(context.Background(), g4PID, "orig", []uuid.UUID{g4AuditID}, "en-US")

	if len(h.semaphore) != 0 {
		t.Errorf("semaphore slot not released: %d", len(h.semaphore))
	}
}

func Test_runProposalJob_FinalWriteNoRows(t *testing.T) {
	h, mdb, mg, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil)
	mdb.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
	mg.EXPECT().Evaluate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&geminiproposalhandler.ProposalResponse{ProposedPrompt: "p", Rationale: "r"}, nil)
	mdb.EXPECT().AIPromptProposalUpdateFinal(gomock.Any(), g4PID, aipromptproposal.StatusCompleted, "p", "r", "").
		Return(int64(0), nil)

	h.runProposalJob(context.Background(), g4PID, "orig", []uuid.UUID{g4AuditID}, "en-US")

	if len(h.semaphore) != 0 {
		t.Errorf("semaphore slot not released: %d", len(h.semaphore))
	}
}

func Test_runProposalJob_EvaluatorPanic_WritesFailed(t *testing.T) {
	h, mdb, mg, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil)
	mdb.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
	mg.EXPECT().Evaluate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, string, []geminiproposalhandler.AuditBlock, string) (*geminiproposalhandler.ProposalResponse, error) {
			panic("synthetic evaluator panic")
		})
	mdb.EXPECT().AIPromptProposalUpdateFinal(gomock.Any(), g4PID, aipromptproposal.StatusFailed, "", "", string(aipromptproposal.ErrorEvaluatorUnavailable)).
		Return(int64(1), nil)

	h.runProposalJob(context.Background(), g4PID, "orig", []uuid.UUID{g4AuditID}, "en-US")

	if len(h.semaphore) != 0 {
		t.Errorf("semaphore slot not released: %d", len(h.semaphore))
	}
}

func Test_loadAuditBlocks_Success(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	callID := uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777")
	score := 87
	eval := []byte(`{"summary":"sum","dimensions":{"helpfulness":{"reason":"h"},"accuracy":{"reason":"a"},"tone":{"reason":"t"},"goal_completion":{"reason":"g"},"tool_usage":{"reason":"u"}}}`)
	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{
		Status:       aiaudit.StatusCompleted,
		AIcallID:     callID,
		OverallScore: &score,
		Evaluation:   eval,
	}, nil)
	mdb.EXPECT().MessageList(gomock.Any(), uint64(500), "", map[message.Field]any{
		message.FieldAIcallID: callID,
		message.FieldDeleted:  false,
	}).Return([]*message.Message{
		{Role: message.RoleUser, Content: "hello"},
		{Role: message.RoleAssistant, Content: "hi"},
	}, nil)

	blocks, err := h.loadAuditBlocks(context.Background(), []uuid.UUID{g4AuditID})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := geminiproposalhandler.AuditBlock{
		Index:           1,
		OverallScore:    87,
		HelpfulnessR:    "h",
		AccuracyR:       "a",
		ToneR:           "t",
		GoalCompletionR: "g",
		ToolUsageR:      "u",
		Summary:         "sum",
		Transcript:      "[user]: hello\n[assistant]: hi\n",
	}
	if len(blocks) != 1 || blocks[0] != want {
		t.Errorf("block mismatch: %+v", blocks)
	}
}

func Test_loadAuditBlocks_SkipsDeletedAndIncomplete(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	idDeleted := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	idPending := uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	del := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	mdb.EXPECT().AIAuditGet(gomock.Any(), idDeleted).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted, TMDelete: &del}, nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), idPending).Return(&aiaudit.AIAudit{Status: aiaudit.StatusProgressing}, nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil)
	mdb.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	blocks, err := h.loadAuditBlocks(context.Background(), []uuid.UUID{idDeleted, idPending, g4AuditID})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// The Index keeps the position in the requested list.
	if len(blocks) != 1 || blocks[0].Index != 3 || blocks[0].OverallScore != 0 {
		t.Errorf("block mismatch: %+v", blocks)
	}
}

func Test_loadAuditBlocks_ParseError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{
		Status:     aiaudit.StatusCompleted,
		Evaluation: []byte(`{not json`),
	}, nil)

	blocks, err := h.loadAuditBlocks(context.Background(), []uuid.UUID{g4AuditID})
	if blocks != nil || err == nil || !strings.Contains(err.Error(), "could not parse evaluation") {
		t.Errorf("unexpected result: %v, err: %v", blocks, err)
	}
}

func Test_loadAuditBlocks_MessageListError(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	dbErr := errors.New("messages failed")
	mdb.EXPECT().AIAuditGet(gomock.Any(), g4AuditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil)
	mdb.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbErr)

	blocks, err := h.loadAuditBlocks(context.Background(), []uuid.UUID{g4AuditID})
	if blocks != nil || !errors.Is(err, dbErr) || !strings.Contains(err.Error(), "could not load messages") {
		t.Errorf("unexpected result: %v, err: %v", blocks, err)
	}
}

func Test_derefScore(t *testing.T) {
	v := 42
	if got := derefScore(nil); got != 0 {
		t.Errorf("nil score: got %d", got)
	}
	if got := derefScore(&v); got != 42 {
		t.Errorf("score: got %d", got)
	}
}

func Test_parseAuditEvaluation(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    evalDims
		wantErr bool
	}{
		{name: "empty", raw: ``, want: evalDims{}},
		{name: "invalid json", raw: `{`, wantErr: true},
		{
			name: "without tool usage",
			raw:  `{"summary":"s","dimensions":{"tone":{"reason":"t"}}}`,
			want: evalDims{Tone: evalDimReason{Reason: "t"}, Summary: "s"},
		},
		{
			name: "with tool usage",
			raw:  `{"dimensions":{"tool_usage":{"reason":"u"}}}`,
			want: evalDims{ToolUsageR: "u"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAuditEvaluation([]byte(tt.raw))
			if tt.wantErr {
				if err == nil || got != nil {
					t.Fatalf("expected error, got %v, %v", got, err)
				}
				return
			}
			if err != nil || *got != tt.want {
				t.Errorf("got %+v, err %v; want %+v", got, err, tt.want)
			}
		})
	}
}

func Test_buildTranscript(t *testing.T) {
	msgs := []*message.Message{
		{Role: message.RoleUser, Content: "aaaa"},
		{Role: message.RoleAssistant, Content: "bbbb"},
		{Role: message.RoleUser, Content: "cccc"},
	}
	full := "[user]: aaaa\n[assistant]: bbbb\n[user]: cccc\n"

	tests := []struct {
		name     string
		msgs     []*message.Message
		maxChars int
		want     string
	}{
		{"empty", nil, 100, ""},
		{"fits", msgs, len(full), full},
		{"stops after limit reached", msgs, 10, "[user]: aaaa\n[assistant]: bbbb\n"[:10] + "...(truncated)"},
		{"one char under total is truncated", msgs, len(full) - 1, full[:len(full)-1] + "...(truncated)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildTranscript(tt.msgs, tt.maxChars); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
