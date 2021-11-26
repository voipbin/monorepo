package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

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
	AgentGetFromDB(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	AgentGetByUsername(ctx context.Context, userID uint64, username string) (*agent.Agent, error)
	AgentGets(ctx context.Context, userID uint64, size uint64, token string) ([]*agent.Agent, error)
	AgentSetToCache(ctx context.Context, u *agent.Agent) error
	AgentSetAddresses(ctx context.Context, id uuid.UUID, addresses []cmaddress.Address) error
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
	TagGets(ctx context.Context, userID uint64, size uint64, token string) ([]*tag.Tag, error)
	TagGetFromCache(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
	TagGetFromDB(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
}

// handler database handler
type handler struct {
	db    *sql.DB
	cache cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("Record not found")
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		db:    db,
		cache: cache,
	}
	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
