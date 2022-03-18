package messagehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package messagehandler -destination ./mock_messagehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/messagehandlermessagebird"
)

// list of hook suffix types
const (
	hookTelnyx = "telnyx"
)

// MessageHandler defines
type MessageHandler interface {
	Create(ctx context.Context, m *message.Message) (*message.Message, error)
	Delete(ctx context.Context, id uuid.UUID) (*message.Message, error)
	UpdateTargets(ctx context.Context, id uuid.UUID, targets []target.Target) (*message.Message, error)
	Get(ctx context.Context, id uuid.UUID) (*message.Message, error)
	Gets(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*message.Message, error)

	Send(ctx context.Context, customerID uuid.UUID, source *cmaddress.Address, destinations []cmaddress.Address, text string) (*message.Message, error)

	Hook(ctx context.Context, uri string, m []byte) error
}

// messageHandler defines
type messageHandler struct {
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	messageHandlerMessagebird messagehandlermessagebird.MessageHandlerMessagebird
}

// NewMessageHandler returns a new MessageHandler
func NewMessageHandler(r requesthandler.RequestHandler, n notifyhandler.NotifyHandler, db dbhandler.DBHandler) MessageHandler {

	messageHandlerMessagebird := messagehandlermessagebird.NewMessageHandlerMessagebird(r, db)

	return &messageHandler{
		db:            db,
		reqHandler:    r,
		notifyHandler: n,

		messageHandlerMessagebird: messageHandlerMessagebird,
	}
}
