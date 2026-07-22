package sessionhandler

//go:generate mockgen -package sessionhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
	"monorepo/bin-webchat-manager/pkg/widgethandler"
)

// SessionHandler interface
type SessionHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, widgetID uuid.UUID, pageURL string) (*session.Session, error)
	Get(ctx context.Context, id uuid.UUID) (*session.Session, error)
	List(ctx context.Context, size uint64, token string, filters map[session.Field]any) ([]*session.Session, error)
	Delete(ctx context.Context, id uuid.UUID) (*session.Session, error)
	End(ctx context.Context, id uuid.UUID) (*session.Session, error)
}

type sessionHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
	widgetHandler widgethandler.WidgetHandler
}

// NewSessionHandler returns SessionHandler interface
func NewSessionHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	dbHandler dbhandler.DBHandler,
	widgetHandler widgethandler.WidgetHandler,
) SessionHandler {
	return &sessionHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
		db:            dbHandler,
		widgetHandler: widgetHandler,
	}
}
