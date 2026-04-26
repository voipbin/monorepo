package listenhandler

import (
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-call-manager/pkg/dbhandler"

	pkgerrors "github.com/pkg/errors"
)

// Test_errorResponse_typed verifies that a *cerrors.VoipbinError is encoded
// via cerrors.ToResponse — the api-manager edge will recover the typed error
// from the resulting sock.Response via FromResponse.
func Test_errorResponse_typed(t *testing.T) {
	ve := cerrors.NotFound(
		commonoutline.ServiceNameCallManager,
		"CALL_NOT_FOUND",
		"The call was not found.",
	)

	resp := errorResponse(ve)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
	}
	if resp.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("DataType mismatch. expected: %s, got: %s", cerrors.DataTypeVoipbinError, resp.DataType)
	}
	if len(resp.Data) == 0 {
		t.Errorf("expected non-empty Data containing JSON-encoded VoipbinError")
	}
}

// Test_errorResponse_typedThroughPkgErrorsWrap verifies that errors.As walks
// the pkg/errors.Wrapf chain — the standard wrapping idiom in business
// handlers — and recovers the typed error.
func Test_errorResponse_typedThroughPkgErrorsWrap(t *testing.T) {
	ve := cerrors.NotFound(
		commonoutline.ServiceNameCallManager,
		"CONFBRIDGE_NOT_FOUND",
		"The conference was not found.",
	)
	wrapped := pkgerrors.Wrap(ve, "outer context")

	resp := errorResponse(wrapped)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
	}
	if resp.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("DataType mismatch. expected: %s, got: %s", cerrors.DataTypeVoipbinError, resp.DataType)
	}
}

// Test_errorResponse_legacyNotFound verifies that dbhandler.ErrNotFound
// (wrapped via pkg/errors) preserves the legacy 404 mapping for code paths
// that have not yet been migrated to typed errors.
func Test_errorResponse_legacyNotFound(t *testing.T) {
	wrapped := pkgerrors.Wrap(dbhandler.ErrNotFound, "could not find resource")

	resp := errorResponse(wrapped)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
	}
	if resp.DataType != "" {
		t.Errorf("DataType should be empty for legacy 404, got: %s", resp.DataType)
	}
}

// Test_errorResponse_unclassifiedError verifies that any unknown error
// (e.g., a DB connection failure) falls through to 500 — becomes INTERNAL
// at the api-manager translator default branch.
func Test_errorResponse_unclassifiedError(t *testing.T) {
	plain := fmt.Errorf("connection refused")

	resp := errorResponse(plain)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode mismatch. expected: 500, got: %d", resp.StatusCode)
	}
}

// Test_errorResponse_nilError defensive test — calling with nil should not
// crash and should return 500.
func Test_errorResponse_nilError(t *testing.T) {
	resp := errorResponse(nil)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode mismatch. expected: 500, got: %d", resp.StatusCode)
	}
}

// Test_errorResponse_typedNotFoundFromBusinessHandler verifies the end-to-end
// path: a business handler returns the exact typed error pattern this PR
// introduces, and errorResponse correctly encodes it.
func Test_errorResponse_typedNotFoundFromBusinessHandler(t *testing.T) {
	// simulate callHandler.Get returning typed NotFound on dbhandler.ErrNotFound
	dbErr := dbhandler.ErrNotFound
	typed := cerrors.NotFound(
		commonoutline.ServiceNameCallManager,
		"CALL_NOT_FOUND",
		"The call was not found.",
	).Wrap(dbErr)

	resp := errorResponse(typed)
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", resp.StatusCode)
	}
	if resp.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("DataType mismatch. expected: %s, got: %s", cerrors.DataTypeVoipbinError, resp.DataType)
	}

	// Cause chain still walks: errors.Is can find dbhandler.ErrNotFound
	if !stderrors.Is(typed, dbhandler.ErrNotFound) {
		t.Errorf("errors.Is should walk VoipbinError.Unwrap chain to find ErrNotFound")
	}
}
