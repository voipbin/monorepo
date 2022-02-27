package request

import "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"

// V1DataConfbridgesPost is
// v1 data type request struct for
// /v1/confbridges POST
type V1DataConfbridgesPost struct {
	Type confbridge.Type `json:"type"`
}
