package flowhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package flowhandler -destination ./mock_flowhandler_flowhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

type flowHandler struct {
	db         dbhandler.DBHandler
	reqHandler requesthandler.RequestHandler
}

// FlowHandler interface
type FlowHandler interface {
	ActionGet(ctx context.Context, flowID uuid.UUID, actionID uuid.UUID) (*action.Action, error)

	ActiveFlowCreate(ctx context.Context, callID, flowID uuid.UUID) (*activeflow.ActiveFlow, error)
	ActiveFlowNextActionGet(ctx context.Context, callID uuid.UUID, caID uuid.UUID) (*action.Action, error)
	ActiveFlowSetForwardActionID(ctx context.Context, callID uuid.UUID, actionID uuid.UUID, forwardNow bool) error

	FlowCreate(ctx context.Context, f *flow.Flow) (*flow.Flow, error)
	FlowDelete(ctx context.Context, id uuid.UUID) error
	FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*flow.Flow, error)
	FlowUpdate(ctx context.Context, f *flow.Flow) (*flow.Flow, error)

	ValidateActions(actions []action.Action) error
}

// list of default values
const (
	defaultTimeStamp = "9999-01-01 00:00:000"
)

// NewFlowHandler return FlowHandler
func NewFlowHandler(db dbhandler.DBHandler, reqHandler requesthandler.RequestHandler) FlowHandler {
	h := &flowHandler{
		db:         db,
		reqHandler: reqHandler,
	}

	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
