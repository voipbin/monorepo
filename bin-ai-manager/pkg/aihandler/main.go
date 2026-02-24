package aihandler

//go:generate mockgen -package aihandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// AIHandler interface
type AIHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		engineModel ai.EngineModel,
		engineData map[string]any,
		engineKey string,
		initPrompt string,
		ttsType ai.TTSType,
		ttsVoiceID string,
		sttType ai.STTType,
		toolNames []tool.ToolName,
	) (*ai.AI, error)
	Get(ctx context.Context, id uuid.UUID) (*ai.AI, error)
	List(ctx context.Context, size uint64, token string, filters map[ai.Field]any) ([]*ai.AI, error)
	Delete(ctx context.Context, id uuid.UUID) (*ai.AI, error)
	Update(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		engineModel ai.EngineModel,
		engineData map[string]any,
		engineKey string,
		initPrompt string,
		ttsType ai.TTSType,
		ttsVoice string,
		sttType ai.STTType,
		toolNames []tool.ToolName,
	) (*ai.AI, error)
}

// aiHandler structure for service handle
type aiHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
}

// NewAIHandler define
func NewAIHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
) AIHandler {
	return &aiHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
		db:            db,
	}
}
