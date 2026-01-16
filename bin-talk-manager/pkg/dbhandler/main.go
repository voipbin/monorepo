//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

package dbhandler

import (
	"context"
	"database/sql"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/models/talk"
)

// DBHandler defines database operations interface
type DBHandler interface {
	// Talk operations
	TalkCreate(ctx context.Context, t *talk.Talk) error
	TalkGet(ctx context.Context, id uuid.UUID) (*talk.Talk, error)
	TalkList(ctx context.Context, filters map[talk.Field]any, token string, size uint64) ([]*talk.Talk, error)
	TalkUpdate(ctx context.Context, id uuid.UUID, fields map[talk.Field]any) error
	TalkDelete(ctx context.Context, id uuid.UUID) error

	// Participant operations
	ParticipantCreate(ctx context.Context, p *participant.Participant) error
	ParticipantGet(ctx context.Context, id uuid.UUID) (*participant.Participant, error)
	ParticipantList(ctx context.Context, filters map[participant.Field]any) ([]*participant.Participant, error)
	ParticipantDelete(ctx context.Context, id uuid.UUID) error

	// Message operations
	MessageCreate(ctx context.Context, m *message.Message) error
	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageList(ctx context.Context, filters map[message.Field]any, token string, size uint64) ([]*message.Message, error)
	MessageUpdate(ctx context.Context, id uuid.UUID, fields map[message.Field]any) error
	MessageDelete(ctx context.Context, id uuid.UUID) error

	// Atomic reaction operations (prevents race conditions)
	MessageAddReactionAtomic(ctx context.Context, messageID uuid.UUID, reactionJSON string) error
	MessageRemoveReactionAtomic(ctx context.Context, messageID uuid.UUID, emoji, ownerType string, ownerID uuid.UUID) error
}

// dbHandler implements DBHandler
type dbHandler struct {
	db    *sql.DB
	redis *redis.Client
}

// New creates a new DBHandler
func New(db *sql.DB, redisClient *redis.Client) DBHandler {
	return &dbHandler{
		db:    db,
		redis: redisClient,
	}
}
