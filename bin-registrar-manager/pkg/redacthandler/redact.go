// Package redacthandler masks credential-shaped values (password, secret,
// token, credential, and any *_hash field) before they are attached to a
// log line. It is the single source of truth for "which field names carry
// a plaintext credential" across this service's transport
// (pkg/listenhandler) and business (pkg/extensionhandler,
// pkg/trunkhandler) layers.
package redacthandler

import "strings"

// Placeholder replaces the value of any sensitive field before logging.
const Placeholder = "[REDACTED]"

// sensitiveKeys are field/JSON-key names whose values must never reach the
// logs in plaintext (case-insensitive match).
var sensitiveKeys = map[string]bool{
	"password":   true,
	"secret":     true,
	"token":      true,
	"credential": true,
}

// IsSensitive reports whether name (matched case-insensitively) is a known
// credential-shaped field name: an exact match against sensitiveKeys, or a
// name ending in "hash" (covers "hash"/"direct_hash"/any future *_hash
// field carrying a direct-dial or similar bearer secret, without needing
// every such field name enumerated explicitly).
func IsSensitive(name string) bool {
	lower := strings.ToLower(name)
	return sensitiveKeys[lower] || strings.HasSuffix(lower, "hash")
}

// Fields returns a shallow copy of fields with the value of any sensitive
// key (see IsSensitive) replaced with Placeholder, safe to attach to a log
// line (e.g. logrus.Fields{"fields": redacthandler.Fields(updateFields)}).
// The input map is left untouched.
func Fields[K ~string, V any](fields map[K]V) map[K]any {
	out := make(map[K]any, len(fields))
	for k, v := range fields {
		if IsSensitive(string(k)) {
			out[k] = Placeholder
			continue
		}
		out[k] = v
	}
	return out
}

// String returns Placeholder if s is non-empty, otherwise "". Use to mask a
// single sensitive scalar (e.g. a password function argument) before
// attaching it to a log line, while still indicating whether a value was
// supplied.
func String(s string) string {
	if s == "" {
		return ""
	}
	return Placeholder
}
