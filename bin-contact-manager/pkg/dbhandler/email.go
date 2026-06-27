package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/contact"
)

// emailFromAddressRow maps a contact_addresses (type='email') row into an Email.
// The mapping is hand-written: target -> Address. The sub-type is dropped
// (§3.2), so Type is always empty.
func emailFromAddressRow(r *addressRow) *contact.Email {
	return &contact.Email{
		ID:         r.ID,
		CustomerID: r.CustomerID,
		ContactID:  r.ContactID,
		Address:    r.Target,
		IsPrimary:  r.IsPrimary,
		TMCreate:   r.TMCreate,
	}
}

// EmailCreate creates a new email record in contact_addresses.
func (h *handler) EmailCreate(ctx context.Context, e *contact.Email) error {
	e.TMCreate = h.utilHandler.TimeNow()

	query, args, err := sq.Insert(addressTable).
		SetMap(map[string]any{
			"id":          e.ID.Bytes(),
			"customer_id": e.CustomerID.Bytes(),
			"contact_id":  e.ContactID.Bytes(),
			"type":        addressTypeEmail,
			"target":      e.Address,
			"target_name": "",
			"is_primary":  e.IsPrimary,
			"tm_create":   e.TMCreate,
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. EmailCreate. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. EmailCreate. err: %v", err)
	}

	// update the contact cache
	_ = h.contactUpdateToCache(ctx, e.ContactID)

	return nil
}

// EmailGet retrieves a single email by ID from contact_addresses.
func (h *handler) EmailGet(_ context.Context, id uuid.UUID) (*contact.Email, error) {
	query, args, err := sq.Select(addressRowColumns()...).
		From(addressTable).
		Where(sq.Eq{"id": id.Bytes(), "type": addressTypeEmail}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. EmailGet. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. EmailGet. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	r, err := scanAddressRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan. EmailGet. err: %v", err)
	}

	return emailFromAddressRow(r), nil
}

// EmailUpdate updates an email record in contact_addresses.
// Caller field keys are translated to the unified table columns: "address" maps
// to "target"; "is_primary" is passed through; the dropped sub-type "type" is
// ignored (§3.2).
func (h *handler) EmailUpdate(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	contactID, err := h.addressContactID(id)
	if err != nil {
		return fmt.Errorf("could not get contact_id. EmailUpdate. err: %v", err)
	}

	q := sq.Update(addressTable).Where(sq.Eq{"id": id.Bytes(), "type": addressTypeEmail})
	for k, v := range fields {
		switch k {
		case "address":
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
		return fmt.Errorf("could not build query. EmailUpdate. err: %v", err)
	}

	if _, err := h.db.Exec(query, args...); err != nil {
		return fmt.Errorf("could not execute. EmailUpdate. err: %v", err)
	}

	if contactID != uuid.Nil {
		_ = h.contactUpdateToCache(ctx, contactID)
	}

	return nil
}

// EmailDelete deletes an email record from contact_addresses.
func (h *handler) EmailDelete(ctx context.Context, id uuid.UUID) error {
	contactID, err := h.addressContactID(id)
	if err != nil {
		return fmt.Errorf("could not get contact_id. EmailDelete. err: %v", err)
	}

	deleteQuery, deleteArgs, err := sq.Delete(addressTable).
		Where(sq.Eq{"id": id.Bytes(), "type": addressTypeEmail}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build delete query. EmailDelete. err: %v", err)
	}

	if _, err := h.db.Exec(deleteQuery, deleteArgs...); err != nil {
		return fmt.Errorf("could not execute delete. EmailDelete. err: %v", err)
	}

	if contactID != uuid.Nil {
		_ = h.contactUpdateToCache(ctx, contactID)
	}

	return nil
}

// EmailResetPrimary clears the primary flag for ALL addresses of the contact
// (across BOTH tel and email types), enforcing the one-primary-per-contact
// invariant of contact_addresses (UNIQUE customer_id, primary_contact_uk).
func (h *handler) EmailResetPrimary(ctx context.Context, contactID uuid.UUID) error {
	return h.addressResetPrimaryForContact(ctx, contactID)
}

// EmailListByContactID returns all emails for a contact as a reverse-projection
// over contact_addresses (type='email').
func (h *handler) EmailListByContactID(_ context.Context, contactID uuid.UUID) ([]contact.Email, error) {
	query, args, err := sq.Select(addressRowColumns()...).
		From(addressTable).
		Where(sq.Eq{"contact_id": contactID.Bytes(), "type": addressTypeEmail}).
		OrderBy("is_primary desc", "tm_create asc").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. EmailListByContactID. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. EmailListByContactID. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []contact.Email{}
	for rows.Next() {
		r, err := scanAddressRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. EmailListByContactID. err: %v", err)
		}
		res = append(res, *emailFromAddressRow(r))
	}

	return res, nil
}
