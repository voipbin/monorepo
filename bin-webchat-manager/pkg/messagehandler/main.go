package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
)

// MessageHandler interface
type MessageHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		sessionID uuid.UUID,
		direction message.Direction,
		senderID uuid.UUID,
		text string,
	) (*message.Message, error)
	Get(ctx context.Context, id uuid.UUID) (*message.Message, error)
	List(ctx context.Context, size uint64, token string, filters map[message.Field]any) ([]*message.Message, error)
	Delete(ctx context.Context, id uuid.UUID) (*message.Message, error)
}

type messageHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
}

// NewMessageHandler returns MessageHandler interface
func NewMessageHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	dbHandler dbhandler.DBHandler,
) MessageHandler {
	return &messageHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
		db:            dbHandler,
	}
}
