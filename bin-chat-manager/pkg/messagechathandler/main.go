package messagechathandler

//go:generate mockgen -package messagechathandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/media"
	"monorepo/bin-chat-manager/models/messagechat"
	"monorepo/bin-chat-manager/pkg/chatroomhandler"
	"monorepo/bin-chat-manager/pkg/dbhandler"
	"monorepo/bin-chat-manager/pkg/messagechatroomhandler"
)

// messagechatHandler defines
type messagechatHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	chatroomHandler        chatroomhandler.ChatroomHandler
	messagechatroomHandler messagechatroomhandler.MessagechatroomHandler
}

// MessagechatHandler defines
type MessagechatHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error)
	Gets(ctx context.Context, token string, limit uint64, filters map[messagechat.Field]any) ([]*messagechat.Messagechat, error)
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		chatID uuid.UUID,
		source *commonaddress.Address,
		messageType messagechat.Type,
		text string,
		medias []media.Media,
	) (*messagechat.Messagechat, error)
	Delete(ctx context.Context, id uuid.UUID) (*messagechat.Messagechat, error)
}

// NewMessagechatHandler returns new ChatHandler
func NewMessagechatHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,

	chatroomHandler chatroomhandler.ChatroomHandler,
	messagechatroomHandler messagechatroomhandler.MessagechatroomHandler,
) MessagechatHandler {

	return &messagechatHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,

		chatroomHandler:        chatroomHandler,
		messagechatroomHandler: messagechatroomHandler,
	}
}
