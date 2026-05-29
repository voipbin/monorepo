package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

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

// aipromptproposalGetFromDB fetches one row by ID, active (tm_delete IS NULL) only.
func (h *handler) aipromptproposalGetFromDB(id uuid.UUID) (*aipromptproposal.AIPromptProposal, error) {
	cols := commondatabasehandler.GetDBFields(aipromptproposal.AIPromptProposal{})

	query, args, err := sq.Select(cols...).
		From(aipromptproposalTable).
		Where(sq.And{sq.Eq{"id": id.Bytes()}, sq.Eq{"tm_delete": nil}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("aipromptproposalGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("aipromptproposalGetFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &aipromptproposal.AIPromptProposal{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("aipromptproposalGetFromDB: could not scan. err: %v", err)
	}
	return res, nil
}

// AIPromptProposalGet returns one proposal by ID.
func (h *handler) AIPromptProposalGet(ctx context.Context, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error) {
	return h.aipromptproposalGetFromDB(id)
}

// AIPromptProposalList — token interpreted as WHERE tm_create < token (mirrors AIAuditList).
func (h *handler) AIPromptProposalList(ctx context.Context, size uint64, token string, filters map[aipromptproposal.Field]any) ([]*aipromptproposal.AIPromptProposal, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(aipromptproposal.AIPromptProposal{})

	builder := sq.Select(cols...).
		From(aipromptproposalTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("AIPromptProposalList: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("AIPromptProposalList: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AIPromptProposalList: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*aipromptproposal.AIPromptProposal{}
	for rows.Next() {
		p := &aipromptproposal.AIPromptProposal{}
		if err := commondatabasehandler.ScanRow(rows, p); err != nil {
			return nil, fmt.Errorf("AIPromptProposalList: could not scan. err: %v", err)
		}
		res = append(res, p)
	}
	return res, nil
}

// AIPromptProposalUpdateFinal writes the goroutine's final result. Guard: WHERE status='progressing' AND tm_delete IS NULL.
func (h *handler) AIPromptProposalUpdateFinal(ctx context.Context, id uuid.UUID, status aipromptproposal.Status, proposedPrompt, rationale, errStr string) (int64, error) {
	ts := h.utilHandler.TimeNow()

	query := fmt.Sprintf(`
		UPDATE %s
		SET status = ?, proposed_prompt = ?, rationale = ?, error = ?, tm_update = ?
		WHERE id = ? AND tm_delete IS NULL AND status = 'progressing'
	`, aipromptproposalTable)

	result, err := h.db.ExecContext(ctx, query,
		string(status),
		sql.NullString{String: proposedPrompt, Valid: proposedPrompt != ""},
		sql.NullString{String: rationale, Valid: rationale != ""},
		errStr,
		ts,
		id.Bytes(),
	)
	if err != nil {
		return 0, fmt.Errorf("AIPromptProposalUpdateFinal: could not execute. err: %v", err)
	}
	return result.RowsAffected()
}

// AIPromptProposalUpdateExpired marks a completed proposal as expired (drift case).
func (h *handler) AIPromptProposalUpdateExpired(ctx context.Context, id uuid.UUID, errStr string) (int64, error) {
	ts := h.utilHandler.TimeNow()
	query := fmt.Sprintf(`
		UPDATE %s
		SET status = 'expired', error = ?, tm_update = ?
		WHERE id = ? AND tm_delete IS NULL AND status = 'completed'
	`, aipromptproposalTable)

	result, err := h.db.ExecContext(ctx, query, errStr, ts, id.Bytes())
	if err != nil {
		return 0, fmt.Errorf("AIPromptProposalUpdateExpired: could not execute. err: %v", err)
	}
	return result.RowsAffected()
}

// AIPromptProposalUpdateRejected marks a completed proposal as rejected by user.
func (h *handler) AIPromptProposalUpdateRejected(ctx context.Context, id uuid.UUID) (int64, error) {
	ts := h.utilHandler.TimeNow()
	query := fmt.Sprintf(`
		UPDATE %s
		SET status = 'rejected', tm_update = ?
		WHERE id = ? AND tm_delete IS NULL AND status = 'completed'
	`, aipromptproposalTable)

	result, err := h.db.ExecContext(ctx, query, ts, id.Bytes())
	if err != nil {
		return 0, fmt.Errorf("AIPromptProposalUpdateRejected: could not execute. err: %v", err)
	}
	return result.RowsAffected()
}
