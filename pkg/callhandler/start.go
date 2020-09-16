package callhandler

import (
	"context"
	"encoding/json"
	"fmt"

	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
)

// StasisStart event's context types
const (
	contextIncomingCall    = "call-in"
	contextOutgoingCall    = "call-out"
	contextFromServiceCall = "call-svc"
)

// domain types
const (
	domainConference = "conference.voipbin.net"
	domainSipService = "sip-service.voipbin.net"
)

// default max timeout for each services. sec.
const (
	defaultMaxTimeoutEcho       = "300"   // maximum call duration for service echo. 5 min
	defaultMaxTimeoutConference = "10800" // maximum call duration for service conf-soft. 3 hours
	defaultMaxTimeoutSipService = "300"   // maximum call duration for service sip-service. 5 min
)

// default sip service option variables
const (
	DefaultSipServiceOptionConferenceID = "037a20b9-d11d-4b63-a135-ae230cafd495" // default conference ID for conference@sip-service
)

func (h *callHandler) createCall(ctx context.Context, c *call.Call) error {
	if err := h.db.CallCreate(ctx, c); err != nil {
		return err
	}
	promCallCreateTotal.WithLabelValues(string(c.Direction), string(c.Type)).Inc()

	return nil
}

// CreateCallOutgoing creates a call for outgoing
func (h *callHandler) CreateCallOutgoing(id uuid.UUID, userID uint64, flowID uuid.UUID, source call.Address, destination call.Address) (*call.Call, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"id":          id,
		"user":        userID,
		"flow":        flowID,
		"source":      source,
		"destination": destination,
	})
	log.Debug("Creating outgoing call.")

	channelID := uuid.Must(uuid.NewV4()).String()
	cTmp := &call.Call{
		ID:          id,
		UserID:      userID,
		ChannelID:   channelID,
		FlowID:      flowID,
		Type:        call.TypeFlow,
		Status:      call.StatusDialing,
		Direction:   call.DirectionOutgoing,
		Source:      source,
		Destination: destination,
		Action: action.Action{
			ID:   action.IDInit,
			Next: action.IDBegin,
		},
		TMCreate: getCurTime(),
	}

	// create a call
	if err := h.createCall(ctx, cTmp); err != nil {
		log.Errorf("Could not create a call for outgoing call. err: %v", err)
		return nil, err
	}

	// get created call
	c, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get a created call for outgoing call. err: %v", err)
		return nil, err
	}

	endpoint := fmt.Sprintf("pjsip/call-out/sip:%s", destination.Target)
	appArgs := fmt.Sprintf("context=%s", contextOutgoingCall)

	// create a channel
	if err := h.reqHandler.AstChannelCreate(requesthandler.AsteriskIDCall, channelID, appArgs, endpoint, "", "", ""); err != nil {
		log.Errorf("Could not create a channel for outgoing call. err: %v", err)
		return nil, err
	}

	return c, nil
}

// StartCallHandle starts the call handle service
func (h *callHandler) StartCallHandle(cn *channel.Channel, data map[string]interface{}) error {

	// check the stasis's context
	switch data["context"].(string) {

	case contextFromServiceCall:
		return h.startHandlerContextFromServiceCall(cn, data)

	case contextIncomingCall:
		return h.startHandlerContextIncomingCall(cn, data)

	case contextOutgoingCall:
		return h.startHandlerContextOutgoingCall(cn, data)

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

// startHandlerContextIncomingCall handles contextIncomingCall context type of StasisStart event.
func (h *callHandler) startHandlerContextIncomingCall(cn *channel.Channel, data map[string]interface{}) error {
	logrus.Infof("Executing startHandlerContextIncomingCall. channel: %s, data: %v", cn.ID, data)

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeCall)); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// get call type
	cType := getTypeContextIncomingCall(data["domain"].(string))

	switch cType {
	case call.TypeConference:
		return h.typeConferenceStart(cn, data)

	case call.TypeSipService:
		return h.typeSipServiceStart(cn, data)

	default:
		// call.TypeNone will get to here.
		// no route found
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNoRouteDestination)
		return fmt.Errorf("no route found for stasisstart. asterisk_id: %s, channel_id: %s", cn.AsteriskID, cn.ID)
	}
}

// startHandlerContextOutgoingCall handles contextOutgoingCall context type of StasisStart event.
func (h *callHandler) startHandlerContextOutgoingCall(cn *channel.Channel, data map[string]interface{}) error {
	logrus.Infof("Executing startHandlerContextOutgoingCall. channel: %s, data: %v", cn.ID, data)

	// set channel's type call.
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "VB-TYPE", string(channel.TypeCall)); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	if err := h.reqHandler.AstChannelDial(cn.AsteriskID, cn.ID, "", 30); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a call type for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// do nothing here
	return nil
}

// getTypeContextIncomingCall returns the service type for incoming call context
func getTypeContextIncomingCall(domain string) call.Type {
	// all of the incoming calls are hitting the same context.
	// so we have to distinguish them using the requested domain name.
	switch domain {
	case domainConference:
		return call.TypeConference

	case domainSipService:
		return call.TypeSipService

	default:
		return call.TypeNone
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

	// create a call
	c := call.NewCallByChannel(cn, cf.UserID, call.TypeConference, call.DirectionIncoming, data)
	if err := h.createCall(ctx, c); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
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
		ID:     action.IDBegin,
		Type:   action.TypeConferenceJoin,
		Option: opt,
		Next:   action.IDEnd,
	}

	c, err = h.db.CallGet(ctx, c.ID)
	if err != nil {
		h.HangingUp(c, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("Could not get created call info. channel: %s, asterisk: %s, call: %s, err: %v", cn.ID, cn.AsteriskID, c.ID, err)
	}

	// execute action
	return h.ActionExecute(c, action)
}

// typeSipServiceStart handles sip-service calltype request.
func (h *callHandler) typeSipServiceStart(cn *channel.Channel, data map[string]interface{}) error {
	ctx := context.Background()

	log := log.WithFields(
		log.Fields{
			"channel":  cn.ID,
			"asterisk": cn.AsteriskID,
		})

	// set absolute timeout for 300 sec
	log.Debugf("Setting absolute timeout for sip-service type call")
	if err := h.reqHandler.AstChannelVariableSet(cn.AsteriskID, cn.ID, "TIMEOUT(absolute)", defaultMaxTimeoutSipService); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
		return fmt.Errorf("could not set a timeout for channel. channel: %s, asterisk: %s, err: %v", cn.ID, cn.AsteriskID, err)
	}

	// create a call
	c := call.NewCallByChannel(cn, call.UserIDAdmin, call.TypeSipService, call.DirectionIncoming, data)
	if err := h.createCall(ctx, c); err != nil {
		h.reqHandler.AstChannelHangup(cn.AsteriskID, cn.ID, ari.ChannelCauseNormalClearing)
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
	return h.ActionExecute(c, act)
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
			ID:     action.IDBegin,
			Type:   action.TypeAnswer,
			Option: opt,
			Next:   action.IDEnd,
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
			ID:     action.IDBegin,
			Type:   action.TypeConferenceJoin,
			Option: opt,
			Next:   action.IDEnd,
		}

	// default
	default:
		logrus.Warnf("Could not find correct sip-service handler. Use default handler. target: %s", c.Destination.Target)
		fallthrough

	// echo
	case string(action.TypeEcho):
		// create default option for echo
		option := action.OptionEcho{
			Duration: 180 * 1000, // duration 300 sec
			DTMF:     true,
		}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("Could not marshal the option echo. action: %s, err: %v", action.TypeEcho, err)
		}

		// create an action
		resAct = &action.Action{
			ID:     action.IDBegin,
			Type:   action.TypeEcho,
			Option: opt,
			Next:   action.IDEnd,
		}

	// play
	case string(action.TypePlay):
		// answer the call first
		if err := h.reqHandler.AstChannelAnswer(c.AsteriskID, c.ChannelID); err != nil {
			return nil, fmt.Errorf("could not answer the call. err: %v", err)
		}

		option := action.OptionPlay{
			StreamURL: []string{"https://github.com/pchero/asterisk-medias/raw/master/samples_codec/pcm_samples/example-mono_16bit_8khz_pcm.wav"},
		}
		opt, err := json.Marshal(option)
		if err != nil {
			return nil, fmt.Errorf("Could not marshal the option. action: %s, err: %v", action.TypePlay, err)
		}

		// create an action
		resAct = &action.Action{
			ID:     action.IDBegin,
			Type:   action.TypePlay,
			Option: opt,
			Next:   action.IDEnd,
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
			ID:     action.IDBegin,
			Type:   action.TypeStreamEcho,
			Option: opt,
			Next:   action.IDEnd,
		}

	}

	return resAct, nil
}
