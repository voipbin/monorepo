package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-ai-manager/models/aicall"
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
	origin             message.Origin
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

// WithOrigin sets the message origin on createParams.
//
// message.OriginProactive marks an AI-initiated notification (notify_agent);
// message.OriginListenInternal marks the mechanical tool-call/tool-result rows a
// listen evaluation turn writes, which are excluded from every future LLM
// replay. Omitting the option leaves message.OriginNone, which is what every
// ordinary message wants. See docs/plans/
// 2026-09-03-insight-ai-realtime-listen-design.md §5.6.2 and §5.4.5.
func WithOrigin(o message.Origin) CreateOption {
	return func(p *createParams) { p.origin = o }
}

// applyCreateOptions resolves opts against Create's defaults. It is the ONE
// place those defaults live: Create, ResolveOriginForTest and
// ApplyCreateOptions all go through it, so a probe can never drift from the row
// Create would actually have built.
//
// DeliveryStatusDelivered matches the legacy semantics and the DB column
// default, so a caller that passes no opts keeps working unchanged.
func applyCreateOptions(opts ...CreateOption) createParams {
	p := createParams{
		pipecatcallID:  uuid.Nil,
		deliveryStatus: message.DeliveryStatusDelivered,
	}
	for _, opt := range opts {
		opt(&p)
	}

	return p
}

// CreateOptionView is the resolved, READ-ONLY result of applying a set of
// CreateOptions. createParams itself is unexported (it is an implementation
// detail of Create), but other packages' tests legitimately need to assert
// WHICH option a caller passed: "the row is stamped with the right active AI
// id" is a real behavioural contract, and matching the variadic argument with
// gomock.Any() waves exactly that away.
// USE ONLY FROM TESTS.
type CreateOptionView struct {
	PipecatcallID      uuid.UUID
	DeliveryStatus     message.DeliveryStatus
	ActiveAIID         uuid.UUID
	InReplyToMessageID uuid.UUID
	Origin             message.Origin
}

// ApplyCreateOptions applies opts through applyCreateOptions and returns what
// they resolved to, so a view is directly comparable with the row Create would
// have built.
// USE ONLY FROM TESTS.
func ApplyCreateOptions(opts ...CreateOption) CreateOptionView {
	p := applyCreateOptions(opts...)

	return CreateOptionView{
		PipecatcallID:      p.pipecatcallID,
		DeliveryStatus:     p.deliveryStatus,
		ActiveAIID:         p.activeAIID,
		InReplyToMessageID: p.inReplyToMessageID,
		Origin:             p.origin,
	}
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

// ResolveOriginForTest applies the given options and reports the resulting
// Origin. createParams is unexported, so a test in another package cannot
// otherwise observe what an option actually set.
// USE ONLY FROM TESTS.
func ResolveOriginForTest(opts ...CreateOption) message.Origin {
	return applyCreateOptions(opts...).origin
}

// isForeignPipecatcall reports whether an inbound pipecat message event came
// from a pipecatcall session the AIcall does not consider its current
// conversational turn -- a listen evaluation turn, or a genuinely stale reply.
// Such an event must not be persisted or delivered.
//
// It is applied only for aicall.ReferenceTypeContactCase, and only in the two
// handlers a listen turn can actually reach. EventPMMessageUserLLM and
// EventPMMessageUserTranscription are both driven by an STT leg, and a listen
// turn starts with STTTypeNone, so the condition this checks for structurally
// cannot occur on those paths.
//
// See docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.4.4(b).
func (h *messageHandler) isForeignPipecatcall(ac *aicall.AIcall, evtPipecatcallID uuid.UUID) bool {
	return ac.PipecatcallID != evtPipecatcallID
}
