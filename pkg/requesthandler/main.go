package requesthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package requesthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"

	uuid "github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	amagentdial "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentdial"
	amtag "gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
	cmari "gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	cmbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmchannel "gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	cmresponse "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/response"
	cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	cacampaigncall "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	caoutplan "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
	chatchat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
	chatchatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	chatmedia "gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	chatmessagechat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	chatmessagechatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	cfconferencecall "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	cvmedia "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	cvmessage "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	fmvariable "gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
	hmhook "gitlab.com/voipbin/bin-manager/hook-manager.git/models/hook"
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	nmavailablenumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	omoutdial "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdial"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	qmqueuecall "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	qmqueuecallreference "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"
	rmastcontact "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	rmprovider "gitlab.com/voipbin/bin-manager/route-manager.git/models/provider"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"
	smbucketfile "gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketfile"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	tmtranscript "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	umuser "gitlab.com/voipbin/bin-manager/user-manager.git/models/user"
	wmwebhook "gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
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

// list of queue names
//
//nolint:deadcode,varcheck // this is ok
const (
	exchangeDelay = "bin-manager.delay"

	queueAgent        = "bin-manager.agent-manager.request"
	queueAPI          = "bin-manager.api-manager.request"
	queueCall         = "bin-manager.call-manager.request"
	queueCampaign     = "bin-manager.campaign-manager.request"
	queueChat         = "bin-manager.chat-manager.request"
	queueConference   = "bin-manager.conference-manager.request"
	queueConversation = "bin-manager.conversation-manager.request"
	queueCustomer     = "bin-manager.customer-manager.request"
	queueFlow         = "bin-manager.flow-manager.request"
	queueMessage      = "bin-manager.message-manager.request"
	queueNumber       = "bin-manager.number-manager.request"
	queueOutdial      = "bin-manager.outdial-manager.request"
	queueQueue        = "bin-manager.queue-manager.request"
	queueRegistrar    = "bin-manager.registrar-manager.request"
	queueRoute        = "bin-manager.route-manager.request"
	queueStorage      = "bin-manager.storage-manager.request"
	queueTranscribe   = "bin-manager.transcribe-manager.request"
	queueTTS          = "bin-manager.tts-manager.request"
	queueUser         = "bin-manager.user-manager.request"
	queueWebhook      = "bin-manager.webhook-manager.request"
)

// default stasis application name.
// normally, we don't need to use this, because proxy will set this automatically.
// but, some of Asterisk ARI required application name. this is for that.
const defaultAstStasisApp = "voipbin"

// list of prometheus metrics
var (
	promRequestProcessTime *prometheus.HistogramVec
)

type resource string

const (
	resourceAstBridges              resource = "ast/bridges"
	resourceAstBridgesAddChannel    resource = "ast/bridges/addchannel"
	resourceAstBridgesRemoveChannel resource = "ast/bridges/removechannel"

	resourceAstAMI resource = "ast/ami"

	resourceAstChannels              resource = "ast/channels"
	resourceAstChannelsAnswer        resource = "ast/channels/answer"
	resourceAstChannelsContinue      resource = "ast/channels/continue"
	resourceAstChannelsDial          resource = "ast/channels/dial"
	resourceAstChannelsExternalMedia resource = "ast/channels/externalmedia"
	resourceAstChannelsHangup        resource = "ast/channels/hangup"
	resourceAstChannelsPlay          resource = "ast/channels/play"
	resourceAstChannelsRecord        resource = "ast/channels/record"
	resourceAstChannelsSnoop         resource = "ast/channels/snoop"
	resourceAstChannelsVar           resource = "ast/channels/var"

	resourceAstPlaybacks resource = "ast/playbacks"

	resourceAstRecordingStop    resource = "ast/recording/<recording_name>/stop"
	resourceAstRecordingPause   resource = "ast/recording/<recording_name>/pause"
	resourceAstRecordingUnpause resource = "ast/recording/<recording_name>/unpause"
	resourceAstRecordingMute    resource = "ast/recording/<recording_name>/mute"
	resourceAstRecordingUnmute  resource = "ast/recording/<recording_name>/unmute"

	resourceAgentAgents resource = "agent/agents"
	resourceAgentTags   resource = "agent/tags"

	resourceCampaignCampaigns     resource = "campaign/campaigns"
	resourceCampaignCampaigncalls resource = "campaign/campaigncalls"
	resourceCampaignOutplans      resource = "campaign/outplans"

	resourceCallCalls              resource = "call/calls"
	resourceCallCallsActionNext    resource = "call/calls/action-next"
	resourceCallCallsActionTimeout resource = "call/calls/action-timeout"
	resourceCallCallsHealth        resource = "call/calls/health"
	resourceCallChannelsHealth     resource = "call/channels/health"
	resourceCallConfbridges        resource = "call/confbridges"
	resourceCallRecordings         resource = "call/recordings"

	resourceChatChats            resource = "chat/chats"
	resourceChatChatrooms        resource = "chat/chatrooms"
	resourceChatMessagechats     resource = "chat/messagechats"
	resourceChatMessagechatrooms resource = "chat/messagechatrooms"

	resourceConferenceConferences     resource = "conference/conferences"
	resourceConferenceConferencecalls resource = "conference/conferencecalls"

	resourceCustomerCustomers resource = "customer/customers"
	resourceCustomerLogin     resource = "customer/login"

	resourceConversationConversations           resource = "conversation/conversations"
	resourceConversationConversationsIDMessages resource = "conversation/conversations/<conversation-id>/messages"
	resourceConversationSetup                   resource = "conversation/setup"

	resourceFlowActions     resource = "flow/actions"
	resourceFlowFlows       resource = "flow/flows"
	resourceFlowActiveFlows resource = "flow/activeflows"
	resourceFlowVariables   resource = "flow/variables"

	resourceMessageMessages resource = "message/messages"

	resourceNumberAvailableNumbers resource = "number/available-number"
	resourceNumberNumbers          resource = "number/numbers"

	resourceOutdialOutdials       resource = "outdial/outdials"
	resourceOutdialOutdialTargets resource = "outdial/outdial_targets"

	resourceQueueQueues              resource = "queue/queues"
	resourceQueueQueuecalls          resource = "queue/queuecalls"
	resourceQueueQueuecallreferences resource = "queue/queuecallreferences"

	resourceRegistrarDomains    resource = "registrar/domain"
	resourceRegistrarExtensions resource = "registrar/extension"

	resourceRouteRoutes    resource = "route/routes"
	resourceRouteProviders resource = "route/providers"

	resourceStorageRecording resource = "storage/recording"

	resourceTranscribeTranscribes = "transcribe/transcribes"
	resourceTranscribeTranscripts = "transcribe/transcripts"

	resourceTTSSpeeches resource = "tts/speeches"

	resourceUserUsers resource = "user/users"

	resourceWebhookWebhooks resource = "webhook/webhooks"
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

	prometheus.MustRegister(
		promRequestProcessTime,
	)
}

// RequestHandler intreface for ARI request handler
type RequestHandler interface {

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
	AstChannelPlay(ctx context.Context, asteriskID string, channelID string, actionID uuid.UUID, medias []string, lang string) error
	AstChannelRecord(ctx context.Context, asteriskID string, channelID string, filename string, format string, duration int, silence int, beep bool, endKey string, ifExists string) error
	AstChannelRing(ctx context.Context, asteriskID string, channelID string) error
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
	AgentV1AgentGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]amagent.Agent, error)
	AgentV1AgentGetsByTagIDs(ctx context.Context, customerID uuid.UUID, tagIDs []uuid.UUID) ([]amagent.Agent, error)
	AgentV1AgentGetsByTagIDsAndStatus(ctx context.Context, customerID uuid.UUID, tagIDs []uuid.UUID, status amagent.Status) ([]amagent.Agent, error)
	AgentV1AgentDelete(ctx context.Context, id uuid.UUID) (*amagent.Agent, error)
	AgentV1AgentDial(ctx context.Context, id uuid.UUID, source *commonaddress.Address, flowID, masterCallID uuid.UUID) (*amagentdial.AgentDial, error)
	AgentV1AgentLogin(ctx context.Context, timeout int, customerID uuid.UUID, username, password string) (*amagent.Agent, error)
	AgentV1AgentUpdate(ctx context.Context, id uuid.UUID, name, detail string, ringMethod amagent.RingMethod) (*amagent.Agent, error)
	AgentV1AgentUpdateAddresses(ctx context.Context, id uuid.UUID, addresses []commonaddress.Address) (*amagent.Agent, error)
	AgentV1AgentUpdatePassword(ctx context.Context, timeout int, id uuid.UUID, password string) (*amagent.Agent, error)
	AgentV1AgentUpdateStatus(ctx context.Context, id uuid.UUID, status amagent.Status) (*amagent.Agent, error)
	AgentV1AgentUpdateTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) (*amagent.Agent, error)

	AgentV1TagCreate(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
	) (*amtag.Tag, error)
	AgentV1TagGet(ctx context.Context, id uuid.UUID) (*amtag.Tag, error)
	AgentV1TagGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]amtag.Tag, error)
	AgentV1TagUpdate(ctx context.Context, id uuid.UUID, name, detail string) (*amtag.Tag, error)
	AgentV1TagDelete(ctx context.Context, id uuid.UUID) (*amtag.Tag, error)

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
	CampaignV1CampaignUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*cacampaign.Campaign, error)
	CampaignV1CampaignUpdateStatus(ctx context.Context, id uuid.UUID, status cacampaign.Status) (*cacampaign.Campaign, error)
	CampaignV1CampaignUpdateServiceLevel(ctx context.Context, id uuid.UUID, serviceLevel int) (*cacampaign.Campaign, error)
	CampaignV1CampaignUpdateActions(ctx context.Context, id uuid.UUID, actions []fmaction.Action) (*cacampaign.Campaign, error)
	CampaignV1CampaignUpdateResourceInfo(ctx context.Context, id uuid.UUID, outplanID uuid.UUID, outdialID uuid.UUID, queueID uuid.UUID) (*cacampaign.Campaign, error)
	CampaignV1CampaignUpdateNextCampaignID(ctx context.Context, id uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.Campaign, error)

	// campaign-manager campaigncalls
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
	ChatV1ChatroomGetsByOwnerID(ctx context.Context, ownerID uuid.UUID, pageToken string, pageSize uint64) ([]chatchatroom.Chatroom, error)
	ChatV1ChatroomDelete(ctx context.Context, chatroomID uuid.UUID) (*chatchatroom.Chatroom, error)

	// chat-manager chats
	ChatV1ChatCreate(
		ctx context.Context,
		customerID uuid.UUID,
		chatType chatchat.Type,
		ownerID uuid.UUID,
		participantIDs []uuid.UUID,
		name string,
		detail string,
	) (*chatchat.Chat, error)
	ChatV1ChatGet(ctx context.Context, chatID uuid.UUID) (*chatchat.Chat, error)
	ChatV1ChatGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]chatchat.Chat, error)
	ChatV1ChatDelete(ctx context.Context, chatID uuid.UUID) (*chatchat.Chat, error)
	ChatV1ChatUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*chatchat.Chat, error)
	ChatV1ChatUpdateOwnerID(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*chatchat.Chat, error)
	ChatV1ChatAddParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chatchat.Chat, error)
	ChatV1ChatRemoveParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chatchat.Chat, error)

	// chat-manager messagerooms
	ChatV1MessagechatroomGetsByChatroomID(ctx context.Context, chatroomID uuid.UUID, pageToken string, pageSize uint64) ([]chatmessagechatroom.Messagechatroom, error)
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
	ChatV1MessagechatGetsByChatID(ctx context.Context, chatID uuid.UUID, pageToken string, pageSize uint64) ([]chatmessagechat.Messagechat, error)
	ChatV1MessagechatDelete(ctx context.Context, chatID uuid.UUID) (*chatmessagechat.Messagechat, error)

	// call-manager call
	CallV1CallHealth(ctx context.Context, id uuid.UUID, delay, retryCount int) error
	CallV1CallAddChainedCall(ctx context.Context, callID uuid.UUID, chainedCallID uuid.UUID) (*cmcall.Call, error)
	CallV1CallRemoveChainedCall(ctx context.Context, callID uuid.UUID, chainedCallID uuid.UUID) (*cmcall.Call, error)
	CallV1CallAddExternalMedia(ctx context.Context, callID uuid.UUID, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string) (*cmresponse.V1ResponseCallsIDExternalMediaPost, error)
	CallV1CallActionNext(ctx context.Context, callID uuid.UUID, force bool) error
	CallV1CallActionTimeout(ctx context.Context, id uuid.UUID, delay int, a *fmaction.Action) error
	CallV1CallsCreate(ctx context.Context, customerID, flowID, masterCallID uuid.UUID, source *commonaddress.Address, destination []commonaddress.Address) ([]cmcall.Call, error)
	CallV1CallCreateWithID(ctx context.Context, id, customerID, flowID, activeflowID, masterCallID uuid.UUID, source, destination *commonaddress.Address) (*cmcall.Call, error)
	CallV1CallDelete(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error)
	CallV1CallGet(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error)
	CallV1CallGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cmcall.Call, error)
	CallV1CallGetDigits(ctx context.Context, callID uuid.UUID) (string, error)
	CallV1CallSendDigits(ctx context.Context, callID uuid.UUID, digits string) error
	CallV1CallSetRecordingID(ctx context.Context, callID uuid.UUID, recordingID uuid.UUID) (*cmcall.Call, error)
	CallV1CallHangup(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error)

	// call-manager channel
	CallV1ChannelHealth(ctx context.Context, channelID string, delay, retryCount, retryCountMax int) error

	// call-manager confbridge
	CallV1ConfbridgeCreate(ctx context.Context, confbridgeType cmconfbridge.Type) (*cmconfbridge.Confbridge, error)
	CallV1ConfbridgeDelete(ctx context.Context, conferenceID uuid.UUID) error
	CallV1ConfbridgeCallKick(ctx context.Context, conferenceID uuid.UUID, callID uuid.UUID) error
	CallV1ConfbridgeCallAdd(ctx context.Context, conferenceID uuid.UUID, callID uuid.UUID) error
	CallV1ConfbridgeGet(ctx context.Context, conferenceID uuid.UUID) (*cmconfbridge.Confbridge, error)

	// call-manager recordings
	CallV1RecordingGet(ctx context.Context, id uuid.UUID) (*cmrecording.Recording, error)
	CallV1RecordingGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]cmrecording.Recording, error)
	CallV1RecordingDelete(ctx context.Context, id uuid.UUID) (*cmrecording.Recording, error)
	CallV1RecordingStart(
		ctx context.Context,
		referenceType cmrecording.ReferenceType,
		referenceID uuid.UUID,
		format string,
		endOfSilence int,
		endOfKey string,
		duration int,
	) (*cmrecording.Recording, error)
	CallV1RecordingStop(ctx context.Context, recordingID uuid.UUID) (*cmrecording.Recording, error)

	// customer-manager customer
	CustomerV1CustomerCreate(
		ctx context.Context,
		requestTimeout int,
		username string,
		password string,
		name string,
		detail string,
		webhookMethod cscustomer.WebhookMethod,
		webhookURI string,
		lineSecret string,
		lineToken string,
		permissionIDs []uuid.UUID,
	) (*cscustomer.Customer, error)
	CustomerV1CustomerDelete(ctx context.Context, id uuid.UUID) (*cscustomer.Customer, error)
	CustomerV1CustomerGet(ctx context.Context, customerID uuid.UUID) (*cscustomer.Customer, error)
	CustomerV1CustomerGets(ctx context.Context, pageToken string, pageSize uint64) ([]cscustomer.Customer, error)
	CustomerV1CustomerUpdate(ctx context.Context, id uuid.UUID, name, detail string, webhookMethod cscustomer.WebhookMethod, webhookURI string) (*cscustomer.Customer, error)
	CustomerV1CustomerUpdateLineInfo(ctx context.Context, id uuid.UUID, lineSecret string, lineToken string) (*cscustomer.Customer, error)
	CustomerV1CustomerUpdatePassword(ctx context.Context, requestTimeout int, id uuid.UUID, password string) (*cscustomer.Customer, error)
	CustomerV1CustomerUpdatePermissionIDs(ctx context.Context, id uuid.UUID, permissionIDs []uuid.UUID) (*cscustomer.Customer, error)

	// customer-manager login
	CustomerV1Login(ctx context.Context, timeout int, username, password string) (*cscustomer.Customer, error)

	// conference-manager conference
	ConferenceV1ConferenceGet(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error)
	ConferenceV1ConferenceGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64, conferenceType string) ([]cfconference.Conference, error)
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
	ConferenceV1ConferenceDelete(ctx context.Context, conferenceID uuid.UUID) error
	ConferenceV1ConferenceDeleteDelay(ctx context.Context, conferenceID uuid.UUID, delay int) error
	ConferenceV1ConferenceUpdate(ctx context.Context, id uuid.UUID, name string, detail string, timeout int, preActions, postActions []fmaction.Action) (*cfconference.Conference, error)
	ConferenceV1ConferenceUpdateRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) (*cfconference.Conference, error)
	ConferenceV1ConferenceRecordingStart(ctx context.Context, conferenceID uuid.UUID) error
	ConferenceV1ConferenceRecordingStop(ctx context.Context, conferenceID uuid.UUID) error

	// conference-manager conferencecall
	ConferenceV1ConferencecallGet(ctx context.Context, conferencecallID uuid.UUID) (*cfconferencecall.Conferencecall, error)
	ConferenceV1ConferencecallCreate(ctx context.Context, conferenceID uuid.UUID, referenceType cfconferencecall.ReferenceType, referenceID uuid.UUID) (*cfconferencecall.Conferencecall, error)
	ConferenceV1ConferencecallKick(ctx context.Context, conferencecallID uuid.UUID) (*cfconferencecall.Conferencecall, error)

	// conversation-manager conversation
	ConversationV1ConversationGet(ctx context.Context, conversationID uuid.UUID) (*cvconversation.Conversation, error)
	ConversationV1ConversationGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cvconversation.Conversation, error)
	ConversationV1MessageSend(ctx context.Context, conversationID uuid.UUID, text string, medias []cvmedia.Media) (*cvmessage.Message, error)
	ConversationV1ConversationMessageGetsByConversationID(ctx context.Context, conversationID uuid.UUID, pageToken string, pageSize uint64) ([]cvmessage.Message, error)
	ConversationV1Setup(ctx context.Context, customerID uuid.UUID, ReferenceType cvconversation.ReferenceType) error

	// conversation-manager hook
	ConversationV1Hook(ctx context.Context, hm *hmhook.Hook) error

	// flow-manager action
	FlowV1ActionGet(ctx context.Context, flowID, actionID uuid.UUID) (*fmaction.Action, error)

	// flow-manager activeflow
	FlowV1ActiveflowCreate(ctx context.Context, id, flowID uuid.UUID, referenceType fmactiveflow.ReferenceType, referenceID uuid.UUID) (*fmactiveflow.Activeflow, error)
	FlowV1ActiveflowDelete(ctx context.Context, activeflowID uuid.UUID) (*fmactiveflow.Activeflow, error)
	FlowV1ActiveflowGetNextAction(ctx context.Context, callID, actionID uuid.UUID) (*fmaction.Action, error)
	FlowV1ActiveflowUpdateForwardActionID(ctx context.Context, callID, forwardActionID uuid.UUID, forwardNow bool) error
	FlowV1ActiveflowExecute(ctx context.Context, activeflowID uuid.UUID) error

	// flow-manager flow
	FlowV1FlowCreate(ctx context.Context, customerID uuid.UUID, flowType fmflow.Type, name string, detail string, actions []fmaction.Action, persist bool) (*fmflow.Flow, error)
	FlowV1FlowDelete(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error)
	FlowV1FlowGet(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error)
	FlowV1FlowGets(ctx context.Context, customerID uuid.UUID, flowType fmflow.Type, pageToken string, pageSize uint64) ([]fmflow.Flow, error)
	FlowV1FlowUpdate(ctx context.Context, f *fmflow.Flow) (*fmflow.Flow, error)
	FlowV1FlowUpdateActions(ctx context.Context, flowID uuid.UUID, actions []fmaction.Action) (*fmflow.Flow, error)

	// flow-manager variables
	FlowV1VariableGet(ctx context.Context, variableID uuid.UUID) (*fmvariable.Variable, error)
	FlowV1VariableDeleteVariable(ctx context.Context, variableID uuid.UUID, key string) error
	FlowV1VariableSetVariable(ctx context.Context, variableID uuid.UUID, key string, value string) error

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
	NumberV1NumberCreate(ctx context.Context, customerID uuid.UUID, num string, callFlowID, messageFlowID uuid.UUID, name, detail string) (*nmnumber.Number, error)
	NumberV1NumberDelete(ctx context.Context, id uuid.UUID) (*nmnumber.Number, error)
	NumberV1NumberFlowDelete(ctx context.Context, flowID uuid.UUID) error
	NumberV1NumberGetByNumber(ctx context.Context, num string) (*nmnumber.Number, error)
	NumberV1NumberGet(ctx context.Context, numberID uuid.UUID) (*nmnumber.Number, error)
	NumberV1NumberGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]nmnumber.Number, error)
	NumberV1NumberUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*nmnumber.Number, error)
	NumberV1NumberUpdateFlowID(ctx context.Context, id, callFlowID, messageFlowID uuid.UUID) (*nmnumber.Number, error)

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
	QueueV1QueueGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]qmqueue.Queue, error)
	QueueV1QueueGet(ctx context.Context, queueID uuid.UUID) (*qmqueue.Queue, error)
	QueueV1QueueGetAgents(ctx context.Context, queueID uuid.UUID, status amagent.Status) ([]amagent.Agent, error)
	QueueV1QueueCreate(ctx context.Context, customerID uuid.UUID, name, detail string, routingMethod qmqueue.RoutingMethod, tagIDs []uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) (*qmqueue.Queue, error)
	QueueV1QueueDelete(ctx context.Context, queueID uuid.UUID) (*qmqueue.Queue, error)
	QueueV1QueueExecuteRun(ctx context.Context, queueID uuid.UUID, executeDelay int) error
	QueueV1QueueUpdate(ctx context.Context, queueID uuid.UUID, name, detail string) (*qmqueue.Queue, error)
	QueueV1QueueUpdateTagIDs(ctx context.Context, queueID uuid.UUID, tagIDs []uuid.UUID) (*qmqueue.Queue, error)
	QueueV1QueueUpdateRoutingMethod(ctx context.Context, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) (*qmqueue.Queue, error)
	QueueV1QueueUpdateActions(ctx context.Context, queueID uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) (*qmqueue.Queue, error)
	QueueV1QueueUpdateExecute(ctx context.Context, queueID uuid.UUID, execute qmqueue.Execute) (*qmqueue.Queue, error)
	QueueV1QueueCreateQueuecall(ctx context.Context, queueID uuid.UUID, referenceType qmqueuecall.ReferenceType, referenceID, referenceActiveflowID, exitActionID uuid.UUID) (*qmqueuecall.Queuecall, error)

	// queue-manager queuecall
	QueueV1QueuecallGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]qmqueuecall.Queuecall, error)
	QueueV1QueuecallGet(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error)
	QueueV1QueuecallDelete(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error)
	QueueV1QueuecallDeleteByReferenceID(ctx context.Context, referenceID uuid.UUID) (*qmqueuecall.Queuecall, error)
	QueueV1QueuecallTimeoutWait(ctx context.Context, queuecallID uuid.UUID, delay int) error
	QueueV1QueuecallTimeoutService(ctx context.Context, queuecallID uuid.UUID, delay int) error
	QueueV1QueuecallUpdateStatusWaiting(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error)

	// queue-manager queuecallreference
	QueueV1QueuecallReferenceGet(ctx context.Context, referenceID uuid.UUID) (*qmqueuecallreference.QueuecallReference, error)

	// registrar-manager contact
	RegistrarV1ContactGets(ctx context.Context, endpoint string) ([]*rmastcontact.AstContact, error)
	RegistrarV1ContactUpdate(ctx context.Context, endpoint string) error

	// registrar-manager domain
	RegistrarV1DomainCreate(ctx context.Context, customerID uuid.UUID, domainName, name, detail string) (*rmdomain.Domain, error)
	RegistrarV1DomainDelete(ctx context.Context, domainID uuid.UUID) (*rmdomain.Domain, error)
	RegistrarV1DomainGet(ctx context.Context, domainID uuid.UUID) (*rmdomain.Domain, error)
	RegistrarV1DomainGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]rmdomain.Domain, error)
	RegistrarV1DomainUpdate(ctx context.Context, id uuid.UUID, name, detail string) (*rmdomain.Domain, error)

	// registrar-manager extension
	RegistrarV1ExtensionCreate(ctx context.Context, customerID uuid.UUID, ext, password string, domainID uuid.UUID, name, detail string) (*rmextension.Extension, error)
	RegistrarV1ExtensionDelete(ctx context.Context, extensionID uuid.UUID) (*rmextension.Extension, error)
	RegistrarV1ExtensionGet(ctx context.Context, extensionID uuid.UUID) (*rmextension.Extension, error)
	RegistrarV1ExtensionGets(ctx context.Context, domainID uuid.UUID, pageToken string, pageSize uint64) ([]rmextension.Extension, error)
	RegistrarV1ExtensionUpdate(ctx context.Context, id uuid.UUID, name, detail, password string) (*rmextension.Extension, error)

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
	RouteV1RouteCreate(ctx context.Context, customerID uuid.UUID, providerID uuid.UUID, priority int, target string) (*rmroute.Route, error)
	RouteV1RouteGet(ctx context.Context, routeID uuid.UUID) (*rmroute.Route, error)
	RouteV1RouteDelete(ctx context.Context, routeID uuid.UUID) (*rmroute.Route, error)
	RouteV1RouteUpdate(ctx context.Context, routeID uuid.UUID, providerID uuid.UUID, priority int, target string) (*rmroute.Route, error)
	RouteV1RouteGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]rmroute.Route, error)

	// storage-manager: recording
	StorageV1RecordingGet(ctx context.Context, id uuid.UUID, requestTimeout int) (*smbucketfile.BucketFile, error)
	StorageV1RecordingDelete(ctx context.Context, recordingID uuid.UUID) error

	// tts-manager speeches
	TTSV1SpeecheCreate(ctx context.Context, callID uuid.UUID, text, gender, language string, timeout int) (string, error)

	// // transcribe-manager
	TranscribeV1TranscribeGet(ctx context.Context, transcribeID uuid.UUID) (*tmtranscribe.Transcribe, error)
	TranscribeV1TranscribeGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]tmtranscribe.Transcribe, error)
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
	TranscribeV1TranscriptGets(ctx context.Context, transcribeID uuid.UUID) ([]tmtranscript.Transcript, error)

	// user-manager
	UserV1UserCreate(ctx context.Context, timeout int, username, password, name, detail string, permission umuser.Permission) (*umuser.User, error)
	UserV1UserDelete(ctx context.Context, id uint64) error
	UserV1UserGet(ctx context.Context, id uint64) (*umuser.User, error)
	UserV1UserGets(ctx context.Context, pageToken string, pageSize uint64) ([]umuser.User, error)
	UserV1UserLogin(ctx context.Context, timeout int, username, password string) (*umuser.User, error)
	UserV1UserUpdateBasicInfo(ctx context.Context, userID uint64, name, detail string) error
	UserV1UserUpdatePassword(ctx context.Context, timeout int, userID uint64, password string) error
	UserV1UserUpdatePermission(ctx context.Context, userID uint64, permission umuser.Permission) error

	// webhook-manager webhooks
	WebhookV1WebhookSend(ctx context.Context, customerID uuid.UUID, dataType wmwebhook.DataType, messageType string, messageData []byte) error
	WebhookV1WebhookSendToDestination(ctx context.Context, customerID uuid.UUID, destination string, method wmwebhook.MethodType, dataType wmwebhook.DataType, messageData []byte) error
}

type requestHandler struct {
	sock rabbitmqhandler.Rabbit

	publisher string
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(sock rabbitmqhandler.Rabbit, publisher string) RequestHandler {
	h := &requestHandler{
		sock: sock,

		publisher: publisher,
	}

	namespace := strings.ReplaceAll(publisher, "-", "_")
	initPrometheus(namespace)

	return h
}
