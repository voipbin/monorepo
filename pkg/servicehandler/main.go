package servicehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package servicehandler -destination ./mock_servicehandler.go -source main.go -build_flags=-mod=mod

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/availablenumber"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/tag"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

const (
	defaultTimestamp string = "9999-01-01 00:00:00.000000" // default timestamp
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {

	// agent handlers
	AgentCreate(
		u *cscustomer.Customer,
		username string,
		password string,
		name string,
		detail string,
		webhookMethod string,
		webhookURI string,
		ringMethod amagent.RingMethod,
		permission uint64,
		tagIDs []uuid.UUID,
		addresses []cmaddress.Address,
	) (*amagent.WebhookMessage, error)
	AgentGet(u *cscustomer.Customer, agentID uuid.UUID) (*amagent.WebhookMessage, error)
	AgentGets(u *cscustomer.Customer, size uint64, token string, tagIDs []uuid.UUID, status amagent.Status) ([]*amagent.WebhookMessage, error)
	AgentDelete(u *cscustomer.Customer, agentID uuid.UUID) error
	AgentLogin(customerID uuid.UUID, username, password string) (string, error)
	AgentUpdate(u *cscustomer.Customer, agentID uuid.UUID, name, detail string, ringMethod amagent.RingMethod) error
	AgentUpdateAddresses(u *cscustomer.Customer, agentID uuid.UUID, addresses []cmaddress.Address) error
	AgentUpdateStatus(u *cscustomer.Customer, agentID uuid.UUID, status amagent.Status) error
	AgentUpdateTagIDs(u *cscustomer.Customer, agentID uuid.UUID, tagIDs []uuid.UUID) error

	// auth handlers
	AuthLogin(username, password string) (string, error)

	// available numbers
	AvailableNumberGets(u *cscustomer.Customer, size uint64, countryCode string) ([]*availablenumber.AvailableNumber, error)

	// call handlers
	CallCreate(u *cscustomer.Customer, flowID uuid.UUID, source, destination *cmaddress.Address) (*cmcall.WebhookMessage, error)
	CallGet(u *cscustomer.Customer, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallGets(u *cscustomer.Customer, size uint64, token string) ([]*cmcall.WebhookMessage, error)
	CallDelete(u *cscustomer.Customer, callID uuid.UUID) error

	// conference handlers
	ConferenceCreate(u *cscustomer.Customer, confType cfconference.Type, name, detail, webhookURI string, preActions, postActions []fmaction.Action) (*cfconference.WebhookMessage, error)
	ConferenceDelete(u *cscustomer.Customer, confID uuid.UUID) error
	ConferenceGet(u *cscustomer.Customer, id uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceGets(u *cscustomer.Customer, size uint64, token string) ([]*cfconference.WebhookMessage, error)
	ConferenceKick(u *cscustomer.Customer, confID uuid.UUID, callID uuid.UUID) error

	// customer handlers
	CustomerCreate(u *cscustomer.Customer, username, password, name, detail string, webhookMethod cscustomer.WebhookMethod, webhookURI string, permissionIDs []uuid.UUID) (*cscustomer.WebhookMessage, error)
	CustomerGet(u *cscustomer.Customer, customerID uuid.UUID) (*cscustomer.WebhookMessage, error)
	CustomerGets(u *cscustomer.Customer, size uint64, token string) ([]*cscustomer.WebhookMessage, error)
	CustomerUpdate(u *cscustomer.Customer, id uuid.UUID, name, detail string, webhookMethod cscustomer.WebhookMethod, webhookURI string) (*cscustomer.WebhookMessage, error)
	CustomerDelete(u *cscustomer.Customer, customerID uuid.UUID) (*cscustomer.WebhookMessage, error)
	CustomerUpdatePassword(u *cscustomer.Customer, customerID uuid.UUID, password string) (*cscustomer.WebhookMessage, error)
	CustomerUpdatePermissionIDs(u *cscustomer.Customer, customerID uuid.UUID, permissionIDs []uuid.UUID) (*cscustomer.WebhookMessage, error)

	// domain handlers
	DomainCreate(u *cscustomer.Customer, domainName, name, detail string) (*domain.Domain, error)
	DomainDelete(u *cscustomer.Customer, id uuid.UUID) error
	DomainGet(u *cscustomer.Customer, id uuid.UUID) (*domain.Domain, error)
	DomainGets(u *cscustomer.Customer, size uint64, token string) ([]*domain.Domain, error)
	DomainUpdate(u *cscustomer.Customer, d *domain.Domain) (*domain.Domain, error)

	// extension handlers
	ExtensionCreate(u *cscustomer.Customer, e *extension.Extension) (*extension.Extension, error)
	ExtensionDelete(u *cscustomer.Customer, id uuid.UUID) error
	ExtensionGet(u *cscustomer.Customer, id uuid.UUID) (*extension.Extension, error)
	ExtensionGets(u *cscustomer.Customer, domainID uuid.UUID, size uint64, token string) ([]*extension.Extension, error)
	ExtensionUpdate(u *cscustomer.Customer, d *extension.Extension) (*extension.Extension, error)

	// flow handlers
	FlowCreate(u *cscustomer.Customer, name, detail, webhookURI string, actions []fmaction.Action, persist bool) (*fmflow.WebhookMessage, error)
	FlowDelete(u *cscustomer.Customer, id uuid.UUID) error
	FlowGet(u *cscustomer.Customer, id uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowGets(u *cscustomer.Customer, pageSize uint64, pageToken string) ([]*fmflow.WebhookMessage, error)
	FlowUpdate(u *cscustomer.Customer, f *fmflow.Flow) (*fmflow.WebhookMessage, error)

	// order numbers handler
	NumberCreate(u *cscustomer.Customer, num string) (*number.Number, error)
	NumberGet(u *cscustomer.Customer, id uuid.UUID) (*number.Number, error)
	NumberGets(u *cscustomer.Customer, size uint64, token string) ([]*number.Number, error)
	NumberDelete(u *cscustomer.Customer, id uuid.UUID) (*number.Number, error)
	NumberUpdate(u *cscustomer.Customer, numb *number.Number) (*number.Number, error)

	// queue handler
	QueueGet(u *cscustomer.Customer, queueID uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueGets(u *cscustomer.Customer, size uint64, token string) ([]*qmqueue.WebhookMessage, error)
	QueueCreate(
		u *cscustomer.Customer,
		name string,
		detail string,
		webhookURI string,
		webhookMethod string,
		routingMethod string,
		tagIDs []uuid.UUID,
		waitActions []fmaction.Action,
		timeoutWait int,
		timeoutService int,
	) (*qmqueue.WebhookMessage, error)
	QueueDelete(u *cscustomer.Customer, queueID uuid.UUID) error
	QueueUpdate(u *cscustomer.Customer, queueID uuid.UUID, name, detail, webhookURI, webhookMethod string) error
	QueueUpdateTagIDs(u *cscustomer.Customer, queueID uuid.UUID, tagIDs []uuid.UUID) error
	QueueUpdateRoutingMethod(u *cscustomer.Customer, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) error
	QueueUpdateActions(u *cscustomer.Customer, queueID uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) error

	// recording handlers
	RecordingGet(u *cscustomer.Customer, id uuid.UUID) (*recording.Recording, error)
	RecordingGets(u *cscustomer.Customer, size uint64, token string) ([]*recording.Recording, error)

	// recordingfile handlers
	RecordingfileGet(u *cscustomer.Customer, id uuid.UUID) (string, error)

	TagCreate(u *cscustomer.Customer, name string, detail string) (*tag.Tag, error)
	TagDelete(u *cscustomer.Customer, id uuid.UUID) error
	TagGet(u *cscustomer.Customer, id uuid.UUID) (*tag.Tag, error)
	TagGets(u *cscustomer.Customer, size uint64, token string) ([]*tag.Tag, error)
	TagUpdate(u *cscustomer.Customer, id uuid.UUID, name, detail string) error

	// transcribe handlers
	TranscribeCreate(u *cscustomer.Customer, referencdID uuid.UUID, language string) (*transcribe.Transcribe, error)

	// // user handlers
	// UserCreate(u *user.User, username, password, name, detail string, permission user.Permission) (*user.User, error)
	// UserDelete(u *user.User, id uint64) error
	// UserGet(u *user.User, userID uint64) (*user.User, error)
	// UserGets(u *user.User, size uint64, token string) ([]*user.User, error)
	// UserUpdate(u *user.User, id uint64, name, detail string) error
	// UserUpdatePassword(u *user.User, id uint64, password string) error
	// UserUpdatePermission(u *user.User, id uint64, permission user.Permission) error
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

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func Find(slice []uuid.UUID, val uuid.UUID) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}
