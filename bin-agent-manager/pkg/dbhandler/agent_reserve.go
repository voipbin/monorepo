package dbhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/metricshandler"
)

// AgentReserve atomically reserves the agent (method B, VOIP-1539 §3).
//
// It performs a compare-and-swap UPDATE guarded by
// `status = 'available' AND reserve_reference_id = <zero>`: only an available,
// not-already-reserved agent can be reserved. It returns true when this call
// won the CAS (RowsAffected == 1), false when the agent was not eligible
// (busy/away/offline/deleted, or already reserved by someone else).
func (h *handler) AgentReserve(ctx context.Context, agentID uuid.UUID, refType string, refID uuid.UUID) (bool, error) {
	start := time.Now()
	var dbErr error
	defer func() {
		elapsed := time.Since(start)
		metricshandler.DBOperationDuration.WithLabelValues("reserve", "agent").Observe(float64(elapsed.Milliseconds()))
		status := "success"
		if dbErr != nil {
			status = "failure"
		}
		metricshandler.DBOperationTotal.WithLabelValues("reserve", "agent", status).Inc()
	}()

	now := h.utilHandler.TimeNow()

	fields, err := commondatabasehandler.PrepareFields(map[agent.Field]any{
		agent.FieldReserveReferenceType: refType,
		agent.FieldReserveReferenceID:   refID,
		agent.FieldTMReserve:            now,
		agent.FieldTMUpdate:             now,
	})
	if err != nil {
		dbErr = err
		return false, fmt.Errorf("could not prepare fields. AgentReserve. err: %v", err)
	}

	sqlStr, args, err := squirrel.
		Update(agentTable).
		SetMap(fields).
		Where(squirrel.Eq{string(agent.FieldID): agentID.Bytes()}).
		Where(squirrel.Eq{string(agent.FieldStatus): string(agent.StatusAvailable)}).
		Where(squirrel.Eq{string(agent.FieldReserveReferenceID): uuid.Nil.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		dbErr = err
		return false, fmt.Errorf("could not build query. AgentReserve. err: %v", err)
	}

	res, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		dbErr = err
		return false, fmt.Errorf("could not execute query. AgentReserve. err: %v", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		dbErr = err
		return false, fmt.Errorf("could not get rows affected. AgentReserve. err: %v", err)
	}

	// refresh the cache so a cache-first read reflects the new reserve state
	_ = h.agentUpdateToCache(ctx, agentID)

	return affected == 1, nil
}

// AgentReserveRelease clears the agent's reservation, restricted to the owning
// reservation token (VOIP-1539 §3.2). The `WHERE reserve_reference_id = refID`
// guard ensures only the caller that currently holds the reservation can
// release it; a stale/losing caller with a different token is a no-op.
func (h *handler) AgentReserveRelease(ctx context.Context, agentID uuid.UUID, refID uuid.UUID) error {
	start := time.Now()
	var dbErr error
	defer func() {
		elapsed := time.Since(start)
		metricshandler.DBOperationDuration.WithLabelValues("reserve_release", "agent").Observe(float64(elapsed.Milliseconds()))
		status := "success"
		if dbErr != nil {
			status = "failure"
		}
		metricshandler.DBOperationTotal.WithLabelValues("reserve_release", "agent", status).Inc()
	}()

	now := h.utilHandler.TimeNow()

	fields, err := commondatabasehandler.PrepareFields(map[agent.Field]any{
		agent.FieldReserveReferenceType: "",
		agent.FieldReserveReferenceID:   uuid.Nil,
		agent.FieldTMReserve:            nil,
		agent.FieldTMUpdate:             now,
	})
	if err != nil {
		dbErr = err
		return fmt.Errorf("could not prepare fields. AgentReserveRelease. err: %v", err)
	}

	sqlStr, args, err := squirrel.
		Update(agentTable).
		SetMap(fields).
		Where(squirrel.Eq{string(agent.FieldID): agentID.Bytes()}).
		Where(squirrel.Eq{string(agent.FieldReserveReferenceID): refID.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		dbErr = err
		return fmt.Errorf("could not build query. AgentReserveRelease. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		dbErr = err
		return fmt.Errorf("could not execute query. AgentReserveRelease. err: %v", err)
	}

	// refresh the cache so a cache-first read reflects the cleared reserve state
	_ = h.agentUpdateToCache(ctx, agentID)

	return nil
}

// AgentReserveSweep reclaims zombie reservations: reserved agents whose
// tm_reserve is older than `before` (VOIP-1539 §3.2, time-only predicate, no
// reverse lookup needed). It clears the reserve fields on every matching row
// and returns the number of reservations reclaimed. This is invoked
// periodically by agent-manager itself and is not exposed over REST.
func (h *handler) AgentReserveSweep(ctx context.Context, before time.Time) (int, error) {
	start := time.Now()
	var dbErr error
	defer func() {
		elapsed := time.Since(start)
		metricshandler.DBOperationDuration.WithLabelValues("reserve_sweep", "agent").Observe(float64(elapsed.Milliseconds()))
		status := "success"
		if dbErr != nil {
			status = "failure"
		}
		metricshandler.DBOperationTotal.WithLabelValues("reserve_sweep", "agent", status).Inc()
	}()

	// collect the ids of the zombie rows first so their caches can be
	// invalidated after the bulk clear (the reserve fields live on the agent
	// struct and are cached alongside it).
	selectSQL, selectArgs, err := squirrel.
		Select(string(agent.FieldID)).
		From(agentTable).
		Where(squirrel.NotEq{string(agent.FieldReserveReferenceID): uuid.Nil.Bytes()}).
		Where(squirrel.Lt{string(agent.FieldTMReserve): before.UTC()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		dbErr = err
		return 0, fmt.Errorf("could not build select query. AgentReserveSweep. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, selectSQL, selectArgs...)
	if err != nil {
		dbErr = err
		return 0, fmt.Errorf("could not query. AgentReserveSweep. err: %v", err)
	}

	ids := []uuid.UUID{}
	for rows.Next() {
		var raw []byte
		if errScan := rows.Scan(&raw); errScan != nil {
			_ = rows.Close()
			dbErr = errScan
			return 0, fmt.Errorf("could not scan row. AgentReserveSweep. err: %v", errScan)
		}
		id, errParse := uuid.FromBytes(raw)
		if errParse != nil {
			_ = rows.Close()
			dbErr = errParse
			return 0, fmt.Errorf("could not parse id. AgentReserveSweep. err: %v", errParse)
		}
		ids = append(ids, id)
	}
	if errRows := rows.Err(); errRows != nil {
		_ = rows.Close()
		dbErr = errRows
		return 0, fmt.Errorf("rows iteration error. AgentReserveSweep. err: %v", errRows)
	}
	_ = rows.Close()

	if len(ids) == 0 {
		return 0, nil
	}

	now := h.utilHandler.TimeNow()
	fields, err := commondatabasehandler.PrepareFields(map[agent.Field]any{
		agent.FieldReserveReferenceType: "",
		agent.FieldReserveReferenceID:   uuid.Nil,
		agent.FieldTMReserve:            nil,
		agent.FieldTMUpdate:             now,
	})
	if err != nil {
		dbErr = err
		return 0, fmt.Errorf("could not prepare fields. AgentReserveSweep. err: %v", err)
	}

	updateSQL, updateArgs, err := squirrel.
		Update(agentTable).
		SetMap(fields).
		Where(squirrel.NotEq{string(agent.FieldReserveReferenceID): uuid.Nil.Bytes()}).
		Where(squirrel.Lt{string(agent.FieldTMReserve): before.UTC()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		dbErr = err
		return 0, fmt.Errorf("could not build update query. AgentReserveSweep. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, updateSQL, updateArgs...); err != nil {
		dbErr = err
		return 0, fmt.Errorf("could not execute update. AgentReserveSweep. err: %v", err)
	}

	// refresh caches for every reclaimed agent
	for _, id := range ids {
		_ = h.agentUpdateToCache(ctx, id)
	}

	return len(ids), nil
}
