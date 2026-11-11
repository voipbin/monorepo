package casehandler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

type syncEnv struct {
	h        *caseHandler
	db       dbhandler.DBHandler
	mockReq  *requesthandler.MockRequestHandler
	mockUtil *utilhandler.MockUtilHandler
	customer uuid.UUID
	seq      int
}

func newSyncEnv(t *testing.T, customerSuffix string) *syncEnv {
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)

	return &syncEnv{
		h:        &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify},
		db:       db,
		mockReq:  mockReq,
		mockUtil: mockUtil,
		customer: uuid.FromStringOrNil("f1b2c3d4-1715-1715-1715-" + customerSuffix),
	}
}

// insertCase inserts an open Case with a unique peer and returns it.
func (e *syncEnv) insertCase(t *testing.T, refType, refID string, prev *uuid.UUID, owner uuid.UUID) *kase.Case {
	e.seq++
	opened := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	c := &kase.Case{
		ID:         uuid.Must(uuid.NewV4()),
		CustomerID: e.customer,
		Peer:       commonaddress.Address{Type: commonaddress.TypeTel, Target: fmt.Sprintf("+1550000%04d", e.seq)},

		ReferenceType: refType, ReferenceID: refID, PreviousCaseID: prev,
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if owner != uuid.Nil {
		c.OwnerType = commonidentity.OwnerTypeAgent
		c.OwnerID = owner
	}
	if err := e.db.CaseInsert(context.Background(), c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}
	return c
}

func (e *syncEnv) conversation(id, owner uuid.UUID) *cvconversation.Conversation {
	return &cvconversation.Conversation{
		Identity:  commonidentity.Identity{ID: id, CustomerID: e.customer},
		OwnerType: commonidentity.OwnerTypeAgent,
		OwnerID:   owner,
	}
}

func ownerFields(id uuid.UUID) map[cvconversation.Field]any {
	return map[cvconversation.Field]any{cvconversation.FieldOwnerID: id}
}

func Test_Assign_SyncsConversationOwner(t *testing.T) {
	agentA := uuid.FromStringOrNil("f1b2c3d4-1715-1715-1715-00000000a001")
	agentB := uuid.FromStringOrNil("f1b2c3d4-1715-1715-1715-00000000a002")
	cvID := uuid.FromStringOrNil("f1b2c3d4-1715-1715-1715-00000000c001")
	ctx := context.Background()

	t.Run("assign sets the conversation owner", func(t *testing.T) {
		e := newSyncEnv(t, "000000000001")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, uuid.Nil)

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, uuid.Nil), nil)
		e.mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, cvID, ownerFields(agentA)).Return(&cvconversation.Conversation{}, nil)

		res, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA)
		if err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
		if res.OwnerID != agentA {
			t.Errorf("expected case owner %v, got %v", agentA, res.OwnerID)
		}
	})

	t.Run("reassign overwrites the owner with the new agent", func(t *testing.T) {
		e := newSyncEnv(t, "000000000002")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, agentA)

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, agentA), nil)
		e.mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, cvID, ownerFields(agentB)).Return(&cvconversation.Conversation{}, nil)

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentB); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("same owner is idempotent and makes no update", func(t *testing.T) {
		e := newSyncEnv(t, "000000000003")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, uuid.Nil)

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, agentA), nil)

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("call-origin case makes no conversation call", func(t *testing.T) {
		e := newSyncEnv(t, "000000000004")
		c := e.insertCase(t, kase.ReferenceTypeCall, uuid.Must(uuid.NewV4()).String(), nil, uuid.Nil)

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("conversation case without or with a bad reference is a no-op", func(t *testing.T) {
		e := newSyncEnv(t, "000000000005")
		c1 := e.insertCase(t, kase.ReferenceTypeConversationMessage, "", nil, uuid.Nil)
		c2 := e.insertCase(t, kase.ReferenceTypeConversationMessage, "not-a-uuid", nil, uuid.Nil)
		c3 := e.insertCase(t, kase.ReferenceTypeConversationMessage, uuid.Nil.String(), nil, uuid.Nil)

		for _, c := range []*kase.Case{c1, c2, c3} {
			if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA); err != nil {
				t.Fatalf("Assign() error = %v", err)
			}
		}
	})

	t.Run("continued case resolves the conversation through the chain", func(t *testing.T) {
		e := newSyncEnv(t, "000000000006")
		root := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, uuid.Nil)
		mid := e.insertCase(t, kase.ReferenceTypeConversationMessage, "", &root.ID, uuid.Nil)
		tip := e.insertCase(t, kase.ReferenceTypeConversationMessage, "", &mid.ID, uuid.Nil)

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, uuid.Nil), nil)
		e.mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, cvID, ownerFields(agentA)).Return(&cvconversation.Conversation{}, nil)

		if _, err := e.h.Assign(ctx, e.customer, tip.ID, commonidentity.OwnerTypeAgent, agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("broken chain is a no-op", func(t *testing.T) {
		e := newSyncEnv(t, "000000000007")
		missing := uuid.Must(uuid.NewV4())
		tip := e.insertCase(t, kase.ReferenceTypeConversationMessage, "", &missing, uuid.Nil)

		if _, err := e.h.Assign(ctx, e.customer, tip.ID, commonidentity.OwnerTypeAgent, agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("chain through another customer is a no-op", func(t *testing.T) {
		e := newSyncEnv(t, "000000000008")
		other := newSyncEnv(t, "000000000009")
		foreign := other.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, uuid.Nil)
		tip := e.insertCase(t, kase.ReferenceTypeConversationMessage, "", &foreign.ID, uuid.Nil)

		if _, err := e.h.Assign(ctx, e.customer, tip.ID, commonidentity.OwnerTypeAgent, agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("chain longer than the hop limit is a no-op", func(t *testing.T) {
		e := newSyncEnv(t, "00000000000a")
		prev := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, uuid.Nil)
		for i := 0; i < maxCaseChainHops+1; i++ {
			prev = e.insertCase(t, kase.ReferenceTypeConversationMessage, "", &prev.ID, uuid.Nil)
		}

		if _, err := e.h.Assign(ctx, e.customer, prev.ID, commonidentity.OwnerTypeAgent, agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("conversation not found is a no-op and the case is updated", func(t *testing.T) {
		e := newSyncEnv(t, "00000000000b")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, uuid.Nil)

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(nil, cerrors.NotFound("conversation-manager", "NOT_FOUND", "gone"))

		res, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA)
		if err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
		if res.OwnerID != agentA {
			t.Errorf("expected case owner %v, got %v", agentA, res.OwnerID)
		}
	})

	t.Run("conversation of another customer is never touched", func(t *testing.T) {
		e := newSyncEnv(t, "00000000000c")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, uuid.Nil)

		foreign := e.conversation(cvID, uuid.Nil)
		foreign.CustomerID = uuid.Must(uuid.NewV4())
		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(foreign, nil)

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("get failure returns an error and leaves the case untouched", func(t *testing.T) {
		e := newSyncEnv(t, "00000000000d")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, uuid.Nil)

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(nil, errors.New("boom"))

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA); err == nil {
			t.Fatalf("expected an error")
		}
		got, _ := e.db.CaseGetByID(ctx, c.ID)
		if got.OwnerID != uuid.Nil {
			t.Errorf("case owner must stay unset, got %v", got.OwnerID)
		}
	})

	t.Run("update failure returns an error, the retry converges", func(t *testing.T) {
		e := newSyncEnv(t, "00000000000e")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, uuid.Nil)

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, uuid.Nil), nil)
		e.mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, cvID, ownerFields(agentA)).Return(nil, errors.New("boom"))
		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA); err == nil {
			t.Fatalf("expected an error")
		}
		got, _ := e.db.CaseGetByID(ctx, c.ID)
		if got.OwnerID != uuid.Nil {
			t.Errorf("case owner must stay unset, got %v", got.OwnerID)
		}

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, uuid.Nil), nil)
		e.mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, cvID, ownerFields(agentA)).Return(&cvconversation.Conversation{}, nil)
		res, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA)
		if err != nil {
			t.Fatalf("retry Assign() error = %v", err)
		}
		if res.OwnerID != agentA {
			t.Errorf("expected case owner %v, got %v", agentA, res.OwnerID)
		}
	})

	t.Run("database error while walking the chain is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		mockDB := dbhandler.NewMockDBHandler(mc)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &caseHandler{reqHandler: mockReq, db: mockDB}

		customer := uuid.FromStringOrNil("f1b2c3d4-1715-1715-1715-000000000031")
		caseID := uuid.Must(uuid.NewV4())
		prevID := uuid.Must(uuid.NewV4())
		tip := &kase.Case{ID: caseID, CustomerID: customer, Status: kase.StatusOpen, ReferenceType: kase.ReferenceTypeConversationMessage, PreviousCaseID: &prevID}

		mockDB.EXPECT().CaseGetByID(ctx, caseID).Return(tip, nil)
		mockDB.EXPECT().CaseGetByID(ctx, prevID).Return(nil, errors.New("boom"))

		if _, err := h.Assign(ctx, customer, caseID, commonidentity.OwnerTypeAgent, agentA); err == nil {
			t.Fatalf("expected an error")
		}
	})

	t.Run("already closed case makes no conversation call", func(t *testing.T) {
		e := newSyncEnv(t, "000000000033")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, agentB)
		now := time.Date(2026, 11, 11, 11, 0, 0, 0, time.UTC)
		by := agentB
		if _, err := e.db.CaseUpdateStatusClosed(ctx, e.customer, c.ID, kase.ClosedReasonAgentClosed, string(commonidentity.OwnerTypeAgent), &by, &now, ""); err != nil {
			t.Fatalf("close error = %v", err)
		}

		// strict mock: neither Assign nor Unassign may touch the conversation.
		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeNone, uuid.Nil); err != nil {
			t.Fatalf("Unassign() error = %v", err)
		}
	})

	t.Run("case closed between the read and the write releases the conversation again", func(t *testing.T) {
		mc := gomock.NewController(t)
		mockDB := dbhandler.NewMockDBHandler(mc)
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &caseHandler{reqHandler: mockReq, db: mockDB}

		customer := uuid.FromStringOrNil("f1b2c3d4-1715-1715-1715-000000000032")
		caseID := uuid.Must(uuid.NewV4())
		open := &kase.Case{ID: caseID, CustomerID: customer, Status: kase.StatusOpen, ReferenceType: kase.ReferenceTypeConversationMessage, ReferenceID: cvID.String()}
		closed := &kase.Case{ID: caseID, CustomerID: customer, Status: kase.StatusClosed, ReferenceType: kase.ReferenceTypeConversationMessage, ReferenceID: cvID.String(), OwnerType: commonidentity.OwnerTypeAgent, OwnerID: agentA}
		cv := &cvconversation.Conversation{Identity: commonidentity.Identity{ID: cvID, CustomerID: customer}}
		cvOwned := &cvconversation.Conversation{Identity: commonidentity.Identity{ID: cvID, CustomerID: customer}, OwnerType: commonidentity.OwnerTypeAgent, OwnerID: agentA}

		gomock.InOrder(
			mockDB.EXPECT().CaseGetByID(ctx, caseID).Return(open, nil),
			mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(cv, nil),
			mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, cvID, ownerFields(agentA)).Return(&cvconversation.Conversation{}, nil),
			mockDB.EXPECT().CaseUpdateOwner(ctx, customer, caseID, commonidentity.OwnerTypeAgent, agentA).Return(nil),
			mockDB.EXPECT().CaseGetByID(ctx, caseID).Return(closed, nil),
			mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(cvOwned, nil),
			mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, cvID, ownerFields(uuid.Nil)).Return(&cvconversation.Conversation{}, nil),
		)

		if _, err := h.Assign(ctx, customer, caseID, commonidentity.OwnerTypeAgent, agentA); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})

	t.Run("unsupported owner combinations make no conversation call", func(t *testing.T) {
		e := newSyncEnv(t, "00000000000f")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, agentA)

		// agent type with a nil id is not an assign and not an unassign.
		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, uuid.Nil); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	})
}

func Test_Unassign_ReleasesConversationOwner(t *testing.T) {
	agentA := uuid.FromStringOrNil("f1b2c3d4-1715-1715-1715-00000000a001")
	agentB := uuid.FromStringOrNil("f1b2c3d4-1715-1715-1715-00000000a002")
	cvID := uuid.FromStringOrNil("f1b2c3d4-1715-1715-1715-00000000c001")
	ctx := context.Background()

	t.Run("releases when the conversation owner equals the previous case owner", func(t *testing.T) {
		e := newSyncEnv(t, "000000000011")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, agentA)

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, agentA), nil)
		e.mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, cvID, ownerFields(uuid.Nil)).Return(&cvconversation.Conversation{}, nil)

		res, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeNone, uuid.Nil)
		if err != nil {
			t.Fatalf("Unassign() error = %v", err)
		}
		if res.OwnerID != uuid.Nil {
			t.Errorf("expected case owner cleared, got %v", res.OwnerID)
		}
	})

	t.Run("keeps a conversation owned by another agent", func(t *testing.T) {
		e := newSyncEnv(t, "000000000012")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, agentA)

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, agentB), nil)

		res, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeNone, uuid.Nil)
		if err != nil {
			t.Fatalf("Unassign() error = %v", err)
		}
		if res.OwnerID != uuid.Nil {
			t.Errorf("expected case owner cleared, got %v", res.OwnerID)
		}
	})

	t.Run("already unowned conversation makes no update", func(t *testing.T) {
		e := newSyncEnv(t, "000000000013")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, agentA)

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, uuid.Nil), nil)

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeNone, uuid.Nil); err != nil {
			t.Fatalf("Unassign() error = %v", err)
		}
	})

	t.Run("case without an owner makes no conversation call", func(t *testing.T) {
		e := newSyncEnv(t, "000000000014")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, uuid.Nil)

		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeNone, uuid.Nil); err != nil {
			t.Fatalf("Unassign() error = %v", err)
		}
	})

	t.Run("update failure leaves the case assigned, the retry converges", func(t *testing.T) {
		e := newSyncEnv(t, "000000000015")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, agentA)

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, agentA), nil)
		e.mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, cvID, ownerFields(uuid.Nil)).Return(nil, errors.New("boom"))
		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeNone, uuid.Nil); err == nil {
			t.Fatalf("expected an error")
		}
		got, _ := e.db.CaseGetByID(ctx, c.ID)
		if got.OwnerID != agentA {
			t.Errorf("case owner must stay %v, got %v", agentA, got.OwnerID)
		}

		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, agentA), nil)
		e.mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, cvID, ownerFields(uuid.Nil)).Return(&cvconversation.Conversation{}, nil)
		if _, err := e.h.Assign(ctx, e.customer, c.ID, commonidentity.OwnerTypeNone, uuid.Nil); err != nil {
			t.Fatalf("retry Unassign() error = %v", err)
		}
	})
}

func Test_Close_ReleasesConversationOwner(t *testing.T) {
	agentA := uuid.FromStringOrNil("f1b2c3d4-1715-1715-1715-00000000a001")
	agentB := uuid.FromStringOrNil("f1b2c3d4-1715-1715-1715-00000000a002")
	cvID := uuid.FromStringOrNil("f1b2c3d4-1715-1715-1715-00000000c001")
	ctx := context.Background()
	now := time.Date(2026, 11, 11, 11, 0, 0, 0, time.UTC)

	t.Run("releases the conversation owner and keeps the case owner", func(t *testing.T) {
		e := newSyncEnv(t, "000000000021")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, agentA)

		e.mockUtil.EXPECT().TimeNow().Return(&now)
		// unconditional: even an owner other than the case owner is released.
		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, agentB), nil)
		e.mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, cvID, ownerFields(uuid.Nil)).Return(&cvconversation.Conversation{}, nil)

		res, err := e.h.Close(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA, "")
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if res.Case.OwnerID != agentA || res.Case.OwnerType != commonidentity.OwnerTypeAgent {
			t.Errorf("case owner must be kept, got %v/%v", res.Case.OwnerType, res.Case.OwnerID)
		}
	})

	t.Run("call-origin case makes no conversation call", func(t *testing.T) {
		e := newSyncEnv(t, "000000000022")
		c := e.insertCase(t, kase.ReferenceTypeCall, uuid.Must(uuid.NewV4()).String(), nil, agentA)

		e.mockUtil.EXPECT().TimeNow().Return(&now)
		if _, err := e.h.Close(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA, ""); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	t.Run("sync failure is not returned, the case stays closed", func(t *testing.T) {
		e := newSyncEnv(t, "000000000023")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, agentA)

		e.mockUtil.EXPECT().TimeNow().Return(&now)
		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(nil, errors.New("boom"))

		res, err := e.h.Close(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA, "")
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		if res.AlreadyClosed || res.Case.Status != kase.StatusClosed {
			t.Errorf("expected a genuine close, got already_closed=%v status=%v", res.AlreadyClosed, res.Case.Status)
		}
	})

	t.Run("already closed case makes no conversation call", func(t *testing.T) {
		e := newSyncEnv(t, "000000000024")
		c := e.insertCase(t, kase.ReferenceTypeConversationMessage, cvID.String(), nil, agentA)

		e.mockUtil.EXPECT().TimeNow().Return(&now).Times(2)
		e.mockReq.EXPECT().ConversationV1ConversationGet(ctx, cvID).Return(e.conversation(cvID, agentA), nil).Times(1)
		e.mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, cvID, ownerFields(uuid.Nil)).Return(&cvconversation.Conversation{}, nil).Times(1)

		if _, err := e.h.Close(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA, ""); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
		res, err := e.h.Close(ctx, e.customer, c.ID, commonidentity.OwnerTypeAgent, agentA, "")
		if err != nil {
			t.Fatalf("second Close() error = %v", err)
		}
		if !res.AlreadyClosed {
			t.Errorf("expected already_closed")
		}
	})
}
