package messagechatroomhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package messagechatroomhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/chatroomhandler"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/dbhandler"
)

// messagechatroomHandler defines
type messagechatroomHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

// MessagechatroomHandler defines
type MessagechatroomHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error)
	Gets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*messagechatroom.Messagechatroom, error)
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		agentID uuid.UUID,
		chatroomID uuid.UUID,
		messagechatID uuid.UUID,
		source *commonaddress.Address,
		messageType messagechatroom.Type,
		text string,
		medias []media.Media,
	) (*messagechatroom.Messagechatroom, error)
	Delete(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error)
}

// NewMessagechatroomHandler returns new ChatroomHandler
func NewMessagechatroomHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,

	chatroomHandler chatroomhandler.ChatroomHandler,
) MessagechatroomHandler {

	return &messagechatroomHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}
}
