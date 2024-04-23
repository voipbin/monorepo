package messagehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package messagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/pkg/dbhandler"
	"monorepo/bin-message-manager/pkg/messagehandlermessagebird"
)

// list of hook suffix types
const (
	hookTelnyx = "telnyx"
)

// MessageHandler defines
type MessageHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*message.Message, error)
	Gets(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*message.Message, error)
	Delete(ctx context.Context, id uuid.UUID) (*message.Message, error)

	Send(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*message.Message, error)

	Hook(ctx context.Context, uri string, m []byte) error
}

// messageHandler defines
type messageHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	messageHandlerMessagebird messagehandlermessagebird.MessageHandlerMessagebird
}

// list of variables
const (
	variableMessageSourceName       = "voipbin.message.source.name"
	variableMessageSourceDetail     = "voipbin.message.source.detail"
	variableMessageSourceTarget     = "voipbin.message.source.target"
	variableMessageSourceTargetName = "voipbin.message.source.target_name"
	variableMessageSourceType       = "voipbin.message.source.type"

	variableMessageTargetDestinationName       = "voipbin.message.target.destination.name"
	variableMessageTargetDestinationDetail     = "voipbin.message.target.destination.detail"
	variableMessageTargetDestinationTarget     = "voipbin.message.target.destination.target"
	variableMessageTargetDestinationTargetName = "voipbin.message.target.destination.target_name"
	variableMessageTargetDestinationType       = "voipbin.message.target.destination.type"

	variableMessageID        = "voipbin.message.id"
	variableMessageText      = "voipbin.message.text"
	variableMessageDirection = "voipbin.message.direction"
)

// NewMessageHandler returns a new MessageHandler
func NewMessageHandler(r requesthandler.RequestHandler, n notifyhandler.NotifyHandler, db dbhandler.DBHandler, messageHandlerMessagebird messagehandlermessagebird.MessageHandlerMessagebird) MessageHandler {

	return &messageHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    r,
		notifyHandler: n,

		messageHandlerMessagebird: messageHandlerMessagebird,
	}
}
