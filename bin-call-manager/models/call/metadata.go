package call

// MetadataKey defines typed keys for Call.Metadata map entries.
type MetadataKey = string

const (
	// MetadataKeyRTPDebug indicates RTP debug capture was enabled for this call.
	// Set at CREATION TIME in callhandler/outgoing_call.go when the customer has
	// RTPDebug enabled in their metadata. status.go reads this key to decide
	// whether to start RTP debug recording, so no customer fetch is needed there.
	MetadataKeyRTPDebug MetadataKey = "rtp_debug"

	// MetadataKeyRouteProviderIDs lists provider UUIDs (as a []string) that the call
	// must be routed through in failover order. Used by internal admin-test flows.
	// Set CREATION-TIME only by server-side trusted code. When present, call-manager
	// forwards the IDs to route-manager's DialrouteList, which returns synthetic
	// dialroutes bypassing normal customer/default merging.
	MetadataKeyRouteProviderIDs MetadataKey = "route_provider_ids"

	// MetadataKeySkipSourceValidation, when set to true, instructs call-manager's
	// getValidatedSourceForOutgoingCall to return the caller-supplied source address
	// verbatim — skipping the customer-ownership lookup and the silent fallback to
	// Customer.DefaultOutgoingSourceNumberID. Used by internal admin-test flows that
	// must preserve a source number the provider's carrier has pre-authorized
	// (which is typically NOT a number owned by any voipbin customer).
	// Set CREATION-TIME only by server-side trusted code. Do not expose in any
	// customer-facing API body.
	MetadataKeySkipSourceValidation MetadataKey = "skip_source_validation"
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
	MetadataKeyRTPDebug:             true,
	MetadataKeyRouteProviderIDs:     true,
	MetadataKeySkipSourceValidation: true,
}
