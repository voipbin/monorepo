package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aiaudit"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
)

const (
	aiauditTable = "ai_ai_audits"
)

// AIAuditUpsert inserts a new audit row or resets an existing non-progressing one.
// Returns rowsAffected from sql.Result:
//   - 0 means the row exists and is already 'progressing' (caller should return 409)
//   - >0 means the row was created or overwritten
//
// NOTE: The DB DSN must NOT include CLIENT_FOUND_ROWS.
func (h *handler) AIAuditUpsert(ctx context.Context, a *aiaudit.AIAudit) (int64, error) {
	a.TMCreate = h.utilHandler.TimeNow()
	a.TMUpdate = nil

	query := fmt.Sprintf(`
		INSERT INTO %s
			(id, customer_id, aicall_id, ai_id, prompt_history_id, status, overall_score, evaluation, language, error, tm_create, tm_update, tm_delete)
		VALUES
			(?, ?, ?, ?, ?, 'progressing', NULL, NULL, ?, NULL, ?, NULL, NULL)
		ON DUPLICATE KEY UPDATE
			status            = IF(status = 'progressing', status, 'progressing'),
			tm_delete         = NULL,
			overall_score     = NULL,
			evaluation        = NULL,
			message_ids       = NULL,
			error             = NULL,
			language          = VALUES(language),
			prompt_history_id = VALUES(prompt_history_id),
			tm_update         = NULL
	`, aiauditTable)

	result, err := h.db.ExecContext(ctx, query,
		a.ID.Bytes(),
		a.CustomerID.Bytes(),
		a.AIcallID.Bytes(),
		a.AIID.Bytes(),
		a.PromptHistoryID.Bytes(),
		a.Language,
		a.TMCreate,
	)
	if err != nil {
		return 0, fmt.Errorf("AIAuditUpsert: could not execute. err: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("AIAuditUpsert: could not get rows affected. err: %v", err)
	}

	return n, nil
}

// aiauditGetFromDB returns an audit record from the DB by ID.
func (h *handler) aiauditGetFromDB(id uuid.UUID) (*aiaudit.AIAudit, error) {
	cols := commondatabasehandler.GetDBFields(aiaudit.AIAudit{})

	query, args, err := sq.Select(cols...).
		From(aiauditTable).
		Where(sq.And{
			sq.Eq{"id": id.Bytes()},
			sq.Eq{"tm_delete": nil},
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("aiauditGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("aiauditGetFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &aiaudit.AIAudit{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("aiauditGetFromDB: could not scan. err: %v", err)
	}

	return res, nil
}

// AIAuditGet returns an audit record by ID.
func (h *handler) AIAuditGet(ctx context.Context, id uuid.UUID) (*aiaudit.AIAudit, error) {
	return h.aiauditGetFromDB(id)
}

// AIAuditList returns a paginated list of audit records.
func (h *handler) AIAuditList(ctx context.Context, size uint64, token string, filters map[aiaudit.Field]any) ([]*aiaudit.AIAudit, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(aiaudit.AIAudit{})

	builder := sq.Select(cols...).
		From(aiauditTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("AIAuditList: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("AIAuditList: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AIAuditList: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*aiaudit.AIAudit{}
	for rows.Next() {
		a := &aiaudit.AIAudit{}
		if err := commondatabasehandler.ScanRow(rows, a); err != nil {
			return nil, fmt.Errorf("AIAuditList: could not scan. err: %v", err)
		}
		res = append(res, a)
	}

	return res, nil
}

// AIAuditDelete soft-deletes an audit record.
func (h *handler) AIAuditDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	query, args, err := sq.Update(aiauditTable).
		SetMap(map[string]any{
			"tm_update": ts,
			"tm_delete": ts,
		}).
		Where(sq.And{
			sq.Eq{"id": id.Bytes()},
			sq.Eq{"tm_delete": nil},
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("AIAuditDelete: could not build query. err: %v", err)
	}

	result, err := h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("AIAuditDelete: could not execute. err: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("AIAuditDelete: could not get rows affected. err: %v", err)
	}
	if n == 0 {
		return ErrNotFound
	}

	return nil
}

// AIAuditUpdateFinal writes the goroutine's final result atomically.
// Only updates rows where status='progressing' AND tm_delete IS NULL to
// prevent overwriting a 'failed' status set by the stale recovery sweep.
// Returns rowsAffected: 0 means the record was already soft-deleted or swept.
func (h *handler) AIAuditUpdateFinal(ctx context.Context, id uuid.UUID, status aiaudit.Status, overallScore *int, evaluation json.RawMessage, errStr string, messageIDs []uuid.UUID) (int64, error) {
	ts := h.utilHandler.TimeNow()

	var evalJSON sql.NullString
	if evaluation != nil {
		evalJSON = sql.NullString{String: string(evaluation), Valid: true}
	}

	var msgIDsJSON sql.NullString
	if len(messageIDs) > 0 {
		b, err := json.Marshal(messageIDs)
		if err != nil {
			return 0, fmt.Errorf("AIAuditUpdateFinal: could not marshal message_ids: %v", err)
		}
		msgIDsJSON = sql.NullString{String: string(b), Valid: true}
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET status = ?, overall_score = ?, evaluation = ?, message_ids = ?, error = ?, tm_update = ?
		WHERE id = ? AND tm_delete IS NULL AND status = 'progressing'
	`, aiauditTable)

	result, err := h.db.ExecContext(ctx, query,
		string(status),  // 1
		overallScore,    // 2
		evalJSON,        // 3
		msgIDsJSON,      // 4
		errStr,          // 5
		ts,              // 6
		id.Bytes(),      // 7 (WHERE)
	)
	if err != nil {
		return 0, fmt.Errorf("AIAuditUpdateFinal: could not execute. err: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("AIAuditUpdateFinal: could not get rows affected. err: %v", err)
	}

	return n, nil
}

// AIAuditCountProgressing returns the number of in-flight audits for a customer.
func (h *handler) AIAuditCountProgressing(ctx context.Context, customerID uuid.UUID) (int64, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE customer_id = ? AND status = 'progressing' AND tm_delete IS NULL
	`, aiauditTable)

	row := h.db.QueryRowContext(ctx, query, customerID.Bytes())

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("AIAuditCountProgressing: could not scan. err: %v", err)
	}

	return count, nil
}
