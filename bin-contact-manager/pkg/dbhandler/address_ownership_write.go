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
	// literal).
	if me, ok := execErr.(*mysql_driver.MySQLError); ok && me.Number == 1062 {
		if strings.Contains(me.Message, "idx_contact_addresses_identifier") {
			return ErrDuplicateTarget
		}
		return fmt.Errorf("could not execute. addressInsertTx. err: %v", execErr)
	}
	if strings.Contains(execErr.Error(), "UNIQUE constraint failed") &&
		strings.Contains(execErr.Error(), "contact_addresses.type") &&
		strings.Contains(execErr.Error(), "contact_addresses.target") {
		return ErrDuplicateTarget
	}
	return fmt.Errorf("could not execute. addressInsertTx. err: %v", execErr)
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
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		if isMySQLDeadlock(err) {
			return ErrDeadlock
		}
		return fmt.Errorf("could not execute. AddressUpdateTx. err: %v", err)
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
	if _, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
		if isMySQLDeadlock(err) {
			return ErrDeadlock
		}
		return fmt.Errorf("could not execute delete. AddressDeleteTx. err: %v", err)
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
