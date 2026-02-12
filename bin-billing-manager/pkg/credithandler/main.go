package credithandler

//go:generate mockgen -package credithandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-billing-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// FreeTierCreditAmount is the maximum free credit amount in USD.
const FreeTierCreditAmount float32 = 1.00

// CreditHandler interface for free tier credit processing
type CreditHandler interface {
	ProcessAll(ctx context.Context) error
}

// handler implements CreditHandler
type handler struct {
	db          dbhandler.DBHandler
	utilHandler utilhandler.UtilHandler
}

// NewCreditHandler creates CreditHandler
func NewCreditHandler(db dbhandler.DBHandler) CreditHandler {
	return &handler{
		db:          db,
		utilHandler: utilhandler.NewUtilHandler(),
	}
}
