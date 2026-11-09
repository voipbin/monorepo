package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-push-manager/models/device"
)

const (
	deviceTable = "push_devices"
)

// deviceGetFromRow scans a single row into a Device struct using db tags
func (h *handler) deviceGetFromRow(rows *sql.Rows) (*device.Device, error) {
	res := &device.Device{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. deviceGetFromRow. err: %w", err)
	}

	return res, nil
}

// DeviceCreate creates a new device record. The refresh time starts at the create time.
func (h *handler) DeviceCreate(ctx context.Context, d *device.Device) error {
	now := h.utilHandler.TimeNow()
	d.TMCreate = now
	d.TMUpdate = now

	fields, err := commondatabasehandler.PrepareFields(d)
	if err != nil {
		return fmt.Errorf("could not prepare fields. DeviceCreate. err: %w", err)
	}

	query, args, err := sq.Insert(deviceTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. DeviceCreate. err: %w", err)
	}

	if _, err = h.db.ExecContext(ctx, query, args...); err != nil {
		return classifyCreateError(err)
	}

	return nil
}

// classifyCreateError maps the error of the insert: a unique key violation is ErrDuplicate. The driver text of
// a duplicate error holds the key value (the token), so it is replaced and not wrapped.
func classifyCreateError(err error) error {
	if isDuplicate(err) {
		return ErrDuplicate
	}

	return fmt.Errorf("could not execute. DeviceCreate. err: %w", err)
}

// deviceGetBy returns the first device that matches the condition.
func (h *handler) deviceGetBy(ctx context.Context, where sq.Eq) (*device.Device, error) {
	columns := commondatabasehandler.GetDBFields(&device.Device{})

	query, args, err := sq.Select(columns...).
		From(deviceTable).
		Where(where).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. deviceGetBy. err: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. deviceGetBy. err: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		// an iteration error (canceled context, lost connection) is not "not found"
		if errRows := rows.Err(); errRows != nil {
			return nil, fmt.Errorf("could not iterate the rows. deviceGetBy. err: %w", errRows)
		}
		return nil, ErrNotFound
	}

	return h.deviceGetFromRow(rows)
}

// DeviceGet returns the device.
func (h *handler) DeviceGet(ctx context.Context, id uuid.UUID) (*device.Device, error) {
	return h.deviceGetBy(ctx, sq.Eq{"id": id.Bytes()})
}

// DeviceGetByToken returns the device that has the token.
func (h *handler) DeviceGetByToken(ctx context.Context, token string) (*device.Device, error) {
	return h.deviceGetBy(ctx, sq.Eq{"token": token})
}

// DeviceList returns devices based on filters.
func (h *handler) DeviceList(ctx context.Context, size uint64, token string, filters map[device.Field]any) ([]*device.Device, error) {
	columns := commondatabasehandler.GetDBFields(&device.Device{})

	builder := sq.Select(columns...).
		From(deviceTable).
		Where("tm_create < ?", token).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. DeviceList. err: %w", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. DeviceList. err: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. DeviceList. err: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*device.Device{}
	for rows.Next() {
		d, err := h.deviceGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. DeviceList. err: %w", err)
		}

		res = append(res, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate the rows. DeviceList. err: %w", err)
	}

	return res, nil
}

// DeviceTouch sets the refresh time of the device to now.
func (h *handler) DeviceTouch(ctx context.Context, id uuid.UUID) error {
	query, args, err := sq.Update(deviceTable).
		Set(string(device.FieldTMUpdate), h.utilHandler.TimeNow()).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. DeviceTouch. err: %w", err)
	}

	if _, err = h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute. DeviceTouch. err: %w", err)
	}

	return nil
}

// deviceDelete deletes the devices that match the condition.
func (h *handler) deviceDelete(ctx context.Context, name string, where sq.Sqlizer) error {
	query, args, err := sq.Delete(deviceTable).Where(where).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. %s. err: %w", name, err)
	}

	if _, err = h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute. %s. err: %w", name, err)
	}

	return nil
}

// DeviceDelete deletes the device.
func (h *handler) DeviceDelete(ctx context.Context, id uuid.UUID) error {
	return h.deviceDelete(ctx, "DeviceDelete", sq.Eq{"id": id.Bytes()})
}

// DeviceDeleteByTokenNotOwner deletes the device that has the token and a different owner.
func (h *handler) DeviceDeleteByTokenNotOwner(ctx context.Context, token string, customerID, agentID uuid.UUID) error {
	where := sq.And{
		sq.Eq{"token": token},
		sq.Or{
			sq.NotEq{"customer_id": customerID.Bytes()},
			sq.NotEq{"agent_id": agentID.Bytes()},
		},
	}

	return h.deviceDelete(ctx, "DeviceDeleteByTokenNotOwner", where)
}

// DeviceDeleteByCustomerID deletes the devices of the customer.
func (h *handler) DeviceDeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error {
	return h.deviceDelete(ctx, "DeviceDeleteByCustomerID", sq.Eq{"customer_id": customerID.Bytes()})
}

// DeviceDeleteByAgentID deletes the devices of the agent.
func (h *handler) DeviceDeleteByAgentID(ctx context.Context, agentID uuid.UUID) error {
	return h.deviceDelete(ctx, "DeviceDeleteByAgentID", sq.Eq{"agent_id": agentID.Bytes()})
}
