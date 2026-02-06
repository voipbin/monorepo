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
	emailTable = "contact_emails"
)

// emailGetFromRow scans a single row into an Email struct
func (h *handler) emailGetFromRow(rows *sql.Rows) (*contact.Email, error) {
	res := &contact.Email{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. emailGetFromRow. err: %v", err)
	}

	return res, nil
}

// EmailCreate creates a new email record
func (h *handler) EmailCreate(ctx context.Context, e *contact.Email) error {
	e.TMCreate = h.utilHandler.TimeNow()

	// prepare fields for insert
	fields, err := commondatabasehandler.PrepareFields(e)
	if err != nil {
		return fmt.Errorf("could not prepare fields. EmailCreate. err: %v", err)
	}

	query, args, err := sq.Insert(emailTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. EmailCreate. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. EmailCreate. err: %v", err)
	}

	// update the contact cache
	_ = h.contactUpdateToCache(ctx, e.ContactID)

	return nil
}

// EmailGet retrieves a single email by ID
func (h *handler) EmailGet(ctx context.Context, id uuid.UUID) (*contact.Email, error) {
	columns := commondatabasehandler.GetDBFields(&contact.Email{})

	query, args, err := sq.Select(columns...).
		From(emailTable).
		Where(sq.Eq{"id": id.Bytes()}).
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

	res, err := h.emailGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan. EmailGet. err: %v", err)
	}

	return res, nil
}

// EmailUpdate updates an email record
func (h *handler) EmailUpdate(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	q := sq.Update(emailTable).Where(sq.Eq{"id": id.Bytes()})
	for k, v := range fields {
		q = q.Set(k, v)
	}

	query, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. EmailUpdate. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. EmailUpdate. err: %v", err)
	}

	return nil
}

// EmailDelete deletes an email record
func (h *handler) EmailDelete(ctx context.Context, id uuid.UUID) error {
	// First get the email to find the contact_id for cache update
	query, args, err := sq.Select("contact_id").
		From(emailTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. EmailDelete. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("could not query. EmailDelete. err: %v", err)
	}

	var contactIDBytes []byte
	if rows.Next() {
		if err := rows.Scan(&contactIDBytes); err != nil {
			_ = rows.Close()
			return fmt.Errorf("could not scan contact_id. EmailDelete. err: %v", err)
		}
	}
	_ = rows.Close()

	// Delete the email
	deleteQuery, deleteArgs, err := sq.Delete(emailTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build delete query. EmailDelete. err: %v", err)
	}

	_, err = h.db.Exec(deleteQuery, deleteArgs...)
	if err != nil {
		return fmt.Errorf("could not execute delete. EmailDelete. err: %v", err)
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

// EmailResetPrimary sets is_primary to false for all emails of a contact
func (h *handler) EmailResetPrimary(ctx context.Context, contactID uuid.UUID) error {
	query, args, err := sq.Update(emailTable).
		Set("is_primary", false).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. EmailResetPrimary. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. EmailResetPrimary. err: %v", err)
	}

	return nil
}

// EmailListByContactID returns all emails for a contact
func (h *handler) EmailListByContactID(ctx context.Context, contactID uuid.UUID) ([]contact.Email, error) {
	columns := commondatabasehandler.GetDBFields(&contact.Email{})

	query, args, err := sq.Select(columns...).
		From(emailTable).
		Where(sq.Eq{"contact_id": contactID.Bytes()}).
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

	var res []contact.Email
	for rows.Next() {
		e, err := h.emailGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. EmailListByContactID. err: %v", err)
		}
		res = append(res, *e)
	}

	return res, nil
}
