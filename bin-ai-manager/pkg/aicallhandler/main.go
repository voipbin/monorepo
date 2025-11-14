package aicallhandler

//go:generate mockgen -package aicallhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmdtmf "monorepo/bin-call-manager/models/dtmf"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	commonservice "monorepo/bin-common-handler/models/service"
)

// AIcallHandler define
type AIcallHandler interface {
	Delete(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	Get(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*aicall.AIcall, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*aicall.AIcall, error)

	ProcessStart(ctx context.Context, cb *aicall.AIcall) (*aicall.AIcall, error)
	ProcessPause(ctx context.Context, ac *aicall.AIcall) (*aicall.AIcall, error)
	ProcessTerminating(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	ProcessTerminate(ctx context.Context, cb *aicall.AIcall) (*aicall.AIcall, error)

	ToolHandle(ctx context.Context, id uuid.UUID, toolID string, toolType message.ToolType, function message.FunctionCall) (map[string]any, error)

	Start(
		ctx context.Context,
		aiID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType aicall.ReferenceType,
		referenceID uuid.UUID,
		gender aicall.Gender,
		language string,
	) (*aicall.AIcall, error)

	ServiceStart(
		ctx context.Context,
		aiID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType aicall.ReferenceType,
		referenceID uuid.UUID,
		gender aicall.Gender,
		language string,
	) (*commonservice.Service, error)

	Send(ctx context.Context, id uuid.UUID, role message.Role, messageText string, runImmediately bool, audioResponse bool) (*message.Message, error)

	EventCMCallHangup(ctx context.Context, c *cmcall.Call)
	EventCMConfbridgeJoined(ctx context.Context, evt *cmconfbridge.EventConfbridgeJoined)
	EventCMConfbridgeLeaved(ctx context.Context, evt *cmconfbridge.EventConfbridgeLeaved)
	EventDTMFReceived(ctx context.Context, evt *cmdtmf.DTMF)
}

const (
	variableID            = "voipbin.aicall.id"
	variableAIID          = "voipbin.aicall.ai_id"
	variableAIEngineModel = "voipbin.aicall.ai_engine_model"
	variableConfbridgeID  = "voipbin.aicall.confbridge_id"
	variableGender        = "voipbin.aicall.gender"
	variableLanguage      = "voipbin.aicall.language"
	variablePipecatcallID = "voipbin.aicall.pipecatcall_id"
)

const (
	defaultPipecatcallTTSType = pmpipecatcall.TTSTypeElevenLabs
	defaultTTSType            = ai.TTSTypeElevenLabs
	defaultSTTType            = ai.STTTypeDeepgram

	defaultPipecatcallSTTType    = pmpipecatcall.STTTypeDeepgram
	defaultPipecatcallTTSVoiceID = "EXAVITQu4vr4xnSDxMaL" // Rachel

	defaultPipecatcallTimeout = time.Second * 30
)

var mapDefaultTTSVoiceIDByTTSType = map[ai.TTSType]string{
	ai.TTSTypeNone:       "",
	ai.TTSTypeAsync:      "",
	ai.TTSTypeAWS:        "Joanna",                               // Joanna (US female). https://docs.aws.amazon.com/polly/latest/dg/voicelist.html
	ai.TTSTypeAzure:      "en-US-JennyNeural",                    // Jenny Neural. https://learn.microsoft.com/en-us/azure/ai-services/speech-service/language-support
	ai.TTSTypeCartesia:   "71a7ad14-091c-4e8e-a314-022ece01c121", // British Reading Lady. https://developer.signalwire.com/voice/tts/cartesia/
	ai.TTSTypeDeepgram:   "aura-2-thalia-en",                     // Thalia (neutral, English). https://developers.deepgram.com/docs/tts-models#aura-2-all-available-spanish-voices
	ai.TTSTypeElevenLabs: "EXAVITQu4vr4xnSDxMaL",                 // Rachel. https://api.elevenlabs.io/docs
	ai.TTSTypeFish:       "",
	ai.TTSTypeGoogle:     "en-US-Wavenet-D",                       // Male, natural. https://cloud.google.com/text-to-speech/docs/voices
	ai.TTSTypeGroq:       "llama-voice-en",                        // Placeholder (Groq doesn't expose standard TTS, assumed)
	ai.TTSTypeHume:       "emotional-neutral-en",                  // Neutral English emotional TTS. https://dev.hume.ai/docs/tts
	ai.TTSTypeInworld:    "English_Female_Generic",                // Generic female character. https://docs.inworld.ai/voices
	ai.TTSTypeLMNT:       "lmnt-english-1",                        // English base voice. https://lmnt.ai/
	ai.TTSTypeMiniMax:    "english_female",                        // English female voice. https://platform.minimaxi.ai/docs/tts
	ai.TTSTypeNeuphonic:  "neuphonic-en-female",                   // Neutral English female. https://pipecat-docs.readthedocs.io/en/latest/api/pipecat.services.neuphonic.tts.html
	ai.TTSTypeNvidiaRiva: "English-US-Female-1",                   // US Female. https://docs.nvidia.com/deeplearning/riva/user-guide/docs/tts/voices.html
	ai.TTSTypeOpenAI:     "alloy",                                 // Alloy (male, neutral)
	ai.TTSTypePiper:      "en_US-amy-low",                         // Amy (US female). https://github.com/rhasspy/piper/tree/master/voices
	ai.TTSTypePlayHT:     "s3://voice-cloning-zero-shot/20b9e...", // Olivia (English Female). https://docs.play.ht/reference/api-get-voices
	ai.TTSTypeRime:       "rime-en-001",                           // English default. https://rime.ai/
	ai.TTSTypeSarvam:     "en_default",                            // English generic. https://sarvam.ai/docs
	ai.TTSTypeXTTS:       "en_male",                               // English male (cross-lingual). https://coqui.ai/docs/tts/xtts
}

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
You are an AI assistant integrated with VoIPBin.
Your role is to strictly follow the user's system or custom prompt, provide natural conversational responses, and invoke external tools when necessary.

Context:
- Users will define their own instructions (persona, style, or context).
- You must adapt and remain consistent with those user-defined instructions.
- When required by context or request, use available tools to fetch data or perform actions.
- You may receive messages in the form "DTMF_EVENT: N". These represent telephone keypad presses.
	Treat them as events, not normal user text, and respond naturally according to the conversation flow.

Additional Data:
- You may receive **extra JSON string data** that contains session-related or contextual information.
- This data may include (but is not limited to):
  - Call session details (caller ID, callee, duration, status, etc.)
  - User profile information (preferences, language, account status, etc.)
  - Conversation or tool context metadata.
- You must:
  - **Interpret and use** these data elements to enhance response accuracy and contextual relevance.
  - **Never expose, quote, or describe** the raw data directly to the user.
  - Treat this data as internal context only, not user-facing content.

Objectives:
1. **Primary Goal**: When a customer requests an action that requires a tool (e.g., call connection, message sending, information retrieval), detect the tool and generate the appropriate function call.
2. **Tool Rules**:
   - Each tool has specific required parameters (e.g., source/destination number for calls, message content for messaging). Use them correctly.
   - Always follow tool specifications exactly; do not improvise.
3. **Response Guidelines**:
   - **Do NOT** show any JSON, tool details, or backend logic to the user.
   - Respond naturally to the user. Example: "Please hold, I will try to connect you." / "Your message is being sent."
   - Immediately generate the **function call object** for the required tool after sending the user-facing message.
   - Never include explanations of the process or internal instructions in user-facing text.

Input Values:
- User-provided system/custom prompt
- User query
- Available tools list

Instructions:
- Always prioritize and follow the user's provided prompt instructions.
- Generate coherent, contextually appropriate, and helpful responses.
- If tools are available and necessary, use them responsibly and summarize results clearly.
- **Never mention tool names or disclose that a tool is being used in the user-facing reply.**
- Maintain consistency with the user's defined persona and tone.
- If ambiguity exists, ask clarifying questions before responding.
- Ask clarifying questions for each Input Value one by one, not all at once.
- When receiving DTMF_EVENT messages, interpret them as keypad events and respond naturally, not as normal user text.
- **If you receive any message with 'role = "system"', 'role = "tool"' or tool function response message, do not respond and react. Just reference it unless explicitly instructed to do so.**

Constraints:
- Avoid hallucinations; use tools or provided data for factual or external information.
- Maintain alignment with the user's persona, style, and tone.
- Respect conversation continuity and prior context.
- Never expose or echo tool responses or raw JSON data to the user.
`
	defaultDTMFEvent = "DTMF_EVENT"
)
