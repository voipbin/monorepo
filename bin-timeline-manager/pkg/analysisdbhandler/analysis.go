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
	"monorepo/bin-timeline-manager/models/analysishistory"
)

const (
	analysisTable        = "timeline_analyses"
	analysisHistoryTable = "timeline_analysis_histories"
)

// mysqlErrDupEntry is the MySQL error number for a duplicate unique key.
const mysqlErrDupEntry = 1062

// AnalysisCreate inserts a fresh progressing row.
//
// On a UNIQUE(activeflow_id) violation (a concurrent trigger won the insert) it
// returns ErrDuplicate so the caller can re-read and return the in-flight row.
func (h *handler) AnalysisCreate(ctx context.Context, a *analysis.Analysis) error {
	a.TMCreate = h.utilHandler.TimeNow()
	a.TMUpdate = nil
	a.TMDelete = nil

	var result sql.NullString
	if len(a.Result) > 0 {
		result = sql.NullString{String: string(a.Result), Valid: true}
	}

	query := fmt.Sprintf(`
		INSERT INTO %s
			(id, customer_id, activeflow_id, status, result, model, error, tm_create, tm_update, tm_delete)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL)
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

// analysisGetFromDB returns a live row by id (no ownership check).
func (h *handler) analysisGetFromDB(id uuid.UUID) (*analysis.Analysis, error) {
	cols := commondatabasehandler.GetDBFields(analysis.Analysis{})

	query, args, err := sq.Select(cols...).
		From(analysisTable).
		Where(sq.And{
			sq.Eq{"id": id.Bytes()},
			sq.Eq{"tm_delete": nil},
		}).
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

// AnalysisGetByActiveflowID returns the live analysis for an activeflow, ownership-checked.
func (h *handler) AnalysisGetByActiveflowID(ctx context.Context, customerID, activeflowID uuid.UUID) (*analysis.Analysis, error) {
	cols := commondatabasehandler.GetDBFields(analysis.Analysis{})

	query, args, err := sq.Select(cols...).
		From(analysisTable).
		Where(sq.And{
			sq.Eq{"activeflow_id": activeflowID.Bytes()},
			sq.Eq{"customer_id": customerID.Bytes()},
			sq.Eq{"tm_delete": nil},
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

// archiveTx writes a history snapshot of the current live row (within the given
// txn) for the supplied reason. It reads the live row inside the txn so the
// snapshot is consistent with the row being mutated. Returns ErrNotFound if the
// live row is gone. progressing rows are still archived on delete; on reanalyze
// the caller restricts to non-progressing via the reset CAS (a progressing row
// won't be reset, so its archive is rolled back with the txn).
func (h *handler) archiveTx(ctx context.Context, tx *sql.Tx, id uuid.UUID, reason analysishistory.Reason) error {
	cols := commondatabasehandler.GetDBFields(analysis.Analysis{})

	query, args, err := sq.Select(cols...).
		From(analysisTable).
		Where(sq.And{
			sq.Eq{"id": id.Bytes()},
			sq.Eq{"tm_delete": nil},
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("archiveTx: could not build select. err: %v", err)
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("archiveTx: could not query live row. err: %v", err)
	}
	live := &analysis.Analysis{}
	found := false
	if rows.Next() {
		if errScan := commondatabasehandler.ScanRow(rows, live); errScan != nil {
			_ = rows.Close()
			return fmt.Errorf("archiveTx: could not scan live row. err: %v", errScan)
		}
		found = true
	}
	_ = rows.Close()
	if !found {
		return ErrNotFound
	}

	var result sql.NullString
	if len(live.Result) > 0 {
		result = sql.NullString{String: string(live.Result), Valid: true}
	}

	insertQuery := fmt.Sprintf(`
		INSERT INTO %s
			(id, analysis_id, customer_id, activeflow_id, status, result, model, error, reason, tm_create)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, analysisHistoryTable)

	if _, err := tx.ExecContext(ctx, insertQuery,
		h.utilHandler.UUIDCreate().Bytes(),
		live.ID.Bytes(),
		live.CustomerID.Bytes(),
		live.ActiveflowID.Bytes(),
		string(live.Status),
		result,
		live.Model,
		live.Error,
		string(reason),
		h.utilHandler.TimeNow(),
	); err != nil {
		return fmt.Errorf("archiveTx: could not insert history. err: %v", err)
	}

	return nil
}

// AnalysisArchiveAndReset archives the current non-progressing live row to
// history (reason='reanalyze') then flips it back to progressing, atomically in
// ONE transaction (CAS on status!='progressing' — review H2). Returns the reset
// UPDATE's rowsAffected: 0 means another reanalyze already won the transition,
// in which case the whole txn (including the archive insert) is rolled back.
func (h *handler) AnalysisArchiveAndReset(ctx context.Context, id uuid.UUID) (int64, error) {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("AnalysisArchiveAndReset: could not begin txn. err: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if errArchive := h.archiveTx(ctx, tx, id, analysishistory.ReasonReanalyze); errArchive != nil {
		return 0, fmt.Errorf("AnalysisArchiveAndReset: could not archive. err: %v", errArchive)
	}

	ts := h.utilHandler.TimeNow()
	updateQuery := fmt.Sprintf(`
		UPDATE %s
		SET status = 'progressing', result = NULL, error = '', tm_update = ?
		WHERE id = ? AND tm_delete IS NULL AND status != 'progressing'
	`, analysisTable)

	result, err := tx.ExecContext(ctx, updateQuery, ts, id.Bytes())
	if err != nil {
		return 0, fmt.Errorf("AnalysisArchiveAndReset: could not execute reset. err: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("AnalysisArchiveAndReset: could not get rows affected. err: %v", err)
	}

	if n == 0 {
		// another reanalyze already flipped it to progressing; roll back the archive.
		return 0, nil
	}

	if errCommit := tx.Commit(); errCommit != nil {
		return 0, fmt.Errorf("AnalysisArchiveAndReset: could not commit. err: %v", errCommit)
	}

	return n, nil
}

// AnalysisUpdateResult writes the final result. Guards on status='progressing'
// AND tm_delete IS NULL so a row soft-deleted while the goroutine ran is NOT
// resurrected (review #8). Returns rowsAffected.
func (h *handler) AnalysisUpdateResult(ctx context.Context, id uuid.UUID, status analysis.Status, result []byte, model, errStr string) (int64, error) {
	ts := h.utilHandler.TimeNow()

	var resultJSON sql.NullString
	if len(result) > 0 {
		resultJSON = sql.NullString{String: string(result), Valid: true}
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET status = ?, result = ?, model = ?, error = ?, tm_update = ?
		WHERE id = ? AND tm_delete IS NULL AND status = 'progressing'
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

// AnalysisArchiveAndDelete archives the current live row to history
// (reason='delete') then soft-deletes it, ownership-checked, atomically in ONE
// transaction. Returns the deleted record (with tm_update/tm_delete set).
func (h *handler) AnalysisArchiveAndDelete(ctx context.Context, customerID, id uuid.UUID) (*analysis.Analysis, error) {
	// ownership check first (masked not-found) — read outside the txn is fine
	// because the delete UPDATE re-checks customer_id + tm_delete IS NULL.
	existing, err := h.AnalysisGet(ctx, customerID, id)
	if err != nil {
		return nil, err
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("AnalysisArchiveAndDelete: could not begin txn. err: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if errArchive := h.archiveTx(ctx, tx, id, analysishistory.ReasonDelete); errArchive != nil {
		return nil, fmt.Errorf("AnalysisArchiveAndDelete: could not archive. err: %v", errArchive)
	}

	ts := h.utilHandler.TimeNow()
	deleteQuery := fmt.Sprintf(`
		UPDATE %s
		SET tm_update = ?, tm_delete = ?
		WHERE id = ? AND customer_id = ? AND tm_delete IS NULL
	`, analysisTable)

	r, err := tx.ExecContext(ctx, deleteQuery, ts, ts, id.Bytes(), customerID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("AnalysisArchiveAndDelete: could not execute delete. err: %v", err)
	}
	n, err := r.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("AnalysisArchiveAndDelete: could not get rows affected. err: %v", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}

	if errCommit := tx.Commit(); errCommit != nil {
		return nil, fmt.Errorf("AnalysisArchiveAndDelete: could not commit. err: %v", errCommit)
	}

	existing.TMUpdate = ts
	existing.TMDelete = ts
	return existing, nil
}

// AnalysisHistoryList returns the append-only history for an activeflow, newest
// first, always filtered by customer_id (review C1).
func (h *handler) AnalysisHistoryList(ctx context.Context, customerID, activeflowID uuid.UUID, size uint64, token string) ([]*analysishistory.History, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(analysishistory.History{})

	query, args, err := sq.Select(cols...).
		From(analysisHistoryTable).
		Where(sq.And{
			sq.Eq{"activeflow_id": activeflowID.Bytes()},
			sq.Eq{"customer_id": customerID.Bytes()},
			sq.Lt{"tm_create": token},
		}).
		OrderBy("tm_create desc").
		Limit(size).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("AnalysisHistoryList: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AnalysisHistoryList: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*analysishistory.History{}
	for rows.Next() {
		hrow := &analysishistory.History{}
		if err := commondatabasehandler.ScanRow(rows, hrow); err != nil {
			return nil, fmt.Errorf("AnalysisHistoryList: could not scan. err: %v", err)
		}
		res = append(res, hrow)
	}

	return res, nil
}

// AnalysisCountProgressing returns the number of in-flight (progressing,
// non-deleted) analyses for a customer (design F1 per-customer cap).
func (h *handler) AnalysisCountProgressing(ctx context.Context, customerID uuid.UUID) (int64, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE customer_id = ? AND status = 'progressing' AND tm_delete IS NULL
	`, analysisTable)

	row := h.db.QueryRowContext(ctx, query, customerID.Bytes())

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("AnalysisCountProgressing: could not scan. err: %v", err)
	}

	return count, nil
}
