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
	variableSummaryID            = "voipbin.summary.id"
	variableSummaryReferenceType = "voipbin.summary.reference_type"
	variableSummaryReferenceID   = "voipbin.summary.reference_id"
	variableSummaryLanguage      = "voipbin.summary.language"
	variableSummaryContent       = "voipbin.summary.content"
)

const (
	defaultModel = openai.GPT4Turbo
)

const (
	defaultSummaryGeneratePrompt = `
Generate a structured and concise call summary based on the provided transcription, recording link, conference details, and other relevant variables.

Format the summary as follows:
1. **Call Type**: Specify whether it was a 1:1 call, conference, support call, sales call, or recorded call.
2. **Key Discussion Points**: Summarize the main topics covered in the call.
3. **Important Decisions & Agreements**: Highlight any agreements, resolutions, or key takeaways.
4. **Action Items & Next Steps**: Clearly list any follow-up tasks or required actions.
5. **Additional Notes** (if applicable): Mention any other important details such as timestamps for key moments in recorded calls.

**If no transcription is provided, do not generate a summary.**  
**Generate the summary in the language of the provided transcription.** Ensure that the summary remains clear, concise, and actionable.
`
)
