package request

import "github.com/gofrs/uuid"

// V1SIPAnalysisPost represents the request for getting SIP analysis (messages + RTCP stats).
type V1SIPAnalysisPost struct {
	CallID    uuid.UUID `json:"call_id"`
	SIPCallID string    `json:"sip_call_id"`
	FromTime  string    `json:"from_time"`
	ToTime    string    `json:"to_time"`
}

// V1SIPPcapPost represents the request for getting PCAP data.
type V1SIPPcapPost struct {
	CallID    uuid.UUID `json:"call_id"`
	SIPCallID string    `json:"sip_call_id"`
	FromTime  string    `json:"from_time"`
	ToTime    string    `json:"to_time"`
}
