package request

// PeerPair is a (peer_type, peer_target) pair carried in the body of
// GET /v1/peer-events.
type PeerPair struct {
	PeerType   string `json:"peer_type"`
	PeerTarget string `json:"peer_target"`
}

// V1DataPeerEventsGet is the body-carried filter for GET /v1/peer-events.
// customer_id/page_token/page_size are NOT here — they arrive as query
// params (parsed directly from m.URI in v1PeerEventsGet), matching
// v1AnalysesGet's own customer_id/page_token/page_size-via-query-param
// split. Only the array filter that doesn't fit cleanly in a query string
// lives in the body, same reason /v1/events keeps `events []string` there.
type V1DataPeerEventsGet struct {
	PeerPairs []PeerPair `json:"peer_pairs"`
}
