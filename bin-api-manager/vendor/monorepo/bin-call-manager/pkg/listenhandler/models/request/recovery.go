package request

// V1DataRecoveryPost is
// v1 data type request struct for
// /v1/recovery POST
type V1DataRecoveryPost struct {
	AsteriskID string `json:"asterisk_id,omitempty"` // Asterisk ID to recover
}
