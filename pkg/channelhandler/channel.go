package channelhandler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// Create creates a channel.
func (h *channelHandler) Create(
	ctx context.Context,

	id string,
	asteriskID string,
	name string,
	channelType channel.Type,
	tech channel.Tech,

	// sip information
	sipCallID string,
	sipTransport channel.SIPTransport,

	// source/destination
	sourceName string,
	sourceNumber string,
	destinationName string,
	destinationNumber string,

	state ari.ChannelState,
	data map[string]interface{},
	stasisName string,
	stasisData map[string]string,
	bridgeID string,
	playbackID string,

	dialResult string,
	hangupCause ari.ChannelCause,

	direction channel.Direction,
) (*channel.Channel, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Create",
			"channel_id": id,
		},
	)

	c := &channel.Channel{
		ID:         id,
		AsteriskID: asteriskID,
		Name:       name,
		Type:       channelType,
		Tech:       tech,

		SIPCallID:    sipCallID,
		SIPTransport: sipTransport,

		SourceName:        sourceName,
		SourceNumber:      sourceNumber,
		DestinationName:   destinationName,
		DestinationNumber: destinationNumber,

		State:      state,
		Data:       data,
		StasisName: stasisName,
		StasisData: stasisData,
		BridgeID:   bridgeID,
		PlaybackID: playbackID,

		DialResult:  dialResult,
		HangupCause: hangupCause,
		Direction:   direction,
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

	return res, nil
}

// Get returns call.
func (h *channelHandler) Get(ctx context.Context, id string) (*channel.Channel, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Get",
			"channel_id": id,
		},
	)

	res, err := h.db.ChannelGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete deletes the channel.
func (h *channelHandler) Delete(ctx context.Context, id string, cause ari.ChannelCause) (*channel.Channel, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Delete",
			"channel_id": id,
		},
	)

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
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "SetDataItem",
			"channel_id": id,
		},
	)
	log.Debugf("Setting channel's data item. key: %s, value: %s", key, value)

	if err := h.db.ChannelSetDataItem(ctx, id, key, value); err != nil {
		log.Errorf("Could not set the channel's data item. channel_id: %s, err: %v", id, err)
		return err
	}

	return nil
}

// SetSIPTransport sets the channel's sip transport.
func (h *channelHandler) SetSIPTransport(ctx context.Context, id string, transport channel.SIPTransport) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "SetSIPTransport",
			"channel_id": id,
		},
	)
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
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "SetDirection",
			"channel_id": id,
		},
	)
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
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "SetSIPCallID",
			"channel_id": id,
		},
	)
	log.Debugf("Setting channel's transport. channel_id: %s, sip_call_id: %s", id, sipCallID)

	if err := h.db.ChannelSetSIPCallID(ctx, id, sipCallID); err != nil {
		log.Errorf("Could not set the channel's sip_call_id. channel_id: %s, err: %v", id, err)
		return err
	}

	return nil
}

// SetType sets the channel's type.
func (h *channelHandler) SetType(ctx context.Context, id string, channelType channel.Type) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "SetType",
			"channel_id": id,
		},
	)
	log.Debugf("Setting channel's type. channel_id: %s, channel_type: %s", id, channelType)

	if err := h.db.ChannelSetType(ctx, id, channelType); err != nil {
		log.Errorf("Could not set the channel's type. channel_id: %s, err: %v", id, err)
		return err
	}

	return nil
}

// UpdateState updates the channel's state.
func (h *channelHandler) UpdateState(ctx context.Context, id string, state ari.ChannelState) (*channel.Channel, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "UpdateState",
			"channel_id": id,
		},
	)
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

// UpdateBridgeID updates the channel's bridge id.
func (h *channelHandler) UpdateBridgeID(ctx context.Context, id string, bridgeID string) (*channel.Channel, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "UpdateBridgeID",
			"channel_id": id,
		},
	)
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

// GetWithTimeout gets the channel with for given timeout.
func (h *channelHandler) GetWithTimeout(ctx context.Context, id string, timeout time.Duration) (*channel.Channel, error) {
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
				tmp, err := h.Get(cctx, id)
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
