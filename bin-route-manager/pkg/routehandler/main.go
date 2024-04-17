package routehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package routehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-route-manager/models/route"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

// routeHandler
type routeHandler struct {
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

// RouteHandler interface
type RouteHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*route.Route, error)
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		providerID uuid.UUID,
		priority int,
		target string,
	) (*route.Route, error)
	GetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*route.Route, error)
	GetsByTarget(ctx context.Context, customerID uuid.UUID, target string) ([]*route.Route, error)
	Delete(ctx context.Context, id uuid.UUID) (*route.Route, error)
	Update(ctx context.Context, id uuid.UUID, name string, detail string, providerID uuid.UUID, priority int, target string) (*route.Route, error)

	// dialroute
	DialrouteGets(ctx context.Context, customerID uuid.UUID, target string) ([]*route.Route, error)
}

// NewRouteHandler return RouteHandler
func NewRouteHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
) RouteHandler {
	h := &routeHandler{
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}

	return h
}
