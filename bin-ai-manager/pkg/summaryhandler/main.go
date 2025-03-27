package summaryhandler

//go:generate mockgen -package summaryhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/sashabaranov/go-openai"
)

type SummaryHandler interface {
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
