package customer

// MetadataKey defines typed keys for customer metadata fields.
type MetadataKey = string

const (
	// MetadataKeyRTPDebug enables RTPEngine RTP capture (PCAP) for this customer's calls.
	MetadataKeyRTPDebug MetadataKey = "rtp_debug"
)

// Metadata holds internal-use configuration flags for a customer.
// Managed exclusively by ProjectSuperAdmin. Not exposed in WebhookMessage.
type Metadata struct {
	RTPDebug bool `json:"rtp_debug"` // enable RTPEngine RTP capture (PCAP)
}
