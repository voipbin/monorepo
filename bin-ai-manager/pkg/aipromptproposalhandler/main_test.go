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

var (
	_ = timePtr              // suppress unused if needed
	_ = injectRealUtilHandler // used by other test files in the package
)
