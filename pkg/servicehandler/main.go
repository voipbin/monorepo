package servicehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package servicehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	amtag "gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	cacampaigncall "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	caoutplan "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
	chatchat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
	chatchatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	chatmedia "gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	chatmessagechat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	chatmessagechatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
	chatbotchatbot "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
	chatbotchatbotcall "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	cfconferencecall "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	cvmedia "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	cvmessage "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
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
	rmprovider "gitlab.com/voipbin/bin-manager/route-manager.git/models/provider"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	tmtranscript "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/websockhandler"
)

const (
	defaultTimestamp string = "9999-01-01 00:00:00.000000" // default timestamp
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {

	// activeflows
	ActiveflowDelete(ctx context.Context, u *cscustomer.Customer, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error)
	ActiveflowGet(ctx context.Context, u *cscustomer.Customer, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error)
	ActiveflowGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*fmactiveflow.WebhookMessage, error)
	ActiveflowStop(ctx context.Context, u *cscustomer.Customer, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error)

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
	AuthLogin(ctx context.Context, username, password string) (string, error)

	// available numbers
	AvailableNumberGets(ctx context.Context, u *cscustomer.Customer, size uint64, countryCode string) ([]*nmavailablenumber.WebhookMessage, error)

	// call handlers
	CallCreate(ctx context.Context, u *cscustomer.Customer, flowID uuid.UUID, actions []fmaction.Action, source *commonaddress.Address, destinations []commonaddress.Address) ([]*cmcall.WebhookMessage, error)
	CallGet(ctx context.Context, u *cscustomer.Customer, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cmcall.WebhookMessage, error)
	CallDelete(ctx context.Context, u *cscustomer.Customer, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallHangup(ctx context.Context, u *cscustomer.Customer, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallTalk(ctx context.Context, u *cscustomer.Customer, callID uuid.UUID, text string, gender string, language string) error

	// campaign handlers
	CampaignCreate(
		ctx context.Context,
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
	CampaignGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cacampaign.WebhookMessage, error)
	CampaignGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cacampaign.WebhookMessage, error)
	CampaignDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cacampaign.WebhookMessage, error)
	CampaignUpdateBasicInfo(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string) (*cacampaign.WebhookMessage, error)
	CampaignUpdateStatus(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, status cacampaign.Status) (*cacampaign.WebhookMessage, error)
	CampaignUpdateServiceLevel(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, serviceLevel int) (*cacampaign.WebhookMessage, error)
	CampaignUpdateActions(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, actions []fmaction.Action) (*cacampaign.WebhookMessage, error)
	CampaignUpdateResourceInfo(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, outplanID uuid.UUID, outdialID uuid.UUID, queueID uuid.UUID) (*cacampaign.WebhookMessage, error)
	CampaignUpdateNextCampaignID(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.WebhookMessage, error)

	// campaigncall handlers
	CampaigncallGetsByCampaignID(ctx context.Context, u *cscustomer.Customer, campaignID uuid.UUID, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error)
	CampaigncallGet(ctx context.Context, u *cscustomer.Customer, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error)
	CampaigncallDelete(ctx context.Context, u *cscustomer.Customer, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error)

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

	// chatbot handlers
	ChatbotCreate(
		ctx context.Context,
		u *cscustomer.Customer,
		name string,
		detail string,
		engineType chatbotchatbot.EngineType,
	) (*chatbotchatbot.WebhookMessage, error)
	ChatbotGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*chatbotchatbot.WebhookMessage, error)
	ChatbotGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatbotchatbot.WebhookMessage, error)
	ChatbotDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatbotchatbot.WebhookMessage, error)

	// chatbotcall handlers
	ChatbotcallGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*chatbotchatbotcall.WebhookMessage, error)
	ChatbotcallGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatbotchatbotcall.WebhookMessage, error)
	ChatbotcallDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*chatbotchatbotcall.WebhookMessage, error)

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
	ConferenceCreate(ctx context.Context, u *cscustomer.Customer, confType cfconference.Type, name, detail string, preActions, postActions []fmaction.Action) (*cfconference.WebhookMessage, error)
	ConferenceDelete(ctx context.Context, u *cscustomer.Customer, confID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cfconference.WebhookMessage, error)
	ConferenceRecordingStart(ctx context.Context, u *cscustomer.Customer, confID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceRecordingStop(ctx context.Context, u *cscustomer.Customer, confID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceTranscribeStart(ctx context.Context, u *cscustomer.Customer, conferenceID uuid.UUID, language string) (*cfconference.WebhookMessage, error)
	ConferenceTranscribeStop(ctx context.Context, u *cscustomer.Customer, conferenceID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceUpdate(
		ctx context.Context,
		u *cscustomer.Customer,
		cfID uuid.UUID,
		name string,
		detail string,
		timeout int,
		preActions []fmaction.Action,
		postActions []fmaction.Action,
	) (*cfconference.WebhookMessage, error)

	// conferencecall handlers
	ConferencecallGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cfconferencecall.WebhookMessage, error)
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
	DomainCreate(ctx context.Context, u *cscustomer.Customer, domainName, name, detail string) (*rmdomain.WebhookMessage, error)
	DomainDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmdomain.WebhookMessage, error)
	DomainGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmdomain.WebhookMessage, error)
	DomainGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*rmdomain.WebhookMessage, error)
	DomainUpdate(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string) (*rmdomain.WebhookMessage, error)

	// extension handlers
	ExtensionCreate(ctx context.Context, u *cscustomer.Customer, ext, password string, domainID uuid.UUID, name, detail string) (*rmextension.WebhookMessage, error)
	ExtensionDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmextension.WebhookMessage, error)
	ExtensionGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmextension.WebhookMessage, error)
	ExtensionGets(ctx context.Context, u *cscustomer.Customer, domainID uuid.UUID, size uint64, token string) ([]*rmextension.WebhookMessage, error)
	ExtensionUpdate(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail, password string) (*rmextension.WebhookMessage, error)

	// flow handlers
	FlowCreate(ctx context.Context, u *cscustomer.Customer, name, detail string, actions []fmaction.Action, persist bool) (*fmflow.WebhookMessage, error)
	FlowDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowGets(ctx context.Context, u *cscustomer.Customer, pageSize uint64, pageToken string) ([]*fmflow.WebhookMessage, error)
	FlowUpdate(ctx context.Context, u *cscustomer.Customer, f *fmflow.Flow) (*fmflow.WebhookMessage, error)

	// grpupcall handlers
	GroupcallGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cmgroupcall.WebhookMessage, error)
	GroupcallGet(ctx context.Context, u *cscustomer.Customer, groupcallID uuid.UUID) (*cmgroupcall.WebhookMessage, error)
	GroupcallCreate(ctx context.Context, u *cscustomer.Customer, source commonaddress.Address, destinations []commonaddress.Address, flowID uuid.UUID, actions []fmaction.Action, ringMethod cmgroupcall.RingMethod, answerMethod cmgroupcall.AnswerMethod) (*cmgroupcall.WebhookMessage, error)
	GroupcallHangup(ctx context.Context, u *cscustomer.Customer, groupcallID uuid.UUID) (*cmgroupcall.WebhookMessage, error)
	GroupcallDelete(ctx context.Context, u *cscustomer.Customer, callID uuid.UUID) (*cmgroupcall.WebhookMessage, error)

	// message handlers
	MessageDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*mmmessage.WebhookMessage, error)
	MessageGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*mmmessage.WebhookMessage, error)
	MessageGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*mmmessage.WebhookMessage, error)
	MessageSend(ctx context.Context, u *cscustomer.Customer, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*mmmessage.WebhookMessage, error)

	// order numbers handler
	NumberCreate(ctx context.Context, u *cscustomer.Customer, num string, callFlowID, messageFlowID uuid.UUID, name, detail string) (*nmnumber.WebhookMessage, error)
	NumberGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*nmnumber.WebhookMessage, error)
	NumberDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberUpdate(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string) (*nmnumber.WebhookMessage, error)
	NumberUpdateFlowIDs(ctx context.Context, u *cscustomer.Customer, id, callFlowID, messageFlowID uuid.UUID) (*nmnumber.WebhookMessage, error)

	// outdials
	OutdialCreate(ctx context.Context, u *cscustomer.Customer, campaignID uuid.UUID, name, detail, data string) (*omoutdial.WebhookMessage, error)
	OutdialGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*omoutdial.WebhookMessage, error)
	OutdialDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*omoutdial.WebhookMessage, error)
	OutdialGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*omoutdial.WebhookMessage, error)
	OutdialtargetGetsByOutdialID(ctx context.Context, u *cscustomer.Customer, outdialID uuid.UUID, size uint64, token string) ([]*omoutdialtarget.WebhookMessage, error)
	OutdialUpdateBasicInfo(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string) (*omoutdial.WebhookMessage, error)
	OutdialUpdateCampaignID(ctx context.Context, u *cscustomer.Customer, id, campaignID uuid.UUID) (*omoutdial.WebhookMessage, error)
	OutdialUpdateData(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, data string) (*omoutdial.WebhookMessage, error)

	// outdialtargets
	OutdialtargetCreate(
		ctx context.Context,
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
	OutdialtargetGet(ctx context.Context, u *cscustomer.Customer, outdialID uuid.UUID, outdialtargetID uuid.UUID) (*omoutdialtarget.WebhookMessage, error)
	OutdialtargetDelete(ctx context.Context, u *cscustomer.Customer, outdialID uuid.UUID, outdialtargetID uuid.UUID) (*omoutdialtarget.WebhookMessage, error)

	OutplanCreate(
		ctx context.Context,
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
	OutplanDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*caoutplan.WebhookMessage, error)
	OutplanGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*caoutplan.WebhookMessage, error)
	OutplanGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*caoutplan.WebhookMessage, error)
	OutplanUpdateBasicInfo(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string) (*caoutplan.WebhookMessage, error)
	OutplanUpdateDialInfo(
		ctx context.Context,
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

	// provider handlers
	ProviderCreate(
		ctx context.Context,
		u *cscustomer.Customer,
		providerType rmprovider.Type,
		hostname string,
		techPrefix string,
		techPostfix string,
		techHeaders map[string]string,
		name string,
		detail string,
	) (*rmprovider.WebhookMessage, error)
	ProviderDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmprovider.WebhookMessage, error)
	ProviderGet(ctx context.Context, u *cscustomer.Customer, providerID uuid.UUID) (*rmprovider.WebhookMessage, error)
	ProviderGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*rmprovider.WebhookMessage, error)
	ProviderUpdate(
		ctx context.Context,
		u *cscustomer.Customer,
		providerID uuid.UUID,
		providerType rmprovider.Type,
		hostname string,
		techPrefix string,
		techPostfix string,
		techHeaders map[string]string,
		name string,
		detail string,
	) (*rmprovider.WebhookMessage, error)

	// queue handlers
	QueueGet(ctx context.Context, u *cscustomer.Customer, queueID uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*qmqueue.WebhookMessage, error)
	QueueCreate(
		ctx context.Context,
		u *cscustomer.Customer,
		name string,
		detail string,
		routingMethod string,
		tagIDs []uuid.UUID,
		waitActions []fmaction.Action,
		timeoutWait int,
		timeoutService int,
	) (*qmqueue.WebhookMessage, error)
	QueueDelete(ctx context.Context, u *cscustomer.Customer, queueID uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueUpdate(ctx context.Context, u *cscustomer.Customer, queueID uuid.UUID, name, detail string) (*qmqueue.WebhookMessage, error)
	QueueUpdateTagIDs(ctx context.Context, u *cscustomer.Customer, queueID uuid.UUID, tagIDs []uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueUpdateRoutingMethod(ctx context.Context, u *cscustomer.Customer, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) (*qmqueue.WebhookMessage, error)
	QueueUpdateActions(ctx context.Context, u *cscustomer.Customer, queueID uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) (*qmqueue.WebhookMessage, error)

	// queuecall handlers
	QueuecallGet(ctx context.Context, u *cscustomer.Customer, queueID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*qmqueuecall.WebhookMessage, error)
	QueuecallDelete(ctx context.Context, u *cscustomer.Customer, queuecallID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallKick(ctx context.Context, u *cscustomer.Customer, queuecallID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallKickByReferenceID(ctx context.Context, u *cscustomer.Customer, referenceID uuid.UUID) (*qmqueuecall.WebhookMessage, error)

	// recording handlers
	RecordingGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cmrecording.WebhookMessage, error)
	RecordingGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cmrecording.WebhookMessage, error)
	RecordingDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*cmrecording.WebhookMessage, error)

	// recordingfile handlers
	RecordingfileGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (string, error)

	// route handlers
	RouteGet(ctx context.Context, u *cscustomer.Customer, routeID uuid.UUID) (*rmroute.WebhookMessage, error)
	RouteGets(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID, size uint64, token string) ([]*rmroute.WebhookMessage, error)
	RouteCreate(
		ctx context.Context,
		u *cscustomer.Customer,
		customerID uuid.UUID,
		providerID uuid.UUID,
		priority int,
		target string,
	) (*rmroute.WebhookMessage, error)
	RouteDelete(ctx context.Context, u *cscustomer.Customer, routeID uuid.UUID) (*rmroute.WebhookMessage, error)
	RouteUpdate(ctx context.Context, u *cscustomer.Customer, routeID, providerID uuid.UUID, priority int, target string) (*rmroute.WebhookMessage, error)

	// tag handlers
	TagCreate(ctx context.Context, u *cscustomer.Customer, name string, detail string) (*amtag.WebhookMessage, error)
	TagDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*amtag.WebhookMessage, error)
	TagGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*amtag.WebhookMessage, error)
	TagGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*amtag.WebhookMessage, error)
	TagUpdate(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string) (*amtag.WebhookMessage, error)

	// transcribe handlers
	TranscribeGet(ctx context.Context, u *cscustomer.Customer, routeID uuid.UUID) (*tmtranscribe.WebhookMessage, error)
	TranscribeGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*tmtranscribe.WebhookMessage, error)
	TranscribeStart(ctx context.Context, u *cscustomer.Customer, referenceType tmtranscribe.ReferenceType, referenceID uuid.UUID, language string, direction tmtranscribe.Direction) (*tmtranscribe.WebhookMessage, error)
	TranscribeStop(ctx context.Context, u *cscustomer.Customer, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error)
	TranscribeDelete(ctx context.Context, u *cscustomer.Customer, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error)

	TranscriptGets(ctx context.Context, u *cscustomer.Customer, transcribeID uuid.UUID) ([]*tmtranscript.WebhookMessage, error)

	WebsockCreate(ctx context.Context, u *cscustomer.Customer, w http.ResponseWriter, r *http.Request) error
}

type serviceHandler struct {
	utilHandler    utilhandler.UtilHandler
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
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  reqHandler,
		dbHandler:   dbHandler,

		websockHandler: websockHandler,
	}
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
