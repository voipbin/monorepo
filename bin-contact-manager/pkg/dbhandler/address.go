package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-contact-manager/models/contact"
)

const (
	// addressTable is the unified store for a contact's identifiers
	// (phone numbers and emails), replacing the legacy contact_phone_numbers
	// and contact_emails tables (VOIP-1207).
	addressTable = "contact_addresses"
)

// addressRow mirrors the contact_addresses columns the handler reads back.
type addressRow struct {
	ID         uuid.UUID  `db:"id,uuid"`
	CustomerID uuid.UUID  `db:"customer_id,uuid"`
	ContactID  uuid.UUID  `db:"contact_id,uuid"`
	Type       string     `db:"type"`
	Target     string     `db:"target"`
	TargetName string     `db:"target_name"`
	Name       string     `db:"name"`
	Detail     string     `db:"detail"`
	IsPrimary  bool       `db:"is_primary"`
	TMCreate   *time.Time `db:"tm_create"`
}

// addressRowColumns is the ordered SELECT column list matching addressRow.
func addressRowColumns() []string {
	return commondatabasehandler.GetDBFields(&addressRow{})
}

// scanFullAddressRow scans a single contact_addresses row into contact.Address.
func scanFullAddressRow(rows *sql.Rows) (*contact.Address, error) {
	r := &addressRow{}
	if err := commondatabasehandler.ScanRow(rows, r); err != nil {
		return nil, fmt.Errorf("could not scan the row. scanFullAddressRow. err: %v", err)
	}
	return &contact.Address{
		Address: commonaddress.Address{
			Type:       commonaddress.Type(r.Type),
			Target:     r.Target,
			TargetName: r.TargetName,
			Name:       r.Name,
			Detail:     r.Detail,
		},
		ID:         r.ID,
		CustomerID: r.CustomerID,
		ContactID:  r.ContactID,
		IsPrimary:  r.IsPrimary,
		TMCreate:   r.TMCreate,
	}, nil
}

// AddressCreate creates a new address in contact_addresses, wrapping
// AddressCreateTx (design §5.1) in a BeginTx/commit/retry loop. Retries
// on ErrDeadlock up to addressMaxDeadlockRetries (design §5.3), matching
// casehandler.getOrCreateAttempt's pattern.
func (h *handler) AddressCreate(ctx context.Context, a *contact.Address) error {
	var lastErr error
	for attempt := 0; attempt < addressMaxDeadlockRetries; attempt++ {
		err := h.addressCreateAttempt(ctx, a)
		if err == nil {
			break
		}
		if err == ErrDeadlock {
			lastErr = err
			continue
		}
		return err
	}
	if lastErr != nil {
		return fmt.Errorf("could not create address: exhausted retries under sustained deadlock. err: %v", lastErr)
	}

	// update the contact cache (skip for unresolved addresses -- there is
	// no contact to refresh yet)
	if a.ContactID != uuid.Nil {
		_ = h.contactUpdateToCache(ctx, a.ContactID)
	}
	return nil
}

func (h *handler) addressCreateAttempt(ctx context.Context, a *contact.Address) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction. AddressCreate. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := h.AddressCreateTx(ctx, tx, a); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction. AddressCreate. err: %v", err)
	}
	committed = true
	return nil
}

// AddressClaim attaches contact_id to a currently-unresolved address.
// Returns ErrConflict if the address is already resolved to a DIFFERENT
// contact_id. No-ops (success) if already resolved to the SAME contact_id.
// Wraps AddressClaimTx in a BeginTx/commit/retry loop (design §5.1/§5.3).
func (h *handler) AddressClaim(ctx context.Context, customerID, addressID, contactID uuid.UUID) error {
	existing, err := h.AddressGet(ctx, customerID, addressID) // tenant-scoped fetch
	if err != nil {
		return err // ErrNotFound propagates as-is
	}
	if existing.ContactID == contactID {
		return nil // already claimed by this contact -- idempotent success
	}
	if existing.ContactID != uuid.Nil {
		return ErrConflict // resolved to a DIFFERENT contact -- reject, no move
	}

	var lastErr error
	for attempt := 0; attempt < addressMaxDeadlockRetries; attempt++ {
		err := h.addressClaimAttempt(ctx, customerID, addressID, contactID, existing.Type, existing.Target)
		if err == nil {
			break
		}
		if err == ErrDeadlock || err == ErrStaleTarget {
			lastErr = err
			continue
		}
		return err
	}
	if lastErr != nil {
		return fmt.Errorf("could not claim address: exhausted retries under sustained deadlock. err: %v", lastErr)
	}

	// Refresh the contact-body cache, mirroring AddressUpdate/AddressDelete.
	_ = h.contactUpdateToCache(ctx, contactID)
	return nil
}

func (h *handler) addressClaimAttempt(ctx context.Context, customerID, addressID, contactID uuid.UUID, addrType commonaddress.Type, target string) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction. AddressClaim. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Post-lock re-read: design §4's round-8/39 stale-target hazard --
	// confirm the row is still unresolved and still has this target
	// under this tx before writing the period.
	current, err := addressTypeTargetContactByID(tx, addressID)
	if err != nil {
		return err
	}
	if current.Type != string(addrType) || current.Target != target {
		return ErrStaleTarget
	}
	if current.ContactID != uuid.Nil && current.ContactID != contactID {
		return ErrConflict // lost the race -- someone else claimed it
	}

	if err := h.AddressClaimTx(ctx, tx, customerID, addressID, contactID, addrType, target); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction. AddressClaim. err: %v", err)
	}
	committed = true
	return nil
}

// AddressGet returns a single address by id, scoped to customerID for tenant isolation.
// Returns ErrNotFound if absent or belongs to a different customer.
func (h *handler) AddressGet(ctx context.Context, customerID, id uuid.UUID) (*contact.Address, error) {
	query, args, err := sq.Select(addressRowColumns()...).
		From(addressTable).
		Where(sq.Eq{"id": id.Bytes()}).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. AddressGet. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. AddressGet. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	a, err := scanFullAddressRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan. AddressGet. err: %v", err)
	}

	return a, nil
}

// AddressList returns addresses for the customer with optional filters.
// filters keys: "contact_id" (uuid.UUID), "type" (string), "unresolved"
// (bool — when true, restricts to rows where contact_id IS NULL and takes
// precedence over "contact_id" if both are given).
func (h *handler) AddressList(_ context.Context, customerID uuid.UUID, filters map[string]any, pageToken string, pageSize uint64) ([]contact.Address, error) {
	q := sq.Select(addressRowColumns()...).
		From(addressTable).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		OrderBy("tm_create desc")

	// unresolved=true takes precedence over contact_id per the OpenAPI spec
	// ("if both are given, unresolved=true wins and contact_id is ignored").
	// PR review finding: applying both filters unconditionally produced a
	// self-contradictory `contact_id = ? AND contact_id IS NULL` clause that
	// always returned zero rows.
	unresolved := false
	if v, ok := filters["unresolved"]; ok {
		if b, ok2 := v.(bool); ok2 && b {
			unresolved = true
			q = q.Where(sq.Eq{"contact_id": nil}) // squirrel renders IS NULL for nil
		}
	}
	if !unresolved {
		if v, ok := filters["contact_id"]; ok {
			if cid, ok2 := v.(uuid.UUID); ok2 && cid != uuid.Nil {
				q = q.Where(sq.Eq{"contact_id": cid.Bytes()})
			}
		}
	}
	if v, ok := filters["type"]; ok {
		if t, ok2 := v.(string); ok2 && t != "" {
			q = q.Where(sq.Eq{"type": t})
		}
	}
	if pageSize > 0 {
		q = q.Limit(pageSize)
	}
	if pageToken != "" {
		q = q.Where(sq.Lt{"tm_create": pageToken})
	}

	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. AddressList. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. AddressList. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []contact.Address{}
	for rows.Next() {
		a, err := scanFullAddressRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. AddressList. err: %v", err)
		}
		res = append(res, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. AddressList. err: %v", err)
	}

	return res, nil
}

// AddressListByContactID returns all addresses for a contact.
// contact_addresses has no soft-delete. NOT modified by this design
// (design §6.1): shared by ContactGet/ContactList/contactUpdateToCache
// to populate the public Contact.Addresses API field.
func (h *handler) AddressListByContactID(_ context.Context, contactID uuid.UUID) ([]contact.Address, error) {
	query, args, err := sq.Select(addressRowColumns()...).
		From(addressTable).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		OrderBy("is_primary desc", "tm_create asc").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. AddressListByContactID. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. AddressListByContactID. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []contact.Address{}
	for rows.Next() {
		a, err := scanFullAddressRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. AddressListByContactID. err: %v", err)
		}
		res = append(res, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error. AddressListByContactID. err: %v", err)
	}

	return res, nil
}

// AddressUpdate updates target and/or is_primary for an address by id.
// fields keys: "target" (string), "is_primary" (bool). Wraps
// AddressUpdateTx in a BeginTx/commit/retry loop (design §5.1/§5.3),
// re-reading (type, target, contact_id) fresh on every retry iteration
// (design §4 round-45's fix for the composed-retry-loop hazard).
func (h *handler) AddressUpdate(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	var lastErr error
	var contactID uuid.UUID
	for attempt := 0; attempt < addressMaxDeadlockRetries; attempt++ {
		// Pre-lock read, re-run fresh every iteration (design §4
		// round-45). Fetched OUTSIDE any tx, before BeginTx, so it
		// never contends with the tx's own connection (the SQLite test
		// harness caps the pool at 1 connection).
		pre, err := addressTypeTargetContactByID(h.db, id)
		if err != nil {
			return err // ErrNotFound propagates immediately, never retried
		}
		contactID = pre.ContactID
		customerID, err := h.addressCustomerID(id)
		if err != nil {
			return err
		}

		err = h.addressUpdateAttempt(ctx, id, customerID, pre.ContactID, commonaddress.Type(pre.Type), pre.Target, fields)
		if err == nil {
			lastErr = nil
			break
		}
		if err == ErrDeadlock || err == ErrStaleTarget {
			lastErr = err
			continue
		}
		return err
	}
	if lastErr != nil {
		return fmt.Errorf("could not update address: exhausted retries under sustained deadlock. err: %v", lastErr)
	}

	if contactID != uuid.Nil {
		_ = h.contactUpdateToCache(ctx, contactID)
	}
	return nil
}

func (h *handler) addressUpdateAttempt(ctx context.Context, id, customerID, contactID uuid.UUID, oldType commonaddress.Type, oldTarget string, fields map[string]any) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction. AddressUpdate. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Post-lock re-read: confirm the row hasn't moved since the
	// pre-lock read (design §4 round-39/53's stale-target hazard).
	current, err := addressTypeTargetContactByID(tx, id)
	if err != nil {
		return err
	}
	if current.Type != string(oldType) || current.Target != oldTarget || current.ContactID != contactID {
		return ErrStaleTarget
	}

	if isPrimary, ok := fields["is_primary"].(bool); ok && isPrimary && contactID != uuid.Nil {
		if err := h.AddressResetPrimaryTx(ctx, tx, contactID); err != nil {
			return err
		}
	}

	if err := h.AddressUpdateTx(ctx, tx, id, customerID, contactID, oldType, oldTarget, fields); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction. AddressUpdate. err: %v", err)
	}
	committed = true
	return nil
}

// addressCustomerID returns the customer_id of an address row by id.
func (h *handler) addressCustomerID(id uuid.UUID) (uuid.UUID, error) {
	query, args, err := sq.Select("customer_id").
		From(addressTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not build query. addressCustomerID. err: %v", err)
	}
	rows, err := h.db.Query(query, args...)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not query. addressCustomerID. err: %v", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return uuid.Nil, ErrNotFound
	}
	var customerIDBytes []byte
	if err := rows.Scan(&customerIDBytes); err != nil {
		return uuid.Nil, fmt.Errorf("could not scan customer_id. addressCustomerID. err: %v", err)
	}
	customerID, err := uuid.FromBytes(customerIDBytes)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not parse customer_id. addressCustomerID. err: %v", err)
	}
	return customerID, nil
}

// AddressDelete deletes an address by id. Wraps AddressDeleteTx in a
// BeginTx/commit/retry loop (design §5.1/§5.3), re-reading
// (type, target, contact_id, customer_id) fresh on every retry
// iteration.
func (h *handler) AddressDelete(ctx context.Context, id uuid.UUID) error {
	var lastErr error
	var contactID uuid.UUID
	for attempt := 0; attempt < addressMaxDeadlockRetries; attempt++ {
		pre, err := addressTypeTargetContactByID(h.db, id)
		if err != nil {
			return err
		}
		contactID = pre.ContactID
		customerID, err := h.addressCustomerID(id)
		if err != nil {
			return err
		}

		err = h.addressDeleteAttempt(ctx, id, customerID, pre.ContactID, commonaddress.Type(pre.Type), pre.Target)
		if err == nil {
			lastErr = nil
			break
		}
		if err == ErrDeadlock || err == ErrStaleTarget {
			lastErr = err
			continue
		}
		return err
	}
	if lastErr != nil {
		return fmt.Errorf("could not delete address: exhausted retries under sustained deadlock. err: %v", lastErr)
	}

	if contactID != uuid.Nil {
		_ = h.contactUpdateToCache(ctx, contactID)
	}
	return nil
}

func (h *handler) addressDeleteAttempt(ctx context.Context, id, customerID, contactID uuid.UUID, addrType commonaddress.Type, target string) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction. AddressDelete. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	current, err := addressTypeTargetContactByID(tx, id)
	if err != nil {
		return err
	}
	if current.Type != string(addrType) || current.Target != target || current.ContactID != contactID {
		return ErrStaleTarget
	}

	if err := h.AddressDeleteTx(ctx, tx, id, customerID, contactID, addrType, target); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction. AddressDelete. err: %v", err)
	}
	committed = true
	return nil
}

// AddressResetPrimary clears is_primary for ALL addresses of a contact
// (cross-type). Wraps AddressResetPrimaryTx in a BeginTx/commit loop.
func (h *handler) AddressResetPrimary(ctx context.Context, contactID uuid.UUID) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction. AddressResetPrimary. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := h.AddressResetPrimaryTx(ctx, tx, contactID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction. AddressResetPrimary. err: %v", err)
	}
	committed = true

	_ = h.contactUpdateToCache(ctx, contactID)
	return nil
}
