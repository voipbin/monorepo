package messagechathandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package messagechathandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/chatroomhandler"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/messagechatroomhandler"
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
	Gets(ctx context.Context, token string, limit uint64, filters map[string]string) ([]*messagechat.Messagechat, error)
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
