package messagehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package messagehandler -destination ./mock_messagehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/linehandler"
)

// MessageHandler defiens
type MessageHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		conversationID uuid.UUID,
		status message.Status,
		referenceType conversation.ReferenceType,
		referenceID string,
		sourceID string,
		messageType message.Type,
		messageData []byte,
	) (*message.Message, error)
	GetsByConversationID(ctx context.Context, conversationID uuid.UUID, pageToken string, pageSize uint64) ([]*message.Message, error)

	SendToConversation(ctx context.Context, cv *conversation.Conversation, messageType message.Type, messageData []byte) (*message.Message, error)
}

type messageHandler struct {
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	lineHandler linehandler.LineHandler
}

// NewMessageHandler returns a new ConversationHandler
func NewMessageHandler(db dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler, lineHandler linehandler.LineHandler) MessageHandler {
	return &messageHandler{
		db:            db,
		notifyHandler: notifyHandler,

		lineHandler: lineHandler,
	}
}
