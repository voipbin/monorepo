package servicehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package servicehandler -destination ./mock_servicehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	amtag "gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	cacampaigncall "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	caoutplan "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
	chatchat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
	chatchatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	chatmedia "gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	chatmessagechat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	chatmessagechatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	cfconferencecall "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	cvmedia "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	cvmessage "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	nmavailablenumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	omoutdial "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdial"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/websockhandler"
)

const (
	defaultTimestamp string = "9999-01-01 00:00:00.000000" // default timestamp
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {

	// agent handlers
	AgentCreate(
		ctx context.Context,
		u *cscustomer.Customer,
		username string,
		password string,
		name string,
		detail string,
		ringMethod amagent.RingMethod,
		permission amagent.Permission,
		tagIDs []uuid.UUID,
		addresses []commonaddress.Address,
	) (*amagent.WebhookMessage, error)
	AgentGet(ctx context.Context, u *cscustomer.Customer, agentID uuid.UUID) (*amagent.WebhookMessage, error)
	AgentGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string, tagIDs []uuid.UUID, status amagent.Status) ([]*amagent.WebhookMessage, error)
	AgentDelete(ctx context.Context, u *cscustomer.Customer, agentID uuid.UUID) (*amagent.WebhookMessage, error)
	AgentLogin(ctx context.Context, customerID uuid.UUID, username, password string) (string, error)
	AgentUpdate(ctx context.Context, u *cscustomer.Customer, agentID uuid.UUID, name, detail string, ringMethod amagent.RingMethod) (*amagent.WebhookMessage, error)
	AgentUpdateAddresses(ctx context.Context, u *cscustomer.Customer, agentID uuid.UUID, addresses []commonaddress.Address) (*amagent.WebhookMessage, error)
	AgentUpdateStatus(ctx context.Context, u *cscustomer.Customer, agentID uuid.UUID, status amagent.Status) (*amagent.WebhookMessage, error)
	AgentUpdateTagIDs(ctx context.Context, u *cscustomer.Customer, agentID uuid.UUID, tagIDs []uuid.UUID) (*amagent.WebhookMessage, error)

	// auth handlers
	AuthLogin(username, password string) (string, error)

	// available numbers
	AvailableNumberGets(u *cscustomer.Customer, size uint64, countryCode string) ([]*nmavailablenumber.WebhookMessage, error)

	// call handlers
	CallCreate(u *cscustomer.Customer, flowID uuid.UUID, actions []fmaction.Action, source *commonaddress.Address, destinations []commonaddress.Address) ([]*cmcall.WebhookMessage, error)
	CallGet(u *cscustomer.Customer, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallGets(u *cscustomer.Customer, size uint64, token string) ([]*cmcall.WebhookMessage, error)
	CallDelete(u *cscustomer.Customer, callID uuid.UUID) error

	// campaign handlers
	CampaignCreate(
		u *cscustomer.Customer,
		name string,
		detail string,
		campaignType cacampaign.Type,
		serviceLevel int,
		endHandle cacampaign.EndHandle,
		actions []fmaction.Action,
		outplanID uuid.UUID,
		outdialID uuid.UUID,
		queueID uuid.UUID,
		nextCampaignID uuid.UUID,
	) (*cacampaign.WebhookMessage, error)
	CampaignGetsByCustomerID(u *cscustomer.Customer, size uint64, token string) ([]*cacampaign.WebhookMessage, error)
	CampaignGet(u *cscustomer.Customer, id uuid.UUID) (*cacampaign.WebhookMessage, error)
	CampaignDelete(u *cscustomer.Customer, id uuid.UUID) (*cacampaign.WebhookMessage, error)
	CampaignUpdateBasicInfo(u *cscustomer.Customer, id uuid.UUID, name, detail string) (*cacampaign.WebhookMessage, error)
	CampaignUpdateStatus(u *cscustomer.Customer, id uuid.UUID, status cacampaign.Status) (*cacampaign.WebhookMessage, error)
	CampaignUpdateServiceLevel(u *cscustomer.Customer, id uuid.UUID, serviceLevel int) (*cacampaign.WebhookMessage, error)
	CampaignUpdateActions(u *cscustomer.Customer, id uuid.UUID, actions []fmaction.Action) (*cacampaign.WebhookMessage, error)
	CampaignUpdateResourceInfo(u *cscustomer.Customer, id uuid.UUID, outplanID uuid.UUID, outdialID uuid.UUID, queueID uuid.UUID) (*cacampaign.WebhookMessage, error)
	CampaignUpdateNextCampaignID(u *cscustomer.Customer, id uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.WebhookMessage, error)

	// campaigncall handlers
	CampaigncallGetsByCampaignID(u *cscustomer.Customer, campaignID uuid.UUID, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error)
	CampaigncallGet(u *cscustomer.Customer, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error)
	CampaigncallDelete(u *cscustomer.Customer, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error)

	// chat handlers
	ChatCreate(
		ctx context.Context,
		u *cscustomer.Customer,
		chatType chatchat.Type,
		ownerID uuid.UUID,
		participantIDs []uuid.UUID,
		name string,
		detail string,
	) (*chatchat.WebhookMessage, error)
	ChatGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*chatchat.WebhookMessage, error)
	ChatGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatchat.WebhookMessage, error)
	ChatDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatchat.WebhookMessage, error)
	ChatUpdateBasicInfo(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string) (*chatchat.WebhookMessage, error)
	ChatUpdateOwnerID(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, ownerID uuid.UUID) (*chatchat.WebhookMessage, error)
	ChatAddParticipantID(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, participantID uuid.UUID) (*chatchat.WebhookMessage, error)
	ChatRemoveParticipantID(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, participantID uuid.UUID) (*chatchat.WebhookMessage, error)

	// chatmessage handlers
	ChatmessageCreate(
		ctx context.Context,
		u *cscustomer.Customer,
		chatID uuid.UUID,
		source commonaddress.Address,
		messageType chatmessagechat.Type,
		text string,
		medias []chatmedia.Media,
	) (*chatmessagechat.WebhookMessage, error)
	ChatmessageGetsByChatID(ctx context.Context, u *cscustomer.Customer, chatID uuid.UUID, size uint64, token string) ([]*chatmessagechat.WebhookMessage, error)
	ChatmessageGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatmessagechat.WebhookMessage, error)
	ChatmessageDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatmessagechat.WebhookMessage, error)

	// chatroom handlers
	ChatroomGetsByOwnerID(ctx context.Context, u *cscustomer.Customer, ownerID uuid.UUID, size uint64, token string) ([]*chatchatroom.WebhookMessage, error)
	ChatroomGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatchatroom.WebhookMessage, error)
	ChatroomDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatchatroom.WebhookMessage, error)

	// chatroommessage handlers
	ChatroommessageGetsByChatroomID(ctx context.Context, u *cscustomer.Customer, chatroomID uuid.UUID, size uint64, token string) ([]*chatmessagechatroom.WebhookMessage, error)
	ChatroommessageGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatmessagechatroom.WebhookMessage, error)
	ChatroommessageDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatmessagechatroom.WebhookMessage, error)

	// conference handlers
	ConferenceCreate(u *cscustomer.Customer, confType cfconference.Type, name, detail string, preActions, postActions []fmaction.Action) (*cfconference.WebhookMessage, error)
	ConferenceDelete(u *cscustomer.Customer, confID uuid.UUID) error
	ConferenceGet(u *cscustomer.Customer, id uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceGets(u *cscustomer.Customer, size uint64, token string) ([]*cfconference.WebhookMessage, error)

	// conferencecall handlers
	ConferencecallGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cfconferencecall.WebhookMessage, error)
	ConferencecallCreate(ctx context.Context, u *cscustomer.Customer, conferenceID uuid.UUID, referenceType cfconferencecall.ReferenceType, referenceID uuid.UUID) (*cfconferencecall.WebhookMessage, error)
	ConferencecallKick(ctx context.Context, u *cscustomer.Customer, conferencecallID uuid.UUID) (*cfconferencecall.WebhookMessage, error)

	// conversation handlers
	ConversationGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cvconversation.WebhookMessage, error)
	ConversationGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cvconversation.WebhookMessage, error)
	ConversationMessageGetsByConversationID(
		ctx context.Context,
		u *cscustomer.Customer,
		conversationID uuid.UUID,
		size uint64,
		token string,
	) ([]*cvmessage.WebhookMessage, error)
	ConversationMessageSend(
		ctx context.Context,
		u *cscustomer.Customer,
		conversationID uuid.UUID,
		text string,
		medias []cvmedia.Media,
	) (*cvmessage.WebhookMessage, error)
	ConversationSetup(ctx context.Context, u *cscustomer.Customer, referenceType cvconversation.ReferenceType) error

	// customer handlers
	CustomerCreate(
		ctx context.Context,
		u *cscustomer.Customer,
		username string,
		password string,
		name string,
		detail string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
		lineSecret string,
		lineToken string,
		permissionIDs []uuid.UUID,
	) (*cscustomer.WebhookMessage, error)
	CustomerGet(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID) (*cscustomer.WebhookMessage, error)
	CustomerGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cscustomer.WebhookMessage, error)
	CustomerUpdate(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string, webhookMethod cscustomer.WebhookMethod, webhookURI string) (*cscustomer.WebhookMessage, error)
	CustomerDelete(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID) (*cscustomer.WebhookMessage, error)
	CustomerUpdateLineInfo(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID, lineSecret string, lineToken string) (*cscustomer.WebhookMessage, error)
	CustomerUpdatePassword(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID, password string) (*cscustomer.WebhookMessage, error)
	CustomerUpdatePermissionIDs(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID, permissionIDs []uuid.UUID) (*cscustomer.WebhookMessage, error)

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
	FlowDelete(u *cscustomer.Customer, id uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowGet(u *cscustomer.Customer, id uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowGets(u *cscustomer.Customer, pageSize uint64, pageToken string) ([]*fmflow.WebhookMessage, error)
	FlowUpdate(u *cscustomer.Customer, f *fmflow.Flow) (*fmflow.WebhookMessage, error)

	// message handlers
	MessageDelete(u *cscustomer.Customer, id uuid.UUID) (*mmmessage.WebhookMessage, error)
	MessageGets(u *cscustomer.Customer, size uint64, token string) ([]*mmmessage.WebhookMessage, error)
	MessageGet(u *cscustomer.Customer, id uuid.UUID) (*mmmessage.WebhookMessage, error)
	MessageSend(u *cscustomer.Customer, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*mmmessage.WebhookMessage, error)

	// order numbers handler
	NumberCreate(u *cscustomer.Customer, num string, callFlowID, messageFlowID uuid.UUID, name, detail string) (*nmnumber.WebhookMessage, error)
	NumberGet(u *cscustomer.Customer, id uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberGets(u *cscustomer.Customer, size uint64, token string) ([]*nmnumber.WebhookMessage, error)
	NumberDelete(u *cscustomer.Customer, id uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberUpdate(u *cscustomer.Customer, id uuid.UUID, name, detail string) (*nmnumber.WebhookMessage, error)
	NumberUpdateFlowIDs(u *cscustomer.Customer, id, callFlowID, messageFlowID uuid.UUID) (*nmnumber.WebhookMessage, error)

	// outdials
	OutdialCreate(u *cscustomer.Customer, campaignID uuid.UUID, name, detail, data string) (*omoutdial.WebhookMessage, error)
	OutdialGetsByCustomerID(u *cscustomer.Customer, size uint64, token string) ([]*omoutdial.WebhookMessage, error)
	OutdialDelete(u *cscustomer.Customer, id uuid.UUID) (*omoutdial.WebhookMessage, error)
	OutdialGet(u *cscustomer.Customer, id uuid.UUID) (*omoutdial.WebhookMessage, error)
	OutdialtargetGetsByOutdialID(u *cscustomer.Customer, outdialID uuid.UUID, size uint64, token string) ([]*omoutdialtarget.WebhookMessage, error)
	OutdialUpdateBasicInfo(u *cscustomer.Customer, id uuid.UUID, name, detail string) (*omoutdial.WebhookMessage, error)
	OutdialUpdateCampaignID(u *cscustomer.Customer, id, campaignID uuid.UUID) (*omoutdial.WebhookMessage, error)
	OutdialUpdateData(u *cscustomer.Customer, id uuid.UUID, data string) (*omoutdial.WebhookMessage, error)

	// outdialtargets
	OutdialtargetCreate(
		u *cscustomer.Customer,
		outdialID uuid.UUID,
		name string,
		detail string,
		data string,
		destination0 *commonaddress.Address,
		destination1 *commonaddress.Address,
		destination2 *commonaddress.Address,
		destination3 *commonaddress.Address,
		destination4 *commonaddress.Address,
	) (*omoutdialtarget.WebhookMessage, error)
	OutdialtargetGet(u *cscustomer.Customer, outdialID uuid.UUID, outdialtargetID uuid.UUID) (*omoutdialtarget.WebhookMessage, error)
	OutdialtargetDelete(u *cscustomer.Customer, outdialID uuid.UUID, outdialtargetID uuid.UUID) (*omoutdialtarget.WebhookMessage, error)

	OutplanCreate(
		u *cscustomer.Customer,
		name string,
		detail string,
		source *commonaddress.Address,
		dialTimeout int,
		tryInterval int,
		maxTryCount0 int,
		maxTryCount1 int,
		maxTryCount2 int,
		maxTryCount3 int,
		maxTryCount4 int,
	) (*caoutplan.WebhookMessage, error)
	OutplanDelete(u *cscustomer.Customer, id uuid.UUID) (*caoutplan.WebhookMessage, error)
	OutplanGetsByCustomerID(u *cscustomer.Customer, size uint64, token string) ([]*caoutplan.WebhookMessage, error)
	OutplanGet(u *cscustomer.Customer, id uuid.UUID) (*caoutplan.WebhookMessage, error)
	OutplanUpdateBasicInfo(u *cscustomer.Customer, id uuid.UUID, name, detail string) (*caoutplan.WebhookMessage, error)
	OutplanUpdateDialInfo(
		u *cscustomer.Customer,
		id uuid.UUID,
		source *commonaddress.Address,
		dialTimeout int,
		tryInterval int,
		maxTryCount0 int,
		maxTryCount1 int,
		maxTryCount2 int,
		maxTryCount3 int,
		maxTryCount4 int,
	) (*caoutplan.WebhookMessage, error)

	// queue handlers
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

	// queuecall handlers
	QueuecallGet(ctx context.Context, u *cscustomer.Customer, queueID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*qmqueuecall.WebhookMessage, error)
	QueuecallDelete(ctx context.Context, u *cscustomer.Customer, queuecallID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallDeleteByReferenceID(ctx context.Context, u *cscustomer.Customer, referenceID uuid.UUID) (*qmqueuecall.WebhookMessage, error)

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

	WebsockCreate(ctx context.Context, u *cscustomer.Customer, w http.ResponseWriter, r *http.Request) error
}

type serviceHandler struct {
	reqHandler     requesthandler.RequestHandler
	dbHandler      dbhandler.DBHandler
	websockHandler websockhandler.WebsockHandler
}

// NewServiceHandler return ServiceHandler interface
func NewServiceHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
	websockHandler websockhandler.WebsockHandler,
) ServiceHandler {
	return &serviceHandler{
		reqHandler: reqHandler,
		dbHandler:  dbHandler,

		websockHandler: websockHandler,
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
