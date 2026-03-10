package callhandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
)

// rtpDebugStartRecording sends "start recording" to RTPEngine if the customer has RTP debug enabled.
// Best-effort: logs errors but does not return them (must not block call flow).
func (h *callHandler) rtpDebugStartRecording(ctx context.Context, cn *channel.Channel) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "rtpDebugStartRecording",
		"channel_id": cn.ID,
	})

	rtpengineAddress := cn.SIPData[channel.SIPDataKeyRTPEngineAddress]
	if rtpengineAddress == "" {
		log.Debugf("No rtpengine_address in SIPData. Skipping RTP debug start.")
		return
	}

	command := map[string]interface{}{
		"command": "start recording",
		"call-id": cn.SIPCallID,
	}

	res, err := h.reqHandler.RTPEngineV1CommandsSend(ctx, rtpengineAddress, command)
	if err != nil {
		log.Errorf("Could not send start recording to RTPEngine. rtpengine_address: %s, err: %v", rtpengineAddress, err)
		return
	}
	log.WithField("response", res).Debugf("Sent start recording to RTPEngine. rtpengine_address: %s, sip_call_id: %s", rtpengineAddress, cn.SIPCallID)
}

// rtpDebugStopRecording sends "stop recording" to RTPEngine for a call that had RTP debug enabled.
// Fetches a fresh channel from DB (hangup channel may be stale).
// Best-effort: logs errors but does not return them (must not block hangup flow).
func (h *callHandler) rtpDebugStopRecording(ctx context.Context, c *call.Call) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "rtpDebugStopRecording",
		"call_id": c.ID,
	})

	cn, err := h.channelHandler.Get(ctx, c.ChannelID)
	if err != nil {
		log.Errorf("Could not get fresh channel for RTP debug stop. channel_id: %s, err: %v", c.ChannelID, err)
		return
	}
	log.WithField("channel", cn).Debugf("Retrieved fresh channel for RTP debug stop. channel_id: %s", cn.ID)

	rtpengineAddress := cn.SIPData[channel.SIPDataKeyRTPEngineAddress]
	if rtpengineAddress == "" {
		log.Debugf("No rtpengine_address in SIPData. Skipping RTP debug stop.")
		return
	}

	command := map[string]interface{}{
		"command": "stop recording",
		"call-id": cn.SIPCallID,
	}

	res, err := h.reqHandler.RTPEngineV1CommandsSend(ctx, rtpengineAddress, command)
	if err != nil {
		log.Errorf("Could not send stop recording to RTPEngine. rtpengine_address: %s, err: %v", rtpengineAddress, err)
		return
	}
	log.WithField("response", res).Debugf("Sent stop recording to RTPEngine. rtpengine_address: %s, sip_call_id: %s", rtpengineAddress, cn.SIPCallID)
}
