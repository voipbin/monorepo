package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/contact"
)

// phoneNumberFromAddressRow maps a contact_addresses (type='tel') row into a
// PhoneNumber. The mapping is hand-written: target -> Number. The sub-type is
// dropped (§3.2), so Type is always empty.
func phoneNumberFromAddressRow(r *addressRow) *contact.PhoneNumber {
	return &contact.PhoneNumber{
		ID:         r.ID,
		CustomerID: r.CustomerID,
		ContactID:  r.ContactID,
		Number:     r.Target,
		IsPrimary:  r.IsPrimary,
		TMCreate:   r.TMCreate,
	}
}

// PhoneNumberCreate creates a new phone number record in contact_addresses.
func (h *handler) PhoneNumberCreate(ctx context.Context, p *contact.PhoneNumber) error {
	p.TMCreate = h.utilHandler.TimeNow()

	query, args, err := sq.Insert(addressTable).
		SetMap(map[string]any{
			"id":          p.ID.Bytes(),
			"customer_id": p.CustomerID.Bytes(),
			"contact_id":  p.ContactID.Bytes(),
			"type":        addressTypeTel,
			"target":      p.Number,
			"target_name": "",
			"is_primary":  p.IsPrimary,
			"tm_create":   p.TMCreate,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. PhoneNumberCreate. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. PhoneNumberCreate. err: %v", err)
	}

	// update the contact cache
	_ = h.contactUpdateToCache(ctx, p.ContactID)

	return nil
}

// PhoneNumberGet retrieves a single phone number by ID from contact_addresses.
func (h *handler) PhoneNumberGet(_ context.Context, id uuid.UUID) (*contact.PhoneNumber, error) {
	query, args, err := sq.Select(addressRowColumns()...).
		From(addressTable).
		Where(sq.Eq{"id": id.Bytes(), "type": addressTypeTel}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. PhoneNumberGet. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. PhoneNumberGet. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	r, err := scanAddressRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan. PhoneNumberGet. err: %v", err)
	}

	return phoneNumberFromAddressRow(r), nil
}

// PhoneNumberUpdate updates a phone number record in contact_addresses.
// Caller field keys are translated to the unified table columns: "number" maps
// to "target"; "is_primary" is passed through; the dropped sub-type "type" is
// ignored (§3.2).
func (h *handler) PhoneNumberUpdate(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	contactID, err := h.addressContactID(id)
	if err != nil {
		return fmt.Errorf("could not get contact_id. PhoneNumberUpdate. err: %v", err)
	}

	q := sq.Update(addressTable).Where(sq.Eq{"id": id.Bytes(), "type": addressTypeTel})
	for k, v := range fields {
		switch k {
		case "number":
			q = q.Set("target", v)
		case "is_primary":
			q = q.Set("is_primary", v)
		default:
			// sub-type and any other legacy field have no column; ignore.
		}
	}
	q = q.Set("tm_update", h.utilHandler.TimeNow())

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. PhoneNumberUpdate. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. PhoneNumberUpdate. err: %v", err)
	}

	if contactID != uuid.Nil {
		_ = h.contactUpdateToCache(ctx, contactID)
	}

	return nil
}

// PhoneNumberDelete deletes a phone number record from contact_addresses.
func (h *handler) PhoneNumberDelete(ctx context.Context, id uuid.UUID) error {
	contactID, err := h.addressContactID(id)
	if err != nil {
		return fmt.Errorf("could not get contact_id. PhoneNumberDelete. err: %v", err)
	}

	deleteQuery, deleteArgs, err := sq.Delete(addressTable).
		Where(sq.Eq{"id": id.Bytes(), "type": addressTypeTel}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build delete query. PhoneNumberDelete. err: %v", err)
	}

	if _, err := h.db.Exec(deleteQuery, deleteArgs...); err != nil {
		return fmt.Errorf("could not execute delete. PhoneNumberDelete. err: %v", err)
	}

	if contactID != uuid.Nil {
		_ = h.contactUpdateToCache(ctx, contactID)
	}

	return nil
}

// PhoneNumberResetPrimary clears the primary flag for ALL addresses of the
// contact (across BOTH tel and email types), enforcing the one-primary-per-
// contact invariant of contact_addresses (UNIQUE customer_id, primary_contact_uk).
func (h *handler) PhoneNumberResetPrimary(ctx context.Context, contactID uuid.UUID) error {
	return h.addressResetPrimaryForContact(ctx, contactID)
}

// PhoneNumberListByContactID returns all phone numbers for a contact as a
// reverse-projection over contact_addresses (type='tel').
func (h *handler) PhoneNumberListByContactID(_ context.Context, contactID uuid.UUID) ([]contact.PhoneNumber, error) {
	query, args, err := sq.Select(addressRowColumns()...).
		From(addressTable).
		Where(sq.Eq{"contact_id": contactID.Bytes(), "type": addressTypeTel}).
		OrderBy("is_primary desc", "tm_create asc").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. PhoneNumberListByContactID. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. PhoneNumberListByContactID. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []contact.PhoneNumber{}
	for rows.Next() {
		r, err := scanAddressRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. PhoneNumberListByContactID. err: %v", err)
		}
		res = append(res, *phoneNumberFromAddressRow(r))
	}

	return res, nil
}
