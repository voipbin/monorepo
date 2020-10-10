package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/number"
)

// NumberGetFromCacheByNumber returns number from the cache by number.
func (h *handler) NumberGetFromCacheByNumber(ctx context.Context, numb string) (*number.Number, error) {

	// get from cache
	res, err := h.cache.NumberGetByNumber(ctx, numb)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// NumberSetToCacheByNumber sets the given number to the cache
func (h *handler) NumberSetToCacheByNumber(ctx context.Context, num *number.Number) error {
	if err := h.cache.NumberSetByNumber(ctx, num); err != nil {
		return err
	}

	return nil
}

// NumberGetFromDBByNumber returns number info from the DB by number.
func (h *handler) NumberGetFromDBByNumber(ctx context.Context, numb string) (*number.Number, error) {

	// prepare
	q := `
	select
		id,
		number,
		flow_id,
		user_id,

		coalesce(tm_create, '') as tm_create,
		coalesce(tm_update, '') as tm_update,
		coalesce(tm_delete, '') as tm_delete

	from
		numbers
	where
		number = ?
	`

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
		return nil, fmt.Errorf("could not get call. NumberGetFromDBByNumber, err: %v", err)
	}

	return res, nil
}

// numberGetFromRow gets the number from the row.
func (h *handler) numberGetFromRow(row *sql.Rows) (*number.Number, error) {
	res := &number.Number{}
	if err := row.Scan(
		&res.ID,
		&res.Number,
		&res.FlowID,
		&res.UserID,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. callGetFromRow. err: %v", err)
	}

	return res, nil
}

// CallGet returns call.
func (h *handler) NumberGetByNumber(ctx context.Context, numb string) (*number.Number, error) {

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
