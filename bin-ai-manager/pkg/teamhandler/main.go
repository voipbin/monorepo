package teamhandler

//go:generate mockgen -package teamhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

// TeamHandler provides CRUD operations for Team resources.
type TeamHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []team.Member, parameter map[string]any) (*team.Team, error)
	Get(ctx context.Context, id uuid.UUID) (*team.Team, error)
	List(ctx context.Context, size uint64, token string, filters map[team.Field]any) ([]*team.Team, error)
	Delete(ctx context.Context, id uuid.UUID) (*team.Team, error)
	Update(ctx context.Context, id uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []team.Member, parameter map[string]any) (*team.Team, error)
}

type teamHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
}

// NewTeamHandler creates a new TeamHandler
func NewTeamHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
) TeamHandler {
	return &teamHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
		db:            db,
	}
}
