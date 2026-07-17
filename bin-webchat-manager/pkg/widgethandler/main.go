package widgethandler

//go:generate mockgen -package widgethandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-webchat-manager/models/widget"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
)

// WidgetHandler interface
type WidgetHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		welcomeMessage string,
		sessionFlowID uuid.UUID,
		messageFlowID uuid.UUID,
		sessionIdleTimeout int,
		themeConfig *widget.ThemeConfig,
	) (*widget.Widget, error)
	Get(ctx context.Context, id uuid.UUID) (*widget.Widget, error)
	List(ctx context.Context, size uint64, token string, filters map[widget.Field]any) ([]*widget.Widget, error)
	UpdateBasicInfo(
		ctx context.Context,
		id uuid.UUID,
		name string,
		welcomeMessage string,
		sessionFlowID uuid.UUID,
		messageFlowID uuid.UUID,
		sessionIdleTimeout int,
		themeConfig *widget.ThemeConfig,
	) (*widget.Widget, error)
	Delete(ctx context.Context, id uuid.UUID) (*widget.Widget, error)
	DirectHashRegenerate(ctx context.Context, id uuid.UUID) (*widget.Widget, error)
}

type widgetHandler struct {
	utilHandler utilhandler.UtilHandler
	reqHandler  requesthandler.RequestHandler
	db          dbhandler.DBHandler
}

// NewWidgetHandler returns WidgetHandler interface
func NewWidgetHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
) WidgetHandler {
	return &widgetHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  reqHandler,
		db:          dbHandler,
	}
}
