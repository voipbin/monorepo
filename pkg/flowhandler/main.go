package flowhandler

//go:generate mockgen -destination ./mock_flowhandler_flowhandler.go -package flowhandler gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler FlowHandler

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/flow"
)

type flowHandler struct {
	db dbhandler.DBHandler
}

// FlowHandler interface
type FlowHandler interface {
	ActionGet(ctx context.Context, flowID uuid.UUID, actionID uuid.UUID) (*action.Action, error)

	ActiveFlowCreate(ctx context.Context, callID, flowID uuid.UUID) (*activeflow.ActiveFlow, error)

	FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowCreate(ctx context.Context, flow *flow.Flow, persist bool) (*flow.Flow, error)
}

// NewFlowHandler return FlowHandler
func NewFlowHandler(db dbhandler.DBHandler) FlowHandler {
	h := &flowHandler{
		db: db,
	}

	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
