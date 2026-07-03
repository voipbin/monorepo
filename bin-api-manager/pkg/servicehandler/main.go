package servicehandler

//go:generate mockgen -package servicehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	multipart "mime/multipart"
	"net/http"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
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
	amaiaudit "monorepo/bin-ai-manager/models/aiaudit"
	amaiprompthistory "monorepo/bin-ai-manager/models/aiprompthistory"
	amaipromptproposal "monorepo/bin-ai-manager/models/aipromptproposal"
	ammessage "monorepo/bin-ai-manager/models/message"
	amparticipant "monorepo/bin-ai-manager/models/participant"
	amsummary "monorepo/bin-ai-manager/models/summary"
	amteam "monorepo/bin-ai-manager/models/team"
	amtool "monorepo/bin-ai-manager/models/tool"

	cfconference "monorepo/bin-conference-manager/models/conference"
	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"

	cmcontact "monorepo/bin-contact-manager/models/contact"
	cminteraction "monorepo/bin-contact-manager/models/interaction"
	cmresolution "monorepo/bin-contact-manager/models/resolution"
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
	rmprovidercall "monorepo/bin-route-manager/models/providercall"
	rmroute "monorepo/bin-route-manager/models/route"

	tmtag "monorepo/bin-tag-manager/models/tag"

	tmsipmessage "monorepo/bin-timeline-manager/models/sipmessage"
	tmanalysis "monorepo/bin-timeline-manager/models/analysis"

	tkchat "monorepo/bin-talk-manager/models/chat"
	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"

	rmrag "monorepo/bin-rag-manager/models/rag"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"

	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/websockhandler"

	"cloud.google.com/go/storage"
)

const (
	TokenExpiration    = time.Hour * 24 * 7 // default token expiration time. 1 week(7 days)
	BootExpiration     = time.Hour * 4      // direct boot token expiration time. 4 hours
	DelegateExpiration = time.Hour * 8      // delegate token expiration. 8 hours
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {

	// accesskeys
	AccesskeyCreate(ctx context.Context, a *auth.AuthIdentity, name string, detail string, expire int32) (*csaccesskey.WebhookMessage, error)
	AccesskeyGet(ctx context.Context, a *auth.AuthIdentity, accesskeyID uuid.UUID) (*csaccesskey.WebhookMessage, error)
	AccesskeyRawGetByToken(ctx context.Context, token string) (*csaccesskey.Accesskey, error)
	AccesskeyList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*csaccesskey.WebhookMessage, error)
	AccesskeyDelete(ctx context.Context, a *auth.AuthIdentity, accesskeyID uuid.UUID) (*csaccesskey.WebhookMessage, error)
	AccesskeyUpdate(ctx context.Context, a *auth.AuthIdentity, accesskeyID uuid.UUID, name string, detail string) (*csaccesskey.WebhookMessage, error)

	// activeflows
	ActiveflowCreate(ctx context.Context, a *auth.AuthIdentity, activeflowID uuid.UUID, flowID uuid.UUID, actions []fmaction.Action, variables map[string]string, webhookURI string, webhookMethod fmactiveflow.WebhookMethod) (*fmactiveflow.WebhookMessage, error)
	ActiveflowDelete(ctx context.Context, a *auth.AuthIdentity, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error)
	ActiveflowGet(ctx context.Context, a *auth.AuthIdentity, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error)
	ActiveflowList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*fmactiveflow.WebhookMessage, error)
	ActiveflowStop(ctx context.Context, a *auth.AuthIdentity, activeflowID uuid.UUID) (*fmactiveflow.WebhookMessage, error)

	// agent handlers
	AgentCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		username string,
		password string,
		name string,
		detail string,
		ringMethod amagent.RingMethod,
		permission amagent.Permission,
		tagIDs []uuid.UUID,
		addresses []commonaddress.Address,
	) (*amagent.WebhookMessage, error)
	AgentGet(ctx context.Context, a *auth.AuthIdentity, agentID uuid.UUID) (*amagent.WebhookMessage, error)
	AgentList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, filters map[string]string) ([]*amagent.WebhookMessage, error)
	AgentDelete(ctx context.Context, a *auth.AuthIdentity, agentID uuid.UUID) (*amagent.WebhookMessage, error)
	AgentUpdate(ctx context.Context, a *auth.AuthIdentity, agentID uuid.UUID, name, detail string, ringMethod amagent.RingMethod) (*amagent.WebhookMessage, error)
	AgentUpdateAddresses(ctx context.Context, a *auth.AuthIdentity, agentID uuid.UUID, addresses []commonaddress.Address) (*amagent.WebhookMessage, error)
	AgentUpdatePassword(ctx context.Context, a *auth.AuthIdentity, agentID uuid.UUID, password string) (*amagent.WebhookMessage, error)
	AgentUpdatePermission(ctx context.Context, a *auth.AuthIdentity, agentID uuid.UUID, permission amagent.Permission) (*amagent.WebhookMessage, error)
	AgentUpdateStatus(ctx context.Context, a *auth.AuthIdentity, agentID uuid.UUID, status amagent.Status) (*amagent.WebhookMessage, error)
	AgentUpdateTagIDs(ctx context.Context, a *auth.AuthIdentity, agentID uuid.UUID, tagIDs []uuid.UUID) (*amagent.WebhookMessage, error)
	AgentDirectHashRegenerate(ctx context.Context, a *auth.AuthIdentity, agentID uuid.UUID) (*amagent.WebhookMessage, error)

	// auth handlers
	AuthLogin(ctx context.Context, username, password string) (string, error)
	AuthJWTGenerate(data map[string]interface{}) (string, error)
	AuthJWTParse(ctx context.Context, tokenString string) (map[string]interface{}, error)
	AuthPasswordForgot(ctx context.Context, username string) error
	AuthPasswordReset(ctx context.Context, token string, password string) error
	AuthBoot(ctx context.Context, directHash string) (*BootResponse, error)
	AuthDelegate(ctx context.Context, a *auth.AuthIdentity, targetCustomerID uuid.UUID, reason string) (*DelegateResponse, error)

	// available numbers
	AvailableNumberList(ctx context.Context, a *auth.AuthIdentity, size uint64, countryCode string, numType string) ([]*nmavailablenumber.WebhookMessage, error)

	// billing accounts
	BillingAccountGet(ctx context.Context, a *auth.AuthIdentity, billingAccountID uuid.UUID) (*bmaccount.Account, error)
	BillingAccountAddBalanceForce(ctx context.Context, a *auth.AuthIdentity, billingAccountID uuid.UUID, balance int64) (*bmaccount.Account, error)
	BillingAccountSubtractBalanceForce(ctx context.Context, a *auth.AuthIdentity, billingAccountID uuid.UUID, balance int64) (*bmaccount.Account, error)
	BillingAccountUpdateBasicInfo(ctx context.Context, a *auth.AuthIdentity, billingAccountID uuid.UUID, name string, detail string) (*bmaccount.Account, error)
	BillingAccountUpdatePaymentInfo(ctx context.Context, a *auth.AuthIdentity, billingAccountID uuid.UUID, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.Account, error)
	BillingAccountSelfGet(ctx context.Context, a *auth.AuthIdentity) (*bmaccount.WebhookMessage, error)
	BillingAccountSelfUpdateBasicInfo(ctx context.Context, a *auth.AuthIdentity, name string, detail string) (*bmaccount.WebhookMessage, error)
	BillingAccountSelfUpdatePaymentInfo(ctx context.Context, a *auth.AuthIdentity, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.WebhookMessage, error)
	BillingAccountSelfCreatePaddlePortalSession(ctx context.Context, a *auth.AuthIdentity) (string, error)
	BillingAccountList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, filters map[string]string) ([]*bmaccount.Account, error)

	// billings
	BillingList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*bmbilling.WebhookMessage, error)
	BillingGet(ctx context.Context, a *auth.AuthIdentity, billingID uuid.UUID) (*bmbilling.WebhookMessage, error)

	// call handlers
	CallCreate(ctx context.Context, a *auth.AuthIdentity, flowID uuid.UUID, actions []fmaction.Action, source *commonaddress.Address, destinations []commonaddress.Address, anonymous string, variables map[string]string) ([]*cmcall.WebhookMessage, []*cmgroupcall.WebhookMessage, error)
	CallGet(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cmcall.WebhookMessage, error)
	CallDelete(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallHangup(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallTalk(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID, text string, language string, provider string, voiceID string) error
	CallHoldOn(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) error
	CallHoldOff(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) error
	CallMediaStreamStart(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID, encapsulation string, w http.ResponseWriter, r *http.Request) error
	CallMuteOn(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID, direction cmcall.MuteDirection) error
	CallMuteOff(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID, direction cmcall.MuteDirection) error
	CallMOHOn(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) error
	CallMOHOff(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) error
	CallRecordingStart(
		ctx context.Context,
		a *auth.AuthIdentity,
		callID uuid.UUID,
		format cmrecording.Format,
		endOfSilence int,
		endOfKey string,
		duration int,
		onEndFlowID uuid.UUID,
	) (*cmcall.WebhookMessage, error)
	CallRecordingStop(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	CallSilenceOn(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) error
	CallSilenceOff(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) error

	// outbound config handlers
	OutboundConfigCreate(ctx context.Context, a *auth.AuthIdentity, req *cmoutboundconfig.UpdateRequest) (*cmoutboundconfig.WebhookMessage, error)
	OutboundConfigDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmoutboundconfig.WebhookMessage, error)
	OutboundConfigGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmoutboundconfig.WebhookMessage, error)
	OutboundConfigList(ctx context.Context, a *auth.AuthIdentity, pageSize uint64, pageToken string) ([]*cmoutboundconfig.WebhookMessage, error)
	OutboundConfigUpdate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, req *cmoutboundconfig.UpdateRequest) (*cmoutboundconfig.WebhookMessage, error)
	OutboundConfigSelfGet(ctx context.Context, a *auth.AuthIdentity) (*cmoutboundconfig.WebhookMessage, error)
	OutboundConfigSelfUpdate(ctx context.Context, a *auth.AuthIdentity, req *cmoutboundconfig.UpdateRequest) (*cmoutboundconfig.WebhookMessage, error)

	// campaign handlers
	CampaignCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
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
	CampaignGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cacampaign.WebhookMessage, error)
	CampaignGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cacampaign.WebhookMessage, error)
	CampaignDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cacampaign.WebhookMessage, error)
	CampaignUpdateBasicInfo(
		ctx context.Context,
		a *auth.AuthIdentity,
		id uuid.UUID,
		name string,
		detail string,
		campaignType cacampaign.Type,
		serviceLevel int,
		endHandle cacampaign.EndHandle,
	) (*cacampaign.WebhookMessage, error)
	CampaignUpdateStatus(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, status cacampaign.Status) (*cacampaign.WebhookMessage, error)
	CampaignUpdateServiceLevel(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, serviceLevel int) (*cacampaign.WebhookMessage, error)
	CampaignUpdateActions(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, actions []fmaction.Action) (*cacampaign.WebhookMessage, error)
	CampaignUpdateResourceInfo(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, outplanID uuid.UUID, outdialID uuid.UUID, queueID uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.WebhookMessage, error)
	CampaignUpdateNextCampaignID(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.WebhookMessage, error)

	// campaigncall handlers
	CampaigncallList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error)
	CampaigncallGetsByCampaignID(ctx context.Context, a *auth.AuthIdentity, campaignID uuid.UUID, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error)
	CampaigncallGet(ctx context.Context, a *auth.AuthIdentity, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error)
	CampaigncallDelete(ctx context.Context, a *auth.AuthIdentity, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error)

	// ai handlers
	AICreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		name string,
		detail string,
		engineModel amai.EngineModel,
		parameter map[string]any,
		engineKey string,
		ragID uuid.UUID,
		initPrompt string,
		ttsType amai.TTSType,
		ttsVoiceID string,
		sttType amai.STTType,
		sttLanguage string,
		toolNames []amtool.ToolName,
		autoAICallAuditEnabled bool,
	) (*amai.WebhookMessage, error)
	AIGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*amai.WebhookMessage, error)
	AIGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amai.WebhookMessage, error)
	AIDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amai.WebhookMessage, error)
	AIUpdate(
		ctx context.Context,
		a *auth.AuthIdentity,
		id uuid.UUID,
		name string,
		detail string,
		engineModel amai.EngineModel,
		parameter map[string]any,
		engineKey string,
		ragID uuid.UUID,
		initPrompt string,
		ttsType amai.TTSType,
		ttsVoiceID string,
		sttType amai.STTType,
		sttLanguage string,
		toolNames []amtool.ToolName,
		autoAICallAuditEnabled bool,
	) (*amai.WebhookMessage, error)
	AIDirectHashRegenerate(ctx context.Context, a *auth.AuthIdentity, aiID uuid.UUID) (*amai.WebhookMessage, error)

	// ai prompt history handlers
	AIPromptHistoryGetsByAIID(ctx context.Context, a *auth.AuthIdentity, aiID uuid.UUID, size uint64, token string) ([]*amaiprompthistory.AIPromptHistory, error)
	AIPromptHistoryGet(ctx context.Context, a *auth.AuthIdentity, aiID uuid.UUID, historyID uuid.UUID) (*amaiprompthistory.AIPromptHistory, error)

	// team handlers
	TeamCreate(ctx context.Context, a *auth.AuthIdentity, name string, detail string, startMemberID uuid.UUID, members []amteam.Member, parameter map[string]any) (*amteam.WebhookMessage, error)
	TeamGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*amteam.WebhookMessage, error)
	TeamGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amteam.WebhookMessage, error)
	TeamDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amteam.WebhookMessage, error)
	TeamUpdate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, name string, detail string, startMemberID uuid.UUID, members []amteam.Member, parameter map[string]any) (*amteam.WebhookMessage, error)
	TeamDirectHashRegenerate(ctx context.Context, a *auth.AuthIdentity, teamID uuid.UUID) (*amteam.WebhookMessage, error)

	// aicall handlers
	AIcallCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		assistanceType amaicall.AssistanceType,
		assistanceID uuid.UUID,
		referenceType amaicall.ReferenceType,
		referenceID uuid.UUID,
	) (*amaicall.WebhookMessage, error)
	AIcallGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*amaicall.WebhookMessage, error)
	AIcallGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error)
	AIcallDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error)
	AIcallTerminate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error)

	// aicall participant handlers
	AIcallParticipantGets(ctx context.Context, a *auth.AuthIdentity, aicallID uuid.UUID, pageToken string, pageSize uint64) ([]*amparticipant.WebhookMessage, error)
	AIParticipantGets(ctx context.Context, a *auth.AuthIdentity, aiID uuid.UUID, pageToken string, pageSize uint64) ([]*amparticipant.WebhookMessage, error)

	// aimessage handlers
	AImessageCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		aicallID uuid.UUID,
		role ammessage.Role,
		content string,
	) (*ammessage.WebhookMessage, error)
	AImessageGetsByAIcallID(ctx context.Context, a *auth.AuthIdentity, aicallID uuid.UUID, size uint64, token string) ([]*ammessage.WebhookMessage, error)
	AImessageGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*ammessage.WebhookMessage, error)
	AImessageDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*ammessage.WebhookMessage, error)

	// ai summary handlers
	AISummaryCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		onEndFlowID uuid.UUID,
		referenceType amsummary.ReferenceType,
		referenceID uuid.UUID,
		language string,
	) (*amsummary.WebhookMessage, error)
	AISummaryGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*amsummary.WebhookMessage, error)
	AISummaryGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amsummary.WebhookMessage, error)
	AISummaryDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amsummary.WebhookMessage, error)

	// ai audit handlers
	AIAuditCreate(ctx context.Context, a *auth.AuthIdentity, aicallID uuid.UUID, language string) ([]*amaiaudit.WebhookMessage, error)
	AIAuditGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, aicallID, aiID uuid.UUID) ([]*amaiaudit.WebhookMessage, error)
	AIAuditGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaiaudit.WebhookMessage, error)
	AIAuditDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaiaudit.WebhookMessage, error)

	// ai prompt proposal handlers
	AIPromptProposalCreate(ctx context.Context, a *auth.AuthIdentity, aiID uuid.UUID, auditIDs []uuid.UUID, language string) (*amaipromptproposal.WebhookMessage, error)
	AIPromptProposalGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, aiID uuid.UUID, status amaipromptproposal.Status) ([]*amaipromptproposal.WebhookMessage, error)
	AIPromptProposalGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaipromptproposal.WebhookMessage, error)
	AIPromptProposalAccept(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaipromptproposal.WebhookMessage, error)
	AIPromptProposalReject(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaipromptproposal.WebhookMessage, error)
	AIPromptProposalDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*amaipromptproposal.WebhookMessage, error)

	// conference handlers
	ConferenceCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		conferenceID uuid.UUID,
		confType cfconference.Type,
		name string,
		detail string,
		data map[string]any,
		timeout int,
		preFlowID uuid.UUID,
		postFlowID uuid.UUID,
	) (*cfconference.WebhookMessage, error)
	ConferenceDelete(ctx context.Context, a *auth.AuthIdentity, confID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cfconference.WebhookMessage, error)
	ConferenceMediaStreamStart(ctx context.Context, a *auth.AuthIdentity, conferenceID uuid.UUID, encapsulation string, w http.ResponseWriter, r *http.Request) error
	ConferenceRecordingStart(
		ctx context.Context,
		a *auth.AuthIdentity,
		conferenceID uuid.UUID,
		format cmrecording.Format,
		duration int,
		onEndFlowID uuid.UUID,
	) (*cfconference.WebhookMessage, error)
	ConferenceRecordingStop(ctx context.Context, a *auth.AuthIdentity, confID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceTranscribeStart(ctx context.Context, a *auth.AuthIdentity, conferenceID uuid.UUID, language string) (*cfconference.WebhookMessage, error)
	ConferenceTranscribeStop(ctx context.Context, a *auth.AuthIdentity, conferenceID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceDirectHashRegenerate(ctx context.Context, a *auth.AuthIdentity, conferenceID uuid.UUID) (*cfconference.WebhookMessage, error)
	ConferenceUpdate(
		ctx context.Context,
		a *auth.AuthIdentity,
		conferenceID uuid.UUID,
		name string,
		detail string,
		data map[string]any,
		timeout int,
		preFlowID uuid.UUID,
		postFlowID uuid.UUID,
	) (*cfconference.WebhookMessage, error)

	// conferencecall handlers
	ConferencecallGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cfconferencecall.WebhookMessage, error)
	ConferencecallList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cfconferencecall.WebhookMessage, error)
	ConferencecallKick(ctx context.Context, a *auth.AuthIdentity, conferencecallID uuid.UUID) (*cfconferencecall.WebhookMessage, error)

	// contact handlers
	ContactCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		firstName string,
		lastName string,
		displayName string,
		company string,
		jobTitle string,
		source string,
		externalID string,
		notes string,
		addresses []cmrequest.AddressCreate,
		tagIDs []uuid.UUID,
	) (*cmcontact.WebhookMessage, error)
	ContactGet(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, filters map[string]string) ([]*cmcontact.WebhookMessage, error)
	ContactUpdate(
		ctx context.Context,
		a *auth.AuthIdentity,
		contactID uuid.UUID,
		firstName *string,
		lastName *string,
		displayName *string,
		company *string,
		jobTitle *string,
		externalID *string,
		notes *string,
	) (*cmcontact.WebhookMessage, error)
	ContactDelete(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactLookup(ctx context.Context, a *auth.AuthIdentity, phoneE164 string, email string) (*cmcontact.WebhookMessage, error)
	ContactAddressCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		contactID uuid.UUID,
		addrType string,
		target string,
		isPrimary bool,
		name string,
		detail string,
	) (*cmcontact.WebhookMessage, error)
	ContactAddressUpdate(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, addressID uuid.UUID, fields map[string]any) (*cmcontact.WebhookMessage, error)
	ContactAddressDelete(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, addressID uuid.UUID) (*cmcontact.WebhookMessage, error)

	// contact_addresses (independent resource)
	ContactAddressList(ctx context.Context, a *auth.AuthIdentity, filters map[string]any, pageToken string, pageSize uint64) ([]cmcontact.Address, error)
	ContactAddressGet(ctx context.Context, a *auth.AuthIdentity, addressID uuid.UUID) (*cmcontact.Address, error)
	ContactAddressCreateIndependent(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, addrType string, target string, isPrimary bool, name string, detail string) (*cmcontact.Address, error)
	ContactAddressUpdateIndependent(ctx context.Context, a *auth.AuthIdentity, addressID uuid.UUID, fields map[string]any) (*cmcontact.Address, error)
	ContactAddressDeleteIndependent(ctx context.Context, a *auth.AuthIdentity, addressID uuid.UUID) (*cmcontact.Address, error)
	ContactAddressClaim(ctx context.Context, a *auth.AuthIdentity, addressID uuid.UUID, contactID uuid.UUID) (*cmcontact.Address, error)

	ContactTagAdd(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ContactTagRemove(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error)

	// interaction handlers
	InteractionList(
		ctx context.Context,
		a *auth.AuthIdentity,
		size uint64,
		token string,
		peerType, peerTarget string,
		contactID, addressID uuid.UUID,
	) ([]*cminteraction.Interaction, string, error)
	InteractionListUnresolved(
		ctx context.Context,
		a *auth.AuthIdentity,
		size uint64,
		token string,
		since string,
	) ([]*cminteraction.Interaction, string, error)
	InteractionGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cminteraction.Interaction, error)
	ResolutionCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		interactionID uuid.UUID,
		contactID uuid.UUID,
		resolutionType string,
		resolvedByType string,
		resolvedByID uuid.UUID,
	) (*cmresolution.Resolution, error)
	ResolutionDelete(ctx context.Context, a *auth.AuthIdentity, interactionID, resolutionID uuid.UUID) error

	// service agent interaction handlers
	ServiceAgentInteractionList(
		ctx context.Context,
		a *auth.AuthIdentity,
		size uint64,
		token string,
		peerType, peerTarget string,
		contactID, addressID uuid.UUID,
	) ([]*cminteraction.Interaction, string, error)
	ServiceAgentInteractionListUnresolved(
		ctx context.Context,
		a *auth.AuthIdentity,
		size uint64,
		token string,
		since string,
	) ([]*cminteraction.Interaction, string, error)
	ServiceAgentInteractionGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cminteraction.Interaction, error)
	ServiceAgentResolutionCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		interactionID uuid.UUID,
		contactID uuid.UUID,
		resolutionType string,
		resolvedByType string,
		resolvedByID uuid.UUID,
	) (*cmresolution.Resolution, error)
	ServiceAgentResolutionDelete(ctx context.Context, a *auth.AuthIdentity, interactionID, resolutionID uuid.UUID) error

	// conversation handlers
	ConversationGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cvconversation.WebhookMessage, error)
	ConversationGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, ownerID uuid.UUID) ([]*cvconversation.WebhookMessage, error)
	ConversationUpdate(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID, fields map[cvconversation.Field]any) (*cvconversation.WebhookMessage, error)
	ConversationUnassign(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error)
	ConversationMessageGetsByConversationID(
		ctx context.Context,
		a *auth.AuthIdentity,
		conversationID uuid.UUID,
		size uint64,
		token string,
	) ([]*cvmessage.WebhookMessage, error)
	ConversationMessageSend(
		ctx context.Context,
		a *auth.AuthIdentity,
		conversationID uuid.UUID,
		text string,
		medias []cvmedia.Media,
	) (*cvmessage.WebhookMessage, error)

	ConversationAccountGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cvaccount.WebhookMessage, error)
	ConversationAccountGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cvaccount.WebhookMessage, error)
	ConversationAccountCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		accountType cvaccount.Type,
		name string,
		detail string,
		secret string,
		token string,
		messageFlowID uuid.UUID,
		providerData json.RawMessage,
	) (*cvaccount.WebhookMessage, error)
	ConversationAccountUpdate(ctx context.Context, a *auth.AuthIdentity, accountID uuid.UUID, fields map[cvaccount.Field]any) (*cvaccount.WebhookMessage, error)
	ConversationAccountDelete(ctx context.Context, a *auth.AuthIdentity, accountID uuid.UUID) (*cvaccount.WebhookMessage, error)

	// customer handlers
	CustomerCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
	) (*cscustomer.Customer, error)
	CustomerGet(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID) (*cscustomer.Customer, error)
	CustomerSelfGet(ctx context.Context, a *auth.AuthIdentity) (*cscustomer.WebhookMessage, error)
	CustomerList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, filters map[string]string) ([]*cscustomer.Customer, error)
	CustomerUpdate(
		ctx context.Context,
		a *auth.AuthIdentity,
		id uuid.UUID,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
	) (*cscustomer.Customer, error)
	CustomerSelfUpdate(
		ctx context.Context,
		a *auth.AuthIdentity,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
	) (*cscustomer.WebhookMessage, error)
	CustomerDelete(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID) (*cscustomer.Customer, error)
	CustomerFreeze(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID) (*cscustomer.Customer, error)
	CustomerRecover(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID) (*cscustomer.Customer, error)
	CustomerSelfFreeze(ctx context.Context, a *auth.AuthIdentity) (*cscustomer.WebhookMessage, error)
	CustomerSelfFreezeAndDelete(ctx context.Context, a *auth.AuthIdentity) (*cscustomer.WebhookMessage, error)
	CustomerSelfRecover(ctx context.Context, a *auth.AuthIdentity) (*cscustomer.WebhookMessage, error)
	CustomerUpdateBillingAccountID(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID, billingAccountID uuid.UUID) (*cscustomer.Customer, error)
	CustomerUpdateMetadata(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID, metadata cscustomer.Metadata) (*cscustomer.Customer, error)
	CustomerSelfUpdateBillingAccountID(ctx context.Context, a *auth.AuthIdentity, billingAccountID uuid.UUID) (*cscustomer.WebhookMessage, error)
	CustomerSelfUpdateMetadata(ctx context.Context, a *auth.AuthIdentity, metadata cscustomer.Metadata) (*cscustomer.WebhookMessage, error)
	CustomerSignup(
		ctx context.Context,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
		clientIP string,
	) (*cscustomer.SignupResultWebhookMessage, error)
	CustomerEmailVerify(ctx context.Context, token string) (*cscustomer.EmailVerifyResultWebhookMessage, error)

	// extension handlers
	ExtensionCreate(ctx context.Context, a *auth.AuthIdentity, ext string, password string, name string, detail string) (*rmextension.WebhookMessage, error)
	ExtensionDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmextension.WebhookMessage, error)
	ExtensionGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmextension.WebhookMessage, error)
	ExtensionList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*rmextension.WebhookMessage, error)
	ExtensionUpdate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, name, detail, password string) (*rmextension.WebhookMessage, error)
	ExtensionDirectHashRegenerate(ctx context.Context, a *auth.AuthIdentity, extensionID uuid.UUID) (*rmextension.WebhookMessage, error)

	// email handlers
	EmailSend(
		ctx context.Context,
		a *auth.AuthIdentity,
		destinations []commonaddress.Address,
		subject string,
		content string,
		attachments []ememail.Attachment,
	) (*ememail.WebhookMessage, error)
	EmailList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*ememail.WebhookMessage, error)
	EmailGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*ememail.WebhookMessage, error)
	EmailDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*ememail.WebhookMessage, error)

	// flow handlers
	FlowCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		name string,
		detail string,
		actions []fmaction.Action,
		onCompleteID uuid.UUID,
		persist bool,
	) (*fmflow.WebhookMessage, error)
	FlowDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowDirectHashRegenerate(ctx context.Context, a *auth.AuthIdentity, flowID uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*fmflow.WebhookMessage, error)
	FlowList(ctx context.Context, a *auth.AuthIdentity, pageSize uint64, pageToken string) ([]*fmflow.WebhookMessage, error)
	FlowUpdate(
		ctx context.Context,
		a *auth.AuthIdentity,
		id uuid.UUID,
		name string,
		detail string,
		actions []fmaction.Action,
		onCompleteID uuid.UUID,
	) (*fmflow.WebhookMessage, error)

	// grpupcall handlers
	GroupcallList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cmgroupcall.WebhookMessage, error)
	GroupcallGet(ctx context.Context, a *auth.AuthIdentity, groupcallID uuid.UUID) (*cmgroupcall.WebhookMessage, error)
	GroupcallCreate(ctx context.Context, a *auth.AuthIdentity, source commonaddress.Address, destinations []commonaddress.Address, flowID uuid.UUID, actions []fmaction.Action, ringMethod cmgroupcall.RingMethod, answerMethod cmgroupcall.AnswerMethod) (*cmgroupcall.WebhookMessage, error)
	GroupcallHangup(ctx context.Context, a *auth.AuthIdentity, groupcallID uuid.UUID) (*cmgroupcall.WebhookMessage, error)
	GroupcallDelete(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*cmgroupcall.WebhookMessage, error)

	// message handlers
	MessageDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*mmmessage.WebhookMessage, error)
	MessageList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*mmmessage.WebhookMessage, error)
	MessageGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*mmmessage.WebhookMessage, error)
	MessageSend(ctx context.Context, a *auth.AuthIdentity, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*mmmessage.WebhookMessage, error)

	// order numbers handler
	NumberCreate(ctx context.Context, a *auth.AuthIdentity, num string, numType nmnumber.Type, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*nmnumber.WebhookMessage, error)
	NumberGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*nmnumber.WebhookMessage, error)
	NumberDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberUpdate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*nmnumber.WebhookMessage, error)
	NumberUpdateFlowIDs(ctx context.Context, a *auth.AuthIdentity, id, callFlowID uuid.UUID, messageFlowID uuid.UUID) (*nmnumber.WebhookMessage, error)
	NumberUpdateMetadata(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, metadata nmnumber.Metadata) (*nmnumber.WebhookMessage, error)
	NumberRenew(ctx context.Context, a *auth.AuthIdentity, tmRenew string) ([]*nmnumber.Number, error)

	// outdials
	OutdialCreate(ctx context.Context, a *auth.AuthIdentity, campaignID uuid.UUID, name, detail, data string) (*omoutdial.WebhookMessage, error)
	OutdialGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*omoutdial.WebhookMessage, error)
	OutdialDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*omoutdial.WebhookMessage, error)
	OutdialGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*omoutdial.WebhookMessage, error)
	OutdialtargetGetsByOutdialID(ctx context.Context, a *auth.AuthIdentity, outdialID uuid.UUID, size uint64, token string) ([]*omoutdialtarget.WebhookMessage, error)
	OutdialUpdateBasicInfo(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, name, detail string) (*omoutdial.WebhookMessage, error)
	OutdialUpdateCampaignID(ctx context.Context, a *auth.AuthIdentity, id, campaignID uuid.UUID) (*omoutdial.WebhookMessage, error)
	OutdialUpdateData(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, data string) (*omoutdial.WebhookMessage, error)

	// outdialtargets
	OutdialtargetCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
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
	OutdialtargetGet(ctx context.Context, a *auth.AuthIdentity, outdialID uuid.UUID, outdialtargetID uuid.UUID) (*omoutdialtarget.WebhookMessage, error)
	OutdialtargetDelete(ctx context.Context, a *auth.AuthIdentity, outdialID uuid.UUID, outdialtargetID uuid.UUID) (*omoutdialtarget.WebhookMessage, error)

	OutplanCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
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
	OutplanDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*caoutplan.WebhookMessage, error)
	OutplanGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*caoutplan.WebhookMessage, error)
	OutplanGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*caoutplan.WebhookMessage, error)
	OutplanUpdateBasicInfo(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, name, detail string) (*caoutplan.WebhookMessage, error)
	OutplanUpdateDialInfo(
		ctx context.Context,
		a *auth.AuthIdentity,
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
		a *auth.AuthIdentity,
		providerType rmprovider.Type,
		hostname string,
		techPrefix string,
		techPostfix string,
		techHeaders map[string]string,
		name string,
		detail string,
		codecs string,
	) (*rmprovider.WebhookMessage, error)
	ProviderDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmprovider.WebhookMessage, error)
	ProviderGet(ctx context.Context, a *auth.AuthIdentity, providerID uuid.UUID) (*rmprovider.WebhookMessage, error)
	ProviderList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*rmprovider.WebhookMessage, error)
	ProviderSetup(ctx context.Context, a *auth.AuthIdentity, carrier string, name string, detail string, apiKey string) (*rmprovider.WebhookMessage, error)
	ProviderUpdate(
		ctx context.Context,
		a *auth.AuthIdentity,
		providerID uuid.UUID,
		providerType rmprovider.Type,
		hostname string,
		techPrefix string,
		techPostfix string,
		techHeaders map[string]string,
		name string,
		detail string,
		codecs string,
	) (*rmprovider.WebhookMessage, error)

	// providercall handlers
	ProviderCallCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		providerID uuid.UUID,
		flowID uuid.UUID,
		actions []fmaction.Action,
		source *commonaddress.Address,
		destinations []commonaddress.Address,
		anonymous string,
	) (*rmprovidercall.WebhookMessage, error)
	ProviderCallGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmprovidercall.WebhookMessage, error)
	ProviderCallGets(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, providerID uuid.UUID) ([]*rmprovidercall.WebhookMessage, error)
	ProviderCallDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmprovidercall.WebhookMessage, error)

	// queue handlers
	QueueGet(ctx context.Context, a *auth.AuthIdentity, queueID uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*qmqueue.WebhookMessage, error)
	QueueCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		name string,
		detail string,
		routingMethod qmqueue.RoutingMethod,
		tagIDs []uuid.UUID,
		waitFlowID uuid.UUID,
		timeoutWait int,
		timeoutService int,
	) (*qmqueue.WebhookMessage, error)
	QueueDelete(ctx context.Context, a *auth.AuthIdentity, queueID uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueUpdate(
		ctx context.Context,
		a *auth.AuthIdentity,
		queueID uuid.UUID,
		name string,
		detail string,
		routingMethod qmqueue.RoutingMethod,
		tagIDs []uuid.UUID,
		waitFlowID uuid.UUID,
		timeoutWait int,
		timeoutService int,
	) (*qmqueue.WebhookMessage, error)
	QueueUpdateTagIDs(ctx context.Context, a *auth.AuthIdentity, queueID uuid.UUID, tagIDs []uuid.UUID) (*qmqueue.WebhookMessage, error)
	QueueUpdateRoutingMethod(ctx context.Context, a *auth.AuthIdentity, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) (*qmqueue.WebhookMessage, error)
	QueueDirectHashRegenerate(ctx context.Context, a *auth.AuthIdentity, queueID uuid.UUID) (*qmqueue.WebhookMessage, error)

	// queuecall handlers
	QueuecallGet(ctx context.Context, a *auth.AuthIdentity, queueID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*qmqueuecall.WebhookMessage, error)
	QueuecallDelete(ctx context.Context, a *auth.AuthIdentity, queuecallID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallKick(ctx context.Context, a *auth.AuthIdentity, queuecallID uuid.UUID) (*qmqueuecall.WebhookMessage, error)
	QueuecallKickByReferenceID(ctx context.Context, a *auth.AuthIdentity, referenceID uuid.UUID) (*qmqueuecall.WebhookMessage, error)

	// recording handlers
	RecordingGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmrecording.WebhookMessage, error)
	RecordingList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cmrecording.WebhookMessage, error)
	RecordingDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cmrecording.WebhookMessage, error)

	// recordingfile handlers
	RecordingfileGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (string, error)

	// route handlers
	RouteGet(ctx context.Context, a *auth.AuthIdentity, routeID uuid.UUID) (*rmroute.Route, error)
	RouteGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID, size uint64, token string) ([]*rmroute.Route, error)
	RouteList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*rmroute.Route, error)
	RouteCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		customerID uuid.UUID,
		name string,
		detail string,
		providerID uuid.UUID,
		priority int,
		target string,
	) (*rmroute.Route, error)
	RouteDelete(ctx context.Context, a *auth.AuthIdentity, routeID uuid.UUID) (*rmroute.Route, error)
	RouteUpdate(
		ctx context.Context,
		a *auth.AuthIdentity,
		routeID uuid.UUID,
		name string,
		detail string,
		providerID uuid.UUID,
		priority int,
		target string,
	) (*rmroute.Route, error)

	// service_agent agent
	ServiceAgentAgentList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*amagent.WebhookMessage, error)
	ServiceAgentAgentGet(ctx context.Context, a *auth.AuthIdentity, agentID uuid.UUID) (*amagent.WebhookMessage, error)

	// service_agent call
	ServiceAgentCallList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cmcall.WebhookMessage, error)
	ServiceAgentCallGet(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*cmcall.WebhookMessage, error)
	ServiceAgentCallDelete(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*cmcall.WebhookMessage, error)

	// service_agent conversation
	ServiceAgentConversationGet(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error)
	ServiceAgentConversationList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cvconversation.WebhookMessage, error)
	ServiceAgentConversationUpdate(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID, fields map[cvconversation.Field]any) (*cvconversation.WebhookMessage, error)
	ServiceAgentConversationUnassign(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID) (*cvconversation.WebhookMessage, error)

	// service_agent conversation message
	ServiceAgentConversationMessageList(ctx context.Context, a *auth.AuthIdentity, conversationID uuid.UUID, size uint64, token string) ([]*cvmessage.WebhookMessage, error)
	ServiceAgentConversationMessageSend(
		ctx context.Context,
		a *auth.AuthIdentity,
		conversationID uuid.UUID,
		text string,
		medias []cvmedia.Media,
	) (*cvmessage.WebhookMessage, error)

	// service_agent talk chat
	ServiceAgentTalkChatGet(ctx context.Context, a *auth.AuthIdentity, chatID uuid.UUID) (*tkchat.WebhookMessage, error)
	ServiceAgentTalkChatList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*tkchat.WebhookMessage, error)
	ServiceAgentTalkChatCreate(ctx context.Context, a *auth.AuthIdentity, talkType tkchat.Type, name string, detail string, participants []tkparticipant.ParticipantInput) (*tkchat.WebhookMessage, error)
	ServiceAgentTalkChatUpdate(ctx context.Context, a *auth.AuthIdentity, chatID uuid.UUID, name *string, detail *string) (*tkchat.WebhookMessage, error)
	ServiceAgentTalkChatDelete(ctx context.Context, a *auth.AuthIdentity, chatID uuid.UUID) (*tkchat.WebhookMessage, error)
	ServiceAgentTalkChatJoin(ctx context.Context, a *auth.AuthIdentity, chatID uuid.UUID) (*tkparticipant.WebhookMessage, error)

	// service_agent talk channel (public channels for discovery)
	ServiceAgentTalkChannelList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*tkchat.WebhookMessage, error)

	// service_agent talk participant
	ServiceAgentTalkParticipantList(ctx context.Context, a *auth.AuthIdentity, chatID uuid.UUID) ([]*tkparticipant.WebhookMessage, error)
	ServiceAgentTalkParticipantCreate(ctx context.Context, a *auth.AuthIdentity, chatID uuid.UUID, ownerType string, ownerID uuid.UUID) (*tkparticipant.WebhookMessage, error)
	ServiceAgentTalkParticipantDelete(ctx context.Context, a *auth.AuthIdentity, chatID uuid.UUID, participantID uuid.UUID) (*tkparticipant.WebhookMessage, error)

	// service_agent talk message
	ServiceAgentTalkMessageGet(ctx context.Context, a *auth.AuthIdentity, messageID uuid.UUID) (*tkmessage.WebhookMessage, error)
	ServiceAgentTalkMessageList(ctx context.Context, a *auth.AuthIdentity, chatID uuid.UUID, size uint64, token string) ([]*tkmessage.WebhookMessage, error)
	ServiceAgentTalkMessageCreate(ctx context.Context, a *auth.AuthIdentity, chatID uuid.UUID, parentID *uuid.UUID, msgType tkmessage.Type, text string, medias []tkmessage.Media) (*tkmessage.WebhookMessage, error)
	ServiceAgentTalkMessageDelete(ctx context.Context, a *auth.AuthIdentity, messageID uuid.UUID) (*tkmessage.WebhookMessage, error)
	ServiceAgentTalkMessageReactionCreate(ctx context.Context, a *auth.AuthIdentity, messageID uuid.UUID, emoji string) (*tkmessage.WebhookMessage, error)

	// service_agent contact
	ServiceAgentContactCreate(
		ctx context.Context,
		a *auth.AuthIdentity,
		firstName string,
		lastName string,
		displayName string,
		company string,
		jobTitle string,
		source string,
		externalID string,
		notes string,
		addresses []cmrequest.AddressCreate,
		tagIDs []uuid.UUID,
	) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactGet(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, filters map[string]string) ([]*cmcontact.WebhookMessage, error)
	ServiceAgentContactUpdate(
		ctx context.Context,
		a *auth.AuthIdentity,
		contactID uuid.UUID,
		firstName *string,
		lastName *string,
		displayName *string,
		company *string,
		jobTitle *string,
		externalID *string,
		notes *string,
	) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactDelete(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactLookup(ctx context.Context, a *auth.AuthIdentity, phoneE164 string, email string) (*cmcontact.WebhookMessage, error)

	// service agent transcribe handlers
	ServiceAgentTranscribeList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, referenceType string, referenceID uuid.UUID) ([]*tmtranscribe.WebhookMessage, error)
	ServiceAgentTranscribeStart(
		ctx context.Context,
		a *auth.AuthIdentity,
		referenceType string,
		referenceID uuid.UUID,
		language string,
		direction tmtranscribe.Direction,
		onEndFlowID uuid.UUID,
		provider tmtranscribe.Provider,
	) (*tmtranscribe.WebhookMessage, error)
	ServiceAgentTranscriptList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, transcribeID uuid.UUID) ([]*tmtranscript.WebhookMessage, error)
	ServiceAgentContactAddressCreate(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, addrType string, target string, isPrimary bool, name string, detail string) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactAddressUpdate(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, addressID uuid.UUID, fields map[string]any) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactAddressDelete(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, addressID uuid.UUID) (*cmcontact.WebhookMessage, error)

	// service_agents/contact_addresses (independent resource)
	ServiceAgentContactAddressList(ctx context.Context, a *auth.AuthIdentity, filters map[string]any, pageToken string, pageSize uint64) ([]cmcontact.Address, error)
	ServiceAgentContactAddressGet(ctx context.Context, a *auth.AuthIdentity, addressID uuid.UUID) (*cmcontact.Address, error)
	ServiceAgentContactAddressCreateIndependent(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, addrType string, target string, isPrimary bool, name string, detail string) (*cmcontact.Address, error)
	ServiceAgentContactAddressUpdateIndependent(ctx context.Context, a *auth.AuthIdentity, addressID uuid.UUID, fields map[string]any) (*cmcontact.Address, error)
	ServiceAgentContactAddressDeleteIndependent(ctx context.Context, a *auth.AuthIdentity, addressID uuid.UUID) (*cmcontact.Address, error)
	ServiceAgentContactAddressClaim(ctx context.Context, a *auth.AuthIdentity, addressID uuid.UUID, contactID uuid.UUID) (*cmcontact.Address, error)

	ServiceAgentContactTagAdd(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error)
	ServiceAgentContactTagRemove(ctx context.Context, a *auth.AuthIdentity, contactID uuid.UUID, tagID uuid.UUID) (*cmcontact.WebhookMessage, error)

	// service_agent tag
	ServiceAgentTagList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*tmtag.WebhookMessage, error)
	ServiceAgentTagGet(ctx context.Context, a *auth.AuthIdentity, tagID uuid.UUID) (*tmtag.WebhookMessage, error)

	// service_agent customer
	ServiceAgentCustomerGet(ctx context.Context, a *auth.AuthIdentity) (*cscustomer.WebhookMessage, error)

	// service_agent extension
	ServiceAgentExtensionGet(ctx context.Context, a *auth.AuthIdentity, extensionID uuid.UUID) (*rmextension.WebhookMessage, error)
	ServiceAgentExtensionList(ctx context.Context, a *auth.AuthIdentity) ([]*rmextension.WebhookMessage, error)

	// storage file handlers
	ServiceAgentFileCreate(ctx context.Context, a *auth.AuthIdentity, f multipart.File, fileType smfile.Type, name string, detail string, filename string) (*smfile.WebhookMessage, error)
	ServiceAgentFileDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*smfile.WebhookMessage, error)
	ServiceAgentFileGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*smfile.WebhookMessage, error)
	ServiceAgentFileList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*smfile.WebhookMessage, error)
	ServiceAgentFileDownloadRedirect(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (string, error)

	// service_agent me
	ServiceAgentMeGet(ctx context.Context, a *auth.AuthIdentity) (*amagent.WebhookMessage, error)
	ServiceAgentMeUpdate(ctx context.Context, a *auth.AuthIdentity, name string, detail string, ringMethod amagent.RingMethod) (*amagent.WebhookMessage, error)
	ServiceAgentMeUpdateAddresses(ctx context.Context, a *auth.AuthIdentity, addresses []commonaddress.Address) (*amagent.WebhookMessage, error)
	ServiceAgentMeUpdateStatus(ctx context.Context, a *auth.AuthIdentity, status amagent.Status) (*amagent.WebhookMessage, error)
	ServiceAgentMeUpdatePassword(ctx context.Context, a *auth.AuthIdentity, password string) (*amagent.WebhookMessage, error)

	// storage account
	StorageAccountCreate(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID) (*smaccount.Account, error)
	StorageAccountGet(ctx context.Context, a *auth.AuthIdentity, storageAccountID uuid.UUID) (*smaccount.WebhookMessage, error)
	StorageAccountGetByCustomerID(ctx context.Context, a *auth.AuthIdentity) (*smaccount.WebhookMessage, error)
	StorageAccountList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*smaccount.Account, error)
	StorageAccountDelete(ctx context.Context, a *auth.AuthIdentity, storageAccountID uuid.UUID) (*smaccount.Account, error)

	// storage file
	StorageFileCreate(ctx context.Context, a *auth.AuthIdentity, f multipart.File, fileType smfile.Type, name string, detail string, filename string) (*smfile.WebhookMessage, error)
	StorageFileGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*smfile.WebhookMessage, error)
	StorageFileList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*smfile.WebhookMessage, error)
	StorageFileDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*smfile.WebhookMessage, error)
	StorageFileDownloadRedirect(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (string, error)

	// tag handlers
	TagCreate(ctx context.Context, a *auth.AuthIdentity, name string, detail string) (*tmtag.WebhookMessage, error)
	TagDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*tmtag.WebhookMessage, error)
	TagGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*tmtag.WebhookMessage, error)
	TagList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*tmtag.WebhookMessage, error)
	TagUpdate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, name, detail string) (*tmtag.WebhookMessage, error)

	// transcribe handlers
	TranscribeGet(ctx context.Context, a *auth.AuthIdentity, routeID uuid.UUID) (*tmtranscribe.WebhookMessage, error)
	TranscribeList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, referenceType string, referenceID uuid.UUID) ([]*tmtranscribe.WebhookMessage, error)
	TranscribeStart(
		ctx context.Context,
		a *auth.AuthIdentity,
		referenceType string,
		referenceID uuid.UUID,
		language string,
		direction tmtranscribe.Direction,
		onEndFlowID uuid.UUID,
		provider tmtranscribe.Provider,
	) (*tmtranscribe.WebhookMessage, error)
	TranscribeStop(ctx context.Context, a *auth.AuthIdentity, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error)
	TranscribeDelete(ctx context.Context, a *auth.AuthIdentity, transcribeID uuid.UUID) (*tmtranscribe.WebhookMessage, error)

	// speaking handlers
	SpeakingCreate(ctx context.Context, a *auth.AuthIdentity, referenceType tmstreaming.ReferenceType, referenceID uuid.UUID, language string, provider string, voiceID string, direction tmstreaming.Direction) (*tmspeaking.WebhookMessage, error)
	SpeakingGet(ctx context.Context, a *auth.AuthIdentity, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error)
	SpeakingList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*tmspeaking.WebhookMessage, error)
	SpeakingSay(ctx context.Context, a *auth.AuthIdentity, speakingID uuid.UUID, text string) (*tmspeaking.WebhookMessage, error)
	SpeakingFlush(ctx context.Context, a *auth.AuthIdentity, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error)
	SpeakingStop(ctx context.Context, a *auth.AuthIdentity, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error)
	SpeakingDelete(ctx context.Context, a *auth.AuthIdentity, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error)

	// transcript handlers
	TranscriptList(ctx context.Context, a *auth.AuthIdentity, transcribeID uuid.UUID) ([]*tmtranscript.WebhookMessage, error)

	// transfer handler
	TransferStart(ctx context.Context, a *auth.AuthIdentity, transferType tmtransfer.Type, transfererCallID uuid.UUID, transfereeAddresses []commonaddress.Address) (*tmtransfer.WebhookMessage, error)

	// trunk
	TrunkCreate(ctx context.Context, a *auth.AuthIdentity, name string, detail string, domainName string, authTypes []rmsipauth.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.WebhookMessage, error)
	TrunkDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmtrunk.WebhookMessage, error)
	TrunkGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmtrunk.WebhookMessage, error)
	TrunkList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*rmtrunk.WebhookMessage, error)
	TrunkUpdateBasicInfo(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, name string, detail string, authTypes []rmsipauth.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.WebhookMessage, error)

	// timeline
	AggregatedEventList(ctx context.Context, a *auth.AuthIdentity, activeflowID uuid.UUID, callID uuid.UUID, pageSize int, pageToken string) ([]*TimelineEvent, string, error)
	TimelineEventList(ctx context.Context, a *auth.AuthIdentity, resourceType string, resourceID uuid.UUID, pageSize int, pageToken string) ([]*TimelineEvent, string, error)
	TimelineSIPAnalysisGet(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*tmsipmessage.SIPAnalysisResponse, error)
	TimelineSIPPcapGet(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) ([]byte, error)

	// timeline analysis (AI analysis of ended activeflows)
	TimelineAnalysisCreate(ctx context.Context, a *auth.AuthIdentity, activeflowID uuid.UUID, reanalyze bool) (*tmanalysis.WebhookMessage, error)
	TimelineAnalysisGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*tmanalysis.WebhookMessage, error)
	TimelineAnalysisGetsByCustomerID(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, activeflowID uuid.UUID, status tmanalysis.Status) ([]*tmanalysis.WebhookMessage, error)
	TimelineAnalysisDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*tmanalysis.WebhookMessage, error)

	WebsockCreate(ctx context.Context, a *auth.AuthIdentity, w http.ResponseWriter, r *http.Request) error

	// RAG
	RagCreate(ctx context.Context, a *auth.AuthIdentity, name, description string, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.WebhookMessage, error)
	RagGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmrag.WebhookMessage, error)
	RagGets(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*rmrag.WebhookMessage, error)
	RagUpdate(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID, fields map[rmrag.Field]any) (*rmrag.WebhookMessage, error)
	RagDelete(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*rmrag.WebhookMessage, error)
	RagAddSources(ctx context.Context, a *auth.AuthIdentity, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.WebhookMessage, error)
	RagRemoveSource(ctx context.Context, a *auth.AuthIdentity, ragID, sourceID uuid.UUID) (*rmrag.WebhookMessage, error)
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
