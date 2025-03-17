package response

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
