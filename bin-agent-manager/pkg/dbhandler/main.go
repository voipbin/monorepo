package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"fmt"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/models/resource"
	"monorepo/bin-agent-manager/pkg/cachehandler"
)

// DBHandler interface
type DBHandler interface {
	AgentCreate(ctx context.Context, a *agent.Agent) error
	AgentDelete(ctx context.Context, id uuid.UUID) error
	AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	AgentGetByUsername(ctx context.Context, username string) (*agent.Agent, error)
	AgentGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*agent.Agent, error)
	AgentGetByCustomerIDAndAddress(ctx context.Context, customerID uuid.UUID, address *commonaddress.Address) (*agent.Agent, error)
	AgentSetAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) error
	AgentSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, ringMethod agent.RingMethod) error
	AgentSetPasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error
	AgentSetPermission(ctx context.Context, id uuid.UUID, permission agent.Permission) error
	AgentSetStatus(ctx context.Context, id uuid.UUID, status agent.Status) error
	AgentSetTagIDs(ctx context.Context, id uuid.UUID, tags []uuid.UUID) error

	ResourceCreate(ctx context.Context, a *resource.Resource) error
	ResourceGet(ctx context.Context, id uuid.UUID) (*resource.Resource, error)
	ResourceGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*resource.Resource, error)
	ResourceDelete(ctx context.Context, id uuid.UUID) error
	ResourceSetData(ctx context.Context, id uuid.UUID, data interface{}) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = fmt.Errorf("record not found")
)

// List of default values
const (
	DefaultTimeStamp = "9999-01-01 00:00:00.000000" // default timestamp
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
	return h
}
