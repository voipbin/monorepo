package channelhandler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

// Create creates a channel.
func (h *channelHandler) Create(
	ctx context.Context,

	id string,
	asteriskID string,
	name string,
	channelType channel.Type,
	tech channel.Tech,

	// source/destination
	sourceName string,
	sourceNumber string,
	destinationName string,
	destinationNumber string,

	state ari.ChannelState,
) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Create",
		"channel_id": id,
	})

	c := &channel.Channel{
		ID:         id,
		AsteriskID: asteriskID,
		Name:       name,
		Type:       channelType,
		Tech:       tech,

		SIPCallID:    "",
		SIPTransport: channel.SIPTransportNone,

		SourceName:        sourceName,
		SourceNumber:      sourceNumber,
		DestinationName:   destinationName,
		DestinationNumber: destinationNumber,

		State:      state,
		Data:       map[string]interface{}{},
		StasisName: "",
		StasisData: map[channel.StasisDataType]string{},
		BridgeID:   "",
		PlaybackID: "",

		DialResult:  "",
		HangupCause: ari.ChannelCauseUnknown,
		Direction:   channel.DirectionNone,
	}
	log.WithField("channel", c).Debugf("Creating a new channel. channel_id: %s", c.ID)

	if err := h.db.ChannelCreate(ctx, c); err != nil {
		log.Errorf("Could not create a channel. err: %v", err)
		return nil, err
	}

	res, err := h.db.ChannelGet(ctx, c.ID)
	if err != nil {
		log.Errorf("Could not get a created channel. err: %v", err)
		return nil, err
	}
	promChannelCreateTotal.WithLabelValues(string(c.Direction), string(c.Type)).Inc()

	// start channel watcher
	if errHealth := h.reqHandler.CallV1ChannelHealth(ctx, res.ID, defaultHealthDelay, 0); errHealth != nil {
		logrus.Errorf("Could not start the channel water. err: %v", errHealth)
	}

	return res, nil
}

// get returns call.
func (h *channelHandler) get(ctx context.Context, id string) (*channel.Channel, error) {
	res, err := h.db.ChannelGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Get returns call.
func (h *channelHandler) Get(ctx context.Context, id string) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Get",
		"channel_id": id,
	})

	// we are giving some delay timeoute to getting a channel info.
	// because we are creating a channel info after receiving a ChannelCreate ari event.
	// but it is possible to receive other ari channel event earlier than creating a chanenl info.
	// for example, it is possible to get StasisStart event earlier than process the ChannelCreate event.
	res, err := h.getWithTimeout(ctx, id, defaultExistTimeout)
	if err != nil {
		log.Errorf("Could not get channel. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetChannelsForRecovery returns channels for recovery.
func (h *channelHandler) GetChannelsForRecovery(ctx context.Context, asteriskID string) ([]*channel.Channel, error) {
	filters := map[string]string{
		"asterisk_id": asteriskID,
		"state":       string(ari.ChannelStateUp),
		"deleted":     "false",
	}

	res, err := h.db.ChannelGets(ctx, 10000, "", filters)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get recovery channels for asterisk_id %s", asteriskID)
	}

	return res, nil
}

// Delete deletes the channel.
func (h *channelHandler) Delete(ctx context.Context, id string, cause ari.ChannelCause) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Delete",
		"channel_id": id,
	})

	if errEnd := h.db.ChannelEndAndDelete(ctx, id, cause); errEnd != nil {
		log.Errorf("Could not end the channel. channel_id: %s, err: %v", id, errEnd)
		return nil, errEnd
	}

	res, err := h.db.ChannelGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel. err: %v", err)
		return nil, err
	}
	promChannelDestroyedTotal.WithLabelValues(string(res.Direction), string(res.Type), strconv.Itoa(int(cause))).Inc()

	return res, nil
}

// SetDataItem sets the channel's data key/value.
func (h *channelHandler) SetDataItem(ctx context.Context, id string, key string, value interface{}) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SetDataItem",
		"channel_id": id,
		"key":        key,
		"value":      value,
	})
	log.Debugf("Setting channel's data item. key: %s, value: %s", key, value)

	if err := h.db.ChannelSetDataItem(ctx, id, key, value); err != nil {
		log.Errorf("Could not set the channel's data item. channel_id: %s, err: %v", id, err)
		return err
	}

	return nil
}

// SetSIPTransport sets the channel's sip transport.
func (h *channelHandler) SetSIPTransport(ctx context.Context, id string, transport channel.SIPTransport) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SetSIPTransport",
		"channel_id": id,
	})
	log.Debugf("Setting channel's transport. channel_id: %s, transport: %s", id, transport)

	if err := h.db.ChannelSetSIPTransport(ctx, id, transport); err != nil {
		log.Errorf("Could not set the channel's data item. channel_id: %s, err: %v", id, err)
		return err
	}

	go func() {
		tmp, err := h.db.ChannelGet(ctx, id)
		if err != nil {
			log.Errorf("Could not get channel. err: %v", err)
			return
		}

		if tmp.Direction != channel.DirectionNone && tmp.SIPTransport != channel.SIPTransportNone {
			promChannelTransportAndDirection.WithLabelValues(string(tmp.SIPTransport), string(tmp.Direction)).Inc()
		}
	}()

	return nil
}

// SetDirection sets the channel's direction.
func (h *channelHandler) SetDirection(ctx context.Context, id string, direction channel.Direction) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SetDirection",
		"channel_id": id,
	})
	log.Debugf("Setting channel's transport. channel_id: %s, direction: %s", id, direction)

	if err := h.db.ChannelSetDirection(ctx, id, direction); err != nil {
		log.Errorf("Could not set the channel's direction. channel_id: %s, err: %v", id, err)
		return err
	}

	go func() {
		tmp, err := h.db.ChannelGet(ctx, id)
		if err != nil {
			log.Errorf("Could not get channel. err: %v", err)
			return
		}

		if tmp.Direction != channel.DirectionNone && tmp.SIPTransport != channel.SIPTransportNone {
			promChannelTransportAndDirection.WithLabelValues(string(tmp.SIPTransport), string(tmp.Direction)).Inc()
		}
	}()

	return nil
}

// SetSIPCallID sets the channel's sip call id.
func (h *channelHandler) SetSIPCallID(ctx context.Context, id string, sipCallID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SetSIPCallID",
		"channel_id": id,
	})
	log.Debugf("Setting channel's transport. channel_id: %s, sip_call_id: %s", id, sipCallID)

	if err := h.db.ChannelSetSIPCallID(ctx, id, sipCallID); err != nil {
		log.Errorf("Could not set the channel's sip_call_id. channel_id: %s, err: %v", id, err)
		return err
	}

	return nil
}

// SetType sets the channel's type.
func (h *channelHandler) SetType(ctx context.Context, id string, channelType channel.Type) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SetType",
		"channel_id": id,
	})
	log.Debugf("Setting channel's type. channel_id: %s, channel_type: %s", id, channelType)

	if err := h.db.ChannelSetType(ctx, id, channelType); err != nil {
		log.Errorf("Could not set the channel's type. channel_id: %s, err: %v", id, err)
		return err
	}

	return nil
}

// UpdateState updates the channel's state.
func (h *channelHandler) UpdateState(ctx context.Context, id string, state ari.ChannelState) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "UpdateState",
		"channel_id": id,
	})
	log.Debugf("Updating channel's state. channel_id: %s, state: %s", id, state)

	var err error
	switch state {
	case ari.ChannelStateUp:
		err = h.db.ChannelSetStateAnswer(ctx, id, state)

	case ari.ChannelStateRing, ari.ChannelStateRinging:
		err = h.db.ChannelSetStateRinging(ctx, id, state)

	default:
		err = fmt.Errorf("no match state. state: %s", state)
	}

	if err != nil {
		return nil, err
	}

	res, err := h.db.ChannelGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated channel info. channel_id: %s, err: %v", id, err)
		return nil, err
	}

	return res, nil
}

// setSIPCallID gets the sip call id and sets to the channel.
func (h *channelHandler) setSIPCallID(ctx context.Context, id string, sipCallID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "setSIPCallID",
		"channel_id":  id,
		"sip_call_id": sipCallID,
	})

	if err := h.db.ChannelSetSIPCallID(ctx, id, sipCallID); err != nil {
		log.Errorf("Could not set the channel's sip_call_id. channel_id: %s, err: %v", id, err)
		return errors.Wrap(err, "could not set channel's sip_call_id")
	}

	return nil
}

// setSIPPai gets the sip pai info and sets to the channel.
func (h *channelHandler) setSIPPai(ctx context.Context, id string, sipPai string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "setSIPPai",
		"channel_id": id,
		"sip_pai":    sipPai,
	})

	if errSet := h.SetDataItem(ctx, id, "sip_pai", sipPai); errSet != nil {
		log.Errorf("could not set channel's sip_call_id. err: %v", errSet)
		return errors.Wrap(errSet, "could not set channel's sip_pai")
	}

	return nil
}

// setSIPPrivacy gets the sip privacy info and sets to the channel.
func (h *channelHandler) setSIPPrivacy(ctx context.Context, id string, sipPrivacy string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "setSIPPrivacy",
		"channel_id":  id,
		"sip_privacy": sipPrivacy,
	})

	if errSet := h.SetDataItem(ctx, id, "sip_privacy", sipPrivacy); errSet != nil {
		log.Errorf("could not set channel's sip_privacy. err: %v", errSet)
		return errors.Wrap(errSet, "could not set channel's sip_privacy")
	}

	return nil
}

// UpdateStasisName updates the channel's stasis_name.
func (h *channelHandler) UpdateStasisName(ctx context.Context, id string, stasisName string) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "UpdateStasisName",
		"channel_id": id,
	})

	if errSet := h.db.ChannelSetStasis(ctx, id, stasisName); errSet != nil {
		log.Errorf("Could not set channel's stasis name. err: %v", errSet)
		return nil, errors.Wrap(errSet, "could not set the channel stasis name")
	}

	res, err := h.db.ChannelGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated channel info. channel_id: %s, err: %v", id, err)
		return nil, err
	}

	return res, nil
}

// UpdateBridgeID updates the channel's bridge id.
func (h *channelHandler) UpdateBridgeID(ctx context.Context, id string, bridgeID string) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "UpdateBridgeID",
		"channel_id": id,
	})
	log.Debugf("Updating channel's bridge. channel_id: %s, bridge_id: %s", id, bridgeID)

	if errSet := h.db.ChannelSetBridgeID(ctx, id, bridgeID); errSet != nil {
		log.Errorf("Could not update the channel's bridge. channel_id: %s, err: %v", id, errSet)
		return nil, errSet
	}

	res, err := h.db.ChannelGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated channel info. channel_id: %s, err: %v", id, err)
		return nil, err
	}

	return res, nil
}

// UpdatePlaybackID updates the channel's playback id.
func (h *channelHandler) UpdatePlaybackID(ctx context.Context, id string, playbackID string) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "UpdatePlaybackID",
		"channel_id": id,
	})
	log.Debugf("Updating channel's playback id. channel_id: %s, playback_id: %s", id, playbackID)

	if errSet := h.db.ChannelSetPlaybackID(ctx, id, playbackID); errSet != nil {
		log.Errorf("Could not update the channel's playback id. channel_id: %s, err: %v", id, errSet)
		return nil, errSet
	}

	res, err := h.db.ChannelGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated channel info. channel_id: %s, err: %v", id, err)
		return nil, err
	}

	return res, nil
}

// getWithTimeout gets the channel within given timeout.
func (h *channelHandler) getWithTimeout(ctx context.Context, id string, timeout time.Duration) (*channel.Channel, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	chanRes := make(chan *channel.Channel)
	chanStop := make(chan bool)

	go func() {
		for {
			select {
			case <-chanStop:
				return

			default:
				tmp, err := h.get(cctx, id)
				if err != nil {
					time.Sleep(defaultDelayTimeout)
					continue
				}
				chanRes <- tmp
				return
			}
		}
	}()

	select {
	case res := <-chanRes:
		return res, nil
	case <-cctx.Done():
		chanStop <- true
		return nil, fmt.Errorf("could not get a channel within timeout. GetUntilTimeout. err: %v", cctx.Err())
	}
}

// UpdateMuteDirection updates the channel's mute direction.
func (h *channelHandler) UpdateMuteDirection(ctx context.Context, id string, muteDirection channel.MuteDirection) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "UpdateMuteDirection",
		"channel_id":     id,
		"mute_direction": muteDirection,
	})

	if errSet := h.db.ChannelSetMuteDirection(ctx, id, muteDirection); errSet != nil {
		log.Errorf("Could not update the channel's mute direction. channel_id: %s, err: %v", id, errSet)
		return nil, errSet
	}

	res, err := h.db.ChannelGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated channel info. channel_id: %s, err: %v", id, err)
		return nil, err
	}

	return res, nil
}

// updateStasisInfo updates the channel's stasis info and sip info.
func (h *channelHandler) updateStasisInfo(
	ctx context.Context,
	id string,
	chType channel.Type,
	stasisName string,
	stasisData map[channel.StasisDataType]string,
	direction channel.Direction,
) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "updateStasisInfo",
		"channel_id":   id,
		"channel_type": chType,
		"stasis_name":  stasisName,
		"stasis_data":  stasisData,
		"direction":    direction,
	})

	// update the channel's stasis and sip info
	if errSet := h.db.ChannelSetStasisInfo(
		ctx,
		id,
		chType,
		stasisName,
		stasisData,
		direction,
	); errSet != nil {
		log.Errorf("Could not update the channel stasis_name and stasis_data. err: %v", errSet)
		return nil, errors.Wrap(errSet, "could not update the channel stasis_name and stasis_data")
	}

	res, err := h.db.ChannelGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated channel info. channel_id: %s, err: %v", id, err)
		return nil, err
	}

	return res, nil
}

// UpdateSIPInfoByChannelVariable updates's channel's SIP info.
func (h *channelHandler) UpdateSIPInfoByChannelVariable(ctx context.Context, cn *channel.Channel) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "UpdateSIPInfo",
		"channel": cn,
	})

	// set sip call-id
	sipCallID, err := h.variableGet(ctx, cn, `CHANNEL(pjsip,Call-ID)`)
	if err != nil {
		log.Errorf("Could not get channel variable. err: %v", err)
		return nil, errors.Wrap(err, "could not get channel variable")
	}

	// set sip pai
	sipPai, err := h.variableGet(ctx, cn, `CHANNEL(pjsip,P-Asserted-Identity)`)
	if err != nil {
		log.Errorf("Could not get channel variable. err: %v", err)
		return nil, errors.Wrap(err, "could not get channel variable")
	}

	// set sip privacy
	sipPrivacy, err := h.variableGet(ctx, cn, `CHANNEL(pjsip,Privacy)`)
	if err != nil {
		log.Errorf("Could not get channel variable. err: %v", err)
		return nil, errors.Wrap(err, "could not get channel variable")
	}

	// set sip sipTransport
	sipTransport := channel.SIPTransport(cn.StasisData[channel.StasisDataTypeTransport])

	// get updated channel
	res, err := h.UpdateSIPInfo(ctx, cn.ID, sipCallID, sipPai, sipPrivacy, sipTransport)
	if err != nil {
		log.Errorf("Could not get updated channel. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateSIPInfoByChannel updates's channel's SIP info.
func (h *channelHandler) UpdateSIPInfo(ctx context.Context, id string, sipCallID string, sipPai string, sipPrivacy string, sipTransport channel.SIPTransport) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "UpdateSIPInfo",
		"channel_id":    id,
		"sip_call_id":   sipCallID,
		"sip_pai":       sipPai,
		"sip_privacy":   sipPrivacy,
		"sip_transport": sipTransport,
	})

	// set sip call-id
	if errSet := h.setSIPCallID(ctx, id, sipCallID); errSet != nil {
		log.Errorf("Could not set sip call id info. err: %v", errSet)
		return nil, errors.Wrap(errSet, "could not set sip call id info")
	}

	// set sip p-asserted-identity
	if errSet := h.setSIPPai(ctx, id, sipPai); errSet != nil {
		log.Errorf("Could not set sip pai info. err: %v", errSet)
		return nil, errors.Wrap(errSet, "could not set sip pai info")
	}

	// set sip privacy
	if errSet := h.setSIPPrivacy(ctx, id, sipPrivacy); errSet != nil {
		log.Errorf("Could not set sip privacy info. err: %v", errSet)
		return nil, errors.Wrap(errSet, "could not set sip privacy info")
	}

	// set sip transport
	if errSet := h.SetSIPTransport(ctx, id, sipTransport); errSet != nil {
		log.Errorf("Could not set sip transport info. err: %v", errSet)
		return nil, errors.Wrap(errSet, "could not set sip transport info")
	}

	// get updated channel
	res, err := h.get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated channel. err: %v", err)
		return nil, err
	}

	return res, nil
}
