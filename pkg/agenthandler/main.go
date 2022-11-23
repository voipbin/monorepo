package agenthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package agenthandler -destination ./mock_agenthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"golang.org/x/crypto/bcrypt"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentdial"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/dbhandler"
)

const (
	maxAgentCount = 999 // maximum agent numbers
)

// AgentHandler interface
type AgentHandler interface {
	AgentCreate(ctx context.Context, customerID uuid.UUID, username, password, name, detail string, ringMethod agent.RingMethod, permission agent.Permission, tagIDs []uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error)
	AgentDelete(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	AgentDial(ctx context.Context, id uuid.UUID, source *commonaddress.Address, confbridgeID, masterCallID uuid.UUID) (*agentdial.AgentDial, error)
	AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	AgentGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*agent.Agent, error)
	AgentGetsByTagIDs(ctx context.Context, customerID uuid.UUID, tags []uuid.UUID) ([]*agent.Agent, error)
	AgentGetsByTagIDsAndStatus(ctx context.Context, customerID uuid.UUID, tags []uuid.UUID, status agent.Status) ([]*agent.Agent, error)
	AgentLogin(ctx context.Context, customerID uuid.UUID, username, password string) (*agent.Agent, error)
	AgentUpdateAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) (*agent.Agent, error)
	AgentUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, ringMethod agent.RingMethod) (*agent.Agent, error)
	AgentUpdatePassword(ctx context.Context, id uuid.UUID, password string) (*agent.Agent, error)
	AgentUpdatePermission(ctx context.Context, id uuid.UUID, permission agent.Permission) (*agent.Agent, error)
	AgentUpdateStatus(ctx context.Context, id uuid.UUID, status agent.Status) (*agent.Agent, error)
	AgentUpdateTagIDs(ctx context.Context, id uuid.UUID, tags []uuid.UUID) (*agent.Agent, error)

	AgentCallAnswered(ctx context.Context, c *cmcall.Call) error
	AgentCallHungup(ctx context.Context, c *cmcall.Call) error
}

type agentHandler struct {
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyhandler notifyhandler.NotifyHandler
}

// NewAgentHandler return AgentHandler interface
func NewAgentHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) AgentHandler {
	return &agentHandler{
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

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

func contains(s []uuid.UUID, x uuid.UUID) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}
