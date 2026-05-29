package aipromptproposalhandler

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestAccept_AlreadyAccepted_IdempotentSuccess(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	cust := uuid.Must(uuid.NewV4())
	pid := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), pid).Return(&aipromptproposal.AIPromptProposal{
		Identity: commonidentity.Identity{ID: pid, CustomerID: cust},
		Status:   aipromptproposal.StatusAccepted,
	}, nil)

	res, err := h.Accept(context.Background(), cust, pid)
	if err != nil {
		t.Fatalf("expected idempotent success, got err: %v", err)
	}
	if res.Status != aipromptproposal.StatusAccepted {
		t.Errorf("status mismatch: %s", res.Status)
	}
}

func TestAccept_NotCompleted_Returns409(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	cust := uuid.Must(uuid.NewV4())
	pid := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), pid).Return(&aipromptproposal.AIPromptProposal{
		Identity: commonidentity.Identity{ID: pid, CustomerID: cust},
		Status:   aipromptproposal.StatusFailed,
	}, nil)

	_, err := h.Accept(context.Background(), cust, pid)
	if err == nil || !strings.Contains(err.Error(), "proposal not completed") {
		t.Fatalf("expected proposal not completed, got: %v", err)
	}
}

func TestAccept_AuditDeleted_MarksExpired(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	cust := uuid.Must(uuid.NewV4())
	pid := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())
	tm := timePtr()

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), pid).Return(&aipromptproposal.AIPromptProposal{
		Identity: commonidentity.Identity{ID: pid, CustomerID: cust},
		Status:   aipromptproposal.StatusCompleted,
		AuditIDs: []uuid.UUID{auditID},
	}, nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), auditID).Return(&aiaudit.AIAudit{
		TMDelete: tm,
	}, nil)
	mdb.EXPECT().AIPromptProposalUpdateExpired(gomock.Any(), pid, string(aipromptproposal.ErrorInvalidAuditSet)).Return(int64(1), nil)

	_, err := h.Accept(context.Background(), cust, pid)
	if err == nil || !strings.Contains(err.Error(), "audit set invalidated") {
		t.Fatalf("expected audit set invalidated, got: %v", err)
	}
}

func TestAccept_Drifted_MarksExpired(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	injectRealUtilHandler(h)

	cust := uuid.Must(uuid.NewV4())
	pid := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), pid).Return(&aipromptproposal.AIPromptProposal{
		Identity:       commonidentity.Identity{ID: pid, CustomerID: cust},
		Status:         aipromptproposal.StatusCompleted,
		AuditIDs:       []uuid.UUID{auditID},
		ProposedPrompt: "new",
	}, nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), auditID).Return(&aiaudit.AIAudit{
		Status: aiaudit.StatusCompleted,
	}, nil)
	mdb.EXPECT().AIAcceptProposal(gomock.Any(), pid, gomock.Any(), "new").Return(dbhandler.ErrPromptVersionDrifted)
	mdb.EXPECT().AIPromptProposalUpdateExpired(gomock.Any(), pid, string(aipromptproposal.ErrorPromptVersionDrifted)).Return(int64(1), nil)

	_, err := h.Accept(context.Background(), cust, pid)
	if err == nil || !strings.Contains(err.Error(), "prompt version drifted") {
		t.Fatalf("expected prompt version drifted, got: %v", err)
	}
}

func TestAccept_ConcurrentAcceptSameProposal_LoserGetsIdempotentSuccess(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	injectRealUtilHandler(h)

	cust := uuid.Must(uuid.NewV4())
	pid := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())
	appliedHistoryID := uuid.Must(uuid.NewV4())

	// Pre-load returns Completed (the loser saw completed before the winner finished).
	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), pid).Return(&aipromptproposal.AIPromptProposal{
		Identity:       commonidentity.Identity{ID: pid, CustomerID: cust},
		Status:         aipromptproposal.StatusCompleted,
		AuditIDs:       []uuid.UUID{auditID},
		ProposedPrompt: "new",
	}, nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), auditID).Return(&aiaudit.AIAudit{
		Status: aiaudit.StatusCompleted,
	}, nil)
	// Inside the tx, proposal is already 'accepted' — AIAcceptProposal returns ErrProposalAlreadyAccepted.
	mdb.EXPECT().AIAcceptProposal(gomock.Any(), pid, gomock.Any(), "new").Return(dbhandler.ErrProposalAlreadyAccepted)
	// Handler reloads the now-accepted proposal.
	mdb.EXPECT().AIPromptProposalGet(gomock.Any(), pid).Return(&aipromptproposal.AIPromptProposal{
		Identity:               commonidentity.Identity{ID: pid, CustomerID: cust},
		Status:                 aipromptproposal.StatusAccepted,
		AppliedPromptHistoryID: appliedHistoryID,
	}, nil)

	res, err := h.Accept(context.Background(), cust, pid)
	if err != nil {
		t.Fatalf("expected idempotent success on concurrent accept, got: %v", err)
	}
	if res.Status != aipromptproposal.StatusAccepted {
		t.Errorf("status mismatch: %s", res.Status)
	}
	if res.AppliedPromptHistoryID != appliedHistoryID {
		t.Errorf("AppliedPromptHistoryID mismatch")
	}
}
