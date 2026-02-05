package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-contact-manager/models/contact"
)

const (
	phoneNumberTable = "contact_phone_numbers"
)

// phoneNumberGetFromRow scans a single row into a PhoneNumber struct
func (h *handler) phoneNumberGetFromRow(rows *sql.Rows) (*contact.PhoneNumber, error) {
	res := &contact.PhoneNumber{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. phoneNumberGetFromRow. err: %v", err)
	}

	return res, nil
}

// PhoneNumberCreate creates a new phone number record
func (h *handler) PhoneNumberCreate(ctx context.Context, p *contact.PhoneNumber) error {
	p.TMCreate = h.utilHandler.TimeNow()

	// prepare fields for insert
	fields, err := commondatabasehandler.PrepareFields(p)
	if err != nil {
		return fmt.Errorf("could not prepare fields. PhoneNumberCreate. err: %v", err)
	}

	query, args, err := sq.Insert(phoneNumberTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. PhoneNumberCreate. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. PhoneNumberCreate. err: %v", err)
	}

	// update the contact cache
	_ = h.contactUpdateToCache(ctx, p.ContactID)

	return nil
}

// PhoneNumberDelete deletes a phone number record
func (h *handler) PhoneNumberDelete(ctx context.Context, id uuid.UUID) error {
	// First get the phone number to find the contact_id for cache update
	query, args, err := sq.Select("contact_id").
		From(phoneNumberTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. PhoneNumberDelete. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("could not query. PhoneNumberDelete. err: %v", err)
	}

	var contactIDBytes []byte
	if rows.Next() {
		if err := rows.Scan(&contactIDBytes); err != nil {
			_ = rows.Close()
			return fmt.Errorf("could not scan contact_id. PhoneNumberDelete. err: %v", err)
		}
	}
	_ = rows.Close()

	// Delete the phone number
	deleteQuery, deleteArgs, err := sq.Delete(phoneNumberTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build delete query. PhoneNumberDelete. err: %v", err)
	}

	_, err = h.db.Exec(deleteQuery, deleteArgs...)
	if err != nil {
		return fmt.Errorf("could not execute delete. PhoneNumberDelete. err: %v", err)
	}

	// Update the contact cache if we have a contact_id
	if len(contactIDBytes) > 0 {
		contactID, err := uuid.FromBytes(contactIDBytes)
		if err == nil {
			_ = h.contactUpdateToCache(ctx, contactID)
		}
	}

	return nil
}

// PhoneNumberListByContactID returns all phone numbers for a contact
func (h *handler) PhoneNumberListByContactID(ctx context.Context, contactID uuid.UUID) ([]contact.PhoneNumber, error) {
	columns := commondatabasehandler.GetDBFields(&contact.PhoneNumber{})

	query, args, err := sq.Select(columns...).
		From(phoneNumberTable).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
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

	var res []contact.PhoneNumber
	for rows.Next() {
		p, err := h.phoneNumberGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. PhoneNumberListByContactID. err: %v", err)
		}
		res = append(res, *p)
	}

	return res, nil
}
