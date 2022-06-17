package callhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
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
	trunkTwilio = "voipbin.pstn.twilio.com" //nolint:varcheck,deadcode // this is ok
	trunkTelnyx = "sip.telnyx.com"
)

// default max timeout for each services. sec.
const (
	defaultMaxTimeoutEcho       = "300" // maximum call duration for service echo. 5 min
	defaultMaxTimeoutSipService = "300" // maximum call duration for service sip-service. 5 min
)

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

	case domainTypeSIPService:
		return h.typeSipServiceStart(ctx, cn, data)

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
	if err := h.db.CallSetAsteriskID(context.Background(), callID, cn.AsteriskID, dbhandler.GetCurTime()); err != nil {
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
	bridgeID := uuid.Must(uuid.NewV4())
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

	case domainSIPService:
		return domainTypeSIPService

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

	// get conference info
	cf, err := h.reqHandler.CFV1ConferenceGet(ctx, cfID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return err
	}

	// generate call info
	tmpCall := call.NewCallByChannel(cn, cf.CustomerID, call.TypeFlow, call.DirectionIncoming, data)
	tmpCall.FlowID = cf.FlowID
	log = log.WithFields(
		logrus.Fields{
			"call_id": tmpCall.ID,
			"flow_id": tmpCall.FlowID,
		},
	)

	callBridgeID, err := h.addCallBridge(ctx, cn, bridge.ReferenceTypeCall, tmpCall.ID)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return fmt.Errorf("could not add the channel to the join bridge. err: %v", err)
	}
	tmpCall.BridgeID = callBridgeID

	// create active flow
	af, err := h.reqHandler.FMV1ActiveflowCreate(ctx, uuid.Nil, tmpCall.FlowID, fmactiveflow.ReferenceTypeCall, tmpCall.ID)
	if err != nil {
		log.Errorf("Could not create active flow. call: %s, flow: %s", tmpCall.ID, tmpCall.FlowID)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return errors.Wrap(err, "could not create an active flow")
	}
	log.WithField("active_flow", af).Debugf("Created an active flow. active_flow_id: %s", af.ID)
	tmpCall.Action = af.CurrentAction
	tmpCall.ActiveFlowID = af.ID

	// create a call
	c, err := h.create(ctx, tmpCall)
	if err != nil {
		log.Errorf("Could not create a call info. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		_ = h.reqHandler.AstBridgeDelete(ctx, cn.AsteriskID, callBridgeID)
		return fmt.Errorf("could not create a call for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}
	log = log.WithFields(
		logrus.Fields{
			"call_id":        c.ID,
			"call_type":      c.Type,
			"call_direction": c.Direction,
		})
	log.WithField("call", c).Debug("Created a call.")

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
			"channel":  cn.ID,
			"asterisk": cn.AsteriskID,
		})
	log.Debugf("Starting the flow incoming call handler. source: %s, destinaiton: %s", cn.SourceNumber, cn.DestinationNumber)

	// get number info
	numb, err := h.reqHandler.NMV1NumberGetByNumber(ctx, cn.DestinationNumber)
	if err != nil {
		log.Debugf("Could not find number info. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination, 0)
		return fmt.Errorf("could not get a number info by the destination. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// create a temp call info
	// todo: need to be fixed to set to the number's customer id
	tmpCall := call.NewCallByChannel(cn, numb.CustomerID, call.TypeFlow, call.DirectionIncoming, data)
	tmpCall.FlowID = numb.CallFlowID
	log = log.WithFields(
		logrus.Fields{
			"call_id": tmpCall.ID,
			"flow_id": tmpCall.FlowID,
		},
	)

	// create call bridge
	callBridgeID, err := h.addCallBridge(ctx, cn, bridge.ReferenceTypeCall, tmpCall.ID)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination, 0)
		return fmt.Errorf("could not add the channel to the join bridge. err: %v", err)
	}
	tmpCall.BridgeID = callBridgeID

	// create active flow
	af, err := h.reqHandler.FMV1ActiveflowCreate(ctx, uuid.Nil, numb.CallFlowID, fmactiveflow.ReferenceTypeCall, tmpCall.ID)
	if err != nil {
		af = &fmactiveflow.Activeflow{}
		log.Errorf("Could not get an active flow info. Created dummy active flow. This call will be hungup. call: %s, flow: %s", tmpCall.ID, tmpCall.FlowID)
	}
	log.Debugf("Created an active flow. active-flow: %v", af)
	tmpCall.Action = af.CurrentAction
	tmpCall.ActiveFlowID = af.ID

	c, err := h.create(ctx, tmpCall)
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

// typeSipServiceStart handles sip-service calltype request.
func (h *callHandler) typeSipServiceStart(ctx context.Context, cn *channel.Channel, data map[string]string) error {
	log := logrus.WithFields(
		logrus.Fields{
			"channel":  cn.ID,
			"asterisk": cn.AsteriskID,
		})
	log.Debugf("Starting the sip-service. source: %s, destinaiton: %s", cn.SourceNumber, cn.DestinationNumber)

	// set absolute timeout for 300 sec
	log.Debugf("Setting absolute timeout for sip-service type call")
	if err := h.reqHandler.AstChannelVariableSet(ctx, cn.AsteriskID, cn.ID, "TIMEOUT(absolute)", defaultMaxTimeoutSipService); err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return fmt.Errorf("could not set a timeout for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// generate a call info
	tmpCall := call.NewCallByChannel(cn, uuid.Nil, call.TypeSipService, call.DirectionIncoming, data)
	tmpCall.FlowID = uuid.Nil

	// create call bridge
	callBridgeID, err := h.addCallBridge(ctx, cn, bridge.ReferenceTypeCall, tmpCall.ID)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		return fmt.Errorf("could not add the channel to the join bridge. err: %v", err)
	}
	tmpCall.BridgeID = callBridgeID

	// create a call
	c, err := h.create(ctx, tmpCall)
	if err != nil {
		log.Errorf("Could not create a call info. Hangup the call. err: %v", err)
		_ = h.reqHandler.AstChannelHangup(ctx, cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing, 0)
		_ = h.reqHandler.AstBridgeDelete(ctx, cn.AsteriskID, callBridgeID)
		return fmt.Errorf("could not create a call for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}
	log = log.WithFields(
		logrus.Fields{
			"call":      c.ID,
			"type":      c.Type,
			"direction": c.Direction,
		})
	log.Debug("Created a call.")

	// get action for sip-service
	act, err := h.getSipServiceAction(ctx, c, cn)
	if err != nil {
		_ = h.HangingUp(ctx, c.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not get action handle for sip-service. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// execute action
	if err := h.ActionExecute(ctx, c, act); err != nil {
		log.Errorf("Could not execte the action. Hanging up the call. action: %s", act.Type)
		_ = h.HangingUp(ctx, c.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not get execute the action. channel: %s, asterisk: %s, call: %s, err: %v", cn.ID, cn.AsteriskID, c.ID, err)
	}

	return nil
}

// getSipServiceAction returns sip-service action handler by the call's destination.
func (h *callHandler) getSipServiceAction(ctx context.Context, c *call.Call, cn *channel.Channel) (*fmaction.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "getSipServiceAction",
		"call_id":    c.ID,
		"channel_id": cn.ID,
	})
	log.Debugf("Getting an action for sip-service. call_id: %s, channel_id: %s, destination: %s", c.ID, cn.ID, cn.DestinationNumber)

	var res *fmaction.Action
	switch c.Destination.Target {

	// answer
	case string(fmaction.TypeAnswer):
		// create an option for action answer
		option := fmaction.OptionAnswer{}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("could not marshal the option. action: %s, err: %v", fmaction.TypeAnswer, err)
		}

		// create an action
		res = &fmaction.Action{
			ID:     fmaction.IDStart,
			Type:   fmaction.TypeAnswer,
			Option: opt,
		}

	// confbridge_join
	case string(fmaction.TypeConfbridgeJoin):
		option := fmaction.OptionConfbridgeJoin{
			ConfbridgeID: uuid.FromStringOrNil(DefaultSipServiceOptionConfbridgeID),
		}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("could not marshal the option. action: %s, err: %v", fmaction.TypeConferenceJoin, err)
		}

		// create an action
		res = &fmaction.Action{
			ID:     fmaction.IDStart,
			Type:   fmaction.TypeConfbridgeJoin,
			Option: opt,
		}

	// echo
	case string(fmaction.TypeEcho):
		// create default option for echo
		option := fmaction.OptionEcho{
			Duration: 180 * 1000, // duration 180 sec
		}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("could not marshal the option echo. action: %s, err: %v", fmaction.TypeEcho, err)
		}

		// create an action
		res = &fmaction.Action{
			ID:     fmaction.IDStart,
			Type:   fmaction.TypeEcho,
			Option: opt,
		}

	// play
	case string(fmaction.TypePlay):
		// answer the call first
		if err := h.reqHandler.AstChannelAnswer(ctx, c.AsteriskID, c.ChannelID); err != nil {
			return nil, fmt.Errorf("could not answer the call. err: %v", err)
		}

		option := fmaction.OptionPlay{
			StreamURLs: []string{"https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"},
		}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("could not marshal the option. action: %s, err: %v", fmaction.TypePlay, err)
		}

		// create an action
		res = &fmaction.Action{
			ID:     fmaction.IDStart,
			Type:   fmaction.TypePlay,
			Option: opt,
		}

	// stream_echo
	case string(fmaction.TypeStreamEcho):
		option := fmaction.OptionStreamEcho{
			Duration: 180 * 1000,
		}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("could not marshal the option. action: %s, err: %v", fmaction.TypeStreamEcho, err)
		}

		// create an action
		res = &fmaction.Action{
			ID:     fmaction.IDStart,
			Type:   fmaction.TypeStreamEcho,
			Option: opt,
		}

	// default
	default:
		log.Errorf("Could not find correct sip-service handler. Hangup the call. call_id: %s, destination: %s", c.ID, c.Destination.Target)
		option := fmaction.OptionHangup{}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("could not marshal the option echo. action: %s, err: %v", fmaction.TypeEcho, err)
		}

		// create an action
		res = &fmaction.Action{
			ID:     fmaction.IDStart,
			Type:   fmaction.TypeHangup,
			Option: opt,
		}
	}

	return res, nil
}
