package response

import (
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
)

// V1ResponseCallsPost is
// v1 response type for
// /v1/calls POST
type V1ResponseCallsPost struct {
	Calls      []*call.Call           `json:"calls,omitempty"`
	Groupcalls []*groupcall.Groupcall `json:"groupcalls,omitempty"`
}

// V1ResponseCallsIDExternalMediaPost is
// v1 response type for
// /v1/calls/<id>/external-media POST
type V1ResponseCallsIDExternalMediaPost struct {
	MediaAddrIP   string `json:"media_addr_ip"`
	MediaAddrPort int    `json:"media_addr_port"`
}

// V1ResponseCallsIDDigitsGet is
// v1 response type for
// /v1/calls/<id>/digits GET
type V1ResponseCallsIDDigitsGet struct {
	Digits string `json:"digits"`
}
