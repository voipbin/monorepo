package listenhandler

import (
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
)

// A nil *VoipbinError wrapped in an error interface matches errors.As but cannot be converted
// into a typed response, so the handler must fall back to a plain 500.
func Test_errorResponse_typedNilPointer(t *testing.T) {
	var typedNil *cerrors.VoipbinError

	resp := errorResponse(typedNil)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusInternalServerError, resp.StatusCode)
	}
	if resp.DataType == cerrors.DataTypeVoipbinError {
		t.Errorf("Wrong match. expect a plain response, got data type: %s", resp.DataType)
	}
}
