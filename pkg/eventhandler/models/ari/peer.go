package ari

// PeerStatus type
type PeerStatus string

// peer status types
const (
	PeerStatusUnreachable PeerStatus = "Unreachable"
	PeerStatusReachable   PeerStatus = "Reachable"
)

// Peer struct
type Peer struct {
	PeerStatus PeerStatus `json:"peer_status"`
	Time       string     `json:"time"`
	Cause      string     `json:"cause"`
	Port       string     `json:"port"`
	Address    string     `json:"address"`
}

// EndpointState type
type EndpointState string

// endpoint state types
const (
	EndpointStateUnknown EndpointState = "unknown"
	EndpointStateOffline EndpointState = "offline"
	EndpointStateOnline  EndpointState = "online"
)

// Endpoint struct
type Endpoint struct {
	Resource   string        `json:"resource"`
	State      EndpointState `json:"state"`
	Technology string        `json:"technology"`
	ChannelIDs []string      `json:"channel_ids"`
}

// PeerStatusChange struct
type PeerStatusChange struct {
	Event
	Endpoint Endpoint `json:"endpoint"`
	Peer     Peer     `json:"peer"`
}
