package messagetargethandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package messagetargethandler -destination ./mock_messagetargethandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/messagetarget"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
)

// MessagetargetHandler is interface for MessageTarget handle
type MessagetargetHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*messagetarget.MessageTarget, error)
	Update(ctx context.Context, m *messagetarget.MessageTarget) error
	UpdateByCustomer(ctx context.Context, m *cscustomer.Customer) (*messagetarget.MessageTarget, error)
}

// webhookHandler structure for service handle
type messagetargetHandler struct {
	db dbhandler.DBHandler

	reqHandler requesthandler.RequestHandler
}

// NewMessageTargetHandler returns new message target handler
func NewMessageTargetHandler(db dbhandler.DBHandler, reqHandler requesthandler.RequestHandler) MessagetargetHandler {

	h := &messagetargetHandler{
		db:         db,
		reqHandler: reqHandler,
	}

	return h
}
