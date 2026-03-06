package customer

// Metadata holds internal-use configuration flags for a customer.
// Managed exclusively by ProjectSuperAdmin. Not exposed in WebhookMessage.
type Metadata struct {
	RTPDebug bool `json:"rtp_debug"` // enable RTPEngine RTP capture (PCAP)
}
