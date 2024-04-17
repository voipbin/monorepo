package channelhandler

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// ARIStasisStart handles ARIStasisStart event message.
func (h *channelHandler) ARIStasisStart(ctx context.Context, e *ari.StasisStart) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ARIStasisStart",
		"event": e,
	})

	// get stasis name and parse the stasis data
	stasisName := e.Application
	stasisData := h.parseStasisData(e)

	// get and set the channel type
	chContext := channel.Context(stasisData[channel.StasisDataTypeContext])
	chType := h.getChannelType(chContext)
	direction := channel.Direction(stasisData[channel.StasisDataTypeDirection])

	// update channel's stasis info
	res, err := h.updateStasisInfo(ctx, e.Channel.ID, chType, stasisName, stasisData, direction)
	if err != nil {
		log.Errorf("Could not update the channel's stasis info. err: %v", err)
		return nil, errors.Wrap(err, "could not update the channel's stasis info")
	}

	if chContext == channel.ContextCallIncoming {
		log.Debugf("The channel is incoming call. Updating SIP info. channel_id: %s", res.ID)
		res, err = h.UpdateSIPInfo(
			ctx,
			res.ID,
			res.StasisData[channel.StasisDataTypeSIPCallID],
			res.StasisData[channel.StasisDataTypeSIPPAI],
			res.StasisData[channel.StasisDataTypeSIPPrivacy],
			channel.SIPTransport(res.StasisData[channel.StasisDataTypeTransport]),
		)
		if err != nil {
			log.Errorf("Could not update the sip info. err: %v", err)
			return nil, errors.Wrap(err, "could not update the sip info")
		}
	}

	return res, nil
}

// getChannelType returns channel type of the given channel context.
func (h *channelHandler) getChannelType(chContext channel.Context) channel.Type {

	mapChannelType := map[channel.Context]channel.Type{
		channel.ContextApplication:   channel.TypeApplication,
		channel.ContextConfIncoming:  channel.TypeConfbridge,
		channel.ContextConfOutgoing:  channel.TypeConfbridge,
		channel.ContextExternalMedia: channel.TypeExternal,
		channel.ContextCallIncoming:  channel.TypeCall,
		channel.ContextCallOutgoing:  channel.TypeCall,
		channel.ContextCallService:   channel.TypeCall,
		channel.ContextJoinCall:      channel.TypeJoin,
		channel.ContextRecording:     channel.TypeRecording,
	}

	res, ok := mapChannelType[chContext]
	if !ok {
		return channel.TypeNone
	}

	return res
}

// ARIChannelStateChange handles ARIChannelStateChange event message.
func (h *channelHandler) ARIChannelStateChange(ctx context.Context, e *ari.ChannelStateChange) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "ARIChannelStateChange",
		"event": e,
	})

	// update state
	res, err := h.UpdateState(ctx, e.Channel.ID, e.Channel.State)
	if err != nil {
		log.Errorf("Could not update the channel state. err: %v", err)
		return nil, err
	}

	if res.Direction == channel.DirectionOutgoing && res.State == ari.ChannelStateUp {
		log.WithField("channel", res).Debugf("The channel is outgoing and got answer. Updating the sip info. channel_id: %s", res.ID)
		res, err = h.UpdateSIPInfoByChannelVariable(ctx, res)
		if err != nil {
			log.Errorf("Could not update the channel. err: %v", err)
			return nil, err
		}
	}

	return res, nil
}

// parseStasisData returns initialized stasis data.
func (h *channelHandler) parseStasisData(e *ari.StasisStart) map[channel.StasisDataType]string {
	res := map[channel.StasisDataType]string{}

	tech := channel.GetTech(e.Channel.Name)
	if tech == channel.TechAudioSocket {
		// the audiosocket tech is special.
		// we can not set the 1 key here because the asterisk doesn't allowed it.
		i := 0
		for k := range e.Args {
			if i == 0 {
				res[channel.StasisDataTypeBridgeID] = k
			}
			i++
		}
		res[channel.StasisDataTypeContext] = string(channel.ContextExternalMedia)
		res[channel.StasisDataTypeContextType] = string(channel.ContextTypeCall)
	} else {
		for k, v := range e.Args {
			res[channel.StasisDataType(k)] = v
		}
	}

	return res
}
