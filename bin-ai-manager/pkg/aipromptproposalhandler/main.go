package aipromptproposalhandler

//go:generate mockgen -package aipromptproposalhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-ai-manager/models/message"
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

// runProposalJob runs the Gemini evaluation for a single proposal in a goroutine.
func (h *aipromptproposalHandler) runProposalJob(parent context.Context, proposalID uuid.UUID, basisPrompt string, auditIDs []uuid.UUID, language string) {
	h.semaphore <- struct{}{}

	log := logrus.WithFields(logrus.Fields{
		"func":        "aipromptproposalHandler.runProposalJob",
		"proposal_id": proposalID,
	})

	ctx, cancel := context.WithTimeout(parent, geminiTimeoutSeconds*time.Second)
	defer cancel()

	finalStatus := aipromptproposal.StatusFailed
	finalProposed := ""
	finalRationale := ""
	finalErr := ""

	defer func() {
		defer func() { <-h.semaphore }()
		if r := recover(); r != nil {
			log.Errorf("panic in runProposalJob: %v\n%s", r, debug.Stack())
			finalErr = string(aipromptproposal.ErrorEvaluatorUnavailable)
			finalStatus = aipromptproposal.StatusFailed
			finalProposed = ""
			finalRationale = ""
		}
		writeCtx, writeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer writeCancel()
		n, dbErr := h.db.AIPromptProposalUpdateFinal(writeCtx, proposalID, finalStatus, finalProposed, finalRationale, finalErr)
		if dbErr != nil {
			log.WithError(dbErr).Error("could not write final proposal result")
		} else if n == 0 {
			log.Warnf("final proposal result not written: row was deleted or swept (intended status=%s)", finalStatus)
		} else {
			log.Debugf("final proposal result written: status=%s", finalStatus)
		}
	}()

	select {
	case <-ctx.Done():
		log.Warn("context cancelled before Gemini call")
		finalErr = string(aipromptproposal.ErrorCancelled)
		return
	default:
	}

	blocks, blockErr := h.loadAuditBlocks(ctx, auditIDs)
	if blockErr != nil {
		log.WithError(blockErr).Error("could not load audit blocks")
		finalErr = string(aipromptproposal.ErrorEvaluatorUnavailable)
		return
	}

	resp, err := h.geminiHandler.Evaluate(ctx, basisPrompt, blocks, language)
	if err != nil {
		log.WithError(err).Error("gemini evaluation failed")
		if strings.Contains(err.Error(), "invalid_evaluator_response") {
			finalErr = string(aipromptproposal.ErrorInvalidEvaluatorResponse)
		} else {
			finalErr = string(aipromptproposal.ErrorEvaluatorUnavailable)
		}
		return
	}

	finalStatus = aipromptproposal.StatusCompleted
	finalProposed = resp.ProposedPrompt
	finalRationale = resp.Rationale
}

// loadAuditBlocks loads audit records and their transcripts into AuditBlocks for Gemini.
func (h *aipromptproposalHandler) loadAuditBlocks(ctx context.Context, auditIDs []uuid.UUID) ([]geminiproposalhandler.AuditBlock, error) {
	out := make([]geminiproposalhandler.AuditBlock, 0, len(auditIDs))
	for i, id := range auditIDs {
		a, err := h.db.AIAuditGet(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not load audit %s: %w", id, err)
		}
		if a.TMDelete != nil || a.Status != aiaudit.StatusCompleted {
			continue
		}

		dim, sumErr := parseAuditEvaluation(a.Evaluation)
		if sumErr != nil {
			return nil, fmt.Errorf("could not parse evaluation for audit %s: %w", id, sumErr)
		}

		msgs, msgErr := h.db.MessageList(ctx, 500, "", map[message.Field]any{
			message.FieldAIcallID: a.AIcallID,
			message.FieldDeleted:  false,
		})
		if msgErr != nil {
			return nil, fmt.Errorf("could not load messages for audit %s: %w", id, msgErr)
		}

		transcript := buildTranscript(msgs, maxTranscriptCharsPerAudit)

		out = append(out, geminiproposalhandler.AuditBlock{
			Index:           i + 1,
			OverallScore:    derefScore(a.OverallScore),
			HelpfulnessR:    dim.Helpfulness.Reason,
			AccuracyR:       dim.Accuracy.Reason,
			ToneR:           dim.Tone.Reason,
			GoalCompletionR: dim.GoalCompletion.Reason,
			ToolUsageR:      dim.ToolUsageR,
			Summary:         dim.Summary,
			Transcript:      transcript,
		})
	}
	return out, nil
}

func derefScore(p *int) int {
	if p == nil {
		return 0
	}
	return *p
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
