package errors

import (
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
)

// FromResponse extracts a typed *VoipbinError from a sock.Response if
// the response signals one (StatusCode >= 400 AND DataType ==
// DataTypeVoipbinError AND Data unmarshals cleanly). Returns nil
// otherwise — callers should apply their own fallback.
func FromResponse(resp *sock.Response) *VoipbinError {
	if resp == nil || resp.StatusCode < 400 || resp.DataType != DataTypeVoipbinError || len(resp.Data) == 0 {
		return nil
	}
	out := &VoipbinError{}
	if err := json.Unmarshal(resp.Data, out); err != nil {
		return nil
	}
	return out
}

// ToResponse encodes a VoipbinError into a sock.Response carrying the
// JSON-serialized error in Data and the canonical HTTP status code.
// Returns an error if e is nil or marshaling fails — neither is
// expected in normal operation.
func ToResponse(e *VoipbinError) (*sock.Response, error) {
	if e == nil {
		return nil, fmt.Errorf("cannot marshal nil VoipbinError")
	}
	body, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("marshal VoipbinError: %w", err)
	}
	return &sock.Response{
		StatusCode: HTTPStatusFor(e.Status),
		DataType:   DataTypeVoipbinError,
		Data:       body,
	}, nil
}

// HTTPStatusFor maps a canonical Status to an HTTP status code.
// This is the single source of truth for the Status-to-HTTP mapping.
func HTTPStatusFor(s Status) int {
	switch s {
	case StatusInvalidArgument:
		return 400
	case StatusUnauthenticated:
		return 401
	case StatusPaymentRequired:
		return 402
	case StatusPermissionDenied:
		return 403
	case StatusNotFound:
		return 404
	case StatusAlreadyExists, StatusFailedPrecondition:
		return 409
	case StatusResourceExhausted:
		return 429
	case StatusUnavailable:
		return 503
	case StatusInternal:
		return 500
	default:
		return 500
	}
}
