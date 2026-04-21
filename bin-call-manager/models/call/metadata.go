package call

// MetadataKey defines typed keys for Call.Metadata map entries.
type MetadataKey = string

const (
	// MetadataKeyRTPDebug indicates RTP debug capture was enabled for this call.
	// Set POST-CREATION in callhandler/start.go based on customer/number metadata.
	// Callers must not pre-set this key — it will be overwritten.
	MetadataKeyRTPDebug MetadataKey = "rtp_debug"

	// MetadataKeyRouteProviderIDs lists provider UUIDs (as a []string) that the call
	// must be routed through in failover order. Used by internal admin-test flows.
	// Set CREATION-TIME only by server-side trusted code. When present, call-manager
	// forwards the IDs to route-manager's DialrouteList, which returns synthetic
	// dialroutes bypassing normal customer/default merging.
	MetadataKeyRouteProviderIDs MetadataKey = "route_provider_ids"
)

// ValidMetadataKeys is the registry of every permitted metadata key.
// Every key that may appear in Call.Metadata MUST be declared here.
// The call-manager listen handler rejects requests with unknown keys.
//
// To add a new key:
//  1. Declare a MetadataKey constant above.
//  2. Add it to this registry.
//  3. Document whether it is creation-time only or post-creation-mutated.
var ValidMetadataKeys = map[MetadataKey]bool{
	MetadataKeyRTPDebug:         true,
	MetadataKeyRouteProviderIDs: true,
}
