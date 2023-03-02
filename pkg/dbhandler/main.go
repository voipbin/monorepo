package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"fmt"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentcall"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentdial"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/cachehandler"
)

// DBHandler interface
type DBHandler interface {
	AgentCreate(ctx context.Context, a *agent.Agent) error
	AgentDelete(ctx context.Context, id uuid.UUID) error
	AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	AgentGetByUsername(ctx context.Context, customerID uuid.UUID, username string) (*agent.Agent, error)
	AgentGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*agent.Agent, error)
	AgentSetToCache(ctx context.Context, u *agent.Agent) error
	AgentSetAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) error
	AgentSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, ringMethod agent.RingMethod) error
	AgentSetPasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error
	AgentSetPermission(ctx context.Context, id uuid.UUID, permission agent.Permission) error
	AgentSetStatus(ctx context.Context, id uuid.UUID, status agent.Status) error
	AgentSetTagIDs(ctx context.Context, id uuid.UUID, tags []uuid.UUID) error
	AgentUpdateToCache(ctx context.Context, id uuid.UUID) error

	AgentCallSetToCache(ctx context.Context, u *agentcall.AgentCall) error
	AgentCallGetFromCache(ctx context.Context, id uuid.UUID) (*agentcall.AgentCall, error)
	AgentCallGet(ctx context.Context, id uuid.UUID) (*agentcall.AgentCall, error)
	AgentCallCreate(ctx context.Context, a *agentcall.AgentCall) error

	AgentDialGet(ctx context.Context, id uuid.UUID) (*agentdial.AgentDial, error)
	AgentDialCreate(ctx context.Context, a *agentdial.AgentDial) error

	TagCreate(ctx context.Context, a *tag.Tag) error
	TagDelete(ctx context.Context, id uuid.UUID) error
	TagUpdateToCache(ctx context.Context, id uuid.UUID) error
	TagSetToCache(ctx context.Context, u *tag.Tag) error
	TagSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error
	TagGet(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
	TagGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*tag.Tag, error)
	TagGetFromCache(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
	TagGetFromDB(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = fmt.Errorf("Record not found")
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
