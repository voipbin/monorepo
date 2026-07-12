package dbhandler

// Tx-suffixed write functions for the five original address write
// operations plus AddressDeleteCompensating (design §5.1). Each calls
// OwnershipPeriodsLockAndResolveTx (address_ownership.go) as its first
// step, then applies the resulting period-table side effect, then
// performs its contact_addresses write -- all inside the caller-supplied
// tx. AddressResetPrimaryTx is the one exception (design §5.1 round-56):
// it never calls the shared primitive, since is_primary is not a
// per-target ownership concern.

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

	"monorepo/bin-contact-manager/models/contact"
)

// addressRowTypeTargetContact is the (type, target, contact_id) triple
// read from a contact_addresses row by id -- the pre-lock read design
// §4's stale-target compare-and-retry rule requires for AddressUpdateTx/
// AddressDeleteTx (round-52/53's contact_id sourcing).
type addressRowTypeTargetContact struct {
	Type      string
	Target    string
	ContactID uuid.UUID
}

// addressTypeTargetContactByID reads (type, target, contact_id) for one
// contact_addresses row by id. Used both as the pre-lock read (outside
// any tx) and the post-lock re-read (inside tx, via execer) that design
// §4's stale-target compare-and-retry rule requires.
func addressTypeTargetContactByID(execer sqlExecutor, id uuid.UUID) (*addressRowTypeTargetContact, error) {
	query, args, err := sq.Select("type", "target", "contact_id").
		From(addressTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. addressTypeTargetContactByID. err: %v", err)
	}
	rows, err := execer.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. addressTypeTargetContactByID. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}
	var addrType, target string
	var contactIDBytes []byte
	if err := rows.Scan(&addrType, &target, &contactIDBytes); err != nil {
		return nil, fmt.Errorf("could not scan. addressTypeTargetContactByID. err: %v", err)
	}
	var contactID uuid.UUID
	if len(contactIDBytes) > 0 {
		contactID, err = uuid.FromBytes(contactIDBytes)
		if err != nil {
			return nil, fmt.Errorf("could not parse contact_id. addressTypeTargetContactByID. err: %v", err)
		}
	}
	return &addressRowTypeTargetContact{Type: addrType, Target: target, ContactID: contactID}, nil
}

// AddressCreateTx is AddressCreate's Tx-suffixed sibling (design §5.1).
// If a.ContactID == uuid.Nil (CreateUnresolvedAddress), it skips the §4
// locking read/period write entirely -- design §4's round-10 rule that
// unresolved addresses get no period until a later AddressClaimTx.
func (h *handler) AddressCreateTx(ctx context.Context, tx *sql.Tx, a *contact.Address) error {
	if a.ContactID != uuid.Nil {
		step, lockedRows, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, a.CustomerID, a.ContactID, a.Type, a.Target)
		if err != nil {
			return err
		}
		var validFrom *time.Time
		if step == StepInsertAfterIntervening || step == StepReassign {
			validFrom = h.utilHandler.TimeNow() // AddressCreate: NOW(), design §4 Step 4
		}
		if err := h.applyOpenResolutionTx(ctx, tx, step, lockedRows, a.CustomerID, a.ContactID, a.Type, a.Target, validFrom); err != nil {
			return err
		}
	}

	return h.addressInsertTx(ctx, tx, a)
}

// addressInsertTx is the contact_addresses INSERT shared by
// AddressCreateTx and AddressDeleteCompensating's retry path -- the same
// statement AddressCreate (address.go) issues today, just parameterized
// over tx instead of h.db.
func (h *handler) addressInsertTx(ctx context.Context, tx *sql.Tx, a *contact.Address) error {
	a.TMCreate = h.utilHandler.TimeNow()

	var contactIDValue any
	if a.ContactID != uuid.Nil {
		contactIDValue = a.ContactID.Bytes()
	}

	query, args, err := sq.Insert(addressTable).
		SetMap(map[string]any{
			"id":          a.ID.Bytes(),
			"customer_id": a.CustomerID.Bytes(),
			"contact_id":  contactIDValue,
			"type":        string(a.Type),
			"target":      a.Target,
			"target_name": a.TargetName,
			"name":        a.Name,
			"detail":      a.Detail,
			"is_primary":  a.IsPrimary,
			"tm_create":   a.TMCreate,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. addressInsertTx. err: %v", err)
	}

	_, execErr := tx.ExecContext(ctx, query, args...)
	if execErr == nil {
		return nil
	}
	if isMySQLDeadlock(execErr) {
		return ErrDeadlock
	}
	// Same 1062 classification AddressCreate already performs (design
	// §4 round-28's index-name-drift bundled fix: match the PRODUCTION
	// index name, not the stale idx_contact_addresses_cust_type_target
	// literal). Design §4 round-32: the repair-and-retry sequence runs
	// in SEPARATE transactions from this poisoned one -- a MySQL
	// transaction that has seen a 1062 is not safely reusable for
	// further writes. This function only classifies and returns;
	// AddressCreate's outer retry loop (address.go) performs the
	// repair (its own fresh BeginTx) and the retry (another fresh
	// BeginTx via a normal loop iteration) once this poisoned
	// transaction has rolled back.
	if isDuplicateTargetErr(execErr) {
		return ErrDuplicateTarget
	}
	return fmt.Errorf("could not execute. addressInsertTx. err: %v", execErr)
}

// isDuplicateTargetErr classifies a contact_addresses INSERT/UPDATE
// error as the (customer_id, type, target) unique-index collision
// (design §4 round-28's bundled index-name-drift fix: match the
// PRODUCTION index name).
func isDuplicateTargetErr(execErr error) bool {
	if me, ok := execErr.(*mysql_driver.MySQLError); ok && me.Number == 1062 {
		return strings.Contains(me.Message, "idx_contact_addresses_identifier")
	}
	return strings.Contains(execErr.Error(), "UNIQUE constraint failed") &&
		strings.Contains(execErr.Error(), "contact_addresses.type") &&
		strings.Contains(execErr.Error(), "contact_addresses.target")
}

// attemptStaleRowRepairNewTx implements design §4 round-32's
// transaction-boundary rule: the duplicate-key repair runs in its OWN
// fresh transaction, never inside the poisoned transaction that just saw
// the 1062 (which InnoDB does not guarantee is safely reusable for
// further writes after an error). Callers (AddressCreate/AddressUpdate/
// AddressClaim's outer retry loops) invoke this once after their inner
// Tx-suffixed attempt returns ErrDuplicateTarget, then retry their own
// attempt in ANOTHER fresh transaction (an ordinary loop iteration) --
// three separate §5.1 transactions total, matching the design's
// specified sequence.
func (h *handler) attemptStaleRowRepairNewTx(ctx context.Context, customerID uuid.UUID, addrType commonaddress.Type, target string, resetToNull bool) (bool, error) {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("could not begin transaction. attemptStaleRowRepairNewTx. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	retry, repairErr := h.staleRowRepairTx(ctx, tx, customerID, addrType, target, resetToNull)
	if repairErr != nil {
		return false, repairErr
	}
	if !retry {
		// Live owner -- nothing was written. Let the deferred
		// rollback close this (empty, read-only) transaction; no
		// commit needed since staleRowRepairTx made no writes on this
		// path.
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("could not commit transaction. attemptStaleRowRepairNewTx. err: %v", err)
	}
	committed = true
	return true, nil
}

// staleRowRepairTx implements design §4 round-27/28/29/30's duplicate-key
// repair path: on a unique-index collision, check whether the occupying
// contact_addresses row's owner is tombstoned. If the owner is LIVE, this
// is a genuine conflict -- returns (false, nil), letting the caller
// surface its own collision error. If the owner is TOMBSTONED (an
// A9-b/A9-c version-skew artifact), this closes a period for the dead
// owner's era (bounded by round-30's GREATEST(latest existing valid_to,
// stale row's tm_create), collapsing to zero-length -- no period written
// -- when that bound is not before the tombstone timestamp, per round-15/31's
// inversion guard) and vacates the slot: hard-DELETE for the AddressCreate/
// AddressUpdate callers (resetToNull=false), or NULL-reset for
// AddressClaimTx (resetToNull=true, design §4 round-28: deleting the row
// AddressClaimTx's own final UPDATE targets would always miss and 409).
// If NO row occupies the slot at all (design §4 round-27(b): a losing
// party in a concurrent repair race sees the winner's DELETE/reset
// already landed), there is nothing to repair here but the caller should
// still retry immediately -- the slot is already vacated.
// Returns (true, nil) if the caller should retry its own write (either a
// repair was just performed, or the row was already gone); (false, nil)
// if the owner is live (caller surfaces its own collision error); (false,
// err) on read failure.
func (h *handler) staleRowRepairTx(ctx context.Context, tx *sql.Tx, customerID uuid.UUID, addrType commonaddress.Type, target string, resetToNull bool) (bool, error) {
	row, err := staleRowByTargetTx(ctx, tx, customerID, addrType, target)
	if err != nil {
		return false, err
	}
	if row == nil {
		// Round-27(b): the occupying row is already gone (a concurrent
		// repair won the race and committed/rolled back before this
		// transaction's read) -- nothing left to repair, but the slot
		// is vacated, so the caller should retry immediately.
		return true, nil
	}

	tmDelete, err := contactTombstoneTx(ctx, tx, row.ContactID)
	if err != nil {
		return false, err
	}
	if tmDelete == nil {
		return false, nil // live owner -- genuine conflict
	}

	// Round-30: valid_from = GREATEST(latest existing closed valid_to for
	// this target, stale row's tm_create) -- the strictly-better lower
	// bound available for free in this same transaction.
	//
	// Note (round-2 code review): when this function is called after
	// AddressCreateTx/AddressUpdateTx's own OwnershipPeriodsLockAndResolveTx
	// call already ran (Step 1's orphan-close branch, in the SAME
	// caller's earlier attempt before the 1062), that earlier Step 1 may
	// have already closed a period for this exact tombstoned contactID
	// at NOW() (>= the tombstone timestamp, since ContactDelete cannot
	// predate its own row's existence). That prior close makes
	// latestValidTo >= tmDelete, which the inversion guard below
	// correctly treats as "already covered" and skips re-inserting.
	// This is intentional, not a coincidence to preserve blindly: Step
	// 1's orphan-close and this function's fabricated-period write are
	// two independent mechanisms for the same invariant (the dead
	// owner's era must be recorded exactly once), and the inversion
	// guard is what keeps them from double-writing when both run
	// against the same dead owner in the same repair sequence.
	latestValidTo, err := latestOwnershipPeriodValidToTx(ctx, tx, customerID, addrType, target)
	if err != nil {
		return false, err
	}
	validFrom := row.TMCreate
	if latestValidTo != nil && (validFrom == nil || latestValidTo.After(*validFrom)) {
		validFrom = latestValidTo
	}

	// Round-15/31 inversion guard: only write the fabricated period if
	// it is non-empty (validFrom strictly before the tombstone time).
	// A zero-length/inverted bound means the dead owner's era left no
	// history to record -- skip the INSERT (zero-length disposition).
	if validFrom == nil || validFrom.Before(*tmDelete) {
		id := uuid.Must(uuid.NewV4())
		now := h.utilHandler.TimeNow()
		insQuery, insArgs, err := sq.Insert(ownershipPeriodTable).
			SetMap(map[string]any{
				"id":          id.Bytes(),
				"customer_id": customerID.Bytes(),
				"contact_id":  row.ContactID.Bytes(),
				"type":        string(addrType),
				"target":      target,
				"valid_from":  validFrom,
				"valid_to":    tmDelete,
				"tm_create":   now,
				"tm_update":   now,
			}).
			ToSql()
		if err != nil {
			return false, fmt.Errorf("could not build query. staleRowRepairTx. err: %v", err)
		}
		if _, err := tx.ExecContext(ctx, insQuery, insArgs...); err != nil {
			if isMySQLDeadlock(err) {
				return false, ErrDeadlock
			}
			return false, fmt.Errorf("could not execute. staleRowRepairTx. err: %v", err)
		}
	}

	if resetToNull {
		resetQuery, resetArgs, err := sq.Update(addressTable).
			Set("contact_id", nil).
			Set("tm_update", h.utilHandler.TimeNow()).
			Where(sq.Eq{"id": row.ID.Bytes()}).
			ToSql()
		if err != nil {
			return false, fmt.Errorf("could not build reset query. staleRowRepairTx. err: %v", err)
		}
		if _, err := tx.ExecContext(ctx, resetQuery, resetArgs...); err != nil {
			if isMySQLDeadlock(err) {
				return false, ErrDeadlock
			}
			return false, fmt.Errorf("could not execute reset. staleRowRepairTx. err: %v", err)
		}
	} else {
		delQuery, delArgs, err := sq.Delete(addressTable).
			Where(sq.Eq{"id": row.ID.Bytes()}).
			ToSql()
		if err != nil {
			return false, fmt.Errorf("could not build delete query. staleRowRepairTx. err: %v", err)
		}
		if _, err := tx.ExecContext(ctx, delQuery, delArgs...); err != nil {
			if isMySQLDeadlock(err) {
				return false, ErrDeadlock
			}
			return false, fmt.Errorf("could not execute delete. staleRowRepairTx. err: %v", err)
		}
	}
	return true, nil
}

// staleRowByID row shape for staleRowRepairTx's occupying-row read.
type staleRowByID struct {
	ID        uuid.UUID
	ContactID uuid.UUID
	TMCreate  *time.Time
}

// staleRowByTargetTx reads the (id, contact_id, tm_create) of the
// contact_addresses row currently occupying (customer_id, type, target),
// if any. Returns (nil, nil) if no row occupies the slot.
func staleRowByTargetTx(ctx context.Context, tx *sql.Tx, customerID uuid.UUID, addrType commonaddress.Type, target string) (*staleRowByID, error) {
	query, args, err := sq.Select("id", "contact_id", "tm_create").
		From(addressTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"type": string(addrType)}).
		Where(sq.Eq{"target": target}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. staleRowByTargetTx. err: %v", err)
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		if isMySQLDeadlock(err) {
			return nil, ErrDeadlock
		}
		return nil, fmt.Errorf("could not query. staleRowByTargetTx. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, nil
	}
	var idBytes, contactIDBytes []byte
	var tmCreate sql.NullString
	if err := rows.Scan(&idBytes, &contactIDBytes, &tmCreate); err != nil {
		return nil, fmt.Errorf("could not scan. staleRowByTargetTx. err: %v", err)
	}
	id, err := uuid.FromBytes(idBytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse id. staleRowByTargetTx. err: %v", err)
	}
	var contactID uuid.UUID
	if len(contactIDBytes) > 0 {
		contactID, err = uuid.FromBytes(contactIDBytes)
		if err != nil {
			return nil, fmt.Errorf("could not parse contact_id. staleRowByTargetTx. err: %v", err)
		}
	}
	tmCreatePtr, err := parseNullableDBTime(tmCreate)
	if err != nil {
		return nil, fmt.Errorf("could not parse tm_create. staleRowByTargetTx. err: %v", err)
	}
	return &staleRowByID{ID: id, ContactID: contactID, TMCreate: tmCreatePtr}, nil
}

// contactTombstoneTx reads contact_contacts.tm_delete for one contact.
// Returns nil if the contact is live (tm_delete IS NULL) or absent.
func contactTombstoneTx(ctx context.Context, tx *sql.Tx, contactID uuid.UUID) (*time.Time, error) {
	if contactID == uuid.Nil {
		return nil, nil
	}
	query, args, err := sq.Select("tm_delete").
		From(contactTable).
		Where(sq.Eq{"id": contactID.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. contactTombstoneTx. err: %v", err)
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		if isMySQLDeadlock(err) {
			return nil, ErrDeadlock
		}
		return nil, fmt.Errorf("could not query. contactTombstoneTx. err: %v", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, nil // Contact row gone entirely -- treat as live/unknown, no repair
	}
	var tmDelete sql.NullString
	if err := rows.Scan(&tmDelete); err != nil {
		return nil, fmt.Errorf("could not scan. contactTombstoneTx. err: %v", err)
	}
	return parseNullableDBTime(tmDelete)
}

// latestOwnershipPeriodValidToTx returns the latest valid_to among all
// CLOSED periods for (customer_id, type, target), or nil if none exist.
func latestOwnershipPeriodValidToTx(ctx context.Context, tx *sql.Tx, customerID uuid.UUID, addrType commonaddress.Type, target string) (*time.Time, error) {
	query, args, err := sq.Select("valid_to").
		From(ownershipPeriodTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"type": string(addrType)}).
		Where(sq.Eq{"target": target}).
		Where(sq.NotEq{"valid_to": nil}).
		OrderBy("valid_to DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. latestOwnershipPeriodValidToTx. err: %v", err)
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		if isMySQLDeadlock(err) {
			return nil, ErrDeadlock
		}
		return nil, fmt.Errorf("could not query. latestOwnershipPeriodValidToTx. err: %v", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, nil
	}
	var validTo sql.NullString
	if err := rows.Scan(&validTo); err != nil {
		return nil, fmt.Errorf("could not scan. latestOwnershipPeriodValidToTx. err: %v", err)
	}
	return parseNullableDBTime(validTo)
}

// AddressUpdateTx is AddressUpdate's Tx-suffixed sibling. contactID and
// the OLD (type, target) must be supplied by the caller (the thin
// wrapper's pre-lock read, per design §4 round-39/52/53's stale-target
// hazard). If fields["target"] differs from oldType/oldTarget, this
// closes the OLD target's period (CLOSE-ing) and opens/reuses the NEW
// target's period (OPEN-ing), acquiring the two locking reads in
// ascending (type, target) order (design §5.2's two-target rule).
func (h *handler) AddressUpdateTx(ctx context.Context, tx *sql.Tx, id, customerID, contactID uuid.UUID, oldType commonaddress.Type, oldTarget string, fields map[string]any) error {
	newTarget, targetChanged := fields["target"].(string)
	if targetChanged && newTarget == oldTarget {
		targetChanged = false // same value supplied -- not an actual change
	}

	if targetChanged {
		// Type never changes on update -- only target does.
		newAddrType := oldType

		// Ascending (type, target) order -- design §5.2 round-6.
		oldKey := string(oldType) + oldTarget
		newKey := string(newAddrType) + newTarget

		doOld := func() error {
			return h.closeOwnOpenPeriodTx(ctx, tx, customerID, contactID, oldType, oldTarget)
		}
		doNew := func() error {
			return h.addressUpdateOpenNewTargetTx(ctx, tx, customerID, contactID, newAddrType, newTarget)
		}

		if oldKey < newKey {
			if err := doOld(); err != nil {
				return err
			}
			if err := doNew(); err != nil {
				return err
			}
		} else {
			if err := doNew(); err != nil {
				return err
			}
			if err := doOld(); err != nil {
				return err
			}
		}
	}

	q := sq.Update(addressTable).Where(sq.Eq{"id": id.Bytes()})
	for k, v := range fields {
		switch k {
		case "target":
			q = q.Set("target", v)
		case "name":
			q = q.Set("name", v)
		case "detail":
			q = q.Set("detail", v)
		case "is_primary":
			q = q.Set("is_primary", v)
		}
	}
	q = q.Set("tm_update", h.utilHandler.TimeNow())

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. AddressUpdateTx. err: %v", err)
	}
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		if isMySQLDeadlock(err) {
			return ErrDeadlock
		}
		if targetChanged && isDuplicateTargetErr(err) {
			// Design §4 round-30's extension of the round-28
			// duplicate-key repair path to AddressUpdate's
			// target-change side. Round-32: the repair itself does
			// NOT run here -- it needs a fresh transaction, since
			// this one has just seen a 1062. This function only
			// classifies and returns; AddressUpdate's outer retry
			// loop (address.go) performs the repair (fresh BeginTx)
			// and the retry (another fresh BeginTx, an ordinary loop
			// iteration).
			return ErrDuplicateTarget
		}
		return fmt.Errorf("could not execute. AddressUpdateTx. err: %v", err)
	}
	// B5 fix: the post-lock re-read above (addressUpdateAttempt's
	// addressTypeTargetContactByID) is a plain SELECT, not SELECT ...
	// FOR UPDATE -- it confirms the row's shape a moment before this
	// UPDATE, not atomically with it. If a concurrent AddressDelete
	// removes the row in the gap between that read and this write
	// (TOCTOU, same class of race AddressClaim's RowsAffected guard was
	// already written to catch), the UPDATE affects zero rows and would
	// otherwise return a silent success. Explicitly check RowsAffected
	// and surface ErrStaleTarget, which the outer retry loop (address.go)
	// already treats as retry-eligible -- the retry's fresh pre-lock
	// read will correctly resolve to ErrNotFound this time.
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("could not read rows affected. AddressUpdateTx. err: %v", err)
	} else if n == 0 {
		return ErrStaleTarget
	}
	return nil
}

// addressUpdateOpenNewTargetTx is AddressUpdateTx's OPEN-ing call for the
// NEW target.
func (h *handler) addressUpdateOpenNewTargetTx(ctx context.Context, tx *sql.Tx, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string) error {
	step, lockedRows, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, contactID, addrType, target)
	if err != nil {
		return err
	}
	var validFrom *time.Time
	if step == StepInsertAfterIntervening || step == StepReassign {
		validFrom = h.utilHandler.TimeNow() // AddressUpdate: NOW(), design §4 Step 4
	}
	return h.applyOpenResolutionTx(ctx, tx, step, lockedRows, customerID, contactID, addrType, target, validFrom)
}

// AddressDeleteTx is AddressDelete's Tx-suffixed sibling -- a CLOSE-ing
// caller (design §5.1 round-48/49/52/57).
func (h *handler) AddressDeleteTx(ctx context.Context, tx *sql.Tx, addressID, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string) error {
	if err := h.closeOwnOpenPeriodTx(ctx, tx, customerID, contactID, addrType, target); err != nil {
		return err
	}

	deleteQuery, deleteArgs, err := sq.Delete(addressTable).
		Where(sq.Eq{"id": addressID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build delete query. AddressDeleteTx. err: %v", err)
	}
	res, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...)
	if err != nil {
		if isMySQLDeadlock(err) {
			return ErrDeadlock
		}
		return fmt.Errorf("could not execute delete. AddressDeleteTx. err: %v", err)
	}
	// B5 fix: same TOCTOU class as AddressUpdateTx's guard above -- the
	// post-lock re-read (addressDeleteAttempt's addressTypeTargetContactByID)
	// is a plain SELECT, not SELECT ... FOR UPDATE. If a concurrent
	// AddressDelete/AddressUpdate (re-target) already removed this exact
	// row in the gap since that read, this DELETE affects zero rows and
	// would otherwise return a silent success while having ALREADY
	// closed this contact's period above -- a double-close of history
	// for a row that a concurrent winner is also closing/moving.
	// Surfacing ErrStaleTarget lets the outer retry loop (address.go)
	// re-read fresh and correctly resolve to ErrNotFound.
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("could not read rows affected. AddressDeleteTx. err: %v", err)
	} else if n == 0 {
		return ErrStaleTarget
	}
	return nil
}

// closeOwnOpenPeriodTx is the shared CLOSE-ing-caller logic
// AddressDeleteTx and AddressUpdateTx's old-target close both use
// (design §5.1 round-48/49/57): acquire the locking read (ignoring
// step), find this contact's own open row in lockedRows and close it.
// If lockedRows is entirely empty, this is design §9's round-16/17
// rolling-deploy version-skew state, not an error -- skip silently
// (TODO: increment a Prometheus skew counter post-commit, out of Phase 1
// scope). If lockedRows is non-empty but no row is open for this
// contact, that is a genuine conflict -- ErrConflict (design §5.1
// round-49's B5-parity rule).
func (h *handler) closeOwnOpenPeriodTx(ctx context.Context, tx *sql.Tx, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string) error {
	_, lockedRows, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, contactID, addrType, target)
	if err != nil {
		return err
	}

	if len(lockedRows) == 0 {
		// Skew: no period ever existed for this target. Not an error --
		// the contact_addresses write proceeds and commits normally.
		return nil
	}

	for i := range lockedRows {
		p := &lockedRows[i]
		if p.ContactID == contactID && p.ValidTo == nil {
			return h.ownershipPeriodCloseByIDTx(ctx, tx, p.ID)
		}
	}

	// lockedRows is non-empty but no row was open for this contact --
	// a genuine conflict (a concurrent close already ran, or this
	// contact never actually held the target).
	return ErrConflict
}

// AddressClaimTx is AddressClaim's Tx-suffixed sibling -- an OPEN-ing
// caller.
func (h *handler) AddressClaimTx(ctx context.Context, tx *sql.Tx, customerID, addressID, contactID uuid.UUID, addrType commonaddress.Type, target string) error {
	step, lockedRows, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, contactID, addrType, target)
	if err != nil {
		return err
	}

	var validFrom *time.Time
	if step == StepInsertAfterIntervening || step == StepReassign {
		// AddressClaim: valid_from = latest closed valid_to for this
		// target (design §4 Step 4's round-38 corrected bound).
		validFrom = ownershipPeriodLatestClosedValidTo(lockedRows)
	}
	if err := h.applyOpenResolutionTx(ctx, tx, step, lockedRows, customerID, contactID, addrType, target, validFrom); err != nil {
		return err
	}

	query, args, err := sq.Update(addressTable).
		Set("contact_id", contactID.Bytes()).
		Set("tm_update", h.utilHandler.TimeNow()).
		Where(sq.Eq{"id": addressID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. AddressClaimTx. err: %v", err)
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		if isMySQLDeadlock(err) {
			return ErrDeadlock
		}
		return fmt.Errorf("could not execute. AddressClaimTx. err: %v", err)
	}
	return nil
}

// AddressResetPrimaryTx is AddressResetPrimary's Tx-suffixed sibling
// (design §5.1 round-56): it never calls
// OwnershipPeriodsLockAndResolveTx -- clearing is_primary on a contact's
// other addresses is not a per-target ownership operation.
func (h *handler) AddressResetPrimaryTx(ctx context.Context, tx *sql.Tx, contactID uuid.UUID) error {
	query, args, err := sq.Update(addressTable).
		Set("is_primary", false).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. AddressResetPrimaryTx. err: %v", err)
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		if isMySQLDeadlock(err) {
			return ErrDeadlock
		}
		return fmt.Errorf("could not execute. AddressResetPrimaryTx. err: %v", err)
	}
	return nil
}

// AddressDeleteCompensating is the seventh, distinct locking-read caller
// (design §5.1 round-55): a standalone operation (its own BeginTx, its
// own maxDeadlockRetries=3 retry loop) that ContactCreate's per-address
// loop calls to undo an already-committed address when a LATER address
// in the same loop fails. Unlike AddressDeleteTx, it hard-DELETEs the
// period row (not valid_to=NOW()) -- design §4 round-31's sanctioned
// exception for cleaning up a period this same loop just inserted -- and
// has NO skew exemption (design §4 round-57): an empty lockedRows here is
// unconditionally a genuine conflict, never version skew, since the
// period being compensated for was inserted moments earlier in this same
// deploy.
func (h *handler) AddressDeleteCompensating(ctx context.Context, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string) error {
	var lastErr error
	for attempt := 0; attempt < addressMaxDeadlockRetries; attempt++ {
		err := h.addressDeleteCompensatingAttempt(ctx, customerID, contactID, addrType, target)
		if err == nil {
			return nil
		}
		if err == ErrDeadlock {
			lastErr = err
			continue
		}
		return err
	}
	return fmt.Errorf("could not compensate address delete: exhausted retries under sustained deadlock. err: %v", lastErr)
}

func (h *handler) addressDeleteCompensatingAttempt(ctx context.Context, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction. AddressDeleteCompensating. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	_, lockedRows, err := h.OwnershipPeriodsLockAndResolveTx(ctx, tx, customerID, contactID, addrType, target)
	if err != nil {
		return err
	}

	var ownOpen *OwnershipPeriod
	for i := range lockedRows {
		p := &lockedRows[i]
		if p.ContactID == contactID && p.ValidTo == nil {
			ownOpen = p
			break
		}
	}
	if ownOpen == nil {
		// No skew exemption for this caller (design §4 round-57) --
		// an empty/mismatched lockedRows here is always a genuine
		// conflict, since the period being compensated for was just
		// inserted by this same request.
		return ErrConflict
	}

	deletePeriodQuery, deletePeriodArgs, err := sq.Delete(ownershipPeriodTable).
		Where(sq.Eq{"id": ownOpen.ID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build period delete query. AddressDeleteCompensating. err: %v", err)
	}
	if _, err := tx.ExecContext(ctx, deletePeriodQuery, deletePeriodArgs...); err != nil {
		if isMySQLDeadlock(err) {
			return ErrDeadlock
		}
		return fmt.Errorf("could not delete period. AddressDeleteCompensating. err: %v", err)
	}

	deleteAddrQuery, deleteAddrArgs, err := sq.Delete(addressTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where(sq.Eq{"type": string(addrType)}).
		Where(sq.Eq{"target": target}).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build address delete query. AddressDeleteCompensating. err: %v", err)
	}
	if _, err := tx.ExecContext(ctx, deleteAddrQuery, deleteAddrArgs...); err != nil {
		if isMySQLDeadlock(err) {
			return ErrDeadlock
		}
		return fmt.Errorf("could not delete address row. AddressDeleteCompensating. err: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction. AddressDeleteCompensating. err: %v", err)
	}
	committed = true
	return nil
}
