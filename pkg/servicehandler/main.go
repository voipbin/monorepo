package servicehandler

//go:generate mockgen -destination ./mock_servicehandler_servicehandler.go -package servicehandler -source ./main.go ServiceHandler

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler"
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {
	// auth handlers
	AuthLogin(username, password string) (string, error)

	// available numbers
	AvailableNumberGets(u *models.User, size uint64, countryCode string) ([]*models.AvailableNumber, error)

	// call handlers
	CallCreate(u *models.User, flowID uuid.UUID, source, destination models.CallAddress) (*models.Call, error)
	CallGet(u *models.User, callID uuid.UUID) (*models.Call, error)
	CallGets(u *models.User, size uint64, token string) ([]*models.Call, error)
	CallDelete(u *models.User, callID uuid.UUID) error

	// conference handlers
	ConferenceCreate(u *models.User, confType models.ConferenceType, name, detail string) (*models.Conference, error)
	ConferenceDelete(u *models.User, confID uuid.UUID) error
	ConferenceGet(u *models.User, id uuid.UUID) (*models.Conference, error)
	ConferenceGets(u *models.User, size uint64, token string) ([]*models.Conference, error)

	// domain handlers
	DomainCreate(u *models.User, domainName, name, detail string) (*models.Domain, error)
	DomainDelete(u *models.User, id uuid.UUID) error
	DomainGet(u *models.User, id uuid.UUID) (*models.Domain, error)
	DomainGets(u *models.User, size uint64, token string) ([]*models.Domain, error)
	DomainUpdate(u *models.User, d *models.Domain) (*models.Domain, error)

	// extension handlers
	ExtensionCreate(u *models.User, e *models.Extension) (*models.Extension, error)
	ExtensionDelete(u *models.User, id uuid.UUID) error
	ExtensionGet(u *models.User, id uuid.UUID) (*models.Extension, error)
	ExtensionGets(u *models.User, domainID uuid.UUID, size uint64, token string) ([]*models.Extension, error)
	ExtensionUpdate(u *models.User, d *models.Extension) (*models.Extension, error)

	// flow handlers
	FlowCreate(u *models.User, id uuid.UUID, name, detail string, actions []models.Action, persist bool) (*models.Flow, error)
	FlowDelete(u *models.User, id uuid.UUID) error
	FlowGet(u *models.User, id uuid.UUID) (*models.Flow, error)
	FlowGets(u *models.User, pageSize uint64, pageToken string) ([]*models.Flow, error)
	FlowUpdate(u *models.User, f *models.Flow) (*models.Flow, error)

	// order numbers handler
	OrderNumberCreate(u *models.User, num string) (*models.Number, error)
	OrderNumberGets(u *models.User, size uint64, token string) ([]*models.Number, error)

	// recording handlers
	RecordingGet(u *models.User, id uuid.UUID) (*models.Recording, error)
	RecordingGets(u *models.User, size uint64, token string) ([]*models.Recording, error)

	// recordingfile handlers
	RecordingfileGet(u *models.User, id uuid.UUID) (string, error)

	// user handlers
	UserCreate(username, password string, permission uint64) (*models.User, error)
	UserGet(userID uint64) (*models.User, error)
	UserGets() ([]*models.User, error)
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

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
