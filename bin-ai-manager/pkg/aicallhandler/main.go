package aicallhandler

//go:generate mockgen -package aicallhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmdtmf "monorepo/bin-call-manager/models/dtmf"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	kmkase "monorepo/bin-contact-manager/models/kase"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	"monorepo/bin-ai-manager/pkg/participanthandler"
	"monorepo/bin-ai-manager/pkg/teamhandler"
	commonservice "monorepo/bin-common-handler/models/service"
)

// AIcallHandler define
type AIcallHandler interface {
	Delete(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	Get(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	GetSkipCache(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*aicall.AIcall, error)
	List(ctx context.Context, size uint64, token string, filters map[aicall.Field]any) ([]*aicall.AIcall, error)

	ProcessStart(ctx context.Context, cb *aicall.AIcall) (*aicall.AIcall, error)
	ProcessTerminate(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	ProcessListen(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	RunListenTurn(ctx context.Context, aicallID uuid.UUID)

	ToolHandle(ctx context.Context, id uuid.UUID, toolID string, toolType message.ToolType, function message.FunctionCall, pipecatcallID uuid.UUID) (map[string]any, error)

	Start(
		ctx context.Context,
		assistanceType aicall.AssistanceType,
		assistanceID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType aicall.ReferenceType,
		referenceID uuid.UUID,
	) (*aicall.AIcall, error)

	ServiceStart(
		ctx context.Context,
		assistanceType aicall.AssistanceType,
		assistanceID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType aicall.ReferenceType,
		referenceID uuid.UUID,
	) (*commonservice.Service, error)
	ServiceStartTypeTask(ctx context.Context, assistanceType aicall.AssistanceType, assistanceID uuid.UUID, activeflowID uuid.UUID) (*commonservice.Service, error)

	Send(ctx context.Context, id uuid.UUID, role message.Role, messageText string, runImmediately bool, audioResponse bool) (*message.Message, error)

	EventCMCallHangup(ctx context.Context, c *cmcall.Call)
	EventCMConfbridgeJoined(ctx context.Context, evt *cmconfbridge.EventConfbridgeJoined)
	EventCMConfbridgeLeaved(ctx context.Context, evt *cmconfbridge.EventConfbridgeLeaved)
	EventCMDTMFReceived(ctx context.Context, evt *cmdtmf.DTMF)
	EventPMPipecatcallInitialized(ctx context.Context, evt *pmpipecatcall.Pipecatcall)

	UpdateActiveflowID(ctx context.Context, id uuid.UUID, activeflowID uuid.UUID) (*aicall.AIcall, error)
	UpdatePipecatcallIDAndActiveflowID(ctx context.Context, id uuid.UUID, pipecatcallID uuid.UUID, activeflowID uuid.UUID) (*aicall.AIcall, error)
	UpdateCurrentMemberID(ctx context.Context, id uuid.UUID, currentMemberID uuid.UUID) (*aicall.AIcall, error)
}

const (
	variableID            = "voipbin.aicall.id"
	variableAIID          = "voipbin.aicall.ai_id"
	variableAIEngineModel = "voipbin.aicall.ai_engine_model"
	variableConfbridgeID  = "voipbin.aicall.confbridge_id"
	variableSTTLanguage   = "voipbin.aicall.stt_language"
	variablePipecatcallID = "voipbin.aicall.pipecatcall_id"
)

const (
	defaultPipecatcallTTSType = pmpipecatcall.TTSTypeElevenLabs
	defaultTTSType            = ai.TTSTypeElevenLabs
	defaultSTTType            = ai.STTTypeDeepgram

	defaultPipecatcallSTTType    = pmpipecatcall.STTTypeDeepgram
	defaultPipecatcallTTSVoiceID = "EXAVITQu4vr4xnSDxMaL" // Rachel

	defaultAITaskTimeout = 30000 // 30 seconds
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
	ai.TTSTypeGoogle:     "en-US-Chirp3-HD-Charon",                // Male, Chirp 3 HD (required for streaming synthesis). https://cloud.google.com/text-to-speech/docs/chirp3-hd
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

	// cache backs the Insight AI's realtime call listening: the transcribe ->
	// AIcall resolver set, the per-AIcall transcript buffers, the cross-replica
	// debounce lock, the turn counter, and ToolHandle's listen-turn membership
	// check. It is NOT a read-through cache for anything in this handler --
	// entity snapshots still go through db (dbhandler owns that). See
	// pkg/cachehandler/listen.go.
	cache cachehandler.CacheHandler

	aiHandler          aihandler.AIHandler
	teamHandler        teamhandler.TeamHandler
	messageHandler     messagehandler.MessageHandler
	participantHandler participanthandler.ParticipantHandler

	// runListenStartHook is the injected seam ProcessListen's detached
	// goroutine goes through (design §7 item 2: "runListenStart is detached, so
	// assert on a seam ... never on wall-clock timing").
	//
	// nil in production and in every handler NewAIcallHandler builds, in which
	// case runListenStart itself runs. Only the tests that must observe the
	// async stage deterministically set it.
	runListenStartHook func(ctx context.Context, a *ai.AI, c *aicall.AIcall, kase *kmkase.Case, callID uuid.UUID, call *cmcall.Call)
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
	promAIcallEndTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_end_total",
			Help:      "Total number of terminated aicalls by reference type.",
		},
		[]string{"reference_type"},
	)
	promAIcallDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_duration_seconds",
			Help:      "Duration of aicalls in seconds from creation to termination.",
			Buckets:   []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
		[]string{"reference_type"},
	)
	promAIcallToolExecuteTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "aicall_tool_execute_total",
			Help:      "Total number of tool executions by tool name.",
		},
		[]string{"tool_name"},
	)
)

func init() {
	prometheus.MustRegister(
		promAIcallCreateTotal,
		promAIcallEndTotal,
		promAIcallDurationSeconds,
		promAIcallToolExecuteTotal,
	)
}

// NewAIcallHandler creates a new AIHandler
func NewAIcallHandler(
	req requesthandler.RequestHandler,
	notify notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	cache cachehandler.CacheHandler,
	aiHandler aihandler.AIHandler,
	teamHandler teamhandler.TeamHandler,
	messageHandler messagehandler.MessageHandler,
	participantHandler participanthandler.ParticipantHandler,
) AIcallHandler {
	return &aicallHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    req,
		notifyHandler: notify,
		db:            db,
		cache:         cache,

		aiHandler:          aiHandler,
		teamHandler:        teamHandler,
		messageHandler:     messageHandler,
		participantHandler: participantHandler,
	}
}

const (
	defaultCommonAIcallSystemPrompt = `You are an AI assistant for VoIPBin. Follow the user's system/custom prompt strictly. Adapt to their persona, style, and tone.

Tool Usage:
- When a user requests an action you can perform with a tool, ACT IMMEDIATELY — invoke the tool right away.
- Do NOT describe what you could do or ask "would you like me to?". Just do it.
- Only ask for clarification if required parameters are genuinely missing.
- Use tool parameters exactly as specified.
- Never mention tool names, JSON, or backend logic to the user. Respond naturally: "I'll connect you now."

DTMF Events:
- Messages like "DTMF_EVENT: N" are telephone keypad presses. Treat as events, not text. Respond naturally per context.

Additional Data:
- You may receive JSON context data (call details, user profile, metadata). Use it to improve responses. Never expose, quote, or describe raw data to the user.

System/Tool Messages:
- If you receive messages with role "system" or "tool", or tool function responses, do not respond or react. Reference them silently unless explicitly instructed otherwise.

Response Rules:
- Ask clarifying questions one at a time, not all at once.
- Use tools or provided data for facts — avoid hallucinations.
- Maintain conversation continuity and prior context.
- Never expose raw JSON or tool responses to the user.`

	defaultCommonAItaskSystemPrompt = `
You are the AI engine for voipbin.
You operate as a headless, deterministic, sequential workflow executor.

## CRITICAL EXECUTION PROTOCOL (READ CAREFULLY)
1. **NO TEXT OUTPUT:** You must NOT output any text, reasoning, explanations, or chat messages in the 'content' field.
2. **NATIVE TOOL USE:** Do not write the function name as text (e.g., do not write "call: function_name"). Instead, you must strictly use the **Native Function Calling / Tool Use** feature provided by the platform.
3. **SEQUENTIAL EXECUTION:**
   - Analyze the request internally.
   - Trigger the necessary tool function immediately.

## CORE OPERATING RULES

1. **Request Analysis**
   - Identify the final objective and required data sources.
   - If data is missing, call the retrieval tool immediately.

2. **Parameter Defaults (CRITICAL)**
   - **run_llm:** You MUST explicitly set the 'run_llm' parameter to 'true' in every tool call by default, unless the user has specifically requested silent execution (e.g., "do this silently").

3. **Tool Dependency Enforcement**
   - NEVER guess data.
   - If Tool B depends on Tool A, call Tool A -> Wait for system response -> Then call Tool B.

## ERROR HANDLING
- If a tool returns invalid data, stop and wait. Do not generate text explanations.

## TERMINATION
- Call 'stop_service' ONLY when the request is fully completed.

## CURRENT INSTRUCTION
Analyze the user input below and EXECUTE the required tool function immediately.
Keep the message content empty.
`

	defaultDTMFEvent = "DTMF_EVENT"

	// InsightSystemPrompt is the dedicated system prompt used for ai.TypeInsight
	// AIs, in place of defaultCommonAIcallSystemPrompt / defaultCommonAItaskSystemPrompt.
	// Insight AIs are restricted to the Insight tool set (see
	// tool.AllInsightToolNames) and are used to analyze existing contact/
	// conversation data rather than to drive a live call or task workflow.
	InsightSystemPrompt = `You are an Insight AI for VoIPBin. Your role is to analyze existing contact and conversation
data to answer questions and surface insights — you do not drive a live call or execute a task workflow.

Tool Usage:
- Use the available Insight tools (e.g. get_contact_interactions, get_conversation_content) to retrieve the
  data you need before answering. Do not guess or fabricate data.
- Only ask for clarification if required parameters are genuinely missing.
- Never mention tool names, JSON, or backend logic to the user. Respond naturally with your findings.

Response Rules:
- Base every answer strictly on retrieved data — avoid hallucinations.
- Be concise and factual. Cite specific interactions/conversations when relevant.
- Never expose raw JSON or tool responses to the user.`

	// ListenTurnSystemPrompt supplies the MECHANICS of a listen evaluation turn
	// and nothing else. The business conditions -- what actually warrants
	// speaking up -- come entirely from the customer's own init_prompt, which is
	// message #2 of every listen turn. That split is deliberate and is the whole
	// point of the feature: triggering is customer-configurable without a schema
	// change and without a hardcoded rule set.
	ListenTurnSystemPrompt = `You are silently monitoring a live phone call in progress. You are NOT talking to anyone right now.

Below you will see a rolling window of what has been said so far, tagged by speaker. Lines after the "--- NEW SINCE YOUR LAST CHECK ---" marker are what you have not evaluated yet; everything before it you have already considered on a previous check.

Your task on each check:
1. Read the new lines in the context of the conversation so far.
2. Decide whether the instructions in your configured prompt warrant alerting the human agent RIGHT NOW.
3. If and only if they do, call the notify_agent tool with one or two sentences written for a busy human mid-call.

CRITICAL RULES:
- Saying nothing is the correct and expected outcome for most checks. Do not manufacture something to say.
- notify_agent is the ONLY way to reach the agent. Any text you produce instead of a tool call is discarded and nobody will ever see it.
- Never repeat a notification you already sent on this call. Check the conversation above before notifying.
- Do not summarize the call, do not narrate what is happening, and do not greet anyone. You are not a participant.
- Do not use other tools unless answering the alert genuinely requires information the transcript does not contain.`
)
