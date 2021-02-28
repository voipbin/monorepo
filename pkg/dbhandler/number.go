package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models"
)

const (
	numberSelect = `
	select
		id,
		number,
		flow_id,
		user_id,

		provider_name,
		provider_reference_id,

		status,

		t38_enabled,
		emergency_enabled,

		coalesce(tm_purchase, '') as tm_purchase,
		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete

	from
		numbers
	`
)

// numberGetFromRow gets the number from the row.
func (h *handler) numberGetFromRow(row *sql.Rows) (*models.Number, error) {
	res := &models.Number{}
	if err := row.Scan(
		&res.ID,
		&res.Number,
		&res.FlowID,
		&res.UserID,

		&res.ProviderName,
		&res.ProviderReferenceID,

		&res.Status,

		&res.T38Enabled,
		&res.EmergencyEnabled,

		&res.TMPurchase,
		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. numberGetFromRow. err: %v", err)
	}

	return res, nil
}

// NumberGetFromCacheByNumber returns number from the cache by number.
func (h *handler) NumberGetFromCacheByNumber(ctx context.Context, numb string) (*models.Number, error) {

	// get from cache
	res, err := h.cache.NumberGetByNumber(ctx, numb)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// NumberSetToCacheByNumber sets the given number to the cache
func (h *handler) NumberSetToCacheByNumber(ctx context.Context, num *models.Number) error {
	if err := h.cache.NumberSetByNumber(ctx, num); err != nil {
		return err
	}

	return nil
}

// NumberGetFromCache returns number from the cache.
func (h *handler) NumberGetFromCache(ctx context.Context, id uuid.UUID) (*models.Number, error) {

	// get from cache
	res, err := h.cache.NumberGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// NumberSetToCache sets the given number to the cache
func (h *handler) NumberSetToCache(ctx context.Context, num *models.Number) error {
	if err := h.cache.NumberSet(ctx, num); err != nil {
		return err
	}

	return nil
}

// NumberUpdateToCache gets the number from the DB and update the cache.
func (h *handler) NumberUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.NumberGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.NumberSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// NumberUpdateToCacheByNumber gets the number by number from the DB and update the cache.
func (h *handler) NumberUpdateToCacheByNumber(ctx context.Context, num string) error {

	res, err := h.NumberGetFromDBByNumber(ctx, num)
	if err != nil {
		return err
	}

	if err := h.NumberSetToCacheByNumber(ctx, res); err != nil {
		return err
	}

	return nil
}

// NumberGetFromDB returns number info from the DB.
func (h *handler) NumberGetFromDB(ctx context.Context, id uuid.UUID) (*models.Number, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", numberSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. NumberGetFromDB. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.numberGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get number. NumberGetFromDB, err: %v", err)
	}

	return res, nil
}

// NumberGetFromDBByNumber returns number info from the DB by number.
func (h *handler) NumberGetFromDBByNumber(ctx context.Context, numb string) (*models.Number, error) {

	// prepare
	q := fmt.Sprintf("%s where number = ? and tm_delete is null", numberSelect)

	row, err := h.db.Query(q, numb)
	if err != nil {
		return nil, fmt.Errorf("could not query. NumberGetFromDBByNumber. err: %v", err)
	}
	defer row.Close()

	if row.Next() == false {
		return nil, ErrNotFound
	}

	res, err := h.numberGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get number. NumberGetFromDBByNumber, err: %v", err)
	}

	return res, nil
}

// NumberGet returns number.
func (h *handler) NumberGet(ctx context.Context, id uuid.UUID) (*models.Number, error) {

	res, err := h.NumberGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.NumberGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	h.NumberSetToCache(ctx, res)

	return res, nil
}

// NumberGetByNumber returns number by number.
func (h *handler) NumberGetByNumber(ctx context.Context, numb string) (*models.Number, error) {

	res, err := h.NumberGetFromCacheByNumber(ctx, numb)
	if err == nil {
		return res, nil
	}

	res, err = h.NumberGetFromDBByNumber(ctx, numb)
	if err != nil {
		return nil, err
	}

	// set to the cache
	h.NumberSetToCacheByNumber(ctx, res)

	return res, nil
}

// NumberGets returns a list of numbers.
func (h *handler) NumberGets(ctx context.Context, userID uint64, size uint64, token string) ([]*models.Number, error) {

	// prepare
	q := fmt.Sprintf("%s where user_id = ? and tm_create < ? order by tm_create desc limit ?", numberSelect)

	rows, err := h.db.Query(q, userID, token, size)
	if err != nil {
		return nil, fmt.Errorf("could not query. NumberGets. err: %v", err)
	}
	defer rows.Close()

	res := []*models.Number{}
	for rows.Next() {
		u, err := h.numberGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. NumberGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// NumberCreate creates a new number record.
func (h *handler) NumberCreate(ctx context.Context, n *models.Number) error {
	q := `insert into numbers(
		id,
		number,
		flow_id,
		user_id,

		provider_name,
		provider_reference_id,

		status,

		t38_enabled,
		emergency_enabled,

		tm_purchase,
		tm_create
	) values(
		?, ?, ?, ?,
		?, ?,
		?,
		?, ?,
		?, ?
		)`

	_, err := h.db.Exec(q,
		n.ID.Bytes(),
		n.Number,
		n.FlowID.Bytes(),
		n.UserID,

		n.ProviderName,
		n.ProviderReferenceID,

		n.Status,

		n.T38Enabled,
		n.EmergencyEnabled,

		n.TMPurchase,
		getCurTime(),
	)
	if err != nil {
		return fmt.Errorf("could not execute. NumberCreate. err: %v", err)
	}

	// update the cache
	h.NumberUpdateToCache(ctx, n.ID)
	h.NumberUpdateToCacheByNumber(ctx, n.Number)

	return nil
}

// NumberDelete sets the delte timestamp.
func (h *handler) NumberDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
		update
			numbers
		set
			status = ?,
			tm_update = ?,
			tm_delete = ?
		where
			id = ?
		`

	curTime := getCurTime()
	_, err := h.db.Exec(q, string(models.NumberStatusDeleted), curTime, curTime, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. NumberDelete. err: %v", err)
	}

	// update the cache
	h.NumberUpdateToCache(ctx, id)
	tmpNum, _ := h.NumberGet(ctx, id)
	h.NumberSetToCacheByNumber(ctx, tmpNum)

	return nil
}
