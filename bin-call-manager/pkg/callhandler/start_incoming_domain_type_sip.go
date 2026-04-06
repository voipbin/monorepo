package callhandler

import (
	"context"
	"fmt"
	"strings"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	commonaddress "monorepo/bin-common-handler/models/address"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"
	nmnumber "monorepo/bin-number-manager/models/number"

	uuid "github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

// startIncomingDomainTypeSIP handles sip domain type incoming call.
func (h *callHandler) startIncomingDomainTypeSIP(ctx context.Context, cn *channel.Channel) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startIncomingDomainTypeSIP",
		"channel_id": cn.ID,
	})

	// check for direct hash
	if strings.HasPrefix(cn.DestinationNumber, dmdirect.DirectPrefix) {
		return h.startIncomingDomainTypeSIPDirect(ctx, cn, cn.DestinationNumber)
	}

	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeTel)
	destination := h.channelHandler.AddressGetDestination(cn, commonaddress.TypeTel)
	log = log.WithFields(logrus.Fields{
		"source":      source,
		"destination": destination,
	})
	log.Debugf("Starting the sip incoming call handler. source_target: %s, destination_target: %s", source.Target, destination.Target)

	// get number info
	filters := map[nmnumber.Field]any{
		nmnumber.FieldNumber:  destination.Target,
		nmnumber.FieldDeleted: false,
	}
	numbs, err := h.reqHandler.NumberV1NumberList(ctx, "", 1, filters)
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

	if numb.CallFlowID == uuid.Nil {
		log.Errorf("Number has no call flow configured. number_id: %s", numb.ID)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}

	// start the call type flow
	h.startCallTypeFlow(ctx, cn, numb.CustomerID, numb.CallFlowID, source, destination, &numb)
	return nil
}

// startIncomingDomainTypeSIPDirect handles incoming call to a direct resource via sip:direct.<hash>@sip.voipbin.net.
// It resolves the hash via direct-manager and dispatches to the appropriate handler based on resource_type.
func (h *callHandler) startIncomingDomainTypeSIPDirect(ctx context.Context, cn *channel.Channel, hash string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "startIncomingDomainTypeSIPDirect",
		"channel_id": cn.ID,
		"hash":       hash,
	})
	log.Debugf("Starting direct call handler. hash: %s", hash)

	source := h.channelHandler.AddressGetSource(cn, commonaddress.TypeTel)

	// resolve hash to direct record
	d, err := h.reqHandler.DirectV1DirectGetByHash(ctx, hash)
	if err != nil {
		log.Errorf("Could not get direct by hash. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("direct", d).Debugf("Retrieved direct info. direct_id: %s, resource_type: %s, resource_id: %s", d.ID, d.ResourceType, d.ResourceID)

	// dispatch by resource type
	switch d.ResourceType {
	case "extension":
		return h.startIncomingDomainTypeSIPDirectExtension(ctx, cn, d, source)
	case "conference":
		return h.startIncomingDomainTypeSIPDirectConference(ctx, cn, d, source)
	case dmdirect.ResourceTypeAI:
		return h.startIncomingDomainTypeSIPDirectAI(ctx, cn, d, source)
	case dmdirect.ResourceTypeAITeam:
		return h.startIncomingDomainTypeSIPDirectAITeam(ctx, cn, d, source)
	case "agent":
		return h.startIncomingDomainTypeSIPDirectAgent(ctx, cn, d, source)
	case "queue":
		return h.startIncomingDomainTypeSIPDirectQueue(ctx, cn, d, source)
	case "flow":
		return h.startIncomingDomainTypeSIPDirectFlow(ctx, cn, d, source)
	default:
		log.Errorf("Unsupported direct resource type. resource_type: %s", d.ResourceType)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
}

// startIncomingDomainTypeSIPDirectExtension handles direct hash call routed to an extension.
func (h *callHandler) startIncomingDomainTypeSIPDirectExtension(ctx context.Context, cn *channel.Channel, d *dmdirect.Direct, source *commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDirectExtension",
		"channel_id":  cn.ID,
		"resource_id": d.ResourceID,
	})

	// get extension info
	ext, err := h.reqHandler.RegistrarV1ExtensionGet(ctx, d.ResourceID)
	if err != nil {
		log.Errorf("Could not get extension. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("extension", ext).Debugf("Retrieved extension info. extension_id: %s", ext.ID)

	destination := &commonaddress.Address{
		Type:       commonaddress.TypeExtension,
		Target:     ext.ID.String(),
		TargetName: ext.Extension,
	}

	actions := []fmaction.Action{
		{
			Type: fmaction.TypeConnect,
			Option: fmaction.ConvertOption(fmaction.OptionConnect{
				Source:       *source,
				Destinations: []commonaddress.Address{*destination},
				EarlyMedia:   false,
				RelayReason:  false,
			}),
		},
	}

	f, err := h.reqHandler.FlowV1FlowCreate(ctx, ext.CustomerID, fmflow.TypeFlow, "tmp", "tmp flow for direct extension dialing", actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
		return nil
	}

	h.startCallTypeFlow(ctx, cn, ext.CustomerID, f.ID, source, destination, nil)
	return nil
}

// startIncomingDomainTypeSIPDirectConference handles direct hash call routed to a conference.
func (h *callHandler) startIncomingDomainTypeSIPDirectConference(ctx context.Context, cn *channel.Channel, d *dmdirect.Direct, source *commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDirectConference",
		"channel_id":  cn.ID,
		"resource_id": d.ResourceID,
	})

	cf, err := h.reqHandler.ConferenceV1ConferenceGet(ctx, d.ResourceID)
	if err != nil {
		log.Errorf("Could not get conference. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("conference", cf).Debugf("Retrieved conference info. conference_id: %s", cf.ID)

	destination := &commonaddress.Address{
		Type:   commonaddress.TypeConference,
		Target: cf.ID.String(),
	}

	actions := []fmaction.Action{
		{
			Type: fmaction.TypeAnswer,
		},
		{
			Type: fmaction.TypeConferenceJoin,
			Option: fmaction.ConvertOption(fmaction.OptionConferenceJoin{
				ConferenceID: cf.ID,
			}),
		},
	}

	tmpFlow, err := h.reqHandler.FlowV1FlowCreate(ctx, cf.CustomerID, fmflow.TypeFlow, "tmp", fmt.Sprintf("tmp flow for direct conference join. conference_id: %s", cf.ID), actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
		return nil
	}

	h.startCallTypeFlow(ctx, cn, cf.CustomerID, tmpFlow.ID, source, destination, nil)
	return nil
}

// startIncomingDomainTypeSIPDirectAI handles direct hash call routed to an AI resource.
func (h *callHandler) startIncomingDomainTypeSIPDirectAI(ctx context.Context, cn *channel.Channel, d *dmdirect.Direct, source *commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDirectAI",
		"channel_id":  cn.ID,
		"resource_id": d.ResourceID,
	})

	a, err := h.reqHandler.AIV1AIGet(ctx, d.ResourceID)
	if err != nil {
		log.Errorf("Could not get AI. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("ai", a).Debugf("Retrieved AI info. ai_id: %s", a.ID)

	destination := &commonaddress.Address{
		Type:   commonaddress.TypeAI,
		Target: d.ResourceID.String(),
	}

	actions := []fmaction.Action{
		{
			Type: fmaction.TypeAnswer,
		},
		{
			Type: fmaction.TypeAITalk,
			Option: fmaction.ConvertOption(fmaction.OptionAITalk{
				AssistanceType: amaicall.AssistanceTypeAI,
				AssistanceID:   a.ID,
			}),
		},
	}

	tmpFlow, err := h.reqHandler.FlowV1FlowCreate(ctx, a.CustomerID, fmflow.TypeFlow, "tmp", fmt.Sprintf("tmp flow for direct ai call. ai_id: %s", a.ID), actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
		return nil
	}

	h.startCallTypeFlow(ctx, cn, a.CustomerID, tmpFlow.ID, source, destination, nil)
	return nil
}

// startIncomingDomainTypeSIPDirectAITeam handles direct hash call routed to an AI team resource.
func (h *callHandler) startIncomingDomainTypeSIPDirectAITeam(ctx context.Context, cn *channel.Channel, d *dmdirect.Direct, source *commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDirectAITeam",
		"channel_id":  cn.ID,
		"resource_id": d.ResourceID,
	})

	team, err := h.reqHandler.AIV1TeamGet(ctx, d.ResourceID)
	if err != nil {
		log.Errorf("Could not get AI team. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("team", team).Debugf("Retrieved AI team info. team_id: %s", team.ID)

	destination := &commonaddress.Address{
		Type:   commonaddress.TypeAITeam,
		Target: d.ResourceID.String(),
	}

	actions := []fmaction.Action{
		{
			Type: fmaction.TypeAnswer,
		},
		{
			Type: fmaction.TypeAITalk,
			Option: fmaction.ConvertOption(fmaction.OptionAITalk{
				AssistanceType: amaicall.AssistanceTypeTeam,
				AssistanceID:   team.ID,
			}),
		},
	}

	tmpFlow, err := h.reqHandler.FlowV1FlowCreate(ctx, team.CustomerID, fmflow.TypeFlow, "tmp", fmt.Sprintf("tmp flow for direct ai team call. team_id: %s", team.ID), actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
		return nil
	}

	h.startCallTypeFlow(ctx, cn, team.CustomerID, tmpFlow.ID, source, destination, nil)
	return nil
}

// startIncomingDomainTypeSIPDirectAgent handles direct hash call routed to an agent resource.
func (h *callHandler) startIncomingDomainTypeSIPDirectAgent(ctx context.Context, cn *channel.Channel, d *dmdirect.Direct, source *commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDirectAgent",
		"channel_id":  cn.ID,
		"resource_id": d.ResourceID,
	})

	ag, err := h.reqHandler.AgentV1AgentGet(ctx, d.ResourceID)
	if err != nil {
		log.Errorf("Could not get agent. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("agent", ag).Debugf("Retrieved agent info. agent_id: %s", ag.ID)

	destination := &commonaddress.Address{
		Type:       commonaddress.TypeAgent,
		Target:     ag.ID.String(),
		TargetName: ag.Name,
	}

	actions := []fmaction.Action{
		{
			Type: fmaction.TypeConnect,
			Option: fmaction.ConvertOption(fmaction.OptionConnect{
				Source:       *source,
				Destinations: []commonaddress.Address{*destination},
				EarlyMedia:   false,
				RelayReason:  false,
			}),
		},
	}

	tmpFlow, err := h.reqHandler.FlowV1FlowCreate(ctx, ag.CustomerID, fmflow.TypeFlow, "tmp", fmt.Sprintf("tmp flow for direct agent call. agent_id: %s", ag.ID), actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
		return nil
	}

	h.startCallTypeFlow(ctx, cn, ag.CustomerID, tmpFlow.ID, source, destination, nil)
	return nil
}
