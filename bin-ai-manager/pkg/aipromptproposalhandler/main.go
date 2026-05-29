package aipromptproposalhandler

//go:generate mockgen -package aipromptproposalhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/geminiproposalhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

const (
	maxConcurrentGlobal        = 30
	maxConcurrentCustomer      = 3
	geminiTimeoutSeconds       = 60
	maxAuditsPerProposal       = 20
	maxTranscriptCharsPerAudit = 15000
	staleProposalAgeMinutes    = 5
	proposalExpiryHours        = 168
	maxProposedPromptChars     = 32000
)

// AIPromptProposalHandler handles AI prompt proposal lifecycle operations.
type AIPromptProposalHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, aiID uuid.UUID, auditIDs []uuid.UUID, language string) (*aipromptproposal.AIPromptProposal, error)
	Get(ctx context.Context, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error)
	List(ctx context.Context, size uint64, token string, filters map[aipromptproposal.Field]any) ([]*aipromptproposal.AIPromptProposal, error)
	Accept(ctx context.Context, customerID uuid.UUID, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error)
	Reject(ctx context.Context, customerID uuid.UUID, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error)
	Delete(ctx context.Context, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error)
	SweepStaleProposals(ctx context.Context)
	SweepExpiredProposals(ctx context.Context)
}

type aipromptproposalHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	geminiHandler geminiproposalhandler.GeminiProposalHandler
	semaphore     chan struct{}
}

// NewAIPromptProposalHandler creates a new handler.
func NewAIPromptProposalHandler(
	db dbhandler.DBHandler,
	geminiHandler geminiproposalhandler.GeminiProposalHandler,
) *aipromptproposalHandler {
	return &aipromptproposalHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		geminiHandler: geminiHandler,
		semaphore:     make(chan struct{}, maxConcurrentGlobal),
	}
}

// Suppress unused-import until later tasks add code.
var _ = time.Second

// Create validates the request, captures the basis prompt, INSERTs the proposal row,
// then spawns the background goroutine.
func (h *aipromptproposalHandler) Create(ctx context.Context, customerID uuid.UUID, aiID uuid.UUID, auditIDs []uuid.UUID, language string) (*aipromptproposal.AIPromptProposal, error) {
	if len(auditIDs) == 0 {
		return nil, fmt.Errorf("invalid audit set: empty audit list")
	}
	if len(auditIDs) > maxAuditsPerProposal {
		return nil, fmt.Errorf("invalid audit set: too many audits (max %d)", maxAuditsPerProposal)
	}

	ai, err := h.db.AIGet(ctx, aiID)
	if err != nil {
		return nil, fmt.Errorf("ai not found: %w", err)
	}
	if ai.CustomerID != customerID {
		return nil, fmt.Errorf("ai not found")
	}
	if ai.TMDelete != nil {
		return nil, fmt.Errorf("ai not found")
	}

	count, err := h.db.AIPromptProposalCountProgressing(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("could not count progressing proposals: %w", err)
	}
	if count >= maxConcurrentCustomer {
		return nil, fmt.Errorf("rate limit exceeded: customer already has %d proposals in progress", count)
	}

	auditPromptMismatch := []uuid.UUID{}
	for _, auditID := range auditIDs {
		a, gerr := h.db.AIAuditGet(ctx, auditID)
		if gerr != nil {
			return nil, fmt.Errorf("invalid audit set: audit %s not found", auditID)
		}
		if a.CustomerID != customerID {
			return nil, fmt.Errorf("invalid audit set: audit %s not owned", auditID)
		}
		if a.TMDelete != nil {
			return nil, fmt.Errorf("invalid audit set: audit %s deleted", auditID)
		}
		if a.AIID != aiID {
			return nil, fmt.Errorf("invalid audit set: audit %s is for different AI", auditID)
		}
		if a.Status != aiaudit.StatusCompleted {
			return nil, fmt.Errorf("invalid audit set: audit %s not completed (status=%s)", auditID, a.Status)
		}
		if a.PromptHistoryID != ai.CurrentPromptHistoryID {
			auditPromptMismatch = append(auditPromptMismatch, auditID)
		}
	}
	if len(auditPromptMismatch) > 0 {
		ids := make([]string, len(auditPromptMismatch))
		for i, id := range auditPromptMismatch {
			ids[i] = id.String()
		}
		return nil, fmt.Errorf("audit prompt version mismatch: %s", strings.Join(ids, ","))
	}

	basis, err := h.db.AIPromptHistoryGet(ctx, ai.CurrentPromptHistoryID)
	if err != nil {
		return nil, fmt.Errorf("could not load basis prompt history: %w", err)
	}

	proposalID := h.utilHandler.UUIDCreate()
	p := &aipromptproposal.AIPromptProposal{
		Identity:             commonidentity.Identity{ID: proposalID, CustomerID: customerID},
		AIID:                 aiID,
		AuditIDs:             auditIDs,
		BasisPromptHistoryID: ai.CurrentPromptHistoryID,
		OriginalPrompt:       basis.Prompt,
	}
	if err := h.db.AIPromptProposalCreate(ctx, p); err != nil {
		return nil, fmt.Errorf("could not create proposal: %w", err)
	}

	reloaded, err := h.db.AIPromptProposalGet(ctx, proposalID)
	if err != nil {
		return nil, fmt.Errorf("could not reload proposal: %w", err)
	}

	go h.runProposalJob(context.Background(), proposalID, basis.Prompt, auditIDs, language)

	return reloaded, nil
}

// Stub for runProposalJob — filled in by Task 24.
func (h *aipromptproposalHandler) runProposalJob(ctx context.Context, proposalID uuid.UUID, basisPrompt string, auditIDs []uuid.UUID, language string) {
	_, _, _, _, _ = ctx, proposalID, basisPrompt, auditIDs, language
}

// Get returns one proposal by ID.
func (h *aipromptproposalHandler) Get(ctx context.Context, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error) {
	res, err := h.db.AIPromptProposalGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get proposal: %w", err)
	}
	return res, nil
}

// List returns a paginated list of proposals.
func (h *aipromptproposalHandler) List(ctx context.Context, size uint64, token string, filters map[aipromptproposal.Field]any) ([]*aipromptproposal.AIPromptProposal, error) {
	res, err := h.db.AIPromptProposalList(ctx, size, token, filters)
	if err != nil {
		return nil, fmt.Errorf("could not list proposals: %w", err)
	}
	return res, nil
}

// Delete soft-deletes a proposal and returns the pre-delete state.
func (h *aipromptproposalHandler) Delete(ctx context.Context, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error) {
	pre, err := h.db.AIPromptProposalGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get proposal before delete: %w", err)
	}
	if err := h.db.AIPromptProposalDelete(ctx, id); err != nil {
		return nil, fmt.Errorf("could not delete proposal: %w", err)
	}
	return pre, nil
}
