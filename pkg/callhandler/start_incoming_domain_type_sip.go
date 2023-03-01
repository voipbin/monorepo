package callhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/common"
)

// startIncomingDomainTypeSIP handles sip incoming doamin type.
func (h *callHandler) startIncomingDomainTypeSIP(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startIncomingDomainTypeSIP",
		"channel_id": cn.ID,
	})

	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeEndpoint)
	destination := h.channelHandler.AddressGetDestinationWithoutSpecificType(cn)

	log = log.WithFields(logrus.Fields{
		"source":      source,
		"destination": destination,
	})
	log.Debugf("Starting the flow incoming call handler. source_target: %s, destinaiton_target: %s", source.Target, destination.Target)

	// get domain info
	domainName := strings.TrimSuffix(cn.StasisData["domain"], common.DomainSIPSuffix)
	d, err := h.reqHandler.RegistrarV1DomainGetByDomainName(ctx, domainName)
	if err != nil {
		log.Errorf("Could not get domain info. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}
	log.WithField("domain", d).Debugf("Found domain info. domain_id: %s", d.ID)

	switch destination.Type {
	case commonaddress.TypeAgent:
		return h.startIncomingDomainTypeSIPDestinationTypeAgent(ctx, cn, d, source, destination)

	case commonaddress.TypeConference:
		return h.startIncomingDomainTypeSIPDestinationTypeConference(ctx, cn, d, source, destination)

	case commonaddress.TypeEndpoint:
		return h.startIncomingDomainTypeSIPDestinationTypeEndpoint(ctx, cn, d, source, destination)

	case commonaddress.TypeLine:
		log.Debugf("The destination type is %s. Will execute the TypeSIPDestinationTypeLine", destination.Type)

	case commonaddress.TypeTel:
		return h.startIncomingDomainTypeSIPDestinationTypeTel(ctx, cn, d, source, destination)

	default:
		log.Errorf("Unsupported destination type. destination_type: %s", destination.Type)
	}

	log.Errorf("Could not find correct destination type handler. destination_type: %s", destination.Type)
	_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
	return nil
}

// startIncomingDomainTypeSIPDestinationTypeAgent handles incoming call.
// SIP doamin type and destination type is agent.
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
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}
	log.WithField("agent", a).Debugf("Found agent info. agent_id: %s", a.ID)

	// validate the ownership
	if a.CustomerID != d.CustomerID {
		log.Errorf("The agent does not belong to the same customer. domain_customer_id: %s, agent_customer_id: %s", d.CustomerID, a.CustomerID)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}

	// create tmp flow for connect
	option := fmaction.OptionConnect{
		Source: *source,
		Destinations: []commonaddress.Address{
			*destination,
		},
		EarlyMedia:  true,
		RelayReason: true,
	}
	optionData, err := json.Marshal(&option)
	if err != nil {
		log.Errorf("Could not marshal the action option. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return nil
	}
	actions := []fmaction.Action{
		{
			Type:   fmaction.TypeConnect,
			Option: optionData,
		},
	}

	// create flow
	f, err := h.reqHandler.FlowV1FlowCreate(
		ctx,
		d.CustomerID,
		fmflow.TypeFlow,
		"tmp",
		"tmp flow for agent dialing",
		actions,
		false,
	)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return nil
	}

	// start the call type flow
	h.startCallTypeFlow(ctx, cn, d.CustomerID, f.ID, source, destination, ari.ChannelCauseUserBusy)

	return nil
}

// startIncomingDomainTypeSIPDestinationTypeConference handles incoming call.
// SIP doamin type and destination type is conference.
func (h *callHandler) startIncomingDomainTypeSIPDestinationTypeConference(
	ctx context.Context,
	cn *channel.Channel,
	d *rmdomain.Domain,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDestinationTypeConference",
		"channel_id":  cn.ID,
		"domain_id":   d.ID,
		"source":      source,
		"destination": destination,
	})

	// get conference info
	conferenceID := uuid.FromStringOrNil(destination.Target)
	cf, err := h.reqHandler.ConferenceV1ConferenceGet(ctx, conferenceID)
	if err != nil {
		log.Errorf("Could not get conference info. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}
	log.WithField("conference", cf).Debugf("Found conference info. conference_id: %s", cf.ID)

	// validate the ownership
	if cf.CustomerID != d.CustomerID {
		log.Errorf("The conference does not belong to the same customer. domain_customer_id: %s, conference_customer_id: %s", d.CustomerID, cf.CustomerID)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}

	// create tmp flow for conference join
	option := fmaction.OptionConferenceJoin{
		ConferenceID: cf.ID,
	}
	optionData, err := json.Marshal(&option)
	if err != nil {
		log.Errorf("Could not marshal the action option. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return nil
	}
	actions := []fmaction.Action{
		{
			Type:   fmaction.TypeConferenceJoin,
			Option: optionData,
		},
	}

	// create tmp flow
	f, err := h.reqHandler.FlowV1FlowCreate(
		ctx,
		d.CustomerID,
		fmflow.TypeFlow,
		"tmp",
		"tmp flow for conference join",
		actions,
		false,
	)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return nil
	}

	// start the call type flow
	h.startCallTypeFlow(ctx, cn, cf.CustomerID, f.ID, source, destination, ari.ChannelCauseUserBusy)

	return nil
}

// startIncomingDomainTypeSIPDestinationTypeTel handles incoming call.
// SIP doamin type and destination type is tel.
func (h *callHandler) startIncomingDomainTypeSIPDestinationTypeTel(
	ctx context.Context,
	cn *channel.Channel,
	d *rmdomain.Domain,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDestinationTypeTel",
		"channel_id":  cn.ID,
		"domain_id":   d.ID,
		"source":      source,
		"destination": destination,
	})

	// create tmp flow for connect
	option := fmaction.OptionConnect{
		Source: *source,
		Destinations: []commonaddress.Address{
			*destination,
		},
		EarlyMedia:  true,
		RelayReason: true,
	}
	optionData, err := json.Marshal(&option)
	if err != nil {
		log.Errorf("Could not marshal the action option. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return nil
	}
	actions := []fmaction.Action{
		{
			Type:   fmaction.TypeConnect,
			Option: optionData,
		},
	}

	// create tmp flow
	f, err := h.reqHandler.FlowV1FlowCreate(
		ctx,
		d.CustomerID,
		fmflow.TypeFlow,
		"tmp",
		"tmp flow for outgoing call dialing",
		actions,
		false,
	)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return nil
	}

	// start the call type flow
	h.startCallTypeFlow(ctx, cn, d.CustomerID, f.ID, source, destination, ari.ChannelCauseNormalClearing)

	return nil
}

// startIncomingDomainTypeSIPDestinationTypeEndpoint handles incoming call.
// SIP doamin type and destination type is endpoint.
func (h *callHandler) startIncomingDomainTypeSIPDestinationTypeEndpoint(
	ctx context.Context,
	cn *channel.Channel,
	d *rmdomain.Domain,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDestinationTypeEndpoint",
		"channel_id":  cn.ID,
		"domain_id":   d.ID,
		"source":      source,
		"destination": destination,
	})

	// create tmp flow for connect
	option := fmaction.OptionConnect{
		Source: *source,
		Destinations: []commonaddress.Address{
			*destination,
		},
		EarlyMedia:  true,
		RelayReason: true,
	}
	optionData, err := json.Marshal(&option)
	if err != nil {
		log.Errorf("Could not marshal the action option. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return nil
	}
	actions := []fmaction.Action{
		{
			Type:   fmaction.TypeConnect,
			Option: optionData,
		},
	}

	// create tmp flow
	f, err := h.reqHandler.FlowV1FlowCreate(
		ctx,
		d.CustomerID,
		fmflow.TypeFlow,
		"tmp",
		"tmp flow for outgoing call dialing",
		actions,
		false,
	)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder) // return 500. server error
		return nil
	}

	// start the call type flow
	h.startCallTypeFlow(ctx, cn, d.CustomerID, f.ID, source, destination, ari.ChannelCauseNormalClearing)

	return nil
}
