package summaryhandler

//go:generate mockgen -package summaryhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sashabaranov/go-openai"
)

type SummaryHandler interface {
	Start(
		ctx context.Context,
		customerID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType summary.ReferenceType,
		referenceID uuid.UUID,
		language string,
	) (*summary.Summary, error)
	Get(ctx context.Context, id uuid.UUID) (*summary.Summary, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*summary.Summary, error)
	Delete(ctx context.Context, id uuid.UUID) (*summary.Summary, error)
}

type summaryHandler struct {
	utilHandler   utilhandler.UtilHandler
	notifyHandler notifyhandler.NotifyHandler
	reqestHandler requesthandler.RequestHandler
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
		reqestHandler: requestHandler,
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
