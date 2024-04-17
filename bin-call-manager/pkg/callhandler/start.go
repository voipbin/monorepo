package callhandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	rmroute "monorepo/bin-route-manager/models/route"

	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/common"
)

// list of application name
const (
	applicationAMD = "amd"
)

// list of domain types
const (
	domainTypeNone       = "none"
	domainTypeConference = "conference"
	domainTypePSTN       = "pstn"
	domainTypeSIP        = "sip"
	domainTypeRegistrar  = "registrar"
	domainTypeTrunk      = "trunk"
)

// pjsip endpoints
const (
	pjsipEndpointOutgoing       = "call-out"
	pjsipEndpointOutgoingDirect = "call-out-direct-"
)

// default sip service option variables
const (
	DefaultSipServiceOptionConfbridgeID = "037a20b9-d11d-4b63-a135-ae230cafd495" // default conference ID for conference@sip-service
)

// Start starts the call handle service
func (h *callHandler) Start(ctx context.Context, cn *channel.Channel) error {

	// check the stasis's context
	chCtx, ok := cn.StasisData[channel.StasisDataTypeContext]
	if !ok {
		logrus.Errorf("Could not get channel context. stasis_data: %v", cn.StasisData)
		return fmt.Errorf("no context found")
	}

	switch channel.Context(chCtx) {

	case channel.ContextCallService:
		return h.startContextServiceCall(ctx, cn)

	case channel.ContextCallIncoming:
		return h.startContextIncomingCall(ctx, cn)

	case channel.ContextCallOutgoing:
		return h.startContextOutgoingCall(ctx, cn)

	case channel.ContextRecording:
		return h.startContextRecording(ctx, cn)

	case channel.ContextExternalSoop:
		return h.startContextExternalSoop(ctx, cn)

	case channel.ContextExternalMedia:
		return h.startContextExternalMedia(ctx, cn)

	case channel.ContextJoinCall:
		return h.startContextJoinCall(ctx, cn)

	case channel.ContextApplication:
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
	log.Infof("Executing startHandlerContextFromServiceCall. context: %s", cn.StasisData[channel.StasisDataTypeContext])

	fromContext := cn.StasisData[channel.StasisDataTypeContextFrom]
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

	referenceType := cn.StasisData[channel.StasisDataTypeReferenceType]
	referenceID := cn.StasisData[channel.StasisDataTypeReferenceID]
	recordingID := cn.StasisData[channel.StasisDataTypeRecordingID]
	name := cn.StasisData[channel.StasisDataTypeRecordingName]
	format := cn.StasisData[channel.StasisDataTypeRecordingFormat]
	duration, _ := strconv.Atoi(cn.StasisData[channel.StasisDataTypeRecordingDuration])
	silence, _ := strconv.Atoi(cn.StasisData[channel.StasisDataTypeRecordingEndOfSilence])
	endKey := cn.StasisData[channel.StasisDataTypeRecordingEndOfKey]
	direction := cn.StasisData[channel.StasisDataTypeRecordingDirection]

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
	log := logrus.WithFields(logrus.Fields{
		"func":       "startHandlerContextExternalSnoop",
		"channel_id": cn.ID,
	})
	log.Infof("Executing startHandlerContextExternalSnoop. channel_id: %s", cn.ID)

	callID := cn.StasisData[channel.StasisDataTypeCallID]
	bridgeID := cn.StasisData[channel.StasisDataTypeBridgeID]
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
	log := logrus.WithFields(logrus.Fields{
		"func":       "startHandlerContextExternalMedia",
		"channel_id": cn.ID,
	})
	log.Infof("Executing startHandlerContextExternalMedia. channel_id: %s", cn.ID)

	callID := cn.StasisData[channel.StasisDataTypeCallID]
	bridgeID := cn.StasisData[channel.StasisDataTypeBridgeID]
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
	log := logrus.WithFields(logrus.Fields{
		"func":       "startContextJoinCall",
		"channel_id": cn.ID,
	})
	log.Infof("Executing startHandlerContextJoin. channel_id: %s", cn.ID)

	confbridgeID := cn.StasisData[channel.StasisDataTypeConfbridgeID]
	callID := cn.StasisData[channel.StasisDataTypeCallID]
	bridgeID := cn.StasisData[channel.StasisDataTypeBridgeID]
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

	// set the call durationtimeout
	_, err := h.channelHandler.HangingUpWithDelay(ctx, cn.ID, ari.ChannelCauseCallDurationTimeout, defaultTimeoutCallDuration)
	if err != nil {
		// could not send the delayed hangup request.
		// but we don't do anything here. just write log.
		log.Errorf("Could not send the channel hangup request for callprogress timeout. err: %v", err)
	}

	// get domain type
	domainType := getDomainTypeIncomingCall(cn.StasisData[channel.StasisDataTypeDomain])
	switch domainType {
	case domainTypeConference:
		return h.startIncomingDomainTypeConference(ctx, cn)

	case domainTypePSTN:
		return h.startIncomingDomainTypePSTN(ctx, cn)

	case domainTypeTrunk:
		return h.startIncomingDomainTypeTrunk(ctx, cn)

	case domainTypeRegistrar:
		return h.startIncomingDomainTypeRegistrar(ctx, cn)

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
	callID := uuid.FromStringOrNil(cn.StasisData[channel.StasisDataTypeCallID])
	if callID == uuid.Nil {
		log.Errorf("Could not get call_id info.")
		return fmt.Errorf("could not get correct call_id")
	}
	log = log.WithField("call_id", callID.String())

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

	appName := cn.StasisData[channel.StasisDataTypeApplicationName]
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
	bridgeID := h.utilHandler.UUIDCreate().String()
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

	if strings.HasSuffix(domain, common.DomainRegistrarSuffix) {
		return domainTypeRegistrar
	}

	if strings.HasSuffix(domain, common.DomainTrunkSuffix) {
		return domainTypeTrunk
	}

	return domainTypeNone
}

// startIncomingDomainTypeConference handles conference domain type incoming call.
func (h *callHandler) startIncomingDomainTypeConference(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startIncomingDomainTypeConference",
		"channel_id": cn.ID,
	})

	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeSIP)
	destination := h.channelHandler.AddressGetDestination(cn, commonaddress.TypeConference)
	log = log.WithFields(logrus.Fields{
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
	h.startCallTypeFlow(ctx, cn, cf.CustomerID, cf.FlowID, source, destination)

	return nil
}

// startIncomingDomainTypePSTN handles flow calltype start.
func (h *callHandler) startIncomingDomainTypePSTN(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startIncomingDomainTypePSTN",
		"channel_id": cn.ID,
	})

	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeTel)
	destination := h.channelHandler.AddressGetDestination(cn, commonaddress.TypeTel)
	log = log.WithFields(logrus.Fields{
		"source":      source,
		"destination": destination,
	})
	log.Debugf("Starting the flow incoming call handler. source_target: %s, destinaiton_target: %s", source.Target, destination.Target)

	// get number info
	filters := map[string]string{
		"number":  destination.Target,
		"deleted": "false",
	}
	numbs, err := h.reqHandler.NumberV1NumberGets(ctx, "", 1, filters)
	if err != nil {
		log.Errorf("Could not get numbers info. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}

	if len(numbs) == 0 {
		log.Errorf("No number info found. len: %d", len(numbs))
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}

	numb := numbs[0]
	log.WithField("number", numb).Infof("Found number info. number_id: %s", numb.ID)

	// start the call type flow
	h.startCallTypeFlow(ctx, cn, numb.CustomerID, numb.CallFlowID, source, destination)
	return nil
}

// startCallTypeFlow handles flow calltype start.
func (h *callHandler) startCallTypeFlow(ctx context.Context, cn *channel.Channel, customerID uuid.UUID, flowID uuid.UUID, source *commonaddress.Address, destination *commonaddress.Address) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startCallTypeFlow",
		"channel":     cn,
		"customer_id": customerID,
		"flow_id":     flowID,
		"source":      source,
		"destination": destination,
	})

	// create call id
	id := h.utilHandler.UUIDCreate()

	// validate balance
	if validBalance := h.ValidateCustomerBalance(ctx, id, customerID, call.DirectionIncoming, *source, *destination); !validBalance {
		log.Errorf("Could not pass the balance validation. customer_id: %s", customerID)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return
	}

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

		uuid.Nil,

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
