package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-contact-manager/models/contact"
)

const (
	// addressTable is the unified store for a contact's identifiers
	// (phone numbers and emails), replacing the legacy contact_phone_numbers
	// and contact_emails tables (VOIP-1207).
	addressTable = "contact_addresses"

	// addressTypeTel and addressTypeEmail are the channel-type discriminators
	// stored in contact_addresses.type.
	addressTypeTel   = "tel"
	addressTypeEmail = "email"
)

// addressRow mirrors the contact_addresses columns the handler reads back.
type addressRow struct {
	ID         uuid.UUID  `db:"id,uuid"`
	CustomerID uuid.UUID  `db:"customer_id,uuid"`
	ContactID  uuid.UUID  `db:"contact_id,uuid"`
	Type       string     `db:"type"`
	Target     string     `db:"target"`
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
		ID:         r.ID,
		CustomerID: r.CustomerID,
		ContactID:  r.ContactID,
		Type:       r.Type,
		Target:     r.Target,
		Name:       r.Name,
		Detail:     r.Detail,
		IsPrimary:  r.IsPrimary,
		TMCreate:   r.TMCreate,
	}, nil
}

// addressContactID returns the contact_id of an address row by id (any type),
// used to refresh the contact cache after mutations. Returns uuid.Nil if the
// row is absent.
func (h *handler) addressContactID(id uuid.UUID) (uuid.UUID, error) {
	query, args, err := sq.Select("contact_id").
		From(addressTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not build query. addressContactID. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not query. addressContactID. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return uuid.Nil, nil
	}

	var contactIDBytes []byte
	if err := rows.Scan(&contactIDBytes); err != nil {
		return uuid.Nil, fmt.Errorf("could not scan contact_id. addressContactID. err: %v", err)
	}
	if len(contactIDBytes) == 0 {
		return uuid.Nil, nil
	}

	contactID, err := uuid.FromBytes(contactIDBytes)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not parse contact_id. addressContactID. err: %v", err)
	}
	return contactID, nil
}

// AddressCreate creates a new address in contact_addresses.
// Calls contactUpdateToCache after the DB write.
func (h *handler) AddressCreate(ctx context.Context, a *contact.Address) error {
	a.TMCreate = h.utilHandler.TimeNow()

	// contact_id is nullable (NULL = unresolved). uuid.Nil.Bytes() is 16 zero
	// bytes, NOT SQL NULL, so we must explicitly pass Go nil when unresolved.
	var contactIDValue any
	if a.ContactID != uuid.Nil {
		contactIDValue = a.ContactID.Bytes()
	}

	query, args, err := sq.Insert(addressTable).
		SetMap(map[string]any{
			"id":          a.ID.Bytes(),
			"customer_id": a.CustomerID.Bytes(),
			"contact_id":  contactIDValue,
			"type":        a.Type,
			"target":      a.Target,
			"target_name": "",
			"name":        a.Name,
			"detail":      a.Detail,
			"is_primary":  a.IsPrimary,
			"tm_create":   a.TMCreate,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. AddressCreate. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. AddressCreate. err: %v", err)
	}

	// update the contact cache (skip for unresolved addresses — there is no
	// contact to refresh yet)
	if a.ContactID != uuid.Nil {
		_ = h.contactUpdateToCache(ctx, a.ContactID)
	}

	return nil
}

// AddressClaim attaches contact_id to a currently-unresolved address.
// Returns ErrConflict if the address is already resolved to a DIFFERENT
// contact_id. No-ops (success) if already resolved to the SAME contact_id.
func (h *handler) AddressClaim(ctx context.Context, customerID, addressID, contactID uuid.UUID) error {
	existing, err := h.AddressGet(ctx, customerID, addressID) // tenant-scoped fetch
	if err != nil {
		return err // ErrNotFound propagates as-is
	}
	if existing.ContactID == contactID {
		return nil // already claimed by this contact — idempotent success
	}
	if existing.ContactID != uuid.Nil {
		return ErrConflict // resolved to a DIFFERENT contact — reject, no move
	}

	query, args, err := sq.Update(addressTable).
		Set("contact_id", contactID.Bytes()).
		Set("tm_update", h.utilHandler.TimeNow()).
		Where(sq.Eq{"id": addressID.Bytes()}).
		Where(sq.Eq{"contact_id": nil}). // race guard: only claim if STILL unresolved
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. AddressClaim. err: %v", err)
	}
	result, err := h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. AddressClaim. err: %v", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrConflict // lost the race — someone else claimed it between Get and Update
	}

	// Refresh the contact-body cache, mirroring AddressUpdate/AddressDelete.
	// This is the dbhandler layer's own responsibility (unexported method,
	// not reachable from contactHandler) and is independent of publishEvent.
	_ = h.contactUpdateToCache(ctx, contactID)

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
// filters keys: "contact_id" (uuid.UUID), "type" (string).
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
// contact_addresses has no soft-delete.
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
// fields keys: "target" (string), "is_primary" (bool).
// Resolves contactID via addressContactID() BEFORE the update, then calls contactUpdateToCache.
func (h *handler) AddressUpdate(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	contactID, err := h.addressContactID(id)
	if err != nil {
		return fmt.Errorf("could not get contact_id. AddressUpdate. err: %v", err)
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
		default:
			// ignore unknown fields
		}
	}
	q = q.Set("tm_update", h.utilHandler.TimeNow())

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. AddressUpdate. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. AddressUpdate. err: %v", err)
	}

	if contactID != uuid.Nil {
		_ = h.contactUpdateToCache(ctx, contactID)
	}

	return nil
}

// AddressDelete deletes an address by id.
// Resolves contactID via addressContactID() BEFORE the delete, then calls contactUpdateToCache.
func (h *handler) AddressDelete(ctx context.Context, id uuid.UUID) error {
	contactID, err := h.addressContactID(id)
	if err != nil {
		return fmt.Errorf("could not get contact_id. AddressDelete. err: %v", err)
	}

	deleteQuery, deleteArgs, err := sq.Delete(addressTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build delete query. AddressDelete. err: %v", err)
	}

	if _, err := h.db.Exec(deleteQuery, deleteArgs...); err != nil {
		return fmt.Errorf("could not execute delete. AddressDelete. err: %v", err)
	}

	if contactID != uuid.Nil {
		_ = h.contactUpdateToCache(ctx, contactID)
	}

	return nil
}

// AddressResetPrimary clears is_primary for ALL addresses of a contact (cross-type).
// Calls contactUpdateToCache after the reset.
func (h *handler) AddressResetPrimary(ctx context.Context, contactID uuid.UUID) error {
	query, args, err := sq.Update(addressTable).
		Set("is_primary", false).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. AddressResetPrimary. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. AddressResetPrimary. err: %v", err)
	}

	_ = h.contactUpdateToCache(ctx, contactID)

	return nil
}
