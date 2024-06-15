package resourcehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package resourcehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	whwebhook "monorepo/bin-webhook-manager/models/webhook"

	"monorepo/bin-agent-manager/models/resource"
	"monorepo/bin-agent-manager/pkg/agenthandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
)

// ResourceHandler interface
type ResourceHandler interface {
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*resource.Resource, error)
	Get(ctx context.Context, id uuid.UUID) (*resource.Resource, error)
	Create(ctx context.Context, customerID uuid.UUID, ownerID uuid.UUID, referenceType resource.ReferenceType, referenceID uuid.UUID, data interface{}) (*resource.Resource, error)
	Delete(ctx context.Context, id uuid.UUID) (*resource.Resource, error)
	UpdateData(ctx context.Context, id uuid.UUID, data interface{}) (*resource.Resource, error)

	EventWebhookPublished(ctx context.Context, w *whwebhook.Webhook) error
}

type resourceHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	agentHandler agenthandler.AgentHandler
}

// NewResourceHandler return ResourceHandler interface
func NewResourceHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler, agentHandler agenthandler.AgentHandler) ResourceHandler {
	return &resourceHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyHandler: notifyHandler,

		agentHandler: agentHandler,
	}
}
