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

// Test_Assign_Success verifies the square-talk Cases menu design §3.2's
// Assign happy path: tenant-checked via CaseGetByID, then
// CaseUpdateOwner, then re-fetched and returned. Asserts the
// owner_type == "agent" on the re-fetched Case, not merely that
// owner_id changed.
func Test_Assign_Success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9201-9201-9201-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9201-9201-9201-000000000002")
	ownerID := uuid.FromStringOrNil("f1b2c3d4-9201-9201-9201-000000000003")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"}, ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	res, err := h.Assign(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, ownerID)
	if err != nil {
		t.Fatalf("Assign() error = %v", err)
	}
	if res.OwnerType != commonidentity.OwnerTypeAgent {
		t.Errorf("expected owner_type: %v, got: %v", commonidentity.OwnerTypeAgent, res.OwnerType)
	}
	if res.OwnerID != ownerID {
		t.Errorf("expected owner_id: %v, got: %v", ownerID, res.OwnerID)
	}
}

// Test_Assign_CrossTenant verifies Assign rejects a case belonging to a
// different customer, mirroring Continue's tenant-check pattern:
// dbhandler.ErrNotFound, never leaking existence.
func Test_Assign_CrossTenant(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	victimCustomerID := uuid.FromStringOrNil("f1b2c3d4-9202-9202-9202-000000000001")
	attackerCustomerID := uuid.FromStringOrNil("f1b2c3d4-9202-9202-9202-000000000002")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9202-9202-9202-000000000003")
	ownerID := uuid.FromStringOrNil("f1b2c3d4-9202-9202-9202-000000000004")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: victimCustomerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0002"}, ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	_, err := h.Assign(ctx, attackerCustomerID, caseID, commonidentity.OwnerTypeAgent, ownerID)
	if err != dbhandler.ErrNotFound {
		t.Errorf("expected dbhandler.ErrNotFound for cross-tenant Assign, got: %v", err)
	}

	reread, err := db.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if reread.OwnerID == ownerID {
		t.Errorf("expected owner_id unchanged after rejected cross-tenant Assign, got: %v", reread.OwnerID)
	}
}
