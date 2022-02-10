package queuecallreferencehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package queuecallreferencehandler -destination ./mock_queuecallreferencehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

// QueuecallReferenceHandler interface
type QueuecallReferenceHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*queuecallreference.QueuecallReference, error)
	Delete(ctx context.Context, id uuid.UUID) (*queuecallreference.QueuecallReference, error)
	SetCurrentQueuecallID(ctx context.Context, referenceID uuid.UUID, queuecallType queuecall.ReferenceType, queuecallID uuid.UUID) error
}

// queuecallReferenceHandler defines
type queuecallReferenceHandler struct {
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyhandler notifyhandler.NotifyHandler
}

// NewQueuecallReferenceHandler return QueuecallReferenceHandler interface
func NewQueuecallReferenceHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
) QueuecallReferenceHandler {
	return &queuecallReferenceHandler{
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyhandler: notifyHandler,
	}
}
