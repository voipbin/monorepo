package messagetargethandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package messagetargethandler -destination ./mock_messagetargethandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/messagetarget"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
)

// MessageTargetHandler is interface for MessageTarget handle
type MessageTargetHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*messagetarget.MessageTarget, error)
	Update(ctx context.Context, m *messagetarget.MessageTarget) error
}

// webhookHandler structure for service handle
type messageTargetHandler struct {
	db dbhandler.DBHandler

	reqHandler requesthandler.RequestHandler
}

var (
	metricsNamespace = "webhook_manager"
)

func init() {
	prometheus.MustRegister()
}

// NewMessageTargetHandler returns new message target handler
func NewMessageTargetHandler(db dbhandler.DBHandler, reqHandler requesthandler.RequestHandler) MessageTargetHandler {

	h := &messageTargetHandler{
		db:         db,
		reqHandler: reqHandler,
	}

	return h
}
