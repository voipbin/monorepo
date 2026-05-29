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

// AIPromptProposalDelete soft-deletes a proposal row.
func (h *handler) AIPromptProposalDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	query, args, err := sq.Update(aipromptproposalTable).
		SetMap(map[string]any{"tm_update": ts, "tm_delete": ts}).
		Where(sq.And{sq.Eq{"id": id.Bytes()}, sq.Eq{"tm_delete": nil}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("AIPromptProposalDelete: could not build query. err: %v", err)
	}

	result, err := h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("AIPromptProposalDelete: could not execute. err: %v", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("AIPromptProposalDelete: rows affected. err: %v", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// AIPromptProposalCountProgressing returns the number of in-flight proposals for a customer.
func (h *handler) AIPromptProposalCountProgressing(ctx context.Context, customerID uuid.UUID) (int64, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE customer_id = ? AND status = 'progressing' AND tm_delete IS NULL
	`, aipromptproposalTable)

	row := h.db.QueryRowContext(ctx, query, customerID.Bytes())
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("AIPromptProposalCountProgressing: scan. err: %v", err)
	}
	return count, nil
}

// AIAcceptProposal atomically applies an accepted proposal. Lock order: proposal -> AI.
func (h *handler) AIAcceptProposal(ctx context.Context, proposalID uuid.UUID, newHistoryID uuid.UUID, proposedPrompt string) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("AIAcceptProposal: BeginTx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 1. Lock proposal row.
	var pCustomer, pAIID, pBasis []byte
	var pTMDelete sql.NullTime
	var pStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT customer_id, ai_id, basis_prompt_history_id, status, tm_delete
		FROM `+aipromptproposalTable+`
		WHERE id = ?
		FOR UPDATE
	`, proposalID.Bytes()).Scan(&pCustomer, &pAIID, &pBasis, &pStatus, &pTMDelete)
	if err == sql.ErrNoRows {
		err = ErrNotFound
		return err
	}
	if err != nil {
		err = fmt.Errorf("AIAcceptProposal: select proposal: %w", err)
		return err
	}
	if pTMDelete.Valid {
		err = ErrNotFound
		return err
	}
	if pStatus != string(aipromptproposal.StatusCompleted) {
		err = ErrProposalNotAcceptable
		return err
	}

	aiID, _ := uuid.FromBytes(pAIID)
	basisID, _ := uuid.FromBytes(pBasis)
	customerID, _ := uuid.FromBytes(pCustomer)

	// 2. Lock AI row and re-check basis.
	var aiCurrentHistory []byte
	var aiTMDelete sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT current_prompt_history_id, tm_delete
		FROM ai_ais
		WHERE id = ?
		FOR UPDATE
	`, aiID.Bytes()).Scan(&aiCurrentHistory, &aiTMDelete)
	if err == sql.ErrNoRows {
		err = ErrNotFound
		return err
	}
	if err != nil {
		err = fmt.Errorf("AIAcceptProposal: select AI: %w", err)
		return err
	}
	if aiTMDelete.Valid {
		err = ErrNotFound
		return err
	}
	currentID, _ := uuid.FromBytes(aiCurrentHistory)
	if currentID != basisID {
		err = ErrPromptVersionDrifted
		return err
	}

	now := h.utilHandler.TimeNow()

	// 3. Insert new prompt history row.
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO ai_ai_prompt_histories (id, customer_id, ai_id, prompt, proposal_id, tm_create)
		VALUES (?, ?, ?, ?, ?, ?)
	`, newHistoryID.Bytes(), customerID.Bytes(), aiID.Bytes(), proposedPrompt, proposalID.Bytes(), now); err != nil {
		err = fmt.Errorf("AIAcceptProposal: insert history: %w", err)
		return err
	}

	// 4. Update AI.
	if _, err = tx.ExecContext(ctx, `
		UPDATE ai_ais
		SET init_prompt = ?, current_prompt_history_id = ?, tm_update = ?
		WHERE id = ?
	`, proposedPrompt, newHistoryID.Bytes(), now, aiID.Bytes()); err != nil {
		err = fmt.Errorf("AIAcceptProposal: update AI: %w", err)
		return err
	}

	// 5. Update proposal.
	result, uerr := tx.ExecContext(ctx, `
		UPDATE `+aipromptproposalTable+`
		SET status = 'accepted', applied_prompt_history_id = ?, tm_update = ?
		WHERE id = ? AND status = 'completed' AND tm_delete IS NULL
	`, newHistoryID.Bytes(), now, proposalID.Bytes())
	if uerr != nil {
		err = fmt.Errorf("AIAcceptProposal: update proposal: %w", uerr)
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		err = fmt.Errorf("AIAcceptProposal: proposal not updated (lock invariant violated)")
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("AIAcceptProposal: commit: %w", err)
	}
	return nil
}
