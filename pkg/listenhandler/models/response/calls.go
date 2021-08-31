package response

// V1ResponseCallsIDExternalMediaPost is
// v1 response type for V1DataCallsIDExternalMediaPost request
// /v1/calls/<id>/external-media POST
type V1ResponseCallsIDExternalMediaPost struct {
	MediaAddrIP   string `json:"media_addr_ip"`
	MediaAddrPort int    `json:"media_addr_port"`
}
