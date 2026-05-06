package customer

// MetadataKey defines typed keys for customer metadata fields.
type MetadataKey = string

const (
	// MetadataKeyRTPDebug enables RTPEngine RTP capture (PCAP) for this customer's calls.
	MetadataKeyRTPDebug MetadataKey = "rtp_debug"
)

// Metadata holds configuration flags for a customer.
// Can be updated by ProjectSuperAdmin via PUT /customers/{id}/metadata
// or by CustomerAdmin via PUT /customer/metadata.
type Metadata struct {
	RTPDebug       bool   `json:"rtp_debug"`       // enable RTPEngine RTP capture (PCAP)
	OutboundCodecs string `json:"outbound_codecs"` // comma-separated preferred codec list for outbound calls (e.g. "PCMU,PCMA,G729"); empty means use server default
}
