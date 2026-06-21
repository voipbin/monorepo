package dbhandler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
)

const (
	agentAddressTable = "agent_addresses"

	// agentAddressINChunkSize bounds the number of agent ids placed in a single
	// IN(...) clause, to stay well under driver placeholder limits.
	agentAddressINChunkSize = 900

	// mysqlErrDupEntry is MySQL's ER_DUP_ENTRY errno, raised on a UNIQUE/PK
	// constraint violation.
	mysqlErrDupEntry = 1062
)

// isDuplicateKeyErr reports whether the given error is a duplicate-key/unique
// constraint violation. It covers MySQL (errno 1062, matched via the typed
// *mysql.MySQLError so an unrelated message containing "1062" cannot be
// misclassified) and sqlite (used by the test harness).
func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}

	var myErr *gomysql.MySQLError
	if errors.As(err, &myErr) && myErr.Number == mysqlErrDupEntry {
		return true
	}

	// sqlite (test harness) surfaces a textual error, not a typed code.
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return true
	}

	return false
}

// agentAddressInsert inserts the given addresses for an agent, assigning
// idx = 0..n-1 in slice order, a generated uuid id, and tm_create = now.
// A duplicate-key violation is mapped to ErrAlreadyExists.
func (h *handler) agentAddressInsert(ctx context.Context, db dbExecQuerier, agentID uuid.UUID, customerID uuid.UUID, addresses []commonaddress.Address) error {
	if len(addresses) == 0 {
		return nil
	}

	now := h.utilHandler.TimeNow()

	for i, addr := range addresses {
		id, err := uuid.NewV4()
		if err != nil {
			return fmt.Errorf("could not generate uuid. agentAddressInsert. err: %v", err)
		}

		query, args, err := squirrel.
			Insert(agentAddressTable).
			SetMap(map[string]any{
				"id":          id.Bytes(),
				"agent_id":    agentID.Bytes(),
				"customer_id": customerID.Bytes(),
				"type":        string(addr.Type),
				"target":      addr.Target,
				"target_name": addr.TargetName,
				"name":        addr.Name,
				"detail":      addr.Detail,
				"idx":         i,
				"tm_create":   now,
				"tm_update":   nil,
			}).
			PlaceholderFormat(squirrel.Question).
			ToSql()
		if err != nil {
			return fmt.Errorf("could not build query. agentAddressInsert. err: %v", err)
		}

		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			if isDuplicateKeyErr(err) {
				return ErrAlreadyExists
			}
			return fmt.Errorf("could not execute query. agentAddressInsert. err: %v", err)
		}
	}

	return nil
}

// agentAddressDeleteByAgentID hard-deletes all address rows for an agent.
func (h *handler) agentAddressDeleteByAgentID(ctx context.Context, db dbExecQuerier, agentID uuid.UUID) error {
	query, args, err := squirrel.
		Delete(agentAddressTable).
		Where(squirrel.Eq{"agent_id": agentID.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. agentAddressDeleteByAgentID. err: %v", err)
	}

	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. agentAddressDeleteByAgentID. err: %v", err)
	}

	return nil
}

// agentAddressListByAgentID returns the addresses for a single agent ordered by idx.
func (h *handler) agentAddressListByAgentID(ctx context.Context, db dbExecQuerier, agentID uuid.UUID) ([]commonaddress.Address, error) {
	query, args, err := squirrel.
		Select("type", "target", "target_name", "name", "detail").
		From(agentAddressTable).
		Where(squirrel.Eq{"agent_id": agentID.Bytes()}).
		OrderBy("idx ASC").
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. agentAddressListByAgentID. err: %v", err)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. agentAddressListByAgentID. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []commonaddress.Address{}
	for rows.Next() {
		var addrType, target, targetName, name, detail string
		if err := rows.Scan(&addrType, &target, &targetName, &name, &detail); err != nil {
			return nil, fmt.Errorf("could not scan row. agentAddressListByAgentID. err: %v", err)
		}
		res = append(res, commonaddress.Address{
			Type:       commonaddress.Type(addrType),
			Target:     target,
			TargetName: targetName,
			Name:       name,
			Detail:     detail,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. agentAddressListByAgentID. err: %v", err)
	}

	return res, nil
}

// agentAddressListByAgentIDs returns a map of agent_id -> ordered addresses using
// one query per chunk of ids (chunked to bound the IN(...) size). Each agent's
// slice is ordered by idx.
func (h *handler) agentAddressListByAgentIDs(ctx context.Context, db dbExecQuerier, agentIDs []uuid.UUID) (map[uuid.UUID][]commonaddress.Address, error) {
	res := map[uuid.UUID][]commonaddress.Address{}
	if len(agentIDs) == 0 {
		return res, nil
	}

	for start := 0; start < len(agentIDs); start += agentAddressINChunkSize {
		end := start + agentAddressINChunkSize
		if end > len(agentIDs) {
			end = len(agentIDs)
		}
		chunk := agentIDs[start:end]

		idBytes := make([][]byte, 0, len(chunk))
		for _, id := range chunk {
			idBytes = append(idBytes, id.Bytes())
		}

		query, args, err := squirrel.
			Select("agent_id", "type", "target", "target_name", "name", "detail").
			From(agentAddressTable).
			Where(squirrel.Eq{"agent_id": idBytes}).
			OrderBy("agent_id ASC", "idx ASC").
			PlaceholderFormat(squirrel.Question).
			ToSql()
		if err != nil {
			return nil, fmt.Errorf("could not build query. agentAddressListByAgentIDs. err: %v", err)
		}

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("could not query. agentAddressListByAgentIDs. err: %v", err)
		}

		err = func() error {
			defer func() {
				_ = rows.Close()
			}()

			for rows.Next() {
				var agentIDRaw []byte
				var addrType, target, targetName, name, detail string
				if err := rows.Scan(&agentIDRaw, &addrType, &target, &targetName, &name, &detail); err != nil {
					return fmt.Errorf("could not scan row. agentAddressListByAgentIDs. err: %v", err)
				}

				aID, err := uuid.FromBytes(agentIDRaw)
				if err != nil {
					return fmt.Errorf("could not parse agent_id. agentAddressListByAgentIDs. err: %v", err)
				}

				res[aID] = append(res[aID], commonaddress.Address{
					Type:       commonaddress.Type(addrType),
					Target:     target,
					TargetName: targetName,
					Name:       name,
					Detail:     detail,
				})
			}
			return rows.Err()
		}()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

// agentAddressLookupOwnerAgentID returns the owning agent_id for a given
// (customer_id, type, target). Returns ErrNotFound if no row matches.
func (h *handler) agentAddressLookupOwnerAgentID(ctx context.Context, db dbExecQuerier, customerID uuid.UUID, addrType commonaddress.Type, target string) (uuid.UUID, error) {
	query, args, err := squirrel.
		Select("agent_id").
		From(agentAddressTable).
		Where(squirrel.Eq{"customer_id": customerID.Bytes()}).
		Where(squirrel.Eq{"type": string(addrType)}).
		Where(squirrel.Eq{"target": target}).
		Limit(1).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not build query. agentAddressLookupOwnerAgentID. err: %v", err)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not query. agentAddressLookupOwnerAgentID. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return uuid.Nil, fmt.Errorf("rows iteration error. agentAddressLookupOwnerAgentID. err: %v", err)
		}
		return uuid.Nil, ErrNotFound
	}

	var agentIDRaw []byte
	if err := rows.Scan(&agentIDRaw); err != nil {
		return uuid.Nil, fmt.Errorf("could not scan row. agentAddressLookupOwnerAgentID. err: %v", err)
	}

	agentID, err := uuid.FromBytes(agentIDRaw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not parse agent_id. agentAddressLookupOwnerAgentID. err: %v", err)
	}

	return agentID, nil
}
