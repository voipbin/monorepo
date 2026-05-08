package callhandler

import "github.com/gofrs/uuid"

// dialTarget carries per-attempt dial parameters resolved at channel-creation time.
// Each call to createChannelOutgoing produces a fresh dialTarget, so failover
// to a different provider automatically picks up the new provider's Codecs.
type dialTarget struct { //nolint:unused
	URI         string
	TechHeaders map[string]string
	Codecs      string    // provider codec list; empty = no constraint
	ProviderID  uuid.UUID // uuid.Nil for non-provider (SIP) paths
}
