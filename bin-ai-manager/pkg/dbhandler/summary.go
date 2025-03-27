package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"monorepo/bin-ai-manager/models/summary"
	"strconv"

	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

const (
	summarySelect = `
	select
		id,
		customer_id,

		reference_type,
		reference_id,

		language,
		content,

		tm_create,
		tm_update,
		tm_delete

	from
		ai_summaries
	`
)

// summaryGetFromRow gets the message from the row.
func (h *handler) summaryGetFromRow(row *sql.Rows) (*summary.Summary, error) {
	res := &summary.Summary{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.ReferenceType,
		&res.ReferenceID,

		&res.Language,
		&res.Content,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, errors.Wrap(err, "summaryGetFromRow: Could not scan the row")
	}

	return res, nil
}

// SummaryCreate creates a new summary record.
func (h *handler) SummaryCreate(ctx context.Context, c *summary.Summary) error {
	q := `insert into ai_summaries(
		id,
		customer_id,

		reference_type,
		reference_id,

		language,
		content,
		
		tm_create,
		tm_update,
		tm_delete
	) values (
		?, ?,
		?, ?,
		?, ?,
		?, ?, ?
		)
	`

	_, err := h.db.Exec(q,
		c.ID.Bytes(),
		c.CustomerID.Bytes(),

		c.ReferenceType,
		c.ReferenceID.Bytes(),

		c.Language,
		c.Content,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("SummaryCreate: Could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.summaryUpdateToCache(ctx, c.ID)

	return nil
}

// summaryGetFromCache returns summary from the cache.
func (h *handler) summaryGetFromCache(ctx context.Context, id uuid.UUID) (*summary.Summary, error) {

	// get from cache
	res, err := h.cache.SummaryGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// summaryGetFromDB returns summary from the DB.
func (h *handler) summaryGetFromDB(ctx context.Context, id uuid.UUID) (*summary.Summary, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", summarySelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. summaryGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.summaryGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// summaryUpdateToCache gets the message from the DB and updates the cache.
func (h *handler) summaryUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.summaryGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.summarySetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// summarySetToCache sets the given message to the cache.
func (h *handler) summarySetToCache(ctx context.Context, c *summary.Summary) error {
	if err := h.cache.SummarySet(ctx, c); err != nil {
		return err
	}

	return nil
}

// SummaryGet returns message.
func (h *handler) SummaryGet(ctx context.Context, id uuid.UUID) (*summary.Summary, error) {

	res, err := h.summaryGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.summaryGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.summarySetToCache(ctx, res)

	return res, nil
}

// SummaryDelete deletes the message.
func (h *handler) SummaryDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update ai_summaries set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. SummaryDelete. err: %v", err)
	}

	// update the cache
	_ = h.summaryUpdateToCache(ctx, id)

	return nil
}

// SummaryGets returns a list of summaries.
func (h *handler) SummaryGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*summary.Summary, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// prepare the query
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, summarySelect)

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "reference_id":
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, uuid.FromStringOrNil(v).Bytes())

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
		return nil, fmt.Errorf("could not query. SummaryGets. err: %v", err)
	}
	defer rows.Close()

	res := []*summary.Summary{}
	for rows.Next() {
		u, err := h.summaryGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. SummaryGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}
