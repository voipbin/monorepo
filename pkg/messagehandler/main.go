package messagehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package messagehandler -destination ./mock_messagehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/accounthandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/linehandler"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/smshandler"
)

// MessageHandler defiens
type MessageHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		conversationID uuid.UUID,
		direction message.Direction,
		status message.Status,
		referenceType conversation.ReferenceType,
		referenceID string,
		transactionID string,
		source *commonaddress.Address,
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
