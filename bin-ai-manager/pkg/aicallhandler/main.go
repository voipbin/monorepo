package aicallhandler

//go:generate mockgen -package aicallhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	tmmessage "monorepo/bin-tts-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	commonservice "monorepo/bin-common-handler/models/service"
)

// AIcallHandler define
type AIcallHandler interface {
	Create(
		ctx context.Context,
		c *ai.AI,
		activeflowID uuid.UUID,
		referenceType aicall.ReferenceType,
		referenceID uuid.UUID,
		confbridgeID uuid.UUID,
		gender aicall.Gender,
		language string,
		ttsStreamingID uuid.UUID,
		ttsStreamingPodID string,
	) (*aicall.AIcall, error)
	Delete(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	Get(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*aicall.AIcall, error)
	GetByStreamingID(ctx context.Context, transcribeID uuid.UUID) (*aicall.AIcall, error)
	GetByTranscribeID(ctx context.Context, transcribeID uuid.UUID) (*aicall.AIcall, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*aicall.AIcall, error)

	ProcessStart(ctx context.Context, cb *aicall.AIcall) (*aicall.AIcall, error)
	ProcessPause(ctx context.Context, ac *aicall.AIcall) (*aicall.AIcall, error)
	ProcessTerminating(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	ProcessTerminate(ctx context.Context, cb *aicall.AIcall) (*aicall.AIcall, error)

	Start(
		ctx context.Context,
		aiID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType aicall.ReferenceType,
		referenceID uuid.UUID,
		gender aicall.Gender,
		language string,
		resume bool,
	) (*aicall.AIcall, error)

	ServiceStart(
		ctx context.Context,
		aiID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType aicall.ReferenceType,
		referenceID uuid.UUID,
		gender aicall.Gender,
		language string,
		resuming bool,
	) (*commonservice.Service, error)

	ChatMessage(ctx context.Context, cb *aicall.AIcall, text string) error

	EventCMCallHangup(ctx context.Context, c *cmcall.Call)
	EventCMConfbridgeJoined(ctx context.Context, evt *cmconfbridge.EventConfbridgeJoined)
	EventCMConfbridgeLeaved(ctx context.Context, evt *cmconfbridge.EventConfbridgeLeaved)
	EventTMPlayFinished(ctx context.Context, evt *tmmessage.Message)
}

const (
	variableAIcallID      = "voipbin.aicall.id"
	variableAIID          = "voipbin.aicall.ai_id"
	variableAIEngineModel = "voipbin.aicall.ai_engine_model"
	variableConfbridgeID  = "voipbin.aicall.confbridge_id"
	variableGender        = "voipbin.aicall.gender"
	variableLanguage      = "voipbin.aicall.language"
)

// aicallHandler define
type aicallHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler

	aiHandler      aihandler.AIHandler
	messageHandler messagehandler.MessageHandler
}

var (
	metricsNamespace = "ai_manager"

	promAIcallCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_create_total",
			Help:      "Total number of created aicall with reference type.",
		},
		[]string{"reference_type"},
	)
	promAIcallInitProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_init_process_time",
			Help:      "Process time of aicall initialization.",
			Buckets: []float64{
				50, 100, 500, 1000, 3000, 6000,
			},
		},
		[]string{"engine_type"},
	)
	promAIcallMessageProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_message_process_time",
			Help:      "Process time of aicall message.",
			Buckets: []float64{
				50, 100, 500, 1000, 3000, 6000,
			},
		},
		[]string{"engine_type"},
	)
)

func init() {
	prometheus.MustRegister(
		promAIcallCreateTotal,
		promAIcallInitProcessTime,
		promAIcallMessageProcessTime,
	)
}

// NewAIcallHandler creates a new AIHandler
func NewAIcallHandler(
	req requesthandler.RequestHandler,
	notify notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	aiHandler aihandler.AIHandler,
	messageHandler messagehandler.MessageHandler,
) AIcallHandler {
	return &aicallHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    req,
		notifyHandler: notify,
		db:            db,

		aiHandler:      aiHandler,
		messageHandler: messageHandler,
	}
}

const (
	defaultCommonSystemPrompt = `
Role:
You are an AI assistant integrated with voipbin.
Your role is to follow the user's system or custom prompt strictly, provide natural responses, and call external tools when necessary.

Context:
- Users will set their own instructions (persona, style, context).
- You must adapt to those instructions consistently.
- If user requests or situation requires, use available tools to gather data or perform actions.

Input Values:
- User-provided system/custom prompt
- User query
- Available tools list

Instructions:
- Always prioritize the user's provided prompt instructions.
- Generate a helpful, coherent, and contextually appropriate response.
- If tools are available and required, call them responsibly and return results clearly.
- **Do not mention tool names or the fact that a tool is being used in the user-facing response.**
- Maintain consistency with the user-defined tone and role.
- If ambiguity exists, ask clarifying questions before answering.
- Before giving the final answer, outline a short execution plan (2-4 steps), then provide a concise summary (1-2 sentences) and the final answer.
- For each Input Value, ask clarifying questions **one at a time in sequence**. Wait for the user's answer before moving to the next question.

Constraints:
- Avoid hallucination; use tools for factual queries.  
- Keep answers aligned with user's persona and tone.  
- Respect conversation history and continuity.  
`
)
