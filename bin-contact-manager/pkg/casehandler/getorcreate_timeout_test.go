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

// Test_GetOrCreate_TimedOutCase_ClosesAndReopens verifies design §4's
// timeout branch: an open Case whose tm_update is older than
// CASE_TIMEOUT_HOURS is closed (closed_reason='timeout',
// closed_by_type='system') and a fresh Case is inserted, chained to it
// via previous_case_id.
func Test_GetOrCreate_TimedOutCase_ClosesAndReopens(t *testing.T) {
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

	customerID := uuid.FromStringOrNil("f1b2c3d4-7004-7004-7004-000000000001")
	staleCaseID := uuid.FromStringOrNil("f1b2c3d4-7004-7004-7004-000000000002")
	newCaseID := uuid.FromStringOrNil("f1b2c3d4-7004-7004-7004-000000000003")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	staleUpdate := now.Add(-25 * time.Hour) // older than the 24h timeout
	opened := now.Add(-30 * time.Hour)

	stale := &kase.Case{
		ID: staleCaseID, CustomerID: customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551140001"}, ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &staleUpdate,
	}
	if err := db.CaseInsert(ctx, stale); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockUtil.EXPECT().UUIDCreate().Return(newCaseID)

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551140001"}, "call", nil)
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != newCaseID {
		t.Errorf("expected a fresh case %s, got: %v", newCaseID, res)
	}
	if res.PreviousCaseID == nil || *res.PreviousCaseID != staleCaseID {
		t.Errorf("expected previous_case_id chained to the timed-out case %s, got: %v", staleCaseID, res.PreviousCaseID)
	}

	// The stale case must actually be closed with reason='timeout'.
	reread, err := db.CaseGetByID(ctx, staleCaseID)
	if err != nil {
		t.Fatalf("CaseGetByID(stale) error = %v", err)
	}
	if reread.Status != kase.StatusClosed {
		t.Errorf("expected stale case status closed, got: %s", reread.Status)
	}
	if reread.ClosedReason != kase.ClosedReasonTimeout {
		t.Errorf("expected closed_reason timeout, got: %s", reread.ClosedReason)
	}
	if reread.ClosedByType != kase.ClosedByTypeSystem {
		t.Errorf("expected closed_by_type system, got: %s", reread.ClosedByType)
	}

	// The new case must be open and NOT the same row as the stale one.
	newReread, err := db.CaseGetByID(ctx, newCaseID)
	if err != nil {
		t.Fatalf("CaseGetByID(new) error = %v", err)
	}
	if newReread.Status != kase.StatusOpen {
		t.Errorf("expected new case status open, got: %s", newReread.Status)
	}
}
