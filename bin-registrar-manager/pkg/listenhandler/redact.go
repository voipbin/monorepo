package listenhandler

import (
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-registrar-manager/pkg/redacthandler"
)

// redactSensitiveJSON returns a copy of data with the values of any
// sensitive keys (password, secret, token, credential — see
// redacthandler.IsSensitive) replaced with a fixed placeholder, so it is
// safe to attach to a log line. extension/trunk create and update payloads
// carry a plaintext SIP password; this must run before any such payload is
// logged.
//
// If data is not valid JSON, the raw bytes are never returned — a fixed
// placeholder is used instead, since a malformed body can still contain a
// plaintext password in a field the caller intended to redact.
func redactSensitiveJSON(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return []byte(`"[unparseable body redacted]"`)
	}

	out, err := json.Marshal(redactJSONValue(parsed))
	if err != nil {
		return []byte(`"[redaction failed]"`)
	}

	return out
}

// redactJSONValue walks a decoded JSON value (as produced by
// json.Unmarshal into `any`) and replaces the value of any sensitive key
// in every object, at any nesting depth, including inside arrays.
func redactJSONValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			if redacthandler.IsSensitive(k) {
				out[k] = redacthandler.Placeholder
				continue
			}
			out[k] = redactJSONValue(vv)
		}
		return out

	case []any:
		out := make([]any, len(val))
		for i, vv := range val {
			out[i] = redactJSONValue(vv)
		}
		return out

	default:
		return val
	}
}

// redactRequestForLog returns a shallow copy of m with Data passed through
// redactSensitiveJSON, safe to attach to a log line (e.g. via
// logrus.Fields{"request": redactRequestForLog(m)}). The original request
// is left untouched.
func redactRequestForLog(m *sock.Request) *sock.Request {
	if m == nil {
		return nil
	}

	redacted := *m
	redacted.Data = redactSensitiveJSON(m.Data)
	return &redacted
}

// redactResponseForLog returns a shallow copy of r with Data passed through
// redactSensitiveJSON, safe to attach to a log line. The original response
// is left untouched.
func redactResponseForLog(r *sock.Response) *sock.Response {
	if r == nil {
		return nil
	}

	redacted := *r
	redacted.Data = redactSensitiveJSON(r.Data)
	return &redacted
}
