package taghandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package taghandler -destination ./mock_taghandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/agenthandler"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/dbhandler"
)

// List of default values
const (
	defaultTimeStamp = "9999-01-01 00:00:00.000000" // default timestamp
)

// TagHandler interfaces
type TagHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, name, detail string) (*tag.Tag, error)
	Delete(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
	Get(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
	Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*tag.Tag, error)
	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*tag.Tag, error)
}

type tagHandler struct {
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyhandler notifyhandler.NotifyHandler

	agentHandler agenthandler.AgentHandler
}

// NewTagHandler return TagHandler interface
func NewTagHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler, agentHandler agenthandler.AgentHandler) TagHandler {
	return &tagHandler{
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyhandler: notifyHandler,

		agentHandler: agentHandler,
	}
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
