package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	// common
	GetCurTime() string

	// chat
	ChatCreate(ctx context.Context, c *chat.Chat) error
	ChatDelete(ctx context.Context, id uuid.UUID) error
	ChatGet(ctx context.Context, id uuid.UUID) (*chat.Chat, error)
	ChatGetByTypeAndParticipantsID(ctx context.Context, customerID uuid.UUID, chatType chat.Type, participantIDs []uuid.UUID) (*chat.Chat, error)
	ChatGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*chat.Chat, error)
	ChatGetsByType(ctx context.Context, customerID uuid.UUID, chatType chat.Type, token string, limit uint64) ([]*chat.Chat, error)
	ChatUpdateOwnerID(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) error
	ChatUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error
	ChatAddParticipantID(ctx context.Context, id, participantID uuid.UUID) error
	ChatRemoveParticipantID(ctx context.Context, id, participantID uuid.UUID) error

	// chatroom
	ChatroomCreate(ctx context.Context, c *chatroom.Chatroom) error
	ChatroomGet(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error)
	ChatroomGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*chatroom.Chatroom, error)
	ChatroomGetsByType(ctx context.Context, customerID uuid.UUID, chatType chatroom.Type, token string, limit uint64) ([]*chatroom.Chatroom, error)
	ChatroomGetsByChatID(ctx context.Context, chatID uuid.UUID, token string, limit uint64) ([]*chatroom.Chatroom, error)
	ChatroomGetsByOwnerID(ctx context.Context, ownerID uuid.UUID, token string, limit uint64) ([]*chatroom.Chatroom, error)
	ChatroomUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error
	ChatroomDelete(ctx context.Context, id uuid.UUID) error
	ChatroomAddParticipantID(ctx context.Context, id, participantID uuid.UUID) error
	ChatroomRemoveParticipantID(ctx context.Context, id, participantID uuid.UUID) error

	// messagechat
	MessagechatCreate(ctx context.Context, m *messagechat.Messagechat) error
	MessagechatGet(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error)
	MessagechatGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*messagechat.Messagechat, error)
	MessagechatGetsByChatID(ctx context.Context, chatID uuid.UUID, token string, limit uint64) ([]*messagechat.Messagechat, error)
	MessagechatDelete(ctx context.Context, id uuid.UUID) error

	// messagechatroom
	MessagechatroomCreate(ctx context.Context, m *messagechatroom.Messagechatroom) error
	MessagechatroomGet(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error)
	MessagechatroomGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*messagechatroom.Messagechatroom, error)
	MessagechatroomGetsByChatroomID(ctx context.Context, chatroomID uuid.UUID, token string, limit uint64) ([]*messagechatroom.Messagechatroom, error)
	MessagechatroomGetsByMessagechatID(ctx context.Context, messagechatID uuid.UUID, token string, limit uint64) ([]*messagechatroom.Messagechatroom, error)
	MessagechatroomDelete(ctx context.Context, id uuid.UUID) error
}

// handler database handler
type handler struct {
	db    *sql.DB
	cache cachehandler.CacheHandler
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
		db:    db,
		cache: cache,
	}
	return h
}

func (h *handler) GetCurTime() string {
	return GetCurTime()
}

// GetCurTime return current utc time string
func GetCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
