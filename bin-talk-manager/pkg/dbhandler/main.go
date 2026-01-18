//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

package dbhandler

import (
	"context"
	"database/sql"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/models/participant"
	"monorepo/bin-talk-manager/models/chat"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
)

// DBHandler defines database operations interface
type DBHandler interface {
	// *Chat operations
	ChatCreate(ctx context.Context, t *chat.Chat) error
	ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error)
	ChatList(ctx context.Context, filters map[chat.Field]any, token string, size uint64) ([]*chat.Chat, error)
	TalkUpdate(ctx context.Context, id uuid.UUID, fields map[chat.Field]any) error
	ChatDelete(ctx context.Context, id uuid.UUID) error

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
	db          *sql.DB
	redis       *redis.Client
	utilHandler commonutil.UtilHandler
}

// New creates a new DBHandler
func New(db *sql.DB, redisClient *redis.Client, utilHandler commonutil.UtilHandler) DBHandler {
	return &dbHandler{
		db:          db,
		redis:       redisClient,
		utilHandler: utilHandler,
	}
}
