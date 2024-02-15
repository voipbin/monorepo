package agenthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package agenthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	"golang.org/x/crypto/bcrypt"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/dbhandler"
)

// AgentHandler interface
type AgentHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, username, password, name, detail string, ringMethod agent.RingMethod, permission agent.Permission, tagIDs []uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error)
	Delete(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	Get(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
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
}

type agentHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyhandler notifyhandler.NotifyHandler
}

// NewAgentHandler return AgentHandler interface
func NewAgentHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) AgentHandler {
	return &agentHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyhandler: notifyHandler,
	}
}

// checkHash returns true if the given hashstring is correct
func checkHash(password, hashString string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hashString), []byte(password)); err != nil {
		return false
	}

	return true
}

// GenerateHash generates hash from auth
func generateHash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}
