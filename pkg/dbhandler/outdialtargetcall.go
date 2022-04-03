package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtargetcall"
)

const (
	// select query for outdial get
	outdialTargetCallSelect = `
	select
		id,
		customer_id,
		campaign_id,
		outdial_id,
		outdial_target_id,

		activeflow_id,
		reference_type,
		reference_id,

		status,

		destination,
		destination_index,
		try_count,

		tm_create,
		tm_update,
		tm_delete
	from
		outdialtargetcalls
	`
)

// outdialTargetCallGetFromRow gets the outdialtargetcall from the row.
func (h *handler) outdialTargetCallGetFromRow(row *sql.Rows) (*outdialtargetcall.OutdialTargetCall, error) {
	var destination string

	res := &outdialtargetcall.OutdialTargetCall{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.CampaignID,
		&res.OutdialID,
		&res.OutdialTargetID,

		&res.ActiveflowID,
		&res.ReferenceType,
		&res.ReferenceID,

		&res.Status,

		&destination,
		&res.DestinationIndex,
		&res.TryCount,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. outdialTargetCallGetFromRow. err: %v", err)
	}

	if errDestination := json.Unmarshal([]byte(destination), &res.Destination); errDestination != nil {
		return nil, fmt.Errorf("could not unmarshal the destination. outdialTargetCallGetFromRow. err: %v", errDestination)
	}

	return res, nil
}

// OutdialTargetCallCreate insert a new outdialtargetcall record
func (h *handler) OutdialTargetCallCreate(ctx context.Context, t *outdialtargetcall.OutdialTargetCall) error {

	q := `insert into outdialtargetcalls(
		id,
		customer_id,
		campaign_id,
		outdial_id,
		outdial_target_id,

		activeflow_id,
		reference_type,
		reference_id,

		status,

		destination,
		destination_index,
		try_count,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?, ?, ?,
		?, ?, ?,
		?,
		?, ?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. OutdialTargetCallCreate. err: %v", err)
	}
	defer stmt.Close()

	destination, err := json.Marshal(t.Destination)
	if err != nil {
		return fmt.Errorf("could not marshal the destination. OutdialTargetCallCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		t.ID.Bytes(),
		t.CustomerID.Bytes(),
		t.CampaignID.Bytes(),
		t.OutdialID.Bytes(),
		t.OutdialTargetID.Bytes(),

		t.ActiveflowID.Bytes(),
		t.ReferenceType,
		t.ReferenceID.Bytes(),

		t.Status,

		destination,
		t.DestinationIndex,
		t.TryCount,

		t.TMCreate,
		t.TMUpdate,
		t.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. OutdialTargetCallCreate. err: %v", err)
	}

	_ = h.outdialTargetCallUpdateToCache(ctx, t.ID)

	return nil
}

// outdialTargetCallUpdateToCache gets the outdialtargetcall from the DB and update the cache.
func (h *handler) outdialTargetCallUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.outdialTargetCallGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.outdialTargetCallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// outdialTargetCallSetToCache sets the given outdialTargetCall to the cache
func (h *handler) outdialTargetCallSetToCache(ctx context.Context, t *outdialtargetcall.OutdialTargetCall) error {
	if err := h.cache.OutdialTargetCallSet(ctx, t); err != nil {
		return err
	}

	if t.ActiveflowID != uuid.Nil {
		if errActiveflow := h.cache.OutdialTargetCallSetByActiveflowID(ctx, t); errActiveflow != nil {
			return errActiveflow
		}
	}

	if t.ReferenceID != uuid.Nil {
		if errReferenceID := h.cache.OutdialTargetCallSetByReferenceID(ctx, t); errReferenceID != nil {
			return errReferenceID
		}
	}

	return nil
}

// outdialTargetCallGetFromCache returns outdialTargetCall from the cache if possible.
func (h *handler) outdialTargetCallGetFromCache(ctx context.Context, id uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	// get from cache
	res, err := h.cache.OutdialTargetCallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// outdialTargetCallGetFromCacheByActiveflowID returns outdialTargetCall from the cache if possible.
func (h *handler) outdialTargetCallGetFromCacheByActiveflowID(ctx context.Context, activeflowID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	// get from cache
	res, err := h.cache.OutdialTargetCallGetByActiveflowID(ctx, activeflowID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// outdialTargetCallGetFromCacheByReferenceID returns outdialTargetCall from the cache if possible.
func (h *handler) outdialTargetCallGetFromCacheByReferenceID(ctx context.Context, referenceID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	// get from cache
	res, err := h.cache.OutdialTargetCallGetByReferenceID(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// outdialTargetCallGetFromDB gets the outdialTargetCall info from the db.
func (h *handler) outdialTargetCallGetFromDB(ctx context.Context, id uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", outdialTargetCallSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. outdialTargetCallGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. outdialTargetCallGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.outdialTargetCallGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// OutdialTargetCallGet returns outdialtargetcall.
func (h *handler) OutdialTargetCallGet(ctx context.Context, id uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	res, err := h.outdialTargetCallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.outdialTargetCallGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.outdialTargetCallSetToCache(ctx, res)

	return res, nil
}

// OutdialTargetCallGetByReferenceID gets the outdialtargetcall by reference_id.
func (h *handler) OutdialTargetCallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	tmp, err := h.outdialTargetCallGetFromCacheByReferenceID(ctx, referenceID)
	if err == nil {
		return tmp, nil
	}

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			reference_id = ?
		order by
			tm_create desc
	`, outdialTargetCallSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. OutdialTargetCallGetByReferenceID. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, referenceID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialTargetCallGetByReferenceID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.outdialTargetCallGetFromRow(row)
	if err != nil {
		return nil, err
	}

	_ = h.outdialTargetCallSetToCache(ctx, res)

	return res, nil
}

// OutdialTargetCallGetByActiveflowID gets the outdialtargetcall by activeflow_id.
func (h *handler) OutdialTargetCallGetByActiveflowID(ctx context.Context, activeflowID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	tmp, err := h.outdialTargetCallGetFromCacheByActiveflowID(ctx, activeflowID)
	if err == nil {
		return tmp, nil
	}

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			activeflow_id = ?
		order by
			tm_create desc
	`, outdialTargetCallSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. OutdialTargetCallGetByActiveflowID. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, activeflowID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialTargetCallGetByActiveflowID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.outdialTargetCallGetFromRow(row)
	if err != nil {
		return nil, err
	}

	_ = h.outdialTargetCallSetToCache(ctx, res)

	return res, nil
}

// OutdialTargetCallGetsByOutdialIDAndStatus returns list of outdialtargetcalls.
func (h *handler) OutdialTargetCallGetsByOutdialIDAndStatus(ctx context.Context, outdialID uuid.UUID, status outdialtargetcall.Status) ([]*outdialtargetcall.OutdialTargetCall, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			outdial_id = ?
			and status = ?
		order by
			tm_create desc
	`, outdialTargetCallSelect)

	rows, err := h.db.Query(q, outdialID.Bytes(), status)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialTargetCallGetsByOutdialIDAndStatus. err: %v", err)
	}
	defer rows.Close()

	var res []*outdialtargetcall.OutdialTargetCall
	for rows.Next() {
		u, err := h.outdialTargetCallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. OutdialTargetCallGetsByOutdialIDAndStatus. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// OutdialTargetCallGetsByCampaignIDAndStatus returns list of outdialtargetcalls.
func (h *handler) OutdialTargetCallGetsByCampaignIDAndStatus(ctx context.Context, campaignID uuid.UUID, status outdialtargetcall.Status) ([]*outdialtargetcall.OutdialTargetCall, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			campaign_id = ?
			and status = ?
		order by
			tm_create desc
	`, outdialTargetCallSelect)

	rows, err := h.db.Query(q, campaignID.Bytes(), status)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialTargetCallGetsByCampaignIDAndStatus. err: %v", err)
	}
	defer rows.Close()

	var res []*outdialtargetcall.OutdialTargetCall
	for rows.Next() {
		u, err := h.outdialTargetCallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. OutdialTargetCallGetsByCampaignIDAndStatus. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}
