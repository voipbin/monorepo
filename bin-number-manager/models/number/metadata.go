package number

// MetadataKey defines typed keys for number metadata fields.
type MetadataKey = string

const (
	// MetadataKeyRTPDebug enables RTPEngine RTP capture (PCAP) for calls to this number.
	MetadataKeyRTPDebug MetadataKey = "rtp_debug"
)

// Metadata holds configuration flags for a number.
// Can be updated via PUT /numbers/{id}/metadata.
type Metadata struct {
	RTPDebug bool `json:"rtp_debug"` // enable RTPEngine RTP capture (PCAP)
}
