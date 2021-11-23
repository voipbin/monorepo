package servicehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package servicehandler -destination ./mock_servicehandler_servicehandler.go -source main.go -build_flags=-mod=mod

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/availablenumber"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

const (
	defaultTimestamp string = "9999-01-01 00:00:00.000000" // default timestamp
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {
	// auth handlers
	AuthLogin(username, password string) (string, error)

	// available numbers
	AvailableNumberGets(u *user.User, size uint64, countryCode string) ([]*availablenumber.AvailableNumber, error)

	// call handlers
	CallCreate(u *user.User, flowID uuid.UUID, source, destination *call.Address) (*call.Call, error)
	CallGet(u *user.User, callID uuid.UUID) (*call.Call, error)
	CallGets(u *user.User, size uint64, token string) ([]*call.Call, error)
	CallDelete(u *user.User, callID uuid.UUID) error

	// conference handlers
	ConferenceCreate(u *user.User, confType conference.Type, name, detail, webhookURI string, preActions, postActions []action.Action) (*conference.Conference, error)
	ConferenceDelete(u *user.User, confID uuid.UUID) error
	ConferenceGet(u *user.User, id uuid.UUID) (*conference.Conference, error)
	ConferenceGets(u *user.User, size uint64, token string) ([]*conference.Conference, error)
	ConferenceKick(u *user.User, confID uuid.UUID, callID uuid.UUID) error

	// domain handlers
	DomainCreate(u *user.User, domainName, name, detail string) (*domain.Domain, error)
	DomainDelete(u *user.User, id uuid.UUID) error
	DomainGet(u *user.User, id uuid.UUID) (*domain.Domain, error)
	DomainGets(u *user.User, size uint64, token string) ([]*domain.Domain, error)
	DomainUpdate(u *user.User, d *domain.Domain) (*domain.Domain, error)

	// extension handlers
	ExtensionCreate(u *user.User, e *extension.Extension) (*extension.Extension, error)
	ExtensionDelete(u *user.User, id uuid.UUID) error
	ExtensionGet(u *user.User, id uuid.UUID) (*extension.Extension, error)
	ExtensionGets(u *user.User, domainID uuid.UUID, size uint64, token string) ([]*extension.Extension, error)
	ExtensionUpdate(u *user.User, d *extension.Extension) (*extension.Extension, error)

	// flow handlers
	FlowCreate(u *user.User, name, detail, webhookURI string, actions []action.Action, persist bool) (*flow.Flow, error)
	FlowDelete(u *user.User, id uuid.UUID) error
	FlowGet(u *user.User, id uuid.UUID) (*flow.Flow, error)
	FlowGets(u *user.User, pageSize uint64, pageToken string) ([]*flow.Flow, error)
	FlowUpdate(u *user.User, f *flow.Flow) (*flow.Flow, error)

	// order numbers handler
	NumberCreate(u *user.User, num string) (*number.Number, error)
	NumberGet(u *user.User, id uuid.UUID) (*number.Number, error)
	NumberGets(u *user.User, size uint64, token string) ([]*number.Number, error)
	NumberDelete(u *user.User, id uuid.UUID) (*number.Number, error)
	NumberUpdate(u *user.User, numb *number.Number) (*number.Number, error)

	// recording handlers
	RecordingGet(u *user.User, id uuid.UUID) (*recording.Recording, error)
	RecordingGets(u *user.User, size uint64, token string) ([]*recording.Recording, error)

	// recordingfile handlers
	RecordingfileGet(u *user.User, id uuid.UUID) (string, error)

	// transcribe handlers
	TranscribeCreate(u *user.User, referencdID uuid.UUID, language string) (*transcribe.Transcribe, error)

	// user handlers
	UserCreate(username, password string, permission uint64) (*user.User, error)
	UserGet(userID uint64) (*user.User, error)
	UserGets(size uint64, token string) ([]*user.User, error)
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
