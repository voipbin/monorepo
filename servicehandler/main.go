package servicehandler

//go:generate mockgen -destination ./mock_servicehandler_servicehandler.go -package servicehandler -source ./main.go ServiceHandler

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager/models/call"
	"gitlab.com/voipbin/bin-manager/api-manager/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler"
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {
	AuthLogin(username, password string) (string, error)

	// call handlers
	CallCreate(u *user.User, flowID uuid.UUID, source, destination string) (*call.Call, error)

	ConferenceCreate(u *user.User, confType conference.Type, name, detail string) (*conference.Conference, error)
	ConferenceDelete(u *user.User, confID uuid.UUID) error

	UserCreate(username, password string) (*user.User, error)
}

type servicHandler struct {
	reqHandler requesthandler.RequestHandler
	dbHandler  dbhandler.DBHandler
}

// NewServiceHandler return ServiceHandler interface
func NewServiceHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler) ServiceHandler {
	return &servicHandler{
		reqHandler: reqHandler,
		dbHandler:  dbHandler,
	}
}

// ReqHandler for service
var ReqHandler requesthandler.RequestHandler

// Setup initiates service
func Setup(sock rabbitmq.Rabbit, exchangeDelay, queueCall, queueFlow string) error {
	ReqHandler = requesthandler.NewRequestHandler(sock, exchangeDelay, queueCall, queueFlow)
	return nil
}
