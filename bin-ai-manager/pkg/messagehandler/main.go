package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_dialogflow_handler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmmessage "monorepo/bin-pipecat-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

type MessageHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		aicallID uuid.UUID,
		direction message.Direction,
		role message.Role,
		content string,
		toolCalls []message.ToolCall,
		toolCallID string,
	) (*message.Message, error)
	Get(ctx context.Context, id uuid.UUID) (*message.Message, error)
	List(ctx context.Context, size uint64, token string, filters map[message.Field]any) ([]*message.Message, error)

	EventPMMessageUserTranscription(ctx context.Context, evt *pmmessage.Message)
	EventPMMessageBotLLM(ctx context.Context, evt *pmmessage.Message)
	EventPMMessageUserLLM(ctx context.Context, evt *pmmessage.Message)
}

type messageHandler struct {
	utilHandler   utilhandler.UtilHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler

	engineOpenaiHandler     engine_openai_handler.EngineOpenaiHandler
	engineDialogflowHandler engine_dialogflow_handler.EngineDialogflowHandler
}

var (
	metricsNamespace = "ai_manager"

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
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,

	engineOpenaiHandler engine_openai_handler.EngineOpenaiHandler,
	engineDialogflowHandler engine_dialogflow_handler.EngineDialogflowHandler,
) MessageHandler {

	return &messageHandler{
		reqHandler:    reqHandler,
		utilHandler:   utilhandler.NewUtilHandler(),
		notifyHandler: notifyHandler,
		db:            db,

		engineOpenaiHandler:     engineOpenaiHandler,
		engineDialogflowHandler: engineDialogflowHandler,
	}
}
