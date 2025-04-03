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
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*summary.Summary, error)
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

**Language:**  
- Generate the summary in the language specified in 'voipbin.summary.language', regardless of the transcription's language.  

**Formatting:**  
1. **Call Type**: Identify if it was a 1:1 call, conference, support call, sales call, or recorded call.  
2. **Key Discussion Points**: Summarize only meaningful conversations. Ignore small talk, random words, or numerical sequences without context.  
3. **Important Decisions & Agreements**: Highlight confirmed agreements, resolutions, or commitments.  
4. **Action Items & Next Steps**: List only concrete follow-up tasks and responsible parties.  
5. **Additional Notes** (if applicable): Add relevant timestamps or contextual information if needed.  

**Conditions:**  
- If no transcription is provided, do not generate a summary.  
- If the transcription contains only unrelated numbers or words without context, return: "No meaningful content available for summary."  
`
)
