package call

// MetadataKey defines typed keys for Call.Metadata map entries.
type MetadataKey = string

const (
	// MetadataKeyRTPDebug indicates RTP debug capture was enabled for this call.
	MetadataKeyRTPDebug MetadataKey = "rtp_debug"
)
