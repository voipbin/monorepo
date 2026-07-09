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

// Test_CaseList_ScopesToCustomerAndAppliesFilters verifies the
// casehandler-level CaseList thin wrapper (design §9's GET /v1/cases
// list surface) delegates to dbhandler.CaseList with the given filters.
func Test_CaseList_ScopesToCustomerAndAppliesFilters(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9711-9711-9711-000000000001")
	otherCustomerID := uuid.FromStringOrNil("f1b2c3d4-9711-9711-9711-000000000009")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9711-9711-9711-000000000002")
	otherCaseID := uuid.FromStringOrNil("f1b2c3d4-9711-9711-9711-000000000003")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+155****2001", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	other := &kase.Case{
		ID: otherCaseID, CustomerID: otherCustomerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+155****2002", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}
	if err := db.CaseInsert(ctx, other); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	res, err := h.CaseList(ctx, customerID, "", commonidentity.OwnerTypeNone, uuid.Nil, uuid.Nil)
	if err != nil {
		t.Fatalf("CaseList() error = %v", err)
	}
	found := map[uuid.UUID]bool{}
	for _, item := range res {
		found[item.ID] = true
	}
	if !found[caseID] {
		t.Errorf("expected the customer's case to appear")
	}
	if found[otherCaseID] {
		t.Errorf("CaseList() must not leak another customer's case")
	}
}

// Test_CaseGet_ScopesToCustomer verifies the public, tenant-checked
// CaseGet wrapper (design §9's GET /v1/cases/{id}): a case belonging to
// a different customer must return dbhandler.ErrNotFound, never leak
// existence, mirroring verifyCaseOwnership's contract used elsewhere.
func Test_CaseGet_ScopesToCustomer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	victimCustomerID := uuid.FromStringOrNil("f1b2c3d4-9702-9702-9702-000000000001")
	attackerCustomerID := uuid.FromStringOrNil("f1b2c3d4-9702-9702-9702-000000000002")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9702-9702-9702-000000000003")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	c := &kase.Case{
		ID: caseID, CustomerID: victimCustomerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+155****2003", ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	got, err := h.CaseGet(ctx, victimCustomerID, caseID)
	if err != nil {
		t.Fatalf("CaseGet() error = %v", err)
	}
	if got.ID != caseID {
		t.Errorf("CaseGet() = %v, expected id %v", got.ID, caseID)
	}

	if _, err := h.CaseGet(ctx, attackerCustomerID, caseID); err != dbhandler.ErrNotFound {
		t.Errorf("CaseGet() cross-tenant = %v, expected dbhandler.ErrNotFound", err)
	}
}
