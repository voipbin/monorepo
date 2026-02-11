package servicehandler

//go:generate mockgen -package servicehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	multipart "mime/multipart"
	"net/http"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmrecording "monorepo/bin-call-manager/models/recording"
	ememail "monorepo/bin-email-manager/models/email"
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
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	ammessage "monorepo/bin-ai-manager/models/message"
	amsummary "monorepo/bin-ai-manager/models/summary"
	amtool "monorepo/bin-ai-manager/models/tool"

	cfconference "monorepo/bin-conference-manager/models/conference"
	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"

	cmcontact "monorepo/bin-contact-manager/models/contact"
	cmrequest "monorepo/bin-contact-manager/pkg/listenhandler/models/request"

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

	tmsipmessage "monorepo/bin-timeline-manager/models/sipmessage"

	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"
	tkchat "monorepo/bin-talk-manager/models/chat"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	rmrag "monorepo/bin-rag-manager/pkg/raghandler"

	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/websockhandler"

	"cloud.google.com/go/storage"
)

const (
	TokenExpiration = time.Hour * 24 * 7 // default token expiration time. 1 week(7 days)
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {

	// accesskeys
	AccesskeyCreate(ctx context.Context, a *amagent.Agent, name string, detail string, expire int32) (*csaccesskey.WebhookMessage, error)
	AccesskeyGet(ctx context.Context, a *amagent.Agent, accesskeyID uuid.UUID) (*csaccesskey.WebhookMessage, error)
	AccesskeyRawGetByToken(ctx context.Context, token string) (*csaccesskey.Accesskey, error)
	AccesskeyList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*csaccesskey.WebhookMessage, error)
	AccesskeyDelete(ctx context.Context, a *amagent.Agent, accesskeyID uuid.UUID) (*csaccesskey.WebhookMessage, error)
	AccesskeyUpdate(ctx context.Context, a *amagent.Agent, accesskeyID uuid.UUID, name string, detail string) (*csaccesskey.WebhookMessage, error)

	// activeflows
	ActiveflowCreate(ctx context.Context, a *amagent.Agent, activeflowID uuid.UUID, flowID uuid.UUID, actions []fmaction.Action) (*fmactiveflow.WebhookMessage, error)
	ActiveflowDelete(ctx context.Context, a *amagent.Agent, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error)
	ActiveflowGet(ctx context.Context, a *amagent.Agent, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error)
	ActiveflowList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*fmactiveflow.WebhookMessage, error)
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
	AgentList(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*amagent.WebhookMessage, error)
	AgentDelete(ctx context.Context, a *amagent.Agent, agentID uuid.UUID) (*amagent.WebhookMessage, error)
	AgentUpdate(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, name, detail string, ringMethod amagent.RingMethod) (*amagent.WebhookMessage, error)
	AgentUpdateAddresses(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, addresses []commonaddress.Address) (*amagent.WebhookMessage, error)
	AgentUpdatePassword(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, password string) (*amagent.WebhookMessage, error)
	AgentUpdatePermission(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, permission amagent.Permission) (*amagent.WebhookMessage, error)
	AgentUpdateStatus(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, status amagent.Status) (*amagent.WebhookMessage, error)
	AgentUpdateTagIDs(ctx context.Context, a *amagent.Agent, agentID uuid.UUID, tagIDs []uuid.UUID) (*amagent.WebhookMessage, error)

	// auth handlers
	AuthLogin(ctx context.Context, username, password string) (string, error)
	AuthJWTGenerate(data map[string]interface{}) (string, error)
	AuthJWTParse(ctx context.Context, tokenString string) (map[string]interface{}, error)
	AuthAccesskeyParse(ctx context.Context, accesskey string) (map[string]interface{}, error)
	AuthPasswordForgot(ctx context.Context, username string) error
	AuthPasswordReset(ctx context.Context, token string, password string) error

	// available numbers
	AvailableNumberList(ctx context.Context, a *amagent.Agent, size uint64, countryCode string, numType string) ([]*nmavailablenumber.WebhookMessage, error)

	// billing accounts
	BillingAccountGet(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID) (*bmaccount.WebhookMessage, error)
	BillingAccountAddBalanceForce(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, balance float32) (*bmaccount.WebhookMessage, error)
	BillingAccountSubtractBalanceForce(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, balance float32) (*bmaccount.WebhookMessage, error)
	BillingAccountUpdateBasicInfo(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, name string, detail string) (*bmaccount.WebhookMessage, error)
	BillingAccountUpdatePaymentInfo(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.WebhookMessage, error)

	// billings
	BillingList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*bmbilling.WebhookMessage, error)
	BillingGet(ctx context.Context, a *amagent.Agent, billingID uuid.UUID) (*bmbilling.WebhookMessage, error)

	// call handlers
	CallCreate(ctx context.Context, a *amagent.Agent, flowID uuid.UUID, actions []fmaction.Action, source *commonaddress.Address, destinations []commonaddress.Address) ([]*cmcall.WebhookMessage, []*cmgroupcall.WebhookMessage, error)
	CallGet(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cmcall.WebhookMessage, error)
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
	CallRecordingStart(
		ctx context.Context,
		a *amagent.Agent,
		callID uuid.UUID,
		format cmrecording.Format,
		endOfSilence int,
		endOfKey string,
		duration int,
		onEndFlowID uuid.UUID,
	) (*cmcall.WebhookMessage, error)
	CallRecordingStop(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error)
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
	CampaigncallList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error)
	CampaigncallGetsByCampaignID(ctx context.Context, a *amagent.Agent, campaignID uuid.UUID, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error)
	CampaigncallGet(ctx context.Context, a *amagent.Agent, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error)
	CampaigncallDelete(ctx context.Context, a *amagent.Agent, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error)

	// ai handlers
	AICreate(
		ctx context.Context,
		a *amagent.Agent,
		name string,
		detail string,
		engineType amai.EngineType,
		engineModel amai.EngineModel,
		engineData map[string]any,
		engineKey string,
		initPrompt string,
		ttsType amai.TTSType,
		ttsVoiceID string,
		sttType amai.STTType,
		toolNames []amtool.ToolName,
	) (*amai.WebhookMessage, error)
	AIGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*amai.WebhookMessage, error)
	AIGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amai.WebhookMessage, error)
	AIDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amai.WebhookMessage, error)
	AIUpdate(
		ctx context.Context,
		a *amagent.Agent,
		id uuid.UUID,
		name string,
		detail string,
		engineType amai.EngineType,
		engineModel amai.EngineModel,
		engineData map[string]any,
		engineKey string,
		initPrompt string,
		ttsType amai.TTSType,
		ttsVoiceID string,
		sttType amai.STTType,
		toolNames []amtool.ToolName,
	) (*amai.WebhookMessage, error)

	// aicall handlers
	AIcallCreate(
		ctx context.Context,
		a *amagent.Agent,
		aiID uuid.UUID,
		referenceType amaicall.ReferenceType,
		referenceID uuid.UUID,
		gender amaicall.Gender,
		language string,
	) (*amaicall.WebhookMessage, error)
	AIcallGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*amaicall.WebhookMessage, error)
	AIcallGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amaicall.WebhookMessage, error)
	AIcallDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amaicall.WebhookMessage, error)

	// aimessage handlers
	AImessageCreate(
		ctx context.Context,
		a *amagent.Agent,
		aicallID uuid.UUID,
		role ammessage.Role,
		content string,
	) (*ammessage.WebhookMessage, error)
	AImessageGetsByAIcallID(ctx context.Context, a *amagent.Agent, aicallID uuid.UUID, size uint64, token string) ([]*ammessage.WebhookMessage, error)
	AImessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*ammessage.WebhookMessage, error)
	AImessageDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*ammessage.WebhookMessage, error)

	// ai summary handlers
	AISummaryCreate(
		ctx context.Context,
		a *amagent.Agent,
		onEndFlowID uuid.UUID,
		referenceType amsummary.ReferenceType,
		referenceID uuid.UUID,
		language string,
	) (*amsummary.WebhookMessage, error)
	AISummaryGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*amsummary.WebhookMessage, error)
	AISummaryGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amsummary.WebhookMessage, error)
	AISummaryDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*amsummary.WebhookMessage, error)

	// conference handlers
	ConferenceCreate(
		ctx context.Context,
		a *amagent.Agent,
		conferenceID uuid.UUID,
		confType cfconference.Type,
		name string,
		detail string,
		data map[string]any,
		timeout int,
		preFlowID uuid.UUID,
		postFlowID uuid.UUID,
	) (*cfconference.WebhookMessage, error)
	ConferenceDelete(ctx context.Context, a *amagent.Agent, confID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cfconference.WebhookMessage, error)
	ConferenceMediaStreamStart(ctx context.Context, a *amagent.Agent, conferenceID uuid.UUID, encapsulation string, w http.ResponseWriter, r *http.Request) error
	ConferenceRecordingStart(
		ctx context.Context,
		a *amagent.Agent,
		conferenceID uuid.UUID,
		format cmrecording.Format,
		duration int,
		onEndFlowID uuid.UUID,
	) (*cfconference.WebhookMessage, error)
	ConferenceRecordingStop(ctx context.Context, a *amagent.Agent, confID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceTranscribeStart(ctx context.Context, a *amagent.Agent, conferenceID uuid.UUID, language string) (*cfconference.WebhookMessage, error)
	ConferenceTranscribeStop(ctx context.Context, a *amagent.Agent, conferenceID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceUpdate(
		ctx context.Context,
		a *amagent.Agent,
		conferenceID uuid.UUID,
		name string,
		detail string,
		data map[string]any,
		timeout int,
		preFlowID uuid.UUID,
		postFlowID uuid.UUID,
	) (*cfconference.WebhookMessage, error)

	// conferencecall handlers
	ConferencecallGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cfconferencecall.WebhookMessage, error)
	ConferencecallList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cfconferencecall.WebhookMessage, error)
	ConferencecallKick(ctx context.Context, a *amagent.Agent, conferencecallID uuid.UUID) (*cfconferencecall.WebhookMessage, error)

	// contact handlers
	ContactCreate(
		ctx context.Context,
		a *amagent.Agent,
		firstName string,
		lastName string,
		displayName string,
		company string,
		jobTitle string,
		source string,
		externalID string,
		notes string,
		phoneNumbers []cmrequest.PhoneNumberCreate,
		emails []cmrequest.EmailCreate,
		tagIDs []uuid.UUID,
	) (*cmcontact.WebhookMessage, error)
	ContactGet(ctx context.Context, a *amagent.Agent, contactID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactList(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*cmcontact.WebhookMessage, error)
	ContactUpdate(
		ctx context.Context,
		a *amagent.Agent,
		contactID uuid.UUID,
		firstName *string,
		lastName *string,
		displayName *string,
		company *string,
		jobTitle *string,
		externalID *string,
		notes *string,
	) (*cmcontact.WebhookMessage, error)
	ContactDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactLookup(ctx context.Context, a *amagent.Agent, phoneE164 string, email string) (*cmcontact.WebhookMessage, error)
	ContactPhoneNumberCreate(
		ctx context.Context,
		a *amagent.Agent,
		contactID uuid.UUID,
		number string,
		numberE164 string,
		phoneType string,
		isPrimary bool,
	) (*cmcontact.WebhookMessage, error)
	ContactPhoneNumberUpdate(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, phoneNumberID uuid.UUID, fields map[string]any) (*cmcontact.WebhookMessage, error)
	ContactPhoneNumberDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, phoneNumberID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactEmailCreate(
		ctx context.Context,
		a *amagent.Agent,
		contactID uuid.UUID,
		address string,
		emailType string,
		isPrimary bool,
	) (*cmcontact.WebhookMessage, error)
	ContactEmailUpdate(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, emailID uuid.UUID, fields map[string]any) (*cmcontact.WebhookMessage, error)
	ContactEmailDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, emailID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactTagAdd(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactTagRemove(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error)

	// conversation handlers
	ConversationGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cvconversation.WebhookMessage, error)
	ConversationGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cvconversation.WebhookMessage, error)
	ConversationUpdate(ctx context.Context, a *amagent.Agent, conversationID uuid.UUID, fields map[cvconversation.Field]any) (*cvconversation.WebhookMessage, error)
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
	ConversationAccountUpdate(ctx context.Context, a *amagent.Agent, accountID uuid.UUID, fields map[cvaccount.Field]any) (*cvaccount.WebhookMessage, error)
	ConversationAccountDelete(ctx context.Context, a *amagent.Agent, accountID uuid.UUID) (*cvaccount.WebhookMessage, error)

	// customer handlers
	CustomerCreate(
		ctx context.Context,
		a *amagent.Agent,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
	) (*cscustomer.WebhookMessage, error)
	CustomerGet(ctx context.Context, a *amagent.Agent, customerID uuid.UUID) (*cscustomer.WebhookMessage, error)
	CustomerList(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*cscustomer.WebhookMessage, error)
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
	CustomerSignup(
		ctx context.Context,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
	) (*cscustomer.WebhookMessage, error)
	CustomerEmailVerify(ctx context.Context, token string) (*cscustomer.WebhookMessage, error)

	// extension handlers
	ExtensionCreate(ctx context.Context, a *amagent.Agent, ext string, password string, name string, detail string) (*rmextension.WebhookMessage, error)
	ExtensionDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmextension.WebhookMessage, error)
	ExtensionGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmextension.WebhookMessage, error)
	ExtensionList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmextension.WebhookMessage, error)
	ExtensionUpdate(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail, password string) (*rmextension.WebhookMessage, error)

	// email handlers
	EmailSend(
		ctx context.Context,
		a *amagent.Agent,
		destinations []commonaddress.Address,
		subject string,
		content string,
		attachments []ememail.Attachment,
	) (*ememail.WebhookMessage, error)
	EmailList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*ememail.WebhookMessage, error)
	EmailGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*ememail.WebhookMessage, error)
	EmailDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*ememail.WebhookMessage, error)

	// flow handlers
	FlowCreate(
		ctx context.Context,
		a *amagent.Agent,
		name string,
		detail string,
		actions []fmaction.Action,
		onCompleteID uuid.UUID,
		persist bool,
	) (*fmflow.WebhookMessage, error)
	FlowDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowList(ctx context.Context, a *amagent.Agent, pageSize uint64, pageToken string) ([]*fmflow.WebhookMessage, error)
	FlowUpdate(
		ctx context.Context,
		a *amagent.Agent,
		id uuid.UUID,
		name string,
		detail string,
		actions []fmaction.Action,
		onCompleteID uuid.UUID,
	) (*fmflow.WebhookMessage, error)

	// grpupcall handlers
	GroupcallList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cmgroupcall.WebhookMessage, error)
	GroupcallGet(ctx context.Context, a *amagent.Agent, groupcallID uuid.UUID) (*cmgroupcall.WebhookMessage, error)
	GroupcallCreate(ctx context.Context, a *amagent.Agent, source commonaddress.Address, destinations []commonaddress.Address, flowID uuid.UUID, actions []fmaction.Action, ringMethod cmgroupcall.RingMethod, answerMethod cmgroupcall.AnswerMethod) (*cmgroupcall.WebhookMessage, error)
	GroupcallHangup(ctx context.Context, a *amagent.Agent, groupcallID uuid.UUID) (*cmgroupcall.WebhookMessage, error)
	GroupcallDelete(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmgroupcall.WebhookMessage, error)

	// message handlers
	MessageDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*mmmessage.WebhookMessage, error)
	MessageList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*mmmessage.WebhookMessage, error)
	MessageGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*mmmessage.WebhookMessage, error)
	MessageSend(ctx context.Context, a *amagent.Agent, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*mmmessage.WebhookMessage, error)

	// order numbers handler
	NumberCreate(ctx context.Context, a *amagent.Agent, num string, numType nmnumber.Type, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*nmnumber.WebhookMessage, error)
	NumberGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*nmnumber.WebhookMessage, error)
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
	ProviderList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmprovider.WebhookMessage, error)
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
	QueueList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*qmqueue.WebhookMessage, error)
	QueueCreate(
		ctx context.Context,
		a *amagent.Agent,
		name string,
		detail string,
		routingMethod qmqueue.RoutingMethod,
		tagIDs []uuid.UUID,
		waitFlowID uuid.UUID,
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
		waitFlowID uuid.UUID,
		timeoutWait int,
		timeoutService int,
	) (*qmqueue.WebhookMessage, error)
	QueueUpdateTagIDs(ctx context.Context, a *amagent.Agent, queueID uuid.UUID, tagIDs []uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueUpdateRoutingMethod(ctx context.Context, a *amagent.Agent, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) (*qmqueue.WebhookMessage, error)

	// queuecall handlers
	QueuecallGet(ctx context.Context, a *amagent.Agent, queueID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*qmqueuecall.WebhookMessage, error)
	QueuecallDelete(ctx context.Context, a *amagent.Agent, queuecallID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallKick(ctx context.Context, a *amagent.Agent, queuecallID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallKickByReferenceID(ctx context.Context, a *amagent.Agent, referenceID uuid.UUID) (*qmqueuecall.WebhookMessage, error)

	// recording handlers
	RecordingGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cmrecording.WebhookMessage, error)
	RecordingList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cmrecording.WebhookMessage, error)
	RecordingDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cmrecording.WebhookMessage, error)

	// recordingfile handlers
	RecordingfileGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (string, error)

	// route handlers
	RouteGet(ctx context.Context, a *amagent.Agent, routeID uuid.UUID) (*rmroute.WebhookMessage, error)
	RouteGetsByCustomerID(ctx context.Context, a *amagent.Agent, customerID uuid.UUID, size uint64, token string) ([]*rmroute.WebhookMessage, error)
	RouteList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmroute.WebhookMessage, error)
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

	// service_agent agent
	ServiceAgentAgentList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*amagent.WebhookMessage, error)
	ServiceAgentAgentGet(ctx context.Context, a *amagent.Agent, agentID uuid.UUID) (*amagent.WebhookMessage, error)

	// service_agent call
	ServiceAgentCallList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cmcall.WebhookMessage, error)
	ServiceAgentCallGet(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	ServiceAgentCallDelete(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*cmcall.WebhookMessage, error)

	// service_agent conversation
	ServiceAgentConversationGet(ctx context.Context, a *amagent.Agent, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error)
	ServiceAgentConversationList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cvconversation.WebhookMessage, error)

	// service_agent conversation message
	ServiceAgentConversationMessageList(ctx context.Context, a *amagent.Agent, conversationID uuid.UUID, size uint64, token string) ([]*cvmessage.WebhookMessage, error)
	ServiceAgentConversationMessageSend(
		ctx context.Context,
		a *amagent.Agent,
		conversationID uuid.UUID,
		text string,
		medias []cvmedia.Media,
	) (*cvmessage.WebhookMessage, error)

	// service_agent talk chat
	ServiceAgentTalkChatGet(ctx context.Context, a *amagent.Agent, chatID uuid.UUID) (*tkchat.WebhookMessage, error)
	ServiceAgentTalkChatList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tkchat.WebhookMessage, error)
	ServiceAgentTalkChatCreate(ctx context.Context, a *amagent.Agent, talkType tkchat.Type, name string, detail string, participants []tkparticipant.ParticipantInput) (*tkchat.WebhookMessage, error)
	ServiceAgentTalkChatUpdate(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, name *string, detail *string) (*tkchat.WebhookMessage, error)
	ServiceAgentTalkChatDelete(ctx context.Context, a *amagent.Agent, chatID uuid.UUID) (*tkchat.WebhookMessage, error)
	ServiceAgentTalkChatJoin(ctx context.Context, a *amagent.Agent, chatID uuid.UUID) (*tkparticipant.WebhookMessage, error)

	// service_agent talk channel (public channels for discovery)
	ServiceAgentTalkChannelList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tkchat.WebhookMessage, error)

	// service_agent talk participant
	ServiceAgentTalkParticipantList(ctx context.Context, a *amagent.Agent, chatID uuid.UUID) ([]*tkparticipant.WebhookMessage, error)
	ServiceAgentTalkParticipantCreate(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, ownerType string, ownerID uuid.UUID) (*tkparticipant.WebhookMessage, error)
	ServiceAgentTalkParticipantDelete(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, participantID uuid.UUID) (*tkparticipant.WebhookMessage, error)

	// service_agent talk message
	ServiceAgentTalkMessageGet(ctx context.Context, a *amagent.Agent, messageID uuid.UUID) (*tkmessage.WebhookMessage, error)
	ServiceAgentTalkMessageList(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, size uint64, token string) ([]*tkmessage.WebhookMessage, error)
	ServiceAgentTalkMessageCreate(ctx context.Context, a *amagent.Agent, chatID uuid.UUID, parentID *uuid.UUID, msgType tkmessage.Type, text string, medias []tkmessage.Media) (*tkmessage.WebhookMessage, error)
	ServiceAgentTalkMessageDelete(ctx context.Context, a *amagent.Agent, messageID uuid.UUID) (*tkmessage.WebhookMessage, error)
	ServiceAgentTalkMessageReactionCreate(ctx context.Context, a *amagent.Agent, messageID uuid.UUID, emoji string) (*tkmessage.WebhookMessage, error)

	// service_agent contact
	ServiceAgentContactCreate(
		ctx context.Context,
		a *amagent.Agent,
		firstName string,
		lastName string,
		displayName string,
		company string,
		jobTitle string,
		source string,
		externalID string,
		notes string,
		phoneNumbers []cmrequest.PhoneNumberCreate,
		emails []cmrequest.EmailCreate,
		tagIDs []uuid.UUID,
	) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactGet(ctx context.Context, a *amagent.Agent, contactID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactList(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*cmcontact.WebhookMessage, error)
	ServiceAgentContactUpdate(
		ctx context.Context,
		a *amagent.Agent,
		contactID uuid.UUID,
		firstName *string,
		lastName *string,
		displayName *string,
		company *string,
		jobTitle *string,
		externalID *string,
		notes *string,
	) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactLookup(ctx context.Context, a *amagent.Agent, phoneE164 string, email string) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactPhoneNumberCreate(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, number string, numberE164 string, phoneType string, isPrimary bool) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactPhoneNumberUpdate(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, phoneNumberID uuid.UUID, fields map[string]any) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactPhoneNumberDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, phoneNumberID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactEmailCreate(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, address string, emailType string, isPrimary bool) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactEmailUpdate(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, emailID uuid.UUID, fields map[string]any) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactEmailDelete(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, emailID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactTagAdd(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactTagRemove(ctx context.Context, a *amagent.Agent, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error)

	// service_agent tag
	ServiceAgentTagList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tmtag.WebhookMessage, error)
	ServiceAgentTagGet(ctx context.Context, a *amagent.Agent, tagID uuid.UUID) (*tmtag.WebhookMessage, error)

	// service_agent customer
	ServiceAgentCustomerGet(ctx context.Context, a *amagent.Agent) (*cscustomer.WebhookMessage, error)

	// service_agent extension
	ServiceAgentExtensionGet(ctx context.Context, a *amagent.Agent, extensionID uuid.UUID) (*rmextension.WebhookMessage, error)
	ServiceAgentExtensionList(ctx context.Context, a *amagent.Agent) ([]*rmextension.WebhookMessage, error)

	// storage file handlers
	ServiceAgentFileCreate(ctx context.Context, a *amagent.Agent, f multipart.File, name string, detail string, filename string) (*smfile.WebhookMessage, error)
	ServiceAgentFileDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*smfile.WebhookMessage, error)
	ServiceAgentFileGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*smfile.WebhookMessage, error)
	ServiceAgentFileList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*smfile.WebhookMessage, error)

	// service_agent me
	ServiceAgentMeGet(ctx context.Context, a *amagent.Agent) (*amagent.WebhookMessage, error)
	ServiceAgentMeUpdate(ctx context.Context, a *amagent.Agent, name string, detail string, ringMethod amagent.RingMethod) (*amagent.WebhookMessage, error)
	ServiceAgentMeUpdateAddresses(ctx context.Context, a *amagent.Agent, addresses []commonaddress.Address) (*amagent.WebhookMessage, error)
	ServiceAgentMeUpdateStatus(ctx context.Context, a *amagent.Agent, status amagent.Status) (*amagent.WebhookMessage, error)
	ServiceAgentMeUpdatePassword(ctx context.Context, a *amagent.Agent, password string) (*amagent.WebhookMessage, error)

	// storage account
	StorageAccountCreate(ctx context.Context, a *amagent.Agent, customerID uuid.UUID) (*smaccount.WebhookMessage, error)
	StorageAccountGet(ctx context.Context, a *amagent.Agent, storageAccountID uuid.UUID) (*smaccount.WebhookMessage, error)
	StorageAccountGetByCustomerID(ctx context.Context, a *amagent.Agent) (*smaccount.WebhookMessage, error)
	StorageAccountList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*smaccount.WebhookMessage, error)
	StorageAccountDelete(ctx context.Context, a *amagent.Agent, storageAccountID uuid.UUID) (*smaccount.WebhookMessage, error)

	// storage file
	StorageFileCreate(ctx context.Context, a *amagent.Agent, f multipart.File, name string, detail string, filename string) (*smfile.WebhookMessage, error)
	StorageFileGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*smfile.WebhookMessage, error)
	StorageFileList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*smfile.WebhookMessage, error)
	StorageFileDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*smfile.WebhookMessage, error)

	// tag handlers
	TagCreate(ctx context.Context, a *amagent.Agent, name string, detail string) (*tmtag.WebhookMessage, error)
	TagDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmtag.WebhookMessage, error)
	TagGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*tmtag.WebhookMessage, error)
	TagList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tmtag.WebhookMessage, error)
	TagUpdate(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail string) (*tmtag.WebhookMessage, error)

	// transcribe handlers
	TranscribeGet(ctx context.Context, a *amagent.Agent, routeID uuid.UUID) (*tmtranscribe.WebhookMessage, error)
	TranscribeList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tmtranscribe.WebhookMessage, error)
	TranscribeStart(
		ctx context.Context,
		a *amagent.Agent,
		referenceType string,
		referenceID uuid.UUID,
		language string,
		direction tmtranscribe.Direction,
		onEndFlowID uuid.UUID,
	) (*tmtranscribe.WebhookMessage, error)
	TranscribeStop(ctx context.Context, a *amagent.Agent, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error)
	TranscribeDelete(ctx context.Context, a *amagent.Agent, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error)

	// transcript handlers
	TranscriptList(ctx context.Context, a *amagent.Agent, transcribeID uuid.UUID) ([]*tmtranscript.WebhookMessage, error)

	// transfer handler
	TransferStart(ctx context.Context, a *amagent.Agent, transferType tmtransfer.Type, transfererCallID uuid.UUID, transfereeAddresses []commonaddress.Address) (*tmtransfer.WebhookMessage, error)

	// trunk
	TrunkCreate(ctx context.Context, a *amagent.Agent, name string, detail string, domainName string, authTypes []rmsipauth.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.WebhookMessage, error)
	TrunkDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmtrunk.WebhookMessage, error)
	TrunkGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmtrunk.WebhookMessage, error)
	TrunkList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmtrunk.WebhookMessage, error)
	TrunkUpdateBasicInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, name string, detail string, authTypes []rmsipauth.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.WebhookMessage, error)

	// timeline
	TimelineEventList(ctx context.Context, a *amagent.Agent, resourceType string, resourceID uuid.UUID, pageSize int, pageToken string) ([]*TimelineEvent, string, error)
	TimelineSIPAnalysisGet(ctx context.Context, a *amagent.Agent, callID uuid.UUID) (*tmsipmessage.SIPAnalysisResponse, error)
	TimelineSIPPcapGet(ctx context.Context, a *amagent.Agent, callID uuid.UUID) ([]byte, error)

	// rag handlers
	RagQuery(ctx context.Context, a *amagent.Agent, query string, docTypes []string, topK int) (*rmrag.QueryResponse, error)

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

	// etc
	jwtKey []byte
}

// NewServiceHandler return ServiceHandler interface
func NewServiceHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
	websockHandler websockhandler.WebsockHandler,

	projectID string,
	bucketName string,

	jwtKey string,
) ServiceHandler {
	log := logrus.WithField("func", "NewServiceHandler")

	// Create storage client using the decoded credentials
	storageClient, err := storage.NewClient(context.Background())
	if err != nil {
		log.Errorf("Could not create a new storage client. Please ensure the environment is configured for Application Default Credentials (ADC) (for example via GOOGLE_APPLICATION_CREDENTIALS, workload identity, or in-cluster metadata). error: %v", err)
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

		jwtKey: []byte(jwtKey),
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
