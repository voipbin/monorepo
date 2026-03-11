package callhandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
)

// rtpDebugStartRecording queries RTPEngine for allocated ports and starts a tcpdump capture
// via rtpengine-proxy. Best-effort: logs errors but does not return them (must not block call flow).
func (h *callHandler) rtpDebugStartRecording(ctx context.Context, c *call.Call, cn *channel.Channel) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "rtpDebugStartRecording",
		"call_id":    c.ID,
		"channel_id": cn.ID,
	})

	rtpengineAddress := cn.SIPData[channel.SIPDataKeyRTPEngineAddress]
	if rtpengineAddress == "" {
		log.Debugf("No rtpengine_address in SIPData. Skipping RTP debug start.")
		return
	}

	// Step 1: Query RTPEngine for allocated ports
	queryCommand := map[string]interface{}{
		"type":    "ng",
		"command": "query",
		"call-id": cn.SIPCallID,
	}

	queryRes, err := h.reqHandler.RTPEngineV1CommandsSend(ctx, rtpengineAddress, queryCommand)
	if err != nil {
		log.Errorf("Could not query RTPEngine. rtpengine_address: %s, err: %v", rtpengineAddress, err)
		return
	}
	log.WithField("query_response", queryRes).Debugf("RTPEngine query response. sip_call_id: %s", cn.SIPCallID)

	// Step 2: Extract external-facing local ports from query response.
	// For outgoing calls, from_tag is our internal side (Asterisk); exclude it
	// so we only capture RTP between RTPEngine and the external endpoint.
	fromTag := cn.SIPData["from_tag"]
	ports, err := extractLocalPorts(queryRes, fromTag)
	if err != nil {
		log.Errorf("Could not extract ports from query response. err: %v", err)
		return
	}
	log.Debugf("Extracted external ports from RTPEngine: %v (excluded from_tag: %s)", ports, fromTag)

	// Step 3: Build exec message with BPF filter
	// Use SIP Call-ID (not internal call UUID) so the PCAP filename matches
	// what timeline-manager uses to search GCS (it searches by SIP Call-ID).
	bpfFilter := buildBPFFilter(ports)

	execMsg := map[string]interface{}{
		"type":       "exec",
		"id":         cn.SIPCallID,
		"command":    "tcpdump",
		"parameters": []string{bpfFilter},
	}

	// Step 4: Send exec message to rtpengine-proxy via /v1/commands
	if _, err := h.reqHandler.RTPEngineV1CommandsSend(ctx, rtpengineAddress, execMsg); err != nil {
		log.Errorf("Could not send exec to rtpengine-proxy. err: %v", err)
		return
	}

	log.Debugf("Sent tcpdump exec to rtpengine-proxy. rtpengine_address: %s, sip_call_id: %s, ports: %v", rtpengineAddress, cn.SIPCallID, ports)
}

// rtpDebugStopRecording sends a kill message to rtpengine-proxy to stop the tcpdump capture.
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

	rtpengineAddress := cn.SIPData[channel.SIPDataKeyRTPEngineAddress]
	if rtpengineAddress == "" {
		log.Debugf("No rtpengine_address in SIPData. Skipping RTP debug stop.")
		return
	}

	killMsg := map[string]interface{}{
		"type": "kill",
		"id":   cn.SIPCallID,
	}

	if _, err := h.reqHandler.RTPEngineV1CommandsSend(ctx, rtpengineAddress, killMsg); err != nil {
		log.Errorf("Could not send kill to rtpengine-proxy. err: %v", err)
		return
	}

	log.Debugf("Sent tcpdump kill to rtpengine-proxy. rtpengine_address: %s, sip_call_id: %s", rtpengineAddress, cn.SIPCallID)
}
