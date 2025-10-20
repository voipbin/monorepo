package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-email-manager/models/email"

	commonaddress "monorepo/bin-common-handler/models/address"
)

const (
	// select query for email get
	emailSelect = `
	select
		id,
		customer_id,

		activeflow_id,

		provider_type,
		provider_reference_id,

		source,
		destinations,

		status,
		subject,
		content,

		attachments,

		tm_create,
		tm_update,
		tm_delete
	from
		email_emails
	`
)

// emailGetFromRow gets the email from the row.
func (h *handler) emailGetFromRow(row *sql.Rows) (*email.Email, error) {
	var source sql.NullString
	var destinations sql.NullString
	var attachments sql.NullString

	res := &email.Email{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.ActiveflowID,

		&res.ProviderType,
		&res.ProviderReferenceID,

		&source,
		&destinations,

		&res.Status,
		&res.Subject,
		&res.Content,

		&attachments,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, errors.Wrapf(err, "could not scan the row.")
	}

	if source.Valid {
		if errUnmarshal := json.Unmarshal([]byte(source.String), &res.Source); errUnmarshal != nil {
			return nil, errors.Wrapf(errUnmarshal, "could not unmarshal the data.")
		}
	}
	if res.Source == nil {
		res.Source = &commonaddress.Address{}
	}

	if destinations.Valid {
		if errUnmarshal := json.Unmarshal([]byte(destinations.String), &res.Destinations); errUnmarshal != nil {
			return nil, errors.Wrapf(errUnmarshal, "could not unmarshal the data.")
		}
	}
	if res.Destinations == nil {
		res.Destinations = []commonaddress.Address{}
	}

	if attachments.Valid {
		if errUnmarshal := json.Unmarshal([]byte(attachments.String), &res.Attachments); errUnmarshal != nil {
			return nil, errors.Wrapf(errUnmarshal, "could not unmarshal the data.")
		}
	}
	if res.Attachments == nil {
		res.Attachments = []email.Attachment{}
	}

	return res, nil
}

func (h *handler) EmailCreate(ctx context.Context, e *email.Email) error {

	q := `insert into email_emails(
		id,
		customer_id,

		activeflow_id,

		provider_type,
		provider_reference_id,

		source,
		destinations,

		status,
		subject,
		content,

		attachments,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?,
		?, ?,
		?, ?,
		?, ?, ?,
		?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return errors.Wrapf(err, "could not prepare query.")
	}
	defer func() {
		_ = stmt.Close()
	}()

	tmpSource, err := json.Marshal(e.Source)
	if err != nil {
		return errors.Wrapf(err, "could not marshal source.")
	}

	tmpDestinations, err := json.Marshal(e.Destinations)
	if err != nil {
		return errors.Wrapf(err, "could not marshal destinations.")
	}

	tmpAttachments, err := json.Marshal(e.Attachments)
	if err != nil {
		return errors.Wrapf(err, "could not marshal attachments.")
	}

	_, err = stmt.ExecContext(ctx,
		e.ID.Bytes(),
		e.CustomerID.Bytes(),

		e.ActiveflowID.Bytes(),

		e.ProviderType,
		e.ProviderReferenceID,

		tmpSource,
		tmpDestinations,

		e.Status,
		e.Subject,
		e.Content,

		tmpAttachments,

		h.util.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return errors.Wrapf(err, "could not execute query.")
	}

	_ = h.emailUpdateToCache(ctx, e.ID)

	return nil
}

// emailUpdateToCache gets the email from the DB and update the cache.
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

	// get from cache
	res, err := h.cache.EmailGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// emailGetFromDB gets the email info from the db.
func (h *handler) emailGetFromDB(ctx context.Context, id uuid.UUID) (*email.Email, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", emailSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, errors.Wrapf(err, "could not prepare query.")
	}
	defer func() {
		_ = stmt.Close()
	}()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, errors.Wrapf(err, "could not query.")
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.emailGetFromRow(row)
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

// EmailGets returns emails.
func (h *handler) EmailGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*email.Email, error) {
	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, emailSelect)

	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	values := []any{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id":
			q = fmt.Sprintf("%s and customer_id = ?", q)
			tmp := uuid.FromStringOrNil(v)
			values = append(values, tmp.Bytes())

		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}

		default:
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, v)
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))
	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, errors.Wrapf(err, "could not query. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*email.Email{}
	for rows.Next() {
		u, err := h.emailGetFromRow(rows)
		if err != nil {
			return nil, errors.Wrapf(err, "could not scan the row.")
		}

		res = append(res, u)
	}

	return res, nil
}

// EmailDelete deletes the given email
func (h *handler) EmailDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update email_emails set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.util.TimeGetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return errors.Wrapf(err, "could not execute the query.")
	}

	_ = h.emailUpdateToCache(ctx, id)

	return nil
}

// EmailUpdateStatus updates the status.
func (h *handler) EmailUpdateStatus(ctx context.Context, id uuid.UUID, status email.Status) error {
	q := `
	update email_emails set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, status, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return errors.Wrapf(err, "could not execute the query.")
	}

	// set to the cache
	_ = h.emailUpdateToCache(ctx, id)

	return nil
}

// EmailUpdateProviderReferenceID updates the provider_reference_id.
func (h *handler) EmailUpdateProviderReferenceID(ctx context.Context, id uuid.UUID, providerReferenceID string) error {
	q := `
	update email_emails set
		provider_reference_id = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, providerReferenceID, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return errors.Wrapf(err, "could not execute the query.")
	}

	// set to the cache
	_ = h.emailUpdateToCache(ctx, id)

	return nil
}
