package analysisdbhandler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-timeline-manager/models/analysis"
)

const (
	analysisTable = "timeline_analyses"
)

// mysqlErrDupEntry is the MySQL error number for a duplicate unique key.
const mysqlErrDupEntry = 1062

// AnalysisCreate inserts a fresh progressing row.
//
// On a UNIQUE(activeflow_id) violation (a concurrent trigger won the insert) it
// returns ErrDuplicate so the caller can re-read and return the in-flight row.
// Because delete is a hard delete, there is never a lingering row to collide
// with except a genuinely live (in-flight or terminal) one.
func (h *handler) AnalysisCreate(ctx context.Context, a *analysis.Analysis) error {
	a.TMCreate = h.utilHandler.TimeNow()
	a.TMUpdate = nil

	var result sql.NullString
	if len(a.Result) > 0 {
		result = sql.NullString{String: string(a.Result), Valid: true}
	}

	query := fmt.Sprintf(`
		INSERT INTO %s
			(id, customer_id, activeflow_id, status, result, model, error, tm_create, tm_update)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, NULL)
	`, analysisTable)

	_, err := h.db.ExecContext(ctx, query,
		a.ID.Bytes(),
		a.CustomerID.Bytes(),
		a.ActiveflowID.Bytes(),
		string(a.Status),
		result,
		a.Model,
		a.Error,
		a.TMCreate,
	)
	if err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == mysqlErrDupEntry {
			return ErrDuplicate
		}
		return fmt.Errorf("AnalysisCreate: could not execute. err: %v", err)
	}

	return nil
}

// analysisGetFromDB returns a row by id (no ownership check).
func (h *handler) analysisGetFromDB(id uuid.UUID) (*analysis.Analysis, error) {
	cols := commondatabasehandler.GetDBFields(analysis.Analysis{})

	query, args, err := sq.Select(cols...).
		From(analysisTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("analysisGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("analysisGetFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &analysis.Analysis{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("analysisGetFromDB: could not scan. err: %v", err)
	}

	return res, nil
}

// AnalysisGet returns an analysis by id, ownership-checked (masked not-found).
func (h *handler) AnalysisGet(ctx context.Context, customerID, id uuid.UUID) (*analysis.Analysis, error) {
	res, err := h.analysisGetFromDB(id)
	if err != nil {
		return nil, err
	}
	if res.CustomerID != customerID {
		// masked: no existence oracle.
		return nil, ErrNotFound
	}
	return res, nil
}

// AnalysisGetByActiveflowID returns the analysis for an activeflow, ownership-checked.
func (h *handler) AnalysisGetByActiveflowID(ctx context.Context, customerID, activeflowID uuid.UUID) (*analysis.Analysis, error) {
	cols := commondatabasehandler.GetDBFields(analysis.Analysis{})

	query, args, err := sq.Select(cols...).
		From(analysisTable).
		Where(sq.And{
			sq.Eq{"activeflow_id": activeflowID.Bytes()},
			sq.Eq{"customer_id": customerID.Bytes()},
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("AnalysisGetByActiveflowID: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AnalysisGetByActiveflowID: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &analysis.Analysis{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("AnalysisGetByActiveflowID: could not scan. err: %v", err)
	}

	return res, nil
}

// AnalysisList returns a paginated list (always filtered by customer_id, the
// server-side authority — review F2/C1).
func (h *handler) AnalysisList(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[analysis.Field]any) ([]*analysis.Analysis, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(analysis.Analysis{})

	builder := sq.Select(cols...).
		From(analysisTable).
		Where(sq.Lt{"tm_create": token}).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("AnalysisList: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("AnalysisList: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AnalysisList: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*analysis.Analysis{}
	for rows.Next() {
		a := &analysis.Analysis{}
		if err := commondatabasehandler.ScanRow(rows, a); err != nil {
			return nil, fmt.Errorf("AnalysisList: could not scan. err: %v", err)
		}
		res = append(res, a)
	}

	return res, nil
}

// AnalysisReset flips a non-progressing live row back to progressing for a
// re-analyze, in place (CAS on status!='progressing' — review H2). Returns the
// reset UPDATE's rowsAffected: 0 means another reanalyze already won the
// transition (or the row is already progressing), so the caller must not start
// a second chain. The superseded verdict is intentionally overwritten (no
// history table; the analysis is a reproducible derivative of the timeline).
func (h *handler) AnalysisReset(ctx context.Context, id uuid.UUID) (int64, error) {
	ts := h.utilHandler.TimeNow()
	updateQuery := fmt.Sprintf(`
		UPDATE %s
		SET status = 'progressing', result = NULL, error = '', tm_update = ?
		WHERE id = ? AND status != 'progressing'
	`, analysisTable)

	result, err := h.db.ExecContext(ctx, updateQuery, ts, id.Bytes())
	if err != nil {
		return 0, fmt.Errorf("AnalysisReset: could not execute reset. err: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("AnalysisReset: could not get rows affected. err: %v", err)
	}

	return n, nil
}

// AnalysisUpdateResult writes the final result. Guards on status='progressing'
// so a row hard-deleted while the goroutine ran is NOT resurrected (the row is
// gone, rowsAffected=0). Returns rowsAffected.
func (h *handler) AnalysisUpdateResult(ctx context.Context, id uuid.UUID, status analysis.Status, result []byte, model, errStr string) (int64, error) {
	ts := h.utilHandler.TimeNow()

	var resultJSON sql.NullString
	if len(result) > 0 {
		resultJSON = sql.NullString{String: string(result), Valid: true}
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET status = ?, result = ?, model = ?, error = ?, tm_update = ?
		WHERE id = ? AND status = 'progressing'
	`, analysisTable)

	res, err := h.db.ExecContext(ctx, query,
		string(status),
		resultJSON,
		model,
		errStr,
		ts,
		id.Bytes(),
	)
	if err != nil {
		return 0, fmt.Errorf("AnalysisUpdateResult: could not execute. err: %v", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("AnalysisUpdateResult: could not get rows affected. err: %v", err)
	}

	return n, nil
}

// AnalysisDelete hard-deletes the live row, ownership-checked. The row is
// physically removed so the activeflow becomes freshly analyzable again (no
// soft-delete column, no UNIQUE(activeflow_id) collision on re-create). Returns
// the deleted record. A cross-customer or absent row returns masked ErrNotFound.
func (h *handler) AnalysisDelete(ctx context.Context, customerID, id uuid.UUID) (*analysis.Analysis, error) {
	// resolve + ownership check first (masked not-found). The delete itself
	// re-checks customer_id so a concurrent delete can't widen the scope.
	existing, err := h.AnalysisGet(ctx, customerID, id)
	if err != nil {
		return nil, err
	}

	deleteQuery := fmt.Sprintf(`
		DELETE FROM %s
		WHERE id = ? AND customer_id = ?
	`, analysisTable)

	r, err := h.db.ExecContext(ctx, deleteQuery, id.Bytes(), customerID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("AnalysisDelete: could not execute delete. err: %v", err)
	}
	n, err := r.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("AnalysisDelete: could not get rows affected. err: %v", err)
	}
	if n == 0 {
		// lost a concurrent delete race; masked not-found.
		return nil, ErrNotFound
	}

	return existing, nil
}

// AnalysisCountProgressing returns the number of in-flight (progressing)
// analyses for a customer (design F1 per-customer cap).
func (h *handler) AnalysisCountProgressing(ctx context.Context, customerID uuid.UUID) (int64, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE customer_id = ? AND status = 'progressing'
	`, analysisTable)

	row := h.db.QueryRowContext(ctx, query, customerID.Bytes())

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("AnalysisCountProgressing: could not scan. err: %v", err)
	}

	return count, nil
}
