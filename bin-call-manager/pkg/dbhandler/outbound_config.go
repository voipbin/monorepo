package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

const outboundConfigTable = "call_outbound_configs"

// outboundConfigGetFromRow scans a single call_outbound_configs row into an OutboundConfig.
//
// Replaces the previous hand-written scan, which parsed DATETIME columns inline
// from sql.RawBytes because the MySQL driver returns them as raw []byte without
// parseTime=true in the DSN. ScanRow handles that conversion from the db: tags,
// which is why ~40 lines of bespoke datetime parsing could be deleted.
//
// BEHAVIOR CHANGE (plan §5 R-A / B-2, intentional and tested): the old helper
// called json.Unmarshal on destination_whitelist unconditionally, so a NULL or
// empty column value was an ERROR. ScanRow's copyJSON skips NULL/empty and
// leaves the field nil instead. The column is JSON NOT NULL in MySQL, so this
// should not arise in practice, but the change is real and is covered by
// Test_OutboundConfig_destinationWhitelist_threeWay.
func (h *handler) outboundConfigGetFromRow(row *sql.Rows) (*outboundconfig.OutboundConfig, error) {
	res := &outboundconfig.OutboundConfig{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan outbound_config row: %w", err)
	}

	// R-A read table: destination_whitelist is NOT normalized on read here,
	// matching the pre-migration helper, which also left whatever json.Unmarshal
	// produced. Do not add a nil -> [] guard.

	return res, nil
}

// OutboundConfigCreate inserts a new outbound_config record.
func (h *handler) OutboundConfigCreate(ctx context.Context, c *outboundconfig.OutboundConfig) error {
	log := logrus.WithField("func", "OutboundConfigCreate")

	now := h.utilHandler.TimeNow()

	// R-A write table: DestinationWhitelist normalizes nil -> [] at THIS call
	// site (but not at Update). This is load-bearing, not cosmetic: the column
	// is JSON NOT NULL, and struct-based PrepareFields turns a nil SLICE into
	// SQL NULL (convertValueForDB's reflect.Slice nil guard), which would
	// violate the constraint.
	c.DestinationWhitelist = normalizeSlice(c.DestinationWhitelist)

	// Set timestamps BEFORE PrepareFields. Struct-based PrepareFields emits
	// every db-tagged field unconditionally with no nil-skip branch, so an
	// omitted assignment here would write SQL NULL for tm_create.
	//
	// NOTE: this deliberately does NOT copy recording.go's pattern verbatim.
	// outbound_config sets tm_create AND tm_update to now (recording leaves
	// TMUpdate nil), and never populated tm_delete.
	c.TMCreate = now
	c.TMUpdate = now
	c.TMDelete = nil

	// Struct-based PrepareFields is correct here: this is a full-row INSERT with
	// no partial-update semantics to preserve. It also handles the BINARY(16)
	// UUID columns via the ,uuid tags, replacing the previous manual .Bytes()
	// calls (docs/conventions/database.md §7.0a).
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. OutboundConfigCreate. err: %v", err)
	}

	query, args, err := squirrel.
		Insert(outboundConfigTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. OutboundConfigCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		log.Errorf("Could not create outbound_config. err: %v", err)
		return err
	}

	return nil
}

// OutboundConfigDelete soft-deletes the outbound_config with the given id.
func (h *handler) OutboundConfigDelete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithField("func", "OutboundConfigDelete")

	query, args, err := squirrel.
		Update(outboundConfigTable).
		Set(string(outboundconfig.FieldTMDelete), h.utilHandler.TimeNow()).
		Where(squirrel.Eq{string(outboundconfig.FieldID): id.Bytes()}).
		Where(squirrel.Eq{string(outboundconfig.FieldTMDelete): nil}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. OutboundConfigDelete. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		log.Errorf("Could not delete outbound_config. err: %v", err)
		return err
	}

	return nil
}

// outboundConfigGetBy runs a single-row SELECT filtered by the given predicate.
func (h *handler) outboundConfigGetBy(ctx context.Context, funcName string, pred squirrel.Eq) (*outboundconfig.OutboundConfig, error) {
	log := logrus.WithField("func", funcName)

	fields := commondatabasehandler.GetDBFields(&outboundconfig.OutboundConfig{})
	query, args, err := squirrel.
		Select(fields...).
		From(outboundConfigTable).
		Where(pred).
		Where(squirrel.Eq{string(outboundconfig.FieldTMDelete): nil}).
		Limit(1).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. %s. err: %v", funcName, err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Errorf("Could not get outbound_config. err: %v", err)
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. %s. err: %v", funcName, err)
		}
		return nil, nil
	}

	return h.outboundConfigGetFromRow(rows)
}

// OutboundConfigGetByID returns the outbound_config with the given id.
// Returns nil, nil if no record is found.
func (h *handler) OutboundConfigGetByID(ctx context.Context, id uuid.UUID) (*outboundconfig.OutboundConfig, error) {
	return h.outboundConfigGetBy(ctx, "OutboundConfigGetByID", squirrel.Eq{string(outboundconfig.FieldID): id.Bytes()})
}

// OutboundConfigGetByCustomerID returns the active outbound_config for a customer.
// Returns nil, nil if no record is found.
func (h *handler) OutboundConfigGetByCustomerID(ctx context.Context, customerID uuid.UUID) (*outboundconfig.OutboundConfig, error) {
	return h.outboundConfigGetBy(ctx, "OutboundConfigGetByCustomerID", squirrel.Eq{string(outboundconfig.FieldCustomerID): customerID.Bytes()})
}

// OutboundConfigUpdate applies the non-nil fields in req to the record identified by id,
// then returns the updated record.
func (h *handler) OutboundConfigUpdate(ctx context.Context, id uuid.UUID, req *outboundconfig.UpdateRequest) (*outboundconfig.OutboundConfig, error) {
	log := logrus.WithField("func", "OutboundConfigUpdate")

	// Map-based PrepareFields is the ONLY correct approach here (plan §5 R-C).
	// Struct-based PrepareFields emits every db-tagged field unconditionally
	// with no nil-skip branch, so it cannot express UpdateRequest's 3-state
	// semantics (nil = no change, set = write) — it would overwrite every
	// untouched column, writing SQL NULL for each nil pointer field, on every
	// update.
	fields := map[outboundconfig.Field]any{
		outboundconfig.FieldTMUpdate: h.utilHandler.TimeNow(),
	}

	if req.Name != nil {
		fields[outboundconfig.FieldName] = *req.Name
	}
	if req.Detail != nil {
		fields[outboundconfig.FieldDetail] = *req.Detail
	}
	if req.DestinationWhitelist != nil {
		// R-A write table: DestinationWhitelist is NOT normalized at this call
		// site, unlike at Create. A pointer to a nil slice already writes the
		// JSON literal "null" today; preserved deliberately.
		fields[outboundconfig.FieldDestinationWhitelist] = *req.DestinationWhitelist
	}
	if req.Codecs != nil {
		fields[outboundconfig.FieldCodecs] = *req.Codecs
	}
	if req.DefaultOutgoingSourceNumberID != nil {
		// MUST be dereferenced to a value-typed uuid.UUID (plan §5 R-C).
		// processMapValues type-switches on uuid.UUID and calls .Bytes(); it does
		// NOT match *uuid.UUID, which would fall through convertValueForDB
		// unchanged and then be handed to gofrs/uuid's driver.Valuer, producing a
		// 36-char string for a BINARY(16) column — silent truncation under
		// permissive SQL mode, "Data too long for column" under strict mode.
		fields[outboundconfig.FieldDefaultOutgoingSourceNumberID] = *req.DefaultOutgoingSourceNumberID
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return nil, fmt.Errorf("could not prepare fields. OutboundConfigUpdate. err: %v", err)
	}

	query, args, err := squirrel.
		Update(outboundConfigTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(outboundconfig.FieldID): id.Bytes()}).
		Where(squirrel.Eq{string(outboundconfig.FieldTMDelete): nil}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. OutboundConfigUpdate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		log.Errorf("Could not update outbound_config. err: %v", err)
		return nil, err
	}

	return h.OutboundConfigGetByID(ctx, id)
}

// OutboundConfigList returns a page of call_outbound_configs for the given customer,
// ordered by tm_create DESC. pageToken is an ISO-8601 timestamp used as a cursor.
func (h *handler) OutboundConfigList(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*outboundconfig.OutboundConfig, error) {
	log := logrus.WithField("func", "OutboundConfigList")

	fields := commondatabasehandler.GetDBFields(&outboundconfig.OutboundConfig{})
	sb := squirrel.
		Select(fields...).
		From(outboundConfigTable).
		Where(squirrel.Eq{string(outboundconfig.FieldCustomerID): customerID.Bytes()}).
		Where(squirrel.Eq{string(outboundconfig.FieldTMDelete): nil}).
		OrderBy(string(outboundconfig.FieldTMCreate) + " DESC").
		Limit(pageSize).
		PlaceholderFormat(squirrel.Question)

	if pageToken != "" {
		sb = sb.Where(squirrel.Lt{string(outboundconfig.FieldTMCreate): pageToken})
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. OutboundConfigList. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Errorf("Could not list call_outbound_configs. err: %v", err)
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var res []*outboundconfig.OutboundConfig
	for rows.Next() {
		c, err := h.outboundConfigGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. OutboundConfigList. err: %v", err)
		}
		res = append(res, c)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. OutboundConfigList. err: %v", err)
	}

	return res, nil
}
