package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"monorepo/bin-ai-manager/models/aipromptproposal"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
)

const aipromptproposalTable = "ai_ai_prompt_proposals"

// Sentinel errors specific to AIAcceptProposal.
var (
	ErrPromptVersionDrifted  = fmt.Errorf("prompt_version_drifted")
	ErrProposalNotAcceptable = fmt.Errorf("proposal_not_acceptable")
)

// AIPromptProposalCreate inserts a new proposal row with status='progressing'.
func (h *handler) AIPromptProposalCreate(ctx context.Context, p *aipromptproposal.AIPromptProposal) error {
	p.TMCreate = h.utilHandler.TimeNow()
	p.TMUpdate = nil
	p.TMDelete = nil
	p.Status = aipromptproposal.StatusProgressing
	p.Error = ""

	fields, err := commondatabasehandler.PrepareFields(p)
	if err != nil {
		return fmt.Errorf("AIPromptProposalCreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(aipromptproposalTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("AIPromptProposalCreate: could not build query. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("AIPromptProposalCreate: could not execute. err: %v", err)
	}
	return nil
}
