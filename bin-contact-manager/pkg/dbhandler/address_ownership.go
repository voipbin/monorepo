package dbhandler

// Contact-address ownership period write path (Phase 1 of
// NOJIRA-contact-address-ownership-periods). See
// docs/plans/2026-07-11-contact-address-ownership-integrity-design.md
// §4/§5.1-5.4 for the full design and rationale.
//
// This file implements the shared locking-read/resolve primitive
// (OwnershipPeriodsLockAndResolveTx, §4's Step 1-5 decision procedure)
// and the Tx-suffixed write functions that call it. The existing
// non-Tx methods in address.go are thin BeginTx/commit/retry wrappers
// around these.

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
)

const (
	// ownershipPeriodTable is the new table introduced by migration
	// 2d8f0ea90565 (design §3.1/§9).
	ownershipPeriodTable = "contact_address_ownership_periods"
)

// Step values returned by OwnershipPeriodsLockAndResolveTx, naming §4's
// Step 2-5 outcomes for OPEN-ing callers. Step 1 never returns a step
// value to the caller -- a live conflict returns ErrConflict directly,
// and an orphan close falls through to be evaluated as Step 2 (i.e. the
// step value callers see is always one of the constants below).
const (
	// StepOpenReuse is §4 Step 2: this contact already owns an open
	// period for this target. No period write needed -- reuse as-is.
	StepOpenReuse = 2
	// StepReopen is §4 Step 3's "no intervening owner" branch: reopen
	// this contact's own most-recently-closed period.
	StepReopen = 3
	// StepInsertAfterIntervening is §4 Step 3's "intervening owner"
	// branch (the A->B->A case): do not reopen, INSERT a new period
	// instead.
	StepInsertAfterIntervening = 4
	// StepReassign is §4 Step 4: a different contact's closed row
	// exists for this target (or none of this contact's own). INSERT a
	// new period with a caller-specific valid_from.
	StepReassign = 5
	// StepFirstRegistration is §4 Step 5: no period row exists at all
	// for this target. INSERT with valid_from=NULL, valid_to=NULL.
	StepFirstRegistration = 6
)

// ErrStaleTarget is returned by the thin non-Tx wrappers' compare-and-retry
// when a pre-lock read of an address row's (type, target, contact_id) no
// longer matches a fresh post-lock re-read -- design §4's round-39/40
// stale-target hazard. Treated as retry-eligible, capped by the same
// maxDeadlockRetries budget as a genuine deadlock (design §5.3).
var ErrStaleTarget = fmt.Errorf("target changed between pre-lock read and lock acquisition")

// maxDeadlockRetries bounds the outer retry loop that restarts an entire
// address-write transaction (fresh BeginTx) on ErrDeadlock or
// ErrStaleTarget. Mirrors casehandler.getOrCreateAttempt's identical
// constant (design §5.3: "reuse casehandler's exact pattern verbatim").
const addressMaxDeadlockRetries = 3

// OwnershipPeriod mirrors a single contact_address_ownership_periods row.
type OwnershipPeriod struct {
	ID         uuid.UUID  `db:"id,uuid"`
	CustomerID uuid.UUID  `db:"customer_id,uuid"`
	ContactID  uuid.UUID  `db:"contact_id,uuid"`
	Type       string     `db:"type"`
	Target     string     `db:"target"`
	ValidFrom  *time.Time `db:"valid_from"`
	ValidTo    *time.Time `db:"valid_to"`
}

// ownershipPeriodRowColumns is the column subset the §4 locking read
// fetches -- id, contact_id, valid_from, valid_to (design §4: "SELECT id,
// contact_id, valid_from, valid_to FROM contact_address_ownership_periods
// WHERE ... FOR UPDATE").
func ownershipPeriodRowColumns() []string {
	return []string{"id", "contact_id", "valid_from", "valid_to"}
}

func scanOwnershipPeriodRow(rows *sql.Rows, customerID uuid.UUID, addrType commonaddress.Type, target string) (*OwnershipPeriod, error) {
	var idBytes, contactIDBytes []byte
	var validFrom, validTo sql.NullString
	if err := rows.Scan(&idBytes, &contactIDBytes, &validFrom, &validTo); err != nil {
		return nil, fmt.Errorf("could not scan the row. scanOwnershipPeriodRow. err: %v", err)
	}
	id, err := uuid.FromBytes(idBytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse id. scanOwnershipPeriodRow. err: %v", err)
	}
	contactID, err := uuid.FromBytes(contactIDBytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse contact_id. scanOwnershipPeriodRow. err: %v", err)
	}
	validFromPtr, err := parseNullableDBTime(validFrom)
	if err != nil {
		return nil, fmt.Errorf("could not parse valid_from. scanOwnershipPeriodRow. err: %v", err)
	}
	validToPtr, err := parseNullableDBTime(validTo)
	if err != nil {
		return nil, fmt.Errorf("could not parse valid_to. scanOwnershipPeriodRow. err: %v", err)
	}
	return &OwnershipPeriod{
		ID:         id,
		CustomerID: customerID,
		ContactID:  contactID,
		Type:       string(addrType),
		Target:     target,
		ValidFrom:  validFromPtr,
		ValidTo:    validToPtr,
	}, nil
}

// parseNullableDBTime parses a nullable DATETIME column (MySQL or SQLite
// text-affinity formats) into *time.Time, mirroring
// commondatabasehandler's ScanRow *time.Time handling for db-tagged
// structs -- this file scans a raw column subset via sq.Select, not a
// full struct, so it needs the same parsing logic applied manually.
func parseNullableDBTime(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid || ns.String == "" {
		return nil, nil
	}
	layouts := []string{
		"2006-01-02 15:04:05.000000",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02T15:04:05.000000Z",
		"2006-01-02T15:04:05Z",
		time.RFC3339Nano,
		time.RFC3339,
	}
	var parsed time.Time
	var err error
	for _, layout := range layouts {
		parsed, err = time.Parse(layout, ns.String)
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("could not parse time %q: %v", ns.String, err)
}

// sqOwnershipPeriodSelectAll builds a plain (non-locking) SELECT for all
// rows -- open and closed -- matching (customer_id, type, target). Used
// by tests to inspect period-table state without going through the
// locking-read/resolve primitive (which requires a contactID and
// performs Step 1-5 write-path side effects, not a pure read).
func sqOwnershipPeriodSelectAll(customerID uuid.UUID, addrType commonaddress.Type, target string) (string, []interface{}, error) {
	return sq.Select(ownershipPeriodRowColumns()...).
		From(ownershipPeriodTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"type": string(addrType)}).
		Where(sq.Eq{"target": target}).
		ToSql()
}

// OwnershipPeriodsLockAndResolveTx is the shared locking-read/resolve
// primitive design §4/§5.1 (rounds 43-51) define. It runs §4's Step 1-5
// decision procedure against ONE already-locked row set:
//
//	SELECT id, contact_id, valid_from, valid_to
//	FROM contact_address_ownership_periods
//	WHERE customer_id = ? AND type = ? AND target = ?
//	FOR UPDATE
//
// For OPEN-ing callers (AddressCreateTx, AddressClaimTx,
// AddressUpdateTx's new-target side) the returned step tells the caller
// whether to reuse an already-open period (StepOpenReuse), reopen a
// closed one (StepReopen), or INSERT a new one (the remaining steps).
// For CLOSE-ing callers (AddressDeleteTx, AddressUpdateTx's old-target
// side, AddressDeleteCompensating, the ContactDelete family) step is
// meaningless and ignored -- they only need lockedRows, to find and
// close their own open row directly (design §5.1 round-48).
func (h *handler) OwnershipPeriodsLockAndResolveTx(ctx context.Context, tx *sql.Tx, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string) (int, []OwnershipPeriod, error) {
	builder := sq.Select(ownershipPeriodRowColumns()...).
		From(ownershipPeriodTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"type": string(addrType)}).
		Where(sq.Eq{"target": target})
	if h.forUpdateSuffix != "" {
		builder = builder.Suffix(h.forUpdateSuffix)
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return 0, nil, fmt.Errorf("could not build query. OwnershipPeriodsLockAndResolveTx. err: %v", err)
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		if isMySQLDeadlock(err) {
			return 0, nil, ErrDeadlock
		}
		return 0, nil, fmt.Errorf("could not query. OwnershipPeriodsLockAndResolveTx. err: %v", err)
	}

	var lockedRows []OwnershipPeriod
	for rows.Next() {
		p, err := scanOwnershipPeriodRow(rows, customerID, addrType, target)
		if err != nil {
			_ = rows.Close()
			return 0, nil, err
		}
		lockedRows = append(lockedRows, *p)
	}
	rowsErr := rows.Err()
	_ = rows.Close()
	if rowsErr != nil {
		return 0, nil, fmt.Errorf("row iteration error. OwnershipPeriodsLockAndResolveTx. err: %v", rowsErr)
	}

	// Step 1: is there an open row for a DIFFERENT contact_id?
	var otherOpen *OwnershipPeriod
	for i := range lockedRows {
		if lockedRows[i].ValidTo == nil && lockedRows[i].ContactID != contactID {
			otherOpen = &lockedRows[i]
			break
		}
	}
	if otherOpen != nil {
		live, err := h.ownershipAgreementHoldsTx(ctx, tx, customerID, otherOpen.ContactID, addrType, target)
		if err != nil {
			return 0, nil, err
		}
		if live {
			return 0, nil, ErrConflict
		}
		// Orphan: close it and continue evaluating as if it were
		// already closed (design §4 Step 1's orphan-close branch;
		// repair-in-place logic for the vacated contact_addresses
		// row is out of Phase 1 scope).
		if err := h.ownershipPeriodCloseByIDTx(ctx, tx, otherOpen.ID); err != nil {
			return 0, nil, err
		}
		now := h.utilHandler.TimeNow()
		otherOpen.ValidTo = now
	}

	// Step 2: is there an open row for THIS contact_id? (Step 1 already
	// ruled out any other contact's open row reaching here live, and the
	// in-memory close above means we must skip the closed-in-Step-1 row
	// when looking.)
	for i := range lockedRows {
		if lockedRows[i].ContactID == contactID && lockedRows[i].ValidTo == nil {
			return StepOpenReuse, lockedRows, nil
		}
	}

	// Step 3: does this contact_id have closed row(s) of its own?
	var latestOwn *OwnershipPeriod
	for i := range lockedRows {
		p := &lockedRows[i]
		if p.ContactID != contactID {
			continue
		}
		if p == otherOpen {
			continue // the row we just closed above belongs to a DIFFERENT contact by definition
		}
		if p.ValidTo == nil {
			continue // already handled by Step 2 (shouldn't reach here, defensive)
		}
		if latestOwn == nil || (p.ValidTo != nil && latestOwn.ValidTo != nil && p.ValidTo.After(*latestOwn.ValidTo)) {
			latestOwn = p
		}
	}
	if latestOwn != nil {
		intervened := false
		for i := range lockedRows {
			p := &lockedRows[i]
			if p.ContactID == contactID {
				continue
			}
			if p.ValidFrom == nil {
				intervened = true
				break
			}
			if !p.ValidFrom.Before(*latestOwn.ValidTo) { // >= per round-19
				intervened = true
				break
			}
		}
		if !intervened {
			return StepReopen, lockedRows, nil
		}
		return StepInsertAfterIntervening, lockedRows, nil
	}

	// Step 4: does ANY row exist for this target (necessarily another
	// contact_id's closed row, since steps 1-3 already ruled out
	// everything else)?
	if len(lockedRows) > 0 {
		return StepReassign, lockedRows, nil
	}

	// Step 5: nothing matched. First-ever registration.
	return StepFirstRegistration, lockedRows, nil
}

// ownershipAgreementHoldsTx implements §4 Step 1's ownership-agreement
// check: the contact_addresses row for this (customer_id, type, target)
// must exist, must carry blockingContactID, AND that Contact must not be
// tombstoned (contact_contacts.tm_delete IS NULL). All three conditions
// must hold for the blocking open period to represent a genuine live
// conflict; otherwise it is an orphan.
func (h *handler) ownershipAgreementHoldsTx(ctx context.Context, tx *sql.Tx, customerID, blockingContactID uuid.UUID, addrType commonaddress.Type, target string) (bool, error) {
	query, args, err := sq.Select("a.contact_id", "c.tm_delete").
		From(addressTable + " a").
		Join(contactTable + " c ON c.id = a.contact_id").
		Where(sq.Eq{"a.customer_id": customerID.Bytes()}).
		Where(sq.Eq{"a.type": string(addrType)}).
		Where(sq.Eq{"a.target": target}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("could not build query. ownershipAgreementHoldsTx. err: %v", err)
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		if isMySQLDeadlock(err) {
			return false, ErrDeadlock
		}
		return false, fmt.Errorf("could not query. ownershipAgreementHoldsTx. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return false, nil // row gone -- orphan
	}
	var contactIDBytes []byte
	var tmDelete sql.NullString
	if err := rows.Scan(&contactIDBytes, &tmDelete); err != nil {
		return false, fmt.Errorf("could not scan. ownershipAgreementHoldsTx. err: %v", err)
	}
	rowContactID, err := uuid.FromBytes(contactIDBytes)
	if err != nil {
		return false, fmt.Errorf("could not parse contact_id. ownershipAgreementHoldsTx. err: %v", err)
	}
	if rowContactID != blockingContactID {
		return false, nil // owned by someone else -- orphan
	}
	if tmDelete.Valid && tmDelete.String != "" {
		return false, nil // tombstoned -- orphan
	}
	return true, nil
}

// ownershipPeriodCloseByIDTx closes one period row (valid_to = NOW()) by
// its primary key, inside tx.
func (h *handler) ownershipPeriodCloseByIDTx(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	now := h.utilHandler.TimeNow()
	query, args, err := sq.Update(ownershipPeriodTable).
		Set("valid_to", now).
		Set("tm_update", now).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ownershipPeriodCloseByIDTx. err: %v", err)
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		if isMySQLDeadlock(err) {
			return ErrDeadlock
		}
		return fmt.Errorf("could not execute. ownershipPeriodCloseByIDTx. err: %v", err)
	}
	return nil
}

// ownershipPeriodReopenByIDTx reopens one closed period row (valid_to =
// NULL) by its primary key -- design §4 Step 3's "no intervening owner"
// branch.
func (h *handler) ownershipPeriodReopenByIDTx(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	now := h.utilHandler.TimeNow()
	query, args, err := sq.Update(ownershipPeriodTable).
		Set("valid_to", nil).
		Set("tm_update", now).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ownershipPeriodReopenByIDTx. err: %v", err)
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		if isMySQLDeadlock(err) {
			return ErrDeadlock
		}
		return fmt.Errorf("could not execute. ownershipPeriodReopenByIDTx. err: %v", err)
	}
	return nil
}

// ownershipPeriodInsertTx inserts a new period row.
func (h *handler) ownershipPeriodInsertTx(ctx context.Context, tx *sql.Tx, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string, validFrom *time.Time) error {
	now := h.utilHandler.TimeNow()
	id := uuid.Must(uuid.NewV4())
	query, args, err := sq.Insert(ownershipPeriodTable).
		SetMap(map[string]any{
			"id":          id.Bytes(),
			"customer_id": customerID.Bytes(),
			"contact_id":  contactID.Bytes(),
			"type":        string(addrType),
			"target":      target,
			"valid_from":  validFrom,
			"valid_to":    nil,
			"tm_create":   now,
			"tm_update":   now,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ownershipPeriodInsertTx. err: %v", err)
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		if isMySQLDeadlock(err) {
			return ErrDeadlock
		}
		return fmt.Errorf("could not execute. ownershipPeriodInsertTx. err: %v", err)
	}
	return nil
}

// ownershipPeriodLatestClosedValidToTx returns the latest valid_to among
// closed rows already locked for this target (design §4 Step 4's
// ClaimAddress-specific valid_from bound, round-38's corrected formula).
// Returns nil if there are no closed rows.
func ownershipPeriodLatestClosedValidTo(lockedRows []OwnershipPeriod) *time.Time {
	var latest *time.Time
	for i := range lockedRows {
		p := &lockedRows[i]
		if p.ValidTo == nil {
			continue
		}
		if latest == nil || p.ValidTo.After(*latest) {
			latest = p.ValidTo
		}
	}
	return latest
}

// applyOpenResolutionTx executes the period-table side effect implied by
// a step returned from OwnershipPeriodsLockAndResolveTx, for an OPEN-ing
// caller. validFromOnInsert is used only for StepInsertAfterIntervening/
// StepReassign/StepFirstRegistration -- callers pass the caller-specific
// value design §4 Step 4 requires (NOW() for AddressCreate/AddressUpdate,
// the latest closed valid_to in lockedRows for AddressClaim; nil for
// Step 5's first registration, per §4's own rule regardless of caller).
func (h *handler) applyOpenResolutionTx(ctx context.Context, tx *sql.Tx, step int, lockedRows []OwnershipPeriod, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string, validFromOnInsert *time.Time) error {
	switch step {
	case StepOpenReuse:
		return nil // nothing to do -- already open for this contact
	case StepReopen:
		var latestOwn *OwnershipPeriod
		for i := range lockedRows {
			p := &lockedRows[i]
			if p.ContactID != contactID || p.ValidTo == nil {
				continue
			}
			if latestOwn == nil || p.ValidTo.After(*latestOwn.ValidTo) {
				latestOwn = p
			}
		}
		if latestOwn == nil {
			return fmt.Errorf("could not find own closed period to reopen. applyOpenResolutionTx")
		}
		return h.ownershipPeriodReopenByIDTx(ctx, tx, latestOwn.ID)
	case StepInsertAfterIntervening, StepReassign, StepFirstRegistration:
		vf := validFromOnInsert
		if step == StepFirstRegistration {
			vf = nil // §4 Step 5: always valid_from=NULL regardless of caller
		}
		return h.ownershipPeriodInsertTx(ctx, tx, customerID, contactID, addrType, target, vf)
	default:
		return fmt.Errorf("unknown step %d. applyOpenResolutionTx", step)
	}
}
