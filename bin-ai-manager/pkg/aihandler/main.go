package aihandler

//go:generate mockgen -package aihandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// AIHandler interface
type AIHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		engineType ai.EngineType,
		engineModel ai.EngineModel,
		engineData map[string]any,
		initPrompt string,
	) (*ai.AI, error)
	Get(ctx context.Context, id uuid.UUID) (*ai.AI, error)
	Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*ai.AI, error)
	Delete(ctx context.Context, id uuid.UUID) (*ai.AI, error)
	Update(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		engineType ai.EngineType,
		engineModel ai.EngineModel,
		engineData map[string]any,
		initPrompt string,
	) (*ai.AI, error)
}

// aiHandler structure for service handle
type aiHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
}

var (
	metricsNamespace = "ai_manager"

	promAICreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "ai_create_total",
			Help:      "Total number of created ai with engine type.",
		},
		[]string{"engine_type"},
	)
)

func init() {
	prometheus.MustRegister(
		promAICreateTotal,
	)
}

// NewAIHandler define
func NewAIHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
) AIHandler {
	return &aiHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
		db:            db,
	}
}
