package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/cachehandler"
)

// DBHandler interface
type DBHandler interface {
	AgentCountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error)
	AgentCreate(ctx context.Context, a *agent.Agent) error
	AgentDelete(ctx context.Context, id uuid.UUID) error
	AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	AgentGetByUsername(ctx context.Context, username string) (*agent.Agent, error)
	AgentList(ctx context.Context, size uint64, token string, filters map[agent.Field]any) ([]*agent.Agent, error)
	AgentGetByCustomerIDAndAddress(ctx context.Context, customerID uuid.UUID, address *commonaddress.Address) (*agent.Agent, error)
	AgentSetAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) error
	AgentSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, ringMethod agent.RingMethod) error
	AgentSetPasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error
	AgentSetPermission(ctx context.Context, id uuid.UUID, permission agent.Permission) error
	AgentSetStatus(ctx context.Context, id uuid.UUID, status agent.Status) error
	AgentSetTagIDs(ctx context.Context, id uuid.UUID, tags []uuid.UUID) error
	AgentUpdate(ctx context.Context, id uuid.UUID, fields map[agent.Field]any) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound      = fmt.Errorf("record not found")
	ErrAlreadyExists = fmt.Errorf("record already exists")
)

// dbExecQuerier is satisfied by both *sql.DB and *sql.Tx, so the child-table
// helpers can run either standalone or inside a transaction.
type dbExecQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// withTx runs the given function within a single transaction. It commits on
// success and rolls back on error (or panic).
func (h *handler) withTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction. err: %v", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction. err: %v", err)
	}

	return nil
}


// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
	return h
}
