package casehandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_Close_FirstCallSucceeds verifies design §5.1's normal path: the
// first close actually persists and the response reflects it.
func Test_Close_FirstCallSucceeds(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9001-9001-9001-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9001-9001-9001-000000000002")
	agentID := uuid.FromStringOrNil("f1b2c3d4-9001-9001-9001-000000000003")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551300001", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(&now)

	res, err := h.Close(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, agentID)
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if res.ClosedReason != kase.ClosedReasonAgentClosed {
		t.Errorf("expected closed_reason agent_closed, got: %s", res.ClosedReason)
	}
	if res.ClosedByType != string(commonidentity.OwnerTypeAgent) {
		t.Errorf("expected closed_by_type agent, got: %s", res.ClosedByType)
	}
	if res.ClosedByID == nil || *res.ClosedByID != agentID {
		t.Errorf("expected closed_by_id: %s, got: %v", agentID, res.ClosedByID)
	}
	if res.Case.Status != kase.StatusClosed {
		t.Errorf("expected status closed, got: %s", res.Case.Status)
	}
}

// Test_Close_DoubleClose_ReturnsTruthfulPersistedState verifies design
// §5.1's corrected requirement: a second close call against an
// already-closed case must NOT error, and must NOT claim the caller's
// own action won -- it must return the ACTUALLY persisted
// closed_reason/closed_by (from the first, real close), with
// already_closed indicated.
func Test_Close_DoubleClose_ReturnsTruthfulPersistedState(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9002-9002-9002-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9002-9002-9002-000000000002")
	firstAgentID := uuid.FromStringOrNil("f1b2c3d4-9002-9002-9002-000000000003")
	secondAgentID := uuid.FromStringOrNil("f1b2c3d4-9002-9002-9002-000000000004")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	firstCloseTime := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)
	secondCloseTime := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551300002", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(&firstCloseTime)
	first, err := h.Close(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, firstAgentID)
	if err != nil {
		t.Fatalf("Close() (first) error = %v", err)
	}
	if first.AlreadyClosed {
		t.Errorf("expected first close to NOT be already_closed")
	}

	mockUtil.EXPECT().TimeNow().Return(&secondCloseTime)
	second, err := h.Close(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, secondAgentID)
	if err != nil {
		t.Fatalf("Close() (second, double-close) error = %v", err)
	}
	if !second.AlreadyClosed {
		t.Errorf("expected second close to report already_closed=true")
	}
	// Must reflect the FIRST agent's close, not the second caller's.
	if second.ClosedByID == nil || *second.ClosedByID != firstAgentID {
		t.Errorf("expected closed_by_id to remain the first agent %s (truthful persisted state), got: %v", firstAgentID, second.ClosedByID)
	}
	if second.ClosedReason != kase.ClosedReasonAgentClosed {
		t.Errorf("expected closed_reason to remain agent_closed, got: %s", second.ClosedReason)
	}
}

// Test_Close_NonExistentCase_Returns404Signal verifies the "id genuinely
// doesn't exist" branch is distinguished from "already closed."
func Test_Close_NonExistentCase_Returns404Signal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9003-9003-9003-000000000001")
	ghostCaseID := uuid.FromStringOrNil("f1b2c3d4-9003-9003-9003-000000000099")
	agentID := uuid.FromStringOrNil("f1b2c3d4-9003-9003-9003-000000000002")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().TimeNow().Return(&now)

	_, err := h.Close(ctx, customerID, ghostCaseID, commonidentity.OwnerTypeAgent, agentID)
	if err != dbhandler.ErrNotFound {
		t.Errorf("expected ErrNotFound for a genuinely non-existent case, got: %v", err)
	}
}
