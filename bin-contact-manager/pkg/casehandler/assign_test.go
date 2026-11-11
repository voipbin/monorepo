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

	mockNotify.EXPECT().PublishEvent(ctx, kase.EventTypeCaseAssigned, &kase.CaseAssignedEvent{
		CaseID: caseID, CustomerID: customerID, OwnerID: ownerID,
	})

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

// Test_Assign_PublishesCaseAssigned verifies that an owner change of an open Case to an agent is
// published once with the case, the customer and the new owner, whoever assigned it.
func Test_Assign_PublishesCaseAssigned(t *testing.T) {
	agentA := uuid.FromStringOrNil("f1b2c3d4-1718-1718-1718-00000000a001")
	agentB := uuid.FromStringOrNil("f1b2c3d4-1718-1718-1718-00000000a002")
	ctx := context.Background()

	tests := []struct {
		name   string
		suffix string
		prev   uuid.UUID
		next   uuid.UUID
	}{
		{"unowned to an agent", "000000000001", uuid.Nil, agentA},
		{"one agent to another agent", "000000000002", agentA, agentB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newStrictSyncEnv(t, tt.suffix)
			c := e.insertCase(t, kase.ReferenceTypeCall, uuid.Must(uuid.NewV4()).String(), nil, tt.prev)

			e.mockNotify.EXPECT().PublishEvent(ctx, kase.EventTypeCaseAssigned, &kase.CaseAssignedEvent{
				CaseID:     c.ID,
				CustomerID: e.customer,
				OwnerID:    tt.next,
			}).Times(1)

			if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, tt.next); err != nil {
				t.Fatalf("Assign() error = %v", err)
			}
		})
	}
}

// Test_Assign_NoCaseAssignedEvent verifies the paths that must not notify the owner: the strict
// mock has no PublishEvent expectation, so any publish fails the test.
func Test_Assign_NoCaseAssignedEvent(t *testing.T) {
	agentA := uuid.FromStringOrNil("f1b2c3d4-1718-1718-1718-00000000b001")
	ctx := context.Background()

	t.Run("the same owner again (retry of the same request)", func(t *testing.T) {
		e := newStrictSyncEnv(t, "000000000011")
		c := e.insertCase(t, kase.ReferenceTypeCall, uuid.Must(uuid.NewV4()).String(), nil, agentA)

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("unassign", func(t *testing.T) {
		e := newStrictSyncEnv(t, "000000000012")
		c := e.insertCase(t, kase.ReferenceTypeCall, uuid.Must(uuid.NewV4()).String(), nil, agentA)

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeNone, uuid.Nil); err != nil {
			t.Fatalf("Unassign() error = %v", err)
		}
	})

	t.Run("agent owner type with a nil id", func(t *testing.T) {
		e := newStrictSyncEnv(t, "000000000013")
		c := e.insertCase(t, kase.ReferenceTypeCall, uuid.Must(uuid.NewV4()).String(), nil, uuid.Nil)

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, uuid.Nil); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("agent owner type with a nil id on a case that has an owner", func(t *testing.T) {
		e := newStrictSyncEnv(t, "000000000017")
		c := e.insertCase(t, kase.ReferenceTypeCall, uuid.Must(uuid.NewV4()).String(), nil, agentA)

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, uuid.Nil); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("an owner type other than agent with an id", func(t *testing.T) {
		e := newStrictSyncEnv(t, "000000000018")
		c := e.insertCase(t, kase.ReferenceTypeCall, uuid.Must(uuid.NewV4()).String(), nil, uuid.Nil)

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerType("ai"), agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("closed case", func(t *testing.T) {
		e := newStrictSyncEnv(t, "000000000014")
		c := e.insertCase(t, kase.ReferenceTypeCall, uuid.Must(uuid.NewV4()).String(), nil, uuid.Nil)
		now := time.Date(2026, 11, 12, 11, 0, 0, 0, time.UTC)
		by := agentA
		if _, err := e.db.CaseUpdateStatusClosed(ctx, e.customer, c.ID, kase.ClosedReasonAgentClosed, string(commonidentity.OwnerTypeAgent), &by, &now, ""); err != nil {
			t.Fatalf("close error = %v", err)
		}

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("case closed between the read and the write", func(t *testing.T) {
		mc := gomock.NewController(t)
		mockDB := dbhandler.NewMockDBHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &caseHandler{db: mockDB, notifyHandler: mockNotify}

		customer := uuid.FromStringOrNil("f1b2c3d4-1718-1718-1718-000000000015")
		caseID := uuid.Must(uuid.NewV4())
		open := &kase.Case{ID: caseID, CustomerID: customer, Status: kase.StatusOpen, ReferenceType: kase.ReferenceTypeCall}
		closed := &kase.Case{ID: caseID, CustomerID: customer, Status: kase.StatusClosed, ReferenceType: kase.ReferenceTypeCall, OwnerType: commonidentity.OwnerTypeAgent, OwnerID: agentA}

		gomock.InOrder(
			mockDB.EXPECT().CaseGetByID(ctx, caseID).Return(open, nil),
			mockDB.EXPECT().CaseUpdateOwner(ctx, customer, caseID, commonidentity.OwnerTypeAgent, agentA).Return(nil),
			mockDB.EXPECT().CaseGetByID(ctx, caseID).Return(closed, nil),
		)

		if _, err := h.Assign(ctx, customer, caseID, commonidentity.OwnerTypeAgent, agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("owner update fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		mockDB := dbhandler.NewMockDBHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &caseHandler{db: mockDB, notifyHandler: mockNotify}

		customer := uuid.FromStringOrNil("f1b2c3d4-1718-1718-1718-000000000016")
		caseID := uuid.Must(uuid.NewV4())
		open := &kase.Case{ID: caseID, CustomerID: customer, Status: kase.StatusOpen, ReferenceType: kase.ReferenceTypeCall}

		mockDB.EXPECT().CaseGetByID(ctx, caseID).Return(open, nil)
		mockDB.EXPECT().CaseUpdateOwner(ctx, customer, caseID, commonidentity.OwnerTypeAgent, agentA).Return(context.DeadlineExceeded)

		if _, err := h.Assign(ctx, customer, caseID, commonidentity.OwnerTypeAgent, agentA); err == nil {
			t.Fatalf("expected an error")
		}
	})
}

// newStrictSyncEnv is newSyncEnv without the tolerance for the case_assigned event: any publish
// fails the test.
func newStrictSyncEnv(t *testing.T, customerSuffix string) *syncEnv {
	t.Helper()
	return newSyncEnvWith(t, customerSuffix, false)
}
