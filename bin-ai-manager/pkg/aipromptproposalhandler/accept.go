package aipromptproposalhandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// Accept atomically applies a completed proposal: writes a new prompt history row,
// updates the AI's init_prompt and current_prompt_history_id, and marks the proposal accepted.
func (h *aipromptproposalHandler) Accept(ctx context.Context, customerID uuid.UUID, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error) {
	log := logrus.WithFields(logrus.Fields{"func": "aipromptproposalHandler.Accept", "id": id})

	p, err := h.db.AIPromptProposalGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("proposal not found: %w", err)
	}
	if p.CustomerID != customerID {
		return nil, fmt.Errorf("proposal not found")
	}

	if p.Status == aipromptproposal.StatusAccepted {
		log.Debug("proposal already accepted; returning idempotent success")
		return p, nil
	}
	if p.Status != aipromptproposal.StatusCompleted {
		return nil, fmt.Errorf("proposal not completed (status=%s)", p.Status)
	}

	for _, auditID := range p.AuditIDs {
		a, gerr := h.db.AIAuditGet(ctx, auditID)
		if gerr != nil || a.TMDelete != nil || a.Status != aiaudit.StatusCompleted {
			if _, uerr := h.db.AIPromptProposalUpdateExpired(ctx, id, string(aipromptproposal.ErrorInvalidAuditSet)); uerr != nil {
				log.WithError(uerr).Error("could not mark proposal expired after audit invalidation")
			}
			return nil, fmt.Errorf("audit set invalidated")
		}
	}

	newHistoryID := h.utilHandler.UUIDCreate()
	err = h.db.AIAcceptProposal(ctx, id, newHistoryID, p.ProposedPrompt)
	switch {
	case err == nil:
	case errors.Is(err, dbhandler.ErrPromptVersionDrifted):
		if _, uerr := h.db.AIPromptProposalUpdateExpired(ctx, id, string(aipromptproposal.ErrorPromptVersionDrifted)); uerr != nil {
			log.WithError(uerr).Error("could not mark proposal expired after drift")
		}
		return nil, fmt.Errorf("prompt version drifted")
	case errors.Is(err, dbhandler.ErrNotFound):
		return nil, fmt.Errorf("proposal not found")
	case errors.Is(err, dbhandler.ErrProposalNotAcceptable):
		return nil, fmt.Errorf("proposal not completed")
	default:
		return nil, fmt.Errorf("could not apply proposal: %w", err)
	}

	post, err := h.db.AIPromptProposalGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not reload accepted proposal: %w", err)
	}
	return post, nil
}

// Reject marks a completed proposal as user-rejected.
func (h *aipromptproposalHandler) Reject(ctx context.Context, customerID uuid.UUID, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error) {
	p, err := h.db.AIPromptProposalGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("proposal not found: %w", err)
	}
	if p.CustomerID != customerID {
		return nil, fmt.Errorf("proposal not found")
	}

	if p.Status == aipromptproposal.StatusRejected {
		return p, nil
	}
	if p.Status != aipromptproposal.StatusCompleted {
		return nil, fmt.Errorf("proposal not completed (status=%s)", p.Status)
	}

	n, err := h.db.AIPromptProposalUpdateRejected(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not reject proposal: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("proposal not completed (race)")
	}

	return h.db.AIPromptProposalGet(ctx, id)
}
