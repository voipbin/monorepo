package call

import "testing"

func Test_ValidMetadataKeys_contains_all_declared_constants(t *testing.T) {
	// Every declared MetadataKey constant must be registered.
	required := []MetadataKey{
		MetadataKeyRTPDebug,
		MetadataKeyRouteProviderIDs,
		MetadataKeySkipSourceValidation,
		MetadataKeyCodecs, // add this
	}
	for _, k := range required {
		if !ValidMetadataKeys[k] {
			t.Errorf("MetadataKey %q is declared but missing from ValidMetadataKeys", k)
		}
	}
}

func Test_ValidMetadataKeys_rejects_unknown(t *testing.T) {
	if ValidMetadataKeys["route_providers_ids"] { // typo
		t.Error("ValidMetadataKeys should not accept typo key")
	}
	if ValidMetadataKeys[""] {
		t.Error("ValidMetadataKeys should not accept empty key")
	}
}
