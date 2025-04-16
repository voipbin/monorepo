package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_messagehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/smshandler"
)

// MessageHandler defiens
type MessageHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		conversationID uuid.UUID,
		direction message.Direction,
		status message.Status,
		referenceType conversation.Type,
		referenceID string,
		transactionID string,
		text string,
		medias []media.Media,
	) (*message.Message, error)
	Delete(ctx context.Context, id uuid.UUID) (*message.Message, error)
	GetsByConversationID(ctx context.Context, conversationID uuid.UUID, pageToken string, pageSize uint64) ([]*message.Message, error)
	GetsByTransactionID(ctx context.Context, transactionID string, pageToken string, pageSize uint64) ([]*message.Message, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status message.Status) (*message.Message, error)

	Send(ctx context.Context, cv *conversation.Conversation, text string, medias []media.Media) (*message.Message, error)
}

type messageHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	accountHandler accounthandler.AccountHandler
	lineHandler    linehandler.LineHandler
	smsHandler     smshandler.SMSHandler
}

// NewMessageHandler returns a new ConversationHandler
func NewMessageHandler(
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	accountHandler accounthandler.AccountHandler,
	lineHandler linehandler.LineHandler,
	smsHandler smshandler.SMSHandler,
) MessageHandler {
	return &messageHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		notifyHandler: notifyHandler,

		accountHandler: accountHandler,
		lineHandler:    lineHandler,
		smsHandler:     smsHandler,
	}
}
