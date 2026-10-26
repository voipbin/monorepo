package ai

import "strings"

// servicePrefix returns the part of an engine model before its first dot (the whole string when
// there is no dot). It is case-sensitive, like the model catalog.
func servicePrefix(m EngineModel) string {
	prefix, _, _ := strings.Cut(string(m), ".")

	return prefix
}

// MergeEngineKey decides what an update does with the engine key. key is the request value: nil
// means omitted (keep), a pointer to "" means delete and any other value replaces.
//
// It returns write, the value to store (nil means the column is not written and the stored key
// stays), and final, the key the AI holds after the update. Both come from this one place, so the
// validation and the write cannot disagree.
//
// An omitted key is dropped, not kept, when the service prefix of the engine model changes and the
// new model is not a platform model, so a key never moves to another vendor. A new platform model
// keeps the key (the runtime ignores it). An empty new model is treated as the stored one.
func MergeEngineKey(stored *AI, newModel EngineModel, key *string) (write *string, final string) {
	if key != nil {
		return key, *key
	}

	if newModel != "" && servicePrefix(newModel) != servicePrefix(stored.EngineModel) && !IsPlatformModel(newModel) {
		empty := ""

		return &empty, ""
	}

	return nil, stored.EngineKey
}
