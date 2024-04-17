package chatbothandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package chatbothandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/dbhandler"
)

// ChatbotHandler interface
type ChatbotHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		engineType chatbot.EngineType,
		initPrompt string,
	) (*chatbot.Chatbot, error)
	Get(ctx context.Context, id uuid.UUID) (*chatbot.Chatbot, error)
	Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*chatbot.Chatbot, error)
	Delete(ctx context.Context, id uuid.UUID) (*chatbot.Chatbot, error)
	Update(ctx context.Context, id uuid.UUID, name string, detail string, engineType chatbot.EngineType, initPrompt string) (*chatbot.Chatbot, error)
}

// chatbotHandler structure for service handle
type chatbotHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
}

var (
	metricsNamespace = "chatbot_manager"

	promChatbotCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "chatbot_create_total",
			Help:      "Total number of created chatbot with engine type.",
		},
		[]string{"engine_type"},
	)
)

func init() {
	prometheus.MustRegister(
		promChatbotCreateTotal,
	)
}

// NewChatbotHandler define
func NewChatbotHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
) ChatbotHandler {
	return &chatbotHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
		db:            db,
	}
}
