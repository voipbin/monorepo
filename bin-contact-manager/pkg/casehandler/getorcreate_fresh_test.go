package casehandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/internal/config"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_GetOrCreate_FreshInsert_NoPriorCase verifies design §4's
// fresh-insert branch: a peer with NO prior Case at all (no open row,
// no closed row) gets a brand-new Case with previous_case_id nil.
func Test_GetOrCreate_FreshInsert_NoPriorCase(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	config.SetCaseTimeoutHoursForTest(24)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7005-7005-7005-000000000001")
	newCaseID := uuid.FromStringOrNil("f1b2c3d4-7005-7005-7005-000000000002")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(newCaseID)

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.TypeTel, "+15551150001", "call", nil)
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != newCaseID {
		t.Errorf("expected a fresh case %s, got: %v", newCaseID, res)
	}
	if res.PreviousCaseID != nil {
		t.Errorf("expected previous_case_id nil for a peer with no prior case, got: %v", *res.PreviousCaseID)
	}
	if res.Status != kase.StatusOpen {
		t.Errorf("expected status open, got: %s", res.Status)
	}
	if res.OpenedAt == nil || !res.OpenedAt.Equal(now) {
		t.Errorf("expected opened_at: %v, got: %v", now, res.OpenedAt)
	}
}

// Test_GetOrCreate_FreshInsert_ChainsToLastClosed verifies design §4's
// "not found" branch chains previous_case_id to the most recently
// closed Case for this peer, when one exists (re-contact after a fully
// resolved-and-closed prior Case, no open row at all).
func Test_GetOrCreate_FreshInsert_ChainsToLastClosed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	config.SetCaseTimeoutHoursForTest(24)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7006-7006-7006-000000000001")
	oldClosedCaseID := uuid.FromStringOrNil("f1b2c3d4-7006-7006-7006-000000000002")
	newCaseID := uuid.FromStringOrNil("f1b2c3d4-7006-7006-7006-000000000003")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	opened := now.Add(-72 * time.Hour)
	closedAt := now.Add(-48 * time.Hour)

	oldClosed := &kase.Case{
		ID: oldClosedCaseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551150002", ReferenceType: "call",
		Status: kase.StatusClosed, OpenedAt: &opened, ClosedAt: &closedAt,
		ClosedReason: kase.ClosedReasonAgentClosed, ClosedByType: kase.ClosedByTypeAgent,
		TMCreate: &opened, TMUpdate: &closedAt,
	}
	if err := db.CaseInsert(ctx, oldClosed); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(newCaseID)

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.TypeTel, "+15551150002", "call", nil)
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != newCaseID {
		t.Errorf("expected a fresh case %s, got: %v", newCaseID, res)
	}
	if res.PreviousCaseID == nil || *res.PreviousCaseID != oldClosedCaseID {
		t.Errorf("expected previous_case_id chained to %s, got: %v", oldClosedCaseID, res.PreviousCaseID)
	}
}
