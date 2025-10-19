package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"

	"monorepo/bin-campaign-manager/models/campaign"
)

const (
	// select query for campaign get
	campaignSelect = `
	select
		id,
		customer_id,

		type,

		execute,

		name,
		detail,

		status,
		service_level,
		end_handle,

		flow_id,
		actions,

		outplan_id,
		outdial_id,
		queue_id,

		next_campaign_id,

		tm_create,
		tm_update,
		tm_delete
	from
		campaign_campaigns
	`
)

// campaignGetFromRow gets the campaign from the row.
func (h *handler) campaignGetFromRow(row *sql.Rows) (*campaign.Campaign, error) {

	var actions string

	res := &campaign.Campaign{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Type,

		&res.Execute,

		&res.Name,
		&res.Detail,

		&res.Status,
		&res.ServiceLevel,
		&res.EndHandle,

		&res.FlowID,
		&actions,

		&res.OutplanID,
		&res.OutdialID,
		&res.QueueID,

		&res.NextCampaignID,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. campaignGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(actions), &res.Actions); err != nil {
		return nil, fmt.Errorf("could not unmarshal the action. campaignGetFromRow. err: %v", err)
	}

	return res, nil
}

// CampaignCreate insert a new campaign record
func (h *handler) CampaignCreate(ctx context.Context, t *campaign.Campaign) error {
	q := `insert into campaign_campaigns(
		id,
		customer_id,

		type,

		execute,

		name,
		detail,

		status,
		service_level,
		end_handle,

		flow_id,
		actions,

		outplan_id,
		outdial_id,
		queue_id,

		next_campaign_id,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?,
		?,
		?, ?,
		?, ?, ?,
		?, ?,
		?, ?, ?,
		?,
		?, ?, ?
	)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. CampaignCreate. err: %v", err)
	}
	defer stmt.Close()

	actions, err := json.Marshal(t.Actions)
	if err != nil {
		return fmt.Errorf("could not marshal the action. CampaignCreate. err: %v", err)
	}

	_, err = stmt.ExecContext(ctx,
		t.ID.Bytes(),
		t.CustomerID.Bytes(),

		t.Type,

		t.Execute,

		t.Name,
		t.Detail,

		t.Status,
		t.ServiceLevel,
		t.EndHandle,

		t.FlowID.Bytes(),
		actions,

		t.OutplanID.Bytes(),
		t.OutdialID.Bytes(),
		t.QueueID.Bytes(),

		t.NextCampaignID.Bytes(),

		h.util.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. CampaignCreate. err: %v", err)
	}

	_ = h.campaignUpdateToCache(ctx, t.ID)

	return nil
}

// campaignUpdateToCache gets the campaign from the DB and update the cache.
func (h *handler) campaignUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.campaignGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.campaignSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// campaignSetToCache sets the given campaign to the cache
func (h *handler) campaignSetToCache(ctx context.Context, f *campaign.Campaign) error {
	if err := h.cache.CampaignSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// campaignGetFromCache returns campaign from the cache if possible.
func (h *handler) campaignGetFromCache(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {

	// get from cache
	res, err := h.cache.CampaignGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// campaignGetFromDB gets the campaign info from the db.
func (h *handler) campaignGetFromDB(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", campaignSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. campaignGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. campaignGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.campaignGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// CampaignDelete deletes the given campaign
func (h *handler) CampaignDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update campaign_campaigns set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.util.TimeGetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaignDelete. err: %v", err)
	}

	// update cache
	_ = h.campaignUpdateToCache(ctx, id)

	return nil
}

// CampaignGet returns campaign.
func (h *handler) CampaignGet(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {

	res, err := h.campaignGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.campaignGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.campaignSetToCache(ctx, res)

	return res, nil
}

// CampaignGetsByCustomerID returns list of campaigns.
func (h *handler) CampaignGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*campaign.Campaign, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and customer_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, campaignSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. CampaignGetsByCustomerID. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var res []*campaign.Campaign
	for rows.Next() {
		u, err := h.campaignGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. CampaignGetsByCustomerID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// CampaignUpdateBasicInfo updates campaign's basic information.
func (h *handler) CampaignUpdateBasicInfo(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	campaignType campaign.Type,
	serviceLevel int,
	endHandle campaign.EndHandle,
) error {
	q := `
	update campaign_campaigns set
		name = ?,
		detail = ?,
		type = ?,
		service_level = ?,
		end_handle = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, name, detail, campaignType, serviceLevel, endHandle, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaignUpdateBasicInfo. err: %v", err)
	}

	// set to the cache
	_ = h.campaignUpdateToCache(ctx, id)

	return nil
}

// CampaignUpdateResourceInfo updates campaign's resource information.
func (h *handler) CampaignUpdateResourceInfo(ctx context.Context, id, outplanID, outdialID, queueID, nextCampaignID uuid.UUID) error {
	q := `
	update campaign_campaigns set
		outplan_id = ?,
		outdial_id = ?,
		queue_id = ?,
		next_campaign_id = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, outplanID.Bytes(), outdialID.Bytes(), queueID.Bytes(), nextCampaignID.Bytes(), h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaignUpdateResourceInfo. err: %v", err)
	}

	// set to the cache
	_ = h.campaignUpdateToCache(ctx, id)

	return nil
}

// CampaignUpdateNextCampaignID updates campaign's next_campaign_id information.
func (h *handler) CampaignUpdateNextCampaignID(ctx context.Context, id, nextCampaignID uuid.UUID) error {
	q := `
	update campaign_campaigns set
		next_campaign_id = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, nextCampaignID.Bytes(), h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaignUpdateNextCampaignID. err: %v", err)
	}

	// set to the cache
	_ = h.campaignUpdateToCache(ctx, id)

	return nil
}

// CampaignUpdateStatus updates campaign's status.
func (h *handler) CampaignUpdateStatus(ctx context.Context, id uuid.UUID, status campaign.Status) error {
	q := `
	update campaign_campaigns set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, status, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaignUpdateStatus. err: %v", err)
	}

	// set to the cache
	_ = h.campaignUpdateToCache(ctx, id)

	return nil
}

// CampaignUpdateStatusAndExecute updates campaign's status and execute.
func (h *handler) CampaignUpdateStatusAndExecute(ctx context.Context, id uuid.UUID, status campaign.Status, execute campaign.Execute) error {
	q := `
	update campaign_campaigns set
		status = ?,
		execute = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, status, execute, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaignUpdateStatusAndExecute. err: %v", err)
	}

	// set to the cache
	_ = h.campaignUpdateToCache(ctx, id)

	return nil
}

// CampaignUpdateExecute updates campaign's execute.
func (h *handler) CampaignUpdateExecute(ctx context.Context, id uuid.UUID, execute campaign.Execute) error {
	q := `
	update campaign_campaigns set
		execute = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, execute, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaignUpdateExecute. err: %v", err)
	}

	// set to the cache
	_ = h.campaignUpdateToCache(ctx, id)

	return nil
}

// CampaignUpdateServiceLevel updates campaign's service_level.
func (h *handler) CampaignUpdateServiceLevel(ctx context.Context, id uuid.UUID, serviceLevel int) error {
	q := `
	update campaign_campaigns set
		service_level = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, serviceLevel, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaignUpdateServiceLevel. err: %v", err)
	}

	// set to the cache
	_ = h.campaignUpdateToCache(ctx, id)

	return nil
}

// CampaignUpdateEndHandle updates campaign's end_handle.
func (h *handler) CampaignUpdateEndHandle(ctx context.Context, id uuid.UUID, endHandle campaign.EndHandle) error {
	q := `
	update campaign_campaigns set
		end_handle = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, endHandle, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaignUpdateEndHandle. err: %v", err)
	}

	// set to the cache
	_ = h.campaignUpdateToCache(ctx, id)

	return nil
}

// CampaignUpdateActions updates campaign's actions.
func (h *handler) CampaignUpdateActions(ctx context.Context, id uuid.UUID, actions []fmaction.Action) error {
	q := `
	update campaign_campaigns set
		actions = ?,
		tm_update = ?
	where
		id = ?
	`

	tmpActions, err := json.Marshal(actions)
	if err != nil {
		return fmt.Errorf("could not marshal the actions. CampaignUpdateActions. err: %v", err)
	}

	if _, err := h.db.Exec(q, tmpActions, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaignUpdateActions. err: %v", err)
	}

	// set to the cache
	_ = h.campaignUpdateToCache(ctx, id)

	return nil
}

// CampaignUpdateType updates campaign's type.
func (h *handler) CampaignUpdateType(ctx context.Context, id uuid.UUID, campaignType campaign.Type) error {
	q := `
	update campaign_campaigns set
		type = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, campaignType, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. CampaignUpdateType. err: %v", err)
	}

	// set to the cache
	_ = h.campaignUpdateToCache(ctx, id)

	return nil
}
