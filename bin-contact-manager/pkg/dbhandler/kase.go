package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	mysql_driver "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-contact-manager/models/kase"
)

const caseTable = "contact_cases"

// sqlExecutor is the common subset of *sql.DB and *sql.Tx used by the
// case* functions below, letting the same query-building logic run either
// standalone (h.db) or inside a caller-managed transaction (a *sql.Tx from
// BeginTx) -- needed because the design §4 get-or-create algorithm must
// perform several of these operations atomically within one transaction.
type sqlExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

// BeginTx starts a new transaction for callers (casehandler's get-or-create,
// design §4) that need to run several Case/Interaction operations
// atomically.
func (h *handler) BeginTx(ctx context.Context) (*sql.Tx, error) {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction. BeginTx. err: %v", err)
	}
	return tx, nil
}

// caseGetFromRow scans a single row into a Case struct.
func caseGetFromRow(rows *sql.Rows) (*kase.Case, error) {
	res := &kase.Case{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. caseGetFromRow. err: %v", err)
	}
	return res, nil
}

// caseInsertExec is the shared implementation for CaseInsert/CaseInsertTx.
func caseInsertExec(exec sqlExecutor, c *kase.Case) error {
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. CaseInsert. err: %v", err)
	}

	query, args, err := sq.Insert(caseTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseInsert. err: %v", err)
	}

	if _, err := exec.Exec(query, args...); err != nil {
		// Detect a unique-constraint violation on uq_case_open_peer
		// specifically (open_peer_uk), same detection idiom as
		// AddressCreate's ErrDuplicateTarget classification: MySQL names
		// the violated key, SQLite names the violated column.
		if me, ok := err.(*mysql_driver.MySQLError); ok && me.Number == 1062 {
			if strings.Contains(me.Message, "uq_case_open_peer") {
				return ErrDuplicate
			}
			return fmt.Errorf("could not execute. CaseInsert. err: %v", err)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") &&
			strings.Contains(err.Error(), "open_peer_uk") {
			return ErrDuplicate
		}
		return fmt.Errorf("could not execute. CaseInsert. err: %v", err)
	}

	return nil
}

// CaseInsert inserts a new Case row into contact_cases. Returns ErrDuplicate
// if the insert violates uq_case_open_peer (an open case already exists
// for this customer/peer/reference_type -- design §3.1/§4).
func (h *handler) CaseInsert(ctx context.Context, c *kase.Case) error {
	return caseInsertExec(h.db, c)
}

// CaseInsertTx is CaseInsert scoped to a caller-managed transaction (design
// §4's get-or-create insert branches, run atomically with the rest of the
// transaction).
func (h *handler) CaseInsertTx(ctx context.Context, tx *sql.Tx, c *kase.Case) error {
	return caseInsertExec(tx, c)
}

// caseGetByIDExec is the shared implementation for CaseGetByID/CaseGetByIDTx.
func caseGetByIDExec(exec sqlExecutor, id uuid.UUID) (*kase.Case, error) {
	columns := commondatabasehandler.GetDBFields(&kase.Case{})

	query, args, err := sq.Select(columns...).
		From(caseTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CaseGetByID. err: %v", err)
	}

	rows, err := exec.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseGetByID. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	return caseGetFromRow(rows)
}

// CaseGetByID returns a Case by id, unlocked (for read APIs).
func (h *handler) CaseGetByID(ctx context.Context, id uuid.UUID) (*kase.Case, error) {
	return caseGetByIDExec(h.db, id)
}

// CaseGetByIDTx is CaseGetByID scoped to a caller-managed transaction.
func (h *handler) CaseGetByIDTx(ctx context.Context, tx *sql.Tx, id uuid.UUID) (*kase.Case, error) {
	return caseGetByIDExec(tx, id)
}

// CaseGetOpenByPeer returns the currently-OPEN Case for a given
// (customer_id, peer_type, peer_target, reference_type), locked FOR UPDATE
// within the given transaction. Returns (nil, nil) if no open case exists
// -- this is the design §4 step 1 locked-select primitive, used both for
// the plain reuse-if-open path and the retry-on-conflict re-select.
func (h *handler) CaseGetOpenByPeer(ctx context.Context, tx *sql.Tx, customerID uuid.UUID, peerType commonaddress.Type, peerTarget, referenceType string) (*kase.Case, error) {
	columns := commondatabasehandler.GetDBFields(&kase.Case{})

	builder := sq.Select(columns...).
		From(caseTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"peer_type": string(peerType)}).
		Where(sq.Eq{"peer_target": peerTarget}).
		Where(sq.Eq{"reference_type": referenceType}).
		Where(sq.Eq{"status": string(kase.StatusOpen)})
	if h.forUpdateSuffix != "" {
		builder = builder.Suffix(h.forUpdateSuffix)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CaseGetOpenByPeer. err: %v", err)
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseGetOpenByPeer. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, nil
	}

	return caseGetFromRow(rows)
}

// CaseGetByIDForUpdate returns a Case by id, locked FOR UPDATE within the
// given transaction, scoped to customerID (tenant guard). Used by design
// §4 step 1's explicit case_id hint validation: SELECT ... WHERE id=? AND
// customer_id=? AND status='open' FOR UPDATE. Returns (nil, nil) if not
// found / wrong tenant / not open -- an invalid hint is never an error,
// only a signal to fall through to the peer/reference_type path.
func (h *handler) CaseGetByIDForUpdate(ctx context.Context, tx *sql.Tx, customerID, id uuid.UUID) (*kase.Case, error) {
	columns := commondatabasehandler.GetDBFields(&kase.Case{})

	builder := sq.Select(columns...).
		From(caseTable).
		Where(sq.Eq{"id": id.Bytes()}).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"status": string(kase.StatusOpen)})
	if h.forUpdateSuffix != "" {
		builder = builder.Suffix(h.forUpdateSuffix)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CaseGetByIDForUpdate. err: %v", err)
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseGetByIDForUpdate. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, nil
	}

	return caseGetFromRow(rows)
}

// caseUpdateStatusClosedExec is the shared implementation for
// CaseUpdateStatusClosed/CaseUpdateStatusClosedTx.
func caseUpdateStatusClosedExec(exec sqlExecutor, id uuid.UUID, closedReason, closedByType string, closedByID *uuid.UUID, closedAt *time.Time) (bool, error) {
	setMap := sq.Eq{
		"status":         string(kase.StatusClosed),
		"closed_reason":  closedReason,
		"closed_by_type": closedByType,
		"closed_at":      closedAt,
	}
	if closedByID != nil {
		setMap["closed_by_id"] = closedByID.Bytes()
	}

	query, args, err := sq.Update(caseTable).
		SetMap(setMap).
		Where(sq.Eq{"id": id.Bytes()}).
		Where(sq.Eq{"status": string(kase.StatusOpen)}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("could not build query. CaseUpdateStatusClosed. err: %v", err)
	}

	result, err := exec.Exec(query, args...)
	if err != nil {
		return false, fmt.Errorf("could not execute. CaseUpdateStatusClosed. err: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("could not get rows affected. CaseUpdateStatusClosed. err: %v", err)
	}

	return rows > 0, nil
}

// CaseUpdateStatusClosed transitions a Case to closed, guarded by
// WHERE status='open' (design §5.1's optimistic idempotent-double-close
// invariant). Returns (true, nil) if the close actually happened (1 row
// affected), (false, nil) if the Case was already closed (0 rows
// affected -- a no-op, not an error): callers must re-read the Case to
// discover the actually-persisted closed_reason/closed_by, not assume
// their own call's arguments won.
func (h *handler) CaseUpdateStatusClosed(ctx context.Context, id uuid.UUID, closedReason, closedByType string, closedByID *uuid.UUID, closedAt *time.Time) (bool, error) {
	return caseUpdateStatusClosedExec(h.db, id, closedReason, closedByType, closedByID, closedAt)
}

// CaseUpdateStatusClosedTx is CaseUpdateStatusClosed scoped to a
// caller-managed transaction (design §4's timeout-close branch).
func (h *handler) CaseUpdateStatusClosedTx(ctx context.Context, tx *sql.Tx, id uuid.UUID, closedReason, closedByType string, closedByID *uuid.UUID, closedAt *time.Time) (bool, error) {
	return caseUpdateStatusClosedExec(tx, id, closedReason, closedByType, closedByID, closedAt)
}

// caseUpdateTMUpdateExec is the shared implementation for
// CaseUpdateTMUpdate/CaseUpdateTMUpdateTx.
func caseUpdateTMUpdateExec(exec sqlExecutor, id uuid.UUID, tmUpdate *time.Time) error {
	query, args, err := sq.Update(caseTable).
		Set("tm_update", tmUpdate).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseUpdateTMUpdate. err: %v", err)
	}

	if _, err := exec.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. CaseUpdateTMUpdate. err: %v", err)
	}

	return nil
}

// CaseUpdateTMUpdate bumps a Case's tm_update, used at the end of the
// get-or-create transaction (design §4 step 4).
func (h *handler) CaseUpdateTMUpdate(ctx context.Context, id uuid.UUID, tmUpdate *time.Time) error {
	return caseUpdateTMUpdateExec(h.db, id, tmUpdate)
}

// CaseUpdateTMUpdateTx is CaseUpdateTMUpdate scoped to a caller-managed
// transaction.
func (h *handler) CaseUpdateTMUpdateTx(ctx context.Context, tx *sql.Tx, id uuid.UUID, tmUpdate *time.Time) error {
	return caseUpdateTMUpdateExec(tx, id, tmUpdate)
}

// caseUpdateContactIDExec is the shared implementation for
// CaseUpdateContactID/CaseUpdateContactIDTx.
func caseUpdateContactIDExec(exec sqlExecutor, id, contactID uuid.UUID) error {
	query, args, err := sq.Update(caseTable).
		Set("contact_id", contactID.Bytes()).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseUpdateContactID. err: %v", err)
	}

	if _, err := exec.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. CaseUpdateContactID. err: %v", err)
	}

	return nil
}

// CaseUpdateContactID updates a Case's denormalized contact_id cache
// (design §3.4; single source of truth is Resolution).
func (h *handler) CaseUpdateContactID(ctx context.Context, id, contactID uuid.UUID) error {
	return caseUpdateContactIDExec(h.db, id, contactID)
}

// CaseUpdateContactIDTx is CaseUpdateContactID scoped to a caller-managed
// transaction (design §4 step 2's contact auto-match write).
func (h *handler) CaseUpdateContactIDTx(ctx context.Context, tx *sql.Tx, id, contactID uuid.UUID) error {
	return caseUpdateContactIDExec(tx, id, contactID)
}

// CaseListUnresolved returns Cases with contact_id IS NULL, scoped to
// customerID (design §6; backed by idx_case_unresolved).
func (h *handler) CaseListUnresolved(ctx context.Context, customerID uuid.UUID) ([]*kase.Case, error) {
	columns := commondatabasehandler.GetDBFields(&kase.Case{})

	query, args, err := sq.Select(columns...).
		From(caseTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"contact_id": nil}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CaseListUnresolved. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseListUnresolved. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*kase.Case
	for rows.Next() {
		item, scanErr := caseGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. CaseListUnresolved. err: %v", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. CaseListUnresolved. err: %v", err)
	}

	return res, nil
}

// CaseListByOwner returns Cases owned by the given owner, scoped to
// customerID (design §7 "my cases" list; backed by idx_case_owner).
func (h *handler) CaseListByOwner(ctx context.Context, customerID uuid.UUID, ownerType commonidentity.OwnerType, ownerID uuid.UUID) ([]*kase.Case, error) {
	columns := commondatabasehandler.GetDBFields(&kase.Case{})

	query, args, err := sq.Select(columns...).
		From(caseTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"owner_type": string(ownerType)}).
		Where(sq.Eq{"owner_id": ownerID.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CaseListByOwner. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseListByOwner. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*kase.Case
	for rows.Next() {
		item, scanErr := caseGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. CaseListByOwner. err: %v", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. CaseListByOwner. err: %v", err)
	}

	return res, nil
}

// caseGetLastClosedByPeerExec is the shared implementation for
// CaseGetLastClosedByPeer/CaseGetLastClosedByPeerTx.
func caseGetLastClosedByPeerExec(exec sqlExecutor, customerID uuid.UUID, peerType commonaddress.Type, peerTarget, referenceType string) (*kase.Case, error) {
	columns := commondatabasehandler.GetDBFields(&kase.Case{})

	query, args, err := sq.Select(columns...).
		From(caseTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"peer_type": string(peerType)}).
		Where(sq.Eq{"peer_target": peerTarget}).
		Where(sq.Eq{"reference_type": referenceType}).
		Where(sq.Eq{"status": string(kase.StatusClosed)}).
		OrderBy("closed_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CaseGetLastClosedByPeer. err: %v", err)
	}

	rows, err := exec.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseGetLastClosedByPeer. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, nil
	}

	return caseGetFromRow(rows)
}

// CaseGetLastClosedByPeer returns the most recently closed Case for a
// given (customer_id, peer_type, peer_target, reference_type), or
// (nil, nil) if none exists. Used for previous_case_id chaining on the
// fresh-insert path (design §4).
func (h *handler) CaseGetLastClosedByPeer(ctx context.Context, customerID uuid.UUID, peerType commonaddress.Type, peerTarget, referenceType string) (*kase.Case, error) {
	return caseGetLastClosedByPeerExec(h.db, customerID, peerType, peerTarget, referenceType)
}

// CaseGetLastClosedByPeerTx is CaseGetLastClosedByPeer scoped to a
// caller-managed transaction (design §4's fresh-insert previous_case_id
// chaining lookup).
func (h *handler) CaseGetLastClosedByPeerTx(ctx context.Context, tx *sql.Tx, customerID uuid.UUID, peerType commonaddress.Type, peerTarget, referenceType string) (*kase.Case, error) {
	return caseGetLastClosedByPeerExec(tx, customerID, peerType, peerTarget, referenceType)
}
