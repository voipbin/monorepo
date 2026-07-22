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
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_CaseListUnresolved verifies design §6's agent-facing unresolved
// queue: WHERE customer_id=? AND status='open' AND contact_id IS NULL.
// An open case with contact_id set must NOT appear; a closed case with
// contact_id NULL must NOT appear either (closing removes it from the
// queue regardless of resolution state, per §6).
func Test_CaseListUnresolved(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9401-9401-9401-000000000001")
	unresolvedCaseID := uuid.FromStringOrNil("f1b2c3d4-9401-9401-9401-000000000002")
	resolvedCaseID := uuid.FromStringOrNil("f1b2c3d4-9401-9401-9401-000000000003")
	closedUnresolvedCaseID := uuid.FromStringOrNil("f1b2c3d4-9401-9401-9401-000000000004")
	contactID := uuid.FromStringOrNil("f1b2c3d4-9401-9401-9401-000000000005")
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	closed := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC)

	cases := []*kase.Case{
		{
			ID: unresolvedCaseID, CustomerID: customerID,
			Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551600001"}, ReferenceType: "call",
			Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
		},
		{
			ID: resolvedCaseID, CustomerID: customerID,
			Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551600002"}, ReferenceType: "call",
			ContactID: &contactID,
			Status:    kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
		},
		{
			ID: closedUnresolvedCaseID, CustomerID: customerID,
			Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551600003"}, ReferenceType: "call",
			Status: kase.StatusClosed, OpenedAt: &opened, ClosedAt: &closed,
			ClosedReason: kase.ClosedReasonAgentClosed, ClosedByType: kase.ClosedByTypeAgent,
			TMCreate: &opened, TMUpdate: &closed,
		},
	}
	for _, c := range cases {
		if err := db.CaseInsert(ctx, c); err != nil {
			t.Fatalf("CaseInsert(%s) error = %v", c.ID, err)
		}
	}

	res, err := h.CaseListUnresolved(ctx, customerID)
	if err != nil {
		t.Fatalf("CaseListUnresolved() error = %v", err)
	}

	found := map[uuid.UUID]bool{}
	for _, c := range res {
		found[c.ID] = true
	}
	if !found[unresolvedCaseID] {
		t.Errorf("expected the open, unresolved case to appear in CaseListUnresolved()")
	}
	if found[resolvedCaseID] {
		t.Errorf("expected the resolved (contact_id set) case to NOT appear in CaseListUnresolved()")
	}
	if found[closedUnresolvedCaseID] {
		t.Errorf("expected the closed, unresolved case to NOT appear in CaseListUnresolved() (closing removes it from the queue per §6)")
	}
}
