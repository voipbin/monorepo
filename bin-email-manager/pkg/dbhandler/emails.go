package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commonaddress "monorepo/bin-common-handler/models/address"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-email-manager/models/email"
)

const (
	emailTable = "email_emails"
)

// emailGetFromRow scans a single row into an Email struct using db tags
func (h *handler) emailGetFromRow(rows *sql.Rows) (*email.Email, error) {
	res := &email.Email{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. emailGetFromRow. err: %v", err)
	}

	// Initialize nil pointers to empty values
	if res.Source == nil {
		res.Source = &commonaddress.Address{}
	}
	if res.Destinations == nil {
		res.Destinations = []commonaddress.Address{}
	}
	if res.Attachments == nil {
		res.Attachments = []email.Attachment{}
	}

	return res, nil
}

// EmailCreate creates a new email record.
func (h *handler) EmailCreate(ctx context.Context, e *email.Email) error {
	e.TMCreate = h.util.TimeGetCurTime()
	e.TMUpdate = DefaultTimeStamp
	e.TMDelete = DefaultTimeStamp

	// prepare fields for insert
	fields, err := commondatabasehandler.PrepareFields(e)
	if err != nil {
		return errors.Wrap(err, "could not prepare fields. EmailCreate")
	}

	query, args, err := sq.Insert(emailTable).SetMap(fields).ToSql()
	if err != nil {
		return errors.Wrap(err, "could not build query. EmailCreate")
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "could not execute. EmailCreate")
	}

	_ = h.emailUpdateToCache(ctx, e.ID)

	return nil
}

// emailUpdateToCache gets the email from the DB and updates the cache.
func (h *handler) emailUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.emailGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.emailSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// emailSetToCache sets the given email to the cache
func (h *handler) emailSetToCache(ctx context.Context, e *email.Email) error {
	if err := h.cache.EmailSet(ctx, e); err != nil {
		return err
	}

	return nil
}

// emailGetFromCache returns email from the cache if possible.
func (h *handler) emailGetFromCache(ctx context.Context, id uuid.UUID) (*email.Email, error) {
	res, err := h.cache.EmailGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// emailGetFromDB gets the email info from the db.
func (h *handler) emailGetFromDB(ctx context.Context, id uuid.UUID) (*email.Email, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&email.Email{})

	query, args, err := sq.Select(columns...).
		From(emailTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "could not build query. emailGetFromDB")
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "could not query. emailGetFromDB")
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.emailGetFromRow(rows)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// EmailGet returns email.
func (h *handler) EmailGet(ctx context.Context, id uuid.UUID) (*email.Email, error) {
	res, err := h.emailGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.emailGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.emailSetToCache(ctx, res)

	return res, nil
}

// EmailList returns emails based on filters.
func (h *handler) EmailList(ctx context.Context, token string, size uint64, filters map[email.Field]any) ([]*email.Email, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&email.Email{})

	builder := sq.Select(columns...).
		From(emailTable).
		Where("tm_create < ?", token).
		OrderBy("tm_create desc").
		Limit(size)

	// apply filters
	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, errors.Wrap(err, "could not apply filters. EmailList")
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "could not build query. EmailList")
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "could not query. EmailList")
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*email.Email{}
	for rows.Next() {
		e, err := h.emailGetFromRow(rows)
		if err != nil {
			return nil, errors.Wrap(err, "could not scan the row. EmailList")
		}

		res = append(res, e)
	}

	return res, nil
}

// EmailUpdate updates an email with the given fields.
func (h *handler) EmailUpdate(ctx context.Context, id uuid.UUID, fields map[email.Field]any) error {
	// add update timestamp
	fields[email.FieldTMUpdate] = h.util.TimeGetCurTime()

	// prepare fields for update
	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return errors.Wrap(err, "could not prepare fields. EmailUpdate")
	}

	query, args, err := sq.Update(emailTable).
		SetMap(data).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "could not build query. EmailUpdate")
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "could not execute. EmailUpdate")
	}

	// update the cache
	_ = h.emailUpdateToCache(ctx, id)

	return nil
}

// EmailDelete deletes the given email
func (h *handler) EmailDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.util.TimeGetCurTime()
	fields := map[email.Field]any{
		email.FieldTMUpdate: ts,
		email.FieldTMDelete: ts,
	}

	// prepare fields for update
	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return errors.Wrap(err, "could not prepare fields. EmailDelete")
	}

	query, args, err := sq.Update(emailTable).
		SetMap(data).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return errors.Wrap(err, "could not build query. EmailDelete")
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return errors.Wrap(err, "could not execute. EmailDelete")
	}

	_ = h.emailUpdateToCache(ctx, id)

	return nil
}

// EmailUpdateStatus updates the status.
func (h *handler) EmailUpdateStatus(ctx context.Context, id uuid.UUID, status email.Status) error {
	fields := map[email.Field]any{
		email.FieldStatus: status,
	}

	return h.EmailUpdate(ctx, id, fields)
}

// EmailUpdateProviderReferenceID updates the provider_reference_id.
func (h *handler) EmailUpdateProviderReferenceID(ctx context.Context, id uuid.UUID, providerReferenceID string) error {
	fields := map[email.Field]any{
		email.FieldProviderReferenceID: providerReferenceID,
	}

	return h.EmailUpdate(ctx, id, fields)
}
