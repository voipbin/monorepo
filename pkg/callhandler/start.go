package callhandler

import (
	"context"
	"fmt"
	"strconv"

	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// StasisStart event's context types
const (
	ContextIncomingCall  = "call-in"            // context for the incoming channel
	ContextOutgoingCall  = "call-out"           // context for the outgoing channel
	ContextRecording     = "call-record"        // context for the channel which created only for recording
	ContextServiceCall   = "call-svc"           // context for the channel where it came back to stasis from the other asterisk application
	ContextJoinCall      = "call-join"          // context for the channel for conference joining
	ContextExternalMedia = "call-externalmedia" // context for the external media channel. this channel will get the media from the external
	ContextExternalSoop  = "call-externalsnoop" // context for the external snoop channel
	ContextApplication   = "call-application"   // context for dialplan application execution
)

// list of application name
const (
	applicationAMD = "amd"
)

// domains
const (
	domainConference  = "conference.voipbin.net"
	domainPSTNService = "pstn.voipbin.net"
)

// list of domain types
const (
	domainTypeNone       = "none"
	domainTypeConference = "conference"
	domainTypePSTN       = "pstn"
)

// pjsip endpoints
const (
	pjsipEndpointOutgoing = "call-out"
)

// fixed trunks
// const (
// trunkTwilio = "voipbin.pstn.twilio.com" //nolint:varcheck,deadcode // this is ok
// trunkTelnyx = "sip.telnyx.com"
// )

// default sip service option variables
const (
	DefaultSipServiceOptionConfbridgeID = "037a20b9-d11d-4b63-a135-ae230cafd495" // default conference ID for conference@sip-service
)

// StartCallHandle starts the call handle service
func (h *callHandler) StartCallHandle(ctx context.Context, cn *channel.Channel, data map[string]string) error {

	// check the stasis's context
	chCtx, ok := data["context"]
	if !ok {
		logrus.Errorf("Could not get channel context. data: %v", data)
		return fmt.Errorf("no context found")
	}

	switch chCtx {

	case ContextServiceCall:
		return h.startHandlerContextFromServiceCall(ctx, cn, data)

	case ContextIncomingCall:
		return h.startHandlerContextIncomingCall(ctx, cn, data)

	case ContextOutgoingCall:
		return h.startHandlerContextOutgoingCall(cn, data)

	case ContextRecording:
		return h.startHandlerContextRecording(ctx, cn, data)

	case ContextExternalSoop:
		return h.startHandlerContextExternalSnoop(ctx, cn, data)

	case ContextExternalMedia:
		return h.startHandlerContextExternalMedia(ctx, cn, data)

	case ContextJoinCall:
		return h.startHandlerContextJoin(ctx, cn, data)

	case ContextApplication:
		return h.startHandlerContextApplication(ctx, cn, data)

	default:
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination, 0)
		return fmt.Errorf("no route found for stasisstart. asterisk_id: %s, channel_id: %s, data: %v", cn.AsteriskID, cn.ID, data)
	}
}

// startHandlerContextFromServiceCall handles contextFromServiceCall context type of StasisStart event.
func (h *callHandler) startHandlerContextFromServiceCall(ctx context.Context, cn *channel.Channel, data map[string]string) error {
	log := logrus.WithFields(logrus.Fields{
		"asterisk_id": cn.AsteriskID,
		"channel_id":  cn.ID,
		"func":        "startHandlerContextFromServiceCall",
	})
	log.Infof("Executing startHandlerContextFromServiceCall. context: %s", data["context"])

	fromContext := data["context_from"]
	switch fromContext {
	case serviceContextAMD:
		return h.startServiceFromAMD(ctx, cn, data)
	default:
		return h.startServiceFromDefault(ctx, cn, data)
	}
}

// startHandlerContextRecording handles contextFromServiceCall context type of StasisStart event.
func (h *callHandler) startHandlerContextRecording(ctx context.Context, cn *channel.Channel, data map[string]string) error {
	logrus.Infof("Executing startHandlerContextRecording. channel: %s", cn.ID)

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(ctx, cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeRecording)); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	id := data["recording_id"]
	name := data["recording_name"]
	format := data["format"]
	duration, _ := strconv.Atoi(data["duration"])
	silence, _ := strconv.Atoi(data["end_of_silence"])
	endKey := data["end_of_key"]
	callID := data["call_id"]

	if err := h.reqHandler.AstChannelRecord(ctx, cn.AsteriskID, cn.ID, name, format, duration, silence, false, endKey, "fail"); err != nil {
		logrus.Errorf("Could not start the recording. Destorying the chanel. err: %v", err)

		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
	}
	logrus.Infof("Recording started. id: %s, name: %s, call: %s", id, name, callID)

	return nil
}

// startHandlerContextExternalSnoop handles contextExternalSnoop context type of StasisStart event.
func (h *callHandler) startHandlerContextExternalSnoop(ctx context.Context, cn *channel.Channel, data map[string]string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"channel": cn.ID,
		},
	)
	log.Infof("Executing startHandlerContextExternalSnoop. channel: %s", cn.ID)

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(ctx, cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeExternal)); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	callID := data["call_id"]
	bridgeID := data["bridge_id"]
	log.Debugf("Parsed info. call: %s, bridge: %s", callID, bridgeID)

	// put the channel to the bridge
	if err := h.reqHandler.AstBridgeAddChannel(ctx, cn.AsteriskID, bridgeID, cn.ID, "", false, false); err != nil {
		log.Errorf("Could not add the external snoop channel to the bridge. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return err
	}

	return nil
}

// startHandlerContextExternalMedia handles contextExternalMedia context type of StasisStart event.
func (h *callHandler) startHandlerContextExternalMedia(ctx context.Context, cn *channel.Channel, data map[string]string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"channel": cn.ID,
		},
	)
	log.Infof("Executing startHandlerContextExternalMedia. channel: %s", cn.ID)

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(ctx, cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeExternal)); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	callID := data["call_id"]
	bridgeID := data["bridge_id"]
	log.Debugf("Parsed info. call: %s, bridge: %s", callID, bridgeID)

	// put the channel to the bridge
	if err := h.reqHandler.AstBridgeAddChannel(ctx, cn.AsteriskID, bridgeID, cn.ID, "", false, false); err != nil {
		log.Errorf("Could not add the external snoop channel to the bridge. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return err
	}

	return nil
}

// startHandlerContextJoin handles contextJoinCall context type of StasisStart event.
func (h *callHandler) startHandlerContextJoin(ctx context.Context, cn *channel.Channel, data map[string]string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"channel": cn.ID,
		},
	)
	log.Infof("Executing startHandlerContextJoin. channel: %s", cn.ID)

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(ctx, cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeJoin)); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	confbridgeID := data["confbridge_id"]
	callID := data["call_id"]
	bridgeID := data["bridge_id"]
	log.Debugf("Parsed info. call_id: %s, bridge_id: %s, confbridge_id: %s", callID, bridgeID, confbridgeID)

	// put the channel to the bridge
	if err := h.reqHandler.AstBridgeAddChannel(ctx, cn.AsteriskID, bridgeID, cn.ID, "", false, false); err != nil {
		log.Errorf("Could not add the external snoop channel to the bridge. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return err
	}

	// // set sip header
	// if errSet := h.reqHandler.AstChannelVariableSet(ctx, cn.AsteriskID, cn.ID, "PJSIP_HEADER(add,VB-CALL-ID)", callID); errSet != nil {
	// 	log.Errorf("Could not set sip header. err: %v", errSet)
	// 	return errSet
	// }
	// if errSet := h.reqHandler.AstChannelVariableSet(ctx, cn.AsteriskID, cn.ID, "PJSIP_HEADER(add,VB-CONFBRIDGE-ID)", confbridgeID); errSet != nil {
	// 	log.Errorf("Could not set sip header. err: %v", errSet)
	// 	return errSet
	// }

	// dial to the destination
	if err := h.reqHandler.AstChannelDial(ctx, cn.AsteriskID, cn.ID, "", defaultDialTimeout); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseUnallocated, 0)
		return fmt.Errorf("could not dial the channel. id: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	return nil
}

// startHandlerContextIncomingCall handles contextIncomingCall context type of StasisStart event.
func (h *callHandler) startHandlerContextIncomingCall(ctx context.Context, cn *channel.Channel, data map[string]string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startHandlerContextIncomingCall",
		"channel_id": cn.ID,
	})
	log.Infof("Executing startHandlerContextIncomingCall. data: %v", data)

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(ctx, cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeCall)); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// set the call durationtimeout
	if err := h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration); err != nil {
		// could not send the delayed hangup request.
		// but we don't do anything here. just write log.
		log.Errorf("Could not send the channel hangup request for callprogress timeout. err: %v", err)
	}

	// get call type
	domainType := getDomainTypeIncomingCall(data["domain"])

	switch domainType {
	case domainTypeConference:
		return h.typeConferenceStart(ctx, cn, data)

	case domainTypePSTN:
		return h.typeFlowStart(ctx, cn, data)

	default:
		// call.TypeNone will get to here.
		// no route found
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination, 0)
		log.Errorf("Could not find correct domain type handler. domain_type: %v", domainType)
		return fmt.Errorf("no route found for stasisstart. asterisk_id: %s, channel_id: %s", cn.AsteriskID, cn.ID)
	}
}

// startHandlerContextOutgoingCall handles contextOutgoingCall context type of StasisStart event.
func (h *callHandler) startHandlerContextOutgoingCall(cn *channel.Channel, data map[string]string) error {
	ctx := context.Background()

	log := logrus.WithFields(logrus.Fields{
		"func":       "startHandlerContextOutgoingCall",
		"channel_id": cn.ID,
	})
	log.Infof("Executing startHandlerContextOutgoingCall. channel: %s, data: %v", cn.ID, data)

	// get
	callID := uuid.FromStringOrNil(data["call_id"])
	if callID == uuid.Nil {
		log.Errorf("Could not get call_id info.")
		return fmt.Errorf("could not get correct call_id. channel: %s, asterisk: %s", cn.ID, cn.AsteriskID)
	}
	log = log.WithField("call_id", callID.String())

	// update call's asterisk id
	if err := h.db.CallSetAsteriskID(context.Background(), callID, cn.AsteriskID, h.util.GetCurTime()); err != nil {
		log.Errorf("Could not set call id to the channel. err: %v", err)
		return fmt.Errorf("could not set asterisk id to call. channel: %s, asterisk: %s", cn.ID, cn.AsteriskID)
	}

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(ctx, cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeCall)); err != nil {
		log.Errorf("Could not set channel's type. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// set the call durationtimeout
	if err := h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration); err != nil {
		// could not send the delayed hangup request.
		// but we don't do anything here. just write log.
		log.Errorf("Could not send the channel hangup request for callprogress timeout. err: %v", err)
	}

	// create call bridge
	bridgeID, err := h.addCallBridge(ctx, cn, bridge.ReferenceTypeCall, callID)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination, 0)
		return fmt.Errorf("could not add the channel to the join bridge. err: %v", err)
	}

	if err := h.db.CallSetBridgeID(ctx, callID, bridgeID); err != nil {
		log.Errorf("could not set call bridge id. err: %v", err)
		_ = h.reqHandler.AstBridgeDelete(ctx, cn.AsteriskID, bridgeID)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination, 0)
		return fmt.Errorf("could not set call bridge id. err: %v", err)
	}

	if errDial := h.reqHandler.AstChannelDial(ctx, cn.AsteriskID, cn.ID, cn.ID, defaultDialTimeout); errDial != nil {
		log.Errorf("Could not dial the channel. err: %v", errDial)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, errDial)
	}

	// do nothing here
	return nil
}

// startHandlerContextApplication handles contextApplication context type of StasisStart event.
func (h *callHandler) startHandlerContextApplication(ctx context.Context, cn *channel.Channel, data map[string]string) error {
	log := logrus.WithFields(logrus.Fields{
		"channel_id": cn.ID,
		"func":       "startHandlerContextApplication",
	})

	appName := data["application_name"]
	log.Debugf("Parsed application info. application: %s", appName)

	switch appName {
	case applicationAMD:
		return h.applicationHandleAMD(ctx, cn, data)

	default:
		log.Errorf("Could not find correct event handler. app_name: %s", appName)
	}

	return nil
}

// addCallBridge creates a join bridge and put the channel into the join bridge.
func (h *callHandler) addCallBridge(ctx context.Context, cn *channel.Channel, referenceType bridge.ReferenceType, referenceID uuid.UUID) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "addJoinBridge",
		"channel": cn,
	})

	// create join bridge
	bridgeID := h.util.CreateUUID()
	bridgeName := fmt.Sprintf("reference_type=%s,reference_id=%s", referenceType, referenceID)
	if errBridge := h.reqHandler.AstBridgeCreate(ctx, cn.AsteriskID, bridgeID.String(), bridgeName, []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}); errBridge != nil {
		log.Errorf("Could not create a bridge for external media. error: %v", errBridge)
		return "", errBridge
	}

	// add the channel to the bridge
	if errAddCh := h.reqHandler.AstBridgeAddChannel(ctx, cn.AsteriskID, bridgeID.String(), cn.ID, "", false, false); errAddCh != nil {
		log.Errorf("Could not add the channel to the join bridge. error: %v", errAddCh)
		_ = h.reqHandler.AstBridgeDelete(ctx, cn.AsteriskID, bridgeID.String())
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

	case domainPSTNService:
		return domainTypePSTN

	default:
		return domainTypeNone
	}
}

// serviceConferenceStart handles conference calltype start.
func (h *callHandler) typeConferenceStart(ctx context.Context, cn *channel.Channel, data map[string]string) error {
	cfID := uuid.FromStringOrNil(cn.DestinationNumber)

	log := logrus.WithFields(
		logrus.Fields{
			"channel":    cn.ID,
			"asterisk":   cn.AsteriskID,
			"conference": cfID,
		})
	log.Debugf("Starting the conference to joining. source: %s", cn.SourceNumber)

	id := h.util.CreateUUID()
	log = log.WithField("call_id", id)

	// get conference info
	cf, err := h.reqHandler.ConferenceV1ConferenceGet(ctx, cfID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return err
	}

	callBridgeID, err := h.addCallBridge(ctx, cn, bridge.ReferenceTypeCall, id)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return fmt.Errorf("could not add the channel to the join bridge. err: %v", err)
	}

	// create active flow
	af, err := h.reqHandler.FlowV1ActiveflowCreate(ctx, uuid.Nil, cf.FlowID, fmactiveflow.ReferenceTypeCall, id)
	if err != nil {
		log.Errorf("Could not create active flow. call: %s, flow: %s", id, cf.FlowID)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return errors.Wrap(err, "could not create an active flow")
	}
	log.WithField("activeflow", af).Debugf("Created an active flow. active_flow_id: %s", af.ID)

	source := commonaddress.CreateAddressByChannelSource(cn)
	destination := commonaddress.CreateAddressByChannelDestination(cn)
	status := call.GetStatusByChannelState(cn.State)
	log.WithFields(logrus.Fields{
		"source":      source,
		"destination": destination,
		"status":      status,
	}).Debug("Parsed address and status info.")

	c, err := h.Create(
		ctx,

		id,
		cf.CustomerID,

		cn.AsteriskID,
		cn.ID,
		callBridgeID,

		cf.FlowID,
		af.ID,
		uuid.Nil,
		call.TypeFlow,

		uuid.Nil,
		[]uuid.UUID{},

		uuid.Nil,
		[]uuid.UUID{},

		source,
		destination,

		status,

		data,
		af.CurrentAction,
		call.DirectionIncoming,

		uuid.Nil,
		[]rmroute.Route{},
	)
	if err != nil {
		log.Errorf("Could not create a call info. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		_ = h.reqHandler.AstBridgeDelete(ctx, cn.AsteriskID, callBridgeID)
		return fmt.Errorf("could not create a call for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}
	log = log.WithFields(
		logrus.Fields{
			"call_type":      c.Type,
			"call_direction": c.Direction,
		})
	log.WithField("call", c).Debugf("Created a call. call_id: %s", c.ID)

	// set variables
	if errVariables := h.setVariablesCall(ctx, c); errVariables != nil {
		log.Errorf("Could not set variables. err: %v", errVariables)
		return errVariables
	}

	return h.ActionNext(ctx, c)
}

// typeFlowStart handles flow calltype start.
func (h *callHandler) typeFlowStart(ctx context.Context, cn *channel.Channel, data map[string]string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"channel_id":  cn.ID,
			"asterisk_id": cn.AsteriskID,
		})
	log.Debugf("Starting the flow incoming call handler. source: %s, destinaiton: %s", cn.SourceNumber, cn.DestinationNumber)

	id := h.util.CreateUUID()
	log = log.WithField("call_id", id)

	// get number info
	numb, err := h.reqHandler.NumberV1NumberGetByNumber(ctx, cn.DestinationNumber)
	if err != nil {
		log.Debugf("Could not find number info. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination, 0)
		return fmt.Errorf("could not get a number info by the destination. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// create call bridge
	callBridgeID, err := h.addCallBridge(ctx, cn, bridge.ReferenceTypeCall, id)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination, 0)
		return fmt.Errorf("could not add the channel to the join bridge. err: %v", err)
	}

	// create active flow
	af, err := h.reqHandler.FlowV1ActiveflowCreate(ctx, uuid.Nil, numb.CallFlowID, fmactiveflow.ReferenceTypeCall, id)
	if err != nil {
		af = &fmactiveflow.Activeflow{}
		log.Errorf("Could not get an active flow info. Created dummy active flow. This call will be hungup. call_id: %s, flow_id: %s", id, numb.CallFlowID)
	}
	log.Debugf("Created an active flow. active-flow: %v", af)

	source := commonaddress.CreateAddressByChannelSource(cn)
	destination := commonaddress.CreateAddressByChannelDestination(cn)
	status := call.GetStatusByChannelState(cn.State)

	c, err := h.Create(
		ctx,

		id,
		numb.CustomerID,

		cn.AsteriskID,
		cn.ID,
		callBridgeID,

		numb.CallFlowID,
		af.ID,
		uuid.Nil,
		call.TypeFlow,

		uuid.Nil,
		[]uuid.UUID{},

		uuid.Nil,
		[]uuid.UUID{},

		source,
		destination,

		status,

		data,
		af.CurrentAction,
		call.DirectionIncoming,

		uuid.Nil,
		[]rmroute.Route{},
	)

	if err != nil {
		log.Errorf("Could not create a call info. Hangup the call. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		_ = h.reqHandler.AstBridgeDelete(ctx, cn.AsteriskID, callBridgeID)
		return fmt.Errorf("could not create a call. call: %s, err: %v", c.ID, err)
	}
	log = log.WithFields(
		logrus.Fields{
			"call": c,
		})
	log.Debugf("Created a call. call: %s", c.ID)

	// set variables
	if errVariables := h.setVariablesCall(ctx, c); errVariables != nil {
		log.Errorf("Could not set variables. err: %v", errVariables)
		return errVariables
	}

	return h.ActionNext(ctx, c)
}
