package servicehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package servicehandler -destination ./mock_servicehandler_servicehandler.go -source main.go -build_flags=-mod=mod

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/availablenumber"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/tag"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

const (
	defaultTimestamp string = "9999-01-01 00:00:00.000000" // default timestamp
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {

	// agent handlers
	AgentCreate(
		u *user.User,
		username string,
		password string,
		name string,
		detail string,
		ringMethod string,
		permission uint64,
		tagIDs []uuid.UUID,
		addresses []address.Address,
	) (*agent.Agent, error)
	AgentGet(u *user.User, agentID uuid.UUID) (*agent.Agent, error)
	AgentGets(u *user.User, size uint64, token string, tagIDs []uuid.UUID, status agent.Status) ([]*agent.Agent, error)
	AgentDelete(u *user.User, agentID uuid.UUID) error
	AgentLogin(userID uint64, username, password string) (string, error)
	AgentUpdate(u *user.User, agentID uuid.UUID, name, detail string, ringMethod agent.RingMethod) error
	AgentUpdateAddresses(u *user.User, agentID uuid.UUID, addresses []address.Address) error
	AgentUpdateStatus(u *user.User, agentID uuid.UUID, status agent.Status) error
	AgentUpdateTagIDs(u *user.User, agentID uuid.UUID, tagIDs []uuid.UUID) error

	// auth handlers
	AuthLogin(username, password string) (string, error)

	// available numbers
	AvailableNumberGets(u *user.User, size uint64, countryCode string) ([]*availablenumber.AvailableNumber, error)

	// call handlers
	CallCreate(u *user.User, flowID uuid.UUID, source, destination *address.Address) (*call.Call, error)
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

	// queue handler
	QueueGet(u *user.User, queueID uuid.UUID) (*qmqueue.Event, error)
	QueueGets(u *user.User, size uint64, token string) ([]*qmqueue.Event, error)
	QueueCreate(
		u *user.User,
		name string,
		detail string,
		webhookURI string,
		webhookMethod string,
		routingMethod string,
		tagIDs []uuid.UUID,
		waitActions []fmaction.Action,
		timeoutWait int,
		timeoutService int,
	) (*qmqueue.Event, error)
	QueueDelete(u *user.User, queueID uuid.UUID) error
	QueueUpdate(u *user.User, queueID uuid.UUID, name, detail, webhookURI, webhookMethod string) error
	QueueUpdateTagIDs(u *user.User, queueID uuid.UUID, tagIDs []uuid.UUID) error
	QueueUpdateRoutingMethod(u *user.User, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) error
	QueueUpdateActions(u *user.User, queueID uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) error

	// recording handlers
	RecordingGet(u *user.User, id uuid.UUID) (*recording.Recording, error)
	RecordingGets(u *user.User, size uint64, token string) ([]*recording.Recording, error)

	// recordingfile handlers
	RecordingfileGet(u *user.User, id uuid.UUID) (string, error)

	TagCreate(u *user.User, name string, detail string) (*tag.Tag, error)
	TagDelete(u *user.User, id uuid.UUID) error
	TagGet(u *user.User, id uuid.UUID) (*tag.Tag, error)
	TagGets(u *user.User, size uint64, token string) ([]*tag.Tag, error)
	TagUpdate(u *user.User, id uuid.UUID, name, detail string) error

	// transcribe handlers
	TranscribeCreate(u *user.User, referencdID uuid.UUID, language string) (*transcribe.Transcribe, error)

	// user handlers
	UserCreate(u *user.User, username, password, name, detail string, permission user.Permission) (*user.User, error)
	UserDelete(u *user.User, id uint64) error
	UserGet(u *user.User, userID uint64) (*user.User, error)
	UserGets(u *user.User, size uint64, token string) ([]*user.User, error)
	UserUpdate(u *user.User, id uint64, name, detail string) error
	UserUpdatePassword(u *user.User, id uint64, password string) error
	UserUpdatePermission(u *user.User, id uint64, permission user.Permission) error
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
