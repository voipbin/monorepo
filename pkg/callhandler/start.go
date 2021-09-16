package callhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
)

// StasisStart event's context types
const (
	ContextIncomingCall    = "call-in"            // context for the incoming channel
	ContextOutgoingCall    = "call-out"           // context for the outgoing channel
	ContextRecording       = "call-record"        // context for the channel which created only for recording
	ContextFromServiceCall = "call-svc"           // context for the channel where it came back to stasis from the other asterisk application
	ContextJoinCall        = "call-join"          // context for the channel for conference joining
	ContextExternalMedia   = "call-externalmedia" // context for the external media channel. this channel will get the media from the external
	ContextExternalSoop    = "call-externalsnoop" // context for the external snoop channel
)

// domains
const (
	domainConference  = "conference.voipbin.net"
	domainSIPService  = "sip-service.voipbin.net"
	domainPSTNService = "pstn.voipbin.net"
)

// list of domain types
const (
	domainTypeNone       = "none"
	domainTypeConference = "conference"
	domainTypeSIPService = "sip"
	domainTypePSTN       = "pstn"
)

// pjsip endpoints
const (
	pjsipEndpointOutgoing = "call-out"
)

// fixed trunks
const (
	trunkTwilio = "voipbin.pstn.twilio.com"
	trunkTelnyx = "sip.telnyx.com"
)

// default max timeout for each services. sec.
const (
	defaultMaxTimeoutEcho       = "300"   // maximum call duration for service echo. 5 min
	defaultMaxTimeoutConference = "10800" // maximum call duration for service conf-soft. 3 hours
	defaultMaxTimeoutSipService = "300"   // maximum call duration for service sip-service. 5 min
	defaultMaxTimeoutFlow       = "3600"  // maximum call duration for service flow. 1 hour
)

// default sip service option variables
const (
	DefaultSipServiceOptionConferenceID = "037a20b9-d11d-4b63-a135-ae230cafd495" // default conference ID for conference@sip-service
)

// createCall create a call record. All of call creation process need to use this.
func (h *callHandler) createCall(ctx context.Context, c *call.Call) (*call.Call, error) {

	// set default time stamp
	c.TMUpdate = defaultTimeStamp
	c.TMRinging = defaultTimeStamp
	c.TMProgressing = defaultTimeStamp
	c.TMHangup = defaultTimeStamp

	if err := h.db.CallCreate(ctx, c); err != nil {
		return nil, err
	}
	promCallCreateTotal.WithLabelValues(string(c.Direction), string(c.Type)).Inc()

	res, err := h.db.CallGet(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	h.notifyHandler.NotifyEvent(notifyhandler.EventTypeCallCreated, res)

	return res, nil
}

// StartCallHandle starts the call handle service
func (h *callHandler) StartCallHandle(cn *channel.Channel, data map[string]interface{}) error {

	// check the stasis's context
	chCtx, ok := data["context"]
	if !ok {
		logrus.Errorf("Could not get channel context. data: %v", data)
		return fmt.Errorf("no context found")
	}

	switch chCtx.(string) {

	case ContextFromServiceCall:
		return h.startHandlerContextFromServiceCall(cn, data)

	case ContextIncomingCall:
		return h.startHandlerContextIncomingCall(cn, data)

	case ContextOutgoingCall:
		return h.startHandlerContextOutgoingCall(cn, data)

	case ContextRecording:
		return h.startHandlerContextRecording(cn, data)

	case ContextExternalSoop:
		return h.startHandlerContextExternalSnoop(cn, data)

	case ContextExternalMedia:
		return h.startHandlerContextExternalMedia(cn, data)

	case ContextJoinCall:
		return h.startHandlerContextJoin(cn, data)

	default:
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		return fmt.Errorf("no route found for stasisstart. asterisk_id: %s, channel_id: %s, data: %v", cn.AsteriskID, cn.ID, data)
	}
}

// startHandlerContextFromServiceCall handles contextFromServiceCall context type of StasisStart event.
func (h *callHandler) startHandlerContextFromServiceCall(cn *channel.Channel, data map[string]interface{}) error {
	logrus.Infof("Executing startHandlerContextFromServiceCall. channel: %s", cn.ID)

	// get call by the channel id
	c, err := h.db.CallGetByChannelID(context.Background(), cn.ID)
	if err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		return err
	}

	return h.ActionNext(c)
}

// startHandlerContextRecording handles contextFromServiceCall context type of StasisStart event.
func (h *callHandler) startHandlerContextRecording(cn *channel.Channel, data map[string]interface{}) error {
	logrus.Infof("Executing startHandlerContextRecording. channel: %s", cn.ID)

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeRecording)); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	id := data["recording_id"].(string)
	name := data["recording_name"].(string)
	format := data["format"].(string)
	duration, _ := strconv.Atoi(data["duration"].(string))
	silence, _ := strconv.Atoi(data["end_of_silence"].(string))
	endKey := data["end_of_key"].(string)
	callID := data["call_id"].(string)

	if err := h.reqHandler.AstChannelRecord(cn.AsteriskID, cn.ID, name, format, duration, silence, false, endKey, "fail"); err != nil {
		logrus.Errorf("Could not start the recording. Destorying the chanel. err: %v", err)

		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
	}
	logrus.Infof("Recording started. id: %s, name: %s, call: %s", id, name, callID)

	return nil
}

// startHandlerContextExternalSnoop handles contextExternalSnoop context type of StasisStart event.
func (h *callHandler) startHandlerContextExternalSnoop(cn *channel.Channel, data map[string]interface{}) error {
	log := logrus.WithFields(
		logrus.Fields{
			"channel": cn.ID,
		},
	)
	log.Infof("Executing startHandlerContextExternalSnoop. channel: %s", cn.ID)

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeExternal)); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	callID := data["call_id"].(string)
	bridgeID := data["bridge_id"].(string)
	log.Debugf("Parsed info. call: %s, bridge: %s", callID, bridgeID)

	// put the channel to the bridge
	if err := h.reqHandler.AstBridgeAddChannel(cn.AsteriskID, bridgeID, cn.ID, "", false, false); err != nil {
		log.Errorf("Could not add the external snoop channel to the bridge. err: %v", err)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return err
	}

	return nil
}

// startHandlerContextExternalMedia handles contextExternalMedia context type of StasisStart event.
func (h *callHandler) startHandlerContextExternalMedia(cn *channel.Channel, data map[string]interface{}) error {
	log := logrus.WithFields(
		logrus.Fields{
			"channel": cn.ID,
		},
	)
	log.Infof("Executing startHandlerContextExternalMedia. channel: %s", cn.ID)

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeExternal)); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	callID := data["call_id"].(string)
	bridgeID := data["bridge_id"].(string)
	log.Debugf("Parsed info. call: %s, bridge: %s", callID, bridgeID)

	// put the channel to the bridge
	if err := h.reqHandler.AstBridgeAddChannel(cn.AsteriskID, bridgeID, cn.ID, "", false, false); err != nil {
		log.Errorf("Could not add the external snoop channel to the bridge. err: %v", err)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return err
	}

	return nil
}

// startHandlerContextJoin handles contextJoinCall context type of StasisStart event.
func (h *callHandler) startHandlerContextJoin(cn *channel.Channel, data map[string]interface{}) error {
	log := logrus.WithFields(
		logrus.Fields{
			"channel": cn.ID,
		},
	)
	log.Infof("Executing startHandlerContextJoin. channel: %s", cn.ID)

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeJoin)); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	callID := data["call_id"].(string)
	bridgeID := data["bridge_id"].(string)
	log.Debugf("Parsed info. call: %s, bridge: %s", callID, bridgeID)

	// put the channel to the bridge
	if err := h.reqHandler.AstBridgeAddChannel(cn.AsteriskID, bridgeID, cn.ID, "", false, false); err != nil {
		log.Errorf("Could not add the external snoop channel to the bridge. err: %v", err)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return err
	}

	// dial to the destination
	if err := h.reqHandler.AstChannelDial(cn.AsteriskID, cn.ID, "", defaultDialTimeout); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated)
		return fmt.Errorf("could not dial the channel. id: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	return nil
}

// startHandlerContextIncomingCall handles contextIncomingCall context type of StasisStart event.
func (h *callHandler) startHandlerContextIncomingCall(cn *channel.Channel, data map[string]interface{}) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startHandlerContextIncomingCall",
		"channel_id": cn.ID,
	})
	log.Infof("Executing startHandlerContextIncomingCall. data: %v", data)

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeCall)); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// get call type
	domainType := getDomainTypeIncomingCall(data["domain"].(string))

	switch domainType {
	case domainTypeConference:
		return h.typeConferenceStart(cn, data)

	case domainTypeSIPService:
		return h.typeSipServiceStart(cn, data)

	case domainTypePSTN:
		return h.typeFlowStart(cn, data)

	default:
		// call.TypeNone will get to here.
		// no route found
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		log.Errorf("Could not find correct domain type handler. domain_type: %v", domainType)
		return fmt.Errorf("no route found for stasisstart. asterisk_id: %s, channel_id: %s", cn.AsteriskID, cn.ID)
	}
}

// startHandlerContextOutgoingCall handles contextOutgoingCall context type of StasisStart event.
func (h *callHandler) startHandlerContextOutgoingCall(cn *channel.Channel, data map[string]interface{}) error {
	logrus.Infof("Executing startHandlerContextOutgoingCall. channel: %s, data: %v", cn.ID, data)

	// get
	callID := uuid.FromStringOrNil(data["call_id"].(string))
	if callID == uuid.Nil {
		return fmt.Errorf("could not get correct call_id. channel: %s, asterisk: %s", cn.ID, cn.AsteriskID)
	}

	// update call's asterisk id
	if err := h.db.CallSetAsteriskID(context.Background(), callID, cn.AsteriskID, getCurTime()); err != nil {
		return fmt.Errorf("could not set asterisk id to call. channel: %s, asterisk: %s", cn.ID, cn.AsteriskID)
	}

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeCall)); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// create call bridge
	bridgeID, err := h.addCallBridge(cn, bridge.ReferenceTypeCall, callID)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		return fmt.Errorf("could not add the channel to the join bridge. err: %v", err)
	}

	ctx := context.Background()
	if err := h.db.CallSetBridgeID(ctx, callID, bridgeID); err != nil {
		log.Errorf("could not set call bridge id. err: %v", err)
		h.reqHandler.AstBridgeDelete(cn.AsteriskID, bridgeID)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		return fmt.Errorf("could not set call bridge id. err: %v", err)
	}

	if err := h.reqHandler.AstChannelDial(cn.AsteriskID, cn.ID, cn.ID, defaultDialTimeout); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// do nothing here
	return nil
}

// addCallBridge creates a join bridge and put the channel into the join bridge.
func (h *callHandler) addCallBridge(cn *channel.Channel, referenceType bridge.ReferenceType, referenceID uuid.UUID) (string, error) {
	log := logrus.WithFields(log.Fields{
		"func":    "addJoinBridge",
		"channel": cn,
	})

	// create join bridge
	bridgeID := uuid.Must(uuid.NewV4())
	bridgeName := fmt.Sprintf("reference_type=%s,reference_id=%s", referenceType, referenceID)
	if errBridge := h.reqHandler.AstBridgeCreate(cn.AsteriskID, bridgeID.String(), bridgeName, []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}); errBridge != nil {
		log.Errorf("Could not create a bridge for external media. error: %v", errBridge)
		return "", errBridge
	}

	// add the channel to the bridge
	if errAddCh := h.reqHandler.AstBridgeAddChannel(cn.AsteriskID, bridgeID.String(), cn.ID, "", false, false); errAddCh != nil {
		log.Errorf("Could not add the channel to the join bridge. error: %v", errAddCh)
		h.reqHandler.AstBridgeDelete(cn.AsteriskID, bridgeID.String())
		return "", errAddCh
	}

	return bridgeID.String(), nil
}

// getDomainTypeIncomingCall returns the service type for incoming call context
func getDomainTypeIncomingCall(domain string) string {
	// all of the incoming calls are hitting the same context.
	// so we have to distinguish them using the requested domain name.
	switch domain {
	case domainConference:
		return domainTypeConference

	case domainSIPService:
		return domainTypeSIPService

	case domainPSTNService:
		return domainTypePSTN

	default:
		return domainTypeNone
	}
}

// serviceConferenceStart handles conference calltype start.
func (h *callHandler) typeConferenceStart(cn *channel.Channel, data map[string]interface{}) error {
	ctx := context.Background()
	cfID := uuid.FromStringOrNil(cn.DestinationNumber)

	log := log.WithFields(
		log.Fields{
			"channel":    cn.ID,
			"asterisk":   cn.AsteriskID,
			"conference": cfID,
		})
	log.Debugf("Starting the conference to joining. source: %s", cn.SourceNumber)

	// get conference info
	cf, err := h.db.ConferenceGet(ctx, cfID)
	if err != nil {
		log.Debug("The conference has not created.")
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return err
	}

	// Set absolute timeout for conference
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "TIMEOUT(absolute)", defaultMaxTimeoutConference); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a timeout for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// generate call info
	tmpCall := call.NewCallByChannel(cn, cf.UserID, call.TypeFlow, call.DirectionIncoming, data)

	// create call bridge
	callBridgeID, err := h.addCallBridge(cn, bridge.ReferenceTypeCall, tmpCall.ID)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not add the channel to the join bridge. err: %v", err)
	}
	tmpCall.BridgeID = callBridgeID

	// create a call
	c, err := h.createCall(ctx, tmpCall)
	if err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		h.reqHandler.AstBridgeDelete(cn.AsteriskID, callBridgeID)
		return fmt.Errorf("Could not create a call for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}
	log = log.WithFields(
		logrus.Fields{
			"call":      c.ID,
			"type":      c.Type,
			"direction": c.Direction,
		})
	log.Debug("The call has created.")

	// set flowid
	if err := h.db.CallSetFlowID(ctx, c.ID, uuid.Nil); err != nil {
		h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a flow id for call. call: %s, err: %v", c.ID, err)
	}

	// create an option for action echo
	// create default option for echo
	option := action.OptionConferenceJoin{
		ConferenceID: cf.ID.String(),
	}
	opt, err := json.Marshal(option)
	if err != nil {
		h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not marshal the option. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// create an action
	action := &action.Action{
		ID:     action.IDStart,
		Type:   action.TypeConferenceJoin,
		Option: opt,
	}

	c, err = h.db.CallGet(ctx, c.ID)
	if err != nil {
		h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not get created call info. channel: %s, asterisk: %s, call: %s, err: %v", cn.ID, cn.AsteriskID, c.ID, err)
	}

	// execute action
	if err := h.ActionExecute(c, action); err != nil {
		log.Errorf("Could not execte the action. Hanging up the call. action: %s", action.Type)
		h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not get execute the action. channel: %s, asterisk: %s, call: %s, err: %v", cn.ID, cn.AsteriskID, c.ID, err)
	}

	return nil
}

// typeFlowStart handles flow calltype start.
func (h *callHandler) typeFlowStart(cn *channel.Channel, data map[string]interface{}) error {
	ctx := context.Background()

	log := log.WithFields(
		log.Fields{
			"channel":  cn.ID,
			"asterisk": cn.AsteriskID,
		})
	log.Debugf("Starting the flow incoming call handler. source: %s, destinaiton: %s", cn.SourceNumber, cn.DestinationNumber)

	// set absolute timeout for 3600 sec(1 hour)
	log.Debugf("Setting absolute timeout for flow type call")
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "TIMEOUT(absolute)", defaultMaxTimeoutFlow); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a timeout for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// get number info
	numb, err := h.reqHandler.NMV1NumbersNumberGet(cn.DestinationNumber)
	if err != nil {
		log.Debugf("Could not find number info. err: %v", err)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		return fmt.Errorf("could not get a number info by the destination. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// create a temp call info
	tmpCall := call.NewCallByChannel(cn, call.UserIDAdmin, call.TypeSipService, call.DirectionIncoming, data)

	// create call bridge
	callBridgeID, err := h.addCallBridge(cn, bridge.ReferenceTypeCall, tmpCall.ID)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		return fmt.Errorf("could not add the channel to the join bridge. err: %v", err)
	}
	tmpCall.FlowID = numb.FlowID
	tmpCall.BridgeID = callBridgeID

	// create active flow
	af, err := h.reqHandler.FlowActvieFlowPost(tmpCall.ID, numb.FlowID)
	if err != nil {
		af = &activeflow.ActiveFlow{}
		log.Errorf("Could not get an active flow info. Created dummy active flow. This call will be hungup. call: %s, flow: %s", tmpCall.ID, tmpCall.FlowID)
	}
	log.Debugf("Created an active flow. active-flow: %v", af)
	tmpCall.WebhookURI = af.WebhookURI
	tmpCall.Action = af.CurrentAction

	c, err := h.createCall(ctx, tmpCall)
	if err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		h.reqHandler.AstBridgeDelete(cn.AsteriskID, callBridgeID)
		return fmt.Errorf("Could not create a call. call: %s, err: %v", c.ID, err)
	}
	log = log.WithFields(
		logrus.Fields{
			"call": c,
		})
	log.Debugf("Created a call. call: %s", c.ID)

	return h.ActionNext(c)
}

// typeSipServiceStart handles sip-service calltype request.
func (h *callHandler) typeSipServiceStart(cn *channel.Channel, data map[string]interface{}) error {
	ctx := context.Background()

	log := log.WithFields(
		log.Fields{
			"channel":  cn.ID,
			"asterisk": cn.AsteriskID,
		})
	log.Debugf("Starting the sip-service. source: %s, destinaiton: %s", cn.SourceNumber, cn.DestinationNumber)

	// set absolute timeout for 300 sec
	log.Debugf("Setting absolute timeout for sip-service type call")
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "TIMEOUT(absolute)", defaultMaxTimeoutSipService); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a timeout for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// generate a call info
	tmpCall := call.NewCallByChannel(cn, call.UserIDAdmin, call.TypeSipService, call.DirectionIncoming, data)

	// create call bridge
	callBridgeID, err := h.addCallBridge(cn, bridge.ReferenceTypeCall, tmpCall.ID)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not add the channel to the join bridge. err: %v", err)
	}
	tmpCall.BridgeID = callBridgeID

	// create a call
	c, err := h.createCall(ctx, tmpCall)
	if err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		h.reqHandler.AstBridgeDelete(cn.AsteriskID, callBridgeID)
		return fmt.Errorf("Could not create a call for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}
	log = log.WithFields(
		logrus.Fields{
			"call":      c.ID,
			"type":      c.Type,
			"direction": c.Direction,
		})
	log.Debug("Created a call.")

	// set flowid
	// because this call type support only 1 action, we don't set any valid call-flow id here
	if err := h.db.CallSetFlowID(ctx, c.ID, uuid.Nil); err != nil {
		h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a flow id for call. call: %s, err: %v", c.ID, err)
	}

	// get action for sip-service
	act, err := h.getSipServiceAction(ctx, c, cn)
	if err != nil {
		h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not get action handle for sip-service. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// get updated call info
	c, err = h.db.CallGet(ctx, c.ID)
	if err != nil {
		h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not get created call info. channel: %s, asterisk: %s, call: %s, err: %v", cn.ID, cn.AsteriskID, c.ID, err)
	}

	// execute action
	if err := h.ActionExecute(c, act); err != nil {
		log.Errorf("Could not execte the action. Hanging up the call. action: %s", act.Type)
		h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not get execute the action. channel: %s, asterisk: %s, call: %s, err: %v", cn.ID, cn.AsteriskID, c.ID, err)
	}

	return nil
}

// getSipServiceAction returns sip-service action handler by the call's destination.
func (h *callHandler) getSipServiceAction(ctx context.Context, c *call.Call, cn *channel.Channel) (*action.Action, error) {
	logrus.Debugf("Executing action for sip-service. call: %s, channel: %s, destination: %s", c.ID, cn.ID, cn.DestinationNumber)

	var resAct *action.Action = nil
	switch c.Destination.Target {

	// answer
	case string(action.TypeAnswer):
		// create an option for action answer
		option := action.OptionAnswer{}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("Could not marshal the option. action: %s, err: %v", action.TypeAnswer, err)
		}

		// create an action
		resAct = &action.Action{
			ID:     action.IDStart,
			Type:   action.TypeAnswer,
			Option: opt,
		}

	// conference_join
	case string(action.TypeConferenceJoin):
		option := action.OptionConferenceJoin{
			ConferenceID: DefaultSipServiceOptionConferenceID,
		}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("Could not marshal the option. action: %s, err: %v", action.TypeConferenceJoin, err)
		}

		// create an action
		resAct = &action.Action{
			ID:     action.IDStart,
			Type:   action.TypeConferenceJoin,
			Option: opt,
		}

	// default
	default:
		logrus.Warnf("Could not find correct sip-service handler. Use default handler. target: %s", c.Destination.Target)
		fallthrough

	// echo
	case string(action.TypeEcho):
		// create default option for echo
		option := action.OptionEcho{
			Duration: 180 * 1000, // duration 180 sec
		}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("Could not marshal the option echo. action: %s, err: %v", action.TypeEcho, err)
		}

		// create an action
		resAct = &action.Action{
			ID:     action.IDStart,
			Type:   action.TypeEcho,
			Option: opt,
		}

	// play
	case string(action.TypePlay):
		// answer the call first
		if err := h.reqHandler.AstChannelAnswer(c.AsteriskID, c.ChannelID); err != nil {
			return nil, fmt.Errorf("could not answer the call. err: %v", err)
		}

		option := action.OptionPlay{
			StreamURLs: []string{"https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"},
		}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("Could not marshal the option. action: %s, err: %v", action.TypePlay, err)
		}

		// create an action
		resAct = &action.Action{
			ID:     action.IDStart,
			Type:   action.TypePlay,
			Option: opt,
		}

	// stream_echo
	case string(action.TypeStreamEcho):
		option := action.OptionStreamEcho{
			Duration: 180 * 1000,
		}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("Could not marshal the option. action: %s, err: %v", action.TypeStreamEcho, err)
		}

		// create an action
		resAct = &action.Action{
			ID:     action.IDStart,
			Type:   action.TypeStreamEcho,
			Option: opt,
		}

	}

	return resAct, nil
}
