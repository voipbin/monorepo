package conversationhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package conversationhandler -destination ./mock_conversationhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/accounthandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/linehandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/messagehandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/smshandler"
)

// ConversationHandler defines
type ConversationHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error)
	GetByReferenceInfo(ctx context.Context, customerID uuid.UUID, referenceType conversation.ReferenceType, referenceID string) (*conversation.Conversation, error)
	GetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]*conversation.Conversation, error)
	Update(ctx context.Context, id uuid.UUID, name string, detail string) (*conversation.Conversation, error)

	Hook(ctx context.Context, uri string, data []byte) error
	Event(ctx context.Context, referenceType conversation.ReferenceType, data []byte) error

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
