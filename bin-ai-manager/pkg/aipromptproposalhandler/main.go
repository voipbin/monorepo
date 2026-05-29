package aipromptproposalhandler

//go:generate mockgen -package aipromptproposalhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/geminiproposalhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

const (
	maxConcurrentGlobal        = 30
	maxConcurrentCustomer      = 3
	geminiTimeoutSeconds       = 60
	maxAuditsPerProposal       = 20
	maxTranscriptCharsPerAudit = 15000
	staleProposalAgeMinutes    = 5
	proposalExpiryHours        = 168
	maxProposedPromptChars     = 32000
)

// AIPromptProposalHandler handles AI prompt proposal lifecycle operations.
type AIPromptProposalHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, aiID uuid.UUID, auditIDs []uuid.UUID, language string) (*aipromptproposal.AIPromptProposal, error)
	Get(ctx context.Context, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error)
	List(ctx context.Context, size uint64, token string, filters map[aipromptproposal.Field]any) ([]*aipromptproposal.AIPromptProposal, error)
	Accept(ctx context.Context, customerID uuid.UUID, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error)
	Reject(ctx context.Context, customerID uuid.UUID, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error)
	Delete(ctx context.Context, id uuid.UUID) (*aipromptproposal.AIPromptProposal, error)
	SweepStaleProposals(ctx context.Context)
	SweepExpiredProposals(ctx context.Context)
}

type aipromptproposalHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	geminiHandler geminiproposalhandler.GeminiProposalHandler
	semaphore     chan struct{}
}

// NewAIPromptProposalHandler creates a new handler.
func NewAIPromptProposalHandler(
	db dbhandler.DBHandler,
	geminiHandler geminiproposalhandler.GeminiProposalHandler,
) *aipromptproposalHandler {
	return &aipromptproposalHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		geminiHandler: geminiHandler,
		semaphore:     make(chan struct{}, maxConcurrentGlobal),
	}
}

// Suppress unused-import until later tasks add code.
var _ = time.Second
