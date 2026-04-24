package errors

import (
	"encoding/json"

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
