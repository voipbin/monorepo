package callhandler

import (
	"context"
	"encoding/json"
	"strings"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/common"
)

// startIncomingDomainTypeRegistrar handles registrar incoming doamin type.
func (h *callHandler) startIncomingDomainTypeRegistrar(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startIncomingDomainTypeRegistrar",
		"channel_id": cn.ID,
	})

	// get default source/destination info
	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeExtension)
	source.TargetName = source.Target
	destination := h.channelHandler.AddressGetDestinationWithoutSpecificType(cn)

	// get customer
	tmpCustomerID := strings.TrimSuffix(cn.StasisData[channel.StasisDataTypeDomain], common.DomainRegistrarSuffix)
	customerID := uuid.FromStringOrNil(tmpCustomerID)

	log = log.WithFields(logrus.Fields{
		"customer_id": customerID,
		"source":      source,
		"destination": destination,
	})
	log.Debugf("Starting the flow incoming call handler. source_target: %s, destinaiton_target: %s", source.Target, destination.Target)

	switch destination.Type {
	case commonaddress.TypeAgent:
		return h.startIncomingDomainTypeRegistrarDestinationTypeAgent(ctx, cn, customerID, source, destination)

	case commonaddress.TypeConference:
		return h.startIncomingDomainTypeRegistrarDestinationTypeConference(ctx, cn, customerID, source, destination)

	case commonaddress.TypeTel:
		return h.startIncomingDomainTypeRegistrarDestinationTypeTel(ctx, cn, customerID, source, destination)

	case commonaddress.TypeExtension:
		return h.startIncomingDomainTypeRegistrarDestinationTypeExtension(ctx, cn, customerID, source, destination)

	default:
		log.Errorf("Unsupported destination type. destination_type: %s", destination.Type)
	}

	log.Errorf("Could not find correct destination type handler. destination_type: %s", destination.Type)
	_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
	return nil
}

// startIncomingDomainTypeRegistrarDestinationTypeAgent handles incoming call.
// SIP doamin type and destination type is agent.
func (h *callHandler) startIncomingDomainTypeRegistrarDestinationTypeAgent(
	ctx context.Context,
	cn *channel.Channel,
	customerID uuid.UUID,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeRegistrarDestinationTypeAgent",
		"channel_id":  cn.ID,
		"customer_id": customerID,
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
	if a.CustomerID != customerID {
		log.Errorf("The agent does not belong to the same customer. customer_id: %s, agent_customer_id: %s", customerID, a.CustomerID)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}

	// create destination
	tmpDestination := *destination
	tmpDestination.TargetName = a.Name

	// create tmp flow for connect
	option := fmaction.OptionConnect{
		Source: *source,
		Destinations: []commonaddress.Address{
			tmpDestination,
		},
		EarlyMedia:  false,
		RelayReason: false,
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
		customerID,
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
	h.startCallTypeFlow(ctx, cn, customerID, f.ID, source, destination)

	return nil
}

// startIncomingDomainTypeRegistrarDestinationTypeConference handles incoming call.
// Registrar doamin type and destination type is conference.
func (h *callHandler) startIncomingDomainTypeRegistrarDestinationTypeConference(
	ctx context.Context,
	cn *channel.Channel,
	customerID uuid.UUID,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeRegistrarDestinationTypeConference",
		"channel_id":  cn.ID,
		"customer_id": customerID,
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
	if cf.CustomerID != customerID {
		log.Errorf("The conference does not belong to the same customer. customer_id: %s, conference_customer_id: %s", customerID, cf.CustomerID)
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
		customerID,
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
	h.startCallTypeFlow(ctx, cn, cf.CustomerID, f.ID, source, destination)

	return nil
}

// startIncomingDomainTypeRegistrarDestinationTypeTel handles incoming call.
// SIP doamin type and destination type is tel.
func (h *callHandler) startIncomingDomainTypeRegistrarDestinationTypeTel(
	ctx context.Context,
	cn *channel.Channel,
	customerID uuid.UUID,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeRegistrarDestinationTypeTel",
		"channel_id":  cn.ID,
		"customer_id": customerID,
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
		customerID,
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
	h.startCallTypeFlow(ctx, cn, customerID, f.ID, source, destination)

	return nil
}

// startIncomingDomainTypeRegistrarDestinationTypeExtension handles incoming call.
// SIP doamin type and destination type is endpoint.
func (h *callHandler) startIncomingDomainTypeRegistrarDestinationTypeExtension(
	ctx context.Context,
	cn *channel.Channel,
	customerID uuid.UUID,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeRegistrarDestinationTypeExtension",
		"channel_id":  cn.ID,
		"customer_id": customerID,
		"source":      source,
		"destination": destination,
	})

	// get extension info
	filters := map[string]string{
		"customer_id": customerID.String(),
		"deleted":     "false",
		"extension":   destination.Target,
	}
	tmps, err := h.reqHandler.RegistrarV1ExtensionGets(ctx, "", 1, filters)
	if err != nil {
		log.Errorf("Could not get extension info. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}
	if len(tmps) == 0 {
		log.Errorf("The destination extension not found.")
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination) // return 404. destination not found
		return nil
	}

	ext := tmps[0]
	connectDestination := commonaddress.Address{
		Type:       commonaddress.TypeExtension,
		Target:     tmps[0].ID.String(),
		TargetName: tmps[0].Extension,
	}
	log.WithFields(logrus.Fields{
		"extension":           ext,
		"connect_destination": connectDestination,
	}).Debugf("Found destination extension info. extension_id: %s", ext.ID)

	// // create Destination
	// tmpDestination := *destination
	// tmpDestination.TargetName = tmpDestination.Target

	// create tmp flow for connect
	option := fmaction.OptionConnect{
		Source: *source,
		Destinations: []commonaddress.Address{
			connectDestination,
		},
		EarlyMedia:  false,
		RelayReason: false,
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
		customerID,
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
	h.startCallTypeFlow(ctx, cn, customerID, f.ID, source, destination)

	return nil
}
