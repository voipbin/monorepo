package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"
)

// V1DataPeerEventsGet is the body-carried filter for GET /v1/peer-events.
// customer_id/page_token/page_size are NOT here — they arrive as query
// params (parsed directly from m.URI in v1PeerEventsGet), matching
// v1AnalysesGet's own customer_id/page_token/page_size-via-query-param
// split. Only the array filter that doesn't fit cleanly in a query string
// lives in the body, same reason /v1/events keeps `events []string` there.
//
// PeerAddresses is a full commonaddress.Address array (not flat
// peer_type/peer_target pairs) -- callers already hold Address values
// (bin-api-manager's Contact.Addresses), so the wire shape stays
// symmetric end-to-end with no lossy pair-type translation.
type V1DataPeerEventsGet struct {
	PeerAddresses []commonaddress.Address `json:"peer_addresses"`
}
