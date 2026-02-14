package allowancehandler

//go:generate mockgen -package allowancehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/allowance"
	"monorepo/bin-billing-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// AllowanceHandler interface for managing monthly token allowances
type AllowanceHandler interface {
	GetCurrentCycle(ctx context.Context, accountID uuid.UUID) (*allowance.Allowance, error)
	EnsureCurrentCycle(ctx context.Context, accountID uuid.UUID, customerID uuid.UUID, planType account.PlanType) (*allowance.Allowance, error)
	ConsumeTokens(ctx context.Context, accountID uuid.UUID, tokensNeeded int, creditPerUnit float32, tokenPerUnit int) (tokensConsumed int, creditCharged float32, err error)
	ListByAccountID(ctx context.Context, accountID uuid.UUID, size uint64, token string) ([]*allowance.Allowance, error)
	ProcessAllCycles(ctx context.Context) error
}

type allowanceHandler struct {
	db          dbhandler.DBHandler
	utilHandler utilhandler.UtilHandler
}

// NewAllowanceHandler creates AllowanceHandler
func NewAllowanceHandler(db dbhandler.DBHandler) AllowanceHandler {
	return &allowanceHandler{
		db:          db,
		utilHandler: utilhandler.NewUtilHandler(),
	}
}
