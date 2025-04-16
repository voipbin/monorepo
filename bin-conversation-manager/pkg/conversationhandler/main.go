package conversationhandler

//go:generate mockgen -package conversationhandler -destination ./mock_conversationhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
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
		referenceType conversation.Type,
		referenceID string,
		self *commonaddress.Address,
		peer *commonaddress.Address,
	) (*conversation.Conversation, error)
	Get(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error)
	Gets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]*conversation.Conversation, error)
	GetByReferenceInfo(ctx context.Context, customerID uuid.UUID, referenceType conversation.Type, referenceID string) (*conversation.Conversation, error)
	Update(ctx context.Context, id uuid.UUID, name string, detail string) (*conversation.Conversation, error)

	Hook(ctx context.Context, uri string, data []byte) error
	Event(ctx context.Context, referenceType conversation.Type, data []byte) error

	MessageSend(ctx context.Context, conversationID uuid.UUID, text string, medias []media.Media) (*message.Message, error)
}

// conversationHandler defines
type conversationHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	accountHandler accounthandler.AccountHandler
	messageHandler messagehandler.MessageHandler
	lineHandler    linehandler.LineHandler
	smsHandler     smshandler.SMSHandler
}

// NewConversationHandler returns a new ConversationHandler
func NewConversationHandler(
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	accountHandler accounthandler.AccountHandler,
	messageHandler messagehandler.MessageHandler,
	lineHandler linehandler.LineHandler,
	smsHandler smshandler.SMSHandler,
) ConversationHandler {
	return &conversationHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		db:             db,
		notifyHandler:  notifyHandler,
		accountHandler: accountHandler,
		messageHandler: messageHandler,

		lineHandler: lineHandler,
		smsHandler:  smsHandler,
	}
}
