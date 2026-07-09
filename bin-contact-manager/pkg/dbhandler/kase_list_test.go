package dbhandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/kase"
)

// Test_CaseList_FiltersByCustomerStatusAndOwner verifies the new
// customer-scoped CaseList primitive backing the Phase 5 RPC surface's
// GET /v1/cases?status=...&owner_type=...&owner_id=... query. status
// and owner filters are optional (empty status / uuid.Nil owner mean
// "no filter on this dimension"), but customer_id is always required
// and always applied.
func Test_CaseList_FiltersByCustomerStatusAndOwner(t *testing.T) {
	h := NewHandler(dbTest, nil)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000001")
	otherCustomerID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000009")
	ownerID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000002")
	otherOwnerID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000003")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	closed := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)

	openOwnedCaseID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000010")
	closedOwnedCaseID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000011")
	openOtherOwnerCaseID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000012")
	otherCustomerCaseID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000013")

	cases := []*kase.Case{
		{
			ID: openOwnedCaseID, CustomerID: customerID,
			PeerType: commonaddress.TypeTel, PeerTarget: "+155****1010", ReferenceType: "call",
			Owner:  commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: ownerID},
			Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
		},
		{
			ID: closedOwnedCaseID, CustomerID: customerID,
			PeerType: commonaddress.TypeTel, PeerTarget: "+155****1011", ReferenceType: "call",
			Owner:  commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: ownerID},
			Status: kase.StatusClosed, OpenedAt: &opened, ClosedAt: &closed,
			ClosedReason: kase.ClosedReasonAgentClosed, ClosedByType: kase.ClosedByTypeAgent,
			TMCreate: &opened, TMUpdate: &closed,
		},
		{
			ID: openOtherOwnerCaseID, CustomerID: customerID,
			PeerType: commonaddress.TypeTel, PeerTarget: "+155****1012", ReferenceType: "call",
			Owner:  commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: otherOwnerID},
			Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
		},
		{
			ID: otherCustomerCaseID, CustomerID: otherCustomerID,
			PeerType: commonaddress.TypeTel, PeerTarget: "+155****1013", ReferenceType: "call",
			Owner:  commonidentity.Owner{OwnerType: commonidentity.OwnerTypeAgent, OwnerID: ownerID},
			Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
		},
	}
	for _, c := range cases {
		if err := h.CaseInsert(ctx, c); err != nil {
			t.Fatalf("CaseInsert(%s) error = %v", c.ID, err)
		}
	}

	// No filters beyond customer_id: expect all 3 of this customer's cases.
	all, err := h.CaseList(ctx, customerID, 100, "", "", "", uuid.Nil, uuid.Nil)
	if err != nil {
		t.Fatalf("CaseList() error = %v", err)
	}
	foundAll := map[uuid.UUID]bool{}
	for _, c := range all {
		foundAll[c.ID] = true
	}
	if !foundAll[openOwnedCaseID] || !foundAll[closedOwnedCaseID] || !foundAll[openOtherOwnerCaseID] {
		t.Errorf("expected all 3 of this customer's cases, got: %v", foundAll)
	}
	if foundAll[otherCustomerCaseID] {
		t.Errorf("CaseList() must not leak another customer's case")
	}

	// status=open filter.
	openOnly, err := h.CaseList(ctx, customerID, 100, "", string(kase.StatusOpen), "", uuid.Nil, uuid.Nil)
	if err != nil {
		t.Fatalf("CaseList(status=open) error = %v", err)
	}
	foundOpen := map[uuid.UUID]bool{}
	for _, c := range openOnly {
		foundOpen[c.ID] = true
	}
	if foundOpen[closedOwnedCaseID] {
		t.Errorf("status=open filter must exclude closed cases")
	}
	if !foundOpen[openOwnedCaseID] || !foundOpen[openOtherOwnerCaseID] {
		t.Errorf("status=open filter must include both open cases")
	}

	// owner_type+owner_id filter.
	ownedOnly, err := h.CaseList(ctx, customerID, 100, "", "", commonidentity.OwnerTypeAgent, ownerID, uuid.Nil)
	if err != nil {
		t.Fatalf("CaseList(owner) error = %v", err)
	}
	foundOwned := map[uuid.UUID]bool{}
	for _, c := range ownedOnly {
		foundOwned[c.ID] = true
	}
	if !foundOwned[openOwnedCaseID] || !foundOwned[closedOwnedCaseID] {
		t.Errorf("owner filter must include both of ownerID's cases")
	}
	if foundOwned[openOtherOwnerCaseID] {
		t.Errorf("owner filter must exclude the other owner's case")
	}

	// contact_id filter.
	contactID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000004")
	otherContactID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000005")
	contactAttributedCaseID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000014")
	if err := h.CaseInsert(ctx, &kase.Case{
		ID: contactAttributedCaseID, CustomerID: customerID,
		PeerType: commonaddress.TypeTel, PeerTarget: "+155****1014", ReferenceType: "call",
		ContactID: &contactID,
		Status:    kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}); err != nil {
		t.Fatalf("CaseInsert(contact-attributed) error = %v", err)
	}
	contactOnly, err := h.CaseList(ctx, customerID, 100, "", "", "", uuid.Nil, contactID)
	if err != nil {
		t.Fatalf("CaseList(contact_id) error = %v", err)
	}
	if len(contactOnly) != 1 || contactOnly[0].ID != contactAttributedCaseID {
		t.Errorf("contact_id filter = %v, want exactly [%v]", contactOnly, contactAttributedCaseID)
	}
	contactNoMatch, err := h.CaseList(ctx, customerID, 100, "", "", "", uuid.Nil, otherContactID)
	if err != nil {
		t.Fatalf("CaseList(contact_id=other) error = %v", err)
	}
	if len(contactNoMatch) != 0 {
		t.Errorf("contact_id filter for an unmatched contact = %v, want empty", contactNoMatch)
	}
}

// Test_CaseList_OrderedAndPaginated verifies CaseList orders results by
// tm_create DESC and honors size/token pagination (the round-2 review
// finding: page_size was previously accepted but silently ignored, and
// results had no deterministic order).
func Test_CaseList_OrderedAndPaginated(t *testing.T) {
	h := NewHandler(dbTest, nil)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000021")

	t1 := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	id1 := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000031")
	id2 := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000032")
	id3 := uuid.FromStringOrNil("f1b2c3d4-9601-9601-9601-000000000033")

	for _, c := range []*kase.Case{
		{ID: id1, CustomerID: customerID, PeerType: commonaddress.TypeTel, PeerTarget: "+155****2001", ReferenceType: "call", Status: kase.StatusOpen, OpenedAt: &t1, TMCreate: &t1, TMUpdate: &t1},
		{ID: id2, CustomerID: customerID, PeerType: commonaddress.TypeTel, PeerTarget: "+155****2002", ReferenceType: "call", Status: kase.StatusOpen, OpenedAt: &t2, TMCreate: &t2, TMUpdate: &t2},
		{ID: id3, CustomerID: customerID, PeerType: commonaddress.TypeTel, PeerTarget: "+155****2003", ReferenceType: "call", Status: kase.StatusOpen, OpenedAt: &t3, TMCreate: &t3, TMUpdate: &t3},
	} {
		if err := h.CaseInsert(ctx, c); err != nil {
			t.Fatalf("CaseInsert(%s) error = %v", c.ID, err)
		}
	}

	// size=100, no token: expect all 3, newest (id3) first.
	all, err := h.CaseList(ctx, customerID, 100, "", "", "", uuid.Nil, uuid.Nil)
	if err != nil {
		t.Fatalf("CaseList() error = %v", err)
	}
	if len(all) != 3 || all[0].ID != id3 || all[1].ID != id2 || all[2].ID != id1 {
		t.Fatalf("CaseList() order = %v, want [id3, id2, id1] (tm_create DESC)", all)
	}

	// size=1: expect only the newest (id3).
	page1, err := h.CaseList(ctx, customerID, 1, "", "", "", uuid.Nil, uuid.Nil)
	if err != nil {
		t.Fatalf("CaseList(size=1) error = %v", err)
	}
	if len(page1) != 1 || page1[0].ID != id3 {
		t.Fatalf("CaseList(size=1) = %v, want exactly [id3]", page1)
	}

	// token=t3 (as a cursor): expect id2, id1 (strictly older than t3).
	token := t3.UTC().Format("2006-01-02T15:04:05.000000Z")
	page2, err := h.CaseList(ctx, customerID, 100, token, "", "", uuid.Nil, uuid.Nil)
	if err != nil {
		t.Fatalf("CaseList(token) error = %v", err)
	}
	if len(page2) != 2 || page2[0].ID != id2 || page2[1].ID != id1 {
		t.Fatalf("CaseList(token=%s) = %v, want [id2, id1]", token, page2)
	}
}
