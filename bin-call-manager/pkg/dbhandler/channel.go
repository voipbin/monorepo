package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/pkg/errors"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

var (
	channelTable = "call_channels"
)

// channelGetFromRow gets the channel from the row.
func (h *handler) channelGetFromRow(row *sql.Rows) (*channel.Channel, error) {
	res := &channel.Channel{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("channelGetFromRow: Could not scan the row. err: %v", err)
	}

	// Initialize nil maps to empty, per the R-A read table in normalization.go.
	// ScanRow's copyJSON leaves a field nil on SQL NULL or empty, so the
	// pre-migration normalization has to stay here.
	//
	// Data and StasisData normalize; SIPData deliberately does NOT — it is left
	// nil today (channel.go had no nil guard for it) and must stay that way.
	// Do not "tidy" this into a uniform rule for all three.
	if res.Data == nil {
		res.Data = map[string]interface{}{}
	}
	if res.StasisData == nil {
		res.StasisData = map[channel.StasisDataType]string{}
	}

	return res, nil
}

// channelUpdate updates channel fields using a typed field map.
//
// This collapses the ~14 near-identical single-column setters below, each of
// which previously carried its own hand-written UPDATE statement.
// tm_update defaults to now unless the caller supplied it explicitly (some
// setters write tm_update and a second timestamp column from one clock read).
func (h *handler) channelUpdate(ctx context.Context, id string, fields map[channel.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	if _, ok := fields[channel.FieldTMUpdate]; !ok {
		fields[channel.FieldTMUpdate] = h.utilHandler.TimeNow()
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("channelUpdate: prepare fields failed: %w", err)
	}

	q, args, err := squirrel.
		Update(channelTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(channel.FieldID): id}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return fmt.Errorf("channelUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("channelUpdate: exec failed: %w", err)
	}

	_ = h.channelUpdateToCache(ctx, id)
	return nil
}

// ChannelCreate creates new channel record and returns the created channel record.
func (h *handler) ChannelCreate(ctx context.Context, c *channel.Channel) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps, matching the pre-migration INSERT which wrote
	// tm_create = now and left every other timestamp NULL. Struct-based
	// PrepareFields emits every db-tagged field unconditionally, so these must
	// be assigned explicitly or tm_create would be written as SQL NULL.
	c.TMCreate = now
	c.TMUpdate = nil
	c.TMDelete = nil
	c.TMAnswer = nil
	c.TMRinging = nil
	c.TMEnd = nil

	// R-A write table: Data / SIPData / StasisData are NOT normalized at this
	// call site. They are map-typed, so a nil map still marshals to the JSON
	// literal "null" here exactly as the pre-migration json.Marshal did.
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("ChannelCreate: could not prepare fields. err: %v", err)
	}

	query, args, err := squirrel.
		Insert(channelTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return fmt.Errorf("ChannelCreate: could not build query. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("ChannelCreate: Could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, c.ID)

	return nil
}

// ChannelGet returns channel.
func (h *handler) ChannelGet(ctx context.Context, id string) (*channel.Channel, error) {

	res, err := h.channelGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.channelGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.channelSetToCache(ctx, res)

	return res, nil
}

// ChannelSetData sets the data
func (h *handler) ChannelSetData(ctx context.Context, id string, data map[string]interface{}) error {
	// R-A write table: Data is NOT normalized at this call site; a nil map
	// marshals to the JSON literal "null", same as before.
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldData: data,
	})
}

// buildChannelSetDataItem builds the ChannelSetDataItem statement.
//
// WHY squirrel.Expr: MySQL json_set has no squirrel builder form; see
// json_expr.go for the shared rationale and precedent citations.
//
// SECURITY (plan §6 item 1): the pre-migration statement interpolated the key
// straight into the SQL text via fmt.Sprintf with no escaping —
// "json_set(data, '$.%s', ?)" — which is a SQL injection sink. The key is not a
// trusted constant: pkg/arieventhandler/ari_channel.go:209 passes the ARI
// ChannelVarset event's variable name straight through. The path is now bound
// as a query argument instead, the same way ConfbridgeAddChannelCallID already
// binds its path.
//
// The path is also quoted as $."key" rather than the previous bare $.key, again
// matching ConfbridgeAddChannelCallID. For ordinary identifier keys the two are
// equivalent. They differ only for a key containing '.' or other path
// metacharacters, where the bare form silently addressed a NESTED path (e.g.
// key "a.b" wrote data.a.b) while the quoted form writes the literal key "a.b"
// — the intended meaning of "set this data item".
func buildChannelSetDataItem(id, key string, value any, tmUpdate any) squirrel.UpdateBuilder {
	path := fmt.Sprintf("$.%q", key)
	return squirrel.
		Update(channelTable).
		Set(
			string(channel.FieldData),
			squirrel.Expr("json_set("+string(channel.FieldData)+", ?, ?)", path, value),
		).
		Set(string(channel.FieldTMUpdate), tmUpdate).
		Where(squirrel.Eq{string(channel.FieldID): id}).
		PlaceholderFormat(squirrel.Question)
}

// ChannelSetDataItem sets the item into the channel's data
func (h *handler) ChannelSetDataItem(ctx context.Context, id string, key string, value interface{}) error {
	q, args, err := buildChannelSetDataItem(id, key, value, h.utilHandler.TimeNow()).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ChannelSetDataItem. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("could not execute. ChannelSetDataItem. err: %v", err)
	}

	// update the cache
	_ = h.channelUpdateToCache(ctx, id)

	return nil
}

// ChannelSetStasis sets the stasis
func (h *handler) ChannelSetStasis(ctx context.Context, id, stasis string) error {
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldStasisName: stasis,
	})
}

// ChannelSetStateAnswer sets the given channel's state and tm_answer
func (h *handler) ChannelSetStateAnswer(ctx context.Context, id string, state ari.ChannelState) error {
	// one clock read drives both tm_update and tm_answer, as before
	ts := h.utilHandler.TimeNow()
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldState:    state,
		channel.FieldTMUpdate: ts,
		channel.FieldTMAnswer: ts,
	})
}

// ChannelSetStateRinging sets the given channel's state and tm_ringing
func (h *handler) ChannelSetStateRinging(ctx context.Context, id string, state ari.ChannelState) error {
	// one clock read drives both tm_update and tm_ringing, as before
	ts := h.utilHandler.TimeNow()
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldState:     state,
		channel.FieldTMUpdate:  ts,
		channel.FieldTMRinging: ts,
	})
}

// ChannelSetBridgeID sets the bridge_id
func (h *handler) ChannelSetBridgeID(ctx context.Context, id, bridgeID string) error {
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldBridgeID: bridgeID,
	})
}

// ChannelSetDirection sets the channel's direction
func (h *handler) ChannelSetDirection(ctx context.Context, id string, direction channel.Direction) error {
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldDirection: direction,
	})
}

// ChannelSetType sets the bridge_id
func (h *handler) ChannelSetType(ctx context.Context, id string, cType channel.Type) error {
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldType: string(cType),
	})
}

// ChannelSetSIPTransport sets the channel's sip_transport
func (h *handler) ChannelSetSIPTransport(ctx context.Context, id string, transport channel.SIPTransport) error {
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldSIPTransport: transport,
	})
}

// ChannelSetSIPCallID sets the channel's sip_call_id
func (h *handler) ChannelSetSIPCallID(ctx context.Context, id string, sipID string) error {
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldSIPCallID: sipID,
	})
}

// ChannelSetSIPData sets the channel's sip_data
func (h *handler) ChannelSetSIPData(ctx context.Context, id string, sipData map[string]string) error {
	// R-A write table: SIPData is NOT normalized at this call site.
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldSIPData: sipData,
	})
}

// ChannelSetPlaybackID sets the channel's playback_id
func (h *handler) ChannelSetPlaybackID(ctx context.Context, id string, playbackID string) error {
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldPlaybackID: playbackID,
	})
}

// ChannelEnd updates the channel end.
func (h *handler) ChannelEndAndDelete(ctx context.Context, id string, hangup ari.ChannelCause) error {
	// one clock read drives tm_update, tm_end and tm_delete, as before
	ts := h.utilHandler.TimeNow()
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldHangupCause: hangup,
		channel.FieldTMUpdate:    ts,
		channel.FieldTMEnd:       ts,
		channel.FieldTMDelete:    ts,
	})
}

// ChannelSetStasisInfoAndSIPInfo sets stasis info and SIP info
func (h *handler) ChannelSetStasisInfo(ctx context.Context, id string, chType channel.Type, stasisName string, stasisData map[channel.StasisDataType]string, direction channel.Direction) error {
	// R-A write table: StasisData is NOT normalized at this call site.
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldType:       chType,
		channel.FieldStasisName: stasisName,
		channel.FieldStasisData: stasisData,
		channel.FieldDirection:  direction,
	})
}

// channelGetFromCache returns channel from the cache if possible.
func (h *handler) channelGetFromCache(ctx context.Context, id string) (*channel.Channel, error) {

	// get from cache
	res, err := h.cache.ChannelGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// channelGetFromDB returns channel from the DB.
func (h *handler) channelGetFromDB(ctx context.Context, id string) (*channel.Channel, error) {
	fields := commondatabasehandler.GetDBFields(&channel.Channel{})
	query, args, err := squirrel.
		Select(fields...).
		From(channelTable).
		Where(squirrel.Eq{string(channel.FieldID): id}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. channelGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ChannelGet. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. channelGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.channelGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// channelUpdateToCache gets the channel from the DB and update the cache.
func (h *handler) channelUpdateToCache(ctx context.Context, id string) error {

	res, err := h.channelGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.channelSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// channelSetToCache sets the given channel to the cache
func (h *handler) channelSetToCache(ctx context.Context, channel *channel.Channel) error {
	if err := h.cache.ChannelSet(ctx, channel); err != nil {
		return err
	}

	return nil
}

// ChannelSetMuteDirection sets the channel's mute_direction
func (h *handler) ChannelSetMuteDirection(ctx context.Context, id string, muteDirection channel.MuteDirection) error {
	return h.channelUpdate(ctx, id, map[channel.Field]any{
		channel.FieldMuteDirection: muteDirection,
	})
}

// ChannelList returns a list of channels.
//
// SECURITY (plan §6 item 2): the pre-migration implementation interpolated each
// filter map key straight into the WHERE clause via
// fmt.Sprintf("%s and %s = ?", q, k) with no validation, which is the same
// injection class as the old ChannelSetDataItem. Filters are now applied by
// commondatabasehandler.ApplyFields, which binds values as query arguments.
//
// The filter type also changes from map[string]string to
// map[channel.Field]any (plan §6 item 4), matching every other List function in
// this service. ApplyFields switches on the VALUE type for the "deleted" key,
// so it must now be a Go bool, not the string "false" — a string would fall
// through to a comparison against a literal (non-existent) "deleted" column
// instead of being translated to tm_delete IS NULL / IS NOT NULL.
//
// ApplyFields's deleted handling matches this table's existing
// tm_delete IS NULL / IS NOT NULL semantics exactly. Note it would be WRONG for
// a table using the 9999-01-01 sentinel convention; the two conventions coexist
// in this service and are deliberately not unified here (plan §6 item 6).
func (h *handler) ChannelList(ctx context.Context, size uint64, token string, filters map[channel.Field]any) ([]*channel.Channel, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	dbFields := commondatabasehandler.GetDBFields(&channel.Channel{})
	sb := squirrel.
		Select(dbFields...).
		From(channelTable).
		Where(squirrel.Lt{string(channel.FieldTMCreate): token}).
		OrderBy(string(channel.FieldTMCreate) + " DESC").
		// plan §6 item 7: Limit takes a uint64 directly, so the previous
		// strconv.FormatUint(size, 10) pre-formatting is dropped. Intentional
		// simplification, not a behavior change.
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ChannelList. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ChannelList. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "could not query. ChannelList. query: %s, err: %v", query, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*channel.Channel{}
	for rows.Next() {
		u, err := h.channelGetFromRow(rows)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get data. channelGetFromRow, err: %v", err)
		}

		res = append(res, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ChannelList. err: %v", err)
	}

	return res, nil
}

// ChannelGetsForRecovery returns list of channels for recovery.
func (h *handler) ChannelGetsForRecovery(
	ctx context.Context,
	asteriskID string,
	channelType channel.Type,
	startTime *time.Time,
	endTime *time.Time,
	size uint64,
) ([]*channel.Channel, error) {
	dbFields := commondatabasehandler.GetDBFields(&channel.Channel{})
	query, args, err := squirrel.
		Select(dbFields...).
		From(channelTable).
		Where(squirrel.Eq{string(channel.FieldAsteriskID): asteriskID}).
		Where(squirrel.Eq{string(channel.FieldType): channelType}).
		Where(squirrel.Gt{string(channel.FieldTMCreate): startTime}).
		Where(squirrel.Lt{string(channel.FieldTMCreate): endTime}).
		OrderBy(string(channel.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ChannelGetsForRecovery. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "could not query. query: %s, err: %v", query, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*channel.Channel{}
	for rows.Next() {
		u, err := h.channelGetFromRow(rows)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get data. channelGetFromRow, err: %v", err)
		}

		res = append(res, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ChannelGetsForRecovery. err: %v", err)
	}

	return res, nil
}
