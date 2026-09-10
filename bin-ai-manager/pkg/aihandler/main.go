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
		aiType ai.Type,
		engineModel ai.EngineModel,
		parameter map[string]any,
		engineKey string,
		ragID uuid.UUID,
		initPrompt string,
		ttsType ai.TTSType,
		ttsVoiceID string,
		sttType ai.STTType,
		sttLanguage string,
		toolNames []tool.ToolName,
		vadConfig *ai.VADConfig,
		smartTurnEnabled bool,
		autoAICallAuditEnabled bool,
	) (*ai.AI, error)
	Get(ctx context.Context, id uuid.UUID) (*ai.AI, error)
	List(ctx context.Context, size uint64, token string, filters map[ai.Field]any) ([]*ai.AI, error)
	Delete(ctx context.Context, id uuid.UUID) (*ai.AI, error)
	Update(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		aiType ai.Type,
		engineModel ai.EngineModel,
		parameter map[string]any,
		engineKey string,
		ragID uuid.UUID,
		initPrompt string,
		ttsType ai.TTSType,
		ttsVoice string,
		sttType ai.STTType,
		sttLanguage string,
		toolNames []tool.ToolName,
		vadConfig *ai.VADConfig,
		smartTurnEnabled bool,
		autoAICallAuditEnabled bool,
	) (*ai.AI, error)
	ActivateInsight(ctx context.Context, id uuid.UUID) (*ai.AI, error)
	DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*ai.AI, error)

	// ValidateMcpServerIDs checks that every id in ids refers to an
	// existing McpServer row owned by customerID (IDOR guard). Callers
	// MUST call this before UpdateMcpServerIDs whenever the ids come from
	// an untrusted request body.
	ValidateMcpServerIDs(ctx context.Context, customerID uuid.UUID, ids []uuid.UUID) error

	// UpdateMcpServerIDs persists the McpServerIDs whitelist onto the AI.
	UpdateMcpServerIDs(ctx context.Context, id uuid.UUID, mcpServerIDs []uuid.UUID) (*ai.AI, error)
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
