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

// caseGetFromRow scans a single row into a Case struct.
func caseGetFromRow(rows *sql.Rows) (*kase.Case, error) {
	res := &kase.Case{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. caseGetFromRow. err: %v", err)
	}
	return res, nil
}

// CaseInsert inserts a new Case row into contact_cases. Returns ErrDuplicate
// if the insert violates uq_case_open_peer (an open case already exists
// for this customer/peer/reference_type -- design §3.1/§4).
func (h *handler) CaseInsert(ctx context.Context, c *kase.Case) error {
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. CaseInsert. err: %v", err)
	}

	query, args, err := sq.Insert(caseTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseInsert. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
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

// CaseGetByID returns a Case by id, unlocked (for read APIs).
func (h *handler) CaseGetByID(ctx context.Context, id uuid.UUID) (*kase.Case, error) {
	columns := commondatabasehandler.GetDBFields(&kase.Case{})

	query, args, err := sq.Select(columns...).
		From(caseTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CaseGetByID. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseGetByID. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	return caseGetFromRow(rows)
}

// CaseGetOpenByPeer returns the currently-OPEN Case for a given
// (customer_id, peer_type, peer_target, reference_type), locked FOR UPDATE
// within the given transaction. Returns (nil, nil) if no open case exists
// -- this is the design §4 step 1 locked-select primitive, used both for
// the plain reuse-if-open path and the retry-on-conflict re-select.
func (h *handler) CaseGetOpenByPeer(ctx context.Context, tx *sql.Tx, customerID uuid.UUID, peerType commonaddress.Type, peerTarget, referenceType string) (*kase.Case, error) {
	columns := commondatabasehandler.GetDBFields(&kase.Case{})

	query, args, err := sq.Select(columns...).
		From(caseTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"peer_type": string(peerType)}).
		Where(sq.Eq{"peer_target": peerTarget}).
		Where(sq.Eq{"reference_type": referenceType}).
		Where(sq.Eq{"status": string(kase.StatusOpen)}).
		Suffix("FOR UPDATE").
		ToSql()
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

// CaseUpdateStatusClosed transitions a Case to closed, guarded by
// WHERE status='open' (design §5.1's optimistic idempotent-double-close
// invariant). Returns (true, nil) if the close actually happened (1 row
// affected), (false, nil) if the Case was already closed (0 rows
// affected -- a no-op, not an error): callers must re-read the Case to
// discover the actually-persisted closed_reason/closed_by, not assume
// their own call's arguments won.
func (h *handler) CaseUpdateStatusClosed(ctx context.Context, id uuid.UUID, closedReason, closedByType string, closedByID *uuid.UUID, closedAt *time.Time) (bool, error) {
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

	result, err := h.db.Exec(query, args...)
	if err != nil {
		return false, fmt.Errorf("could not execute. CaseUpdateStatusClosed. err: %v", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("could not get rows affected. CaseUpdateStatusClosed. err: %v", err)
	}

	return rows > 0, nil
}

// CaseUpdateTMUpdate bumps a Case's tm_update, used at the end of the
// get-or-create transaction (design §4 step 4).
func (h *handler) CaseUpdateTMUpdate(ctx context.Context, id uuid.UUID, tmUpdate *time.Time) error {
	query, args, err := sq.Update(caseTable).
		Set("tm_update", tmUpdate).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseUpdateTMUpdate. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. CaseUpdateTMUpdate. err: %v", err)
	}

	return nil
}

// CaseUpdateContactID updates a Case's denormalized contact_id cache
// (design §3.4; single source of truth is Resolution).
func (h *handler) CaseUpdateContactID(ctx context.Context, id, contactID uuid.UUID) error {
	query, args, err := sq.Update(caseTable).
		Set("contact_id", contactID.Bytes()).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CaseUpdateContactID. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. CaseUpdateContactID. err: %v", err)
	}

	return nil
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

// CaseGetLastClosedByPeer returns the most recently closed Case for a
// given (customer_id, peer_type, peer_target, reference_type), or
// (nil, nil) if none exists. Used for previous_case_id chaining on the
// fresh-insert path (design §4).
func (h *handler) CaseGetLastClosedByPeer(ctx context.Context, customerID uuid.UUID, peerType commonaddress.Type, peerTarget, referenceType string) (*kase.Case, error) {
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

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CaseGetLastClosedByPeer. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, nil
	}

	return caseGetFromRow(rows)
}
