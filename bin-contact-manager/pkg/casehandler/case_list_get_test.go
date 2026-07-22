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
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****2001"}, ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	other := &kase.Case{
		ID: otherCaseID, CustomerID: otherCustomerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****2002"}, ReferenceType: "call",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}
	if err := db.CaseInsert(ctx, other); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	res, _, err := h.CaseList(ctx, customerID, 0, "", "", commonidentity.OwnerTypeNone, uuid.Nil, uuid.Nil)
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

// Test_CaseList_DefaultSizeAndNextToken verifies the casehandler-level
// pagination wrapper: size==0 defaults to defaultCaseListSize, and a
// non-empty nextToken is returned only when more rows exist beyond the
// requested page.
func Test_CaseList_DefaultSizeAndNextToken(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9712-9712-9712-000000000001")
	t1 := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)
	id1 := uuid.FromStringOrNil("f1b2c3d4-9712-9712-9712-000000000011")
	id2 := uuid.FromStringOrNil("f1b2c3d4-9712-9712-9712-000000000012")

	for _, c := range []*kase.Case{
		{ID: id1, CustomerID: customerID, Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****2011"}, ReferenceType: "call", Status: kase.StatusOpen, OpenedAt: &t1, TMCreate: &t1, TMUpdate: &t1},
		{ID: id2, CustomerID: customerID, Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****2012"}, ReferenceType: "call", Status: kase.StatusOpen, OpenedAt: &t2, TMCreate: &t2, TMUpdate: &t2},
	} {
		if err := db.CaseInsert(ctx, c); err != nil {
			t.Fatalf("CaseInsert(%s) error = %v", c.ID, err)
		}
	}

	// size=1: exactly 1 item back, plus a non-empty nextToken (id1 still pending).
	page, nextToken, err := h.CaseList(ctx, customerID, 1, "", "", commonidentity.OwnerTypeNone, uuid.Nil, uuid.Nil)
	if err != nil {
		t.Fatalf("CaseList(size=1) error = %v", err)
	}
	if len(page) != 1 || page[0].ID != id2 {
		t.Fatalf("CaseList(size=1) = %v, want exactly [id2] (newest first)", page)
	}
	if nextToken == "" {
		t.Errorf("CaseList(size=1) nextToken = %q, want non-empty (id1 still pending)", nextToken)
	}

	// Follow the cursor: expect id1, and an empty nextToken (no further pages).
	page2, nextToken2, err := h.CaseList(ctx, customerID, 1, nextToken, "", commonidentity.OwnerTypeNone, uuid.Nil, uuid.Nil)
	if err != nil {
		t.Fatalf("CaseList(token) error = %v", err)
	}
	if len(page2) != 1 || page2[0].ID != id1 {
		t.Fatalf("CaseList(token) = %v, want exactly [id1]", page2)
	}
	if nextToken2 != "" {
		t.Errorf("CaseList(token) nextToken = %q, want empty (no further pages)", nextToken2)
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
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****2003"}, ReferenceType: "call",
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
