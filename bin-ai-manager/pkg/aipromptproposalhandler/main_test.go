package aipromptproposalhandler

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/geminiproposalhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func newHandlerWithMocks(t *testing.T) (*aipromptproposalHandler, *dbhandler.MockDBHandler, *geminiproposalhandler.MockGeminiProposalHandler, *gomock.Controller) {
	t.Helper()
	mc := gomock.NewController(t)
	mdb := dbhandler.NewMockDBHandler(mc)
	mg := geminiproposalhandler.NewMockGeminiProposalHandler(mc)
	h := &aipromptproposalHandler{
		db:            mdb,
		geminiHandler: mg,
		semaphore:     make(chan struct{}, maxConcurrentGlobal),
	}
	return h, mdb, mg, mc
}

func injectRealUtilHandler(h *aipromptproposalHandler) {
	h.utilHandler = utilhandler.NewUtilHandler()
}

func timePtr() *time.Time {
	now := time.Now()
	return &now
}

func TestCreate_EmptyAudits_Returns400(t *testing.T) {
	h, _, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	_, err := h.Create(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), nil, "en-US")
	if err == nil || !strings.Contains(err.Error(), "invalid audit set") {
		t.Fatalf("expected invalid audit set, got: %v", err)
	}
}

func TestCreate_TooManyAudits_Returns400(t *testing.T) {
	h, _, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	ids := make([]uuid.UUID, maxAuditsPerProposal+1)
	for i := range ids {
		ids[i] = uuid.Must(uuid.NewV4())
	}
	_, err := h.Create(context.Background(), uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()), ids, "en-US")
	if err == nil || !strings.Contains(err.Error(), "too many audits") {
		t.Fatalf("expected too many audits, got: %v", err)
	}
}

func TestCreate_AIDifferentCustomer_Returns404(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	cust := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIGet(gomock.Any(), aiID).Return(&ai.AI{
		Identity: commonidentity.Identity{CustomerID: uuid.Must(uuid.NewV4())},
	}, nil)

	_, err := h.Create(context.Background(), cust, aiID, []uuid.UUID{auditID}, "en-US")
	if err == nil || !strings.Contains(err.Error(), "ai not found") {
		t.Fatalf("expected ai not found, got: %v", err)
	}
}

func TestCreate_RateLimitExceeded_Returns429(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	cust := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIGet(gomock.Any(), aiID).Return(&ai.AI{
		Identity:               commonidentity.Identity{CustomerID: cust},
		CurrentPromptHistoryID: uuid.Must(uuid.NewV4()),
	}, nil)
	mdb.EXPECT().AIPromptProposalCountProgressing(gomock.Any(), cust).Return(int64(maxConcurrentCustomer), nil)

	_, err := h.Create(context.Background(), cust, aiID, []uuid.UUID{auditID}, "en-US")
	if err == nil || !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Fatalf("expected rate limit exceeded, got: %v", err)
	}
}

func TestCreate_AuditPromptVersionMismatch_Returns400(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	cust := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())
	currentHist := uuid.Must(uuid.NewV4())
	oldHist := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIGet(gomock.Any(), aiID).Return(&ai.AI{
		Identity:               commonidentity.Identity{CustomerID: cust},
		CurrentPromptHistoryID: currentHist,
	}, nil)
	mdb.EXPECT().AIPromptProposalCountProgressing(gomock.Any(), cust).Return(int64(0), nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), auditID).Return(&aiaudit.AIAudit{
		Identity:        commonidentity.Identity{CustomerID: cust},
		AIID:            aiID,
		Status:          aiaudit.StatusCompleted,
		PromptHistoryID: oldHist,
	}, nil)

	_, err := h.Create(context.Background(), cust, aiID, []uuid.UUID{auditID}, "en-US")
	if err == nil || !strings.Contains(err.Error(), "audit prompt version mismatch") {
		t.Fatalf("expected audit prompt version mismatch, got: %v", err)
	}
	if !strings.Contains(err.Error(), auditID.String()) {
		t.Fatalf("expected error to list offending audit ID; got: %v", err)
	}
}

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }

func errorsNew(s string) error { return &simpleErr{s} }

func TestRunProposalJob_Success_WritesCompleted(t *testing.T) {
	h, mdb, mg, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	injectRealUtilHandler(h)

	pid := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIAuditGet(gomock.Any(), auditID).Return(&aiaudit.AIAudit{
		Status:     aiaudit.StatusCompleted,
		AIcallID:   uuid.Must(uuid.NewV4()),
		Evaluation: []byte(`{"summary":"good","dimensions":{"helpfulness":{"reason":"h"},"accuracy":{"reason":"a"},"tone":{"reason":"t"},"goal_completion":{"reason":"g"}}}`),
	}, nil)
	mdb.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	mg.EXPECT().Evaluate(gomock.Any(), "orig", gomock.Any(), "en-US").
		Return(&geminiproposalhandler.ProposalResponse{ProposedPrompt: "new prompt", Rationale: "rationale text"}, nil)

	mdb.EXPECT().AIPromptProposalUpdateFinal(gomock.Any(), pid, aipromptproposal.StatusCompleted, "new prompt", "rationale text", "").
		Return(int64(1), nil)

	h.runProposalJob(context.Background(), pid, "orig", []uuid.UUID{auditID}, "en-US")
}

func TestRunProposalJob_GeminiError_WritesFailedEvaluatorUnavailable(t *testing.T) {
	h, mdb, mg, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	injectRealUtilHandler(h)

	pid := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIAuditGet(gomock.Any(), auditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil)
	mdb.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
	mg.EXPECT().Evaluate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errorsNew("network down"))
	mdb.EXPECT().AIPromptProposalUpdateFinal(gomock.Any(), pid, aipromptproposal.StatusFailed, "", "", string(aipromptproposal.ErrorEvaluatorUnavailable)).
		Return(int64(1), nil)

	h.runProposalJob(context.Background(), pid, "orig", []uuid.UUID{auditID}, "en-US")
}

func TestRunProposalJob_GeminiBadJSON_WritesFailedInvalidResponse(t *testing.T) {
	h, mdb, mg, mc := newHandlerWithMocks(t)
	defer mc.Finish()
	injectRealUtilHandler(h)

	pid := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIAuditGet(gomock.Any(), auditID).Return(&aiaudit.AIAudit{Status: aiaudit.StatusCompleted}, nil)
	mdb.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
	mg.EXPECT().Evaluate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errorsNew("invalid_evaluator_response: empty rationale"))
	mdb.EXPECT().AIPromptProposalUpdateFinal(gomock.Any(), pid, aipromptproposal.StatusFailed, "", "", string(aipromptproposal.ErrorInvalidEvaluatorResponse)).
		Return(int64(1), nil)

	h.runProposalJob(context.Background(), pid, "orig", []uuid.UUID{auditID}, "en-US")
}

func TestCreate_AuditDifferentCustomer_Returns400(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	cust := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())
	currentHist := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIGet(gomock.Any(), aiID).Return(&ai.AI{
		Identity:               commonidentity.Identity{CustomerID: cust},
		CurrentPromptHistoryID: currentHist,
	}, nil)
	mdb.EXPECT().AIPromptProposalCountProgressing(gomock.Any(), cust).Return(int64(0), nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), auditID).Return(&aiaudit.AIAudit{
		Identity:        commonidentity.Identity{CustomerID: uuid.Must(uuid.NewV4())},
		AIID:            aiID,
		Status:          aiaudit.StatusCompleted,
		PromptHistoryID: currentHist,
	}, nil)

	_, err := h.Create(context.Background(), cust, aiID, []uuid.UUID{auditID}, "en-US")
	if err == nil || !strings.Contains(err.Error(), "not owned") {
		t.Fatalf("expected 'not owned', got: %v", err)
	}
}

func TestCreate_AuditDeleted_Returns400(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	cust := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())
	currentHist := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())
	tm := timePtr()

	mdb.EXPECT().AIGet(gomock.Any(), aiID).Return(&ai.AI{
		Identity:               commonidentity.Identity{CustomerID: cust},
		CurrentPromptHistoryID: currentHist,
	}, nil)
	mdb.EXPECT().AIPromptProposalCountProgressing(gomock.Any(), cust).Return(int64(0), nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), auditID).Return(&aiaudit.AIAudit{
		Identity:        commonidentity.Identity{CustomerID: cust},
		AIID:            aiID,
		Status:          aiaudit.StatusCompleted,
		PromptHistoryID: currentHist,
		TMDelete:        tm,
	}, nil)

	_, err := h.Create(context.Background(), cust, aiID, []uuid.UUID{auditID}, "en-US")
	if err == nil || !strings.Contains(err.Error(), "deleted") {
		t.Fatalf("expected 'deleted', got: %v", err)
	}
}

func TestCreate_AuditDifferentAI_Returns400(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	cust := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())
	otherAI := uuid.Must(uuid.NewV4())
	currentHist := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIGet(gomock.Any(), aiID).Return(&ai.AI{
		Identity:               commonidentity.Identity{CustomerID: cust},
		CurrentPromptHistoryID: currentHist,
	}, nil)
	mdb.EXPECT().AIPromptProposalCountProgressing(gomock.Any(), cust).Return(int64(0), nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), auditID).Return(&aiaudit.AIAudit{
		Identity:        commonidentity.Identity{CustomerID: cust},
		AIID:            otherAI,
		Status:          aiaudit.StatusCompleted,
		PromptHistoryID: currentHist,
	}, nil)

	_, err := h.Create(context.Background(), cust, aiID, []uuid.UUID{auditID}, "en-US")
	if err == nil || !strings.Contains(err.Error(), "different AI") {
		t.Fatalf("expected 'different AI', got: %v", err)
	}
}

func TestCreate_AuditNotCompleted_Returns400(t *testing.T) {
	h, mdb, _, mc := newHandlerWithMocks(t)
	defer mc.Finish()

	cust := uuid.Must(uuid.NewV4())
	aiID := uuid.Must(uuid.NewV4())
	currentHist := uuid.Must(uuid.NewV4())
	auditID := uuid.Must(uuid.NewV4())

	mdb.EXPECT().AIGet(gomock.Any(), aiID).Return(&ai.AI{
		Identity:               commonidentity.Identity{CustomerID: cust},
		CurrentPromptHistoryID: currentHist,
	}, nil)
	mdb.EXPECT().AIPromptProposalCountProgressing(gomock.Any(), cust).Return(int64(0), nil)
	mdb.EXPECT().AIAuditGet(gomock.Any(), auditID).Return(&aiaudit.AIAudit{
		Identity:        commonidentity.Identity{CustomerID: cust},
		AIID:            aiID,
		Status:          aiaudit.StatusProgressing,
		PromptHistoryID: currentHist,
	}, nil)

	_, err := h.Create(context.Background(), cust, aiID, []uuid.UUID{auditID}, "en-US")
	if err == nil || !strings.Contains(err.Error(), "not completed") {
		t.Fatalf("expected 'not completed', got: %v", err)
	}
}
