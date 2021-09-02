package response

import "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

// V1ResponseStreamingsPost is
// v1 response type struct for
// /v1/streamings POST
type V1ResponseStreamingsPost struct {
	*transcribe.Transcribe
	// ID     uuid.UUID `json:"id"`
	// HostID uuid.UUID `json:"host_id"`

	// RTPAddress string `json:"rtp_address"` // ip:port
}
