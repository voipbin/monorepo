package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-campaign-manager/models/campaigncall"
)

const (
	// select query for campaigncall get
	campaigncallSelect = `
	select
		id,
		customer_id,
		campaign_id,

		outplan_id,
		outdial_id,
		outdial_target_id,
		queue_id,

		activeflow_id,
		flow_id,

		reference_type,
		reference_id,

		status,
		result,

		source,
		destination,
		destination_index,
		try_count,

		tm_create,
		tm_update,
		tm_delete
	from
		campaign_campaigncalls
	`
)

// campaigncallGetFromRow gets the campaigncall from the row.
func (h *handler) campaigncallGetFromRow(row *sql.Rows) (*campaigncall.Campaigncall, error) {

	var source string
	var destination string

	res := &campaigncall.Campaigncall{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.CampaignID,

		&res.OutplanID,
		&res.OutdialID,
		&res.OutdialTargetID,
		&res.QueueID,

		&res.ActiveflowID,
		&res.FlowID,

		&res.ReferenceType,
		&res.ReferenceID,

		&res.Status,
		&res.Result,

		&source,
		&destination,
		&res.DestinationIndex,
		&res.TryCount,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. campaignGetFromRow. err: %v", err)
	}

	if errSource := json.Unmarshal([]byte(source), &res.Source); errSource != nil {
		return nil, fmt.Errorf("could not unmarshal the source. campaignGetFromRow. err: %v", errSource)
	}

	if errDestination := json.Unmarshal([]byte(destination), &res.Destination); errDestination != nil {
		return nil, fmt.Errorf("could not unmarshal the destination. campaignGetFromRow. err: %v", errDestination)
	}

	return res, nil
}

// CampaigncallCreate insert a new campaigncall record
func (h *handler) CampaigncallCreate(ctx context.Context, t *campaigncall.Campaigncall) error {
	q := `insert into campaign_campaigncalls(
		id,
		customer_id,
		campaign_id,

		outplan_id,
		outdial_id,
		outdial_target_id,
		queue_id,

		activeflow_id,
		flow_id,

		reference_type,
		reference_id,

		status,
		result,

		source,
		destination,
		destination_index,
		try_count,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?, ?, ?, ?,
		?, ?,
		?, ?,
		?, ?,
		?, ?, ?, ?,
		?, ?, ?
	)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. CampaigncallCreate. err: %v", err)
	}
	defer stmt.Close()

	source, err := json.Marshal(t.Source)
	if err != nil {
		return fmt.Errorf("could not marshal source. CampaigncallCreate. err: %v", err)
	}

	destination, err := json.Marshal(t.Destination)
	if err != nil {
		return fmt.Errorf("could not marshal destination. CampaigncallCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		t.ID.Bytes(),
		t.CustomerID.Bytes(),
		t.CampaignID.Bytes(),

		t.OutplanID.Bytes(),
		t.OutdialID.Bytes(),
		t.OutdialTargetID.Bytes(),
		t.QueueID.Bytes(),

		t.ActiveflowID.Bytes(),
		t.FlowID.Bytes(),

		t.ReferenceType,
		t.ReferenceID.Bytes(),

		t.Status,
		t.Result,

		source,
		destination,
		t.DestinationIndex,
		t.TryCount,

		h.util.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. CampaigncallCreate. err: %v", err)
	}

	_ = h.campaigncallUpdateToCache(ctx, t.ID)

	return nil
}

// campaigncallUpdateToCache gets the campaigncall from the DB and update the cache.
func (h *handler) campaigncallUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.campaigncallGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.campaigncallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// campaigncallSetToCache sets the given campaigncall to the cache
func (h *handler) campaigncallSetToCache(ctx context.Context, f *campaigncall.Campaigncall) error {
	if err := h.cache.CampaigncallSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// campaigncallGetFromCache returns campaigncall from the cache if possible.
func (h *handler) campaigncallGetFromCache(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error) {

	// get from cache
	res, err := h.cache.CampaigncallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// campaigncallGetFromDB gets the campaigncall info from the db.
func (h *handler) campaigncallGetFromDB(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", campaigncallSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. campaigncallGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. campaigncallGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.campaigncallGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// CampaigncallGet returns campaigncall.
func (h *handler) CampaigncallGet(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error) {

	res, err := h.campaigncallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.campaigncallGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.campaigncallSetToCache(ctx, res)

	return res, nil
}

// CampaigncallGetByReferenceID returns campaigncall of the reference_id.
func (h *handler) CampaigncallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*campaigncall.Campaigncall, error) {

	// prepare
	q := fmt.Sprintf("%s where reference_id = ?", campaigncallSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. CampaigncallGetByReferenceID. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, referenceID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. CampaigncallGetByReferenceID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.campaigncallGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// CampaigncallGetByActiveflowID returns campaigncall of the activeflow_id.
func (h *handler) CampaigncallGetByActiveflowID(ctx context.Context, activeflowID uuid.UUID) (*campaigncall.Campaigncall, error) {

	// prepare
	q := fmt.Sprintf("%s where activeflow_id = ?", campaigncallSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. CampaigncallGetByReferenceID. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, activeflowID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. CampaigncallGetByReferenceID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.campaigncallGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// CampaigncallGetsByCustomerID returns list of campaigncall.
func (h *handler) CampaigncallGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			customer_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, campaigncallSelect)

	rows, err := h.db.Query(q, customerID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. CampaigncallGetsByCustomerID. err: %v", err)
	}
	defer rows.Close()

	var res []*campaigncall.Campaigncall
	for rows.Next() {
		u, err := h.campaigncallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. CampaigncallGetsByCustomerID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// CampaigncallGetsByCampaignID returns list of campaigncall.
func (h *handler) CampaigncallGetsByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			campaign_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, campaigncallSelect)

	rows, err := h.db.Query(q, campaignID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. CampaigncallGetsByCampaignID. err: %v", err)
	}
	defer rows.Close()

	var res []*campaigncall.Campaigncall
	for rows.Next() {
		u, err := h.campaigncallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. CampaigncallGetsByCampaignID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// CampaigncallGetsByCampaignIDAndStatus returns list of campaigncall.
func (h *handler) CampaigncallGetsByCampaignIDAndStatus(ctx context.Context, campaignID uuid.UUID, status campaigncall.Status, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			status = ?
			and campaign_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, campaigncallSelect)

	rows, err := h.db.Query(q, status, campaignID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. CampaigncallGetsByCampaignIDAndStatus. err: %v", err)
	}
	defer rows.Close()

	var res []*campaigncall.Campaigncall
	for rows.Next() {
		u, err := h.campaigncallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. CampaigncallGetsByCampaignIDAndStatus. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// CampaigncallGetsByCampaignIDAndStatusOngoing returns list of campaigncall.
func (h *handler) CampaigncallGetsOngoingByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			(status = ? or status = ?)
			and campaign_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, campaigncallSelect)

	rows, err := h.db.Query(q, campaigncall.StatusDialing, campaigncall.StatusProgressing, campaignID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. CampaigncallGetsOngoingByCampaignID. err: %v", err)
	}
	defer rows.Close()

	var res []*campaigncall.Campaigncall
	for rows.Next() {
		u, err := h.campaigncallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. CampaigncallGetsOngoingByCampaignID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// CampaigncallUpdateStatus updates campaigncall's status.
func (h *handler) CampaigncallUpdateStatus(ctx context.Context, id uuid.UUID, status campaigncall.Status) error {
	q := `
	update campaign_campaigncalls set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, status, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaigncallUpdateStatus. err: %v", err)
	}

	// set to the cache
	_ = h.campaigncallUpdateToCache(ctx, id)

	return nil
}

// CampaigncallUpdateStatusAndResult updates campaigncall's status and result.
func (h *handler) CampaigncallUpdateStatusAndResult(ctx context.Context, id uuid.UUID, status campaigncall.Status, result campaigncall.Result) error {
	q := `
	update campaign_campaigncalls set
		result = ?,
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, result, status, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaigncallUpdateStatusAndResult. err: %v", err)
	}

	// set to the cache
	_ = h.campaigncallUpdateToCache(ctx, id)

	return nil
}

// CampaigncallDelete deletes the given campaigncall
func (h *handler) CampaigncallDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update campaign_campaigncalls set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.util.TimeGetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaigncallDelete. err: %v", err)
	}

	// update cache
	_ = h.campaigncallUpdateToCache(ctx, id)

	return nil
}
