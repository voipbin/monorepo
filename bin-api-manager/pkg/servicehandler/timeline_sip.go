package servicehandler

import (
	"context"
	"fmt"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	tmsipmessage "monorepo/bin-timeline-manager/models/sipmessage"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// TimelineSIPAnalysisGet retrieves SIP analysis (messages + RTCP stats) for a call.
func (h *serviceHandler) TimelineSIPAnalysisGet(
	ctx context.Context,
	a *amagent.Agent,
	callID uuid.UUID,
) (*tmsipmessage.SIPAnalysisResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TimelineSIPAnalysisGet",
		"customer_id": a.CustomerID,
		"call_id":     callID,
	})

	// Get call to verify ownership and get timing
	call, err := h.callGet(ctx, callID)
	if err != nil {
		log.Infof("Could not get call: %v", err)
		return nil, fmt.Errorf("call not found")
	}
	log.WithField("call", call).Debugf("Retrieved call info. call_id: %s", call.ID)

	// Check permission
	if !h.hasPermission(ctx, a, call.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("Agent has no permission")
		return nil, fmt.Errorf("permission denied")
	}

	// Get channel to retrieve SIP Call-ID
	if call.ChannelID == "" {
		log.Info("Call has no channel ID")
		return nil, fmt.Errorf("no SIP data available for this call")
	}

	ch, err := h.reqHandler.CallV1ChannelGet(ctx, call.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel: %v", err)
		return nil, fmt.Errorf("no SIP data available for this call")
	}
	log.WithField("channel", ch).Debugf("Retrieved channel info. channel_id: %s", ch.ID)

	if ch.SIPCallID == "" {
		log.Info("Channel has no SIP Call-ID")
		return nil, fmt.Errorf("no SIP data available for this call")
	}

	// Determine time range from call timestamps
	fromTime := call.TMCreate
	toTime := call.TMHangup
	if toTime == nil {
		toTime = call.TMUpdate
	}

	// Call timeline-manager
	res, err := h.reqHandler.TimelineV1SIPAnalysisGet(ctx, callID, ch.SIPCallID, fromTime.Format(time.RFC3339), toTime.Format(time.RFC3339))
	if err != nil {
		log.Errorf("Could not get SIP analysis: %v", err)
		return nil, fmt.Errorf("upstream service unavailable")
	}

	return res, nil
}

// TimelineSIPPcapGet retrieves PCAP data for a call.
func (h *serviceHandler) TimelineSIPPcapGet(
	ctx context.Context,
	a *amagent.Agent,
	callID uuid.UUID,
) ([]byte, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TimelineSIPPcapGet",
		"customer_id": a.CustomerID,
		"call_id":     callID,
	})

	// Get call to verify ownership and get timing
	call, err := h.callGet(ctx, callID)
	if err != nil {
		log.Infof("Could not get call: %v", err)
		return nil, fmt.Errorf("call not found")
	}
	log.WithField("call", call).Debugf("Retrieved call info. call_id: %s", call.ID)

	// Check permission
	if !h.hasPermission(ctx, a, call.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("Agent has no permission")
		return nil, fmt.Errorf("permission denied")
	}

	// Get channel to retrieve SIP Call-ID
	if call.ChannelID == "" {
		log.Info("Call has no channel ID")
		return nil, fmt.Errorf("no SIP data available for this call")
	}

	ch, err := h.reqHandler.CallV1ChannelGet(ctx, call.ChannelID)
	if err != nil {
		log.Errorf("Could not get channel: %v", err)
		return nil, fmt.Errorf("no SIP data available for this call")
	}
	log.WithField("channel", ch).Debugf("Retrieved channel info. channel_id: %s", ch.ID)

	if ch.SIPCallID == "" {
		log.Info("Channel has no SIP Call-ID")
		return nil, fmt.Errorf("no SIP data available for this call")
	}

	// Determine time range from call timestamps
	fromTime := call.TMCreate
	toTime := call.TMHangup
	if toTime == nil {
		toTime = call.TMUpdate
	}

	// Call timeline-manager
	res, err := h.reqHandler.TimelineV1SIPPcapGet(ctx, callID, ch.SIPCallID, fromTime.Format(time.RFC3339), toTime.Format(time.RFC3339))
	if err != nil {
		log.Errorf("Could not get PCAP: %v", err)
		return nil, fmt.Errorf("upstream service unavailable")
	}

	return res, nil
}
