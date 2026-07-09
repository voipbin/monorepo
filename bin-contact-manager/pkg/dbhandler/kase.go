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

// mysqlDeadlockErrno is MySQL's errno for "Deadlock found when trying to
// get lock; try restarting transaction" (VOIP-1232).
const mysqlDeadlockErrno = 1213

// isMySQLDeadlock reports whether err is a MySQL deadlock (errno 1213).
// Mirrors the existing *mysql_driver.MySQLError type-assertion idiom used
// for the 1062/ErrDuplicate check below -- inlined per call site rather
// than a shared cross-package helper, matching this codebase's existing
// convention (see also interaction.go/address.go's own inline 1062
// checks). SQLite (the test harness driver) has no deadlock concept, so
// this is unconditionally false against SQLite errors -- consistent with
// forUpdateSuffix's SQLite-has-no-real-locking rationale in main.go.
func isMySQLDeadlock(err error) bool {
	me, ok := err.(*mysql_driver.MySQLError)
	return ok && me.Number == mysqlDeadlockErrno
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
		// VOIP-1232: concurrent INSERTs racing on the same uq_case_open_peer
		// value can surface as a deadlock (1213) instead of a clean 1062,
		// per InnoDB's insert-intention gap-lock interaction. The caller
		// (casehandler.GetOrCreate's outer retry loop) must discard the
		// whole transaction (already server-side rolled back) and restart
		// from a fresh BeginTx -- unlike ErrDuplicate, which is safely
		// retryable within the SAME tx via a locked re-select.
		if isMySQLDeadlock(err) {
			return ErrDeadlock
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
		// VOIP-1232: a FOR UPDATE select can itself be the victim of an
		// InnoDB deadlock under concurrent same-tuple GetOrCreate calls.
		if isMySQLDeadlock(err) {
			return nil, ErrDeadlock
		}
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
		// VOIP-1232: see CaseGetOpenByPeer's identical rationale.
		if isMySQLDeadlock(err) {
			return nil, ErrDeadlock
		}
		return nil, fmt.Errorf("could not query. CaseGetByIDForUpdate. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, nil
	}

	return caseGetFromRow(rows)
}

// caseUpdateStatusClosedExec is the shared implementation for
// CaseUpdateStatusClosed/CaseUpdateStatusClosedTx. The customer_id
// predicate is included in the same UPDATE statement as the mutation
// itself (not checked separately afterward) so a cross-tenant caller's
// UPDATE can never match a row at all -- there is no window where the
// mutation could commit before tenancy is verified.
func caseUpdateStatusClosedExec(exec sqlExecutor, customerID, id uuid.UUID, closedReason, closedByType string, closedByID *uuid.UUID, closedAt *time.Time) (bool, error) {
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
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"status": string(kase.StatusOpen)}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("could not build query. CaseUpdateStatusClosed. err: %v", err)
	}

	result, err := exec.Exec(query, args...)
	if err != nil {
		// VOIP-1232: the timeout-close UPDATE (design §4's close-and-reopen
		// branch) can also be a deadlock victim under concurrent same-tuple
		// GetOrCreate calls.
		if isMySQLDeadlock(err) {
			return false, ErrDeadlock
		}
		return false, fmt.Errorf("could not execute. CaseUpdateStatusClosed. err: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("could not get rows affected. CaseUpdateStatusClosed. err: %v", err)
	}

	return rows > 0, nil
}

// CaseUpdateStatusClosed transitions a Case to closed, guarded by
// WHERE customer_id=? AND status='open' (design §5.1's optimistic
// idempotent-double-close invariant, plus tenant isolation baked into
// the mutating statement itself). Returns (true, nil) if the close
// actually happened (1 row affected), (false, nil) if the Case was
// already closed OR belongs to a different tenant OR doesn't exist (0
// rows affected -- a no-op, not an error): callers must re-read the
// Case (itself tenant-scoped) to distinguish these cases and to
// discover the actually-persisted closed_reason/closed_by, not assume
// their own call's arguments won.
func (h *handler) CaseUpdateStatusClosed(ctx context.Context, customerID, id uuid.UUID, closedReason, closedByType string, closedByID *uuid.UUID, closedAt *time.Time) (bool, error) {
	return caseUpdateStatusClosedExec(h.db, customerID, id, closedReason, closedByType, closedByID, closedAt)
}

// CaseUpdateStatusClosedTx is CaseUpdateStatusClosed scoped to a
// caller-managed transaction (design §4's timeout-close branch).
func (h *handler) CaseUpdateStatusClosedTx(ctx context.Context, tx *sql.Tx, customerID, id uuid.UUID, closedReason, closedByType string, closedByID *uuid.UUID, closedAt *time.Time) (bool, error) {
	return caseUpdateStatusClosedExec(tx, customerID, id, closedReason, closedByType, closedByID, closedAt)
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
		// VOIP-1232: the final tm_update bump can itself deadlock under
		// concurrent same-tuple GetOrCreate calls.
		if isMySQLDeadlock(err) {
			return ErrDeadlock
		}
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
// CaseUpdateContactID/CaseUpdateContactIDTx. Scoped by customer_id as a
// defense-in-depth measure -- every caller today (ResolutionCreateCase
// Level/ResolutionDeleteCaseLevel/ReconcileContact) already verifies
// case ownership via verifyCaseOwnership before calling this, but the
// round-2 review's finding (multiple case-scoped handler methods
// accepted customerID without using it) makes an unscoped mutation
// primitive itself a latent risk for any future caller that forgets the
// upstream check.
func caseUpdateContactIDExec(exec sqlExecutor, customerID, id, contactID uuid.UUID) error {
	query, args, err := sq.Update(caseTable).
		Set("contact_id", contactID.Bytes()).
		Where(sq.Eq{"id": id.Bytes()}).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
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
func (h *handler) CaseUpdateContactID(ctx context.Context, customerID, id, contactID uuid.UUID) error {
	return caseUpdateContactIDExec(h.db, customerID, id, contactID)
}

// caseClearContactIDExec is the shared implementation for
// CaseClearContactIDTx: reverts Case.contact_id to NULL, used when
// deriveCaseContactID (design §3.4) finds no active case-level positive
// Resolution left (e.g. the sole one was just soft-deleted). Scoped by
// customer_id as defense-in-depth (see caseUpdateContactIDExec's
// comment).
func caseClearContactIDExec(exec sqlExecutor, customerID, id uuid.UUID) error {
	query, args, err := sq.Update(caseTable).
		Set("contact_id", nil).
		Where(sq.Eq{"id": id.Bytes()}).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseClearContactID. err: %v", err)
	}

	if _, err := exec.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. CaseClearContactID. err: %v", err)
	}

	return nil
}

// CaseClearContactIDTx reverts a Case's denormalized contact_id cache to
// NULL, scoped to a caller-managed transaction.
func (h *handler) CaseClearContactIDTx(ctx context.Context, tx *sql.Tx, customerID, id uuid.UUID) error {
	return caseClearContactIDExec(tx, customerID, id)
}

// CaseUpdateContactIDTx is CaseUpdateContactID scoped to a caller-managed
// transaction (design §4 step 2's contact auto-match write).
func (h *handler) CaseUpdateContactIDTx(ctx context.Context, tx *sql.Tx, customerID, id, contactID uuid.UUID) error {
	return caseUpdateContactIDExec(tx, customerID, id, contactID)
}

// CaseListAll returns every Case across all tenants. CLI-only usage
// (case-control's `--all` reconcile-contact sweep) -- never exposed via
// a customer-facing RPC/route.
func (h *handler) CaseListAll(ctx context.Context) ([]*kase.Case, error) {
	columns := commondatabasehandler.GetDBFields(&kase.Case{})

	query, args, err := sq.Select(columns...).
		From(caseTable).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CaseListAll. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseListAll. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*kase.Case
	for rows.Next() {
		item, scanErr := caseGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. CaseListAll. err: %v", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. CaseListAll. err: %v", err)
	}

	return res, nil
}

// CaseListUnresolved returns OPEN Cases with contact_id IS NULL, scoped
// to customerID (design §6; backed by idx_case_unresolved). A closed
// case is never in this queue regardless of contact_id -- closing IS the
// "no further action needed" signal per §6, independent of whether the
// case was ever resolved to a contact.
func (h *handler) CaseListUnresolved(ctx context.Context, customerID uuid.UUID) ([]*kase.Case, error) {
	columns := commondatabasehandler.GetDBFields(&kase.Case{})

	query, args, err := sq.Select(columns...).
		From(caseTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"status": kase.StatusOpen}).
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

// CaseList returns Cases scoped to customerID, optionally filtered by
// status, owner, and/or contact_id (design §9's GET /v1/cases?...
// list surface). status == "" means no status filter; ownerType == ""
// or ownerID == uuid.Nil means no owner filter; contactID == uuid.Nil
// means no contact filter.
func (h *handler) CaseList(ctx context.Context, customerID uuid.UUID, status string, ownerType commonidentity.OwnerType, ownerID uuid.UUID, contactID uuid.UUID) ([]*kase.Case, error) {
	columns := commondatabasehandler.GetDBFields(&kase.Case{})

	builder := sq.Select(columns...).
		From(caseTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()})
	if status != "" {
		builder = builder.Where(sq.Eq{"status": status})
	}
	if ownerType != "" && ownerID != uuid.Nil {
		builder = builder.
			Where(sq.Eq{"owner_type": string(ownerType)}).
			Where(sq.Eq{"owner_id": ownerID.Bytes()})
	}
	if contactID != uuid.Nil {
		builder = builder.Where(sq.Eq{"contact_id": contactID.Bytes()})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CaseList. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseList. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var res []*kase.Case
	for rows.Next() {
		item, scanErr := caseGetFromRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("could not scan the row. CaseList. err: %v", scanErr)
		}
		res = append(res, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. CaseList. err: %v", err)
	}

	return res, nil
}
