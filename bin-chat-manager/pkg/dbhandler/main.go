package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"sort"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/chat"
	"monorepo/bin-chat-manager/models/chatroom"
	"monorepo/bin-chat-manager/models/messagechat"
	"monorepo/bin-chat-manager/models/messagechatroom"
	"monorepo/bin-chat-manager/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	// chat
	ChatCreate(ctx context.Context, c *chat.Chat) error
	ChatDelete(ctx context.Context, id uuid.UUID) error
	ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error)
	ChatGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*chat.Chat, error)
	ChatUpdateRoomOwnerID(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) error
	ChatUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error
	ChatUpdateParticipantID(ctx context.Context, id uuid.UUID, participantIDs []uuid.UUID) error

	// chatroom
	ChatroomCreate(ctx context.Context, c *chatroom.Chatroom) error
	ChatroomGet(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error)
	ChatroomGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*chatroom.Chatroom, error)
	ChatroomUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error
	ChatroomDelete(ctx context.Context, id uuid.UUID) error
	ChatroomAddParticipantID(ctx context.Context, id, participantID uuid.UUID) error
	ChatroomRemoveParticipantID(ctx context.Context, id, participantID uuid.UUID) error

	// messagechat
	MessagechatCreate(ctx context.Context, m *messagechat.Messagechat) error
	MessagechatGet(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error)
	MessagechatGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*messagechat.Messagechat, error)
	MessagechatDelete(ctx context.Context, id uuid.UUID) error

	// messagechatroom
	MessagechatroomCreate(ctx context.Context, m *messagechatroom.Messagechatroom) error
	MessagechatroomGet(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error)
	MessagechatroomGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*messagechatroom.Messagechatroom, error)
	MessagechatroomDelete(ctx context.Context, id uuid.UUID) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("record not found")
)

// list of default values
const (
	DefaultTimeStamp = "9999-01-01 00:00:000"
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

// sortUUIDs sort the given participant ids
func sortUUIDs(uuids []uuid.UUID) []uuid.UUID {

	res := make([]uuid.UUID, len(uuids))
	copy(res, uuids)

	sort.Slice(res, func(i, j int) bool {
		return res[i].String() < res[j].String()
	})

	return res
}
