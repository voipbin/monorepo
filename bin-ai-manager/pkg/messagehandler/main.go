package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_dialogflow_handler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	"monorepo/bin-ai-manager/pkg/participanthandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

// CreateOption configures optional parameters for Create.
type CreateOption func(*createParams)

// createParams holds optional parameters for Create.
type createParams struct {
	pipecatcallID      uuid.UUID
	deliveryStatus     message.DeliveryStatus
	activeAIID         uuid.UUID
	inReplyToMessageID uuid.UUID
}

// WithPipecatcallID sets the pipecatcall ID on createParams.
func WithPipecatcallID(id uuid.UUID) CreateOption {
	return func(p *createParams) { p.pipecatcallID = id }
}

// WithDeliveryStatus sets the delivery status on createParams.
func WithDeliveryStatus(s message.DeliveryStatus) CreateOption {
	return func(p *createParams) { p.deliveryStatus = s }
}

// WithActiveAIID sets the active AI ID on createParams.
func WithActiveAIID(id uuid.UUID) CreateOption {
	return func(p *createParams) { p.activeAIID = id }
}

// WithInReplyToMessageID sets the in-reply-to message ID on createParams.
// See VOIP-1234 design doc §4-1 for the cross-talk prevention this supports.
func WithInReplyToMessageID(id uuid.UUID) CreateOption {
	return func(p *createParams) { p.inReplyToMessageID = id }
}

type MessageHandler interface {
	Create(
		ctx context.Context,
		id uuid.UUID,
		customerID uuid.UUID,
		aicallID uuid.UUID,
		activeflowID uuid.UUID,
		direction message.Direction,
		role message.Role,
		content string,
		toolCalls []message.ToolCall,
		toolCallID string,
		opts ...CreateOption,
	) (*message.Message, error)
	Get(ctx context.Context, id uuid.UUID) (*message.Message, error)
	List(ctx context.Context, size uint64, token string, filters map[message.Field]any) ([]*message.Message, error)

	EventPMMessageUserTranscription(ctx context.Context, evt *pmmessage.Message)
	EventPMMessageBotLLM(ctx context.Context, evt *pmmessage.Message)
	EventPMMessageBotLLMIntermediate(ctx context.Context, evt *pmmessage.Message)
	EventPMMessageUserLLM(ctx context.Context, evt *pmmessage.Message)
	EventPMTeamMemberSwitched(ctx context.Context, evt *pmmessage.MemberSwitchedEvent)
	EventPMPipecatcallTerminated(ctx context.Context, evt *pmpipecatcall.Pipecatcall) error
}

type messageHandler struct {
	utilHandler   utilhandler.UtilHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler

	engineOpenaiHandler     engine_openai_handler.EngineOpenaiHandler
	engineDialogflowHandler engine_dialogflow_handler.EngineDialogflowHandler
	participantHandler      participanthandler.ParticipantHandler
}

var (
	metricsNamespace = "ai_manager"

	promMessageCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "message_create_total",
			Help:      "Total number of created message with role.",
		},
		[]string{"role"},
	)
)

func init() {
	prometheus.MustRegister(
		promMessageCreateTotal,
	)
}

func NewMessageHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,

	engineOpenaiHandler engine_openai_handler.EngineOpenaiHandler,
	engineDialogflowHandler engine_dialogflow_handler.EngineDialogflowHandler,
	participantHandler participanthandler.ParticipantHandler,
) MessageHandler {

	return &messageHandler{
		reqHandler:    reqHandler,
		utilHandler:   utilhandler.NewUtilHandler(),
		notifyHandler: notifyHandler,
		db:            db,

		engineOpenaiHandler:     engineOpenaiHandler,
		engineDialogflowHandler: engineDialogflowHandler,
		participantHandler:      participantHandler,
	}
}
