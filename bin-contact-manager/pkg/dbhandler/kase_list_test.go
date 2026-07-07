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
	all, err := h.CaseList(ctx, customerID, "", "", uuid.Nil)
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
	openOnly, err := h.CaseList(ctx, customerID, string(kase.StatusOpen), "", uuid.Nil)
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
	ownedOnly, err := h.CaseList(ctx, customerID, "", commonidentity.OwnerTypeAgent, ownerID)
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
}
