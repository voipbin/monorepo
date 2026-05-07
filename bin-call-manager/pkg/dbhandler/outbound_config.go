package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
)

const outboundConfigTable = "call_outbound_configs"

// outboundConfigSelectCols is the canonical column list for SELECT queries against outboundConfigTable.
const outboundConfigSelectCols = "id, customer_id, name, detail, destination_whitelist, codecs, tm_create, tm_update, tm_delete"

// outboundConfigGetFromRow scans a single call_outbound_configs row into an OutboundConfig.
func (h *handler) outboundConfigGetFromRow(row *sql.Rows) (*outboundconfig.OutboundConfig, error) {
	res := &outboundconfig.OutboundConfig{}

	var whitelistJSON []byte
	var tmCreate, tmUpdate, tmDelete sql.NullTime

	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.Name,
		&res.Detail,
		&whitelistJSON,
		&res.Codecs,
		&tmCreate,
		&tmUpdate,
		&tmDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan outbound_config row: %w", err)
	}

	if err := json.Unmarshal(whitelistJSON, &res.DestinationWhitelist); err != nil {
		return nil, fmt.Errorf("could not unmarshal destination_whitelist: %w", err)
	}

	if tmCreate.Valid {
		t := tmCreate.Time
		res.TMCreate = &t
	}
	if tmUpdate.Valid {
		t := tmUpdate.Time
		res.TMUpdate = &t
	}
	if tmDelete.Valid {
		t := tmDelete.Time
		res.TMDelete = &t
	}

	return res, nil
}

// OutboundConfigCreate inserts a new outbound_config record.
func (h *handler) OutboundConfigCreate(ctx context.Context, c *outboundconfig.OutboundConfig) error {
	log := logrus.WithField("func", "OutboundConfigCreate")

	if c.DestinationWhitelist == nil {
		c.DestinationWhitelist = []string{}
	}

	whitelistJSON, err := json.Marshal(c.DestinationWhitelist)
	if err != nil {
		return fmt.Errorf("could not marshal destination_whitelist: %w", err)
	}

	now := time.Now()
	q := "INSERT INTO " + outboundConfigTable + " (id, customer_id, name, detail, destination_whitelist, codecs, tm_create, tm_update) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	if _, err := h.db.ExecContext(ctx, q, c.ID, c.CustomerID, c.Name, c.Detail, whitelistJSON, c.Codecs, now, now); err != nil {
		log.Errorf("Could not create outbound_config. err: %v", err)
		return err
	}

	return nil
}

// OutboundConfigDelete soft-deletes the outbound_config with the given id.
func (h *handler) OutboundConfigDelete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithField("func", "OutboundConfigDelete")

	q := "UPDATE " + outboundConfigTable + " SET tm_delete = ? WHERE id = ? AND tm_delete IS NULL"
	if _, err := h.db.ExecContext(ctx, q, time.Now(), id); err != nil {
		log.Errorf("Could not delete outbound_config. err: %v", err)
		return err
	}

	return nil
}

// OutboundConfigGetByID returns the outbound_config with the given id.
// Returns nil, nil if no record is found.
func (h *handler) OutboundConfigGetByID(ctx context.Context, id uuid.UUID) (*outboundconfig.OutboundConfig, error) {
	log := logrus.WithField("func", "OutboundConfigGetByID")

	q := "SELECT " + outboundConfigSelectCols + " FROM " + outboundConfigTable + " WHERE id = ? AND tm_delete IS NULL LIMIT 1"
	rows, err := h.db.QueryContext(ctx, q, id)
	if err != nil {
		log.Errorf("Could not get outbound_config. err: %v", err)
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. OutboundConfigGetByID. err: %v", err)
		}
		return nil, nil
	}

	return h.outboundConfigGetFromRow(rows)
}

// OutboundConfigGetByCustomerID returns the active outbound_config for a customer.
// Returns nil, nil if no record is found.
func (h *handler) OutboundConfigGetByCustomerID(ctx context.Context, customerID uuid.UUID) (*outboundconfig.OutboundConfig, error) {
	log := logrus.WithField("func", "OutboundConfigGetByCustomerID")

	q := "SELECT " + outboundConfigSelectCols + " FROM " + outboundConfigTable + " WHERE customer_id = ? AND tm_delete IS NULL LIMIT 1"
	rows, err := h.db.QueryContext(ctx, q, customerID)
	if err != nil {
		log.Errorf("Could not get outbound_config by customer. err: %v", err)
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. OutboundConfigGetByCustomerID. err: %v", err)
		}
		return nil, nil
	}

	return h.outboundConfigGetFromRow(rows)
}

// OutboundConfigUpdate applies the non-nil fields in req to the record identified by id,
// then returns the updated record.
func (h *handler) OutboundConfigUpdate(ctx context.Context, id uuid.UUID, req *outboundconfig.UpdateRequest) (*outboundconfig.OutboundConfig, error) {
	log := logrus.WithField("func", "OutboundConfigUpdate")

	sets := []string{"tm_update = ?"}
	args := []interface{}{time.Now()}

	if req.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *req.Name)
	}
	if req.Detail != nil {
		sets = append(sets, "detail = ?")
		args = append(args, *req.Detail)
	}
	if req.DestinationWhitelist != nil {
		b, err := json.Marshal(*req.DestinationWhitelist)
		if err != nil {
			return nil, fmt.Errorf("could not marshal destination_whitelist: %w", err)
		}
		sets = append(sets, "destination_whitelist = ?")
		args = append(args, b)
	}
	if req.Codecs != nil {
		sets = append(sets, "codecs = ?")
		args = append(args, *req.Codecs)
	}

	args = append(args, id)
	q := fmt.Sprintf("UPDATE "+outboundConfigTable+" SET %s WHERE id = ? AND tm_delete IS NULL", strings.Join(sets, ", "))

	if _, err := h.db.ExecContext(ctx, q, args...); err != nil {
		log.Errorf("Could not update outbound_config. err: %v", err)
		return nil, err
	}

	return h.OutboundConfigGetByID(ctx, id)
}

// OutboundConfigList returns a page of call_outbound_configs for the given customer,
// ordered by tm_create DESC. pageToken is an ISO-8601 timestamp used as a cursor.
func (h *handler) OutboundConfigList(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*outboundconfig.OutboundConfig, error) {
	log := logrus.WithField("func", "OutboundConfigList")

	q := "SELECT " + outboundConfigSelectCols + " FROM " + outboundConfigTable + " WHERE customer_id = ? AND tm_delete IS NULL"
	args := []interface{}{customerID}

	if pageToken != "" {
		q += " AND tm_create < ?"
		args = append(args, pageToken)
	}
	q += " ORDER BY tm_create DESC LIMIT ?"
	args = append(args, pageSize)

	rows, err := h.db.QueryContext(ctx, q, args...)
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
