package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
	tmsipmessage "monorepo/bin-timeline-manager/models/sipmessage"

	"github.com/gofrs/uuid"
)

// TimelineV1SIPMessagesGet sends a request to timeline-manager
// to retrieve SIP messages matching the given criteria.
func (r *requestHandler) TimelineV1SIPMessagesGet(ctx context.Context, callID uuid.UUID, sipCallID string, fromTime, toTime string) (*tmsipmessage.SIPMessagesResponse, error) {
	uri := "/v1/sip/messages"

	req := map[string]any{
		"call_id":     callID,
		"sip_call_id": sipCallID,
		"from_time":   fromTime,
		"to_time":     toTime,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodPost, "timeline/sip-messages", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res tmsipmessage.SIPMessagesResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// TimelineV1SIPPcapGet sends a request to timeline-manager
// to retrieve SIP PCAP data for the given criteria.
func (r *requestHandler) TimelineV1SIPPcapGet(ctx context.Context, callID uuid.UUID, sipCallID string, fromTime, toTime string) ([]byte, error) {
	uri := "/v1/sip/pcap"

	req := map[string]any{
		"call_id":     callID,
		"sip_call_id": sipCallID,
		"from_time":   fromTime,
		"to_time":     toTime,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTimeline(ctx, uri, sock.RequestMethodPost, "timeline/sip-pcap", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d", tmp.StatusCode)
	}

	return tmp.Data, nil
}
