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

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_ResolutionCreateCaseLevel_DerivesContactID verifies design §3.4's
// write-path call site 1: creating a case-level positive Resolution
// writes Case.contact_id directly from deriveCaseContactID's result,
// inside the same transaction.
func Test_ResolutionCreateCaseLevel_DerivesContactID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9301-9301-9301-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9301-9301-9301-000000000002")
	contactID := uuid.FromStringOrNil("f1b2c3d4-9301-9301-9301-000000000003")
	agentID := uuid.FromStringOrNil("f1b2c3d4-9301-9301-9301-000000000004")
	resolutionID := uuid.FromStringOrNil("f1b2c3d4-9301-9301-9301-000000000005")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551500001", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockUtil.EXPECT().UUIDCreate().Return(resolutionID)
	mockUtil.EXPECT().TimeNow().Return(&now)

	res, err := h.ResolutionCreateCaseLevel(ctx, customerID, caseID, contactID, resolution.ResolutionTypePositive, resolution.ResolvedByTypeAgent, agentID)
	if err != nil {
		t.Fatalf("ResolutionCreateCaseLevel() error = %v", err)
	}
	if res.CaseID == nil || *res.CaseID != caseID {
		t.Errorf("expected case_id: %s, got: %v", caseID, res.CaseID)
	}

	reread, err := db.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if reread.ContactID == nil || *reread.ContactID != contactID {
		t.Errorf("expected Case.contact_id derived to %s, got: %v", contactID, reread.ContactID)
	}
}

// Test_ResolutionDeleteCaseLevel_RederivesContactID verifies write-path
// call site 1's soft-delete direction: soft-deleting the active
// case-level positive Resolution re-derives Case.contact_id back to nil
// (no other active positive resolution exists).
func Test_ResolutionDeleteCaseLevel_RederivesContactID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9302-9302-9302-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9302-9302-9302-000000000002")
	contactID := uuid.FromStringOrNil("f1b2c3d4-9302-9302-9302-000000000003")
	agentID := uuid.FromStringOrNil("f1b2c3d4-9302-9302-9302-000000000004")
	resolutionID := uuid.FromStringOrNil("f1b2c3d4-9302-9302-9302-000000000005")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	createTime := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+15551500002", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockUtil.EXPECT().UUIDCreate().Return(resolutionID)
	mockUtil.EXPECT().TimeNow().Return(&createTime)
	if _, err := h.ResolutionCreateCaseLevel(ctx, customerID, caseID, contactID, resolution.ResolutionTypePositive, resolution.ResolvedByTypeAgent, agentID); err != nil {
		t.Fatalf("ResolutionCreateCaseLevel() error = %v", err)
	}

	if err := h.ResolutionDeleteCaseLevel(ctx, customerID, caseID, resolutionID); err != nil {
		t.Fatalf("ResolutionDeleteCaseLevel() error = %v", err)
	}

	reread, err := db.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if reread.ContactID != nil {
		t.Errorf("expected Case.contact_id re-derived to nil after delete, got: %v", *reread.ContactID)
	}
}
