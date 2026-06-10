package server

import (
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

const (
	// variablesMaxKeys is the maximum number of keys allowed in a caller-supplied variables map.
	variablesMaxKeys = 100
	// variablesMaxTotalBytes is the maximum total UTF-8 size (sum of len(key)+len(value)) allowed.
	variablesMaxTotalBytes = 64 * 1024
	// variablesMaxValueBytes is the maximum size of a single variable value.
	variablesMaxValueBytes = 32 * 1024
)

// validateVariables performs a fast-fail edge validation on the externally-supplied
// variables map before it is handed to the RPC layer. flow-manager remains the
// authority for empty/reserved-key handling and never fails creation; this only
// rejects clearly abusive payloads with an HTTP 400-class error.
func validateVariables(variables map[string]string) *cerrors.VoipbinError {
	if len(variables) == 0 {
		return nil
	}

	// NOTE: this edge check counts ALL supplied keys (including reserved voipbin.* and empty
	// keys) toward the key-count and total-byte caps. This is a deliberately conservative
	// pre-RPC guard against clearly abusive payloads. flow-manager is the authority and drops
	// reserved/empty keys BEFORE counting, so it enforces the caps only on surviving keys.
	if len(variables) > variablesMaxKeys {
		return cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_VARIABLES",
			"too many variables (max 100)",
		)
	}

	total := 0
	for k, v := range variables {
		if len(v) > variablesMaxValueBytes {
			return cerrors.InvalidArgument(
				commonoutline.ServiceNameAPIManager,
				"INVALID_VARIABLES",
				"variable value exceeds 32KB",
			)
		}
		total += len(k) + len(v)
	}

	if total > variablesMaxTotalBytes {
		return cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_VARIABLES",
			"variables total size exceeds 64KB",
		)
	}

	return nil
}

// convertVariables dereferences an optional request variables pointer into a plain map.
func convertVariables(in *map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	return *in
}
