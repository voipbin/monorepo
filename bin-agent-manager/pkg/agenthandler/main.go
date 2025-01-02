package agenthandler

//go:generate mockgen -package agenthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/dbhandler"
)

const (
	defaultPasswordHashCost = 10
)

// AgentHandler interface
type AgentHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, username, password, name, detail string, ringMethod agent.RingMethod, permission agent.Permission, tagIDs []uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error)
	Delete(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	Get(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	GetByCustomerIDAndAddress(ctx context.Context, customerID uuid.UUID, addr *commonaddress.Address) (*agent.Agent, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*agent.Agent, error)
	Login(ctx context.Context, username, password string) (*agent.Agent, error)
	UpdateAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error)
	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, ringMethod agent.RingMethod) (*agent.Agent, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, password string) (*agent.Agent, error)
	UpdatePermission(ctx context.Context, id uuid.UUID, permission agent.Permission) (*agent.Agent, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status agent.Status) (*agent.Agent, error)
	UpdateTagIDs(ctx context.Context, id uuid.UUID, tags []uuid.UUID) (*agent.Agent, error)

	EventGroupcallCreated(ctx context.Context, groupcall *cmgroupcall.Groupcall) error
	EventGroupcallProgressing(ctx context.Context, groupcall *cmgroupcall.Groupcall) error
	EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error
	EventCustomerCreated(ctx context.Context, cu *cmcustomer.Customer) error
}

type agentHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

// NewAgentHandler return AgentHandler interface
func NewAgentHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) AgentHandler {
	return &agentHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyHandler: notifyHandler,
	}
}
