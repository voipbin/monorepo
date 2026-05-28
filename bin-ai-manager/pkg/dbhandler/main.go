package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/aiprompthistory"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/participant"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	AICreate(ctx context.Context, c *ai.AI) error
	AIDelete(ctx context.Context, id uuid.UUID) error
	AIGet(ctx context.Context, id uuid.UUID) (*ai.AI, error)
	AIList(ctx context.Context, size uint64, token string, filters map[ai.Field]any) ([]*ai.AI, error)
	AIUpdate(ctx context.Context, id uuid.UUID, fields map[ai.Field]any) error

	AIPromptHistoryCreate(ctx context.Context, h *aiprompthistory.AIPromptHistory) error
	AIPromptHistoryGet(ctx context.Context, id uuid.UUID) (*aiprompthistory.AIPromptHistory, error)
	AIPromptHistoryGetsByAIID(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*aiprompthistory.AIPromptHistory, error)

	AIcallCreate(ctx context.Context, cb *aicall.AIcall) error
	AIcallDelete(ctx context.Context, id uuid.UUID) error
	AIcallGet(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	AIcallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*aicall.AIcall, error)
	AIcallList(ctx context.Context, size uint64, token string, filters map[aicall.Field]any) ([]*aicall.AIcall, error)
	AIcallUpdate(ctx context.Context, id uuid.UUID, fields map[aicall.Field]any) error

	MessageCreate(ctx context.Context, c *message.Message) error
	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageList(ctx context.Context, size uint64, token string, filters map[message.Field]any) ([]*message.Message, error)
	MessageDelete(ctx context.Context, id uuid.UUID) error
	MessageUpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status message.DeliveryStatus) error
	MessageAssistantReplyExists(ctx context.Context, pipecatcallID uuid.UUID) (bool, error)

	SummaryCreate(ctx context.Context, c *summary.Summary) error
	SummaryGet(ctx context.Context, id uuid.UUID) (*summary.Summary, error)
	SummaryDelete(ctx context.Context, id uuid.UUID) error
	SummaryList(ctx context.Context, size uint64, token string, filters map[summary.Field]any) ([]*summary.Summary, error)
	SummaryUpdate(ctx context.Context, id uuid.UUID, fields map[summary.Field]any) error

	AIAuditUpsert(ctx context.Context, a *aiaudit.AIAudit) (rowsAffected int64, err error)
	AIAuditGet(ctx context.Context, id uuid.UUID) (*aiaudit.AIAudit, error)
	AIAuditList(ctx context.Context, size uint64, token string, filters map[aiaudit.Field]any) ([]*aiaudit.AIAudit, error)
	AIAuditDelete(ctx context.Context, id uuid.UUID) error
	AIAuditUpdateFinal(ctx context.Context, id uuid.UUID, status aiaudit.Status, overallScore *int, evaluation json.RawMessage, errStr string, messageIDs []uuid.UUID) (rowsAffected int64, err error)
	AIAuditCountProgressing(ctx context.Context, customerID uuid.UUID) (int64, error)

	TeamCreate(ctx context.Context, t *team.Team) error
	TeamDelete(ctx context.Context, id uuid.UUID) error
	TeamGet(ctx context.Context, id uuid.UUID) (*team.Team, error)
	TeamList(ctx context.Context, size uint64, token string, filters map[team.Field]any) ([]*team.Team, error)
	TeamUpdate(ctx context.Context, id uuid.UUID, fields map[team.Field]any) error

	// Participant
	ParticipantCreate(ctx context.Context, aicallID uuid.UUID, aiID uuid.UUID) error
	ParticipantListByAIcallID(ctx context.Context, aicallID uuid.UUID, size uint64, token string) ([]*participant.Participant, error)
	ParticipantListByAIID(ctx context.Context, aiID uuid.UUID, size uint64, token string) ([]*participant.Participant, error)
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("record not found")
)


// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
	return h
}
