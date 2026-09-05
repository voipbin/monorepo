package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	uuid "github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/models/team"
)

type handler struct {
	Addr     string
	Password string
	DB       int

	Cache *redis.Client
}

// CacheHandler interface
type CacheHandler interface {
	Connect() error

	AIGet(ctx context.Context, id uuid.UUID) (*ai.AI, error)
	AISet(ctx context.Context, data *ai.AI) error

	AIcallGet(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	AIcallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*aicall.AIcall, error)
	AIcallGetByPipecatcallID(ctx context.Context, transcribeID uuid.UUID) (*aicall.AIcall, error)
	AIcallSet(ctx context.Context, data *aicall.AIcall) error

	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageSet(ctx context.Context, data *message.Message) error

	SummaryGet(ctx context.Context, id uuid.UUID) (*summary.Summary, error)
	SummarySet(ctx context.Context, data *summary.Summary) error

	TeamGet(ctx context.Context, id uuid.UUID) (*team.Team, error)
	TeamSet(ctx context.Context, data *team.Team) error

	// Insight AI realtime call listening (see listen.go).
	ListenAIcallIDsGet(ctx context.Context, transcribeID uuid.UUID) ([]uuid.UUID, error)
	ListenAIcallIDAdd(ctx context.Context, transcribeID uuid.UUID, aicallID uuid.UUID, ttl time.Duration) error
	ListenAIcallIDRemove(ctx context.Context, transcribeID uuid.UUID, aicallID uuid.UUID) error

	ListenConversationAIcallIDsGet(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error)
	ListenConversationAIcallIDAdd(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID, ttl time.Duration) error
	ListenConversationAIcallIDRemove(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID) error
	ListenConversationAIcallIDIsMember(ctx context.Context, conversationID uuid.UUID, aicallID uuid.UUID) (bool, error)

	ListenPendingPush(ctx context.Context, aicallID uuid.UUID, line string, ttl time.Duration) error
	ListenPendingPopAll(ctx context.Context, aicallID uuid.UUID) ([]string, error)
	ListenPendingLen(ctx context.Context, aicallID uuid.UUID) (int64, error)

	ListenWindowPush(ctx context.Context, aicallID uuid.UUID, line string, windowSize int, ttl time.Duration) error
	ListenWindowGet(ctx context.Context, aicallID uuid.UUID) ([]string, error)

	ListenTurnTryLock(ctx context.Context, aicallID uuid.UUID, ttl time.Duration) (bool, error)
	ListenTurnCountIncr(ctx context.Context, aicallID uuid.UUID, ttl time.Duration) (int64, error)

	ListenTurnPipecatcallIDAdd(ctx context.Context, aicallID uuid.UUID, pipecatcallID uuid.UUID, ttl time.Duration) error
	ListenTurnPipecatcallIDIsMember(ctx context.Context, aicallID uuid.UUID, pipecatcallID uuid.UUID) (bool, error)

	// The per-AIcall create-or-reuse lock (design §5.2.2). A matched, symmetric
	// pair -- the key format lives in these two functions and nowhere else.
	ListenStartLockAcquire(ctx context.Context, aicallID uuid.UUID, token string, ttl time.Duration) (bool, error)
	ListenStartLockRelease(ctx context.Context, aicallID uuid.UUID, token string) error

	ListenStateClear(ctx context.Context, aicallID uuid.UUID) error
}

// NewHandler creates DBHandler
func NewHandler(addr string, password string, db int) CacheHandler {

	cache := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	h := &handler{
		Addr:     addr,
		Password: password,
		DB:       db,
		Cache:    cache,
	}

	return h
}

// Connect connects to the cache server
func (h *handler) Connect() error {
	_, err := h.Cache.Ping(context.Background()).Result()
	if err != nil {
		return err
	}

	return nil
}
