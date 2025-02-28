package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-chatbot-manager/models/message"
	"monorepo/bin-chatbot-manager/pkg/chatbotcallhandler"
	"monorepo/bin-chatbot-manager/pkg/dbhandler"
	openaihandler "monorepo/bin-chatbot-manager/pkg/openai_handler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

type MessageHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*message.Message, error)
	Gets(ctx context.Context, chatbotcallID uuid.UUID, size uint64, token string, filters map[string]string) ([]*message.Message, error)

	Send(ctx context.Context, chatbotcallID uuid.UUID, role message.Role, content string) (*message.Message, error)
}

type messageHandler struct {
	utilHandler   utilhandler.UtilHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler

	chatbotcallHandler chatbotcallhandler.ChatbotcallHandler

	openaiHandler openaihandler.OpenaiHandler
}

var (
	metricsNamespace = "chatbot_manager"

	promMessageCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "message_create_total",
			Help:      "Total number of created message with role.",
		},
		[]string{"engine_type"},
	)
	promMessageProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "message_process_time",
			Help:      "Process time of message.",
			Buckets: []float64{
				50, 100, 500, 1000, 3000, 6000,
			},
		},
		[]string{"engine_type"},
	)
)

func init() {
	prometheus.MustRegister(
		promMessageCreateTotal,
		promMessageProcessTime,
	)
}

func NewMessageHandler(
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	chatbotcallHandler chatbotcallhandler.ChatbotcallHandler,

	chatgptHandler openaihandler.OpenaiHandler,
) MessageHandler {
	return &messageHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		notifyHandler: notifyHandler,
		db:            db,

		chatbotcallHandler: chatbotcallHandler,

		openaiHandler: chatgptHandler,
	}
}
