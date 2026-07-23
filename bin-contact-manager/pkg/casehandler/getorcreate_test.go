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

// Test_GetOrCreate_ReusesOpenCase verifies design §4's simplest branch:
// an existing open Case for (customerID, peerType, peerTarget,
// referenceType) that has NOT timed out is reused as-is -- no INSERT, no
// status change. Uses the real SQLite in-memory dbhandler (not a mock)
// because GetOrCreate's core correctness lives in what it does inside a
// real transaction (FOR UPDATE locking, tm_update bump), which a mocked
// DBHandler cannot meaningfully exercise.
func Test_GetOrCreate_ReusesOpenCase(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	config.SetCaseTimeoutHoursForTest(24)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)

	h := &caseHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		db:            db,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7001-7001-7001-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-7001-7001-7001-000000000002")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	recentUpdate := now.Add(-1 * time.Hour) // well within the 24h timeout
	opened := now.Add(-2 * time.Hour)

	existing := &kase.Case{
		ID:            caseID,
		CustomerID:    customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551120001"},
		ReferenceType: "call",
		Status:        kase.StatusOpen,
		OpenedAt:      &opened,
		TMCreate:      &opened,
		TMUpdate:      &recentUpdate,
	}
	if err := db.CaseInsert(ctx, existing); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(&now)

	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551120001"}, "call", nil, "")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != caseID {
		t.Errorf("expected to reuse the existing open case %s, got: %v", caseID, res)
	}

	// tm_update must have been bumped to "now".
	reread, err := db.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if reread.TMUpdate == nil || !reread.TMUpdate.Equal(now) {
		t.Errorf("expected tm_update bumped to %v, got: %v", now, reread.TMUpdate)
	}
}
