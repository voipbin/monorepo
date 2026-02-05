package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	fmaction "monorepo/bin-flow-manager/models/action"

	"monorepo/bin-campaign-manager/models/campaign"
)

const (
	campaignsTable = "campaign_campaigns"
)

// campaignGetFromRow gets the campaign from the row.
func (h *handler) campaignGetFromRow(row *sql.Rows) (*campaign.Campaign, error) {
	res := &campaign.Campaign{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. campaignGetFromRow. err: %v", err)
	}

	return res, nil
}

// CampaignCreate insert a new campaign record
func (h *handler) CampaignCreate(ctx context.Context, c *campaign.Campaign) error {
	now := h.util.TimeNow()

	// Set timestamps
	c.TMCreate = now
	c.TMUpdate = nil
	c.TMDelete = nil

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. CampaignCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(campaignsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CampaignCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. CampaignCreate. err: %v", err)
	}

	_ = h.campaignUpdateToCache(ctx, c.ID)

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
	fields := commondatabasehandler.GetDBFields(&campaign.Campaign{})
	query, args, err := squirrel.
		Select(fields...).
		From(campaignsTable).
		Where(squirrel.Eq{string(campaign.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. campaignGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. campaignGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. campaignGetFromDB. err: %v", err)
		}
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
	ts := h.util.TimeNow()

	fields := map[campaign.Field]any{
		campaign.FieldTMUpdate: ts,
		campaign.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("CampaignDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(campaignsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(campaign.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("CampaignDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("CampaignDelete: exec failed: %w", err)
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

// CampaignGets returns list of campaigns with filters.
func (h *handler) CampaignList(ctx context.Context, token string, size uint64, filters map[campaign.Field]any) ([]*campaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "CampaignGets",
		"size":    size,
		"token":   token,
		"filters": filters,
	})
	log.Debug("CampaignGets called with filters (check customer_id type)")

	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&campaign.Campaign{})
	sb := squirrel.
		Select(fields...).
		From(campaignsTable).
		Where(squirrel.Lt{string(campaign.FieldTMCreate): token}).
		OrderBy(string(campaign.FieldTMCreate) + " DESC", string(campaign.FieldID) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. CampaignList. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CampaignList. err: %v", err)
	}

	log.WithFields(logrus.Fields{
		"query": query,
		"args":  args,
	}).Debug("Generated SQL query (check if customer_id is binary UUID)")

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CampaignList. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*campaign.Campaign{}
	for rows.Next() {
		u, err := h.campaignGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. CampaignGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. CampaignList. err: %v", err)
	}

	return res, nil
}

// CampaignGetsByCustomerID returns list of campaigns.
func (h *handler) CampaignListByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*campaign.Campaign, error) {
	filters := map[campaign.Field]any{
		campaign.FieldCustomerID: customerID,
		campaign.FieldDeleted:    false,
	}

	// Debug log to verify customerID type in filters
	log := logrus.WithFields(logrus.Fields{
		"func":             "CampaignGetsByCustomerID",
		"customer_id":      customerID,
		"customer_id_type": fmt.Sprintf("%T", customerID),
		"filters":          filters,
	})
	log.Debug("Created filters with customer_id (check UUID type)")

	return h.CampaignList(ctx, token, limit, filters)
}

// CampaignUpdate updates campaign fields.
func (h *handler) CampaignUpdate(ctx context.Context, id uuid.UUID, fields map[campaign.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[campaign.FieldTMUpdate] = h.util.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("CampaignUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(campaignsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(campaign.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("CampaignUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.Exec(sqlStr, args...); err != nil {
		return fmt.Errorf("CampaignUpdate: exec failed: %w", err)
	}

	_ = h.campaignUpdateToCache(ctx, id)
	return nil
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
	fields := map[campaign.Field]any{
		campaign.FieldName:         name,
		campaign.FieldDetail:       detail,
		campaign.FieldType:         campaignType,
		campaign.FieldServiceLevel: serviceLevel,
		campaign.FieldEndHandle:    endHandle,
	}

	return h.CampaignUpdate(ctx, id, fields)
}

// CampaignUpdateResourceInfo updates campaign's resource information.
func (h *handler) CampaignUpdateResourceInfo(ctx context.Context, id, outplanID, outdialID, queueID, nextCampaignID uuid.UUID) error {
	fields := map[campaign.Field]any{
		campaign.FieldOutplanID:      outplanID,
		campaign.FieldOutdialID:      outdialID,
		campaign.FieldQueueID:        queueID,
		campaign.FieldNextCampaignID: nextCampaignID,
	}

	return h.CampaignUpdate(ctx, id, fields)
}

// CampaignUpdateNextCampaignID updates campaign's next_campaign_id information.
func (h *handler) CampaignUpdateNextCampaignID(ctx context.Context, id, nextCampaignID uuid.UUID) error {
	fields := map[campaign.Field]any{
		campaign.FieldNextCampaignID: nextCampaignID,
	}

	return h.CampaignUpdate(ctx, id, fields)
}

// CampaignUpdateStatus updates campaign's status.
func (h *handler) CampaignUpdateStatus(ctx context.Context, id uuid.UUID, status campaign.Status) error {
	fields := map[campaign.Field]any{
		campaign.FieldStatus: status,
	}

	return h.CampaignUpdate(ctx, id, fields)
}

// CampaignUpdateStatusAndExecute updates campaign's status and execute.
func (h *handler) CampaignUpdateStatusAndExecute(ctx context.Context, id uuid.UUID, status campaign.Status, execute campaign.Execute) error {
	fields := map[campaign.Field]any{
		campaign.FieldStatus:  status,
		campaign.FieldExecute: execute,
	}

	return h.CampaignUpdate(ctx, id, fields)
}

// CampaignUpdateExecute updates campaign's execute.
func (h *handler) CampaignUpdateExecute(ctx context.Context, id uuid.UUID, execute campaign.Execute) error {
	fields := map[campaign.Field]any{
		campaign.FieldExecute: execute,
	}

	return h.CampaignUpdate(ctx, id, fields)
}

// CampaignUpdateServiceLevel updates campaign's service_level.
func (h *handler) CampaignUpdateServiceLevel(ctx context.Context, id uuid.UUID, serviceLevel int) error {
	fields := map[campaign.Field]any{
		campaign.FieldServiceLevel: serviceLevel,
	}

	return h.CampaignUpdate(ctx, id, fields)
}

// CampaignUpdateEndHandle updates campaign's end_handle.
func (h *handler) CampaignUpdateEndHandle(ctx context.Context, id uuid.UUID, endHandle campaign.EndHandle) error {
	fields := map[campaign.Field]any{
		campaign.FieldEndHandle: endHandle,
	}

	return h.CampaignUpdate(ctx, id, fields)
}

// CampaignUpdateActions updates campaign's actions.
func (h *handler) CampaignUpdateActions(ctx context.Context, id uuid.UUID, actions []fmaction.Action) error {
	fields := map[campaign.Field]any{
		campaign.FieldActions: actions,
	}

	return h.CampaignUpdate(ctx, id, fields)
}

// CampaignUpdateType updates campaign's type.
func (h *handler) CampaignUpdateType(ctx context.Context, id uuid.UUID, campaignType campaign.Type) error {
	fields := map[campaign.Field]any{
		campaign.FieldType: campaignType,
	}

	return h.CampaignUpdate(ctx, id, fields)
}
