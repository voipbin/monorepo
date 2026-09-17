package summaryhandler

//go:generate mockgen -package summaryhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
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
- Generate the summary in the language specified in 'voipbin.ai_summary.language', regardless of the transcription's language.

Always produce all of the following sections, in this exact order, even when the transcription is short, low quality, or empty. Never skip a section and never replace the whole summary with a single sentence. When a section has nothing to report, write exactly one item under it: "- None".

- Call Type: A "reference_type" value is provided in the input. Use ONLY that value to state the call type, and translate the label into the summary's output language. Map exactly: "call" -> "Call", "conference" -> "Conference", "recording" -> "Recorded Call", "transcribe" -> "Transcribed Call". Do NOT infer the type from the transcription. If "reference_type" is missing or empty, write "- Unknown".
- Key Discussion Points: Summarize any meaningful content, even briefly. Ignore filler, but do your best to capture what was said. If nothing meaningful was said, write "- None".
- Important Decisions & Agreements: Highlight confirmed agreements, resolutions, or commitments.
- Action Items & Next Steps: List concrete follow-up tasks and responsible parties.
- Additional Notes: Add relevant timestamps or contextual information if helpful.
`
)
