package summaryhandler

//go:generate mockgen -package summaryhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	cmcall "monorepo/bin-call-manager/models/call"
	commonservice "monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cfconference "monorepo/bin-conference-manager/models/conference"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sashabaranov/go-openai"
)

type SummaryHandler interface {
	Start(
		ctx context.Context,
		customerID uuid.UUID,
		activeflowID uuid.UUID,
		onEndFlowID uuid.UUID,
		referenceType summary.ReferenceType,
		referenceID uuid.UUID,
		language string,
	) (*summary.Summary, error)
	Get(ctx context.Context, id uuid.UUID) (*summary.Summary, error)
	List(ctx context.Context, size uint64, token string, filters map[summary.Field]any) ([]*summary.Summary, error)
	Delete(ctx context.Context, id uuid.UUID) (*summary.Summary, error)

	ServiceStart(
		ctx context.Context,
		customerID uuid.UUID,
		activeflowID uuid.UUID,
		onEndFlowID uuid.UUID,
		referenceType summary.ReferenceType,
		referenceID uuid.UUID,
		language string,
	) (*commonservice.Service, error)

	EventCMCallHangup(ctx context.Context, c *cmcall.Call)
	EventCMConferenceUpdated(ctx context.Context, c *cfconference.Conference)
}

type summaryHandler struct {
	utilHandler   utilhandler.UtilHandler
	notifyHandler notifyhandler.NotifyHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler

	engineOpenaiHandler engine_openai_handler.EngineOpenaiHandler
}

var (
	metricsNamespace = "ai_manager"

	// summary_start_total counts summary starts by reference type.
	promSummaryStartTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "summary_start_total",
			Help:      "Total number of summary starts by reference type.",
		},
		[]string{"reference_type"},
	)

	// summary_done_total counts summary completions by reference type.
	promSummaryDoneTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "summary_done_total",
			Help:      "Total number of summary completions by reference type.",
		},
		[]string{"reference_type"},
	)
)

func init() {
	prometheus.MustRegister(
		promSummaryStartTotal,
		promSummaryDoneTotal,
	)
}

func NewSummaryHandler(
	requestHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,

	engineOpenaiHandler engine_openai_handler.EngineOpenaiHandler,
) SummaryHandler {
	return &summaryHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    requestHandler,
		notifyHandler: notifyHandler,
		db:            db,

		engineOpenaiHandler: engineOpenaiHandler,
	}
}

// list of variables
const (
	variableSummaryID            = "voipbin.ai_summary.id"
	variableSummaryReferenceType = "voipbin.ai_summary.reference_type"
	variableSummaryReferenceID   = "voipbin.ai_summary.reference_id"
	variableSummaryLanguage      = "voipbin.ai_summary.language"
	variableSummaryContent       = "voipbin.ai_summary.content"
)

const (
	defaultModel = openai.GPT4Turbo

	// defaultVerifyModel is the cheaper model used for the summary output-language
	// verification harness (yes/no detection). Kept separate from defaultModel:
	// a top-tier model is unnecessary for a one-word judgement.
	defaultVerifyModel = openai.GPT4oMini

	// maxSummaryRegenerations bounds the output-language retry loop. Total
	// generation attempts = 1 + maxSummaryRegenerations (= 3).
	maxSummaryRegenerations = 2

	// languageVerifySampleLen caps how many leading runes of the summary are
	// sent to the verifier (rune-based, never byte-sliced).
	languageVerifySampleLen = 1500

	// languageVerifyMinProse is the minimum prose length (in runes, excluding
	// section headers and "- None" items) required to run verification. Below
	// this, verification is skipped and treated as a pass.
	languageVerifyMinProse = 20

	// verifyTimeout bounds a single verification call (via SendOnce, which has
	// no backoff so this deadline actually holds).
	verifyTimeout = 10 * time.Second

	// englishPrimarySubtag: when the effective output language's canonical
	// primary subtag equals this, verification is skipped (see design §5.1.1).
	englishPrimarySubtag = "en"
)

const (
	// languageSystemPromptFmt is the first-pass (primary) enforcement: a system
	// message that pins the output language by value (%s = effective BCP47 lang),
	// removing the previous self-reference indirection.
	languageSystemPromptFmt = "You are a call summary assistant. You MUST write the ENTIRE summary, including every section body, in %s (BCP47). This language requirement is absolute and overrides the language of the transcript or any other input. Section labels follow the user instructions."

	// languageVerifyPrompt is the second-pass (safety net) verifier prompt.
	// %s (1) = BCP47 code, %s (2) = sampled prose.
	languageVerifyPrompt = "You are a strict language detector. Answer with exactly one word: yes or no. Ignore section headings/labels, proper nouns, product names, URLs, code, and technical terms — judge only the natural-language prose. Is the following text written mainly in the language with BCP47 code %s? TEXT:\n%s"
)

const (
	defaultSummaryGeneratePrompt = `
Generate a structured and concise call summary based on the provided transcription, recording link, conference details, and other relevant variables.

Output format rules (follow strictly):
- Write in plain text only. Do NOT use any Markdown syntax: no asterisks for bold or emphasis, no leading '#' headings, no backticks.
- Write each section title as plain words followed by a colon (for example, "Call Type:").
- Separate sections with a single blank line.
- Under a section, put each item on its own line starting with a hyphen and a space ("- ").
- The summary must be easy to copy and paste into an email, ticket, or note as clean plain text.

Language:
- Follow the system message's language requirement.

Always produce all of the following sections, in this exact order, even when the transcription is short, low quality, or empty. Never skip a section and never replace the whole summary with a single sentence. When a section has nothing to report, write exactly one item under it: "- None".

- Call Type: A "reference_type" value is provided in the input. Use ONLY that value to state the call type, and translate the label into the summary's output language. Map exactly: "call" -> "Call", "conference" -> "Conference", "recording" -> "Recorded Call", "transcribe" -> "Transcribed Call". Do NOT infer the type from the transcription. If "reference_type" is missing or empty, write "- Unknown".
- Key Discussion Points: Summarize any meaningful content, even briefly. Ignore filler, but do your best to capture what was said. If nothing meaningful was said, write "- None".
- Important Decisions & Agreements: Highlight confirmed agreements, resolutions, or commitments.
- Action Items & Next Steps: List concrete follow-up tasks and responsible parties.
- Additional Notes: Add relevant timestamps or contextual information if helpful.
`
)
