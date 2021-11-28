package agenthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package agenthandler -destination ./mock_agenthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"
	"golang.org/x/crypto/bcrypt"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/notifyhandler"
)

const (
	maxAgentCount = 999 // maximum agent numbers
)

// List of default values
const (
	defaultTimeStamp = "9999-01-01 00:00:00.000000" // default timestamp
)

// AgentHandler interface
type AgentHandler interface {
	AgentCreate(ctx context.Context, userID uint64, username, password, name, detail string, ringMethod agent.RingMethod, permission agent.Permission, tagIDs []uuid.UUID, addresses []cmaddress.Address) (*agent.Agent, error)
	AgentDelete(ctx context.Context, id uuid.UUID) error
	AgentDial(ctx context.Context, id uuid.UUID, source *cmaddress.Address, confbridgeID uuid.UUID) error
	AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	AgentGets(ctx context.Context, userID, size uint64, token string) ([]*agent.Agent, error)
	AgentGetsByTagIDs(ctx context.Context, userID uint64, tags []uuid.UUID) ([]*agent.Agent, error)
	AgentGetsByTagIDsAndStatus(ctx context.Context, userID uint64, tags []uuid.UUID, status agent.Status) ([]*agent.Agent, error)
	AgentLogin(ctx context.Context, userID uint64, username, password string) (*agent.Agent, error)
	AgentUpdateAddresses(ctx context.Context, id uuid.UUID, addresses []cmaddress.Address) error
	AgentUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, ringMethod agent.RingMethod) error
	AgentUpdatePassword(ctx context.Context, id uuid.UUID, username, password string) error
	AgentUpdatePermission(ctx context.Context, id uuid.UUID, permission agent.Permission) error
	AgentUpdateStatus(ctx context.Context, id uuid.UUID, status agent.Status) error
	AgentUpdateTagIDs(ctx context.Context, id uuid.UUID, tags []uuid.UUID) error

	AgentCallAnswered(ctx context.Context, c *cmcall.Call) error
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
