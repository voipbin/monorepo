package messagehandlermessagebird

//go:generate mockgen -package messagehandlermessagebird -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/dbhandler"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

// MessageHandlerMessagebird is interface for service handle
type MessageHandlerMessagebird interface {
	SendMessage(ctx context.Context, messageID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string) ([]target.Target, error)
}

// messageHandlerMessagebird structure for service handle
type messageHandlerMessagebird struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler

	requestExternal requestexternal.RequestExternal
}

// NewMessageHandlerMessagebird returns new service handler
func NewMessageHandlerMessagebird(r requesthandler.RequestHandler, db dbhandler.DBHandler, reqExternal requestexternal.RequestExternal) MessageHandlerMessagebird {
	h := &messageHandlerMessagebird{
		reqHandler:      r,
		db:              db,
		requestExternal: reqExternal,
	}

	return h
}
