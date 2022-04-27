package requesthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package requesthandler -destination ./mock_requesthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"

	uuid "github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	amagentdial "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentdial"
	amtag "gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmari "gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	cmbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmchannel "gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	cmresponse "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/response"
	cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	caoutplan "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
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
	smbucketrecording "gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketrecording"
	tstranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	umuser "gitlab.com/voipbin/bin-manager/user-manager.git/models/user"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// contents type
var (
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
//nolint:deadcode,varcheck // this is ok
const (
	exchangeDelay = "bin-manager.delay"

	queueAgent      = "bin-manager.agent-manager.request"
	queueAPI        = "bin-manager.api-manager.request"
	queueCall       = "bin-manager.call-manager.request"
	queueCampaign   = "bin-manager.campaign-manager.request"
	queueConference = "bin-manager.conference-manager.request"
	queueCustomer   = "bin-manager.customer-manager.request"
	queueFlow       = "bin-manager.flow-manager.request"
	queueMessage    = "bin-manager.message-manager.request"
	queueNumber     = "bin-manager.number-manager.request"
	queueOutdial    = "bin-manager.outdial-manager.request"
	queueQueue      = "bin-manager.queue-manager.request"
	queueRegistrar  = "bin-manager.registrar-manager.request"
	queueStorage    = "bin-manager.storage-manager.request"
	queueTranscribe = "bin-manager.transcribe-manager.request"
	queueTTS        = "bin-manager.tts-manager.request"
	queueUser       = "bin-manager.user-manager.request"
	queueWebhook    = "bin-manager.webhook-manager.request"
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

	resourceAMAgent resource = "am/agents"
	resourceAMTag   resource = "am/tags"

	resourceCACampaigns resource = "ca/campaigns"
	resourceCAOutplans  resource = "ca/outplans"

	resourceCMCall               resource = "cm/calls"
	resourceCMCallsActionNext    resource = "cm/calls/action-next"
	resourceCMCallsActionTimeout resource = "cm/calls/action-timeout"
	resourceCMCallsHealth        resource = "cm/calls/health"
	resourceCMChannelsHealth     resource = "cm/channels/health"

	resourceCMConfbridges resource = "cm/confbridges"

	resourceCFConferences resource = "cm/conferences"

	resourceCallRecordings resource = "cm/recordings"

	resourceCSCustomers resource = "cs/customers"

	resourceCSLogin resource = "cs/login"

	resourceFlowsActions  resource = "flows/actions"
	resourceFMFlows       resource = "fm/flows"
	resourceFMActiveFlows resource = "fm/activeflows"

	resourceMMMessages resource = "mm/messages"

	resourceNumberAvailableNumbers resource = "number-manager/available-number"
	resourceNumberNumbers          resource = "number-manager/numbers"

	resourceOMOutdials       resource = "outdial-manager/outdials"
	resourceOMOutdialTargets resource = "outdial-manager/outdial_targets"

	resourceQMQueues              resource = "qm/queues"
	resourceQMQueuecalls          resource = "qm/queuecalls"
	resourceQMQueuecallreferences resource = "qm/queuecallreferences"

	resourceRegistrarDomains    resource = "rm/domain"
	resourceRegistrarExtensions resource = "rm/extension"

	resourceStorageRecording resource = "sm/recording"

	resourceTranscribeStreamings     = "ts/streamings"
	resourceTranscribeCallRecordings = "ts/call-recordings"

	resourceTTSSpeeches resource = "tts/speeches"

	resourceUMUsers resource = "um/users"
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

	// asterisk channels
	AstChannelAnswer(ctx context.Context, asteriskID, channelID string) error
	AstChannelContinue(ctx context.Context, asteriskID, channelID, context, ext string, pri int, label string) error
	AstChannelCreate(ctx context.Context, asteriskID, channelID, appArgs, endpoint, otherChannelID, originator, formats string, variables map[string]string) error
	AstChannelCreateSnoop(ctx context.Context, asteriskID, channelID, snoopID, appArgs string, spy, whisper cmchannel.SnoopDirection) error
	AstChannelDial(ctx context.Context, asteriskID, channelID, caller string, timeout int) error
	AstChannelDTMF(ctx context.Context, asteriskID, channelID string, digit string, duration, before, between, after int) error
	AstChannelExternalMedia(ctx context.Context, asteriskID string, channelID string, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string, data string, variables map[string]string) (*cmchannel.Channel, error)
	AstChannelGet(ctx context.Context, asteriskID, channelID string) (*cmchannel.Channel, error)
	AstChannelHangup(ctx context.Context, asteriskID, channelID string, code cmari.ChannelCause, delay int) error
	AstChannelPlay(ctx context.Context, asteriskID string, channelID string, actionID uuid.UUID, medias []string, lang string) error
	AstChannelRecord(ctx context.Context, asteriskID string, channelID string, filename string, format string, duration int, silence int, beep bool, endKey string, ifExists string) error
	AstChannelRing(ctx context.Context, asteriskID string, channelID string) error
	AstChannelVariableSet(ctx context.Context, asteriskID, channelID, variable, value string) error

	// asterisk playbacks
	AstPlaybackStop(ctx context.Context, asteriskID string, playabckID string) error

	// agent-manager agent
	AMV1AgentCreate(
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
		addresses []cmaddress.Address,
	) (*amagent.Agent, error)
	AMV1AgentGet(ctx context.Context, agentID uuid.UUID) (*amagent.Agent, error)
	AMV1AgentGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]amagent.Agent, error)
	AMV1AgentGetsByTagIDs(ctx context.Context, customerID uuid.UUID, tagIDs []uuid.UUID) ([]amagent.Agent, error)
	AMV1AgentGetsByTagIDsAndStatus(ctx context.Context, customerID uuid.UUID, tagIDs []uuid.UUID, status amagent.Status) ([]amagent.Agent, error)
	AMV1AgentDelete(ctx context.Context, id uuid.UUID) (*amagent.Agent, error)
	AMV1AgentDial(ctx context.Context, id uuid.UUID, source *cmaddress.Address, flowID, masterCallID uuid.UUID) (*amagentdial.AgentDial, error)
	AMV1AgentLogin(ctx context.Context, timeout int, customerID uuid.UUID, username, password string) (*amagent.Agent, error)
	AMV1AgentUpdate(ctx context.Context, id uuid.UUID, name, detail string, ringMethod amagent.RingMethod) (*amagent.Agent, error)
	AMV1AgentUpdateAddresses(ctx context.Context, id uuid.UUID, addresses []cmaddress.Address) (*amagent.Agent, error)
	AMV1AgentUpdatePassword(ctx context.Context, timeout int, id uuid.UUID, password string) (*amagent.Agent, error)
	AMV1AgentUpdateStatus(ctx context.Context, id uuid.UUID, status amagent.Status) (*amagent.Agent, error)
	AMV1AgentUpdateTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) (*amagent.Agent, error)

	AMV1TagCreate(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
	) (*amtag.Tag, error)
	AMV1TagGet(ctx context.Context, id uuid.UUID) (*amtag.Tag, error)
	AMV1TagGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]amtag.Tag, error)
	AMV1TagUpdate(ctx context.Context, id uuid.UUID, name, detail string) (*amtag.Tag, error)
	AMV1TagDelete(ctx context.Context, id uuid.UUID) (*amtag.Tag, error)

	// campaign-manager campaigns
	CAV1CampaignCreate(
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
	CAV1CampaignGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cacampaign.Campaign, error)
	CAV1CampaignGet(ctx context.Context, id uuid.UUID) (*cacampaign.Campaign, error)
	CAV1CampaignDelete(ctx context.Context, campaignID uuid.UUID) (*cacampaign.Campaign, error)
	CAV1CampaignExecute(ctx context.Context, id uuid.UUID, delay int) error
	CAV1CampaignUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*cacampaign.Campaign, error)
	CAV1CampaignUpdateStatus(ctx context.Context, id uuid.UUID, status cacampaign.Status) (*cacampaign.Campaign, error)
	CAV1CampaignUpdateServiceLevel(ctx context.Context, id uuid.UUID, serviceLevel int) (*cacampaign.Campaign, error)
	CAV1CampaignUpdateActions(ctx context.Context, id uuid.UUID, actions []fmaction.Action) (*cacampaign.Campaign, error)

	// campaign-manager outplans
	CAV1OutplanCreate(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		source *cmaddress.Address,
		dialTimeout int,
		tryInterval int,
		maxTryCount0 int,
		maxTryCount1 int,
		maxTryCount2 int,
		maxTryCount3 int,
		maxTryCount4 int,
	) (*caoutplan.Outplan, error)
	CAV1OutplanGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]caoutplan.Outplan, error)
	CAV1OutplanGet(ctx context.Context, id uuid.UUID) (*caoutplan.Outplan, error)
	CAV1OutplanDelete(ctx context.Context, outplanID uuid.UUID) (*caoutplan.Outplan, error)
	CAV1OutplanUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*caoutplan.Outplan, error)
	CAV1OutplanUpdateDialInfo(
		ctx context.Context,
		id uuid.UUID,
		source *cmaddress.Address,
		dialTimeout int,
		tryInterval int,
		maxTryCount0 int,
		maxTryCount1 int,
		maxTryCount2 int,
		maxTryCount3 int,
		maxTryCount4 int,
	) (*caoutplan.Outplan, error)

	// call-manager call
	CMV1CallHealth(ctx context.Context, id uuid.UUID, delay, retryCount int) error
	CMV1CallAddChainedCall(ctx context.Context, callID uuid.UUID, chainedCallID uuid.UUID) (*cmcall.Call, error)
	CMV1CallRemoveChainedCall(ctx context.Context, callID uuid.UUID, chainedCallID uuid.UUID) (*cmcall.Call, error)
	CMV1CallAddExternalMedia(ctx context.Context, callID uuid.UUID, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string) (*cmresponse.V1ResponseCallsIDExternalMediaPost, error)
	CMV1CallActionNext(ctx context.Context, callID uuid.UUID, force bool) error
	CMV1CallActionTimeout(ctx context.Context, id uuid.UUID, delay int, a *fmaction.Action) error
	CMV1CallsCreate(ctx context.Context, customerID, flowID, masterCallID uuid.UUID, source *cmaddress.Address, destination []cmaddress.Address) ([]cmcall.Call, error)
	CMV1CallCreateWithID(ctx context.Context, id, customerID, flowID, activeflowID, masterCallID uuid.UUID, source, destination *cmaddress.Address) (*cmcall.Call, error)
	CMV1CallGet(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error)
	CMV1CallGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cmcall.Call, error)
	CMV1CallGetDigits(ctx context.Context, callID uuid.UUID) (string, error)
	CMV1CallSetDigits(ctx context.Context, callID uuid.UUID, digits string) error
	CMV1CallHangup(ctx context.Context, callID uuid.UUID) (*cmcall.Call, error)

	// call-manager channel
	CMV1ChannelHealth(ctx context.Context, asteriskID, channelID string, delay, retryCount, retryCountMax int) error

	// call-manager confbridge
	CMV1ConfbridgeCreate(ctx context.Context, confbridgeType cmconfbridge.Type) (*cmconfbridge.Confbridge, error)
	CMV1ConfbridgeDelete(ctx context.Context, conferenceID uuid.UUID) error
	CMV1ConfbridgeCallKick(ctx context.Context, conferenceID uuid.UUID, callID uuid.UUID) error
	CMV1ConfbridgeCallAdd(ctx context.Context, conferenceID uuid.UUID, callID uuid.UUID) error

	// call-manager recordings
	CMV1RecordingGet(ctx context.Context, id uuid.UUID) (*cmrecording.Recording, error)
	CMV1RecordingGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]cmrecording.Recording, error)

	// customer-manager customer
	CSV1CustomerCreate(ctx context.Context, requestTimeout int, username, password, name, detail string, webhookMethod cscustomer.WebhookMethod, webhookURI string, permissionIDs []uuid.UUID) (*cscustomer.Customer, error)
	CSV1CustomerDelete(ctx context.Context, id uuid.UUID) (*cscustomer.Customer, error)
	CSV1CustomerGet(ctx context.Context, customerID uuid.UUID) (*cscustomer.Customer, error)
	CSV1CustomerGets(ctx context.Context, pageToken string, pageSize uint64) ([]cscustomer.Customer, error)
	CSV1CustomerUpdate(ctx context.Context, id uuid.UUID, name, detail string, webhookMethod cscustomer.WebhookMethod, webhookURI string) (*cscustomer.Customer, error)
	CSV1CustomerUpdatePassword(ctx context.Context, requestTimeout int, id uuid.UUID, password string) (*cscustomer.Customer, error)
	CSV1CustomerUpdatePermissionIDs(ctx context.Context, id uuid.UUID, permissionIDs []uuid.UUID) (*cscustomer.Customer, error)

	// customer-manager login
	CSV1Login(ctx context.Context, timeout int, username, password string) (*cscustomer.Customer, error)

	// conference-manager conference
	CFV1ConferenceGet(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error)
	CFV1ConferenceGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64, conferenceType string) ([]cfconference.Conference, error)
	CFV1ConferenceCreate(
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
	CFV1ConferenceDelete(ctx context.Context, conferenceID uuid.UUID) error
	CFV1ConferenceDeleteDelay(ctx context.Context, conferenceID uuid.UUID, delay int) error
	CFV1ConferenceKick(ctx context.Context, conferenceID, callID uuid.UUID) error
	CFV1ConferenceUpdate(ctx context.Context, id uuid.UUID, name string, detail string, timeout int, preActions, postActions []fmaction.Action) (*cfconference.Conference, error)

	// flow-manager action
	FMV1ActionGet(ctx context.Context, flowID, actionID uuid.UUID) (*fmaction.Action, error)

	// flow-manager activeflow
	FMV1ActiveflowCreate(ctx context.Context, id, flowID uuid.UUID, referenceType fmactiveflow.ReferenceType, referenceID uuid.UUID) (*fmactiveflow.Activeflow, error)
	FMV1ActiveflowDelete(ctx context.Context, activeflowID uuid.UUID) (*fmactiveflow.Activeflow, error)
	FMV1ActiveflowGetNextAction(ctx context.Context, callID, actionID uuid.UUID) (*fmaction.Action, error)
	FMV1ActiveflowUpdateForwardActionID(ctx context.Context, callID, forwardActionID uuid.UUID, forwardNow bool) error
	FMV1ActiveflowExecute(ctx context.Context, activeflowID uuid.UUID) error

	// flow-manager flow
	FMV1FlowCreate(ctx context.Context, customerID uuid.UUID, flowType fmflow.Type, name string, detail string, actions []fmaction.Action, persist bool) (*fmflow.Flow, error)
	FMV1FlowDelete(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error)
	FMV1FlowGet(ctx context.Context, flowID uuid.UUID) (*fmflow.Flow, error)
	FMV1FlowGets(ctx context.Context, customerID uuid.UUID, flowType fmflow.Type, pageToken string, pageSize uint64) ([]fmflow.Flow, error)
	FMV1FlowUpdate(ctx context.Context, f *fmflow.Flow) (*fmflow.Flow, error)
	FMV1FlowUpdateActions(ctx context.Context, flowID uuid.UUID, actions []fmaction.Action) (*fmflow.Flow, error)

	// message-manager hook
	MMV1Hook(ctx context.Context, hm *hmhook.Hook) error

	// message-manager message
	MMV1MessageGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]mmmessage.Message, error)
	MMV1MessageGet(ctx context.Context, id uuid.UUID) (*mmmessage.Message, error)
	MMV1MessageDelete(ctx context.Context, id uuid.UUID) (*mmmessage.Message, error)
	MMV1MessageSend(ctx context.Context, customerID uuid.UUID, source *cmaddress.Address, destinations []cmaddress.Address, text string) (*mmmessage.Message, error)

	// number-manager available-number
	NMV1AvailableNumberGets(ctx context.Context, customerID uuid.UUID, pageSize uint64, countryCode string) ([]nmavailablenumber.AvailableNumber, error)

	// number-manager number
	NMV1NumberCreate(ctx context.Context, customerID uuid.UUID, num string, callFlowID, messageFlowID uuid.UUID, name, detail string) (*nmnumber.Number, error)
	NMV1NumberDelete(ctx context.Context, id uuid.UUID) (*nmnumber.Number, error)
	NMV1NumberFlowDelete(ctx context.Context, flowID uuid.UUID) error
	NMV1NumberGetByNumber(ctx context.Context, num string) (*nmnumber.Number, error)
	NMV1NumberGet(ctx context.Context, numberID uuid.UUID) (*nmnumber.Number, error)
	NMV1NumberGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]nmnumber.Number, error)
	NMV1NumberUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*nmnumber.Number, error)
	NMV1NumberUpdateFlowID(ctx context.Context, id, callFlowID, messageFlowID uuid.UUID) (*nmnumber.Number, error)

	// outdial-manager outdial
	OMV1OutdialCreate(ctx context.Context, customerID, campaignID uuid.UUID, name, detail, data string) (*omoutdial.Outdial, error)
	OMV1OutdialDelete(ctx context.Context, outdialID uuid.UUID) (*omoutdial.Outdial, error)
	OMV1OutdialGet(ctx context.Context, id uuid.UUID) (*omoutdial.Outdial, error)
	OMV1OutdialGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]omoutdial.Outdial, error)
	OMV1OutdialUpdateBasicInfo(ctx context.Context, outdialID uuid.UUID, name, detail string) (*omoutdial.Outdial, error)
	OMV1OutdialUpdateCampaignID(ctx context.Context, outdialID, campaignID uuid.UUID) (*omoutdial.Outdial, error)
	OMV1OutdialUpdateData(ctx context.Context, outdialID uuid.UUID, data string) (*omoutdial.Outdial, error)

	// outdial-manager outdialtarget
	OMV1OutdialtargetCreate(
		ctx context.Context,
		outdialID uuid.UUID,
		name string,
		detail string,
		data string,
		destination0 *cmaddress.Address,
		destination1 *cmaddress.Address,
		destination2 *cmaddress.Address,
		destination3 *cmaddress.Address,
		destination4 *cmaddress.Address,
	) (*omoutdialtarget.OutdialTarget, error)
	OMV1OutdialtargetDelete(ctx context.Context, outdialtargetID uuid.UUID) (*omoutdialtarget.OutdialTarget, error)
	OMV1OutdialtargetGet(ctx context.Context, outdialtargetID uuid.UUID) (*omoutdialtarget.OutdialTarget, error)
	OMV1OutdialtargetGetsAvailable(
		ctx context.Context,
		outdialID uuid.UUID,
		tryCount0 int,
		tryCount1 int,
		tryCount2 int,
		tryCount3 int,
		tryCount4 int,
		interval int,
		limit int,
	) ([]omoutdialtarget.OutdialTarget, error)
	OMV1OutdialtargetUpdateStatusProgressing(ctx context.Context, outdialtargetID uuid.UUID, destinationIndex int) (*omoutdialtarget.OutdialTarget, error)
	OMV1OutdialtargetUpdateStatus(ctx context.Context, outdialtargetID uuid.UUID, status omoutdialtarget.Status) (*omoutdialtarget.OutdialTarget, error)

	// queue-manager queue
	QMV1QueueGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]qmqueue.Queue, error)
	QMV1QueueGet(ctx context.Context, queueID uuid.UUID) (*qmqueue.Queue, error)
	QMV1QueueGetAgents(ctx context.Context, queueID uuid.UUID, status amagent.Status) ([]amagent.Agent, error)
	QMV1QueueCreate(ctx context.Context, customerID uuid.UUID, name, detail string, routingMethod qmqueue.RoutingMethod, tagIDs []uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) (*qmqueue.Queue, error)
	QMV1QueueDelete(ctx context.Context, queueID uuid.UUID) (*qmqueue.Queue, error)
	QMV1QueueUpdate(ctx context.Context, queueID uuid.UUID, name, detail string) (*qmqueue.Queue, error)
	QMV1QueueUpdateTagIDs(ctx context.Context, queueID uuid.UUID, tagIDs []uuid.UUID) (*qmqueue.Queue, error)
	QMV1QueueUpdateRoutingMethod(ctx context.Context, queueID uuid.UUID, routingMethod qmqueue.RoutingMethod) (*qmqueue.Queue, error)
	QMV1QueueUpdateActions(ctx context.Context, queueID uuid.UUID, waitActions []fmaction.Action, timeoutWait, timeoutService int) (*qmqueue.Queue, error)
	QMV1QueueCreateQueuecall(ctx context.Context, queueID uuid.UUID, referenceType qmqueuecall.ReferenceType, referenceID, referenceActiveflowID, exitActionID uuid.UUID) (*qmqueuecall.Queuecall, error)

	// queue-manager queuecall
	QMV1QueuecallGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]qmqueuecall.Queuecall, error)
	QMV1QueuecallGet(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error)
	QMV1QueuecallDelete(ctx context.Context, queuecallID uuid.UUID) (*qmqueuecall.Queuecall, error)
	QMV1QueuecallDeleteByReferenceID(ctx context.Context, referenceID uuid.UUID) (*qmqueuecall.Queuecall, error)
	QMV1QueuecallExecute(ctx context.Context, queuecallID uuid.UUID, searchDelay int) (*qmqueuecall.Queuecall, error)
	QMV1QueuecallSearchAgent(ctx context.Context, queuecallID uuid.UUID, delay int) error
	QMV1QueuecallTimeoutWait(ctx context.Context, queuecallID uuid.UUID, delay int) error
	QMV1QueuecallTimeoutService(ctx context.Context, queuecallID uuid.UUID, delay int) error

	// queue-manager queuecallreference
	QMV1QueuecallReferenceGet(ctx context.Context, referenceID uuid.UUID) (*qmqueuecallreference.QueuecallReference, error)

	// registrar-manager contact
	RMV1ContactGets(ctx context.Context, endpoint string) ([]*rmastcontact.AstContact, error)
	RMV1ContactUpdate(ctx context.Context, endpoint string) error

	// registrar-manager domain
	RMV1DomainCreate(ctx context.Context, customerID uuid.UUID, domainName, name, detail string) (*rmdomain.Domain, error)
	RMV1DomainDelete(ctx context.Context, domainID uuid.UUID) (*rmdomain.Domain, error)
	RMV1DomainGet(ctx context.Context, domainID uuid.UUID) (*rmdomain.Domain, error)
	RMV1DomainGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]rmdomain.Domain, error)
	RMV1DomainUpdate(ctx context.Context, id uuid.UUID, name, detail string) (*rmdomain.Domain, error)

	// registrar-manager extension
	RMV1ExtensionCreate(ctx context.Context, customerID uuid.UUID, ext, password string, domainID uuid.UUID, name, detail string) (*rmextension.Extension, error)
	RMV1ExtensionDelete(ctx context.Context, extensionID uuid.UUID) (*rmextension.Extension, error)
	RMV1ExtensionGet(ctx context.Context, extensionID uuid.UUID) (*rmextension.Extension, error)
	RMV1ExtensionGets(ctx context.Context, domainID uuid.UUID, pageToken string, pageSize uint64) ([]rmextension.Extension, error)
	RMV1ExtensionUpdate(ctx context.Context, id uuid.UUID, name, detail, password string) (*rmextension.Extension, error)

	// storage: recording
	SMV1RecordingGet(ctx context.Context, id uuid.UUID) (*smbucketrecording.BucketRecording, error)

	// tts-manager speeches
	TMV1SpeecheCreate(ctx context.Context, callID uuid.UUID, text, gender, language string, timeout int) (string, error)

	// transcribe-manager
	TSV1CallRecordingCreate(ctx context.Context, customerID, callID uuid.UUID, language string, timeout, delay int) ([]tstranscribe.Transcribe, error)
	TSV1StreamingCreate(ctx context.Context, customerID, referenceID uuid.UUID, referenceType tstranscribe.Type, language string) (*tstranscribe.Transcribe, error)
	TSV1RecordingCreate(ctx context.Context, customerID, recordingID uuid.UUID, language string) (*tstranscribe.Transcribe, error)

	// user-manager
	UMV1UserCreate(ctx context.Context, timeout int, username, password, name, detail string, permission umuser.Permission) (*umuser.User, error)
	UMV1UserDelete(ctx context.Context, id uint64) error
	UMV1UserGet(ctx context.Context, id uint64) (*umuser.User, error)
	UMV1UserGets(ctx context.Context, pageToken string, pageSize uint64) ([]umuser.User, error)
	UMV1UserLogin(ctx context.Context, timeout int, username, password string) (*umuser.User, error)
	UMV1UserUpdateBasicInfo(ctx context.Context, userID uint64, name, detail string) error
	UMV1UserUpdatePassword(ctx context.Context, timeout int, userID uint64, password string) error
	UMV1UserUpdatePermission(ctx context.Context, userID uint64, permission umuser.Permission) error

	// webhook-manager webhooks
	WMV1WebhookSend(ctx context.Context, customerID uuid.UUID, dataType, messageType string, messageData []byte) error
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
