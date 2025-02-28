package chatbotcallhandler

//go:generate mockgen -package chatbotcallhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/models/service"
	"monorepo/bin-chatbot-manager/pkg/chatbothandler"
	"monorepo/bin-chatbot-manager/pkg/dbhandler"
	"monorepo/bin-chatbot-manager/pkg/openai_handler"
)

// ChatbotcallHandler define
type ChatbotcallHandler interface {
	Create(
		ctx context.Context,
		c *chatbot.Chatbot,
		activeflowID uuid.UUID,
		referenceType chatbotcall.ReferenceType,
		referenceID uuid.UUID,
		confbridgeID uuid.UUID,
		gender chatbotcall.Gender,
		language string,
	) (*chatbotcall.Chatbotcall, error)
	Delete(ctx context.Context, id uuid.UUID) (*chatbotcall.Chatbotcall, error)
	Get(ctx context.Context, id uuid.UUID) (*chatbotcall.Chatbotcall, error)
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*chatbotcall.Chatbotcall, error)
	GetByTranscribeID(ctx context.Context, transcribeID uuid.UUID) (*chatbotcall.Chatbotcall, error)
	Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*chatbotcall.Chatbotcall, error)

	ProcessStart(ctx context.Context, cb *chatbotcall.Chatbotcall) (*chatbotcall.Chatbotcall, error)
	ProcessEnd(ctx context.Context, cb *chatbotcall.Chatbotcall) (*chatbotcall.Chatbotcall, error)

	Start(
		ctx context.Context,
		chatbotID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType chatbotcall.ReferenceType,
		referenceID uuid.UUID,
		gender chatbotcall.Gender,
		language string,
	) (*chatbotcall.Chatbotcall, error)

	ServiceStart(
		ctx context.Context,
		chatbotID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType chatbotcall.ReferenceType,
		referenceID uuid.UUID,
		gender chatbotcall.Gender,
		language string,
	) (*service.Service, error)

	// ChatMessageByID(ctx context.Context, chatbotcallID uuid.UUID, role chatbotcall.MessageRole, text string) (*chatbotcall.Chatbotcall, error)
	ChatMessage(ctx context.Context, cb *chatbotcall.Chatbotcall, role chatbotcall.MessageRole, text string) error
}

// chatbotcallHandler define
type chatbotcallHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler

	chatbotHandler chatbothandler.ChatbotHandler
	openaiHandler  openai_handler.OpenaiHandler
}

var (
	metricsNamespace = "chatbot_manager"

	promChatbotcallCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "chatbotcall_create_total",
			Help:      "Total number of created chatbotcall with reference type.",
		},
		[]string{"reference_type"},
	)
	promChatInitProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "chatbotcall_chat_init_process_time",
			Help:      "Process time of chat initialization.",
			Buckets: []float64{
				50, 100, 500, 1000, 3000, 6000,
			},
		},
		[]string{"engine_type"},
	)
	promChatMessageProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "chatbotcall_chat_message_process_time",
			Help:      "Process time of chat message.",
			Buckets: []float64{
				50, 100, 500, 1000, 3000, 6000,
			},
		},
		[]string{"engine_type"},
	)
)

func init() {
	prometheus.MustRegister(
		promChatbotcallCreateTotal,
		promChatInitProcessTime,
		promChatMessageProcessTime,
	)
}

// NewChatbotcallHandler creates a new ChatbotHandler
func NewChatbotcallHandler(
	req requesthandler.RequestHandler,
	notify notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	chatbotHandler chatbothandler.ChatbotHandler,
	chatgptHandler openai_handler.OpenaiHandler,
) ChatbotcallHandler {
	return &chatbotcallHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    req,
		notifyHandler: notify,
		db:            db,

		chatbotHandler: chatbotHandler,
		openaiHandler:  chatgptHandler,
	}
}
