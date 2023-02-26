package callhandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/common"
)

// list of application name
const (
	applicationAMD = "amd"
)

// domains
const (
// domainConference = "conference.voipbin.net"
// domainPSTN       = "pstn.voipbin.net"
// doaminSIPSuffix  = ".sip.voipbin.net" // suffix domain name for SIP
)

// list of domain types
const (
	domainTypeNone       = "none"
	domainTypeConference = "conference"
	domainTypePSTN       = "pstn"
	domainTypeSIP        = "sip"
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

// Start starts the call handle service
func (h *callHandler) Start(ctx context.Context, cn *channel.Channel) error {

	// check the stasis's context
	chCtx, ok := cn.StasisData["context"]
	if !ok {
		logrus.Errorf("Could not get channel context. stasis_data: %v", cn.StasisData)
		return fmt.Errorf("no context found")
	}

	switch chCtx {

	case common.ContextServiceCall:
		return h.startContextServiceCall(ctx, cn)

	case common.ContextIncomingCall:
		return h.startContextIncomingCall(ctx, cn)

	case common.ContextOutgoingCall:
		return h.startContextOutgoingCall(ctx, cn)

	case common.ContextRecording:
		return h.startContextRecording(ctx, cn)

	case common.ContextExternalSoop:
		return h.startContextExternalSoop(ctx, cn)

	case common.ContextExternalMedia:
		return h.startContextExternalMedia(ctx, cn)

	case common.ContextJoinCall:
		return h.startContextJoinCall(ctx, cn)

	case common.ContextApplication:
		return h.startContextApplication(ctx, cn)

	default:
		return fmt.Errorf("no route found for stasisstart. channel_id: %s, stasis_data: %v", cn.ID, cn.StasisData)
	}
}

// startContextServiceCall handles contextFromServiceCall context type of StasisStart event.
func (h *callHandler) startContextServiceCall(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startContextServiceCall",
		"channel_id": cn.ID,
	})
	log.Infof("Executing startHandlerContextFromServiceCall. context: %s", cn.StasisData["context"])

	fromContext := cn.StasisData["context_from"]
	switch fromContext {
	case serviceContextAMD:
		return h.startServiceFromAMD(ctx, cn.ID, cn.StasisData)
	default:
		return h.startServiceFromDefault(ctx, cn.ID, cn.StasisData)
	}
}

// startContextRecording handles contextFromServiceCall context type of StasisStart event.
func (h *callHandler) startContextRecording(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startHandlerContextRecording",
		"channel_id": cn.ID,
	})
	log.Infof("Executing startHandlerContextRecording. channel_id: %s", cn.ID)

	// set channel's type call.
	if errSet := h.channelHandler.VariableSet(ctx, cn.ID, "VB-TYPE", string(channel.TypeRecording)); errSet != nil {
		log.Errorf("Could not set a call type for channel. channel_id: %s, err: %v", cn.ID, errSet)
		return errors.Wrap(errSet, "could not set a call type for channel")
	}

	referenceType := cn.StasisData["reference_type"]
	referenceID := cn.StasisData["reference_id"]
	recordingID := cn.StasisData["recording_id"]
	name := cn.StasisData["recording_name"]
	format := cn.StasisData["format"]
	duration, _ := strconv.Atoi(cn.StasisData["duration"])
	silence, _ := strconv.Atoi(cn.StasisData["end_of_silence"])
	endKey := cn.StasisData["end_of_key"]
	direction := cn.StasisData["direction"]

	// parse recording name
	recordingName := fmt.Sprintf("%s_%s", name, direction)
	if errRecord := h.channelHandler.Record(ctx, cn.ID, recordingName, format, duration, silence, false, endKey, "fail"); errRecord != nil {
		log.Errorf("Could not start the recording. channel_id: %s, recording_name: %s, err: %v", cn.ID, recordingName, errRecord)
		return errors.Wrap(errRecord, "could not start the recording")
	}
	log.Infof("Recording started. id: %s, name: %s, reference_type: %s, reference_id: %s", recordingID, recordingName, referenceType, referenceID)

	return nil
}

// startContextExternalSoop handles contextExternalSnoop context type of StasisStart event.
func (h *callHandler) startContextExternalSoop(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "startHandlerContextExternalSnoop",
			"channel_id": cn.ID,
		},
	)
	log.Infof("Executing startHandlerContextExternalSnoop. channel_id: %s", cn.ID)

	// set channel's type call.
	if errSet := h.channelHandler.VariableSet(ctx, cn.ID, "VB-TYPE", string(channel.TypeExternal)); errSet != nil {
		log.Errorf("Could not set a call type for channel. channel_id: %s, err: %v", cn.ID, errSet)
		return errors.Wrap(errSet, "could not set a call type for channel")
	}

	callID := cn.StasisData["call_id"]
	bridgeID := cn.StasisData["bridge_id"]
	log = log.WithFields(logrus.Fields{
		"call_id":   callID,
		"bridge_id": bridgeID,
	})
	log.Debugf("Parsed info. call_id: %s, bridge_id: %s", callID, bridgeID)

	// put the channel to the bridge
	if errJoin := h.bridgeHandler.ChannelJoin(ctx, bridgeID, cn.ID, "", false, false); errJoin != nil {
		log.Errorf("Could not add the external snoop channel to the bridge. channel_id: %s, err: %v", cn.ID, errJoin)
		return errors.Wrap(errJoin, "could not set a call type for channel")
	}

	return nil
}

// startContextExternalMedia handles contextExternalMedia context type of StasisStart event.
func (h *callHandler) startContextExternalMedia(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "startHandlerContextExternalMedia",
			"channel_id": cn.ID,
		},
	)
	log.Infof("Executing startHandlerContextExternalMedia. channel_id: %s", cn.ID)

	// set channel's type call.
	if errSet := h.channelHandler.VariableSet(ctx, cn.ID, "VB-TYPE", string(channel.TypeExternal)); errSet != nil {
		log.Errorf("Could not set a call type for channel. channel_id: %s, err: %v", cn.ID, errSet)
		return errors.Wrap(errSet, "could not set a call type for channel")
	}

	callID := cn.StasisData["call_id"]
	bridgeID := cn.StasisData["bridge_id"]
	log.Debugf("Parsed info. call: %s, bridge: %s", callID, bridgeID)
	log = log.WithField("bridge_id", bridgeID)

	// put the channel to the bridge
	if errJoin := h.bridgeHandler.ChannelJoin(ctx, bridgeID, cn.ID, "", false, false); errJoin != nil {
		log.Errorf("Could not add the external snoop channel to the bridge. err: %v", errJoin)
		return errors.Wrap(errJoin, "could not add the external snoop channel to the bridge")
	}

	return nil
}

// startContextJoinCall handles contextJoinCall context type of StasisStart event.
func (h *callHandler) startContextJoinCall(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "startContextJoinCall",
			"channel_id": cn.ID,
		},
	)
	log.Infof("Executing startHandlerContextJoin. channel_id: %s", cn.ID)

	// set channel's type call.
	if errSet := h.channelHandler.VariableSet(ctx, cn.ID, "VB-TYPE", string(channel.TypeJoin)); errSet != nil {
		log.Errorf("Could not set a call type for channel. channel_id: %s, err: %v", cn.ID, errSet)
		return errors.Wrapf(errSet, "could not set a call type for channel. channel_id: %s", cn.ID)
	}

	confbridgeID := cn.StasisData["confbridge_id"]
	callID := cn.StasisData["call_id"]
	bridgeID := cn.StasisData["bridge_id"]
	log.Debugf("Parsed info. call_id: %s, bridge_id: %s, confbridge_id: %s", callID, bridgeID, confbridgeID)

	// put the channel to the bridge
	if errJoin := h.bridgeHandler.ChannelJoin(ctx, bridgeID, cn.ID, "", false, false); errJoin != nil {
		log.Errorf("Could not add the external snoop channel to the bridge. err: %v", errJoin)
		return errors.Wrap(errJoin, "could not add the external snoop channel to the channel")
	}

	// dial to the destination
	if errDial := h.channelHandler.Dial(ctx, cn.ID, "", defaultDialTimeout); errDial != nil {
		log.Errorf("Could not dial the channel to the destination. channel_id: %s, err: %v", cn.ID, errDial)
		return errors.Wrap(errDial, "could not dial the channel to the destination")
	}

	return nil
}

// startContextIncomingCall handles contextIncomingCall context type of StasisStart event.
func (h *callHandler) startContextIncomingCall(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startContextIncomingCall",
		"asterisk_id": cn.AsteriskID,
		"channel_id":  cn.ID,
	})
	log.Infof("Executing startHandlerContextIncomingCall. data: %v", cn.StasisData)

	// set channel's type call.
	if errSet := h.channelHandler.VariableSet(ctx, cn.ID, "VB-TYPE", string(channel.TypeCall)); errSet != nil {
		log.Errorf("Could not set the call type for the channel. err: %v", errSet)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // response 500
		return nil
	}

	// set the call durationtimeout
	_, err := h.channelHandler.HangingUpWithDelay(ctx, cn.ID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration)
	if err != nil {
		// could not send the delayed hangup request.
		// but we don't do anything here. just write log.
		log.Errorf("Could not send the channel hangup request for callprogress timeout. err: %v", err)
	}

	// get call type
	domainType := getDomainTypeIncomingCall(cn.StasisData["domain"])
	switch domainType {
	case domainTypeConference:
		return h.startIncomingDomainTypeConference(ctx, cn)

	case domainTypePSTN:
		return h.startIncomingDomainTypePSTN(ctx, cn)

	case domainTypeSIP:
		return h.startIncomingDomainTypeSIP(ctx, cn)

	default:
		// no route found
		log.Errorf("Could not find correct domain type handler. domain_type: %v", domainType)
		return fmt.Errorf("no route found for stasisstart. channel_id: %s", cn.ID)
	}
}

// startContextOutgoingCall handles contextOutgoingCall context type of StasisStart event.
func (h *callHandler) startContextOutgoingCall(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startContextOutgoingCall",
		"asterisk_id": cn.AsteriskID,
		"channel_id":  cn.ID,
	})
	log.Infof("Executing startContextOutgoingCall. channel_id: %s, data: %v", cn.ID, cn.StasisData)

	// get
	callID := uuid.FromStringOrNil(cn.StasisData["call_id"])
	if callID == uuid.Nil {
		log.Errorf("Could not get call_id info.")
		return fmt.Errorf("could not get correct call_id")
	}
	log = log.WithField("call_id", callID.String())

	// set channel's type call.
	if errSet := h.channelHandler.VariableSet(ctx, cn.ID, "VB-TYPE", string(channel.TypeCall)); errSet != nil {
		log.Errorf("Could not set channel's type. err: %v", errSet)
		_, _ = h.HangingUp(ctx, callID, call.HangupReasonNormal)
		return errors.Wrap(errSet, "could not set channel's type")
	}

	// set the call durationtimeout
	_, err := h.channelHandler.HangingUpWithDelay(ctx, cn.ID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration)
	if err != nil {
		// could not send the delayed hangup request.
		// but we don't do anything here. just write log.
		log.Errorf("Could not send the channel hangup request for callprogress timeout. err: %v", err)
	}

	// create call bridge
	bridgeID, err := h.addCallBridge(ctx, cn, callID)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		_, _ = h.HangingUp(ctx, callID, call.HangupReasonNormal)
		return errors.Wrap(err, "could not add the channel to the join bridge")
	}

	if errSet := h.db.CallSetBridgeID(ctx, callID, bridgeID); errSet != nil {
		log.Errorf("Could not set call bridge id. err: %v", errSet)
		_, _ = h.HangingUp(ctx, callID, call.HangupReasonNormal)
		return errors.Wrap(errSet, "could not set call bridge id")
	}

	// dial
	if errDial := h.channelHandler.Dial(ctx, cn.ID, cn.ID, defaultDialTimeout); errDial != nil {
		log.Errorf("Could not dial the channel. err: %v", errDial)
		_, _ = h.HangingUp(ctx, callID, call.HangupReasonNormal)
		return errors.Wrap(errDial, "could not dial the channel")
	}

	return nil
}

// startContextApplication handles contextApplication context type of StasisStart event.
func (h *callHandler) startContextApplication(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startHandlerContextApplication",
		"channel_id": cn.ID,
	})

	appName := cn.StasisData["application_name"]
	log.Debugf("Parsed application info. application: %s", appName)

	switch appName {
	case applicationAMD:
		return h.applicationHandleAMD(ctx, cn.ID, cn.StasisData)

	default:
		log.Errorf("Could not find correct event handler. app_name: %s", appName)
		return fmt.Errorf("could not find correct event handler. app_name: %s", appName)
	}
}

// addCallBridge creates a call bridge and put the channel into the join bridge.
func (h *callHandler) addCallBridge(ctx context.Context, cn *channel.Channel, callID uuid.UUID) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "addCallBridge",
		"call_id":    callID,
		"channel_id": cn.ID,
	})

	// create call bridge
	bridgeID := h.utilHandler.CreateUUID().String()
	bridgeName := fmt.Sprintf("reference_type=%s,reference_id=%s", bridge.ReferenceTypeCall, callID)
	tmp, err := h.bridgeHandler.Start(ctx, cn.AsteriskID, bridgeID, bridgeName, []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia})
	if err != nil {
		log.Errorf("Could not create a bridge for external media. error: %v", err)
		return "", err
	}
	log.WithField("bridge", tmp).Debugf("Created a call bridge. bridge_id: %s", tmp.ID)

	// add the channel to the bridge
	if errJoin := h.bridgeHandler.ChannelJoin(ctx, bridgeID, cn.ID, "", false, false); errJoin != nil {
		log.Errorf("Could not add the channel to the join bridge. error: %v", errJoin)
		_ = h.bridgeHandler.Destroy(ctx, bridgeID)
		return "", errJoin
	}

	return bridgeID, nil
}

// getDomainTypeIncomingCall returns the service type for incoming call context
func getDomainTypeIncomingCall(domain string) string {
	// all of the incoming calls are hitting the same context.
	// so we have to distinguish them using the requested domain name.
	switch domain {
	case common.DomainConference:
		return domainTypeConference

	case common.DomainPSTN:
		return domainTypePSTN
	}

	if strings.HasSuffix(domain, common.DomainSIPSuffix) {
		return domainTypeSIP
	}

	return domainTypeNone
}

// startIncomingDomainTypeConference handles conference domain type incoming call.
func (h *callHandler) startIncomingDomainTypeConference(ctx context.Context, cn *channel.Channel) error {
	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeSIP)
	destination := h.channelHandler.AddressGetDestination(cn, commonaddress.TypeConference)
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "startIncomingDomainTypeConference",
			"channel_id":  cn.ID,
			"source":      source,
			"destination": destination,
		})
	log.Debugf("Starting startIncomingDomainTypeConference. source_target: %s, destination_target: %s", source.Target, destination.Target)

	conferenceID := uuid.FromStringOrNil(destination.Target)
	log = log.WithFields(logrus.Fields{
		"conference_id": conferenceID,
	})

	// get conference info
	cf, err := h.reqHandler.ConferenceV1ConferenceGet(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}

	// start the call type flow
	h.startCallTypeFlow(ctx, cn, cf.CustomerID, cf.FlowID, source, destination, ari.ChannelCauseNormalClearing)

	return nil
}

// startIncomingDomainTypePSTN handles flow calltype start.
func (h *callHandler) startIncomingDomainTypePSTN(ctx context.Context, cn *channel.Channel) error {
	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeTel)
	destination := h.channelHandler.AddressGetDestination(cn, commonaddress.TypeTel)
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypePSTN",
		"channel_id":  cn.ID,
		"source":      source,
		"destination": destination,
	})
	log.Debugf("Starting the flow incoming call handler. source_target: %s, destinaiton_target: %s", source.Target, destination.Target)

	// get number info
	numb, err := h.reqHandler.NumberV1NumberGetByNumber(ctx, destination.Target)
	if err != nil {
		log.Debugf("Could not get a number info of the destination. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}
	log.WithField("number", numb).Debugf("Found number info. number_id: %s", numb.ID)

	// start the call type flow
	h.startCallTypeFlow(ctx, cn, numb.CustomerID, numb.CallFlowID, source, destination, ari.ChannelCauseNormalClearing)
	return nil
}

// startCallTypeFlow handles flow calltype start.
func (h *callHandler) startCallTypeFlow(ctx context.Context, cn *channel.Channel, customerID uuid.UUID, flowID uuid.UUID, source *commonaddress.Address, destination *commonaddress.Address, causeActionFail ari.ChannelCause) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startCallTypeFlow",
		"channel_id":  cn.ID,
		"customer_id": customerID,
		"flow_id":     flowID,
	})

	// create call id
	id := h.utilHandler.CreateUUID()

	// create call bridge
	callBridgeID, err := h.addCallBridge(ctx, cn, id)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return
	}

	// create activeflow
	af, err := h.reqHandler.FlowV1ActiveflowCreate(ctx, uuid.Nil, flowID, fmactiveflow.ReferenceTypeCall, id)
	if err != nil {
		log.Errorf("Could not create an activeflow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return
	}
	log.WithField("activeflow", af).Debugf("Created an active flow. activeflow_id: %s", af.ID)

	status := call.GetStatusByChannelState(cn.State)
	c, err := h.Create(
		ctx,

		id,
		customerID,

		cn.ID,
		callBridgeID,

		flowID,
		af.ID,
		uuid.Nil,
		call.TypeFlow,

		source,
		destination,

		status,

		map[call.DataType]string{},
		af.CurrentAction,
		call.DirectionIncoming,

		uuid.Nil,
		[]rmroute.Route{},
	)
	if err != nil {
		log.Errorf("Could not create a call info. call_id: %s, err: %v", id, err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return
	}
	log.WithField("call", c).Debugf("Created a call. call: %s", c.ID)

	// set variables
	if errVariables := h.setVariablesCall(ctx, c); errVariables != nil {
		log.Errorf("Could not set variables. err: %v", errVariables)
		// we are hanging up the call here. Because we've created a call above.
		// hangup the call with ari.ChannelCauseNetworkOutOfOrder.
		// this will response the 500.
		_, _ = h.hangingUpWithCause(ctx, c.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return
	}

	// execute the action
	if errNext := h.ActionNext(ctx, c); errNext != nil {
		// failed execute the action. hanging up the call with the given cause code
		_, _ = h.hangingUpWithCause(ctx, c.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return
	}

}
