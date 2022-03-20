package servicehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package servicehandler -destination ./mock_servicehandler.go -source main.go -build_flags=-mod=mod

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	amtag "gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/availablenumber"
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
		ringMethod amagent.RingMethod,
		permission amagent.Permission,
		tagIDs []uuid.UUID,
		addresses []cmaddress.Address,
	) (*amagent.WebhookMessage, error)
	AgentGet(u *cscustomer.Customer, agentID uuid.UUID) (*amagent.WebhookMessage, error)
	AgentGets(u *cscustomer.Customer, size uint64, token string, tagIDs []uuid.UUID, status amagent.Status) ([]*amagent.WebhookMessage, error)
	AgentDelete(u *cscustomer.Customer, agentID uuid.UUID) (*amagent.WebhookMessage, error)
	AgentLogin(customerID uuid.UUID, username, password string) (string, error)
	AgentUpdate(u *cscustomer.Customer, agentID uuid.UUID, name, detail string, ringMethod amagent.RingMethod) (*amagent.WebhookMessage, error)
	AgentUpdateAddresses(u *cscustomer.Customer, agentID uuid.UUID, addresses []cmaddress.Address) (*amagent.WebhookMessage, error)
	AgentUpdateStatus(u *cscustomer.Customer, agentID uuid.UUID, status amagent.Status) (*amagent.WebhookMessage, error)
	AgentUpdateTagIDs(u *cscustomer.Customer, agentID uuid.UUID, tagIDs []uuid.UUID) (*amagent.WebhookMessage, error)

	// auth handlers
	AuthLogin(username, password string) (string, error)

	// available numbers
	AvailableNumberGets(u *cscustomer.Customer, size uint64, countryCode string) ([]*availablenumber.AvailableNumber, error)

	// call handlers
	CallCreate(u *cscustomer.Customer, flowID uuid.UUID, actions []fmaction.Action, source *cmaddress.Address, destinations []cmaddress.Address) ([]*cmcall.WebhookMessage, error)
	CallGet(u *cscustomer.Customer, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallGets(u *cscustomer.Customer, size uint64, token string) ([]*cmcall.WebhookMessage, error)
	CallDelete(u *cscustomer.Customer, callID uuid.UUID) error

	// conference handlers
	ConferenceCreate(u *cscustomer.Customer, confType cfconference.Type, name, detail string, preActions, postActions []fmaction.Action) (*cfconference.WebhookMessage, error)
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
	DomainCreate(u *cscustomer.Customer, domainName, name, detail string) (*rmdomain.WebhookMessage, error)
	DomainDelete(u *cscustomer.Customer, id uuid.UUID) (*rmdomain.WebhookMessage, error)
	DomainGet(u *cscustomer.Customer, id uuid.UUID) (*rmdomain.WebhookMessage, error)
	DomainGets(u *cscustomer.Customer, size uint64, token string) ([]*rmdomain.WebhookMessage, error)
	DomainUpdate(u *cscustomer.Customer, id uuid.UUID, name, detail string) (*rmdomain.WebhookMessage, error)

	// extension handlers
	ExtensionCreate(u *cscustomer.Customer, ext, password string, domainID uuid.UUID, name, detail string) (*rmextension.WebhookMessage, error)
	ExtensionDelete(u *cscustomer.Customer, id uuid.UUID) (*rmextension.WebhookMessage, error)
	ExtensionGet(u *cscustomer.Customer, id uuid.UUID) (*rmextension.WebhookMessage, error)
	ExtensionGets(u *cscustomer.Customer, domainID uuid.UUID, size uint64, token string) ([]*rmextension.WebhookMessage, error)
	ExtensionUpdate(u *cscustomer.Customer, id uuid.UUID, name, detail, password string) (*rmextension.WebhookMessage, error)

	// flow handlers
	FlowCreate(u *cscustomer.Customer, name, detail string, actions []fmaction.Action, persist bool) (*fmflow.WebhookMessage, error)
	FlowDelete(u *cscustomer.Customer, id uuid.UUID) error
	FlowGet(u *cscustomer.Customer, id uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowGets(u *cscustomer.Customer, pageSize uint64, pageToken string) ([]*fmflow.WebhookMessage, error)
	FlowUpdate(u *cscustomer.Customer, f *fmflow.Flow) (*fmflow.WebhookMessage, error)

	// message handlers
	MessageDelete(u *cscustomer.Customer, id uuid.UUID) (*mmmessage.WebhookMessage, error)
	MessageGets(u *cscustomer.Customer, size uint64, token string) ([]*mmmessage.WebhookMessage, error)
	MessageGet(u *cscustomer.Customer, id uuid.UUID) (*mmmessage.WebhookMessage, error)
	MessageSend(u *cscustomer.Customer, source *cmaddress.Address, destinations []cmaddress.Address, text string) (*mmmessage.WebhookMessage, error)

	// order numbers handler
	NumberCreate(u *cscustomer.Customer, num string, callFlowID, messageFlowID uuid.UUID, name, detail string) (*nmnumber.WebhookMessage, error)
	NumberGet(u *cscustomer.Customer, id uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberGets(u *cscustomer.Customer, size uint64, token string) ([]*nmnumber.WebhookMessage, error)
	NumberDelete(u *cscustomer.Customer, id uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberUpdate(u *cscustomer.Customer, id uuid.UUID, name, detail string) (*nmnumber.WebhookMessage, error)
	NumberUpdateFlowIDs(u *cscustomer.Customer, id, callFlowID, messageFlowID uuid.UUID) (*nmnumber.WebhookMessage, error)

	// queue handler
	QueueGet(u *cscustomer.Customer, queueID uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueGets(u *cscustomer.Customer, size uint64, token string) ([]*qmqueue.WebhookMessage, error)
	QueueCreate(
		u *cscustomer.Customer,
		name string,
		detail string,
		routingMethod string,
		tagIDs []uuid.UUID,
		waitActions []fmaction.Action,
		timeoutWait int,
		timeoutService int,
	) (*qmqueue.WebhookMessage, error)
	QueueDelete(u *cscustomer.Customer, queueID uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueUpdate(u *cscustomer.Customer, queueID uuid.UUID, name, detail string) (*qmqueue.WebhookMessage, error)
	QueueUpdateTagIDs(u *cscustomer.Customer, queueID uuid.UUID, tagIDs []uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueUpdateRoutingMethod(u *cscustomer.Customer, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) (*qmqueue.WebhookMessage, error)
	QueueUpdateActions(u *cscustomer.Customer, queueID uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) (*qmqueue.WebhookMessage, error)

	// recording handlers
	RecordingGet(u *cscustomer.Customer, id uuid.UUID) (*cmrecording.WebhookMessage, error)
	RecordingGets(u *cscustomer.Customer, size uint64, token string) ([]*cmrecording.WebhookMessage, error)

	// recordingfile handlers
	RecordingfileGet(u *cscustomer.Customer, id uuid.UUID) (string, error)

	TagCreate(u *cscustomer.Customer, name string, detail string) (*amtag.WebhookMessage, error)
	TagDelete(u *cscustomer.Customer, id uuid.UUID) (*amtag.WebhookMessage, error)
	TagGet(u *cscustomer.Customer, id uuid.UUID) (*amtag.WebhookMessage, error)
	TagGets(u *cscustomer.Customer, size uint64, token string) ([]*amtag.WebhookMessage, error)
	TagUpdate(u *cscustomer.Customer, id uuid.UUID, name, detail string) (*amtag.WebhookMessage, error)

	// transcribe handlers
	TranscribeCreate(u *cscustomer.Customer, referencdID uuid.UUID, language string) (*tmtranscribe.WebhookMessage, error)

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
