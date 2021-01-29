package servicehandler

//go:generate mockgen -destination ./mock_servicehandler_servicehandler.go -package servicehandler -source ./main.go ServiceHandler

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {
	// auth handlers
	AuthLogin(username, password string) (string, error)

	// call handlers
	CallCreate(u *user.User, flowID uuid.UUID, source, destination call.Address) (*call.Call, error)
	CallGet(u *user.User, callID uuid.UUID) (*call.Call, error)
	CallGets(u *user.User, size uint64, token string) ([]*call.Call, error)
	CallDelete(u *user.User, callID uuid.UUID) error

	// conference handlers
	ConferenceCreate(u *user.User, confType conference.Type, name, detail string) (*conference.Conference, error)
	ConferenceDelete(u *user.User, confID uuid.UUID) error
	ConferenceGet(u *user.User, id uuid.UUID) (*conference.Conference, error)
	ConferenceGets(u *user.User, size uint64, token string) ([]*conference.Conference, error)

	// flow handlers
	FlowCreate(u *user.User, id uuid.UUID, name, detail string, actions []action.Action, persist bool) (*flow.Flow, error)
	FlowGet(u *user.User, id uuid.UUID) (*flow.Flow, error)
	FlowGetsByUserID(u *user.User, pageToken string, pageSize uint64) ([]*flow.Flow, error)

	// recording handlers
	RecordingGet(u *user.User, id uuid.UUID) (*recording.Recording, error)
	RecordingGets(u *user.User, size uint64, token string) ([]*recording.Recording, error)

	// recordingfile handlers
	RecordingfileGet(u *user.User, id uuid.UUID) (string, error)

	// user handlers
	UserCreate(username, password string, permission uint64) (*user.User, error)
	UserGet(userID uint64) (*user.User, error)
	UserGets() ([]*user.User, error)
}

type serviceHandler struct {
	reqHandler requesthandler.RequestHandler
	dbHandler  dbhandler.DBHandler
}

// NewServiceHandler return ServiceHandler interface
func NewServiceHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler) ServiceHandler {
	return &serviceHandler{
		reqHandler: reqHandler,
		dbHandler:  dbHandler,
	}
}

// ReqHandler for service
var ReqHandler requesthandler.RequestHandler

// Setup initiates service
func Setup(sock rabbitmqhandler.Rabbit, exchangeDelay, queueCall, queueFlow, queueStorage string) error {
	ReqHandler = requesthandler.NewRequestHandler(sock, exchangeDelay, queueCall, queueFlow, queueStorage)
	return nil
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
