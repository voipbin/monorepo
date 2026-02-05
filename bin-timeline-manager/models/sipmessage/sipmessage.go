package sipmessage

// SIPMessage represents a single SIP message from Homer.
type SIPMessage struct {
	Timestamp string `json:"timestamp"`
	Method    string `json:"method"`
	SrcIP     string `json:"src_ip"`
	SrcPort   int    `json:"src_port"`
	DstIP     string `json:"dst_ip"`
	DstPort   int    `json:"dst_port"`
	Raw       string `json:"raw"`
}

// SIPMessagesResponse is the response for SIP messages list.
type SIPMessagesResponse struct {
	CallID    string        `json:"call_id"`
	SIPCallID string        `json:"sip_call_id"`
	Messages  []*SIPMessage `json:"messages"`
}

// PcapResponse is the response for PCAP download.
type PcapResponse struct {
	CallID      string `json:"call_id"`
	DownloadURI string `json:"download_uri"`
	ExpiresAt   string `json:"expires_at"`
}
