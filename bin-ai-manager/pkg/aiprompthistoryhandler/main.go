package aiprompthistoryhandler

//go:generate mockgen -package aiprompthistoryhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aiprompthistory"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// AIPromptHistoryHandler provides read access to AI prompt history.
type AIPromptHistoryHandler interface {
	List(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*aiprompthistory.AIPromptHistory, error)
	Get(ctx context.Context, aiID uuid.UUID, historyID uuid.UUID) (*aiprompthistory.AIPromptHistory, error)
}

type aiprompthistoryHandler struct {
	db          dbhandler.DBHandler
	utilHandler utilhandler.UtilHandler
}

// New returns a new AIPromptHistoryHandler.
func New(db dbhandler.DBHandler, util utilhandler.UtilHandler) AIPromptHistoryHandler {
	return &aiprompthistoryHandler{
		db:          db,
		utilHandler: util,
	}
}
