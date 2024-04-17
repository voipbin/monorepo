package channelhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// StartSnoop starts snoop channel
func (h *channelHandler) StartSnoop(ctx context.Context, id string, snoopID string, appArgs string, directionSpy channel.SnoopDirection, directionWhisper channel.SnoopDirection) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "StartSnoop",
		"channel_id":        id,
		"snoop_id":          snoopID,
		"app_args":          appArgs,
		"direction_spy":     directionSpy,
		"direction_whisper": directionWhisper,
	})

	cn, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return nil, errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete < dbhandler.DefaultTimeStamp {
		log.Errorf("The channel has hungup already.")
		return nil, fmt.Errorf("the channel has hungup already")
	}

	res, err := h.reqHandler.AstChannelCreateSnoop(ctx, cn.AsteriskID, cn.ID, snoopID, appArgs, directionSpy, directionWhisper)
	if err != nil {
		log.Errorf("Could not create a snoop channel. err: %v", err)
		return nil, errors.Wrap(err, "could not create a snoop channel")
	}

	return res, nil
}

// StartExternalMedia starts a external media channel
func (h *channelHandler) StartExternalMedia(ctx context.Context, asteriskID string, id string, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string, data string, variables map[string]string) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "StartExternalMedia",
		"asterisk_id":     asteriskID,
		"channel_id":      id,
		"external_host":   externalHost,
		"encapsulation":   encapsulation,
		"transport":       transport,
		"connection_type": connectionType,
		"format":          format,
		"direction":       direction,
		"data":            data,
		"variables":       variables,
	})

	res, err := h.reqHandler.AstChannelExternalMedia(ctx, asteriskID, id, externalHost, encapsulation, transport, connectionType, format, direction, data, variables)
	if err != nil {
		log.Errorf("Could not create an external media channel. err: %v", err)
		return nil, errors.Wrap(err, "could not create an external media channel")
	}

	return res, nil
}

// StartChannel starts a channel
func (h *channelHandler) StartChannel(ctx context.Context, asteriskID string, id string, appArgs string, endpoint string, otherChannelID string, originator string, formats string, variables map[string]string) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "StartChannel",
		"asterisk_id":      asteriskID,
		"channel_id":       id,
		"app_args":         appArgs,
		"endpoint":         endpoint,
		"other_channel_id": otherChannelID,
		"originator":       originator,
		"formats":          formats,
		"variables":        variables,
	})

	res, err := h.reqHandler.AstChannelCreate(ctx, asteriskID, id, appArgs, endpoint, otherChannelID, originator, formats, variables)
	if err != nil {
		log.Errorf("Could not create a channel. err: %v", err)
		return nil, errors.Wrap(err, "could not create a channel")
	}

	return res, nil
}

// StartChannelWithBaseChannel creates a new channel with base channel.
func (h *channelHandler) StartChannelWithBaseChannel(ctx context.Context, baseChannelID string, id string, appArgs string, endpoint string, otherChannelID string, originator string, formats string, variables map[string]string) (*channel.Channel, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "StartChannelWithBaseChannel",
		"base_channel_id":  baseChannelID,
		"channel_id":       id,
		"app_args":         appArgs,
		"endpoint":         endpoint,
		"other_channel_id": otherChannelID,
		"originator":       originator,
		"formats":          formats,
		"variables":        variables,
	})

	tmp, err := h.Get(ctx, baseChannelID)
	if err != nil {
		log.Errorf("Could not get base channel. err: %v", err)
		return nil, errors.Wrap(err, "could not get base channel")
	}

	res, err := h.reqHandler.AstChannelCreate(ctx, tmp.AsteriskID, id, appArgs, endpoint, otherChannelID, originator, formats, variables)
	if err != nil {
		log.Errorf("Could not create a channel. err: %v", err)
		return nil, errors.Wrap(err, "could not create a channel")
	}

	return res, nil
}
