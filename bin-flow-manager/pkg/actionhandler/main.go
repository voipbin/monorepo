package actionhandler

//go:generate mockgen -package actionhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-flow-manager/models/action"
)

// actionHandler defines
type actionHandler struct {
	utilHandler utilhandler.UtilHandler
}

// ActionHandler fefines
type ActionHandler interface {
	ValidateActions(actions []action.Action) error
	ActionFetchGet(act *action.Action, activeflowID uuid.UUID, referenceID uuid.UUID) ([]action.Action, error)
	GenerateFlowActions(ctx context.Context, actions []action.Action) ([]action.Action, error)
}

// NewActionHandler returns ActionHandler
func NewActionHandler() ActionHandler {
	return &actionHandler{
		utilHandler: utilhandler.NewUtilHandler(),
	}
}
