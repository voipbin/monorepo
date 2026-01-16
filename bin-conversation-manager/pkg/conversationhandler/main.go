package conversationhandler

//go:generate mockgen -package conversationhandler -destination ./mock_conversationhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
	"monorepo/bin-conversation-manager/pkg/smshandler"
)

// ConversationHandler defines
type ConversationHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		conversationType conversation.Type,
		dialogID string,
		self commonaddress.Address,
		peer commonaddress.Address,
	) (*conversation.Conversation, error)
	Get(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error)
	List(ctx context.Context, pageToken string, pageSize uint64, filters map[conversation.Field]any) ([]*conversation.Conversation, error)
	// GetByTypeAndDialogID(ctx context.Context, conversationType conversation.Type, dialogID string) (*conversation.Conversation, error)
	Update(ctx context.Context, id uuid.UUID, fields map[conversation.Field]any) (*conversation.Conversation, error)

	Hook(ctx context.Context, uri string, data []byte) error
	Event(ctx context.Context, conversationType conversation.Type, data []byte) error

	MessageSend(ctx context.Context, conversationID uuid.UUID, text string, medias []media.Media) (*message.Message, error)
}

// conversationHandler defines
type conversationHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
	reqHandler    requesthandler.RequestHandler

	accountHandler accounthandler.AccountHandler
	messageHandler messagehandler.MessageHandler
	lineHandler    linehandler.LineHandler
	smsHandler     smshandler.SMSHandler
}

// NewConversationHandler returns a new ConversationHandler
func NewConversationHandler(
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	reqHandler requesthandler.RequestHandler,
	accountHandler accounthandler.AccountHandler,
	messageHandler messagehandler.MessageHandler,
	lineHandler linehandler.LineHandler,
	smsHandler smshandler.SMSHandler,
) ConversationHandler {
	return &conversationHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		notifyHandler: notifyHandler,
		reqHandler:    reqHandler,

		accountHandler: accountHandler,
		messageHandler: messageHandler,

		lineHandler: lineHandler,
		smsHandler:  smsHandler,
	}
}

// list of variables
const (
	variableConversationSelfName       = "voipbin.conversation.self.name"
	variableConversationSelfDetail     = "voipbin.conversation.self.detail"
	variableConversationSelfTarget     = "voipbin.conversation.self.target"
	variableConversationSelfTargetName = "voipbin.conversation.self.target_name"
	variableConversationSelfType       = "voipbin.conversation.self.type"

	variableConversationPeerName       = "voipbin.conversation.peer.name"
	variableConversationPeerDetail     = "voipbin.conversation.peer.detail"
	variableConversationPeerTarget     = "voipbin.conversation.peer.target"
	variableConversationPeerTargetName = "voipbin.conversation.peer.target_name"
	variableConversationPeerType       = "voipbin.conversation.peer.type"

	variableConversationID      = "voipbin.conversation.id"
	variableConversationOwnerID = "voipbin.conversation.owner_id"

	// conversation_message
	variableConversationMessageID        = "voipbin.conversation_message.id"
	variableConversationMessageText      = "voipbin.conversation_message.text"
	variableConversationMessageDirection = "voipbin.conversation_message.direction"
)
