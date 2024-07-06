package servicehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package servicehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"mime/multipart"
	"net/http"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmrecording "monorepo/bin-call-manager/models/recording"
	smaccount "monorepo/bin-storage-manager/models/account"
	smfile "monorepo/bin-storage-manager/models/file"

	bmaccount "monorepo/bin-billing-manager/models/account"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	cacampaign "monorepo/bin-campaign-manager/models/campaign"
	cacampaigncall "monorepo/bin-campaign-manager/models/campaigncall"
	caoutplan "monorepo/bin-campaign-manager/models/outplan"
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	chatchat "monorepo/bin-chat-manager/models/chat"
	chatchatroom "monorepo/bin-chat-manager/models/chatroom"
	chatmedia "monorepo/bin-chat-manager/models/media"
	chatmessagechat "monorepo/bin-chat-manager/models/messagechat"
	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"
	chatbotchatbot "monorepo/bin-chatbot-manager/models/chatbot"
	chatbotchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"

	cfconference "monorepo/bin-conference-manager/models/conference"
	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"

	cvaccount "monorepo/bin-conversation-manager/models/account"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmflow "monorepo/bin-flow-manager/models/flow"

	mmmessage "monorepo/bin-message-manager/models/message"

	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"
	nmnumber "monorepo/bin-number-manager/models/number"

	omoutdial "monorepo/bin-outdial-manager/models/outdial"
	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"
	qmqueue "monorepo/bin-queue-manager/models/queue"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	rmextension "monorepo/bin-registrar-manager/models/extension"
	rmsipauth "monorepo/bin-registrar-manager/models/sipauth"
	rmtrunk "monorepo/bin-registrar-manager/models/trunk"

	rmprovider "monorepo/bin-route-manager/models/provider"
	rmroute "monorepo/bin-route-manager/models/route"

	tmtag "monorepo/bin-tag-manager/models/tag"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"

	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/websockhandler"

	"cloud.google.com/go/storage"
)

const (
	defaultTimestamp string = "9999-01-01 00:00:00.000000" // default timestamp
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {

	// activeflows
	ActiveflowCreate(ctx context.Context, a *amagent.Agent, activeflowID uuid.UUID, flowID uuid.UUID, actions []fmaction.Action) (*fmactiveflow.WebhookMessage, error)
	ActiveflowDelete(ctx context.Context, a *amagent.Agent, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error)
	ActiveflowGet(ctx context.Context, a *amagent.Agent, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error)
	ActiveflowGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*fmactiveflow.WebhookMessage, error)
	ActiveflowStop(ctx context.Context, a *amagent.Agent, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error)

	// agent handlers
	AgentCreate(
		ctx context.Context,
		a *amagent.Agent,
		username string,
		password string,
		name string,
		detail string,
		ringMethod amagent.RingMethod,
		permission amagent.Permission,
		tagIDs []uuid.UUID,
		addresses []commonaddress.Address,
	) (*amagent.WebhookMessage, error)
	AgentGet(ctx context.Context, a *amagent.Agent, agentID uuid.UUID) (*amagent.WebhookMessage, error)
	AgentGets(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*amagent.WebhookMessage, error)
	AgentDelete(ctx context.Context, a *amagent.Agent, agentID uuid.UUID) (*amagent.WebhookMessage, error)
	AgentUpdate(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, name, detail string, ringMethod amagent.RingMethod) (*amagent.WebhookMessage, error)
	AgentUpdateAddresses(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, addresses []commonaddress.Address) (*amagent.WebhookMessage, error)
	AgentUpdatePassword(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, password string) (*amagent.WebhookMessage, error)
	AgentUpdatePermission(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, permission amagent.Permission) (*amagent.WebhookMessage, error)
	AgentUpdateStatus(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, status amagent.Status) (*amagent.WebhookMessage, error)
	AgentUpdateTagIDs(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, tagIDs []uuid.UUID) (*amagent.WebhookMessage, error)

	// auth handlers
	AuthLogin(ctx context.Context, username, password string) (string, error)

	// available numbers
	AvailableNumberGets(ctx context.Context, a *amagent.Agent, size uint64, countryCode string) ([]*nmavailablenumber.WebhookMessage, error)

	// billing accounts
	BillingAccountCreate(ctx context.Context, a *amagent.Agent, name string, detail string, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.WebhookMessage, error)
	BillingAccountGet(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID) (*bmaccount.WebhookMessage, error)
	BillingAccountGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*bmaccount.WebhookMessage, error)
	BillingAccountDelete(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID) (*bmaccount.WebhookMessage, error)
	BillingAccountAddBalanceForce(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, balance float32) (*bmaccount.WebhookMessage, error)
	BillingAccountSubtractBalanceForce(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, balance float32) (*bmaccount.WebhookMessage, error)
	BillingAccountUpdateBasicInfo(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, name string, detail string) (*bmaccount.WebhookMessage, error)
	BillingAccountUpdatePaymentInfo(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.WebhookMessage, error)

	// billings
	BillingGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*bmbilling.WebhookMessage, error)

	// call handlers
	CallCreate(ctx context.Context, a *amagent.Agent, flowID uuid.UUID, actions []fmaction.Action, source *commonaddress.Address, destinations []commonaddress.Address) ([]*cmcall.WebhookMessage, []*cmgroupcall.WebhookMessage, error)
	CallGet(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cmcall.WebhookMessage, error)
	CallDelete(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallHangup(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallTalk(ctx context.Context, a *amagent.Agent, callID uuid.UUID, text string, gender string, language string) error
	CallHoldOn(ctx context.Context, a *amagent.Agent, callID uuid.UUID) error
	CallHoldOff(ctx context.Context, a *amagent.Agent, callID uuid.UUID) error
	CallMediaStreamStart(ctx context.Context, a *amagent.Agent, callID uuid.UUID, encapsulation string, w http.ResponseWriter, r *http.Request) error
	CallMuteOn(ctx context.Context, a *amagent.Agent, callID uuid.UUID, direction cmcall.MuteDirection) error
	CallMuteOff(ctx context.Context, a *amagent.Agent, callID uuid.UUID, direction cmcall.MuteDirection) error
	CallMOHOn(ctx context.Context, a *amagent.Agent, callID uuid.UUID) error
	CallMOHOff(ctx context.Context, a *amagent.Agent, callID uuid.UUID) error
	CallSilenceOn(ctx context.Context, a *amagent.Agent, callID uuid.UUID) error
	CallSilenceOff(ctx context.Context, a *amagent.Agent, callID uuid.UUID) error

	// campaign handlers
	CampaignCreate(
		ctx context.Context,
		a *amagent.Agent,
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
	CampaignGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cacampaign.WebhookMessage, error)
	CampaignGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cacampaign.WebhookMessage, error)
	CampaignDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cacampaign.WebhookMessage, error)
	CampaignUpdateBasicInfo(
		ctx context.Context,
		a *amagent.Agent,
		id uuid.UUID,
		name string,
		detail string,
		campaignType cacampaign.Type,
		serviceLevel int,
		endHandle cacampaign.EndHandle,
	) (*cacampaign.WebhookMessage, error)
	CampaignUpdateStatus(ctx context.Context, a *amagent.Agent, id uuid.UUID, status cacampaign.Status) (*cacampaign.WebhookMessage, error)
	CampaignUpdateServiceLevel(ctx context.Context, a *amagent.Agent, id uuid.UUID, serviceLevel int) (*cacampaign.WebhookMessage, error)
	CampaignUpdateActions(ctx context.Context, a *amagent.Agent, id uuid.UUID, actions []fmaction.Action) (*cacampaign.WebhookMessage, error)
	CampaignUpdateResourceInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, outplanID uuid.UUID, outdialID uuid.UUID, queueID uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.WebhookMessage, error)
	CampaignUpdateNextCampaignID(ctx context.Context, a *amagent.Agent, id uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.WebhookMessage, error)

	// campaigncall handlers
	CampaigncallGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error)
	CampaigncallGetsByCampaignID(ctx context.Context, a *amagent.Agent, campaignID uuid.UUID, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error)
	CampaigncallGet(ctx context.Context, a *amagent.Agent, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error)
	CampaigncallDelete(ctx context.Context, a *amagent.Agent, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error)

	// chat handlers
	ChatCreate(
		ctx context.Context,
		a *amagent.Agent,
		chatType chatchat.Type,
		roomOwnerID uuid.UUID,
		participantIDs []uuid.UUID,
		name string,
		detail string,
	) (*chatchat.WebhookMessage, error)
	ChatGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*chatchat.WebhookMessage, error)
	ChatGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatchat.WebhookMessage, error)
	ChatDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatchat.WebhookMessage, error)
	ChatUpdateBasicInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail string) (*chatchat.WebhookMessage, error)
	ChatUpdateRoomOwnerID(ctx context.Context, a *amagent.Agent, id uuid.UUID, roomOwnerID uuid.UUID) (*chatchat.WebhookMessage, error)
	ChatAddParticipantID(ctx context.Context, a *amagent.Agent, id uuid.UUID, participantID uuid.UUID) (*chatchat.WebhookMessage, error)
	ChatRemoveParticipantID(ctx context.Context, a *amagent.Agent, id uuid.UUID, participantID uuid.UUID) (*chatchat.WebhookMessage, error)

	// chatbot handlers
	ChatbotCreate(
		ctx context.Context,
		a *amagent.Agent,
		name string,
		detail string,
		engineType chatbotchatbot.EngineType,
		initPrompt string,
	) (*chatbotchatbot.WebhookMessage, error)
	ChatbotGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*chatbotchatbot.WebhookMessage, error)
	ChatbotGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatbotchatbot.WebhookMessage, error)
	ChatbotDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatbotchatbot.WebhookMessage, error)
	ChatbotUpdate(
		ctx context.Context,
		a *amagent.Agent,
		id uuid.UUID,
		name string,
		detail string,
		engineType chatbotchatbot.EngineType,
		initPrompt string,
	) (*chatbotchatbot.WebhookMessage, error)

	// chatbotcall handlers
	ChatbotcallGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*chatbotchatbotcall.WebhookMessage, error)
	ChatbotcallGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatbotchatbotcall.WebhookMessage, error)
	ChatbotcallDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatbotchatbotcall.WebhookMessage, error)

	// chatmessage handlers
	ChatmessageCreate(
		ctx context.Context,
		a *amagent.Agent,
		chatID uuid.UUID,
		source commonaddress.Address,
		messageType chatmessagechat.Type,
		text string,
		medias []chatmedia.Media,
	) (*chatmessagechat.WebhookMessage, error)
	ChatmessageGetsByChatID(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, size uint64, token string) ([]*chatmessagechat.WebhookMessage, error)
	ChatmessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechat.WebhookMessage, error)
	ChatmessageDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechat.WebhookMessage, error)

	// chatroom handlers
	ChatroomCreate(ctx context.Context, a *amagent.Agent, participantIDs []uuid.UUID, name string, detail string) (*chatchatroom.WebhookMessage, error)
	ChatroomGetsByOwnerID(ctx context.Context, a *amagent.Agent, ownerID uuid.UUID, size uint64, token string) ([]*chatchatroom.WebhookMessage, error)
	ChatroomGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatchatroom.WebhookMessage, error)
	ChatroomDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatchatroom.WebhookMessage, error)
	ChatroomUpdateBasicInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail string) (*chatchatroom.WebhookMessage, error)

	// chatroommessage handlers
	ChatroommessageCreate(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID, message string, medias []chatmedia.Media) (*chatmessagechatroom.WebhookMessage, error)
	ChatroommessageGetsByChatroomID(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID, size uint64, token string) ([]*chatmessagechatroom.WebhookMessage, error)
	ChatroommessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechatroom.WebhookMessage, error)
	ChatroommessageDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechatroom.WebhookMessage, error)

	// conference handlers
	ConferenceCreate(
		ctx context.Context,
		a *amagent.Agent,
		confType cfconference.Type,
		name string,
		detail string,
		timeout int,
		data map[string]interface{},
		preActions []fmaction.Action,
		postActions []fmaction.Action,
	) (*cfconference.WebhookMessage, error)
	ConferenceDelete(ctx context.Context, a *amagent.Agent, confID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cfconference.WebhookMessage, error)
	ConferenceMediaStreamStart(ctx context.Context, a *amagent.Agent, conferenceID uuid.UUID, encapsulation string, w http.ResponseWriter, r *http.Request) error
	ConferenceRecordingStart(ctx context.Context, a *amagent.Agent, confID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceRecordingStop(ctx context.Context, a *amagent.Agent, confID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceTranscribeStart(ctx context.Context, a *amagent.Agent, conferenceID uuid.UUID, language string) (*cfconference.WebhookMessage, error)
	ConferenceTranscribeStop(ctx context.Context, a *amagent.Agent, conferenceID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceUpdate(
		ctx context.Context,
		a *amagent.Agent,
		cfID uuid.UUID,
		name string,
		detail string,
		timeout int,
		preActions []fmaction.Action,
		postActions []fmaction.Action,
	) (*cfconference.WebhookMessage, error)

	// conferencecall handlers
	ConferencecallGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cfconferencecall.WebhookMessage, error)
	ConferencecallGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cfconferencecall.WebhookMessage, error)
	ConferencecallKick(ctx context.Context, a *amagent.Agent, conferencecallID uuid.UUID) (*cfconferencecall.WebhookMessage, error)

	// conversation handlers
	ConversationGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cvconversation.WebhookMessage, error)
	ConversationGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cvconversation.WebhookMessage, error)
	ConversationUpdate(ctx context.Context, a *amagent.Agent, conversationID uuid.UUID, name string, detail string) (*cvconversation.WebhookMessage, error)
	ConversationMessageGetsByConversationID(
		ctx context.Context,
		a *amagent.Agent,
		conversationID uuid.UUID,
		size uint64,
		token string,
	) ([]*cvmessage.WebhookMessage, error)
	ConversationMessageSend(
		ctx context.Context,
		a *amagent.Agent,
		conversationID uuid.UUID,
		text string,
		medias []cvmedia.Media,
	) (*cvmessage.WebhookMessage, error)

	ConversationAccountGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cvaccount.WebhookMessage, error)
	ConversationAccountGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cvaccount.WebhookMessage, error)
	ConversationAccountCreate(
		ctx context.Context,
		a *amagent.Agent,
		accountType cvaccount.Type,
		name string,
		detail string,
		secret string,
		token string,
	) (*cvaccount.WebhookMessage, error)
	ConversationAccountUpdate(
		ctx context.Context,
		a *amagent.Agent,
		accountID uuid.UUID,
		name string,
		detail string,
		secret string,
		token string,
	) (*cvaccount.WebhookMessage, error)
	ConversationAccountDelete(ctx context.Context, a *amagent.Agent, accountID uuid.UUID) (*cvaccount.WebhookMessage, error)

	// customer handlers
	CustomerCreate(
		ctx context.Context,
		a *amagent.Agent,
		username string,
		password string,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
	) (*cscustomer.WebhookMessage, error)
	CustomerGet(ctx context.Context, a *amagent.Agent, customerID uuid.UUID) (*cscustomer.WebhookMessage, error)
	CustomerGets(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*cscustomer.WebhookMessage, error)
	CustomerUpdate(
		ctx context.Context,
		a *amagent.Agent,
		id uuid.UUID,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
	) (*cscustomer.WebhookMessage, error)
	CustomerDelete(ctx context.Context, a *amagent.Agent, customerID uuid.UUID) (*cscustomer.WebhookMessage, error)
	CustomerUpdateBillingAccountID(ctx context.Context, a *amagent.Agent, customerID uuid.UUID, billingAccountID uuid.UUID) (*cscustomer.WebhookMessage, error)

	// extension handlers
	ExtensionCreate(ctx context.Context, a *amagent.Agent, ext string, password string, name string, detail string) (*rmextension.WebhookMessage, error)
	ExtensionDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmextension.WebhookMessage, error)
	ExtensionGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmextension.WebhookMessage, error)
	ExtensionGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmextension.WebhookMessage, error)
	ExtensionUpdate(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail, password string) (*rmextension.WebhookMessage, error)

	// flow handlers
	FlowCreate(ctx context.Context, a *amagent.Agent, name, detail string, actions []fmaction.Action, persist bool) (*fmflow.WebhookMessage, error)
	FlowDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowGets(ctx context.Context, a *amagent.Agent, pageSize uint64, pageToken string) ([]*fmflow.WebhookMessage, error)
	FlowUpdate(ctx context.Context, a *amagent.Agent, f *fmflow.Flow) (*fmflow.WebhookMessage, error)

	// grpupcall handlers
	GroupcallGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cmgroupcall.WebhookMessage, error)
	GroupcallGet(ctx context.Context, a *amagent.Agent, groupcallID uuid.UUID) (*cmgroupcall.WebhookMessage, error)
	GroupcallCreate(ctx context.Context, a *amagent.Agent, source commonaddress.Address, destinations []commonaddress.Address, flowID uuid.UUID, actions []fmaction.Action, ringMethod cmgroupcall.RingMethod, answerMethod cmgroupcall.AnswerMethod) (*cmgroupcall.WebhookMessage, error)
	GroupcallHangup(ctx context.Context, a *amagent.Agent, groupcallID uuid.UUID) (*cmgroupcall.WebhookMessage, error)
	GroupcallDelete(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmgroupcall.WebhookMessage, error)

	// message handlers
	MessageDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*mmmessage.WebhookMessage, error)
	MessageGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*mmmessage.WebhookMessage, error)
	MessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*mmmessage.WebhookMessage, error)
	MessageSend(ctx context.Context, a *amagent.Agent, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*mmmessage.WebhookMessage, error)

	// order numbers handler
	NumberCreate(ctx context.Context, a *amagent.Agent, num string, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*nmnumber.WebhookMessage, error)
	NumberGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*nmnumber.WebhookMessage, error)
	NumberDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberUpdate(ctx context.Context, a *amagent.Agent, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*nmnumber.WebhookMessage, error)
	NumberUpdateFlowIDs(ctx context.Context, a *amagent.Agent, id, callFlowID uuid.UUID, messageFlowID uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberRenew(ctx context.Context, a *amagent.Agent, tmRenew string) ([]*nmnumber.WebhookMessage, error)

	// outdials
	OutdialCreate(ctx context.Context, a *amagent.Agent, campaignID uuid.UUID, name, detail, data string) (*omoutdial.WebhookMessage, error)
	OutdialGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*omoutdial.WebhookMessage, error)
	OutdialDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*omoutdial.WebhookMessage, error)
	OutdialGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*omoutdial.WebhookMessage, error)
	OutdialtargetGetsByOutdialID(ctx context.Context, a *amagent.Agent, outdialID uuid.UUID, size uint64, token string) ([]*omoutdialtarget.WebhookMessage, error)
	OutdialUpdateBasicInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail string) (*omoutdial.WebhookMessage, error)
	OutdialUpdateCampaignID(ctx context.Context, a *amagent.Agent, id, campaignID uuid.UUID) (*omoutdial.WebhookMessage, error)
	OutdialUpdateData(ctx context.Context, a *amagent.Agent, id uuid.UUID, data string) (*omoutdial.WebhookMessage, error)

	// outdialtargets
	OutdialtargetCreate(
		ctx context.Context,
		a *amagent.Agent,
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
	OutdialtargetGet(ctx context.Context, a *amagent.Agent, outdialID uuid.UUID, outdialtargetID uuid.UUID) (*omoutdialtarget.WebhookMessage, error)
	OutdialtargetDelete(ctx context.Context, a *amagent.Agent, outdialID uuid.UUID, outdialtargetID uuid.UUID) (*omoutdialtarget.WebhookMessage, error)

	OutplanCreate(
		ctx context.Context,
		a *amagent.Agent,
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
	OutplanDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*caoutplan.WebhookMessage, error)
	OutplanGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*caoutplan.WebhookMessage, error)
	OutplanGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*caoutplan.WebhookMessage, error)
	OutplanUpdateBasicInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail string) (*caoutplan.WebhookMessage, error)
	OutplanUpdateDialInfo(
		ctx context.Context,
		a *amagent.Agent,
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
		a *amagent.Agent,
		providerType rmprovider.Type,
		hostname string,
		techPrefix string,
		techPostfix string,
		techHeaders map[string]string,
		name string,
		detail string,
	) (*rmprovider.WebhookMessage, error)
	ProviderDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmprovider.WebhookMessage, error)
	ProviderGet(ctx context.Context, a *amagent.Agent, providerID uuid.UUID) (*rmprovider.WebhookMessage, error)
	ProviderGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmprovider.WebhookMessage, error)
	ProviderUpdate(
		ctx context.Context,
		a *amagent.Agent,
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
	QueueGet(ctx context.Context, a *amagent.Agent, queueID uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*qmqueue.WebhookMessage, error)
	QueueCreate(
		ctx context.Context,
		a *amagent.Agent,
		name string,
		detail string,
		routingMethod qmqueue.RoutingMethod,
		tagIDs []uuid.UUID,
		waitActions []fmaction.Action,
		timeoutWait int,
		timeoutService int,
	) (*qmqueue.WebhookMessage, error)
	QueueDelete(ctx context.Context, a *amagent.Agent, queueID uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueUpdate(
		ctx context.Context,
		a *amagent.Agent,
		queueID uuid.UUID,
		name string,
		detail string,
		routingMethod qmqueue.RoutingMethod,
		tagIDs []uuid.UUID,
		waitActions []fmaction.Action,
		timeoutWait int,
		timeoutService int,
	) (*qmqueue.WebhookMessage, error)
	QueueUpdateTagIDs(ctx context.Context, a *amagent.Agent, queueID uuid.UUID, tagIDs []uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueUpdateRoutingMethod(ctx context.Context, a *amagent.Agent, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) (*qmqueue.WebhookMessage, error)
	QueueUpdateActions(ctx context.Context, a *amagent.Agent, queueID uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) (*qmqueue.WebhookMessage, error)

	// queuecall handlers
	QueuecallGet(ctx context.Context, a *amagent.Agent, queueID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*qmqueuecall.WebhookMessage, error)
	QueuecallDelete(ctx context.Context, a *amagent.Agent, queuecallID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallKick(ctx context.Context, a *amagent.Agent, queuecallID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallKickByReferenceID(ctx context.Context, a *amagent.Agent, referenceID uuid.UUID) (*qmqueuecall.WebhookMessage, error)

	// recording handlers
	RecordingGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cmrecording.WebhookMessage, error)
	RecordingGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cmrecording.WebhookMessage, error)
	RecordingDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cmrecording.WebhookMessage, error)

	// recordingfile handlers
	RecordingfileGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (string, error)

	// route handlers
	RouteGet(ctx context.Context, a *amagent.Agent, routeID uuid.UUID) (*rmroute.WebhookMessage, error)
	RouteGetsByCustomerID(ctx context.Context, a *amagent.Agent, customerID uuid.UUID, size uint64, token string) ([]*rmroute.WebhookMessage, error)
	RouteGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmroute.WebhookMessage, error)
	RouteCreate(
		ctx context.Context,
		a *amagent.Agent,
		customerID uuid.UUID,
		name string,
		detail string,
		providerID uuid.UUID,
		priority int,
		target string,
	) (*rmroute.WebhookMessage, error)
	RouteDelete(ctx context.Context, a *amagent.Agent, routeID uuid.UUID) (*rmroute.WebhookMessage, error)
	RouteUpdate(
		ctx context.Context,
		a *amagent.Agent,
		routeID uuid.UUID,
		name string,
		detail string,
		providerID uuid.UUID,
		priority int,
		target string,
	) (*rmroute.WebhookMessage, error)

	// service_agent call
	ServiceAgentCallGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cmcall.WebhookMessage, error)
	ServiceAgentCallGet(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	ServiceAgentCallDelete(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error)

	// service_agent chatroom
	ServiceAgentChatroomGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*chatchatroom.WebhookMessage, error)
	ServiceAgentChatroomGet(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID) (*chatchatroom.WebhookMessage, error)
	ServiceAgentChatroomDelete(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID) (*chatchatroom.WebhookMessage, error)
	ServiceAgentChatroomCreate(ctx context.Context, a *amagent.Agent, participantIDs []uuid.UUID, name string, detail string) (*chatchatroom.WebhookMessage, error)
	ServiceAgentChatroomUpdateBasicInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail string) (*chatchatroom.WebhookMessage, error)

	// service_agent chatroom message
	ServiceAgentChatroommessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*chatmessagechatroom.WebhookMessage, error)
	ServiceAgentChatroommessageGets(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID, size uint64, token string) ([]*chatmessagechatroom.WebhookMessage, error)
	ServiceAgentChatroommessageCreate(ctx context.Context, a *amagent.Agent, chatroomID uuid.UUID, message string, medias []chatmedia.Media) (*chatmessagechatroom.WebhookMessage, error)

	// storage account
	StorageAccountCreate(ctx context.Context, a *amagent.Agent, customerID uuid.UUID) (*smaccount.WebhookMessage, error)
	StorageAccountGet(ctx context.Context, a *amagent.Agent, storageAccountID uuid.UUID) (*smaccount.WebhookMessage, error)
	StorageAccountGetByCustomerID(ctx context.Context, a *amagent.Agent) (*smaccount.WebhookMessage, error)
	StorageAccountGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*smaccount.WebhookMessage, error)
	StorageAccountDelete(ctx context.Context, a *amagent.Agent, storageAccountID uuid.UUID) (*smaccount.WebhookMessage, error)

	// storage file handlers
	StorageFileCreate(ctx context.Context, a *amagent.Agent, f multipart.File, name string, detail string, filename string) (*smfile.WebhookMessage, error)
	StorageFileDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*smfile.WebhookMessage, error)
	StorageFileGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*smfile.WebhookMessage, error)
	StorageFileGetsByOnwerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*smfile.WebhookMessage, error)

	// tag handlers
	TagCreate(ctx context.Context, a *amagent.Agent, name string, detail string) (*tmtag.WebhookMessage, error)
	TagDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmtag.WebhookMessage, error)
	TagGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmtag.WebhookMessage, error)
	TagGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tmtag.WebhookMessage, error)
	TagUpdate(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail string) (*tmtag.WebhookMessage, error)

	// transcribe handlers
	TranscribeGet(ctx context.Context, a *amagent.Agent, routeID uuid.UUID) (*tmtranscribe.WebhookMessage, error)
	TranscribeGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tmtranscribe.WebhookMessage, error)
	TranscribeStart(ctx context.Context, a *amagent.Agent, referenceType request.TranscribeReferenceType, referenceID uuid.UUID, language string, direction tmtranscribe.Direction) (*tmtranscribe.WebhookMessage, error)
	TranscribeStop(ctx context.Context, a *amagent.Agent, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error)
	TranscribeDelete(ctx context.Context, a *amagent.Agent, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error)

	// transcript handlers
	TranscriptGets(ctx context.Context, a *amagent.Agent, transcribeID uuid.UUID) ([]*tmtranscript.WebhookMessage, error)

	// transfer handler
	TransferStart(ctx context.Context, a *amagent.Agent, transferType tmtransfer.Type, transfererCallID uuid.UUID, transfereeAddresses []commonaddress.Address) (*tmtransfer.WebhookMessage, error)

	// trunk
	TrunkCreate(ctx context.Context, a *amagent.Agent, name string, detail string, domainName string, authTypes []rmsipauth.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.WebhookMessage, error)
	TrunkDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmtrunk.WebhookMessage, error)
	TrunkGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmtrunk.WebhookMessage, error)
	TrunkGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmtrunk.WebhookMessage, error)
	TrunkUpdateBasicInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, name string, detail string, authTypes []rmsipauth.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.WebhookMessage, error)

	WebsockCreate(ctx context.Context, a *amagent.Agent, w http.ResponseWriter, r *http.Request) error
}

type serviceHandler struct {
	utilHandler    utilhandler.UtilHandler
	reqHandler     requesthandler.RequestHandler
	dbHandler      dbhandler.DBHandler
	websockHandler websockhandler.WebsockHandler

	// storage information
	storageClient *storage.Client
	projectID     string
	bucketName    string
}

// NewServiceHandler return ServiceHandler interface
func NewServiceHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
	websockHandler websockhandler.WebsockHandler,

	credentialPath string,
	projectID string,
	bucketName string,
) ServiceHandler {

	// init storage client
	ctx := context.Background()

	// create storageClient
	storageClient, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialPath))
	if err != nil {
		logrus.Errorf("Could not create a new client. err: %v", err)
		return nil
	}

	return &serviceHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		reqHandler:     reqHandler,
		dbHandler:      dbHandler,
		websockHandler: websockHandler,

		storageClient: storageClient,
		projectID:     projectID,
		bucketName:    bucketName,
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
