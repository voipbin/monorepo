package callhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// startIncomingDomainTypeSIP handles sip incoming doamin type.
func (h *callHandler) startIncomingDomainTypeSIP(ctx context.Context, cn *channel.Channel) error {
	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeSIP)
	destination := h.channelHandler.AddressGetDestinationWithoutSpecificType(cn)
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIP",
		"channel_id":  cn.ID,
		"source":      source,
		"destination": destination,
	})
	log.Debugf("Starting the flow incoming call handler. source_target: %s, destinaiton_target: %s", source.Target, destination.Target)

	// get domain info
	domainName := strings.TrimSuffix(cn.StasisData["domain"], doaminSIPSuffix)
	d, err := h.reqHandler.RegistrarV1DomainGetByDomainName(ctx, domainName)
	if err != nil {
		log.Errorf("Could not get domain info. err: %v", err)
		return errors.Wrap(err, "could not get domain info")
	}
	log.WithField("domain", d).Debugf("Found domain info. domain_id: %s", d.ID)

	switch destination.Type {
	case commonaddress.TypeAgent:
		return h.startIncomingDomainTypeSIPDestinationTypeAgent(ctx, cn, d, source, destination)

	case commonaddress.TypeConference:
		log.Debugf("The destination type is conference. Will execute the TypeSIPDestinationConference.")

	case commonaddress.TypeEndpoint:
		log.Debugf("The destination type is %s. Will execute the TypeSIPDestinationTypeEndpoint", destination.Type)

	case commonaddress.TypeLine:
		log.Debugf("The destination type is %s. Will execute the TypeSIPDestinationTypeLine", destination.Type)

	case commonaddress.TypeTel:
		log.Debugf("The destination type is %s. Will execute the TypeSIPDestinationTypeTel", destination.Type)

	default:
		log.Errorf("Unsupported destination type. destination_type: %s", destination.Type)
	}

	return nil
}

// startIncomingDomainTypeSIPDestinationTypeAgent handles incoming call.
// SIP doamin type and destination type is ßagent.
func (h *callHandler) startIncomingDomainTypeSIPDestinationTypeAgent(
	ctx context.Context,
	cn *channel.Channel,
	d *rmdomain.Domain,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDestinationTypeAgent",
		"channel_id":  cn.ID,
		"domain_id":   d.ID,
		"source":      source,
		"destination": destination,
	})

	// get agent info
	agentID := uuid.FromStringOrNil(destination.Target)
	a, err := h.reqHandler.AgentV1AgentGet(ctx, agentID)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return errors.Wrap(err, "could not get agent info")
	}
	log.WithField("agent", a).Debugf("Found agent info. agent_id: %s", a.ID)

	// validate the ownership
	if a.CustomerID != d.CustomerID {
		log.Errorf("The agent does not belong to the same customer. domain_customer_id: %s, agent_customer_id: %s", d.CustomerID, a.CustomerID)
		return fmt.Errorf("wrong customer")
	}

	id := h.utilHandler.CreateUUID()
	log = log.WithFields(logrus.Fields{
		"call_id":  id,
		"agent_id": a.ID,
	})

	callBridgeID, err := h.addCallBridge(ctx, cn, id)
	if err != nil {
		log.Errorf("Could not add the channel to the join bridge. err: %v", err)
		return errors.Wrap(err, "could not add the channel to the join bridge")
	}

	// create tmp flow for agent call
	option := fmaction.OptionAgentCall{
		AgentID: a.ID,
	}
	optionData, err := json.Marshal(&option)
	if err != nil {
		log.Errorf("Could not marshal the action option. err: %v", err)
		return errors.Wrap(err, "could not marshal the action option")
	}

	actions := []fmaction.Action{
		{
			Type:   fmaction.TypeAgentCall,
			Option: optionData,
		},
	}

	// create flow
	f, err := h.reqHandler.FlowV1FlowCreate(
		ctx,
		d.CustomerID,
		fmflow.TypeFlow,
		"",
		"",
		actions,
		false,
	)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		return errors.Wrap(err, "could not create flow")
	}

	// create activeflow
	af, err := h.reqHandler.FlowV1ActiveflowCreate(ctx, uuid.Nil, f.ID, fmactiveflow.ReferenceTypeCall, id)
	if err != nil {
		log.Errorf("Could not create active flow. call_id: %s, flow+id: %s", id, f.ID)
		return errors.Wrap(err, "could not create an activeflow")
	}
	log.WithField("activeflow", af).Debugf("Created an active flow. active_flow_id: %s", af.ID)

	status := call.GetStatusByChannelState(cn.State)
	c, err := h.Create(
		ctx,

		id,
		d.CustomerID,

		cn.ID,
		callBridgeID,

		f.ID,
		af.ID,
		uuid.Nil,
		call.TypeFlow,

		source,
		destination,

		status,

		cn.StasisData,
		af.CurrentAction,
		call.DirectionIncoming,

		uuid.Nil,
		[]rmroute.Route{},
	)
	if err != nil {
		log.Errorf("Could not create a call info. err: %v", err)
		return errors.Wrap(err, "could not create a call for channel")
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
		_, _ = h.HangingUp(ctx, id, ari.ChannelCauseNormalClearing)
		return errors.Wrap(errVariables, "could not set variables")
	}

	return h.ActionNext(ctx, c)
}
