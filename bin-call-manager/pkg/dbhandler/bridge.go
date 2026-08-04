package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-call-manager/models/bridge"
)

var (
	bridgeTable = "call_bridges"
)

// bridgeGetFromRow gets the record from the row.
func (h *handler) bridgeGetFromRow(row *sql.Rows) (*bridge.Bridge, error) {
	res := &bridge.Bridge{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. bridgeGetFromRow. err: %v", err)
	}

	// Initialize nil slices to empty.
	// R-A read table (normalization.go): bridge.ChannelIDs normalizes nil -> [].
	// ScanRow's copyJSON leaves the field nil on SQL NULL or empty, so this must
	// stay here to preserve the pre-migration read behavior.
	if res.ChannelIDs == nil {
		res.ChannelIDs = []string{}
	}

	return res, nil
}

// BridgeCreate creates a new bridge record.
func (h *handler) BridgeCreate(ctx context.Context, b *bridge.Bridge) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps, matching the pre-migration INSERT which wrote
	// tm_create = now and left tm_update / tm_delete NULL. Struct-based
	// PrepareFields emits every db-tagged field unconditionally, so these must
	// be assigned explicitly or tm_create would be written as SQL NULL.
	b.TMCreate = now
	b.TMUpdate = nil
	b.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(b)
	if err != nil {
		return fmt.Errorf("could not prepare fields. BridgeCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(bridgeTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. BridgeCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute. BridgeCreate. err: %v", err)
	}

	// update the cache
	_ = h.bridgeUpdateToCache(ctx, b.ID)

	return nil
}

// bridgeGetFromDB returns bridge from the DB.
func (h *handler) bridgeGetFromDB(ctx context.Context, id string) (*bridge.Bridge, error) {
	fields := commondatabasehandler.GetDBFields(&bridge.Bridge{})
	query, args, err := squirrel.
		Select(fields...).
		From(bridgeTable).
		Where(squirrel.Eq{string(bridge.FieldID): id}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. bridgeGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. bridgeGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. bridgeGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.bridgeGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data. bridgeGetFromDB. err: %v", err)
	}

	return res, nil
}

// bridgeUpdateToCache gets the bridge from the DB and update the cache.
func (h *handler) bridgeUpdateToCache(ctx context.Context, id string) error {

	res, err := h.bridgeGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.bridgeSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// bridgeSetToCache sets the given bridge to the cache
func (h *handler) bridgeSetToCache(ctx context.Context, bridge *bridge.Bridge) error {
	if err := h.cache.BridgeSet(ctx, bridge); err != nil {
		return err
	}

	return nil
}

// bridgeGetFromCache returns bridge from the cache.
func (h *handler) bridgeGetFromCache(ctx context.Context, id string) (*bridge.Bridge, error) {

	// get from cache
	res, err := h.cache.BridgeGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// BridgeGet returns bridge.
func (h *handler) BridgeGet(ctx context.Context, id string) (*bridge.Bridge, error) {

	res, err := h.bridgeGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.bridgeGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.bridgeSetToCache(ctx, res)

	return res, nil
}

// BridgeEnd updates the bridge end.
func (h *handler) BridgeEnd(ctx context.Context, id string) error {
	ts := h.utilHandler.TimeNow()

	q, args, err := squirrel.
		Update(bridgeTable).
		Set(string(bridge.FieldTMUpdate), ts).
		Set(string(bridge.FieldTMDelete), ts).
		Where(squirrel.Eq{string(bridge.FieldID): id}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. BridgeEnd. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("could not execute. BridgeEnd. err: %v", err)
	}

	// update the cache
	_ = h.bridgeUpdateToCache(ctx, id)

	return nil
}

// buildBridgeAddChannelID builds the BridgeAddChannelID statement.
//
// WHY squirrel.Expr: squirrel has no builder form for MySQL JSON functions —
// json_array_append() must be emitted as a raw expression in Set() position.
// Sanctioned by docs/conventions/database.md §7.1 (computed expressions
// squirrel cannot express). Precedent: bin-agent-manager/pkg/dbhandler/agent.go:709
// (MySQL JSON function + ? arg + PlaceholderFormat(Question)) and
// bin-rag-manager/pkg/dbhandler/document.go:355 (Expr inside Set/SetMap).
//
// Extracted from the method body so the golden ToSql() test (plan §5 R1) can
// assert the generated SQL and arg order against the real builder rather than
// a copy of it — these JSON-function sites have no other automated coverage,
// since SQLite (the unit-test DB) does not implement json_array_append.
func buildBridgeAddChannelID(id, channelID string, tmUpdate any) squirrel.UpdateBuilder {
	return squirrel.
		Update(bridgeTable).
		Set(string(bridge.FieldChannelIDs), exprJSONArrayAppend(string(bridge.FieldChannelIDs), channelID)).
		Set(string(bridge.FieldTMUpdate), tmUpdate).
		Where(squirrel.Eq{string(bridge.FieldID): id}).
		PlaceholderFormat(squirrel.Question)
}

// BridgeAddChannel adds the channel to the bridge.
func (h *handler) BridgeAddChannelID(ctx context.Context, id, channelID string) error {
	q, args, err := buildBridgeAddChannelID(id, channelID, h.utilHandler.TimeNow()).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. BridgeAddChannelID. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("could not execute. BridgeAddChannelID. err: %v", err)
	}

	// update the cache
	_ = h.bridgeUpdateToCache(ctx, id)

	return nil
}

// buildBridgeRemoveChannelID builds the BridgeRemoveChannelID statement.
//
// WHY squirrel.Expr: squirrel has no builder form for MySQL JSON functions.
// This site nests json_remove/replace/json_search to delete an array element by
// value, which has no builder equivalent at all. Sanctioned by
// docs/conventions/database.md §7.1. Precedent:
// bin-agent-manager/pkg/dbhandler/agent.go:709 (MySQL JSON function + ? arg +
// PlaceholderFormat(Question)).
func buildBridgeRemoveChannelID(id, channelID string, tmUpdate any) squirrel.UpdateBuilder {
	return squirrel.
		Update(bridgeTable).
		Set(
			string(bridge.FieldChannelIDs),
			exprJSONArrayRemoveByValue(string(bridge.FieldChannelIDs), channelID),
		).
		Set(string(bridge.FieldTMUpdate), tmUpdate).
		Where(squirrel.Eq{string(bridge.FieldID): id}).
		PlaceholderFormat(squirrel.Question)
}

// BridgeRemoveChannelID removes the channel from the bridge.
func (h *handler) BridgeRemoveChannelID(ctx context.Context, id, channelID string) error {
	q, args, err := buildBridgeRemoveChannelID(id, channelID, h.utilHandler.TimeNow()).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. BridgeRemoveChannelID. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("could not execute. BridgeRemoveChannelID. err: %v", err)
	}

	// update the cache
	_ = h.bridgeUpdateToCache(ctx, id)

	return nil
}
