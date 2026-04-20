package healthcheckhandler

//go:generate mockgen -package healthcheckhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

// HealthCheckHandler manages periodic SIP OPTIONS health checks for providers.
type HealthCheckHandler interface {
	Run(ctx context.Context, interval time.Duration)
}

type healthCheckHandler struct {
	db         dbhandler.DBHandler
	reqHandler requesthandler.RequestHandler
}

// NewHealthCheckHandler creates a HealthCheckHandler.
func NewHealthCheckHandler(db dbhandler.DBHandler, reqHandler requesthandler.RequestHandler) HealthCheckHandler {
	return &healthCheckHandler{
		db:         db,
		reqHandler: reqHandler,
	}
}
