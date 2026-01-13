package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-campaign-manager/models/campaigncall"
)

const (
	campaigncallsTable = "campaign_campaigncalls"
)

// campaigncallGetFromRow gets the campaigncall from the row.
func (h *handler) campaigncallGetFromRow(row *sql.Rows) (*campaigncall.Campaigncall, error) {
	res := &campaigncall.Campaigncall{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. campaigncallGetFromRow. err: %v", err)
	}

	return res, nil
}

// CampaigncallCreate insert a new campaigncall record
func (h *handler) CampaigncallCreate(ctx context.Context, c *campaigncall.Campaigncall) error {
	now := h.util.TimeGetCurTime()

	// Set timestamps
	c.TMCreate = now
	c.TMUpdate = commondatabasehandler.DefaultTimeStamp
	c.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. CampaigncallCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(campaigncallsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. CampaigncallCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. CampaigncallCreate. err: %v", err)
	}

	_ = h.campaigncallUpdateToCache(ctx, c.ID)

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
	fields := commondatabasehandler.GetDBFields(&campaigncall.Campaigncall{})
	query, args, err := squirrel.
		Select(fields...).
		From(campaigncallsTable).
		Where(squirrel.Eq{string(campaigncall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. campaigncallGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. campaigncallGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. campaigncallGetFromDB. err: %v", err)
		}
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
	fields := commondatabasehandler.GetDBFields(&campaigncall.Campaigncall{})
	query, args, err := squirrel.
		Select(fields...).
		From(campaigncallsTable).
		Where(squirrel.Eq{string(campaigncall.FieldReferenceID): referenceID.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. CampaigncallGetByReferenceID. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CampaigncallGetByReferenceID. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. CampaigncallGetByReferenceID. err: %v", err)
		}
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
	fields := commondatabasehandler.GetDBFields(&campaigncall.Campaigncall{})
	query, args, err := squirrel.
		Select(fields...).
		From(campaigncallsTable).
		Where(squirrel.Eq{string(campaigncall.FieldActiveflowID): activeflowID.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. CampaigncallGetByActiveflowID. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CampaigncallGetByActiveflowID. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. CampaigncallGetByActiveflowID. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.campaigncallGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// CampaigncallGets returns list of campaigncalls with filters.
func (h *handler) CampaigncallGets(ctx context.Context, token string, size uint64, filters map[campaigncall.Field]any) ([]*campaigncall.Campaigncall, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&campaigncall.Campaigncall{})
	sb := squirrel.
		Select(fields...).
		From(campaigncallsTable).
		Where(squirrel.Lt{string(campaigncall.FieldTMCreate): token}).
		OrderBy(string(campaigncall.FieldTMCreate) + " DESC", string(campaigncall.FieldID) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. CampaigncallGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CampaigncallGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CampaigncallGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*campaigncall.Campaigncall{}
	for rows.Next() {
		u, err := h.campaigncallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. CampaigncallGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. CampaigncallGets. err: %v", err)
	}

	return res, nil
}

// CampaigncallGetsByCustomerID returns list of campaigncall.
func (h *handler) CampaigncallGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {
	filters := map[campaigncall.Field]any{
		campaigncall.FieldCustomerID: customerID,
	}

	return h.CampaigncallGets(ctx, token, limit, filters)
}

// CampaigncallGetsByCampaignID returns list of campaigncall.
func (h *handler) CampaigncallGetsByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {
	filters := map[campaigncall.Field]any{
		campaigncall.FieldCampaignID: campaignID,
	}

	return h.CampaigncallGets(ctx, token, limit, filters)
}

// CampaigncallGetsByCampaignIDAndStatus returns list of campaigncall.
func (h *handler) CampaigncallGetsByCampaignIDAndStatus(ctx context.Context, campaignID uuid.UUID, status campaigncall.Status, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {
	filters := map[campaigncall.Field]any{
		campaigncall.FieldCampaignID: campaignID,
		campaigncall.FieldStatus:     status,
	}

	return h.CampaigncallGets(ctx, token, limit, filters)
}

// CampaigncallGetsOngoingByCampaignID returns list of ongoing campaigncalls (dialing or progressing).
func (h *handler) CampaigncallGetsOngoingByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&campaigncall.Campaigncall{})
	sb := squirrel.
		Select(fields...).
		From(campaigncallsTable).
		Where(squirrel.Eq{string(campaigncall.FieldCampaignID): campaignID.Bytes()}).
		Where(squirrel.Or{
			squirrel.Eq{string(campaigncall.FieldStatus): campaigncall.StatusDialing},
			squirrel.Eq{string(campaigncall.FieldStatus): campaigncall.StatusProgressing},
		}).
		Where(squirrel.Lt{string(campaigncall.FieldTMCreate): token}).
		OrderBy(string(campaigncall.FieldTMCreate) + " DESC", string(campaigncall.FieldID) + " DESC").
		Limit(limit).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. CampaigncallGetsOngoingByCampaignID. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. CampaigncallGetsOngoingByCampaignID. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*campaigncall.Campaigncall{}
	for rows.Next() {
		u, err := h.campaigncallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. CampaigncallGetsOngoingByCampaignID, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. CampaigncallGetsOngoingByCampaignID. err: %v", err)
	}

	return res, nil
}

// CampaigncallUpdate updates campaigncall fields.
func (h *handler) CampaigncallUpdate(ctx context.Context, id uuid.UUID, fields map[campaigncall.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[campaigncall.FieldTMUpdate] = h.util.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("CampaigncallUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(campaigncallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(campaigncall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("CampaigncallUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.Exec(sqlStr, args...); err != nil {
		return fmt.Errorf("CampaigncallUpdate: exec failed: %w", err)
	}

	_ = h.campaigncallUpdateToCache(ctx, id)
	return nil
}

// CampaigncallUpdateStatus updates campaigncall's status.
func (h *handler) CampaigncallUpdateStatus(ctx context.Context, id uuid.UUID, status campaigncall.Status) error {
	fields := map[campaigncall.Field]any{
		campaigncall.FieldStatus: status,
	}

	return h.CampaigncallUpdate(ctx, id, fields)
}

// CampaigncallUpdateStatusAndResult updates campaigncall's status and result.
func (h *handler) CampaigncallUpdateStatusAndResult(ctx context.Context, id uuid.UUID, status campaigncall.Status, result campaigncall.Result) error {
	fields := map[campaigncall.Field]any{
		campaigncall.FieldStatus: status,
		campaigncall.FieldResult: result,
	}

	return h.CampaigncallUpdate(ctx, id, fields)
}

// CampaigncallDelete deletes the given campaigncall
func (h *handler) CampaigncallDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.util.TimeGetCurTime()

	fields := map[campaigncall.Field]any{
		campaigncall.FieldTMUpdate: ts,
		campaigncall.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("CampaigncallDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(campaigncallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(campaigncall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("CampaigncallDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("CampaigncallDelete: exec failed: %w", err)
	}

	// update cache
	_ = h.campaigncallUpdateToCache(ctx, id)

	return nil
}
