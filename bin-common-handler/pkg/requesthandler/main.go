package requesthandler

//go:generate mockgen -package requesthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"

	cmari "monorepo/bin-call-manager/models/ari"
	cmbridge "monorepo/bin-call-manager/models/bridge"
	cmcall "monorepo/bin-call-manager/models/call"
	cmchannel "monorepo/bin-call-manager/models/channel"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmrecording "monorepo/bin-call-manager/models/recording"

	bmaccount "monorepo/bin-billing-manager/models/account"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	cacampaign "monorepo/bin-campaign-manager/models/campaign"
	cacampaigncall "monorepo/bin-campaign-manager/models/campaigncall"
	caoutplan "monorepo/bin-campaign-manager/models/outplan"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	chatchat "monorepo/bin-chat-manager/models/chat"
	chatchatroom "monorepo/bin-chat-manager/models/chatroom"
	chatmedia "monorepo/bin-chat-manager/models/media"
	chatmessagechat "monorepo/bin-chat-manager/models/messagechat"
	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"
	cbchatbot "monorepo/bin-chatbot-manager/models/chatbot"
	cbchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"
	cbservice "monorepo/bin-chatbot-manager/models/service"

	cfconference "monorepo/bin-conference-manager/models/conference"
	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"
	cfservice "monorepo/bin-conference-manager/models/service"

	cvaccount "monorepo/bin-conversation-manager/models/account"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmflow "monorepo/bin-flow-manager/models/flow"
	fmvariable "monorepo/bin-flow-manager/models/variable"

	hmhook "monorepo/bin-hook-manager/models/hook"
	mmmessage "monorepo/bin-message-manager/models/message"
	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"
	nmnumber "monorepo/bin-number-manager/models/number"

	omoutdial "monorepo/bin-outdial-manager/models/outdial"
	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"
	qmqueue "monorepo/bin-queue-manager/models/queue"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"
	qmservice "monorepo/bin-queue-manager/models/service"

	rmastcontact "monorepo/bin-registrar-manager/models/astcontact"
	rmextension "monorepo/bin-registrar-manager/models/extension"
	rmsipauth "monorepo/bin-registrar-manager/models/sipauth"
	rmtrunk "monorepo/bin-registrar-manager/models/trunk"

	rmprovider "monorepo/bin-route-manager/models/provider"
	rmroute "monorepo/bin-route-manager/models/route"

	smaccount "monorepo/bin-storage-manager/models/account"
	smbucketfile "monorepo/bin-storage-manager/models/bucketfile"
	smcompressfile "monorepo/bin-storage-manager/models/compressfile"
	smfile "monorepo/bin-storage-manager/models/file"

	tmtag "monorepo/bin-tag-manager/models/tag"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	tmtts "monorepo/bin-tts-manager/models/tts"

	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	amagent "monorepo/bin-agent-manager/models/agent"

	uuid "github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// contents type
var (
	ContentTypeNone = ""
	ContentTypeText = "text/plain"
	ContentTypeJSON = "application/json"
)

// group asterisk id
var (
	AsteriskIDCall       = "call"       // asterisk-call
	AsteriskIDConference = "conference" // asterisk-conference
)

const requestTimeoutDefault int = 3000 // default request timeout(3 sec)

// delay units
const (
	DelayNow    int = 0
	DelaySecond int = 1000
	DelayMinute int = DelaySecond * 60
	DelayHour   int = DelayMinute * 60
)

// default stasis application name.
// normally, we don't need to use this, because proxy will set this automatically.
// but, some of Asterisk ARI required application name. this is for that.
const defaultAstStasisApp = "voipbin"

// list of prometheus metrics
var (
	promRequestProcessTime *prometheus.HistogramVec
	promEventCount         *prometheus.CounterVec
)

func initPrometheus(namespace string) {

	promRequestProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "request_process_time",
			Help:      "Process time of send/receiv requests",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"target", "resource", "method"},
	)

	promEventCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "event_publish_total",
			Help:      "Total number of published event with event_type.",
		},
		[]string{"event_type"},
	)

	prometheus.MustRegister(
		promRequestProcessTime,
		promEventCount,
	)
}

// RequestHandler intreface for ARI request handler
type RequestHandler interface {

	// send
	SendRequest(ctx context.Context, queue commonoutline.QueueName, uri string, method sock.RequestMethod, timeout int, delay int, dataType string, data json.RawMessage) (*sock.Response, error)

	// asterisk AMI
	AstAMIRedirect(ctx context.Context, asteriskID, channelID, context, exten, priority string) error

	// asterisk bridges
	AstBridgeAddChannel(ctx context.Context, asteriskID, bridgeID, channelID, role string, absorbDTMF, mute bool) error
	AstBridgeCreate(ctx context.Context, asteriskID, bridgeID, bridgeName string, bridgeType []cmbridge.Type) error
	AstBridgeDelete(ctx context.Context, asteriskID, bridgeID string) error
	AstBridgeGet(ctx context.Context, asteriskID, bridgeID string) (*cmbridge.Bridge, error)
	AstBridgeRemoveChannel(ctx context.Context, asteriskID, bridgeID, channelID string) error
	AstBridgeRecord(ctx context.Context, asteriskID string, bridgeID string, filename string, format string, duration int, silence int, beep bool, endKey string, ifExists string) error

	// asterisk channels
	AstChannelAnswer(ctx context.Context, asteriskID, channelID string) error
	AstChannelContinue(ctx context.Context, asteriskID, channelID, context, ext string, pri int, label string) error
	AstChannelCreate(ctx context.Context, asteriskID, channelID, appArgs, endpoint, otherChannelID, originator, formats string, variables map[string]string) (*cmchannel.Channel, error)
	AstChannelCreateSnoop(ctx context.Context, asteriskID, channelID, snoopID, appArgs string, spy, whisper cmchannel.SnoopDirection) (*cmchannel.Channel, error)
	AstChannelDial(ctx context.Context, asteriskID, channelID, caller string, timeout int) error
	AstChannelDTMF(ctx context.Context, asteriskID, channelID string, digit string, duration, before, between, after int) error
	AstChannelExternalMedia(ctx context.Context, asteriskID string, channelID string, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string, data string, variables map[string]string) (*cmchannel.Channel, error)
	AstChannelGet(ctx context.Context, asteriskID, channelID string) (*cmchannel.Channel, error)
	AstChannelHangup(ctx context.Context, asteriskID, channelID string, code cmari.ChannelCause, delay int) error
	AstChannelHoldOn(ctx context.Context, asteriskID string, channelID string) error
	AstChannelHoldOff(ctx context.Context, asteriskID string, channelID string) error
	AstChannelMusicOnHoldOn(ctx context.Context, asteriskID string, channelID string) error
	AstChannelMusicOnHoldOff(ctx context.Context, asteriskID string, channelID string) error
	AstChannelMuteOn(ctx context.Context, asteriskID string, channelID string, direction string) error
	AstChannelMuteOff(ctx context.Context, asteriskID string, channelID string, direction string) error
	AstChannelPlay(ctx context.Context, asteriskID string, channelID string, actionID uuid.UUID, medias []string, lang string) error
	AstChannelRecord(ctx context.Context, asteriskID string, channelID string, filename string, format string, duration int, silence int, beep bool, endKey string, ifExists string) error
	AstChannelRing(ctx context.Context, asteriskID string, channelID string) error
	AstChannelSilenceOn(ctx context.Context, asteriskID string, channelID string) error
	AstChannelSilenceOff(ctx context.Context, asteriskID string, channelID string) error
	AstChannelVariableGet(ctx context.Context, asteriskID, channelID, variable string) (string, error)
	AstChannelVariableSet(ctx context.Context, asteriskID, channelID, variable, value string) error

	// asterisk playbacks
	AstPlaybackStop(ctx context.Context, asteriskID string, playabckID string) error

	// asterisk recordings
	AstRecordingStop(ctx context.Context, asteriskID, recordingName string) error
	AstRecordingPause(ctx context.Context, asteriskID, recordingName string) error
	AstRecordingUnpause(ctx context.Context, asteriskID, recordingName string) error
	AstRecordingMute(ctx context.Context, asteriskID, recordingName string) error
	AstRecordingUnmute(ctx context.Context, asteriskID, recordingName string) error

	// agent-manager agent
	AgentV1AgentCreate(
		ctx context.Context,
		timeout int,
		customerID uuid.UUID,
		username string,
		password string,
		name string,
		detail string,
		ringMethod amagent.RingMethod,
		permission amagent.Permission,
		tagIDs []uuid.UUID,
		addresses []commonaddress.Address,
	) (*amagent.Agent, error)
	AgentV1AgentGet(ctx context.Context, agentID uuid.UUID) (*amagent.Agent, error)
	AgentV1AgentGetByCustomerIDAndAddress(ctx context.Context, timeout int, customerID uuid.UUID, addr commonaddress.Address) (*amagent.Agent, error)
	AgentV1AgentGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]amagent.Agent, error)
	AgentV1AgentDelete(ctx context.Context, id uuid.UUID) (*amagent.Agent, error)
	AgentV1AgentUpdate(ctx context.Context, id uuid.UUID, name, detail string, ringMethod amagent.RingMethod) (*amagent.Agent, error)
	AgentV1AgentUpdateAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) (*amagent.Agent, error)
	AgentV1AgentUpdatePassword(ctx context.Context, timeout int, id uuid.UUID, password string) (*amagent.Agent, error)
	AgentV1AgentUpdatePermission(ctx context.Context, id uuid.UUID, permission amagent.Permission) (*amagent.Agent, error)
	AgentV1AgentUpdateStatus(ctx context.Context, id uuid.UUID, status amagent.Status) (*amagent.Agent, error)
	AgentV1AgentUpdateTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) (*amagent.Agent, error)

	// agent-manager login
	AgentV1Login(ctx context.Context, timeout int, username string, password string) (*amagent.Agent, error)

	// billing-manager account
	BillingV1AccountGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]bmaccount.Account, error)
	BillingV1AccountGet(ctx context.Context, accountID uuid.UUID) (*bmaccount.Account, error)
	BillingV1AccountCreate(ctx context.Context, custoerID uuid.UUID, name string, detail string, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.Account, error)
	BillingV1AccountDelete(ctx context.Context, accountID uuid.UUID) (*bmaccount.Account, error)
	BillingV1AccountAddBalanceForce(ctx context.Context, accountID uuid.UUID, balance float32) (*bmaccount.Account, error)
	BillingV1AccountSubtractBalanceForce(ctx context.Context, accountID uuid.UUID, balance float32) (*bmaccount.Account, error)
	BillingV1AccountIsValidBalance(ctx context.Context, accountID uuid.UUID, billingType bmbilling.ReferenceType, country string, count int) (bool, error)
	BillingV1AccountUpdateBasicInfo(ctx context.Context, accountID uuid.UUID, name string, detail string) (*bmaccount.Account, error)
	BillingV1AccountUpdatePaymentInfo(ctx context.Context, accountID uuid.UUID, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.Account, error)

	// billing-manager billing
	BillingV1BillingGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]bmbilling.Billing, error)

	// call-manager event
	CallPublishEvent(ctx context.Context, eventType string, publisher string, dataType string, data []byte) error

	// call-manager call
	CallV1CallHealth(ctx context.Context, id uuid.UUID, delay, retryCount int) error
	CallV1CallAddChainedCall(ctx context.Context, callID uuid.UUID, chainedCallID uuid.UUID) (*cmcall.Call, error)
	CallV1CallRemoveChainedCall(ctx context.Context, callID uuid.UUID, chainedCallID uuid.UUID) (*cmcall.Call, error)
	CallV1CallExternalMediaStart(ctx context.Context, callID uuid.UUID, externalMediaID uuid.UUID, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string) (*cmcall.Call, error)
	CallV1CallExternalMediaStop(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error)
	CallV1CallActionNext(ctx context.Context, callID uuid.UUID, force bool) error
	CallV1CallActionTimeout(ctx context.Context, id uuid.UUID, delay int, a *fmaction.Action) error
	CallV1CallsCreate(
		ctx context.Context,
		customerID uuid.UUID,
		flowID uuid.UUID,
		masterCallID uuid.UUID,
		source *commonaddress.Address,
		destinations []commonaddress.Address,
		ealryExecution bool,
		connect bool,
	) ([]*cmcall.Call, []*cmgroupcall.Groupcall, error)
	CallV1CallCreateWithID(
		ctx context.Context,
		id uuid.UUID,
		customerID uuid.UUID,
		flowID uuid.UUID,
		activeflowID uuid.UUID,
		masterCallID uuid.UUID,
		source *commonaddress.Address,
		destination *commonaddress.Address,
		groupcallID uuid.UUID,
		ealryExecution bool,
		connect bool,
	) (*cmcall.Call, error)
	CallV1CallDelete(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error)
	CallV1CallGet(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error)
	CallV1CallGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cmcall.Call, error)
	CallV1CallGetDigits(ctx context.Context, callID uuid.UUID) (string, error)
	CallV1CallMediaStop(ctx context.Context, callID uuid.UUID) error
	CallV1CallPlay(ctx context.Context, callID uuid.UUID, mediaURLs []string) error
	CallV1CallRecordingStart(ctx context.Context, callID uuid.UUID, format cmrecording.Format, endOfSilence int, endOfKey string, duration int) (*cmcall.Call, error)
	CallV1CallRecordingStop(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error)
	CallV1CallSendDigits(ctx context.Context, callID uuid.UUID, digits string) error
	CallV1CallTalk(ctx context.Context, callID uuid.UUID, text string, gender string, language string, rqeuestTimeout int) error
	CallV1CallUpdateConfbridgeID(ctx context.Context, callID uuid.UUID, confbirdgeID uuid.UUID) (*cmcall.Call, error)
	CallV1CallHangup(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error)
	CallV1CallHoldOn(ctx context.Context, callID uuid.UUID) error
	CallV1CallHoldOff(ctx context.Context, callID uuid.UUID) error
	CallV1CallMuteOn(ctx context.Context, callID uuid.UUID, direction cmcall.MuteDirection) error
	CallV1CallMuteOff(ctx context.Context, callID uuid.UUID, direction cmcall.MuteDirection) error
	CallV1CallMusicOnHoldOn(ctx context.Context, callID uuid.UUID) error
	CallV1CallMusicOnHoldOff(ctx context.Context, callID uuid.UUID) error
	CallV1CallSilenceOn(ctx context.Context, callID uuid.UUID) error
	CallV1CallSilenceOff(ctx context.Context, callID uuid.UUID) error

	// call-manager channel
	CallV1ChannelHealth(ctx context.Context, channelID string, delay, retryCount int) error

	// call-manager confbridge
	CallV1ConfbridgeCreate(ctx context.Context, customerID uuid.UUID, confbridgeType cmconfbridge.Type) (*cmconfbridge.Confbridge, error)
	CallV1ConfbridgeDelete(ctx context.Context, confbridgeID uuid.UUID) (*cmconfbridge.Confbridge, error)
	CallV1ConfbridgeCallKick(ctx context.Context, confbridgeID uuid.UUID, callID uuid.UUID) error
	CallV1ConfbridgeCallAdd(ctx context.Context, confbridgeID uuid.UUID, callID uuid.UUID) error
	CallV1ConfbridgeFlagAdd(ctx context.Context, confbridgeID uuid.UUID, flag cmconfbridge.Flag) (*cmconfbridge.Confbridge, error)
	CallV1ConfbridgeFlagRemove(ctx context.Context, confbridgeID uuid.UUID, flag cmconfbridge.Flag) (*cmconfbridge.Confbridge, error)
	CallV1ConfbridgeGet(ctx context.Context, confbridgeID uuid.UUID) (*cmconfbridge.Confbridge, error)
	CallV1ConfbridgeRecordingStart(ctx context.Context, confbridgeID uuid.UUID, format cmrecording.Format, endOfSilence int, endOfKey string, duration int) (*cmconfbridge.Confbridge, error)
	CallV1ConfbridgeRecordingStop(ctx context.Context, confbridgeID uuid.UUID) (*cmconfbridge.Confbridge, error)
	CallV1ConfbridgeTerminate(ctx context.Context, confbridgeID uuid.UUID) (*cmconfbridge.Confbridge, error)
	CallV1ConfbridgeRing(ctx context.Context, confbridgeID uuid.UUID) error
	CallV1ConfbridgeAnswer(ctx context.Context, confbridgeID uuid.UUID) error
	CallV1ConfbridgeExternalMediaStart(
		ctx context.Context,
		confbridgeID uuid.UUID,
		externalMediaID uuid.UUID,
		externalHost string, // external host:port
		encapsulation string, // rtp
		transport string, // udp
		connectionType string, // client,server
		format string, // ulaw
		direction string, // in,out,both
	) (*cmconfbridge.Confbridge, error)
	CallV1ConfbridgeExternalMediaStop(ctx context.Context, confbridgeID uuid.UUID) (*cmconfbridge.Confbridge, error)

	// call-manager external-media
	CallV1ExternalMediaGet(ctx context.Context, externalMediaID uuid.UUID) (*cmexternalmedia.ExternalMedia, error)
	CallV1ExternalMediaGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cmexternalmedia.ExternalMedia, error)
	CallV1ExternalMediaStart(ctx context.Context, externalMediaID uuid.UUID, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, noInsertMedia bool, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string) (*cmexternalmedia.ExternalMedia, error)
	CallV1ExternalMediaStop(ctx context.Context, externalMediaID uuid.UUID) (*cmexternalmedia.ExternalMedia, error)

	// call-manager groupcall
	CallV1GroupcallCreate(
		ctx context.Context,
		id uuid.UUID,
		customerID uuid.UUID,
		flowID uuid.UUID,
		source commonaddress.Address,
		destinations []commonaddress.Address,
		masterCallID uuid.UUID,
		masterGroupcallID uuid.UUID,
		ringMethod cmgroupcall.RingMethod,
		answerMethod cmgroupcall.AnswerMethod,
	) (*cmgroupcall.Groupcall, error)
	CallV1GroupcallGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cmgroupcall.Groupcall, error)
	CallV1GroupcallGet(ctx context.Context, groupcallID uuid.UUID) (*cmgroupcall.Groupcall, error)
	CallV1GroupcallDelete(ctx context.Context, groupcallID uuid.UUID) (*cmgroupcall.Groupcall, error)
	CallV1GroupcallHangup(ctx context.Context, groupcallID uuid.UUID) (*cmgroupcall.Groupcall, error)
	CallV1GroupcallUpdateAnswerGroupcallID(ctx context.Context, groupcallID uuid.UUID, answerGroupcallID uuid.UUID) (*cmgroupcall.Groupcall, error)
	CallV1GroupcallHangupOthers(ctx context.Context, groupcallID uuid.UUID) error
	CallV1GroupcallHangupCall(ctx context.Context, groupcallID uuid.UUID) error
	CallV1GroupcallHangupGroupcall(ctx context.Context, groupcallID uuid.UUID) error

	// call-manager recordings
	CallV1RecordingGet(ctx context.Context, id uuid.UUID) (*cmrecording.Recording, error)
	CallV1RecordingGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cmrecording.Recording, error)
	CallV1RecordingDelete(ctx context.Context, id uuid.UUID) (*cmrecording.Recording, error)
	CallV1RecordingStart(
		ctx context.Context,
		referenceType cmrecording.ReferenceType,
		referenceID uuid.UUID,
		format cmrecording.Format,
		endOfSilence int,
		endOfKey string,
		duration int,
	) (*cmrecording.Recording, error)
	CallV1RecordingStop(ctx context.Context, recordingID uuid.UUID) (*cmrecording.Recording, error)

	// campaign-manager campaigns
	CampaignV1CampaignCreate(
		ctx context.Context,
		id uuid.UUID,
		customerID uuid.UUID,
		campaignType cacampaign.Type,
		name string,
		detail string,
		serviceLevel int,
		endHandle cacampaign.EndHandle,
		actions []fmaction.Action,
		outplanID uuid.UUID,
		outdialID uuid.UUID,
		queueID uuid.UUID,
		nextCampaignID uuid.UUID,
	) (*cacampaign.Campaign, error)
	CampaignV1CampaignGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cacampaign.Campaign, error)
	CampaignV1CampaignGet(ctx context.Context, id uuid.UUID) (*cacampaign.Campaign, error)
	CampaignV1CampaignDelete(ctx context.Context, campaignID uuid.UUID) (*cacampaign.Campaign, error)
	CampaignV1CampaignExecute(ctx context.Context, id uuid.UUID, delay int) error
	CampaignV1CampaignUpdateBasicInfo(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		campaignType cacampaign.Type,
		serviceLevel int,
		endHandle cacampaign.EndHandle,
	) (*cacampaign.Campaign, error)
	CampaignV1CampaignUpdateStatus(ctx context.Context, id uuid.UUID, status cacampaign.Status) (*cacampaign.Campaign, error)
	CampaignV1CampaignUpdateServiceLevel(ctx context.Context, id uuid.UUID, serviceLevel int) (*cacampaign.Campaign, error)
	CampaignV1CampaignUpdateActions(ctx context.Context, id uuid.UUID, actions []fmaction.Action) (*cacampaign.Campaign, error)
	CampaignV1CampaignUpdateResourceInfo(ctx context.Context, id uuid.UUID, outplanID uuid.UUID, outdialID uuid.UUID, queueID uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.Campaign, error)
	CampaignV1CampaignUpdateNextCampaignID(ctx context.Context, id uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.Campaign, error)

	// campaign-manager campaigncalls
	CampaignV1CampaigncallGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cacampaigncall.Campaigncall, error)
	CampaignV1CampaigncallGetsByCampaignID(ctx context.Context, campaignID uuid.UUID, pageToken string, pageSize uint64) ([]cacampaigncall.Campaigncall, error)
	CampaignV1CampaigncallGet(ctx context.Context, id uuid.UUID) (*cacampaigncall.Campaigncall, error)
	CampaignV1CampaigncallDelete(ctx context.Context, id uuid.UUID) (*cacampaigncall.Campaigncall, error)

	// campaign-manager outplans
	CampaignV1OutplanCreate(
		ctx context.Context,
		customerID uuid.UUID,
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
	) (*caoutplan.Outplan, error)
	CampaignV1OutplanGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]caoutplan.Outplan, error)
	CampaignV1OutplanGet(ctx context.Context, id uuid.UUID) (*caoutplan.Outplan, error)
	CampaignV1OutplanDelete(ctx context.Context, outplanID uuid.UUID) (*caoutplan.Outplan, error)
	CampaignV1OutplanUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*caoutplan.Outplan, error)
	CampaignV1OutplanUpdateDialInfo(
		ctx context.Context,
		id uuid.UUID,
		source *commonaddress.Address,
		dialTimeout int,
		tryInterval int,
		maxTryCount0 int,
		maxTryCount1 int,
		maxTryCount2 int,
		maxTryCount3 int,
		maxTryCount4 int,
	) (*caoutplan.Outplan, error)

	// chat-manager chatrooms
	ChatV1ChatroomGet(ctx context.Context, chatroomID uuid.UUID) (*chatchatroom.Chatroom, error)
	ChatV1ChatroomGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]chatchatroom.Chatroom, error)
	ChatV1ChatroomDelete(ctx context.Context, chatroomID uuid.UUID) (*chatchatroom.Chatroom, error)
	ChatV1ChatroomUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*chatchatroom.Chatroom, error)

	// chat-manager chats
	ChatV1ChatCreate(
		ctx context.Context,
		customerID uuid.UUID,
		chatType chatchat.Type,
		roomOwnerID uuid.UUID,
		participantIDs []uuid.UUID,
		name string,
		detail string,
	) (*chatchat.Chat, error)
	ChatV1ChatGet(ctx context.Context, chatID uuid.UUID) (*chatchat.Chat, error)
	ChatV1ChatGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]chatchat.Chat, error)
	ChatV1ChatDelete(ctx context.Context, chatID uuid.UUID) (*chatchat.Chat, error)
	ChatV1ChatUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*chatchat.Chat, error)
	ChatV1ChatUpdateRoomOwnerID(ctx context.Context, id uuid.UUID, roomOwnerID uuid.UUID) (*chatchat.Chat, error)
	ChatV1ChatAddParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chatchat.Chat, error)
	ChatV1ChatRemoveParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chatchat.Chat, error)

	// chat-manager messagerooms
	ChatV1MessagechatroomGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]chatmessagechatroom.Messagechatroom, error)
	ChatV1MessagechatroomGet(ctx context.Context, messagechatroomID uuid.UUID) (*chatmessagechatroom.Messagechatroom, error)
	ChatV1MessagechatroomDelete(ctx context.Context, messagechatroomID uuid.UUID) (*chatmessagechatroom.Messagechatroom, error)

	// chat-manager messagechats
	ChatV1MessagechatCreate(
		ctx context.Context,
		customerID uuid.UUID,
		chatID uuid.UUID,
		source commonaddress.Address,
		messageType chatmessagechat.Type,
		text string,
		medias []chatmedia.Media,
	) (*chatmessagechat.Messagechat, error)
	ChatV1MessagechatGet(ctx context.Context, messagechatID uuid.UUID) (*chatmessagechat.Messagechat, error)
	ChatV1MessagechatGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]chatmessagechat.Messagechat, error)
	ChatV1MessagechatDelete(ctx context.Context, chatID uuid.UUID) (*chatmessagechat.Messagechat, error)

	// chatbot-manager chatbot
	ChatbotV1ChatbotGet(ctx context.Context, chatbotID uuid.UUID) (*cbchatbot.Chatbot, error)
	ChatbotV1ChatbotGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64, filters map[string]string) ([]cbchatbot.Chatbot, error)
	ChatbotV1ChatbotCreate(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		engineType cbchatbot.EngineType,
		engineModel cbchatbot.EngineModel,
		initPrompt string,
		credentialBase64 string,
		credentialProjectID string,
	) (*cbchatbot.Chatbot, error)
	ChatbotV1ChatbotDelete(ctx context.Context, chatbotID uuid.UUID) (*cbchatbot.Chatbot, error)
	ChatbotV1ChatbotUpdate(
		ctx context.Context,
		chatbotID uuid.UUID,
		name string,
		detail string,
		engineType cbchatbot.EngineType,
		engineModel cbchatbot.EngineModel,
		initPrompt string,
		credentialBase64 string,
		credentialProjectID string,
	) (*cbchatbot.Chatbot, error)

	// chatbot-manager chatbotcall
	ChatbotV1ChatbotcallGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64, filters map[string]string) ([]cbchatbotcall.Chatbotcall, error)
	ChatbotV1ChatbotcallGet(ctx context.Context, chatbotcallID uuid.UUID) (*cbchatbotcall.Chatbotcall, error)
	ChatbotV1ChatbotcallDelete(ctx context.Context, chatbotcallID uuid.UUID) (*cbchatbotcall.Chatbotcall, error)

	// chatbot-manager service
	ChatbotV1ServiceTypeChabotcallStart(
		ctx context.Context,
		chatbotID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType cbchatbotcall.ReferenceType,
		referenceID uuid.UUID,
		gender cbchatbotcall.Gender,
		language string,
		requestTimeout int,
	) (*cbservice.Service, error)

	// customer-manager accesskeys
	CustomerV1AccesskeyCreate(ctx context.Context, customerID uuid.UUID, name string, detail string, expire int32) (*csaccesskey.Accesskey, error)
	CustomerV1AccesskeyDelete(ctx context.Context, accesskeyID uuid.UUID) (*csaccesskey.Accesskey, error)
	CustomerV1AccesskeyGet(ctx context.Context, accesskeyID uuid.UUID) (*csaccesskey.Accesskey, error)
	CustomerV1AccesskeyGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]csaccesskey.Accesskey, error)
	CustomerV1AccesskeyUpdate(ctx context.Context, accesskeyID uuid.UUID, name string, detail string) (*csaccesskey.Accesskey, error)

	// customer-manager customer
	CustomerV1CustomerCreate(
		ctx context.Context,
		requestTimeout int,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
	) (*cscustomer.Customer, error)
	CustomerV1CustomerDelete(ctx context.Context, id uuid.UUID) (*cscustomer.Customer, error)
	CustomerV1CustomerGet(ctx context.Context, customerID uuid.UUID) (*cscustomer.Customer, error)
	CustomerV1CustomerGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cscustomer.Customer, error)
	CustomerV1CustomerUpdate(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
	) (*cscustomer.Customer, error)
	CustomerV1CustomerIsValidBalance(ctx context.Context, customerID uuid.UUID, referenceType bmbilling.ReferenceType, country string, count int) (bool, error)
	CustomerV1CustomerUpdateBillingAccountID(ctx context.Context, customerID uuid.UUID, biillingAccountID uuid.UUID) (*cscustomer.Customer, error)

	// conference-manager conference
	ConferenceV1ConferenceGet(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error)
	ConferenceV1ConferenceGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cfconference.Conference, error)
	ConferenceV1ConferenceCreate(
		ctx context.Context,
		customerID uuid.UUID,
		conferenceType cfconference.Type,
		name string,
		detail string,
		timeout int,
		data map[string]interface{},
		preActions []fmaction.Action,
		postActions []fmaction.Action,
	) (*cfconference.Conference, error)
	ConferenceV1ConferenceDelete(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error)
	ConferenceV1ConferenceDeleteDelay(ctx context.Context, conferenceID uuid.UUID, delay int) error
	ConferenceV1ConferenceUpdate(ctx context.Context, id uuid.UUID, name string, detail string, timeout int, preActions, postActions []fmaction.Action) (*cfconference.Conference, error)
	ConferenceV1ConferenceUpdateRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) (*cfconference.Conference, error)
	ConferenceV1ConferenceRecordingStart(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error)
	ConferenceV1ConferenceRecordingStop(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error)
	ConferenceV1ConferenceStop(ctx context.Context, conferenceID uuid.UUID, delay int) (*cfconference.Conference, error)
	ConferenceV1ConferenceTranscribeStart(ctx context.Context, conferenceID uuid.UUID, language string) (*cfconference.Conference, error)
	ConferenceV1ConferenceTranscribeStop(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error)

	// conference-manager conferencecall
	ConferenceV1ConferencecallGet(ctx context.Context, conferencecallID uuid.UUID) (*cfconferencecall.Conferencecall, error)
	ConferenceV1ConferencecallGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cfconferencecall.Conferencecall, error)
	ConferenceV1ConferencecallKick(ctx context.Context, conferencecallID uuid.UUID) (*cfconferencecall.Conferencecall, error)
	ConferenceV1ConferencecallHealthCheck(ctx context.Context, conferencecallID uuid.UUID, retryCount int, delay int) error

	// conference-manager service
	ConferenceV1ServiceTypeConferencecallStart(ctx context.Context, conferenceID uuid.UUID, referenceType cfconferencecall.ReferenceType, referenceID uuid.UUID) (*cfservice.Service, error)

	// conversation-manager account
	ConversationV1AccountGet(ctx context.Context, accountID uuid.UUID) (*cvaccount.Account, error)
	ConversationV1AccountGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cvaccount.Account, error)
	ConversationV1AccountCreate(ctx context.Context, customerID uuid.UUID, accountType cvaccount.Type, name string, detail string, secret string, token string) (*cvaccount.Account, error)
	ConversationV1AccountUpdate(ctx context.Context, accountID uuid.UUID, name string, detail string, secret string, token string) (*cvaccount.Account, error)
	ConversationV1AccountDelete(ctx context.Context, accountID uuid.UUID) (*cvaccount.Account, error)

	// conversation-manager conversation
	ConversationV1ConversationGet(ctx context.Context, conversationID uuid.UUID) (*cvconversation.Conversation, error)
	ConversationV1ConversationGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cvconversation.Conversation, error)
	ConversationV1MessageSend(ctx context.Context, conversationID uuid.UUID, text string, medias []cvmedia.Media) (*cvmessage.Message, error)
	ConversationV1ConversationMessageGetsByConversationID(ctx context.Context, conversationID uuid.UUID, pageToken string, pageSize uint64) ([]cvmessage.Message, error)
	ConversationV1ConversationUpdate(ctx context.Context, conversationID uuid.UUID, name string, detail string) (*cvconversation.Conversation, error)

	// conversation-manager hook
	ConversationV1Hook(ctx context.Context, hm *hmhook.Hook) error

	// flow-manager action
	FlowV1ActionGet(ctx context.Context, flowID, actionID uuid.UUID) (*fmaction.Action, error)

	// flow-manager activeflow
	FlowV1ActiveflowCreate(ctx context.Context, activeflowID, flowID uuid.UUID, referenceType fmactiveflow.ReferenceType, referenceID uuid.UUID) (*fmactiveflow.Activeflow, error)
	FlowV1ActiveflowDelete(ctx context.Context, activeflowID uuid.UUID) (*fmactiveflow.Activeflow, error)
	FlowV1ActiveflowGet(ctx context.Context, activeflowID uuid.UUID) (*fmactiveflow.Activeflow, error)
	FlowV1ActiveflowGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]fmactiveflow.Activeflow, error)
	FlowV1ActiveflowGetNextAction(ctx context.Context, activeflowID, actionID uuid.UUID) (*fmaction.Action, error)
	FlowV1ActiveflowUpdateForwardActionID(ctx context.Context, activeflowID, forwardActionID uuid.UUID, forwardNow bool) error
	FlowV1ActiveflowExecute(ctx context.Context, activeflowID uuid.UUID) error
	FlowV1ActiveflowStop(ctx context.Context, activeflowID uuid.UUID) (*fmactiveflow.Activeflow, error)
	FlowV1ActiveflowPushActions(ctx context.Context, activeflowID uuid.UUID, actions []fmaction.Action) (*fmactiveflow.Activeflow, error)

	// flow-manager flow
	FlowV1FlowCreate(ctx context.Context, customerID uuid.UUID, flowType fmflow.Type, name string, detail string, actions []fmaction.Action, persist bool) (*fmflow.Flow, error)
	FlowV1FlowDelete(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error)
	FlowV1FlowGet(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error)
	FlowV1FlowGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]fmflow.Flow, error)
	FlowV1FlowUpdate(ctx context.Context, f *fmflow.Flow) (*fmflow.Flow, error)
	FlowV1FlowUpdateActions(ctx context.Context, flowID uuid.UUID, actions []fmaction.Action) (*fmflow.Flow, error)

	// flow-manager variables
	FlowV1VariableGet(ctx context.Context, variableID uuid.UUID) (*fmvariable.Variable, error)
	FlowV1VariableDeleteVariable(ctx context.Context, variableID uuid.UUID, key string) error
	FlowV1VariableSetVariable(ctx context.Context, variableID uuid.UUID, variables map[string]string) error

	// message-manager hook
	MessageV1Hook(ctx context.Context, hm *hmhook.Hook) error

	// message-manager message
	MessageV1MessageGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]mmmessage.Message, error)
	MessageV1MessageGet(ctx context.Context, id uuid.UUID) (*mmmessage.Message, error)
	MessageV1MessageDelete(ctx context.Context, id uuid.UUID) (*mmmessage.Message, error)
	MessageV1MessageSend(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*mmmessage.Message, error)

	// number-manager available-number
	NumberV1AvailableNumberGets(ctx context.Context, customerID uuid.UUID, pageSize uint64, countryCode string) ([]nmavailablenumber.AvailableNumber, error)

	// number-manager number
	NumberV1NumberCreate(ctx context.Context, customerID uuid.UUID, num string, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*nmnumber.Number, error)
	NumberV1NumberDelete(ctx context.Context, id uuid.UUID) (*nmnumber.Number, error)
	NumberV1NumberGet(ctx context.Context, numberID uuid.UUID) (*nmnumber.Number, error)
	NumberV1NumberGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]nmnumber.Number, error)
	NumberV1NumberUpdate(ctx context.Context, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*nmnumber.Number, error)
	NumberV1NumberUpdateFlowID(ctx context.Context, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID) (*nmnumber.Number, error)
	NumberV1NumberRenewByTmRenew(ctx context.Context, tmRenew string) ([]nmnumber.Number, error)
	NumberV1NumberRenewByDays(ctx context.Context, days int) ([]nmnumber.Number, error)
	NumberV1NumberRenewByHours(ctx context.Context, hours int) ([]nmnumber.Number, error)

	// outdial-manager outdial
	OutdialV1OutdialCreate(ctx context.Context, customerID, campaignID uuid.UUID, name, detail, data string) (*omoutdial.Outdial, error)
	OutdialV1OutdialDelete(ctx context.Context, outdialID uuid.UUID) (*omoutdial.Outdial, error)
	OutdialV1OutdialGet(ctx context.Context, id uuid.UUID) (*omoutdial.Outdial, error)
	OutdialV1OutdialGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]omoutdial.Outdial, error)
	OutdialV1OutdialUpdateBasicInfo(ctx context.Context, outdialID uuid.UUID, name, detail string) (*omoutdial.Outdial, error)
	OutdialV1OutdialUpdateCampaignID(ctx context.Context, outdialID, campaignID uuid.UUID) (*omoutdial.Outdial, error)
	OutdialV1OutdialUpdateData(ctx context.Context, outdialID uuid.UUID, data string) (*omoutdial.Outdial, error)

	// outdial-manager outdialtarget
	OutdialV1OutdialtargetCreate(
		ctx context.Context,
		outdialID uuid.UUID,
		name string,
		detail string,
		data string,
		destination0 *commonaddress.Address,
		destination1 *commonaddress.Address,
		destination2 *commonaddress.Address,
		destination3 *commonaddress.Address,
		destination4 *commonaddress.Address,
	) (*omoutdialtarget.OutdialTarget, error)
	OutdialV1OutdialtargetDelete(ctx context.Context, outdialtargetID uuid.UUID) (*omoutdialtarget.OutdialTarget, error)
	OutdialV1OutdialtargetGet(ctx context.Context, outdialtargetID uuid.UUID) (*omoutdialtarget.OutdialTarget, error)
	OutdialV1OutdialtargetGetsByOutdialID(ctx context.Context, outdialID uuid.UUID, pageToken string, pageSize uint64) ([]omoutdialtarget.OutdialTarget, error)
	OutdialV1OutdialtargetGetsAvailable(
		ctx context.Context,
		outdialID uuid.UUID,
		tryCount0 int,
		tryCount1 int,
		tryCount2 int,
		tryCount3 int,
		tryCount4 int,
		limit int,
	) ([]omoutdialtarget.OutdialTarget, error)
	OutdialV1OutdialtargetUpdateStatusProgressing(ctx context.Context, outdialtargetID uuid.UUID, destinationIndex int) (*omoutdialtarget.OutdialTarget, error)
	OutdialV1OutdialtargetUpdateStatus(ctx context.Context, outdialtargetID uuid.UUID, status omoutdialtarget.Status) (*omoutdialtarget.OutdialTarget, error)

	// queue-manager queue
	QueueV1QueueGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]qmqueue.Queue, error)
	QueueV1QueueGet(ctx context.Context, queueID uuid.UUID) (*qmqueue.Queue, error)
	QueueV1QueueGetAgents(ctx context.Context, queueID uuid.UUID, status amagent.Status) ([]amagent.Agent, error)
	QueueV1QueueCreate(ctx context.Context, customerID uuid.UUID, name, detail string, routingMethod qmqueue.RoutingMethod, tagIDs []uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) (*qmqueue.Queue, error)
	QueueV1QueueDelete(ctx context.Context, queueID uuid.UUID) (*qmqueue.Queue, error)
	QueueV1QueueExecuteRun(ctx context.Context, queueID uuid.UUID, executeDelay int) error
	QueueV1QueueUpdate(
		ctx context.Context,
		queueID uuid.UUID,
		name string,
		detail string,
		routingMethod qmqueue.RoutingMethod,
		tagIDs []uuid.UUID,
		waitActions []fmaction.Action,
		waitTimeout int,
		serviceTimeout int,
	) (*qmqueue.Queue, error)
	QueueV1QueueUpdateTagIDs(ctx context.Context, queueID uuid.UUID, tagIDs []uuid.UUID) (*qmqueue.Queue, error)
	QueueV1QueueUpdateRoutingMethod(ctx context.Context, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) (*qmqueue.Queue, error)
	QueueV1QueueUpdateActions(ctx context.Context, queueID uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) (*qmqueue.Queue, error)
	QueueV1QueueUpdateExecute(ctx context.Context, queueID uuid.UUID, execute qmqueue.Execute) (*qmqueue.Queue, error)
	QueueV1QueueCreateQueuecall(ctx context.Context, queueID uuid.UUID, referenceType qmqueuecall.ReferenceType, referenceID, referenceActiveflowID, exitActionID uuid.UUID) (*qmqueuecall.Queuecall, error)

	// queue-manager queuecall
	QueueV1QueuecallGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]qmqueuecall.Queuecall, error)
	QueueV1QueuecallGet(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error)
	QueueV1QueuecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*qmqueuecall.Queuecall, error)
	QueueV1QueuecallDelete(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error)
	QueueV1QueuecallExecute(ctx context.Context, queuecallID uuid.UUID, agentID uuid.UUID) (*qmqueuecall.Queuecall, error)
	QueueV1QueuecallHealthCheck(ctx context.Context, id uuid.UUID, delay int, retryCount int) error
	QueueV1QueuecallKick(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error)
	QueueV1QueuecallKickByReferenceID(ctx context.Context, referenceID uuid.UUID) (*qmqueuecall.Queuecall, error)
	QueueV1QueuecallTimeoutWait(ctx context.Context, queuecallID uuid.UUID, delay int) error
	QueueV1QueuecallTimeoutService(ctx context.Context, queuecallID uuid.UUID, delay int) error
	QueueV1QueuecallUpdateStatusWaiting(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error)

	// queue-manager service
	QueueV1ServiceTypeQueuecallStart(ctx context.Context, queueID uuid.UUID, activeflowID uuid.UUID, referenceType qmqueuecall.ReferenceType, referenceID uuid.UUID, exitActionID uuid.UUID) (*qmservice.Service, error)

	// registrar-manager contact
	RegistrarV1ContactGets(ctx context.Context, customerID uuid.UUID, extension string) ([]rmastcontact.AstContact, error)
	RegistrarV1ContactRefresh(ctx context.Context, customerID uuid.UUID, extension string) error

	// registrar-manager extension
	RegistrarV1ExtensionCreate(ctx context.Context, customerID uuid.UUID, ext string, password string, name string, detail string) (*rmextension.Extension, error)
	RegistrarV1ExtensionDelete(ctx context.Context, extensionID uuid.UUID) (*rmextension.Extension, error)
	RegistrarV1ExtensionGet(ctx context.Context, extensionID uuid.UUID) (*rmextension.Extension, error)
	RegistrarV1ExtensionGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]rmextension.Extension, error)
	RegistrarV1ExtensionUpdate(ctx context.Context, id uuid.UUID, name, detail, password string) (*rmextension.Extension, error)

	// registrar-manager trunk
	RegistrarV1TrunkCreate(ctx context.Context, customerID uuid.UUID, name string, detail string, domainName string, authTypes []rmsipauth.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.Trunk, error)
	RegistrarV1TrunkGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]rmtrunk.Trunk, error)
	RegistrarV1TrunkGet(ctx context.Context, trunkID uuid.UUID) (*rmtrunk.Trunk, error)
	RegistrarV1TrunkGetByDomainName(ctx context.Context, domainName string) (*rmtrunk.Trunk, error)
	RegistrarV1TrunkDelete(ctx context.Context, trunkID uuid.UUID) (*rmtrunk.Trunk, error)
	RegistrarV1TrunkUpdateBasicInfo(ctx context.Context, trunkID uuid.UUID, name string, detail string, authTypes []rmsipauth.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.Trunk, error)

	// route-manager dialroutes
	RouteV1DialrouteGets(ctx context.Context, customerID uuid.UUID, target string) ([]rmroute.Route, error)

	// route-manager providers
	RouteV1ProviderCreate(ctx context.Context, provierType rmprovider.Type, hostname string, techPrefix string, techPostfix string, techHeaders map[string]string, name string, detail string) (*rmprovider.Provider, error)
	RouteV1ProviderGet(ctx context.Context, providerID uuid.UUID) (*rmprovider.Provider, error)
	RouteV1ProviderDelete(ctx context.Context, providerID uuid.UUID) (*rmprovider.Provider, error)
	RouteV1ProviderUpdate(
		ctx context.Context,
		providerID uuid.UUID,
		providerType rmprovider.Type,
		hostname string,
		techPrefix string,
		techPostfix string,
		techHeaders map[string]string,
		name string,
		detail string,
	) (*rmprovider.Provider, error)
	RouteV1ProviderGets(ctx context.Context, pageToken string, pageSize uint64) ([]rmprovider.Provider, error)

	// route-manager routes
	RouteV1RouteCreate(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		providerID uuid.UUID,
		priority int,
		target string,
	) (*rmroute.Route, error)
	RouteV1RouteGet(ctx context.Context, routeID uuid.UUID) (*rmroute.Route, error)
	RouteV1RouteDelete(ctx context.Context, routeID uuid.UUID) (*rmroute.Route, error)
	RouteV1RouteUpdate(
		ctx context.Context,
		routeID uuid.UUID,
		name string,
		detail string,
		providerID uuid.UUID,
		priority int,
		target string,
	) (*rmroute.Route, error)
	RouteV1RouteGets(ctx context.Context, pageToken string, pageSize uint64) ([]rmroute.Route, error)
	RouteV1RouteGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]rmroute.Route, error)

	// storage-manager account
	StorageV1AccountCreate(ctx context.Context, customerID uuid.UUID) (*smaccount.Account, error)
	StorageV1AccountGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]smaccount.Account, error)
	StorageV1AccountGet(ctx context.Context, accountID uuid.UUID) (*smaccount.Account, error)
	StorageV1AccountDelete(ctx context.Context, fileID uuid.UUID, requestTimeout int) (*smaccount.Account, error)

	// storage-manager compressfile
	StorageV1CompressfileCreate(ctx context.Context, referenceIDs []uuid.UUID, fileIDs []uuid.UUID, requestTimeout int) (*smcompressfile.CompressFile, error)

	// storage-manager recording
	StorageV1RecordingGet(ctx context.Context, id uuid.UUID, requestTimeout int) (*smbucketfile.BucketFile, error)
	StorageV1RecordingDelete(ctx context.Context, recordingID uuid.UUID) error

	// storage-manager file
	StorageV1FileCreate(
		ctx context.Context,
		customerID uuid.UUID,
		ownerID uuid.UUID,
		referenceType smfile.ReferenceType,
		referenceID uuid.UUID,
		name string,
		detail string,
		filename string,
		bucketName string,
		filepath string,
		requestTimeout int, // milliseconds
	) (*smfile.File, error)
	StorageV1FileCreateWithDelay(
		ctx context.Context,
		customerID uuid.UUID,
		ownerID uuid.UUID,
		referenceType smfile.ReferenceType,
		referenceID uuid.UUID,
		name string,
		detail string,
		filename string,
		bucketName string,
		filepath string,
		delay int, // milliseconds
	) error
	StorageV1FileGet(ctx context.Context, fileID uuid.UUID) (*smfile.File, error)
	StorageV1FileGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]smfile.File, error)
	StorageV1FileDelete(ctx context.Context, fileID uuid.UUID, requestTimeout int) (*smfile.File, error)

	// tag-manager
	TagV1TagCreate(ctx context.Context, customerID uuid.UUID, name string, detail string) (*tmtag.Tag, error)
	TagV1TagUpdate(ctx context.Context, tagID uuid.UUID, name string, detail string) (*tmtag.Tag, error)
	TagV1TagDelete(ctx context.Context, tagID uuid.UUID) (*tmtag.Tag, error)
	TagV1TagGet(ctx context.Context, tagID uuid.UUID) (*tmtag.Tag, error)
	TagV1TagGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]tmtag.Tag, error)

	// tts-manager speeches
	TTSV1SpeecheCreate(ctx context.Context, callID uuid.UUID, text string, gender tmtts.Gender, language string, timeout int) (*tmtts.TTS, error)

	// transcribe-manager
	TranscribeV1TranscribeGet(ctx context.Context, transcribeID uuid.UUID) (*tmtranscribe.Transcribe, error)
	TranscribeV1TranscribeGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]tmtranscribe.Transcribe, error)
	TranscribeV1TranscribeHealthCheck(ctx context.Context, id uuid.UUID, delay int, retryCount int) error
	TranscribeV1TranscribeStart(
		ctx context.Context,
		customerID uuid.UUID,
		referenceType tmtranscribe.ReferenceType,
		referenceID uuid.UUID,
		language string,
		direction tmtranscribe.Direction,
	) (*tmtranscribe.Transcribe, error)
	TranscribeV1TranscribeStop(ctx context.Context, transcribeID uuid.UUID) (*tmtranscribe.Transcribe, error)
	TranscribeV1TranscribeDelete(ctx context.Context, transcribeID uuid.UUID) (*tmtranscribe.Transcribe, error)
	TranscribeV1TranscriptGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]tmtranscript.Transcript, error)

	// transfer-manager
	TransferV1TransferStart(ctx context.Context, transferType tmtransfer.Type, transfererCallID uuid.UUID, transfereeAddresses []commonaddress.Address) (*tmtransfer.Transfer, error)

	// webhook-manager webhooks
	WebhookV1WebhookSend(ctx context.Context, customerID uuid.UUID, dataType wmwebhook.DataType, messageType string, messageData []byte) error
	WebhookV1WebhookSendToDestination(ctx context.Context, customerID uuid.UUID, destination string, method wmwebhook.MethodType, dataType wmwebhook.DataType, messageData []byte) error
}

type requestHandler struct {
	sock sockhandler.SockHandler

	publisher commonoutline.ServiceName

	utilHandler utilhandler.UtilHandler
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(sock sockhandler.SockHandler, publisher commonoutline.ServiceName) RequestHandler {
	h := &requestHandler{
		sock: sock,

		publisher:   publisher,
		utilHandler: utilhandler.NewUtilHandler(),
	}

	namespace := commonoutline.GetMetricNameSpace(publisher)
	initPrometheus(namespace)

	return h
}
