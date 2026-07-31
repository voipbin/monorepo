package schedulehandler

//go:generate mockgen -package schedulehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-scheduler-manager/models/schedule"
	"monorepo/bin-scheduler-manager/pkg/dbhandler"
)

// ScheduleHandler interface
type ScheduleHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		cronExpr string,
		targetQueue string,
		targetURI string,
		targetMethod string,
		targetDataType string,
		targetData json.RawMessage,
		timeoutMS int,
		retryMax int,
		enabled bool,
	) (*schedule.Schedule, error)
	Get(ctx context.Context, id uuid.UUID) (*schedule.Schedule, error)
	Gets(ctx context.Context, size uint64, token string, filters map[schedule.Field]any) ([]*schedule.Schedule, error)
	Update(ctx context.Context, id uuid.UUID, fields map[schedule.Field]any) (*schedule.Schedule, error)
	Delete(ctx context.Context, id uuid.UUID) (*schedule.Schedule, error)
}

type scheduleHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

// NewScheduleHandler returns ScheduleHandler interface
func NewScheduleHandler(reqHandler requesthandler.RequestHandler, db dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) ScheduleHandler {
	return &scheduleHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            db,
		notifyHandler: notifyHandler,
	}
}
